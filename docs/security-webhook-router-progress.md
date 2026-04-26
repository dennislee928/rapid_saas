# Security Webhook Router Progress Tracker

Last reviewed: 2026-04-26

Source spec: `docs/01-security-webhook-router.md`

## Six-Root Layout

The implementation has been migrated into six root folders:

| Root | Ownership | Current contents | Status |
|---|---|---|---|
| `01-ingress-worker` | Cloudflare Worker ingress | TypeScript Worker, Durable Object rate limiter, Worker tests, Wrangler config | Partial MVP implementation. |
| `02-router-api` | Go backend router API | chi service skeleton, queue consumer route, admin handlers, in-memory store, rule evaluator, template renderer, HTTP sender, retry/DLQ interfaces | Partial backend scaffold. |
| `03-database` | SQLite/libSQL schema and queries | Migrations, sqlc query files, seed data, `sqlc.yaml` | Schema/query scaffold present. |
| `04-dashboard` | Next.js customer dashboard | App Router UI, mock endpoint/event/rule/quota/DLQ data, Clerk dependency | Mock-data prototype. |
| `05-infra-ci` | Deployment and infra examples | Cloudflare/Fly examples, Dockerfile, env examples, root config assets | Scaffold present, live deployment not verified. |
| `06-dev-tooling` | Repo tooling | Makefile, npm scripts, validation scripts, root env example | Tooling present but still references legacy discovery paths. |

Follow-up implementation on 2026-04-26 edited Security Webhook Router implementation folders for route alignment, shared queue-envelope compatibility, validation discovery, and local container scaffolding.

## Current Implementation Snapshot

- `01-ingress-worker` receives webhook POSTs, resolves endpoint config from KV/libSQL/static JSON, enforces body caps, verifies GitHub/Stripe-style/Slack/generic HMAC signatures, optionally checks a Durable Object rate limiter, and enqueues verified events.
- `02-router-api` now exists as the Go backend scaffold. It exposes health/readiness, `POST /_q/consume`, and placeholder admin CRUD routes. It includes rule evaluation, template rendering, outbound HTTP sending, retry scheduling interfaces, DLQ interfaces, and an in-memory repository.
- `03-database` provides the target relational model for tenants, users, API keys, endpoints, destinations, rules, filter lists, delivery logs, usage counters, and DLQ.
- `04-dashboard` renders customer-facing dashboard pages against mock data; it is not yet connected to the router API.
- `05-infra-ci` contains provider configuration examples and deployment notes, but deployment state was not verified.
- `06-dev-tooling` contains validation entrypoints, but component discovery still targets `apps`, `services`, `db`, `infra`, and `packages`, not the six migrated root folders.

## 2026-04-26 Follow-up Implementation

- `01-ingress-worker` now accepts the spec route `POST /w/:endpointId` while preserving the existing `/webhooks/:endpointId` and `/ingest/:endpointId` aliases.
- Ingress queue messages now include the shared event-envelope fields: `event_id`, `tenant_id`, `type`, `schema_version`, `occurred_at`, `idempotency_key`, `payload`, and optional `trace_id`. The legacy top-level payload fields are preserved for compatibility.
- `02-router-api/internal/queue` now accepts both legacy `model.QueueEvent` payloads and `security.webhook.received` shared envelopes. Envelope payloads decode `bodyBase64`, preserve request hashes, and normalize non-JSON bodies into JSON strings before processing.
- `06-dev-tooling/scripts/run_component_tests.sh` now discovers current product folders `07-*` through `12-shared-platform`, so `make validate` exercises the expanded workspace where local dependencies exist.
- Local-development Dockerfiles were added for runnable services and service scaffolds across `01`, `02`, `04`, `07`, `08`, `09`, `10`, and `11`.
- `05-infra-ci/infra/docker-compose.services.yml` now provides a service matrix for `apis`, `workers`, `dashboards`, and `checks` profiles.
- Router API now exposes local delivery-log, usage, DLQ list, and DLQ replay endpoints. The dashboard can read these endpoints when `ROUTER_API_BASE_URL` is configured and falls back to mock data otherwise.
- Remaining live-service blockers are tracked in `docs/residual-production-gaps.md`.

## Orchestration Progress

Shared-platform Phases 0-8 from `docs/07-expect-tech-base-implementation-plan.md` are now represented in the repo without changing Security Webhook Router implementation folders:

- Phase 0 documentation and ownership: `docs/product-readiness-status.md` maps product owners, local verification paths, shared vocabulary, and current production gaps.
- Phase 1 ingress: `12-shared-platform/ingress/` defines a local Nginx gateway and Worker edge-policy helpers for request ID propagation, body caps, security headers, rate-limit placeholders, and auth-signal forwarding.
- Phase 2 events: `12-shared-platform/events/` defines the shared event envelope, memory queue, retry policy, DLQ records, replay behavior, and idempotency tests.
- Phase 3 hot state: `12-shared-platform/hot-state/` defines Redis and memory helpers for locks, token buckets, idempotency keys, velocity counters, blacklists, and PSP health, plus documented fail-open/fail-closed behavior.
- Phase 4 proto: `12-shared-platform/proto/` defines versioned shared contracts for `RiskScoringService`, `PaymentRoutingService`, `WebhookDeliveryService`, and `AuditLogService`, with compatibility and service-auth guidance.
- Phase 5 observability: `12-shared-platform/observability/` defines OpenTelemetry conventions, structured-log schema, SLO targets, alert/dashboard placeholders, and a local trace walkthrough.
- Phase 6 hardening: `docs/phase-6-product-hardening.md` defines release gates and ticket order. Security Webhook Router hardening is tracked as P6-SWR-001 through P6-SWR-009.
- Phase 7 crypto payments: `12-shared-platform/crypto-payments/` isolates testnet invoice, listener, reconciliation, sanctions/AML, and PCI-boundary planning from RouteKit and from this product.
- Phase 8 quantum simulation: `12-shared-platform/quantum-sim/` provides a research-only PSP risk simulation with deterministic data and no production hot-path dependency.

Verification reported by phase workers:

- Events and hot-state packages include Go unit tests for envelope validation, retry/DLQ behavior, idempotency, locks, token buckets, counters, blacklist, and PSP health primitives.
- Observability Phase 5 is marked `implemented-baseline` in `12-shared-platform/observability/phase-5-acceptance-map.yaml`, with product-code OpenTelemetry middleware still called out as future work.
- Crypto payments Phase 7 is documented as planning/prototype contract only and testnet-only; no production secrets, mainnet support, or card-routing dependency is claimed.
- Quantum simulation Phase 8 includes a deterministic script and unittest path under `12-shared-platform/quantum-sim/`.

Remaining Security Webhook Router-specific gaps after shared-platform completion:

- Prove durable Cloudflare Queue-to-router enqueue-to-consume behavior in a deployed or emulator-backed environment.
- Implement router retry scheduling, terminal DLQ records, authenticated replay, and replay audit.
- Enforce quotas for inbound events, destination fan-out, replay, retention tier, and payload limits using durable counters.
- Replace dashboard mock data with live endpoint, delivery-log, quota, and DLQ API data.
- Wire router runtime storage to the `03-database` sqlc/libSQL model instead of in-memory repositories.
- Harden destination secret storage, response/log redaction, usage reconciliation, deployment automation, smoke tests, alerts, and rollback runbooks.

## Spec Section Status

| Spec section | Status | Evidence | Gaps / notes |
|---|---|---|---|
| 1. Product overview | Documented | Spec only | No product-facing onboarding or live deployment evidence yet. |
| 2. Core features | Partial | `01-ingress-worker`, `02-router-api`, `03-database`, `04-dashboard` | API key issuance, rotation/revocation, real persisted admin API, replay, billing, and live dashboard data remain incomplete. |
| 3. System architecture | Partial scaffold | Six-root layout maps to ingress, router API, database, dashboard, infra, tooling; shared gateway and service compose files exist | Cloudflare Queue to Go integration is envelope-compatible but not proven end-to-end against durable queue infrastructure. Koyeb warm spare and live Fly deployment are not verified. |
| 4. Data model | Mostly scaffolded | `03-database/migrations/000001_init_security_webhook_router.up.sql`, `03-database/queries/*` | Schema exists, but router API currently uses in-memory storage. sqlc-generated runtime integration was not observed. Secret encryption is not wired. |
| 5. Request lifecycle | Partial | `01-ingress-worker/src/index.ts`, `02-router-api/internal/queue/*` | Worker implements much of steps 1-6 and now accepts `/w/:endpoint_id`. Go queue processing accepts shared envelopes, but persistence, metering, and durable end-to-end queue delivery are not verified. |
| 6. Filtering & transformation | Partial scaffold | `02-router-api/internal/rules/*`, `03-database` rule columns | In-repo evaluator/template code exists, including custom security-style operators. Full JSONLogic parity, persisted filter-list lookup, presets, and production rule execution are not yet proven. |
| 7. Pricing & metering | Schema only | `03-database` usage counter schema/query files | No quota enforcement, in-memory aggregation, Stripe sync, plan limit enforcement, or billing jobs observed. |
| 8. Auth, multi-tenancy, security | Partial | Worker HMAC, tenant-scoped schema, router API placeholder `X-Tenant-ID`, SSRF helper code | Clerk/admin auth, API key hashing, tenant context enforcement, encrypted customer secrets, key rotation, and full backend egress hardening are incomplete. |
| 9. Observability | Prototype only | Dashboard mock timeline/quota/DLQ data, Go `slog` usage | No real metrics, tracing, status page feed, replay telemetry, or customer-visible live delivery timeline verified. |
| 10. Tech stack & libraries | Partial | TypeScript Worker, Go 1.23 chi service, Next dashboard, SQLite/libSQL schema | Go service currently has minimal dependency surface. Dashboard has Clerk dependency but no live auth integration observed. |
| 11+ operational/MVP plan sections | Not verified | `05-infra-ci`, `06-dev-tooling` | CI/deploy gates, retention cron, DR, and provider-side resources are not proven in this review. |

