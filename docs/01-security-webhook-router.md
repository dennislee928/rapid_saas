# 01 — Security Alert / Webhook Router Middleware

A solo-foundable SaaS implementation plan. Concrete, opinionated, and tuned for a zero-cost initial deploy on Cloudflare Workers + Fly.io + Turso. Target time-to-MVP: 4–6 weeks part-time.

---

## 1. Product Overview & Target Users

**One-liner.** A multi-tenant webhook router that gives every customer a dedicated ingress URL. Inbound payloads from SIEMs, EDR agents, threat intel feeds, status pages, and CI systems are parsed, filtered, transformed, and fanned out to Slack, Discord, PagerDuty, another webhook, or a SOAR/SIEM (Splunk HEC, Elastic, Sentinel).

**Job-To-Be-Done.** *"When a security/ops tool fires an event, help me normalize and route it to the right channel without writing or maintaining a glue script."*

**Target users (MVP wedge → expansion):**

1. **MSSPs and 1–5 person SecOps teams at SMBs (the wedge).** They juggle CrowdStrike/SentinelOne/Wazuh/Defender alerts and want them in Slack with a clean format and IP/hash-based suppression. They don't have a Tines/Torq budget (Tines starts at "contact us", real-world ≥ £1k/mo).
2. **Indie SREs / DevOps consultants** who manage status pages, GitHub webhooks, Sentry, UptimeRobot, and need fan-out + de-dupe.
3. **Compliance-conscious companies in the UK/EU** who want webhook payloads processed in-region with auditable delivery logs (a wedge against US-only competitors).

**Why pay vs. self-host?**

- Self-hosting an n8n/Node-RED instance costs ~$10–25/mo plus the operational tax of patching, secrets, and PII-in-logs governance.
- Tines/Torq/Workato are SOAR-grade and cost 10–100x more than this product.
- Zapier/Make.com don't reliably handle high-volume webhook ingest (5–15s lag, no HMAC verification, weak retry semantics).
- Our wedge: **per-call pricing, sub-100ms ingest acknowledgement at the edge, and signature-aware retry/DLQ** — the things ad-hoc scripts always get wrong.

---

## 2. Core Features (MVP vs. v1)

### MVP (ship in 4–6 weeks)

- Sign up → workspace → generate up to 3 ingress URLs (per environment).
- API key issuance with `whk_live_…` / `whk_test_…` prefix; rotate + revoke.
- Inbound webhook receiver at `https://in.<domain>/w/<endpoint_id>` returning 202 in <100ms.
- HMAC signature verification for the major presets: GitHub (`X-Hub-Signature-256`), Stripe-style, Slack-style, generic shared-secret.
- Routing rules with a small filter DSL (see §6) — JSONPath conditions + allow/deny lists.
- Outbound destinations: Slack incoming webhook, Discord webhook, generic HTTPS POST with custom headers, email (via Resend free tier).
- Templating with Go `text/template` + a curated helper set (`{{ .ip }}`, `{{ .severity | upper }}`).
- Delivery log retention: 7 days on free tier, with status (queued/delivered/failed/dropped).
- Retry policy: exponential backoff 5 attempts (30s, 2m, 10m, 1h, 6h), then DLQ.
- Dashboard: endpoint list, last 100 events per endpoint, retry/replay button.
- Stripe billing with a free tier and a single $19/mo Starter plan.

### v1 (weeks 7–14)

- More transformers: Microsoft Teams, PagerDuty Events v2, Splunk HEC, Elastic Bulk, generic Sentinel HTTP Data Collector.
- Pre-built parsers for CrowdStrike Falcon, SentinelOne, Wazuh, Sentry, GitHub Advanced Security, AbuseIPDB, OTX feeds.
- IP-list feature: managed allow/deny lists you can subscribe to (e.g. Spamhaus DROP, Tor exit nodes via maintained mirror).
- Replay-from-log + bulk replay window.
- Per-endpoint per-destination rate limits ("max 1 Slack message per 30s for this rule, dedupe by `alert.id`").
- Team members / RBAC.
- Audit log + SOC2-friendly export (CSV/JSON).
- Custom domain ingest (`hooks.acme.com` CNAME → Worker).

### Explicit non-goals (for sanity)

- No GUI workflow builder (Tines clone). Rules are config + small DSL.
- No long-running stateful workflows. Fire-and-forget routing only.
- No PII redaction service in MVP — surface as v1.5.

---

## 3. System Architecture

