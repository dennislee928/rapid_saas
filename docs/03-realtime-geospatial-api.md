# Realtime Geospatial Coordinate API — Implementation Plan

A lightweight pub/sub-by-group_id realtime location service. Publishers POST `{lat, lng, group_id}`; subscribers receive a live WebSocket stream of moving markers, optionally filtered by bbox or radius. Monetised on concurrent connections (CCU). Built to run on a zero-cost stack (Cloudflare Workers + Durable Objects + Turso, Fly.io as a fallback compute lane).

---

## 1. Product Overview & Target Users

### What it is
A "Pusher/Ably for things on a map." Two surfaces:

- **Publish API** (HTTP POST or WebSocket): `POST /v1/positions` with `{group_id, entity_id, lat, lng, ts, meta?}`. One call per ping.
- **Subscribe API** (WebSocket): `wss://geo.example.com/v1/groups/{group_id}?bbox=...` returns a JSON stream of position deltas, plus snapshot-on-connect.

Plus a tiny JS SDK that wires the stream straight into Mapbox/MapLibre/Leaflet markers with reasonable defaults (smoothed interpolation, marker pooling, automatic re-subscribe on viewport change).

### Target users
1. **Small dev shops doing delivery / field-service apps.** They have 5–500 drivers, want a "see all vans on a map" view for dispatchers and customers. They don't want to run a Redis Pub/Sub + a WebSocket fleet.
2. **Festival, conference, and event-support apps.** "Find my friends," "where are the volunteers right now," guided tours. Ephemeral groups, very spiky load (10k people online for 8 hours, then zero).
3. **Hackathon teams, indie game/AR devs.** Prototyping `.io`-style multiplayer or AR scavenger hunts. They want a free tier they can ship a demo on tonight.
4. **Asset trackers** (fleet of e-bikes, scooters, rental gear). Lower-CCU but high publisher fan-in.

### Why pay vs. building on Pusher / Ably / Supabase Realtime
| Concern | Pusher/Ably | Supabase Realtime | This service |
|---|---|---|---|
| Geo-aware filtering (bbox/radius) | DIY in client; you pay to receive every channel msg | Postgres-side filter possible but heavy | First-class server-side filter |
| Map SDK helpers | None | None | Drop-in `attachToMap(mapboxgl.Map)` |
| Pricing for "1 driver = 1 message every 2s × 200 drivers × 8h" | Charges per message; gets expensive fast | Charges per concurrent + per message | Flat per-CCU; publisher writes free up to fair-use cap |
| Snapshot-on-subscribe (last known position of every entity in group) | DIY | DIY (presence-ish) | Built in |
| Free tier you can ship a demo on | 100 conn / 200k msg / mo | 200 conn | 200 conn, generous publish |

The wedge is **"you write less code AND your bill is predictable when a marker pings every 2s."**

---

## 2. Core Features — MVP vs. v1

### MVP (ship in 4–6 weeks)
- **Publish over HTTP** (`POST /v1/positions`, `POST /v1/positions/batch`). Auth via API key.
- **Subscribe over WebSocket**, scoped to one `group_id` per socket. Auth via short-lived JWT minted by REST `POST /v1/subscribe-token`.
- **Snapshot-on-connect**: on subscribe, server pushes the last-known position of every entity in the group within the last `retention_ttl` seconds.
- **Server-side bbox filter** sent at subscribe time, updatable mid-session via `update_filter` message.
- **Coalescing**: if an entity pings 5x in 500ms, subscribers receive one merged update.
- **JS/TS browser SDK** with `MapboxAdapter`, `MapLibreAdapter`, `LeafletAdapter`.
- **Dashboard**: create tenant, generate API key, list groups, see live CCU & write rate.
- **Hard CCU cap per tier** with a clean `429`-equivalent close code (`4290`).

### v1 (next 2–3 months, gated by demand)
- **Radius filter** (`?center=lat,lng&radius_m=500`) in addition to bbox.
- **Presence**: server emits `entity_join` / `entity_leave` events when an entity hasn't pinged for `presence_timeout`.
- **Historical playback**: `GET /v1/groups/{id}/history?from=...&to=...`, ndjson stream — plus a SDK helper that replays at Nx speed onto a map.
- **Native SDKs**: iOS (Swift) and Android (Kotlin) — initially thin wrappers over URLSession / OkHttp WebSocket.
- **Webhook fan-out** for server-side subscribers (Slack/Discord pings on geofence cross).
- **Geofences**: define a polygon, get a webhook when an entity crosses it.
- **Per-entity ACLs** ("driver Alice can only publish to her own driver_id").

