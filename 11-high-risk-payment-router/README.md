# RouteKit MVP Scaffold

RouteKit is a token-only high-risk payment routing slice. This scaffold follows `docs/06-high-risk-payment-router.md` and deliberately avoids raw PAN handling: examples, tests, and validation only accept vault/payment-method tokens.

## Components

- `edge-worker/`: Cloudflare Worker-style edge API guard for API-key auth, HMAC signatures, idempotency, schema validation, and PSP webhook ingress stubs.
- `orchestrator/`: Go payment orchestrator with charge/capture/refund primitives, routing rules, PSP adapter interfaces, sandbox Nuvei/Trust/Worldpay/Mollie adapters, inbound webhook storage, and outbound webhook retry planning.
- `migrations/`: Postgres/SQLite-compatible schema baseline for merchants, PSP credentials, routing rules, payment methods, transactions, attempts, webhooks, ledger entries, and PSP health.
- `dashboard/`: Merchant dashboard static stub for routing health, rules, transaction search, keys, and webhook delivery status.

## Local Verification

```sh
cd 11-high-risk-payment-router/orchestrator
go test ./...
go run ./cmd/routekit
```

Then call the local orchestrator:

```sh
curl -s http://localhost:8080/healthz
curl -s -X POST http://localhost:8080/charges \
  -H 'content-type: application/json' \
  -H 'idempotency-key: demo-key-1' \
  -d '{"merchant_id":"m_demo","payment_method_token":"btok_demo_123","amount_minor":4200,"currency":"GBP","country":"GB","brand":"visa","capture":true}'
```

## Token-Only Boundary

- Do not send `card_number`, `pan`, `cvv`, `expiry`, or similar fields to RouteKit.
- The edge worker rejects known raw-card field names and 13-19 digit numeric runs.
- PSP adapters only receive a vault token or stored payment method token.
- Sandbox tests assert that raw-card-shaped payloads are rejected.

## Environment

Copy `.env.example` for local service variables and `edge-worker/.dev.vars.example` for Cloudflare Worker secrets.