```
                  ┌──────────────────────────────────────────────┐
                  │              CUSTOMER SIDE                    │
                  │   CrowdStrike, Wazuh, GitHub, Sentry, etc.    │
                  └────────────────────┬─────────────────────────┘
                                       │ HTTPS POST
                                       ▼
        ┌──────────────────────────────────────────────────────┐
        │  Cloudflare Worker  ──  in.<domain>/w/<endpoint_id>  │
        │  ─────────────────────────────────────────────────── │
        │  • Anycast ingress (sub-50ms globally)               │
        │  • Looks up endpoint_id in Workers KV (cached)       │
        │  • Verifies HMAC if configured                       │
        │  • Cheap rate-limit (Durable Object token bucket)    │
        │  • Pushes raw event to Cloudflare Queue              │
        │  • Returns 202 in <100ms                             │
        └────────────────────┬─────────────────────────────────┘
                             │ Cloudflare Queue (push)
                             ▼
        ┌──────────────────────────────────────────────────────┐
        │  Go service on Fly.io  (3 × 256MB shared-cpu, FRA/   │
        │  LHR/IAD; 1 primary writer, 2 secondaries readers)   │
        │  ─────────────────────────────────────────────────── │
        │  • Queue consumer (HTTP push from Workers)           │
        │  • Rule evaluation (JSONLogic engine)                │
        │  • Template render + HTTP fan-out                    │
        │  • Per-tenant outbound rate-limiter (in-mem + Turso) │
        │  • Retry scheduler (BoltDB on /data volume)          │
        │  • Writes delivery log + usage counter to Turso      │
        └────────┬─────────────────────────────────────────────┘
                 │                                  ▲
                 │ libsql (HTTP) over TLS           │ replay/admin
                 ▼                                  │
        ┌────────────────────────┐    ┌─────────────┴──────────┐
        │  Turso (libSQL)        │    │  Cloudflare Pages       │
        │  • tenants, api_keys   │    │  Next.js dashboard      │
        │  • endpoints, rules    │    │  Calls Go API on Fly.io │
        │  • delivery_log (TTL)  │    │  Auth: Clerk free tier  │
        │  • usage_counter       │    └────────────────────────┘
        │  Replicas in LHR/FRA   │
        └────────────────────────┘
                 │
                 ▼
        ┌────────────────────────┐
        │  Outbound destinations │
        │  Slack/Discord/Teams/  │
        │  PagerDuty/Splunk HEC  │
        │  /custom HTTPS         │
        └────────────────────────┘

                Backup/DR: Koyeb instance running same Go binary
                pulled by `flyctl deploy`-equivalent CI job.
```

### Why each component lives where

| Component | Why here | Free-tier headroom |
|---|---|---|
| **Cloudflare Worker (ingress)** | Webhooks are bursty and want sub-100ms ack. Workers run in 300+ POPs — far closer to the source than any single VM. Fronting Fly.io directly would 502 during cold-starts. | 100k requests/day free, 10ms CPU/request. We do almost no CPU work here — just HMAC + queue-push. |
| **Cloudflare Queue** | Decouples ingress from Go workers. Survives Fly.io reboots without dropping events. | Free tier: 1M ops/month. |
| **Go on Fly.io** | The CPU/memory work (rule eval, templating, retries, fan-out HTTP with timeouts) needs persistent connections and a real scheduler. Workers can't hold open 30+ idle keep-alives or do BoltDB-style local retry queues. Go's tiny binary fits in 256MB easily — a typical idle profile is 30–60MB RSS. | 3 × shared-cpu-1x 256MB + 3GB persistent volume free. |
| **Turso (libSQL)** | Edge-replicated SQLite. Reads from Workers (KV-style endpoint lookups) are <10ms. The 9GB storage + 1B reads/month free ceiling is enormous for a metering DB. | 9GB / 1B reads / 25M writes per month free. |
| **Cloudflare Pages (Next.js dashboard)** | Static hosting + edge-cached APIs at zero cost. | Unlimited requests on free tier. |
| **Koyeb (warm spare)** | Single-region passive deploy. If Fly.io is down, flip a Cloudflare Worker env var to point Queue consumer to Koyeb URL. | 1 free Web Service. |

---

## 4. Data Model

Schema is for libSQL/SQLite (Turso). Use `INTEGER` PKs and string ULIDs for external IDs to keep URLs short.