Explicitly **not in scope ever**: routing/ETA computation, map tile hosting, raster tile cache.

---

## 3. System Architecture

### The big decision: Durable Objects vs. Go-on-Fly

Two viable architectures. Both can theoretically hit MVP. Pick one and commit; mixing is overhead.

#### Option A — Cloudflare Durable Objects with WebSocket Hibernation

```
                       publishers (HTTP)
                              |
                              v
   subscribers ---> Cloudflare Worker (router + auth)
        |                     |
        |                     v
        |        Durable Object per group_id (state + fan-out)
        |                     |   (hibernates between events;
        |                     |    socket count NOT billed while idle)
        |                     |
        +-----WebSocket-------+
                              |
                              v
          Turso (tenants, keys, groups metadata)
          KV / DO storage     (last-known position map)
          R2 (optional)       (history ndjson archives)
```

- Each `group_id` maps to one DO instance; the Worker hashes the group_id and `idFromName` routes there.
- DO holds an in-memory `Map<entity_id, {lat,lng,ts,meta}>` plus the set of subscriber WebSockets.
- **WebSocket Hibernation API** lets the DO be evicted from memory while sockets stay open at the edge; only an inbound message wakes it. This is the killer feature: 10k idle subscribers cost effectively nothing.
- Bbox filter evaluated in the DO before pushing per-subscriber.
- Snapshot built from the DO's in-memory map on subscribe.

**Failure modes on free tier:**
- Workers free plan: 100k requests/day, 10ms CPU/request. WebSocket *messages* count as requests — 200 subscribers each receiving 1 msg/sec = 200 RPS = ~17M/day = vastly over free. **You will need the $5/mo Workers Paid plan from day one of any real load.** The free tier here is for demos, not production.
- DO storage: 1GB on paid plan. Fine for last-known maps. History to R2.
- Cold-start on first message after hibernation: ~50–200ms. Acceptable.
- A single hot group (10k subscribers) all hits one DO — DO is single-threaded JS. ~5–10k msgs/sec broadcast is the realistic ceiling per DO before fan-out latency creeps. Mitigation: shard hot groups into `group_id#shard_n` rooms with a Worker that round-robins subscribers; publishers fan out to all shards.

#### Option B — Go server on Fly.io with hub-and-spoke fan-out

```
                       publishers (HTTP / WS)
                              |
                              v
                    Fly.io Go service (hub)
                    nhooyr/websocket
                    in-process: groups[group_id] -> *Hub
                                Hub: subs map, lastKnown map
                              |
                              v
              Turso (metadata)        Cloudflare KV (optional cache)
```

- Single (or 2–3) `shared-cpu-1x 256MB` Fly VMs running a Go binary.
- One in-process map of `group_id -> *Hub`; each Hub owns its subscribers and a `lastKnown` map.
- Use `nhooyr.io/websocket` (cleaner API, context-friendly, lower allocs than gorilla; default for new code).
- Coalescer goroutine per entity drops mid-window pings.
- Horizontal scaling requires a fan-out bus (NATS, Redis pub/sub) — but free Fly + free NATS Cloud or a self-hosted NATS sidecar buys you a long way.

**Failure modes on free tier:**
- Fly's free Hobby plan was reduced in 2024 — you now get a $5/mo trial credit, not 3 free VMs. **Realistically you will pay ~$3–8/mo for one always-on VM.** Call this honestly; it's not free, but it's tiny.
- 256 MB RAM caps you at roughly 50k–80k idle WS connections (Go WS lib default ~3–5KB per conn with tuning). One DDoS spike OOMs you.
- Single VM = single point of failure. Fly auto-restart helps but you'll drop sockets.
- Cross-region: Fly multi-region is great but state lives in one VM unless you add a bus.
- Bandwidth: Fly free egress is 160GB/mo across regions. 1k subs × 100 bytes × 2 msg/s × 86400s × 30d ≈ 500 GB/mo. **You will blow this on a single popular customer.** Egress is the silent killer.

#### Recommendation: **Option A (Durable Objects + Hibernation) for MVP.**

