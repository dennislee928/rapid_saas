# Rapid SaaS Implementation Workspace

This repository is a multi-product SaaS implementation workspace. It started with the Security Webhook Router and has expanded into several high-risk, compliance, realtime, and infrastructure products. The codebase is intentionally split by product and service boundary so work can proceed in parallel without overlapping write scopes.

## Current Status

The repository contains working MVP scaffolds, tests, schemas, dashboards, shared-platform baselines, and local verification paths for several products. It is not yet production-complete, but Phases 0-8 from `docs/07-expect-tech-base-implementation-plan.md` now have repository evidence and ownership boundaries.

Completed shared-platform baseline:

- Phase 0: `docs/product-readiness-status.md` maps product ownership, local verification, readiness gaps, and shared vocabulary.
- Phase 1: `12-shared-platform/ingress/` defines the local Nginx gateway and Worker edge-policy helpers for request IDs, body caps, security headers, and auth-signal forwarding.
- Phase 2: `12-shared-platform/events/` provides typed event envelopes, memory queue behavior, retry policy, DLQ records, replay, and idempotency tests.
- Phase 3: `12-shared-platform/hot-state/` provides Redis and memory primitives for locks, token buckets, idempotency keys, velocity counters, blacklists, and PSP health, with outage behavior documented.
- Phase 4: `12-shared-platform/proto/` defines versioned protobuf contracts for risk scoring, payment routing, webhook delivery, and audit logging, plus compatibility and service-auth notes.
- Phase 5: `12-shared-platform/observability/` defines OpenTelemetry conventions, structured-log schema, SLO targets, dashboard/alert placeholders, and a local trace walkthrough.
- Phase 6: `docs/phase-6-product-hardening.md` defines production release gates and ticket order for RouteKit, TiltGuard, Aegis, and the Security Webhook Router.
- Phase 7: `12-shared-platform/crypto-payments/` isolates the testnet-only crypto payment plan from RouteKit card-routing and PCI-scoped flows.
- Phase 8: `12-shared-platform/quantum-sim/` contains a reproducible research-only PSP risk simulation with tests and no production hot-path dependency.

Follow-up implementation after the phase baseline added Security Webhook Router `/w/:endpointId` support, shared event-envelope compatibility between the Worker and router queue handler, expanded validation discovery across all product folders, and Dockerfiles for the runnable service roots.

Remaining work is product-specific production hardening: real external integrations, durable product storage where scaffolds still use memory or mocks, live dashboards, deployment automation, rollback runbooks, and production-grade observability in each service.

## Repository Map

| Folder | Product / Ownership | Status |
| --- | --- | --- |
| `01-ingress-worker/` | Cloudflare Worker webhook ingress for the Security Webhook Router | MVP scaffold with tests |
| `02-router-api/` | Go router/admin API, queue consumer, rules, retry, delivery | MVP scaffold with tests |
| `03-database/` | SQLite/libSQL schema, seeds, sqlc query assets | Baseline schema and queries |
| `04-dashboard/` | Next.js dashboard for the Security Webhook Router | MVP dashboard scaffold |
| `05-infra-ci/` | Fly.io, Cloudflare, Docker, CI, deployment templates | Local/deployment templates |
| `06-dev-tooling/` | Local validation scripts and workspace tooling | Tooling scaffold |
| `07-ai-audio-stem-separation/` | Audio stem separation SaaS | Worker, orchestrator, dashboard, HF Space scaffold |
| `08-realtime-geospatial-api/` | Realtime geospatial API using Workers and Durable Objects | MVP scaffold with tests |
| `09-igaming-bonus-abuse/` | TiltGuard iGaming bonus-abuse and multi-accounting detection | MVP scaffold with tests |
| `10-adult-compliance-antipiracy/` | Aegis GateKeep/Reclaim compliance and anti-piracy platform | MVP scaffold with tests |
| `11-high-risk-payment-router/` | RouteKit token-only payment routing and failover gateway | MVP scaffold with tests |
| `12-shared-platform/` | Shared ingress, events, hot state, proto contracts, observability, crypto-payment planning, and quantum research | Phases 1-5, 7, and 8 baseline assets |
| `docs/` | Product specs, progress docs, and implementation plans | Source of planning truth |
| `.ignore/` | Private/local idea and expectation notes | Not production documentation |

## Planning Documents

| Document | Purpose |
| --- | --- |
| `docs/01-security-webhook-router.md` | Security Webhook Router product spec |
| `docs/security-webhook-router-progress.md` | Security Webhook Router progress notes |
| `docs/02-ai-audio-stem-separation.md` | AI audio stem separation product spec |
| `docs/03-realtime-geospatial-api.md` | Realtime geospatial API product spec |
| `docs/04-igaming-bonus-abuse-detection.md` | TiltGuard product spec |
| `docs/05-adult-compliance-anti-piracy.md` | Aegis GateKeep/Reclaim product spec |
| `docs/06-high-risk-payment-router.md` | RouteKit product spec |
| `docs/07-expect-tech-base-implementation-plan.md` | Gap analysis and execution plan for the technical base expectations |
| `docs/product-readiness-status.md` | Phase 0 ownership, maturity, verification, and shared vocabulary matrix |
| `docs/phase-6-product-hardening.md` | Phase 6 release gates and hardening ticket order |
| `docs/residual-production-gaps.md` | Local implementation status and blockers that require live services or credentials |

## Local Verification

Run checks from each product folder. Common examples:

```sh
cd 01-ingress-worker && npm test
cd ../02-router-api && go test ./...
cd ../08-realtime-geospatial-api && npm test
cd ../09-igaming-bonus-abuse && npm test && cd graph-linker && go test ./...
cd ../../10-adult-compliance-antipiracy/worker && npm test
cd ../services/hasher-crawler && go test ./...
cd ../../../11-high-risk-payment-router/orchestrator && go test ./...
cd ../../12-shared-platform/events && go test ./...
cd ../hot-state && go test ./...
cd ../.. && python3 -m unittest discover 12-shared-platform/quantum-sim/tests
```

For root-level release readiness, use the product-specific README files plus `06-dev-tooling/` scripts as the source of truth.

## Local Containers

Most runnable service roots now include a local-development `Dockerfile`. Product-level Compose files exist for `10-adult-compliance-antipiracy/`, `11-high-risk-payment-router/`, and `12-shared-platform/`. The broad service matrix is defined in `05-infra-ci/infra/docker-compose.services.yml`.

Render every service profile:

```sh
docker-compose -f 05-infra-ci/infra/docker-compose.services.yml --profile apis --profile workers --profile dashboards --profile checks config
```

The existing Security Webhook Router focused stack remains in:

```sh
docker-compose -f 05-infra-ci/infra/docker-compose.local.yml --profile app config
```
