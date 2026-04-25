# TiltGuard MVP Scaffold

TiltGuard is a demo-safe iGaming bonus-abuse detection slice based on `docs/04-igaming-bonus-abuse-detection.md`.

This scaffold intentionally avoids invasive browser fingerprinting. The SDK collects coarse, documented demo signals, hashes canvas/font probes locally, and sends them to a Worker-style `/collect` route. Scoring remains server-side.

## Contents

- `src/sdk` - browser JS SDK scaffold exposing `TiltGuard.collect()` and `collectSignals()`.
- `src/worker` - Cloudflare Worker-style hot path with `POST /collect`, `POST /v1/score`, rule-based scoring, reason codes, and queue send stubs.
- `graph-linker` - Go async service skeleton for MinHash-band generation and account linking.
- `migrations` - SQLite/Turso schema for tenants, events, fingerprints, devices, clusters, links, rules, feedback, and webhooks.
- `reviewer-dashboard` - static read-only reviewer dashboard stub.
- `.env.example` and `wrangler.toml` - local configuration examples.

## Local Verification

```sh
npm test
npm run typecheck
cd graph-linker && go test ./...
```

## Worker API

Collect browser signals:

```http
POST /collect
Content-Type: application/json

{
  "tenant_id": "ten_demo",
  "signals": {
    "timezone": "Europe/London",
    "webdriver": false,
    "canvasHash": "sha256-demo"
  }
}
```

Score an operator event:

```http
POST /v1/score
Authorization: Bearer dev_tiltguard_demo_key
Content-Type: application/json

{
  "visit_token": "<from /collect>",
  "account_id": "op_acct_123",
  "event_type": "signup",
  "context": {
    "email_domain": "mailinator.com"
  },
  "client": {
    "ip": "82.31.44.99",
    "country": "GB",
    "is_datacenter": true
  }
}
```

The MVP scorer returns a 0-100 score, `allow`/`review`/`deny`, up to five reason codes, a demo cluster ID, and a queue event stub for async processing.

## Notes

- The Worker uses static `TENANTS_JSON` for local/demo auth. Replace this with KV or Turso lookup before production.
- The Go service currently keeps candidates in memory and is shaped for Cloudflare Queue HTTP delivery. Replace the in-memory store with Turso reads/writes when wiring Fly.io.
- The dashboard is static by design for the scaffold; connect it to authenticated API routes once event storage is live.