Reasons, in order:
1. **Hibernation is the only thing that makes a free-ish CCU-priced product economically possible.** Idle subs cost nothing; you only pay when something happens.
2. **Cloudflare egress is free.** This is the single largest hidden cost in real-time infra and CF gives it away. Fly does not.
3. **No ops.** No VMs, no health checks, no region replication, no NATS.
4. **Sharding story is straightforward** (DO-per-group is the design's natural unit).
5. **You will outgrow Workers' CPU/request quotas** before DO concurrency limits, and that is a "good problem" — by then you have revenue.

Keep Fly.io as the **escape hatch**: if a tenant has needs DOs cannot meet (e.g., 100k subs in one group with binary protocol), spin up a dedicated Go service for them. Koyeb is the same shape if Fly pricing shifts.

For the rest of this plan: **architecture = DO + Hibernation, with a thin Go publisher-ingest sidecar on Fly only if HTTP ingest hits Worker CPU limits.**

---

## 4. Data Model

Turso (libSQL / SQLite at the edge). Five tables.

```sql
-- tenants
CREATE TABLE tenants (
  id              TEXT PRIMARY KEY,         -- ULID
  email           TEXT NOT NULL UNIQUE,
  plan            TEXT NOT NULL DEFAULT 'free',
  ccu_cap         INTEGER NOT NULL DEFAULT 200,
  write_rps_cap   INTEGER NOT NULL DEFAULT 100,
  created_at      INTEGER NOT NULL
);

-- api_keys (publisher keys)
CREATE TABLE api_keys (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  prefix          TEXT NOT NULL,            -- 'pk_live_abc...' first 8 chars, for UI
  hash            TEXT NOT NULL,            -- argon2id of the secret
  scopes          TEXT NOT NULL,            -- 'publish' | 'admin' | 'publish:group:xyz'
  last_used_at    INTEGER,
  revoked_at      INTEGER,
  created_at      INTEGER NOT NULL
);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);

-- groups
CREATE TABLE groups (
  id              TEXT NOT NULL,            -- tenant-chosen, e.g. 'fleet-london-1'
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  retention_ttl   INTEGER NOT NULL DEFAULT 60,    -- seconds for last-known map
  history_enabled INTEGER NOT NULL DEFAULT 0,     -- 0/1 (v1 feature)
  history_days    INTEGER NOT NULL DEFAULT 0,
  created_at      INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, id)
);

-- positions_history (v1; partitioned by day in R2 ndjson, this is just an index)
CREATE TABLE history_chunks (
  tenant_id       TEXT NOT NULL,
  group_id        TEXT NOT NULL,
  day             TEXT NOT NULL,            -- 'YYYY-MM-DD'
  r2_key          TEXT NOT NULL,
  bytes           INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, group_id, day)
);

-- usage_buckets (rolled up every minute by a cron Worker for billing)
CREATE TABLE usage_buckets (
  tenant_id       TEXT NOT NULL,
  bucket_minute   INTEGER NOT NULL,         -- unix minute
  peak_ccu        INTEGER NOT NULL,
  publish_count   INTEGER NOT NULL,
  egress_bytes    INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, bucket_minute)
);
```

**Position state lives in DO storage, not Turso.** Each DO holds:

```ts
// in-memory + persisted to DO storage every N seconds
type EntityState = {
  entity_id: string;
  lat: number; lng: number;
  ts: number;            // server-stamped on receive
  meta?: Record<string, unknown>;  // capped at 256 bytes
  cell_h3_r9?: string;   // pre-computed, see §7
};
type GroupState = {
  entities: Map<string, EntityState>;  // expired by retention_ttl on read
};
```

**Connections** are not persisted; the DO knows its open sockets via the Hibernation API's `getWebSockets()` and that's the source of truth for CCU. Aggregate is reported to a `MetricsDO` (one per tenant) every 30s.

---

## 5. Wire Protocol

JSON for MVP (debuggability, browser-friendly). Optional msgpack frame in v1 for native SDKs (`Sec-WebSocket-Protocol: rgeo.msgpack.v1`). All numbers are JSON numbers; lat/lng are degrees, ts is ms-since-epoch.

### Subscribe handshake
```
GET /v1/groups/{group_id}/ws?token=<jwt>&bbox=51.50,-0.13,51.52,-0.10&snapshot=true
Upgrade: websocket
```
On open, server sends:
```json
{ "t": "hello", "session": "01HX...", "server_time": 1745539200000, "heartbeat_s": 25 }
```
Then (if `snapshot=true`):
```json
{ "t": "snapshot", "entities": [
  {"e":"driver-1","lat":51.510,"lng":-0.118,"ts":1745539199000,"meta":{"plate":"AB12 CDE"}},
  {"e":"driver-7","lat":51.511,"lng":-0.121,"ts":1745539198500}
]}
```

### Server -> client messages
```json
{ "t": "pos", "e": "driver-1", "lat": 51.5101, "lng": -0.1178, "ts": 1745539201000 }
{ "t": "pos_batch", "items": [ ... ] }                  // when coalescer groups
{ "t": "leave", "e": "driver-1", "ts": 1745539250000 }  // presence (v1)
{ "t": "ping", "ts": 1745539225000 }                     // server heartbeat
{ "t": "error", "code": "RATE_LIMITED", "msg": "..." }
```

### Client -> server messages
```json
{ "t": "pong", "ts": 1745539225100 }
{ "t": "update_filter", "bbox": [51.49,-0.14,51.53,-0.09] }
{ "t": "update_filter", "center": [51.51,-0.12], "radius_m": 800 }   // v1
{ "t": "ack", "session": "01HX...", "up_to_ts": 1745539201000 }      // optional flow control
```

### Publish (HTTP)
```http
POST /v1/positions
Authorization: Bearer pk_live_xxx
Content-Type: application/json

{ "group_id":"fleet-london-1", "entity_id":"driver-1",
  "lat":51.5101, "lng":-0.1178, "ts":1745539201000,
  "meta":{"battery":0.83} }
```
Returns `204 No Content` on accept. Errors: `400` malformed, `401` bad key, `403` scope, `429` over write quota.

Batch: `POST /v1/positions/batch` with `{items:[...]}`, max 1000 per request. The Worker explodes the batch to the right DOs in parallel.

### Heartbeats
Server pings every 25s. Client must reply `pong` within 10s or socket is closed `4408`. Client SDK does this automatically.

### Close codes
- `4400` bad protocol
- `4401` auth expired (JWT)
- `4408` heartbeat timeout
- `4290` tenant CCU cap reached (the close *reason* contains plan + upgrade URL)
- `4429` per-conn rate limit
- `4503` DO migrating, please reconnect

---

## 6. SDKs

### `@rgeo/js` — browser & Node
The whole pitch lives in this SDK. It must be small (<15KB gzipped, no map lib dep), tree-shakable, and have map adapters as separate sub-paths.

```ts
import { RgeoClient } from "@rgeo/js";
import { attachToMapbox } from "@rgeo/js/mapbox";

const client = new RgeoClient({
  tokenEndpoint: "/api/rgeo-token",   // your backend mints JWTs
  // OR ephemeralToken: "..." for static demos
});

const sub = client.subscribe("fleet-london-1", {
  bbox: map.getBounds().toArray().flat(),
  snapshot: true,
});

attachToMapbox(sub, map, {
  layerId: "drivers",
  iconImage: "van-15",
  smoothMs: 600,            // CSS-like interp between updates
  removeAfterMs: 30_000,
});

map.on("moveend", () => sub.updateFilter({ bbox: map.getBounds().toArray().flat() }));
```

Implementation notes:
- Auto-reconnect with jittered backoff (250ms, 500, 1s, 2s, 5s, capped, fresh JWT each time).
- Marker pool — never call `addLayer` per entity; one symbol layer + GeoJSON source updated on rAF.
- `smoothMs`: linear-interpolate marker position toward target; if a new update arrives mid-tween, retarget. Avoids the strobe-effect of teleporting markers.
- Backpressure: if browser tab is hidden, client sends `update_filter` with bbox 0,0,0,0 (effective pause) and re-subscribes on `visibilitychange`.

### `@rgeo/go` — Go publisher SDK (priority for MVP)
```go
client := rgeo.New("pk_live_xxx")
client.Publish(ctx, rgeo.Position{
    Group: "fleet-london-1", Entity: "driver-1",
    Lat: 51.51, Lng: -0.118,
})
// Internally batches up to 100 positions or 50ms, whichever first.
```
This is the SDK ride-share/delivery backends will actually use. Must do batching + retry + circuit breaker out of the box.

### iOS / Android — defer to v1
URLSessionWebSocketTask and OkHttp ship with WebSockets. The pain isn't connecting; it's background execution rules, especially on iOS where you cannot keep a socket open in background reliably. v1 SDKs should default to **HTTP push for publishing** (cheap, works in background tasks via BGProcessingTask) and only use WS for subscribe (foreground UI).

---

## 7. Geo Features — Server-side Filtering

The MVP filter is **bbox** because it matches what every map library hands you (`map.getBounds()`). Implementation in DO:

```ts
function inBbox(p: EntityState, b: [number, number, number, number]) {
  return p.lat >= b[0] && p.lat <= b[2] && p.lng >= b[1] && p.lng <= b[3];
}
// On each pos update, iterate sockets, push only to those whose bbox contains p.
```

For groups with <1k subs, naive iteration is fine (CPU dominated by JSON serialization, not the comparison). Above that we need an index.

### Indexing approach: H3 cells, not raw quantization

H3 (Uber's hex grid) at resolution 9 (~150m hexes) gives:
- Each EntityState gets a `cell` string on receive.
- DO maintains `Map<cellId, Set<socketId>>` — sockets register interest in the cells covering their bbox.
- On pos update: lookup the cell's interested sockets, push only to them. Constant-ish work per update.

**Why H3 over simple lat/lng quantization (e.g., 0.01-degree grid):**
- H3 cells have near-uniform area worldwide; lat/lng grid cells stretch toward poles. London is fine; Reykjavik isn't.
- Bbox-to-H3 set conversion (`gridDisk`/`polygonToCells`) is a one-shot computation when subscriber's bbox changes — cheap.
- Radius filter (v1) is `gridDisk(centerCell, kRingForRadius)` — trivially built on the same index.
- The `h3-js` library is ~50KB but tree-shaken to ~15KB if you only import the cell ops you need. Within Workers' CPU budget.

Quantization is simpler but I'd rather pay the H3 dependency tax once than regret it the moment radius filtering ships.

### Retention TTL on the snapshot
On subscribe, snapshot filters out entities older than `group.retention_ttl` (default 60s). This is what makes "show me everyone *currently* in the group" cheap and stops a stale dot from haunting the map for hours.

---

## 8. Backpressure & Rate Limits

The threat model: "1000 taxis pinging every 100ms = 10k writes/sec into one group, fanning out to 5k subscribers = 50M outgoing msg/sec." That kills any free-tier infra and most paid-tier wallets. Three coupled defenses:

### (a) Server-side coalescing window per entity
Inside the DO, each entity has a `pendingFlush` timer:
```
on publish(entity_id, lat, lng):
  state[entity_id] = {lat, lng, ts}
  if !pendingFlush[entity_id]:
    pendingFlush[entity_id] = setTimeout(flush, COALESCE_MS)  # 250ms default

flush(entity_id):
  fanout(state[entity_id])
  delete pendingFlush[entity_id]
```
Effect: regardless of whether a driver pings every 100ms or every 2s, subscribers get at most 4 updates/sec per entity. `COALESCE_MS` is per-group configurable (down to 50ms on paid plans, up to 2000ms on free).

### (b) Per-tenant write rate limit
Token-bucket in Turso (or a `RateLimitDO` per tenant): `write_rps_cap` from the tenants row. Excess writes return `429` with `Retry-After`. Free tier: 100 RPS aggregate. Paid: 1k–10k.

### (c) Per-subscriber send buffer with shedding
DO tracks bytes queued per WebSocket. If `bufferedAmount > 256KB` (slow client), apply lossy shedding — drop intermediate `pos` messages for the same `entity_id`, only keeping the latest. If the buffer crosses 1MB, close the socket with `4290` and let the client reconnect. This is critical: one slow subscriber on a phone with bad signal must not back-pressure the whole DO event loop.

Publish-side floods cannot crash subscribers and subscribe-side slowness cannot crash publishers. That separation is the whole game.

---

## 9. Pricing & Metering

### Tiers (illustrative; tune after first 20 customers)

| Tier      | CCU cap | Write RPS | Coalesce min | History | Price |
|-----------|---------|-----------|--------------|---------|-------|
| Free      | 200     | 100       | 1000ms       | none    | $0    |
| Hobby     | 2,000   | 1,000     | 250ms        | 24h     | $19/mo |
| Team      | 10,000  | 5,000     | 100ms        | 7d      | $79/mo |
| Business  | 50,000  | 20,000    | 50ms         | 30d     | $299/mo |
| Custom    | >50k    | negotiate | 50ms         | custom  | quote |

CCU is the headline metric because that's what the customer sees in their map ("how many viewers?"); writes are bundled because they're correlated and capping both confuses pricing pages.

### How to count CCU accurately on a free stack
On every `accept` and `close` in the DO, push a delta to a per-tenant `MetricsDO`. Every 30s, the MetricsDO writes `{tenant_id, t, current_ccu}` to Turso (rounded to second). The billing job computes peak CCU per minute, then 95th-percentile peak across the month — that's the billed metric. P95-of-peaks is forgiving of brief spikes (a customer's marketing push) and matches what AWS / Cloudflare itself does for many products.

### Boundary behavior
**Free and Hobby: hard cap.** The 201st free-tier subscriber is rejected at WS upgrade with HTTP 429 (and a JSON body pointing to the upgrade URL). Better to let the dev see the wall on day one than to silently rack up an unexpected bill.

**Team and above: soft cap with overage.** Allow 20% over for 60 minutes, then start dropping new connections. Email the customer at 80%, 100%, 120%. Bill overage at the next-tier per-CCU rate, prorated. This is what mature customers expect.

**Never hard-disconnect existing connections** to come under cap. That's catastrophic UX and they won't trust you again. Caps apply to *new* upgrades only.

---

## 10. Auth & Multi-tenancy

### Two credential types

**API keys (long-lived, server-side only)**
- Format: `pk_live_<26 char base32>` (tenant_id-derived prefix not in secret).
- Stored as argon2id hash. Verified at the edge Worker by comparing hash; key body is never logged.
- Scopes: `publish`, `admin`, optionally `publish:group:<id_pattern>` (glob).
- Rotation: each tenant can have N keys, can revoke individually.

**Subscribe JWTs (short-lived, browser-safe)**
- Customer's backend hits `POST /v1/subscribe-token` with their API key, body `{group_id, ttl_s, allowed_filters?}`.
- We return `{token: "<JWT>", expires_at}`. JWT is HS256 with a per-tenant signing secret rotated every 7 days (dual-key window for verification).
- Browser passes JWT in WS query string (`?token=`) or `Sec-WebSocket-Protocol`.
- TTL default 5 min; on token expiry the SDK auto-fetches a new one from the customer's `tokenEndpoint`.

This split means **API keys never reach the browser**. A leaked JWT is bad for 5 minutes for one group; a leaked API key is bad until rotated.

### Multi-tenancy isolation
- Group IDs are namespaced by tenant: the DO ID is `idFromName(tenant_id + "::" + group_id)`. Cross-tenant collision is impossible by construction.
- All Turso queries are parameterised on `tenant_id` (not RLS — Turso doesn't have it; enforced in the data access layer with a single `withTenant(ctx)` helper that wraps all queries).
- Logs / traces tagged with `tenant_id`; never log full keys or JWTs (only key prefix + last 4).

---

## 11. Observability

### Per-tenant dashboard (what the customer sees)
- Live CCU, with 24h sparkline.
- Publish rate (msgs/sec), with deliver rate, with delta = "messages dropped to coalescing."
- Top groups by CCU and by write rate.
- Recent errors (auth, rate-limit) with timestamps and offending API key prefix.

### Internal (what we see)
- Per-DO: msg/sec in, msg/sec out, current sockets, hibernation hit rate, p50/p95/p99 fan-out latency.
- Per-tenant: CCU, write RPS, egress bytes, error rate by code.
- Global: Worker requests/day vs. cap, DO storage GB, Turso query count, R2 egress.

### How to actually collect this on a free stack
- Cloudflare Analytics Engine for high-cardinality timeseries (per-DO, per-tenant). Free tier is 10M data points/day; we sample at 30s intervals.
- Logs: `tail` Workers logs into a Worker -> R2 nightly archive. Don't pay for Logpush until revenue.
- Customer-facing dashboard: a Pages app querying Analytics Engine via a thin Worker proxy.
- Alerting: simple cron Worker every minute checks "any tenant >90% of cap?" and "any DO with >1MB queue?" — sends email via Resend (free tier 3k/mo) or a Discord webhook.

---

## 12. Tech Stack & Libraries

| Layer | Choice | Reason |
|---|---|---|
| Edge / WS termination | Cloudflare Workers + Durable Objects | Hibernation, free egress |
| DO language | TypeScript (Workers runtime) | First-class DO API |
| Map indexing | `h3-js` (tree-shaken) | See §7 |
| Hashing | `@noble/hashes` (argon2id) | Pure JS, runs in Workers |
| JWT | `jose` | Battle-tested, Workers-compatible |
| DB | Turso (libSQL) | Edge replicas, generous free tier |
| DB driver | `@libsql/client` | Official |
| Object storage | R2 | Free egress, history archives |
| Go publisher sidecar (if needed) | Fly.io + `nhooyr.io/websocket` | Cleaner than gorilla, ctx-native, fewer footguns. Use gorilla only if a customer needs deflate-extension compat. |
| Marketing site | Astro on Pages | Static, fast |
| Live demo on landing page | MapLibre + a fake-driver Worker that publishes 50 entities driving around London | Instant "wow" |
| Billing | Stripe | Defer until Hobby tier exists |
| Auth (dashboard) | Stack Auth or Clerk free tier | Don't roll your own for the dashboard |
| Email | Resend | Onboarding + alerts |

---

## 13. Build Roadmap (6 weeks, solo)

**Week 1 — skeleton & ingest.**
- Cloudflare account, Pages project, Workers project, Turso DB, R2 bucket.
- Tenants/api_keys tables, signup + key issuance flow.
- `POST /v1/positions` happy path (no DO yet; just validates auth, writes to a stub).
- Postman/curl works end-to-end with auth.

**Week 2 — Durable Object & basic fan-out.**
- `GroupDO` with in-memory entity map + WS accept/close.
- Worker routes WS upgrade to the right DO.
- Subscribe handshake, snapshot, bare position fan-out (no filtering yet).
- Hibernation API wired up and verified (deploy, leave 1k mock subs idle overnight, check billing).

**Week 3 — bbox filter, coalescing, JS SDK v0.**
- H3 cell indexing in DO.
- Coalescing timer per entity.
- `@rgeo/js` v0 published to npm; works with Mapbox.
- Live demo on marketing site — 50 fake drivers in central London.

**Week 4 — auth hardening, JWT, dashboard.**
- Subscribe JWT issuance endpoint.
- Pages dashboard: signup, key management, live CCU graph, group list.
- Per-tenant write rate limiter (RateLimitDO).
- Stripe checkout for Hobby tier.

**Week 5 — coalescing window per group, backpressure shedding, hard caps.**
- Per-socket buffered-bytes shedding.
- CCU cap enforcement at WS upgrade (read tenant cap, count current sockets in DO via a Tenant->Group fanout query, reject if over).
- MetricsDO + p95-peak billing math.
- Status page (Cloudflare's free).

**Week 6 — Go SDK, docs, launch.**
- `@rgeo/go` with batching/retry.
- Docs site (Astro): quickstart, protocol reference, SDK reference, recipes (Mapbox, Leaflet, Next.js).
- Beta launch: 5–10 invited users from IndieHackers + Discord communities. Free Hobby for early adopters in exchange for testimonials.

**Weeks 7–8 (buffer / v1 starts.)**
- Radius filter.
- Presence (`leave` events).
- History to R2, replay endpoint and SDK helper.
- iOS/Android SDK scoping.

If Week 2's hibernation test isn't clean, *stop and fix it before anything else*. The whole economic model assumes idle sockets are nearly free. If they're not, fall back to the Fly.io plan and re-cost.

---

## 14. Free-tier Risk & Scaling Triggers

### Concrete break-points

**Cloudflare Workers Paid plan ($5/mo) needed at:**
- ~100 daily-active subscribers, since each WS message counts as a request and a few hundred subs at 1 msg/sec exhausts 100k req/day.
- Treat this as table stakes; don't try to ship on the free Workers plan.

**DO request quota under Workers Paid (10M/day included, then $0.15/M):**
- 10M req/day = ~115 req/sec sustained. With coalescing at 250ms = 4 deliveries/sec/entity, fan-out of 1 entity to 30 subs = 120 messages = ~120 requests. So ~1 entity-with-30-subs of activity sustained.
- Practically: **first material bill kicks in around 500 concurrent active markers being watched.** Budget ~$30–80/mo at 5k CCU in a typical fleet pattern. Pricing in §9 covers this with 30–60% margin.

**DO storage:** trivial. Last-known maps fit in a few MB per tenant. No worry until 100k+ groups.

**Turso:** free up to 9GB / 1B reads/mo. Won't hit unless you're dumping every position into history (don't — use R2). Trigger to upgrade: 500 paying tenants.

**R2:** free up to 10GB / 1M Class A ops. History storage will blow this around 50 paying customers with `history_enabled`. R2 is cheap at scale ($0.015/GB/mo), so just budget for it.

**Fly.io fallback:** if a single customer needs sustained >5k msg/sec into one group with <50ms fan-out latency, DO single-thread becomes the bottleneck. Migrate that *one customer* to a dedicated Go service on Fly. Charge them enterprise pricing to fund it.

### Signals to act on (alerts)
- Worker requests >70% of monthly cap by mid-month -> review tenants, hunt the noisy one.
- Any DO p99 fan-out latency >100ms sustained -> shard that group.
- Any single tenant >40% of total egress -> talk to them about enterprise plan.
- Turso row reads >500M/mo -> check for missing indexes on usage_buckets.

---

## 15. Go-to-market

### Positioning
**"Pusher for things on a map. Free for hobby projects, predictable pricing for production."**

Don't pitch as "real-time database" — that's a Supabase/Firestore battlefield. Pitch as the *map layer for real-time*, where competitors don't speak the geo dialect.

### Channels
- **dev.to / Hashnode tutorials**, in this order:
  1. "Show 100 drivers on a Mapbox map in 50 lines of React" (the SDK demo as a viral piece)
  2. "Why your live tracking app's bill explodes (and how to cap it)" — the CCU pricing pitch
  3. "Building a hackathon-grade .io game in 200 lines with Cloudflare DOs" — for the indie-game niche
  4. "GDPR-safe live location: 60-second retention, no historical store" — for the enterprise-curious
- **IndieHackers launch post**, Show HN. Lead with the live demo, not the pricing page.
- **SEO target keywords**: "pusher alternative geo," "ably alternative cheaper," "real-time map sdk," "show moving markers mapbox sdk," "websocket geofence service."
- **Sponsor a small map-related dev YouTuber** ($200 one-shot) once Hobby tier is stable.
- **Mapbox / MapLibre / Leaflet plugin pages** — list the SDK as a community plugin. Free distribution.

### Pricing wedge against Pusher / Ably
On Pusher's Startup plan ($49/mo) you get 500 connections and 1M messages. A 200-driver fleet pinging every 2s eats that in 2.5 hours. Our equivalent tier is $19/mo with 2k CCU and effectively unlimited messages (because we coalesce and we don't bill on messages). Lead with this comparison in a pricing-page table — it's a 5x cost wedge.

### First 10 customers playbook
- Cold email the founders of small UK delivery startups (Bolt food clones, last-mile couriers). London has dozens. Offer 6 months Team-tier free for case study + logo.
- Post in `r/gamedev`'s "I made an .io game" threads with the offer "free hosting for your real-time map layer."
- Show up at hackathons with stickers and a Discord link.

---

## 16. Open Questions / Risks

### Privacy & legal — the main risk
We are, by definition, building **infrastructure for tracking the live location of people**. This is a regulatory and reputational minefield.

**Decisions to bake in from day one:**
1. **Default retention TTL is 60 seconds.** No history for free tier. You have to explicitly opt into history per-group, and the dashboard nags about lawful basis.
2. **No covert surveillance use case.** Terms of service forbids "tracking individuals without their explicit, informed consent" and "any use to track minors without parental consent." Violations = ban + refund. This won't stop bad actors but it makes our position clear.
3. **Refuse stalkerware integrators.** Watch signups for app names matching known stalkerware patterns. Have a policy. Have a kill switch. Publish a transparency report annually once we have customers.
4. **Refuse domestic-violence-adjacent use.** "Find my partner" apps — no. "Find my kid" apps targeted at adult partners — no. We will lose deals over this. Worth it.
5. **GDPR compliance basics:**
   - We are a processor; the customer is controller. Standard DPA template.
   - EU-resident data stays in EU CF colos (DO `locationHint: "weur"`).
   - Right-to-erasure: customer can call `DELETE /v1/groups/{id}/entities/{eid}` and we purge from DO + R2 history within 24h.
   - No tracking-tracking — our SDK does not send analytics back to us beyond aggregate CCU.
6. **UK Online Safety Act:** we are not user-facing; we don't host UGC. We are infra. But customers building consumer apps on top of us may be in scope; document this clearly.

### Other risks

- **Abuse vector — DDoS via free-tier signups.** A flood of free accounts pinning Workers CPU. Mitigation: aggressive signup rate-limiting per IP/email-domain; required email verification before any API key is issued; Turnstile on signup form.
- **Workers/DO pricing changes.** Cloudflare has changed Workers pricing twice in two years. Build a thin abstraction over the DO so a Fly.io fallback is a 2-week port, not a rewrite.
- **Mapbox / MapLibre licensing** for the marketing demo. MapLibre is OSS, use it for the demo to avoid Mapbox's "100k loads/mo free, then per-load" tier biting us if a tutorial goes viral.
- **Single-DO hot-spot.** A customer with a viral event (200k people viewing one group) can saturate one DO. Need an "auto-shard" feature in v1: when CCU on one DO crosses 5k, transparently split into N replica DOs with a fan-out fronting Worker. Design it now even if we don't ship it now.
- **Browser tab behavior.** Mobile Safari throttles WebSockets aggressively in background. Document this, and have the SDK detect and re-snapshot on `visibilitychange`. Otherwise users will report "the map froze when I switched apps" and blame us.
- **Time sync.** Server-stamped `ts` is authoritative; we ignore client-supplied `ts` for ordering (only as a hint). Document this clearly because publishers will ask.
- **Cost of being wrong about hibernation economics.** If real-world hibernation behaves worse than docs suggest (e.g., wakeups too frequent in practice), the unit economics break. Week 2 milestone exists specifically to validate this with a real soak test. Don't skip it.

---

*End of plan. Build this in 6 weeks. Ship the demo, write the dev.to post, charge the second customer.*
