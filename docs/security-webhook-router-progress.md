# Security Webhook Router Progress Tracker

Last reviewed: 2026-04-25

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

No implementation folders were edited for this tracker update.

## Current Implementation Snapshot

- `01-ingress-worker` receives webhook POSTs, resolves endpoint config from KV/libSQL/static JSON, enforces body caps, verifies GitHub/Stripe-style/Slack/generic HMAC signatures, optionally checks a Durable Object rate limiter, and enqueues verified events.
- `02-router-api` now exists as the Go backend scaffold. It exposes health/readiness, `POST /_q/consume`, and placeholder admin CRUD routes. It includes rule evaluation, template rendering, outbound HTTP sending, retry scheduling interfaces, DLQ interfaces, and an in-memory repository.
- `03-database` provides the target relational model for tenants, users, API keys, endpoints, destinations, rules, filter lists, delivery logs, usage counters, and DLQ.
- `04-dashboard` renders customer-facing dashboard pages against mock data; it is not yet connected to the router API.
- `05-infra-ci` contains provider configuration examples and deployment notes, but deployment state was not verified.
- `06-dev-tooling` contains validation entrypoints, but component discovery still targets `apps`, `services`, `db`, `infra`, and `packages`, not the six migrated root folders.

## Spec Section Status

| Spec section | Status | Evidence | Gaps / notes |
|---|---|---|---|
| 1. Product overview | Documented | Spec only | No product-facing onboarding or live deployment evidence yet. |
| 2. Core features | Partial | `01-ingress-worker`, `02-router-api`, `03-database`, `04-dashboard` | API key issuance, rotation/revocation, real persisted admin API, replay, billing, and live dashboard data remain incomplete. |
| 3. System architecture | Partial scaffold | Six-root layout maps to ingress, router API, database, dashboard, infra, tooling | Cloudflare Queue to Go integration is scaffolded but not proven end-to-end. Koyeb warm spare and live Fly deployment are not verified. |
| 4. Data model | Mostly scaffolded | `03-database/migrations/000001_init_security_webhook_router.up.sql`, `03-database/queries/*` | Schema exists, but router API currently uses in-memory storage. sqlc-generated runtime integration was not observed. Secret encryption is not wired. |
| 5. Request lifecycle | Partial | `01-ingress-worker/src/index.ts`, `02-router-api/internal/queue/*` | Worker implements much of steps 1-6. Go queue processing exists as scaffold code for later lifecycle steps, but persistence, metering, and end-to-end queue delivery are not verified. Worker route still accepts `/webhooks/:id` and `/ingest/:id`, while the spec uses `/w/:endpoint_id`. |
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
| Dev Tooling | `06-dev-tooling` | Make/npm validation entrypoints and scripts | Present, but scripts still discover legacy component folders rather than `01-*` through `06-*`. |

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

- `06-dev-tooling/scripts/run_component_tests.sh` and `06-dev-tooling/package.json` still encode the old monorepo layout, so repository validation can silently miss migrated components.
- The ingress URL path is still misaligned: the spec and infra notes use `/w/<endpoint_id>`, while the Worker currently accepts `/webhooks/:endpointId` and `/ingest/:endpointId`.
- Router API storage is currently in-memory; it is not yet wired to `03-database` sqlc/libSQL queries.
- Queue payload contracts must be reconciled between `01-ingress-worker` and `02-router-api`; this review did not verify a successful end-to-end enqueue-to-consume flow.
- Secrets exist in schema/config concepts, but libsodium/Fly `KMS_KEY` encryption for signing secrets and destination credentials is not implemented end-to-end.
- Metering, billing, delivery-log retention, replay, and dashboard live data are still incomplete.
- The dashboard can overstate progress because it renders realistic mock endpoint, event, quota, rule, and DLQ data.

## Next Steps

1. Update `06-dev-tooling` discovery paths and workspace metadata for the six-root layout.
2. Align Worker routing with the spec by adding `/w/:endpoint_id` support while preserving existing aliases if needed.
3. Verify and, if needed, normalize the event contract between `01-ingress-worker` queue messages and `02-router-api` `POST /_q/consume`.
4. Wire `02-router-api` repositories to `03-database` sqlc/libSQL queries and remove in-memory storage from non-test runtime paths.
5. Expand tests across Worker providers, router queue processing, rule evaluation, template rendering, SSRF blocking, retry/DLQ behavior, and database query integration.
6. Replace `04-dashboard` mock data with router API calls after authenticated admin/read endpoints are stable.
7. Implement secret encryption, quota/metering, Stripe sync, retention pruning, replay, and production observability.
