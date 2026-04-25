# 07. Expected Technical Base Implementation Plan

Updated: 2026-04-26

## 1. Summary

The original technical-base expectation in `.ignore/expect_tech_base.md` is not complete. The repository has strong product specs and MVP scaffolds, but the shared platform layer is still missing several production-critical pieces.

This plan turns the expected technical base into executable phases. It is deliberately ordered so the repo gains reusable infrastructure before adding advanced or research-heavy capabilities.

## 2. Current Baseline

Implemented or partially implemented:

- Cloudflare Worker entrypoints for several products.
- Go service scaffolds for router, graph linking, payment routing, and anti-piracy hashing/crawling.
- SQLite/Turso-style migrations for core product data.
- Static or MVP dashboards for several products.
- Local unit tests for TypeScript Workers and Go services.
- Token-only payment-routing boundary in RouteKit.
- WebSocket/Durable Object model in the realtime geospatial product.

Not production-complete:

- Shared ingress gateway and policy layer.
- Multi-region failover and runbooks.
- gRPC contracts and service-to-service authentication.
- Redis-backed hot state.
- Event backbone with typed envelopes, retries, and DLQ.
- Production observability and SLOs.
- Real provider integrations for PSPs, KYC vendors, email, vaulting, and crawl targets.
- TON payments and quantum simulation.

## 3. Phase 0 - Documentation and Ownership

Goal: make the workspace navigable and prevent drift.

Tasks:

- Keep the root `README.md` aligned with actual folders and product status.
- For every product folder, maintain a local README with commands, runtime boundaries, and production gaps.
- Add a root status matrix that maps each product to implementation maturity: spec, scaffold, local tests, external integrations, production hardening.
- Define shared vocabulary for tenants, API keys, events, webhooks, DLQ, audit logs, and usage counters.

Acceptance criteria:

- A new contributor can identify the product owner folder, local test command, and production gaps within 10 minutes.
- Every product has a documented boundary between demo scaffold and production behavior.

## 4. Phase 1 - Shared Ingress and Edge Policy

Goal: standardize traffic handling before adding more services.

Tasks:

- Add `12-shared-platform/ingress/` or equivalent shared folder for ingress policies.
- Provide Envoy or Nginx local config for:
  - TLS termination in production profile.
  - Request body size limits.
  - L7 rate limits.
  - Security headers.
  - Request ID propagation.
  - HMAC/API-key auth forwarding conventions.
- Define a Cloudflare Worker edge-policy package for products that stay Worker-first.
- Add local Docker Compose wiring so services can be exercised through the gateway.

Acceptance criteria:

- At least Security Webhook Router and RouteKit can run behind the local gateway.
- Requests receive a stable `x-request-id`.
- Oversized payloads and unsigned protected requests are rejected consistently.

## 5. Phase 2 - Event Backbone

Goal: make async work reliable and inspectable.

Tasks:

- Define a shared event envelope:
  - `event_id`
  - `tenant_id`
  - `type`
  - `schema_version`
  - `occurred_at`
  - `idempotency_key`
  - `payload`
  - `trace_id`
- Add local queue infrastructure. Use Redpanda/Kafka locally if Kafka compatibility is required; otherwise document Cloudflare Queues as the production target and provide an adapter interface.
- Implement retry and DLQ conventions.
- Add event producers/consumers to one representative product first, preferably RouteKit outbound webhooks or TiltGuard async graph linking.

Acceptance criteria:

- A failed consumer retry is visible.
- A poison event lands in DLQ with a reason.
- Events can be replayed locally without duplicate side effects.

## 6. Phase 3 - Redis Hot State

Goal: support high-frequency decision paths without overloading SQL.

Tasks:

- Add Redis to local infrastructure.
- Create shared helpers for:
  - Distributed locks.
  - Token buckets.
  - Short-lived idempotency keys.
  - Blacklist/lookups for IP, device, BIN, and PSP health.
  - Velocity counters by tenant and entity.
- Wire Redis into TiltGuard scoring and RouteKit PSP health checks first.

Acceptance criteria:

- Concurrent duplicate requests collapse to one logical operation.
- Velocity counters can be tested deterministically.
- Redis outage behavior is documented per product: fail-open or fail-closed.

## 7. Phase 4 - gRPC Internal Contracts

Goal: formalize low-latency service boundaries.