```sql
-- Tenants & users
CREATE TABLE tenant (
  id              TEXT PRIMARY KEY,                  -- ULID, e.g. 01HV...
  name            TEXT NOT NULL,
  plan            TEXT NOT NULL DEFAULT 'free',      -- free|starter|pro
  stripe_cust_id  TEXT,
  created_at      INTEGER NOT NULL                   -- unix ms
);

CREATE TABLE user (
  id              TEXT PRIMARY KEY,                  -- ULID
  tenant_id       TEXT NOT NULL REFERENCES tenant(id),
  email           TEXT NOT NULL UNIQUE,
  clerk_user_id   TEXT,                              -- if using Clerk
  role            TEXT NOT NULL DEFAULT 'owner',     -- owner|admin|member
  created_at      INTEGER NOT NULL
);

-- API keys: caller-side (used by customers calling our admin API)
CREATE TABLE api_key (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenant(id),
  prefix          TEXT NOT NULL,                     -- "whk_live_xxxxx" first 12 chars
  hash            TEXT NOT NULL,                     -- argon2id(secret)
  scopes          TEXT NOT NULL DEFAULT 'read,write',
  last_used_at    INTEGER,
  revoked_at      INTEGER,
  created_at      INTEGER NOT NULL
);
CREATE INDEX idx_api_key_prefix ON api_key(prefix);

-- Webhook ingress endpoints (one per "URL" the customer hands out)
CREATE TABLE endpoint (
  id              TEXT PRIMARY KEY,                  -- e.g. ep_01HV..., used in URL
  tenant_id       TEXT NOT NULL REFERENCES tenant(id),
  name            TEXT NOT NULL,
  source_preset   TEXT,                              -- 'github' | 'stripe' | 'crowdstrike' | 'generic'
  signing_secret  TEXT,                              -- for HMAC verification
  signing_header  TEXT,                              -- e.g. X-Hub-Signature-256
  signing_algo    TEXT,                              -- 'sha256' | 'sha1'
  enabled         INTEGER NOT NULL DEFAULT 1,
  created_at      INTEGER NOT NULL
);
CREATE INDEX idx_endpoint_tenant ON endpoint(tenant_id);

-- Outbound destinations (Slack URL, custom URL, etc.)
CREATE TABLE destination (
  id              TEXT PRIMARY KEY,                  -- dest_01HV...
  tenant_id       TEXT NOT NULL REFERENCES tenant(id),
  kind            TEXT NOT NULL,                     -- 'slack'|'discord'|'http'|'pagerduty'|'splunk_hec'
  name            TEXT NOT NULL,
  config_json     TEXT NOT NULL,                     -- {"url":"...","headers":{...}}
  secret_ref      TEXT,                              -- pointer to Fly secret/Doppler
  created_at      INTEGER NOT NULL
);

-- Rules: ordered chain attached to an endpoint.
-- Each rule: a filter (JSONLogic), a transform (template or preset), and a destination.
CREATE TABLE rule (
  id              TEXT PRIMARY KEY,                  -- rule_01HV...
  endpoint_id     TEXT NOT NULL REFERENCES endpoint(id) ON DELETE CASCADE,
  position        INTEGER NOT NULL,                  -- ordering
  name            TEXT NOT NULL,
  filter_jsonlogic TEXT,                             -- e.g. {"and":[{">":[{"var":"severity_int"},3]},...]}
  transform_kind  TEXT NOT NULL,                     -- 'passthrough'|'template'|'preset'
  transform_body  TEXT,                              -- Go template OR preset name
  destination_id  TEXT NOT NULL REFERENCES destination(id),
  on_match        TEXT NOT NULL DEFAULT 'forward',   -- 'forward'|'drop'|'continue'
  enabled         INTEGER NOT NULL DEFAULT 1,
  created_at      INTEGER NOT NULL
);
CREATE INDEX idx_rule_endpoint ON rule(endpoint_id, position);

-- Filter sets (e.g. blocked IP list shared across rules)
CREATE TABLE filter_list (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenant(id),
  name            TEXT NOT NULL,
  kind            TEXT NOT NULL,                     -- 'ip'|'cidr'|'string'|'regex'
  source          TEXT NOT NULL DEFAULT 'manual',    -- 'manual'|'spamhaus_drop'|'tor_exit'
  refreshed_at    INTEGER
);
CREATE TABLE filter_list_item (
  list_id         TEXT NOT NULL REFERENCES filter_list(id) ON DELETE CASCADE,
  value           TEXT NOT NULL,
  PRIMARY KEY (list_id, value)
);

-- Delivery log. Hot table; partitioned logically by day for TTL.
CREATE TABLE delivery_log (
  id              TEXT PRIMARY KEY,                  -- ULID time-sortable
  tenant_id       TEXT NOT NULL,
  endpoint_id     TEXT NOT NULL,
  rule_id         TEXT,
  destination_id  TEXT,
  status          TEXT NOT NULL,                     -- 'queued'|'delivered'|'failed'|'dropped'|'dlq'
  attempt         INTEGER NOT NULL DEFAULT 0,
  http_status     INTEGER,
  latency_ms      INTEGER,
  error           TEXT,
  request_hash    TEXT,                              -- sha256 of body, for dedupe
  request_size    INTEGER,
  received_at     INTEGER NOT NULL,                  -- unix ms
  delivered_at    INTEGER
);
CREATE INDEX idx_dlog_tenant_time ON delivery_log(tenant_id, received_at DESC);
CREATE INDEX idx_dlog_endpoint_time ON delivery_log(endpoint_id, received_at DESC);

-- Usage counter — bucketed per tenant per hour to keep writes cheap.
CREATE TABLE usage_counter (
  tenant_id       TEXT NOT NULL,
  bucket_hour     INTEGER NOT NULL,                  -- unix hour (received_at / 3600000)
  ingressed       INTEGER NOT NULL DEFAULT 0,
  forwarded       INTEGER NOT NULL DEFAULT 0,
  failed          INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, bucket_hour)
);

-- Dead-letter queue
CREATE TABLE dlq (
  id              TEXT PRIMARY KEY,                  -- ULID
  tenant_id       TEXT NOT NULL,
  endpoint_id     TEXT NOT NULL,
  rule_id         TEXT,
  destination_id  TEXT,
  payload_b64     TEXT NOT NULL,                     -- raw body, gzip+b64
  last_error      TEXT,
  attempts        INTEGER NOT NULL,
  parked_at       INTEGER NOT NULL
);
CREATE INDEX idx_dlq_tenant ON dlq(tenant_id, parked_at DESC);
```

