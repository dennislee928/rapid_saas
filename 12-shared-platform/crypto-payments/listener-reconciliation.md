# Listener And Reconciliation Model

The listener path must tolerate duplicate block reads, RPC gaps, short reorgs,
and worker restarts without duplicating invoice side effects.

## Chain Listener

Responsibilities:

- poll or subscribe to TON testnet account transactions
- persist checkpoints per `network + account`
- normalize transactions into chain observations
- match observations to invoices by deposit address and memo/comment
- emit `crypto.payment.observed`

The listener does not mark invoices paid. It only records observations and
advances invoices to `observed` when a match is found.

## Checkpointing

Checkpoint record:

- `network`
- `listener_account`
- `cursor_type`
- `cursor_value`
- `last_seen_tx_hash`
- `last_seen_lt_or_height`
- `updated_at`

Checkpoint updates must be committed after observations are persisted. On
restart, the listener should replay a small overlap window and rely on
observation idempotency.

## Idempotency

Observation idempotency key:

`network + asset + tx_hash + lt_or_height + to_address + amount`

Invoice side-effect idempotency key:

`invoice_id + event_type + source_observation_id`

Settlement ledger idempotency key:

`invoice_id + entry_type + source_observation_id`

## Confirmation Worker

Responsibilities:

- read unconfirmed observations
- query latest testnet finality marker
- update confirmation counts
- move matching invoices through `confirming`
- write settlement ledger entries when final
- emit `crypto.payment.confirmed`

Recommended first threshold: configurable, defaulting to a conservative testnet
confirmation count. The exact TON finality rule must be verified against the
selected testnet RPC provider before code is promoted beyond prototype.

## Reconciliation Worker

Reconciliation compares three sources of truth:

- chain observations
- invoice state
- settlement ledger entries

Checks:

- observed transfer without matched invoice
- invoice marked paid without a final observation
- invoice paid amount different from ledger sum
- duplicate observations for the same transaction
- expired invoice with late funds
- refund request without a submitted refund observation
- settlement entry without a corresponding invoice transition

Each reconciliation run writes an immutable report with:

- `run_id`
- `network`
- `started_at`
- `completed_at`
- `checked_checkpoint`
- `issue_count`
- `issues`

Issues that affect money movement must create `crypto.compliance.review_required`
or an operator review task before any refund or adjustment is submitted.

## Failure Behavior

- RPC unavailable: pause listener, retain checkpoint, alert after threshold
- duplicate transaction: ignore duplicate observation, retain audit record
- transaction parse failure: write dead-letter observation with raw metadata
- invoice store unavailable: do not advance checkpoint beyond unprocessed data
- reconciliation mismatch: block settlement export for affected invoice

## Observability

Logs and spans must include:

- `request_id`
- `trace_id`
- `tenant_id`
- `product=crypto-payments`
- `operation`
- `network`
- `asset`
- `invoice_id`
- `tx_hash`
- `outcome`
- `latency_ms`

Alerts:

- listener lag above threshold
- confirmation worker lag above threshold
- reconciliation mismatch count above zero
- testnet guardrail startup failure
- compliance review queue older than SLA
