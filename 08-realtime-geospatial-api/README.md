# Realtime Geospatial API MVP

Cloudflare Workers + Durable Objects scaffold for `docs/03-realtime-geospatial-api.md`.

## What Is Included

- Worker routes:
  - `POST /v1/positions`
  - `POST /v1/positions/batch`
  - `POST /v1/subscribe-token`
  - `GET /v1/groups/:group_id/ws?token=...&bbox=minLat,minLng,maxLat,maxLng`
  - `GET /dashboard` and `GET /dashboard/api/summary` dashboard stubs
- `GroupHub` Durable Object:
  - one hub per `tenant_id::group_id`
  - in-memory last-known position map with DO storage persistence
  - snapshot-on-connect
  - bbox filtering and `update_filter`
  - per-entity coalescing window
- SQLite/Turso migration for tenants, API keys, groups, history chunks, and usage buckets.
- Browser SDK skeleton with Mapbox, MapLibre, and Leaflet adapters.
- Vitest tests that run without Cloudflare, Turso, or other external services.

## Local Development

```sh
npm test
npm run typecheck
npm run dev
```

The default `wrangler.toml` includes a demo API key:

```text
pk_live_demo
```

Publish a position:

```sh
curl -i http://localhost:8787/v1/positions \
  -H 'authorization: Bearer pk_live_demo' \
  -H 'content-type: application/json' \
  -d '{"group_id":"fleet-london-1","entity_id":"driver-1","lat":51.5101,"lng":-0.1178}'
```

Mint a browser-safe subscribe token:

```sh
curl -s http://localhost:8787/v1/subscribe-token \
  -H 'authorization: Bearer pk_live_demo' \
  -H 'content-type: application/json' \
  -d '{"group_id":"fleet-london-1","ttl_s":300}'
```

## Production Notes

This scaffold intentionally keeps external services optional for local tests. Before production:

- Replace `DEMO_API_KEYS` with Turso-backed API key lookup and hashed secret verification.
- Add a per-tenant `RateLimitDO` for write RPS caps.
- Wire `usage_buckets` and dashboard cards to Analytics Engine/Turso.
- Add tenant-level CCU accounting before accepting WebSocket upgrades.
- Move `SUBSCRIBE_JWT_SECRET` to a Cloudflare secret.

## SDK Sketch

```ts
import { RgeoClient } from "./sdk";
import { attachToMapbox } from "./sdk/mapbox";

const client = new RgeoClient({ tokenEndpoint: "/api/rgeo-token" });
const sub = client.subscribe("fleet-london-1", {
  bbox: [51.50, -0.13, 51.52, -0.10],
  snapshot: true
});

attachToMapbox(sub, map, {
  layerId: "drivers",
  iconImage: "van-15"
});
```