Notes:
- Secrets (`signing_secret`, destination credentials) are stored encrypted with libsodium secretbox; the master key lives in a Fly.io secret (`KMS_KEY`). Never store plaintext in Turso.
- `delivery_log` is the only chunky table. Daily cron prunes rows older than retention window (7d free, 30d starter, 90d pro).

---

## 5. Request Lifecycle

Tracing one event end-to-end:

1. **POST hits the Worker.** `https://in.example.com/w/ep_01HVABC...`
2. **Endpoint lookup.** Worker reads `ep_01HVABC` from Workers KV (TTL 60s; falls back to libSQL HTTP fetch on miss). Result includes `tenant_id`, `enabled`, `signing_*`, plan limits.
3. **Body cap.** Reject `>1 MiB` with 413 (configurable, free tier 256 KiB).
4. **Rate limit.** Cheap token bucket on Durable Object keyed by `tenant_id` (e.g. 50 rps free, 500 rps starter). 429 if exceeded.
5. **HMAC verify.** If `signing_secret` is set, compute HMAC of body with `signing_algo`, constant-time compare to `signing_header`. 401 on mismatch.
6. **Enqueue.** Worker pushes `{tenant_id, endpoint_id, headers, body, received_at_ms, request_hash}` to Cloudflare Queue. Returns `202 Accepted` with a JSON `{ "id": "evt_01HV..." }`.
7. **Consumer wakes on Fly.io.** Worker→Queue→push HTTPS to `https://api.fly.dev/_q/consume` (mTLS via shared secret). Batch of up to 100 events.
8. **Tenant + plan re-check.** Read tenant record (cached in-memory 30s). If quota exceeded for this billing period: insert `delivery_log` row with `status='dropped', error='quota_exceeded'` and short-circuit.
9. **Fetch rule chain.** `SELECT * FROM rule WHERE endpoint_id = ? AND enabled=1 ORDER BY position`. Cache per endpoint with 30s TTL invalidated by an admin-API mutation hook.
10. **Iterate rules in order.**
    - Decode body as JSON; if not JSON, expose only `headers` + `raw` to filter.
    - Run `filter_jsonlogic` against the event. Library: `github.com/diegoholiveira/jsonlogic/v3`.
    - On match:
      - `drop`: write log row, stop chain.
      - `forward`: render `transform_body` (template or preset); enqueue outbound delivery; if `on_match='forward'` (default) stop, else `'continue'` walks remaining rules.
11. **Outbound delivery.** Per-destination worker pool (semaphore sized per tenant plan, 8 free, 32 starter). Use `net/http` with 10s timeout, follow no redirects, refuse 192.168/10/127/169.254 (SSRF guard). Capture `http_status`, `latency_ms`.
12. **Retry policy.** On 5xx / network error, schedule retry via local BoltDB-backed delay queue (`go.etcd.io/bbolt` keyed by `due_unix_ms`). Backoff: 30s, 2m, 10m, 1h, 6h with jitter ±20%.
13. **DLQ.** After 5 attempts, gzip the payload, base64 it, insert into `dlq`, write final `delivery_log` row with `status='dlq'`.
14. **Usage accounting.** *Single increment per event* on the hourly bucket: `INSERT INTO usage_counter(...) VALUES(...) ON CONFLICT DO UPDATE SET ingressed = ingressed + 1`. We additionally increment `forwarded`/`failed` once per outbound delivery, but always at the hour-bucket grain. This keeps writes <= 1 per event for ingress + N for outbound (typically N=1).
15. **Customer dashboard tail.** Dashboard polls `/v1/events?endpoint_id=…&since=…` every 5s when open; uses `delivery_log` indexes. (No SSE in MVP.)

Worst case latency budget for a delivered event: 80ms ack at edge + ~150ms median fan-out from Fly.io.

---

## 6. Filtering & Transformation Engine

**Rule expression language:** **JSONLogic** for filters, **Go `text/template`** for transforms. Rationale:

- JSONLogic is data-only JSON, easy to render in a UI form and store in SQLite. Battle-tested and small (Go impl is one file, no eval). Avoids the security headaches of CEL/Starlark embeds.
- Go `text/template` gives us a sandboxed, allocation-bounded renderer. We expose a curated funcmap (`upper`, `lower`, `default`, `now`, `ipInList`, `severityToColor`).

We add **three custom JSONLogic operators** for security use cases:

| Operator | Semantics |
|---|---|
| `in_cidr` | `{"in_cidr": [{"var":"src_ip"}, ["10.0.0.0/8","192.0.2.0/24"]]}` → bool |
| `in_list` | `{"in_list": [{"var":"sha256"}, "list_01HV..."]}` → list lookup against `filter_list_item` (cached) |
| `regex_match` | `{"regex_match": [{"var":"alert.signature"}, "(?i)mimikatz|cobalt"]}` |

