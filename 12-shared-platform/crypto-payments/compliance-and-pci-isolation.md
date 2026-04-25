# Compliance And PCI Isolation Notes

These notes are pre-implementation guardrails for Phase 7. They are not legal
advice and must be reviewed by qualified counsel and compliance owners before
any production crypto-payment launch.

## PCI Isolation

Crypto-payment flows must stay outside the card-routing compliance boundary.

Hard requirements:

- no PAN, CVV, track data, card tokens, cardholder names, or card billing
  addresses are accepted by crypto-payment APIs
- no RouteKit PSP adapter, card vault, or card-routing database table is reused
- no crypto listener or refund worker has access to PCI-scoped secrets
- crypto webhook/event topics are separate from card PSP webhook topics
- dashboards must label crypto payments separately from card charges
- audit exports must not join crypto wallet data with PCI-scoped cardholder data

Allowed shared platform services are limited to tenant identity, service auth,
observability, event envelopes, and hot-state primitives.

## Sanctions Screening

Before matching or refunding a payment, the system must support screening for:

- sender wallet address
- recipient/deposit wallet address
- refund destination address
- tenant and merchant identity
- counterparty metadata if collected for travel-rule purposes

Phase 7 prototype behavior:

- record screening inputs and provider/test fixture result
- block automatic settlement export for positive or inconclusive matches
- create manual compliance review for positive, partial, or unavailable results
- never silently mark a screened-positive invoice as clean

## AML Risk Notes

AML risk signals to preserve for review:

- payment from a high-risk address cluster
- rapid repeated payments below review thresholds
- overpayment followed by refund request
- payment to expired invoice followed by refund request
- many invoices funded by one sender across tenants
- mismatch between expected and paid amount
- use of mixers, bridges, peel chains, or sanctioned services when known

Phase 7 should implement the data model and review hooks before automated
decisioning. Automated blocking rules can be introduced only after test fixtures
and review workflows exist.

## Travel-Rule Notes

Travel-rule obligations vary by jurisdiction, asset, amount, and role. The
prototype must keep space for:

- originator name or entity identifier when required
- beneficiary name or entity identifier when required
- wallet ownership attestation
- VASP counterparty details when a transfer involves another VASP
- threshold amount and jurisdiction that triggered collection
- review outcome and reviewer identity

Do not collect unnecessary personal data by default. Add collection only when a
tenant policy, jurisdiction, or compliance review requires it.

## Data Retention

Crypto-payment records should separate operational records from compliance
records:

- invoice and observation records: product retention policy
- screening requests and results: compliance retention policy
- manual review notes: compliance retention policy
- raw chain metadata: minimized where possible, with immutable references to
  public transaction identifiers

## Production Promotion Gates

Mainnet must stay disabled until all gates are satisfied:

- legal review complete for target jurisdictions
- sanctions provider selected and tested
- AML review workflow implemented
- travel-rule policy documented
- key management and wallet signing architecture reviewed
- incident runbook written for stuck, mistaken, or sanctioned funds
- PCI isolation reviewed against RouteKit/card-routing code and data stores
- observability and reconciliation alerts enabled