Tasks:

- Add a `proto/` workspace for shared contracts.
- Define initial services:
  - `RiskScoringService`
  - `PaymentRoutingService`
  - `WebhookDeliveryService`
  - `AuditLogService`
- Generate Go and TypeScript clients where useful.
- Add mTLS or signed internal-service auth plan.

Acceptance criteria:

- One internal call path uses generated gRPC code in local development.
- Protobuf contracts are versioned and documented.
- Backward-compatible field evolution rules are written down.

## 8. Phase 5 - Observability and Operations

Goal: make production behavior measurable.

Tasks:

- Add OpenTelemetry conventions for Workers and Go services.
- Standardize structured logging fields:
  - `request_id`
  - `trace_id`
  - `tenant_id`
  - `product`
  - `operation`
  - `outcome`
  - `latency_ms`
- Define SLOs per hot path:
  - TiltGuard `/v1/score`
  - RouteKit `/charges`
  - Security Webhook Router inbound delivery
  - Realtime geospatial WebSocket update latency
- Add dashboards and alert thresholds.

Acceptance criteria:

- A local request can be traced across gateway, service, queue, and worker where applicable.
- Each production candidate has documented p50/p95/p99 latency targets.

## 9. Phase 6 - Product Hardening Order

Recommended order:

1. `11-high-risk-payment-router/` RouteKit
   - Replace sandbox PSP adapters with real provider adapters behind feature flags.
   - Integrate Basis Theory or equivalent vault.
   - Complete webhook signature verification and reconciliation jobs.
   - Add payment-state-machine persistence and idempotency hardening.

2. `09-igaming-bonus-abuse/` TiltGuard
   - Persist Worker events to the chosen event backbone.
   - Move tenant/rules lookup from static JSON to Turso/KV with cache invalidation.
   - Implement graph-link persistence and feedback labels.
   - Add reviewer actions and webhooks.

3. `10-adult-compliance-antipiracy/` Aegis
   - Add one real KYC provider integration in test mode.
   - Implement audit export and retention controls.
   - Replace crawl placeholders with a curated legal target list and operator approval flow.

4. `01-06` Security Webhook Router
   - Finish queue-backed delivery, DLQ replays, quota enforcement, dashboard live data, and deployment automation.

Acceptance criteria:

- Each product has a production-readiness checklist.
- External integrations are feature-flagged and testable without secrets.
- No product accepts production traffic before observability, audit logs, and rollback paths exist.

## 10. Phase 7 - TON / Crypto Payments

Goal: add crypto payments without contaminating the card-routing compliance boundary.

Tasks:

- Create a separate crypto-payment module or product folder.
- Define wallet, chain listener, confirmation, settlement, refund, and reconciliation flows.
- Add sanctions, AML, and travel-rule risk notes before implementation.
- Start with testnet only.

Acceptance criteria:

- Testnet invoice can be created, detected, confirmed, expired, and reconciled.
- Crypto flows are isolated from PCI-scoped card routing code.

## 11. Phase 8 - Quantum Simulation Research Track

Goal: keep quantum simulation as a research asset, not a blocker for MVP SaaS.

Tasks:

- Add a notebook or service prototype under a clearly marked research folder.
- Define one concrete experiment:
  - Monte Carlo risk exposure simulation for PSP routing.
  - Fraud-ring graph random-walk simulation.
  - AML-style payment-flow path simulation.
- Compare classical baseline vs. quantum simulator output.

Acceptance criteria:

- The prototype has a reproducible input dataset and documented output.
- It does not sit on any production hot path.

## 12. Risks

- Building all advanced tracks at once will slow down the commercially useful products.
- Payment and adult-compliance integrations require legal and provider-specific review before production.
- Redis, Kafka, and gRPC add operational complexity; they should be introduced behind clear product use cases, not as architecture decoration.
- TON and quantum simulation should remain isolated until there is a buyer-facing reason to graduate them.

## 13. Next Implementation Slice

The highest-leverage next slice is:

1. Add shared local infrastructure for gateway, Redis, queue, and observability.
2. Wire RouteKit idempotency and PSP health to Redis.
3. Wire TiltGuard async scoring events to the queue envelope.
4. Add root-level status matrix and product-readiness checklist.

This creates reusable platform value while directly improving the two products closest to the original technical-base expectation.