## Component Ownership

| Component | Directory | Current owner boundary | Implementation status |
|---|---|---|---|
| Ingress Worker | `01-ingress-worker` | Edge receive, signature verification, body cap, rate-limit check, enqueue | Implemented scaffold with tests for health, body cap, and generic HMAC enqueue. Needs spec path alignment and broader provider tests. |
| Rate Limiter Durable Object | `01-ingress-worker/src/index.ts` | Per-key request limiting | Basic fixed-window implementation present. Spec calls for per-tenant token bucket behavior. |
| Router API | `02-router-api` | Queue consume, admin API surface, rule evaluation, transform, outbound delivery, retry/DLQ interfaces | Scaffold present. Current README states admin routes use `X-Tenant-ID` as placeholder tenant context and storage is replaceable in-memory implementation. |
| Database | `03-database` | libSQL/SQLite schema, sqlc queries, seed data | Schema and query files present. Runtime integration from router API to these queries is not observed. |
| Dashboard | `04-dashboard` | Customer UI shell for endpoints, events, rules, quota, DLQ | Mock-data prototype only; no router API integration. |
| Infra / CI | `05-infra-ci` | Cloudflare, Fly, Docker, env examples, deployment notes | Scaffold present; live provider state and CI behavior not verified. |
| Dev Tooling | `06-dev-tooling` | Make/npm validation entrypoints and scripts | Updated to discover the current product folders and shared-platform Go modules. Node packages without component-local `node_modules` are reported as skips. |

## Run Commands

Use these commands from the relevant root:

```sh
cd 01-ingress-worker && npm test
```

Runs the current Worker Vitest suite.

```sh
cd 01-ingress-worker && npm run typecheck
```

Type-checks the ingress Worker.

```sh
cd 02-router-api && GOCACHE="../.cache/go-build" GOMODCACHE="../.cache/go-mod" go test ./...
```

Runs router API tests with repo-local Go caches.

```sh
cd 04-dashboard && npm run build
```

Builds the dashboard. `04-dashboard/package.json` has no `test` script.

```sh
cd 06-dev-tooling && make validate
```

Runs the migrated tooling entrypoint. Current caveat: the discovery script still searches `apps`, `services`, `db`, `infra`, and `packages`, so it may not exercise the six-root implementation until the tooling is updated.

```sh
cd 06-dev-tooling && npm run validate
```

Equivalent package-script entrypoint with the same discovery caveat.

## Key Risks

- `06-dev-tooling` now discovers the expanded workspace, but Node packages without local `node_modules` are skipped rather than installed automatically.
- Durable queue delivery has not been proven against Cloudflare Queue or a queue emulator; the current contract is code-level compatibility.
- Router API storage is currently in-memory; it is not yet wired to `03-database` sqlc/libSQL queries.
- Queue payload contracts must be reconciled between `01-ingress-worker` and `02-router-api`; this review did not verify a successful end-to-end enqueue-to-consume flow.
- Secrets exist in schema/config concepts, but libsodium/Fly `KMS_KEY` encryption for signing secrets and destination credentials is not implemented end-to-end.
- Metering, billing, delivery-log retention, replay, and dashboard live data are still incomplete.
- The dashboard can overstate progress because it renders realistic mock endpoint, event, quota, rule, and DLQ data.

## Next Steps

1. Prove durable queue enqueue-to-consume behavior using Cloudflare Queue bindings or a local emulator-backed adapter.
2. Wire `02-router-api` repositories to `03-database` sqlc/libSQL queries and remove in-memory storage from non-test runtime paths.
3. Implement retry scheduling, terminal DLQ records, authenticated replay, replay audit, and poison-message reason capture.
4. Replace `04-dashboard` mock data with router API calls after authenticated admin/read endpoints are stable.
5. Implement secret encryption, quota/metering, Stripe sync, retention pruning, replay, production observability, and rollback runbooks.
6. Add CI jobs that build the new service Dockerfiles and run package installs for Node services in a controlled cache.
