# Security Webhook Router Progress Tracker

Last reviewed: 2026-04-25

Source spec: `docs/01-security-webhook-router.md`

## Current Implementation Snapshot

The repository currently contains a partial MVP scaffold:

- Edge ingress Worker in `apps/ingress-worker` with health checks, endpoint resolution from KV/libSQL/static JSON, request body cap, HMAC verification for GitHub/Stripe-style/Slack/generic signatures, optional Durable Object rate limiting, and queue enqueue.
- Dashboard prototype in `apps/dashboard` with static/mock endpoint, timeline, rule, quota, and DLQ views.
- SQLite/libSQL schema, sqlc query files, and sample seed data in `db/`.
- Cloudflare/Fly/env examples and validation scripts in `infra/` and `scripts/`.
- No backend Go queue consumer/router service exists yet under `services/`, `cmd/`, or `internal/`.

## Spec Section Status

| Spec section | Status | Evidence | Gaps / notes |
|---|---|---|---|
| 1. Product overview | Documented | Spec only | No product-facing onboarding or live deployment evidence yet. |
| 2. Core features | Partial | Worker, dashboard prototype, DB schema | API key issuance, rotate/revoke, admin API, real destination delivery, replay, billing, and live delivery logs are not implemented. |
| 3. System architecture | Partial scaffold | `apps/ingress-worker`, `apps/dashboard`, `db/`, `infra/` | Go service on Fly.io, queue consumer, retry scheduler, Koyeb spare, and runtime integration are missing. |
| 4. Data model | Mostly scaffolded | `db/migrations/000001_init_security_webhook_router.up.sql`, `db/queries/*` | Schema has tables/indexes/triggers, but generated sqlc code and service usage are absent. Secret encryption is not wired. |
| 5. Request lifecycle | Partial | `apps/ingress-worker/src/index.ts` | Steps 1-6 are partially implemented, but route path is `/webhooks/:id` or `/ingest/:id`, not spec `/w/:endpoint_id`. Steps 7-15 are missing with no Go consumer. |
| 6. Filtering & transformation | Schema only | `rule.filter_jsonlogic`, `transform_*` columns and mock dashboard rules | No JSONLogic evaluator, custom ops, template renderer, presets, or rule execution runtime yet. |
| 7. Pricing & metering | Schema only | `usage_counter` migration and query files | No quota enforcement, in-memory aggregation, Stripe sync, or billing plan logic. |
| 8. Auth, multi-tenancy, security | Partial | Tenant-scoped schema, Worker HMAC, rate limiter, env examples | Clerk/admin auth, API key hashing, tenant context middleware, encrypted secret storage, backend SSRF guard, and key rotation are missing. |
| 9. Observability | Prototype only | Dashboard mock data | No structured backend logs, metrics, tracing, customer status page, replay telemetry, or real delivery metrics. |
| 10. Tech stack & libraries | Partial | Worker TypeScript, Next dashboard, SQLite schema | No Go module/service dependencies. Dashboard has Clerk dependency but no live auth wiring observed. |
| 11+ operational/MVP plan sections | Not verified | Infra examples and scripts | Deployment, CI gates, retention cron, and disaster recovery are not proven in this review. |

## Component Ownership

| Component | Directory | Current owner boundary | Implementation status |
|---|---|---|---|
| Ingress Worker | `apps/ingress-worker` | Edge receive, verify, rate-limit, enqueue | Implemented scaffold with tests for health, body cap, and generic HMAC enqueue. Needs spec path alignment and broader provider tests. |
| Rate Limiter Durable Object | `apps/ingress-worker/src/index.ts` | Per-key fixed-window request limiting | Basic implementation present. Spec calls for tenant token bucket semantics. |
| Dashboard | `apps/dashboard` | Customer UI shell and mocked views | Mock-data prototype only; no API integration. |
| Database schema/query layer | `db/` | Tenants, endpoints, rules, destinations, logs, usage, DLQ | Schema and query files present. Runtime consumers and generated code not observed. |
| Backend router service | Expected under `services/` or Go module path | Queue consumer, rule engine, fan-out, retries, DLQ, metering | Missing. |
| Infrastructure examples | `infra/` | Cloudflare/Fly/env examples | Scaffold present; not verified against live provider state. |
| Developer validation | `Makefile`, `scripts/` | Env checks and component test discovery | Present. Validation commands documented below. |

## Run Commands

Use these commands from the repository root:

```sh
make validate
```

Runs environment example checks and discovered component tests.

```sh
npm run validate
```

Equivalent package-script entrypoint for validation.

```sh
./scripts/validate_env_examples.sh
```

Checks root `.env.example` for obvious committed secret values. Current script only scans root-level env examples.

```sh
./scripts/run_component_tests.sh
```

Discovers component tests under `apps/`, `services/`, `db/`, `infra/`, and `packages/`.

```sh
cd apps/ingress-worker && npm test
```

Runs the current Worker Vitest suite.

```sh
cd apps/ingress-worker && npm run typecheck
```

Type-checks the ingress Worker.

Dashboard note: `apps/dashboard/package.json` has no `test` script. Use `npm run build` from `apps/dashboard` when validating the current UI.

## Key Risks

- The original spec requires `https://in.<domain>/w/<endpoint_id>`, but the current Worker accepts `/webhooks/:endpointId` and `/ingest/:endpointId`; deployment docs still mention `/w/`.
- The Go queue consumer/router is the main missing runtime path, so queue messages cannot currently be evaluated, transformed, delivered, retried, metered, or moved to DLQ by repo code.
- The database stores `signing_secret` and destination config fields but encryption with libsodium/Fly `KMS_KEY` is not implemented in application code.
- Worker generic signatures use `x-webhook-signature` or `x-signature`; the spec names generic `X-Signature` and includes sha1 support, which is not implemented.
- Rate limiting is keyed by endpoint plus source IP in the Worker; the spec calls for per-tenant limits and backend per-destination concurrency controls.
- Dashboard values are mock data, so it can give a false sense of completed observability/delivery-log functionality.
- Validation coverage is uneven: Worker has tests, dashboard lacks tests, and no backend service tests can exist until the service is implemented.

## Next Steps

1. Decide whether to update the Worker route to support `/w/:endpoint_id` while keeping existing aliases for compatibility.
2. Add provider-specific Worker tests for GitHub, Stripe-style, Slack timestamp guard, queue-not-configured handling, disabled endpoint, and rate limiting.
3. Scaffold the Go backend service with a queue consumer endpoint, shared-secret authentication, typed config, and health/readiness endpoints.
4. Generate/use sqlc code and implement tenant-scoped repositories for endpoints, rules, destinations, delivery logs, usage counters, and DLQ.
5. Implement the rule engine: JSONLogic filters, custom security operators, curated `text/template` transforms, and preset transforms.
6. Implement outbound delivery with SSRF-safe HTTP client, per-tenant/per-destination concurrency caps, retry scheduling, and DLQ parking.
7. Replace dashboard mock data with API calls once admin/read endpoints exist.
8. Expand env validation to include nested `infra/env/*.example` and app-level `.env.example` files if that is intended as a repo-wide guard.
