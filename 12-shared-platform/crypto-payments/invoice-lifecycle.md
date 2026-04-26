# Crypto Invoice Lifecycle

This lifecycle is testnet-only and asset-neutral. TON testnet is the first
target, but the state model should not require TON-specific fields outside the
chain-observation adapter.

## Invoice States

- `draft`: request accepted but deposit target has not been allocated
- `open`: deposit target allocated and invoice can receive payment
- `observed`: at least one matching chain transfer has been seen
- `confirming`: observed transfer is waiting for the required finality threshold
- `paid`: expected amount is confirmed and ledgered
- `underpaid`: confirmed amount is below the expected amount after expiry or
  manual close
- `overpaid`: confirmed amount is above the expected amount
- `expired`: no sufficient payment arrived before `expires_at`
- `refund_review`: refund requires operator and compliance review
- `refunded`: refund transaction is submitted and reconciled
- `disputed`: invoice is blocked for investigation

Terminal states:

- `paid`
- `underpaid`
- `overpaid`
- `expired`
- `refunded`
- `disputed`

## State Transitions

Allowed transitions:

- `draft` -> `open`
- `open` -> `observed`
- `open` -> `expired`
- `observed` -> `confirming`
- `observed` -> `expired`
- `confirming` -> `paid`
- `confirming` -> `underpaid`
- `confirming` -> `overpaid`
- `paid` -> `refund_review`
- `underpaid` -> `refund_review`
- `overpaid` -> `refund_review`
- `refund_review` -> `refunded`
- any non-terminal state -> `disputed`

Disallowed transitions:

- terminal state back to `open`, `observed`, or `confirming`
- `expired` directly to `paid` without reopening through an operator-controlled
  adjustment record
- `refund_review` to `paid` without a reconciliation note

## Creation Flow

1. API receives tenant id, asset, network, amount, expiry, and idempotency key.
2. API validates `network` is testnet.
3. API creates or returns the invoice for the idempotency key.
4. Deposit address or memo/comment is allocated.
5. Invoice moves from `draft` to `open`.
6. `crypto.invoice.created` is emitted.

Idempotency scope: `tenant_id + network + asset + idempotency_key`.

## Detection Flow

1. Listener observes a testnet transaction.
2. Listener normalizes transaction data into a chain observation.
3. Listener matches by deposit address plus memo/comment when present.
4. Matching invoice moves to `observed`.
5. `crypto.payment.observed` is emitted.

Multiple observations can match one invoice, but the settlement ledger must be
deduplicated by `network + tx_hash + output/index/logical-time`.

## Confirmation Flow

1. Confirmation worker reloads observations below the finality threshold.
2. Worker checks the latest testnet height/logical time from a trusted endpoint.
3. Invoice moves to `confirming` while waiting.
4. When threshold is met, worker calculates confirmed paid amount.
5. Invoice moves to `paid`, `underpaid`, or `overpaid`.
6. Ledger entries are written in the same transaction as the state transition.
7. `crypto.payment.confirmed` is emitted.

## Expiry Flow

1. Expiry worker scans `open`, `observed`, and stale `confirming` invoices.
2. If `expires_at` passed and confirmed funds are insufficient, the invoice
   moves to `expired` or `underpaid`.
3. Late-arriving funds do not automatically reopen the invoice.
4. Late funds create a compliance and operator review item.

## Refund Flow

Refunds are manual-review first in Phase 7.

1. Operator or reconciliation job requests refund review.
2. Compliance checks run against sender and recipient wallet addresses.
3. Approved testnet refund proposal is signed by an isolated wallet process.
4. Refund transaction is observed by the listener.
5. Reconciliation worker marks the invoice `refunded`.

No automated production refund signing is in scope.