### Worked example: "Block IPs from list X, reformat to Slack blocks, forward."

**Endpoint**: `ep_01HVCROWD` (preset = `crowdstrike`).
**Filter list** `list_01HVKNOWN_BAD` ingests Spamhaus DROP daily.

**Rule 1: Drop known-bad source IPs.**

```json
{
  "name": "Drop Spamhaus DROP IPs",
  "position": 10,
  "filter_jsonlogic": {
    "in_cidr": [
      {"var": "DeviceExternalIP"},
      {"list_cidrs": "list_01HVKNOWN_BAD"}
    ]
  },
  "transform_kind": "passthrough",
  "destination_id": null,
  "on_match": "drop"
}
```

**Rule 2: Forward Critical/High severity to Slack as a formatted block.**

```json
{
  "name": "Critical → Slack #soc",
  "position": 20,
  "filter_jsonlogic": {
    "and": [
      {"in": [{"var": "Severity"}, ["Critical", "High"]]},
      {"==": [{"var": "PatternDispositionDescription"}, "Prevention, process killed."]}
    ]
  },
  "transform_kind": "template",
  "transform_body": "{{ template \"slack_blocks\" . }}",
  "destination_id": "dest_slack_soc",
  "on_match": "forward"
}
```

The named template `slack_blocks` is one of our shipped presets:

```
{{ define "slack_blocks" }}
{
  "blocks": [
    {"type":"header","text":{"type":"plain_text","text":"{{ .Severity | upper }}: {{ .Tactic }}"}},
    {"type":"section","fields":[
      {"type":"mrkdwn","text":"*Host*\n{{ .ComputerName }}"},
      {"type":"mrkdwn","text":"*User*\n{{ .UserName | default "unknown" }}"},
      {"type":"mrkdwn","text":"*IP*\n{{ .DeviceExternalIP }}"},
      {"type":"mrkdwn","text":"*Detection*\n{{ .Description | trunc 200 }}"}
    ]},
    {"type":"actions","elements":[
      {"type":"button","text":{"type":"plain_text","text":"Open in Falcon"},"url":"{{ .FalconHostLink }}"}
    ]}
  ]
}
{{ end }}
```

**Rule 3: Everything else → archive webhook (continue chain).** `on_match: "continue"`, destination is a generic HTTP POST to a long-term archive endpoint.

Library choices:
- Filter: `github.com/diegoholiveira/jsonlogic/v3` with a tiny wrapper to register custom ops.
- Template: stdlib `text/template`, plus `github.com/Masterminds/sprig/v3` (curated subset only — disable `env`, `expandenv`, `getHostByName`).
- JSON path normalization for source presets done by hand-written Go structs per preset (zero deps).

---

## 7. Pricing & Metering

### Tier table

| Plan | Price | Events/mo | Endpoints | Destinations | Retention | Rate limit | Body cap |
|---|---|---|---|---|---|---|---|
| **Free** | £0 | 1,000 | 3 | 3 | 7d | 50 rps | 256 KiB |
| **Starter** | £19/mo | 50,000 | 25 | 25 | 30d | 500 rps | 1 MiB |
| **Pro** | £79/mo | 500,000 | unlimited | unlimited | 90d | 2,000 rps | 4 MiB |
| **Overage** | £0.80 / 1k events | — | — | — | — | — | — |

Hard cutoff for free tier; soft cutoff (overage billing) for paid tiers.

### Real-time metering on the cheap

The scary number is Turso's **25M writes/month** free ceiling. If we wrote one row per event we'd cap out at ~833k events/month — fine for free customers, but kills paid. So:

- **Hourly bucket pattern.** All counter increments target `usage_counter (tenant_id, bucket_hour)` via `INSERT … ON CONFLICT DO UPDATE`. Writes per tenant per hour ≈ 1 (ingress) + N (per outbound destination). For a tenant pushing 50k events/month this is ≤ 720 writes/month.
- **In-memory aggregator.** The Fly.io worker keeps an in-process LRU of `{tenant_id, hour}: count` and flushes every 30s or 1k entries — coalescing thousands of events into a single UPSERT. Worst-case write rate for the whole platform at the free tier is ~`(tenants × 24 × N_destinations)` writes/day.
- **Quota check.** Before processing, the worker reads the current month's running total (sum across `bucket_hour`) from a Turso replica (cheap 1ms read) and caches per tenant for 60s. This keeps quota enforcement at "within ~1 minute" accuracy — fine for billing.
- **Stripe sync.** Daily cron pushes `usage_records` to Stripe metered subscription items. We bill per 1k events for overage.

### Why not increment from the Worker?

Workers can write to Turso via libSQL-over-HTTP, but each ingress would cost a write — burning the 25M/mo budget at 8 events/sec sustained. Buffering on Fly.io is the only way to stay free at scale.

---

## 8. Auth, Multi-Tenancy, Security

### Customer auth

