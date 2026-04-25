# Crypto Payments Testnet Module

Phase 7 introduces a separate crypto-payment planning module for TON and other
chain-based payments. This folder is intentionally isolated from RouteKit card
routing and PSP token flows.

Status: planning and prototype contract only. Testnet only.

## Scope

The first implementation target is a testnet invoice flow:

- create an invoice with a unique deposit address or memo/comment
- listen for inbound testnet transfers
- apply confirmation rules
- mark invoices paid, underpaid, overpaid, expired, refunded, or disputed
- reconcile listener observations against internal invoice state
- emit audit and settlement events through shared platform conventions

Out of scope for this phase:

- mainnet support
- custody of production assets
- card acquiring, card vaulting, or card-present/card-not-present flows
- fiat conversion and production treasury operations
- automated sanction decisions without compliance review

## Isolation Boundary

Crypto payments must be implemented as a separate service/module with its own
storage, secrets, listeners, reconciliation jobs, and compliance review queue.
It must not import RouteKit card-routing internals or share PCI-scoped tables.

Allowed shared dependencies:

- shared event envelope from `12-shared-platform/events`
- shared observability fields from `12-shared-platform/observability`
- shared service-auth conventions from `12-shared-platform/proto`
- shared hot-state primitives for idempotency and listener locks

Disallowed dependencies:

- RouteKit PSP adapters
- card token or vault abstractions
- PCI-scoped cardholder data stores
- webhook handlers that assume card payment semantics

## Testnet-Only Guardrails

Every runtime must enforce testnet mode by configuration and by startup
validation.

Required controls:

- `CRYPTO_PAYMENTS_NETWORK` must equal an explicit testnet value such as
  `ton-testnet`.
- startup fails if a mainnet RPC endpoint, mainnet chain id, or production
  wallet seed is configured
- generated deposit addresses and explorer links must identify the testnet
- invoices record `network` and `asset` immutably
- integration tests use fixtures or testnet RPC only

No production secrets should be created for this module during Phase 7.

## Proposed Components

- `invoice-api`: tenant-facing invoice creation and read model
- `chain-listener`: watches testnet blocks or account transactions
- `confirmation-worker`: advances observed payments after finality thresholds
- `reconciliation-worker`: compares chain observations, invoice state, and
  settlement ledger entries
- `refund-worker`: creates manual-review refund proposals and, later, signed
  testnet refund transactions
- `compliance-review`: stores sanctions, AML, and travel-rule review outcomes

## Event Types

All events use the shared event envelope with tenant id, request id, trace id,
schema version, and idempotency key.

- `crypto.invoice.created`
- `crypto.invoice.expired`
- `crypto.payment.observed`
- `crypto.payment.confirmed`
- `crypto.payment.reconciled`
- `crypto.payment.refund_requested`
- `crypto.payment.refund_submitted`
- `crypto.compliance.review_required`
- `crypto.compliance.review_resolved`

## Minimal Data Model

Invoice:

- `invoice_id`
- `tenant_id`
- `network`
- `asset`
- `amount_expected`
- `amount_paid`
- `deposit_address`
- `deposit_memo`
- `status`
- `expires_at`
- `created_at`
- `updated_at`
- `idempotency_key`

Chain observation:

- `observation_id`
- `network`
- `asset`
- `tx_hash`
- `lt_or_height`
- `from_address`
- `to_address`
- `memo`
- `amount`
- `confirmation_count`
- `observed_at`
- `matched_invoice_id`

Settlement ledger entry:

- `entry_id`
- `invoice_id`
- `tenant_id`
- `network`
- `asset`
- `entry_type`
- `amount`
- `source_observation_id`
- `created_at`

## Verification Target

Phase 7 acceptance is met when a testnet invoice can be:

1. created with an idempotency key
2. detected by the listener from a testnet transfer
3. confirmed after the configured finality threshold
4. expired when unpaid past `expires_at`
5. reconciled without duplicate settlement entries

See `invoice-lifecycle.md` and `listener-reconciliation.md` for the state
machine and worker model.
