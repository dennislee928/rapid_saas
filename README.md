# Rapid SaaS Implementation Workspace

This repository is a multi-product SaaS implementation workspace. It started with the Security Webhook Router and has expanded into several high-risk, compliance, realtime, and infrastructure products. The codebase is intentionally split by product and service boundary so work can proceed in parallel without overlapping write scopes.

## Current Status

The repository contains working MVP scaffolds, tests, schemas, dashboards, and local verification paths for several products. It is not yet a production-complete implementation of the full enterprise architecture described in `.ignore/expect_tech_base.md`.

Major production gaps still to implement include:

- Shared ingress gateway with Nginx or Envoy.
- SDN / multi-region failover design and runbooks.
- Internal gRPC contracts between long-running services.
- Redis-backed hot cache, distributed locks, and velocity counters.
- Kafka-class event backbone, or a documented Cloudflare Queues equivalent per product.
- WebSocket operations dashboards beyond the realtime geospatial API.
- TON / crypto payment integration.
- Quantum simulation research prototype and product boundary.

See `docs/07-expect-tech-base-implementation-plan.md` for the implementation plan.

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
```

For root-level release readiness, use the product-specific README files plus `06-dev-tooling/` scripts as the source of truth.