- **Dashboard auth**: Clerk free tier (5,000 MAU) for email + Google OAuth; on signup, a webhook lands on Fly.io and provisions a `tenant`.
- **Admin API auth**: API keys formatted `whk_live_<26-char base32>`. Stored as `argon2id` hash; we keep the first 12 chars in cleartext (`prefix`) for display + index lookup.
- **Ingress auth**: ingress URLs themselves are unguessable (ULID + 96 bits of entropy). Optionally protected by HMAC.

### Multi-tenancy

- Single shared Postgres-style schema with `tenant_id` columns and **explicit `WHERE tenant_id = ?` on every query**. All access goes through `sqlc`-generated functions where the helper signature requires `ctx, tenantID, …`.
- We don't use Turso's `attach`/per-DB-per-tenant model — at 9GB free, one DB per tenant doesn't help and costs us per-DB connection overhead.
- A `tenantContext{ID, Plan, Limits}` struct rides every request; middleware refuses to call repo functions if the context isn't populated.

### Secret storage

- Fly.io secrets for: `KMS_KEY` (libsodium master), `STRIPE_SECRET`, `TURSO_AUTH_TOKEN`, `WORKER_SHARED_SECRET`.
- Customer-provided destination secrets (Slack URLs, custom-header values) encrypted with libsodium secretbox using `KMS_KEY`. Stored as base64 in `destination.config_json`.
- Worker shared secret is rotated by deploying both Worker and Go binary with the new env in the same hour (key list, not single key).

### SSRF & egress hardening

- Outbound HTTP client uses a custom `Dialer` that resolves the host once, blocks RFC1918/loopback/link-local, and refuses redirects.
- Per-tenant outbound concurrency cap (semaphore) so one tenant can't drain the global goroutine pool.

### Rate limiting

- Edge: Workers Durable Object token-bucket per tenant.
- Backend: in-memory leaky-bucket per tenant per destination, plus a global `singleflight` cache to prevent thundering-herd retries.

### HMAC inbound verification

Built-in presets:

| Source | Header | Algo | Format |
|---|---|---|---|
| GitHub | `X-Hub-Signature-256` | sha256 | `sha256=<hex>` |
| Stripe-style | `Stripe-Signature` | sha256 | `t=…,v1=<hex>` |
| Slack | `X-Slack-Signature` | sha256 | `v0=<hex>`, with timestamp guard |
| Generic | `X-Signature` | sha256/sha1 | raw hex |

Constant-time compare (`crypto/subtle`). 5-minute timestamp window for replay defense on Slack/Stripe-style.

---

## 9. Observability

### What we surface to customers

- **Event timeline** per endpoint (status, http_status, latency, attempt count, redacted body preview).
- **Replay** button on any event (refires through the rule chain).
- **Per-rule match counts** (last 24h, 7d).
- **Per-destination success rate** sparkline.
- **Quota meter** (events used / events remaining).
- **Webhook tester** in dashboard: send a sample event, see exactly how each rule evaluates.

### Internal telemetry

- **Logs**: structured JSON via `log/slog`. Fly.io ships them to Better Stack free tier (1GB/mo) via vector sidecar.
- **Metrics**: Prometheus exposition on `:9090`, scraped by Grafana Cloud free tier (10k series, 14d retention).
- **Tracing**: OpenTelemetry to Honeycomb free tier (20M events/mo) for the `ingress→queue→worker→fan-out` span chain. Trace ID propagates from Worker via `traceparent` header.
- **Alerts**: `delivery_log` status='failed' rate over 5m vs 1h baseline; >3x triggers Slack alert to ourselves (eat your own dogfood).

### Customer-visible status page

`status.<domain>` is a Cloudflare Pages static site that reads two JSON blobs uploaded by a cron in the Go service: ingress p99 and delivery success rate.

---

## 10. Tech Stack & Libraries

### Cloudflare Worker (TypeScript)

- `wrangler` (latest) for build/deploy.
- `@cloudflare/workers-types` types.
- `@libsql/client/web` for Turso reads (KV cache miss path).
- HMAC via `crypto.subtle.importKey` (no extra dep).
- `hono` for routing if it grows beyond `/w/:id`.
- Bindings: KV `ENDPOINT_KV`, Durable Object `RATE_LIMITER`, Queue producer `INGRESS_Q`, secret `WORKER_SHARED_SECRET`.

### Backend (Go 1.23)

- Router: **`github.com/go-chi/chi/v5`** (small, no magic; gin is fine but we want middleware composability).
- DB: **`github.com/tursodatabase/libsql-client-go`** + **`sqlc`** for typed queries.
- JSONLogic: `github.com/diegoholiveira/jsonlogic/v3`.
- Templating: stdlib `text/template` + curated `github.com/Masterminds/sprig/v3` funcs.
- HTTP client: stdlib `net/http` with custom dialer (SSRF guard).
- Local retry queue: `go.etcd.io/bbolt`.
- Crypto: `golang.org/x/crypto/argon2`, `golang.org/x/crypto/nacl/secretbox`.
- ULID: `github.com/oklog/ulid/v2`.
- Structured logs: stdlib `log/slog`.
- Metrics: `github.com/prometheus/client_golang`.
- Tracing: `go.opentelemetry.io/otel` + `otelhttp`.
- Stripe: `github.com/stripe/stripe-go/v79`.
- Tests: stdlib `testing` + `github.com/stretchr/testify` for assertion ergonomics.

### Frontend (Cloudflare Pages)

- **Next.js 15** (App Router) deployed via `@cloudflare/next-on-pages`.
- Auth: **Clerk** (`@clerk/nextjs`).
- UI: Tailwind + **shadcn/ui**.
- Charts: **Tremor** (`@tremor/react`).
- API client: codegen from Go OpenAPI (`oapi-codegen` server, `openapi-typescript` client).
- Forms (rule editor): **React Hook Form** + **Zod**, with a hand-rolled JSONLogic visual builder for v1 (textarea + validate in MVP).

---

## 11. Build Roadmap (4–6 week solo MVP)

### Week 1 — skeleton + auth

- Fly.io Go app skeleton: `chi`, slog, healthcheck, Turso connect.
- Turso DB created; `sqlc` set up; tables 1–6 (tenants, users, api_keys, endpoints, destinations, rules) migrated via `golang-migrate`.
- Cloudflare Pages Next.js app with Clerk login. Tenant provisioning webhook.
- CI: GitHub Actions → `flyctl deploy` + `wrangler deploy` + `pages deploy`.

### Week 2 — ingress + queue

- Cloudflare Worker `/w/:endpoint_id`: KV lookup, body cap, HMAC verify, push to Queue, return 202.
- Durable Object rate limiter.
- Go consumer endpoint that drains Queue; minimal "log only" mode end-to-end.
- Ship the first delivery log row; render last 50 events in the dashboard.

### Week 3 — rules engine + first destinations

- JSONLogic engine wired up with custom ops (`in_cidr`, `in_list`, `regex_match`).
- Slack and generic-HTTP destinations.
- Go template renderer with the `slack_blocks` preset + `default`/`upper`/`trunc` helpers.
- Dashboard: rule list + rule create/edit form (raw JSON in MVP).

### Week 4 — retries, DLQ, metering, billing

- BoltDB-backed delay queue with backoff schedule.
- DLQ table + replay endpoint + dashboard "Park / Replay" UI.
- `usage_counter` writer with 30s flush; quota enforcement.
- Stripe Checkout for Starter; metered subscription item; daily `usage_records` cron.

### Week 5 — polish + observability

- Webhook tester ("send sample event") in dashboard.
- Discord and PagerDuty Events v2 destinations.
- Honeycomb + Grafana Cloud wired up.
- Status page.
- 30 production-safe end-to-end tests run on every CI build.

### Week 6 — ship + first customers

- Public landing page (single Next.js route): copy, pricing, "Connect CrowdStrike to Slack in 5 minutes" demo video.
- Onboard first 5 design partners by hand. (See §13.)
- Post on r/sysadmin, r/devops, /r/cybersecurity, Hacker News Show HN.

### Optional weeks 7–8 — fast follow

- Filter list subscriptions (Spamhaus DROP + Tor exits).
- Microsoft Teams + Splunk HEC destinations.
- Per-rule per-destination dedupe window.
- Audit log export.

---

## 12. Free-Tier Risk & Scaling Triggers

Concrete "you must move" thresholds:

| Resource | Free ceiling | Migration trigger | What you do |
|---|---|---|---|
| **Cloudflare Workers requests** | 100k/day | 70k/day sustained (~7 days) | Move to Workers Paid ($5/mo, 10M req/mo). Trivial: just enable billing. |
| **Cloudflare Queue ops** | 1M/mo | 700k/mo | Same — Workers Paid covers it. |
| **Turso writes** | 25M/mo | 18M/mo | First, audit `usage_counter` flush coalescing. If still tight, move to Turso Scaler ($29/mo, 250M writes). Don't move to Postgres yet — schema fits SQLite. |
| **Turso reads** | 1B/mo | 700M/mo | Same — Scaler. |
| **Turso storage** | 9GB | 7GB | Same — but first verify `delivery_log` retention pruning is running. |
| **Fly.io compute** | 3 × 256MB | sustained CPU >70% on any VM, OR p99 worker latency >500ms | Resize a single VM to `shared-cpu-2x@1024MB` ($5–10/mo). Scale horizontally only after vertical. |
| **Fly.io egress** | 160GB/mo | 120GB | Egress bills are usually triggered by large customer payloads — enable per-tenant body cap pricing or move chatty tenants to a dedicated machine. |
| **Clerk MAUs** | 5,000 | 4,000 | Move to Clerk Pro ($25/mo) or migrate to Supabase Auth. |
| **Stripe** | n/a | n/a | Always paid (% of revenue). |

**Migration order if you must spend money (cheapest unlock first):**

1. Workers Paid — $5/mo unlocks 50× headroom on requests + queues.
2. Fly.io vertical scale of the single hot VM — $5–10/mo.
3. Turso Scaler — $29/mo only when writes/reads/storage actually demand it.
4. Clerk Pro — only at ~5k MAU.

Total realistic cost at ~50k events/day with 200 paying customers: **$30–50/mo infra**.

---

## 13. Go-to-Market — First 10 Customers

### Wedge

"Get every CrowdStrike/Wazuh alert into Slack with the right format in 5 minutes — and actually filter the noise — for £0 below 1k events/month."

A pre-built **CrowdStrike → Slack** integration is the single demo. Everything else is upsell.

### Where to find them

1. **r/sysadmin, r/cybersecurity, r/sysadmin_uk, r/msp** — write a "I built this because Tines is overkill" post. Show the demo video.
2. **MSP Slack/Discord communities** — TMC's Slack, Reddit MSP Discord, IT Pro Tuesday. Offer free Pro for 3 months in exchange for a logo.
3. **LinkedIn outbound** to job titles "Head of IT", "SecOps Lead" at UK companies under 200 people — 30 messages/day, hand-personalized, lead with a 2-min Loom of *their stack* connected.
4. **GitHub repos** that integrate with CrowdStrike/Wazuh APIs — file friendly issues offering a managed alternative.
5. **Indie Hackers + Show HN** at MVP launch.
6. **Replicate Tines/Torq SEO long tail.** Write 5 specific posts: "CrowdStrike Falcon to Slack with filtering", "Wazuh to Discord", "Sentry to PagerDuty deduplication". Rank fast on low-volume queries.

### Pricing strategy for the first 10

- Free tier with 1,000 events/mo for life — generous on purpose.
- Starter £19/mo, but offer **£9/mo for first 6 months** to the first 20 sign-ups. Anchor against Tines/Torq, not Zapier.
- Two design-partner slots at **£0 in exchange for a logo + testimonial**.

### What unlocks word-of-mouth

- 5-minute setup.
- Excellent error messages when their HMAC is wrong (most competitors fail silently).
- Replay button. Once you've used it, you can't go back.

---

## 14. Open Questions & Risks

### Abuse vectors

- **SSRF / outbound abuse.** Someone configures a destination pointing at `http://169.254.169.254/latest/meta-data/`. Mitigation: outbound dialer blocks RFC1918, loopback, link-local, and IPv6 ULA. Refuse redirects. Keep an allowlist override for paying customers only after a human review.
- **Reflective amplification.** A 10KB inbound webhook fanning out to 50 destinations is a 500KB outbound. Cap per-event fan-out at 10 destinations on free, 25 on starter, 100 on pro. Hard byte cap on outbound per minute per tenant.
- **Spam bots discovering ingress URLs.** ULID + 96 bits entropy mitigates discovery; HMAC kills exploitation. We rate-limit unauthenticated traffic harder than authenticated.

### Compliance

- **GDPR.** We process customer data on their behalf (Article 28 Processor). Need a DPA template ready for v1. Default Fly.io region: **lhr** (London) for EU customers. EU-only mode = pin all writes to Turso EU replicas.
- **Data residency for forwarded payloads.** Payloads transit through Cloudflare PoPs (worldwide), Fly.io (lhr), Turso (lhr/fra). Disclose in privacy policy. Offer "EU only" plan that pins to lhr+fra.
- **Logs retention.** Bodies are stored for the retention window. Provide redact-after-N-days option for paying customers; ship body-redaction (regex + JSONPath) in v1.5.
- **Online Safety Act (UK).** N/A — we're B2B infrastructure, not a user-content platform.
- **PCI/SOC2.** Not in scope for MVP. We document controls but don't claim certification until ARR justifies the audit (~$30k/yr).

### DLQ semantics

- **What counts as DLQ-worthy?** 5 retries failed. Configurable per destination eventually.
- **Replay safety.** Replays do *not* re-run the rule chain by default — they refire the *rendered* outbound body. Add "replay through rules" toggle in v1 with a warning that rules may have changed.
- **DLQ retention.** 30 days, then we delete payloads but keep metadata. Communicate explicitly.
- **Ordering.** We do not guarantee event ordering across retries (partial-failure fan-out ordering is hard). Document this; most security alerting consumers tolerate it.

### Other risks

- **Slack / Discord rate limits.** Both throttle aggressively. Per-destination outbound limiter required in MVP, not v1.
- **Customer leaks shared secret to GitHub.** Auto-revoke on detection (Cloudflare's secret-scanning partner program is free-ish).
- **Fly.io free tier policy changes.** Already happened once. Mitigation: keep the Koyeb spare warm; deploy script tests both targets weekly.
- **Cold-start on Workers KV miss.** A first request to a brand-new endpoint takes ~80ms instead of 10ms. Fine, but instrument it.
- **JSONLogic isn't expressive enough for some rules.** Likely true for ~10% of customer requests after launch. Plan: add a "JS expression" rule type running on the Cloudflare Worker (sandboxed) once the demand is proven.

---

*End of plan. Open this file, pick week 1, and start typing.*
