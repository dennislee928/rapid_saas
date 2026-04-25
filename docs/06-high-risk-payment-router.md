# SaaS Implementation Plan: High-Risk Payment Routing & Failover Gateway

**Working name:** RouteKit (placeholder)
**Category:** Payment Orchestration (B2B Fintech infrastructure)
**Vertical focus:** UK / EU "high-risk-but-legal" e-commerce — CBD, vape, spirits & alcohol DTC, nutraceuticals/supplements, firearms accessories, adult-adjacent retail, kratom, nootropics.

---

## 1. Product Overview & Target Users

The user is a UK or EU SMB e-commerce merchant doing £20k–£2M monthly GMV in a vertical that Stripe and PayPal will not reliably underwrite. They have already been frozen at least once. They currently hold accounts at two or three high-risk PSPs (Nuvei, CCBill, Worldpay HR, Trust Payments, Paymentwall, sometimes a specialist EU acquirer like Aircash or Emerchantpay) and they swap which PSP is "live" on their checkout by hand whenever one of them throttles, freezes, or pushes their decline rate above ~15%.

The pitch is: integrate one API, never touch a PSP integration again, and let the router pick the winning PSP on every transaction.

**Why they buy from us instead of Spreedly / Primer / Gr4vy:**

- **Price.** Spreedly starts around £400/mo platform fee plus 0.05–0.15% per txn. Primer is enterprise-priced and effectively unavailable to a £200k/mo CBD shop. We target 0.3% per txn flat, no platform fee, no minimums. We can sustainably undercut Spreedly because we deliberately ship with 6–10 PSPs, not 120+.
- **Vertical specialisation.** Spreedly will not pre-integrate Worldpay HR, CCBill or Paymentwall well — those are not their core demand. We pre-integrate exactly the PSPs a UK CBD shop actually uses, and we ship a "CBD-ready" preset, a "Vape EU" preset, a "Spirits DTC" preset.
- **Onboarding is the product.** The hardest part of high-risk payments is getting underwritten by each PSP. We bundle a guided onboarding wizard, pre-fill MID applications, and have referral relationships with 2–3 high-risk merchant account brokers. This is something the orchestration unicorns explicitly do not do.
- **UK domicile + UK-time support.** Primer/Gr4vy are US-coast oriented; Spreedly is US. A UK Ltd with UK working hours, GBP invoicing, and direct knowledge of FCA scope is a real differentiator at SMB scale.

We are not competing with Stripe Connect or with adyen-for-platforms. We are competing with the spreadsheet a stressed CBD founder maintains at 11 pm.

---

## 2. Core Features — MVP vs. v1

### MVP (weeks 0–14, ship to first paid trial merchant)

- **Single REST API** with three primitives: `POST /charges`, `POST /charges/{id}/capture`, `POST /refunds`. Plus `POST /payment_methods` to vault a card and `GET /transactions` for reconciliation.
- **3DS2 / SCA flow** end-to-end (return URL hosted on our domain, see §10).
- **Static rule-based routing** across 2–4 PSPs: rules expressed as priority + simple predicates (currency, country, BIN range, amount band, MCC).
- **Automatic failover** on hard decline reason codes (insufficient_funds is *not* retried elsewhere; do_not_honor, processor_unavailable, network_error, timeout *are* retried on next-priority PSP within the same logical "charge attempt").
- **Webhook ingest from each PSP** with signature verification, replay protection, idempotent persist.
- **Outbound webhooks** to merchants (HMAC signed, exponential backoff retry, DLQ).
- **Success-rate dashboard** per PSP, per BIN family, per country (last 24h, last 7d, last 30d).
- **Vault integration via Basis Theory** (no PAN ever lands on our infra — see §3).
- **Merchant dashboard** (Cloudflare Pages + React) for keys, PSP credentials, transaction search, basic rules editor.

### v1 (months 4–9)

- **Adaptive routing**: success-rate-weighted EWMA + a Thompson-sampling bandit per (PSP, BIN6, country, currency) tuple. Cold-start falls back to static rules.
- **Smart retry / cascading**: same charge can attempt up to N PSPs, with explicit policy per merchant (max attempts, max latency budget, "never retry if 3DS authenticated on first attempt", etc.).
- **Currency-aware routing**: prefer PSPs that settle natively in merchant's settlement currency to avoid 1.5–3% FX conversion.
- **BIN-based routing**: maintain a BIN table (download from Bin-list / a paid BIN provider) — Visa-credit-Tier1-UK to PSP A, Mastercard-debit-DE to PSP B.
- **Universal vault & network tokens**: Basis Theory tokens reusable across PSPs; opt-in to Visa/MC network tokens for higher auth rates and lifecycle management.
- **Reconciliation**: nightly job that pulls each PSP's settlement report (CSV/SFTP/API) and reconciles to our ledger.
- **Anti-fraud hooks**: optional pre-route call to Sift/Seon/Fingerprint to attach a risk score, used as a routing input.
- **Multi-acquirer split capture**: rare but useful — for very large baskets, split across PSPs.

Explicitly NOT in v1: APMs (PayPal, Klarna, iDEAL), payouts, marketplace/split-pay, KYC of end users, FX execution. Those are v2+.

---

## 3. PCI Scope Decision — The Existential Call

**Recommendation: zero PAN on our infrastructure. Use Basis Theory as the universal card vault. We operate as SAQ-A with no card data ever transiting our servers.**

This is non-negotiable and it shapes the entire architecture, so it is section 3.

The three theoretical options are:

1. We become a Level 1 / SAQ-D processor: cards POST to our API, we hold tokens, we forward PAN to PSPs. This requires a QSA, an annual on-site assessment, ASV scans, a documented ISMS, segmented network, HSMs or equivalent KMS, ~£60–120k/year ongoing cost, and roughly 4–8 months before a single real card can be processed. It is a non-starter for a solo founder.
2. Each merchant uses each PSP's hosted fields directly. This works but defeats the product — the merchant has to integrate three different drop-ins and we cannot reuse a card across PSPs (no universal token). This is what most "orchestrators" started with and it limits routing to non-saved-card flows only.
3. **(Chosen.) We integrate a universal third-party vault — Basis Theory.** The merchant's checkout uses Basis Theory's iframe / Elements (or our white-labelled wrapper of it) to capture the card. Basis Theory returns a token. The merchant calls our `/charges` API with the Basis Theory token. Our Go orchestrator calls Basis Theory's "proxy" / "reactor" feature, which detokenises into the exact request format the chosen PSP expects, signs it, and forwards to that PSP. Card data passes through Basis Theory's PCI Level 1 environment, never ours.

**Why Basis Theory over VGS:**

- BT's "reactors" / proxy is more developer-friendly than VGS's revproxy + aliases model and has lower minimums (BT starts free up to ~250 active tokens, then ~$199–$499/mo realistic; VGS minimum tier is materially higher).
- BT supports network tokenisation, which we want for v1.
- VGS is fine if BT pricing stops working — keep the abstraction layer in Go (`vault.Client` interface) so we can swap.

**Implications of choosing the vault model:**

- We file SAQ-A annually. That is a self-questionnaire, not an audit. Cost: ~£0 + a few hours.
- Our checkout SDK must load the Basis Theory iframe; the merchant's checkout HTML cannot accept PAN through anything we wrote.
- We must never log, never persist, and never forward raw PAN. Logging middleware must scrub `card_number`, `cvv`, `expiry` and 13–19-digit numeric runs as a defence in depth.
- We accept the dependency: if Basis Theory has an outage, we cannot vault new cards. Existing tokens still work because BT is on the hot path of every charge anyway. We document this dependency in the merchant SLA.
- SOC 2: not legally required, but UK fintech buyers ask. Plan for SOC 2 Type 1 around month 9, Type 2 around month 18. Start collecting evidence (Vanta or Drata, ~$3k/yr) from week 1.

This decision means everything in §4 onward describes a system that touches **tokens only**.

---

## 4. System Architecture

```
                    ┌────────────────────────────┐
                    │  Merchant checkout (browser)│
                    │  Basis Theory Elements iframe │
                    └──────────┬─────────────────┘
                               │ BT token (no PAN)
                               ▼
        ┌──────────────────────────────────────────────┐
        │ Cloudflare Worker (edge, anycast)            │
        │  - API key auth + HMAC request signature      │
        │  - Idempotency-Key dedupe (Workers KV)        │
        │  - Schema validation, rate limit              │
        │  - Forward to nearest Fly region              │
        └──────────┬───────────────────────────────────┘
                   │ mTLS, signed
        ┌──────────▼─────────────────────────────────────┐
        │ Go Orchestrator on Fly.io (LHR + AMS, primary)│
        │  - Routing engine (rules + bandit)             │
        │  - Idempotency + state machine in Postgres    │
        │  - Calls Basis Theory proxy with target PSP   │
        │  - Records outcome, emits outbound webhook    │
        └──────────┬─────────────────────────────────────┘
                   │
       ┌───────────┴────────────┐
       ▼                        ▼
┌─────────────┐         ┌────────────────┐
│ Basis Theory│ ───────▶│  Selected PSP   │
│ Proxy/Vault │  PAN     │ (Nuvei / CCBill │
│ (PCI L1)    │  detok'd │  / Worldpay HR) │
└─────────────┘          └────────┬────────┘
                                  │ async webhook
                                  ▼
                  ┌──────────────────────────┐
                  │  Webhook ingester (Worker│
                  │  → Fly orchestrator)     │
                  └──────────┬───────────────┘
                             ▼
        ┌────────────────────────────────────────┐
        │ Ledger: Supabase Postgres (primary)    │
        │   - transactions, ledger_entries (DE)  │
        │ Read replicas / config: Turso (edge)   │
        │   - routing_rules, psp_health, BIN map │
        └────────────────────────────────────────┘
```

### Why multi-region is non-negotiable

Each charge's tail latency is dominated by the slowest leg (orchestrator → vault → PSP). If our orchestrator is 120 ms from the merchant we add 120 ms to checkout. More importantly, if we only run in one region and that region degrades, every merchant on the platform is hard-down at the worst possible moment (Black Friday, Boxing Day, Vape Friday). A payment orchestrator that fails closed is worse than no orchestrator.

Minimum viable HA: **two Fly regions, LHR primary and AMS secondary**, with the orchestrator stateless behind Fly's anycast load balancer. The single shared piece of state (Postgres) is the failure domain we accept on the free / cheap tier; we mitigate by keeping each charge's *required* state on the request itself (idempotency key, attempt counter) and by writing outcomes asynchronously where safe.

### Can this run on the pure free tier? No.

Fly's free allowances changed in 2024 — there is no longer a free Hobby tier covering two always-on VMs in two regions. Realistic minimum:

- Fly.io: Launch plan, 2× shared-cpu-2x 1GB in LHR + AMS, ≈ $15–25/mo combined.
- Supabase Pro (because we need ≥1GB DB, daily backups, point-in-time restore on the ledger): $25/mo.
- Turso: free tier OK at the start (routing rules + BIN data fit easily).
- Basis Theory: free up to evaluation, then $199/mo Starter once we have any real traffic.
- Postmark (transactional email for receipts/alerts): $15/mo.
- Cloudflare Workers Paid: $5/mo (we will exceed 100k req/day on webhooks alone).
- Domain + Cloudflare WAF essentials: ~$2/mo amortised.

**Floor: ~$260/mo before a single transaction.** This is covered by one merchant doing £100k/mo at 0.3% (≈ £300/mo).

---

## 5. Routing Engine

### Phase 1 — Rules (MVP)

A merchant configures an ordered list of rules. Each rule is `(predicate, action)` where action is `route_to(psp_id)` or `route_to_pool(pool_id)`. First match wins. Predicates are AND-combined over: country, currency, amount range, BIN range, MCC, card brand, card type (credit/debit/prepaid).

Default fallback rule = `route_to_pool([primary, secondary, tertiary])` with cascading retry on retriable declines.

#### Concrete example for a UK CBD merchant

```
1. IF country=UK AND brand=Visa  AND amount<=£200 → Nuvei      (best UK Visa rate)
2. IF country=UK AND brand=Mastercard           → Worldpay HR  (better MC auth rate)
3. IF country in (DE,FR,NL,IE)                  → Trust Payments (EU acquiring)
4. IF currency=USD                              → CCBill
5. ELSE                                         → pool[Nuvei, Worldpay HR, Trust Payments]
```

On a hard retriable decline (codes: 91 issuer_unavailable, 96 system_error, 19 reenter, network_timeout, our own PSP-down circuit-breaker), we cascade to next in the pool, capped at 2 retries and a 4-second total budget.

### Phase 2 — Adaptive (v1)

We maintain, per (PSP, BIN6, country) bucket, an EWMA of auth rate over a sliding window (alpha = 0.1, window = 1000 transactions or 24h, whichever first). Below a min-sample threshold (say 30) the bucket falls back to its parent (PSP, BIN2, country), then (PSP, country).

Routing decision combines:

- **Expected success** — EWMA auth rate, capped to [0.5, 0.99].
- **Expected fee** — per-PSP fee schedule per card type and brand, expressed as bps.
- **Health** — circuit breaker state (closed / half-open / open) per PSP, plus rolling p95 latency.

Score = `expected_success * (1 - fee_bps/10000) * health_multiplier`. Pick argmax. Use Thompson sampling (Beta(α, β) over auth/decline) on top to keep exploring instead of locking in.

### Decline-reason taxonomy

PSPs return wildly different decline codes. We map every PSP's code into a normalised set: `{soft_decline, hard_decline, fraud, do_not_honor, insufficient_funds, expired_card, cvv_fail, avs_fail, sca_required, processor_error, network_error, timeout}`. Routing only retries on `{processor_error, network_error, timeout, do_not_honor}` and only when the merchant has not opted out. Retrying `insufficient_funds` is hostile to issuers and hurts our standing with them — never do it.

---

## 6. PSP Integrations for MVP

Pick three for MVP. Recommendation:

1. **Nuvei** (formerly SafeCharge). UK + EU acquiring, accepts CBD, vape and supplements after underwriting. REST API is good. Endpoints: `POST /ppp/api/v1/openOrder`, `POST /ppp/api/v1/payment`, `POST /ppp/api/v1/refundTransaction`, webhook DMN. Strong PSD2/3DS2 SDK.
2. **Trust Payments** (UK, formerly Secure Trading). Excellent UK underwriting team, takes vape and spirits. JSON API: `POST /jwt/` for tokenised flows, `AUTH`, `REFUND`, `THREEDQUERY` request types. Webhooks via URL Notification.
3. **Worldpay (High Risk programme)** via Worldpay From FIS. Endpoints: `POST /payments/authorizations`, `/captures`, `/refunds`, `/voids`. Webhooks via Worldpay's notification service. Underwriting is slow (4–8 weeks) but auth rates on UK-issued cards are best-in-class.

**Plus a low-risk fallback for non-high-risk merchants who use us for orchestration only:**

4. **Mollie**. UK + EU low-risk only, but exceptional API and SEPA/iDEAL coverage. Useful for our supplements vertical where Mollie often underwrites.

**Deliberately deferred:**

- CCBill — strong for adult and nutraceuticals USD, but their API is older and 3DS2 support is awkward. Add in month 4.
- Paymentwall — APMs heavy, only worth it once we add APMs in v2.
- Emerchantpay / Paysafe — enterprise-only sales cycle, not SMB-friendly.

For each PSP integration we ship:

- Auth/capture/refund/void
- 3DS2 frictionless + challenge
- Webhook handler with signature verification
- Settlement-report ingest (daily SFTP for Worldpay, API for Nuvei and Trust)
- A "smoke suite" of 30 canonical card test scenarios run nightly in their sandboxes

PSP integration realistic timeline: 2 weeks per PSP for the integration, plus 2–8 weeks underwriting on the side. Start the underwriting paperwork on day one of the project, in parallel with code.

---

## 7. Data Model

Postgres (Supabase) is the system of record. Turso replicates routing config to the edge.

```sql
merchants (
  id uuid pk,
  legal_name, country, mcc, settlement_currency,
  api_key_hash, hmac_secret_encrypted,
  pci_attestation_signed_at, created_at
)

psp_credentials (
  id uuid pk, merchant_id fk,
  psp_code text,                       -- 'nuvei', 'worldpay_hr', ...
  credential_blob bytea,               -- envelope-encrypted (see §12)
  kek_id text,                         -- which KMS key encrypted it
  status text,                         -- 'active','disabled','underwriting'
  added_at, last_used_at
)

routing_rules (
  id uuid pk, merchant_id fk,
  priority int,
  predicate jsonb,                     -- DSL: {country: ['UK'], brand:['visa'], amount_lte: 20000}
  action jsonb,                        -- {type:'pool', psps:['nuvei','worldpay_hr']}
  enabled bool,
  updated_at
)

payment_methods (
  id uuid pk, merchant_id fk,
  vault_token text,                    -- Basis Theory token, NEVER PAN
  bin6, last4, brand, expiry_month, expiry_year,
  customer_ref text,                   -- merchant's customer id
  network_token_status text,
  created_at
)

transactions (
  id uuid pk, merchant_id fk,
  idempotency_key text,                -- unique (merchant_id, idempotency_key)
  payment_method_id fk null,
  amount_minor bigint, currency char(3),
  state text,                          -- see §8
  attempt_count int,
  current_psp text, current_psp_txn_id text,
  decline_reason_normalised text,
  created_at, updated_at,
  unique(merchant_id, idempotency_key)
)

transaction_attempts (
  id uuid pk, transaction_id fk,
  attempt_no int,
  psp_code text, psp_txn_id text,
  request_blob jsonb, response_blob jsonb,
  success bool, latency_ms int,
  decline_code_raw text, decline_code_normalised text,
  started_at, finished_at
)

webhooks_in (
  id uuid pk, psp_code text,
  signature_verified bool,
  event_id text,                       -- PSP's id, used for dedupe
  raw_body bytea, headers jsonb,
  processed_at, status,
  unique(psp_code, event_id)
)

webhooks_out (
  id uuid pk, merchant_id fk, transaction_id fk,
  event_type, payload jsonb,
  attempts int, next_attempt_at,
  delivered_at, status                 -- 'pending','delivered','dead'
)

ledger_entries (
  id bigserial pk,
  transaction_id fk, attempt_id fk null,
  account text,                        -- 'merchant:<id>:gross', 'psp:nuvei:fees', 'routekit:revenue'
  direction char(2),                   -- 'DR' or 'CR'
  amount_minor bigint, currency char(3),
  occurred_at, source text             -- 'auth','capture','refund','settlement_recon'
)

psp_health (                           -- written by orchestrator, replicated to Turso
  psp_code text, region text,
  ewma_auth_rate float, p95_latency_ms int,
  circuit_state text,                  -- 'closed','open','half_open'
  window_start, window_end,
  pk(psp_code, region)
)
```

Notes:

- `transactions.idempotency_key` is unique per merchant — that is what makes `POST /charges` retry-safe.
- `ledger_entries` is double-entry: every event writes balanced DR/CR rows. Reconciliation is a query over this table joined to PSP settlement reports.
- We never store PAN, CVV, or anything that would make us SAQ-D.

---

## 8. Idempotency, State Machine & Exactly-Once-ish

Payments are not exactly-once on the wire. The system must be exactly-once *as observed by the merchant and the ledger*.

### State machine (transactions.state)

```
created
  └─▶ routing
        └─▶ authorising      (PSP request in flight)
              ├─▶ authorised
              │     ├─▶ capturing
              │     │     └─▶ captured
              │     │           └─▶ refunding ─▶ refunded / partially_refunded
              │     └─▶ voiding ─▶ voided
              ├─▶ requires_3ds   (challenge issued)
              │     └─▶ authorising  (after merchant returns)
              ├─▶ failed_retriable   (cascade to next PSP)
              │     └─▶ authorising
              └─▶ failed_terminal
```

Every transition is written in the same Postgres txn as the `transaction_attempts` row that caused it. We use `SELECT ... FOR UPDATE` on the transactions row to serialise concurrent webhook + API mutations.

### Idempotency

Every write endpoint requires `Idempotency-Key`. The Worker first checks Workers KV (24h TTL); on hit, returns the cached response if the request body hash matches, otherwise 422. On miss, it forwards. The Go orchestrator checks `transactions(merchant_id, idempotency_key)` unique index; on conflict it returns the existing record. Idempotency is therefore enforced at two layers (edge KV and DB unique index), so no race window leaks.

### Webhook dedupe

Inbound webhooks are deduped by `(psp_code, event_id)` unique on `webhooks_in`. We persist first, *then* process; the worker that processes is idempotent against the transaction state machine. Retries are safe.

### Reconciliation job

Nightly at 03:00 UTC, for each (merchant, PSP) pair:

1. Pull the PSP's settlement report (CSV/SFTP for Worldpay, REST for Nuvei and Trust).
2. For each settled txn, look up our transaction by `psp_txn_id`. If state ≠ `captured`, raise an alert; this is either a missed webhook or a divergence.
3. Compute fees from the report and write `ledger_entries` (`psp:<code>:fees` DR, `merchant:<id>:gross` CR with the fee delta).
4. Produce a daily reconciliation report per merchant in the dashboard.

This is the single most important operational job in the whole product.

---

## 9. Webhooks

### Inbound (PSP → us)

Each PSP signs differently. Per-PSP verifier:

- **Nuvei**: HMAC-SHA256 over concatenated DMN fields with merchant secret; constant-time compare.
- **Trust Payments**: site-reference + sitesecurity hash, SHA-256.
- **Worldpay**: signature header, RSA verify against rotating public key; cache JWKS.
- **Mollie**: poll-based — we receive a notification URL ping with only the resource ID, then GET the resource over our authenticated client. Treat it as a "kick" and re-fetch.

Common pipeline: receive at Worker → validate signature at Worker → POST to Fly orchestrator over mTLS with original headers → orchestrator inserts into `webhooks_in` (idempotent on `(psp_code, event_id)`) → 200 OK back to PSP within 5 s. Async worker processes the row and advances the state machine.

Replay protection: reject any webhook whose embedded timestamp is more than 5 minutes off our clock.

### Outbound (us → merchant)

- Payload signed with HMAC-SHA256 using a per-merchant `webhook_signing_secret`. Header: `X-Routekit-Signature: t=<unix>,v1=<hmac>`.
- Retries: 8 attempts at 30s, 1m, 5m, 15m, 1h, 6h, 24h, 72h with full jitter.
- Dead-letter after final attempt; visible in dashboard, replayable.
- Strict ordering not guaranteed — events carry monotonic `sequence` per transaction so the merchant can reorder.

---

## 10. 3DS & SCA (PSD2)

PSD2 makes SCA mandatory in the UK and EU for the vast majority of consumer card-not-present transactions. We must handle 3DS2 challenge flows through our orchestrator without breaking the merchant's checkout and without leaking the choice of PSP to the customer.

### Frictionless flow (~70% of txns post-tuning)

1. Merchant calls `POST /charges` with `payment_method_id` and `browser_data` (UA, timezone, screen, accept headers — these populate the device profile section of the 3DS2 AReq).
2. Orchestrator picks PSP, calls Basis Theory proxy with the PSP's "auth + 3DS" endpoint.
3. PSP's ACS returns a frictionless authentication result (CAVV + ECI). Orchestrator captures, returns success to merchant. Customer never sees a challenge.

### Challenge flow (~30% of txns)

The big subtlety: the 3DS2 challenge is a redirect/iframe to the issuer's ACS, with a `termURL` for where the ACS posts the result. **The `termURL` must be on our domain (e.g. `https://3ds.routekit.io/return`), not on the chosen PSP's domain.** Otherwise the customer would be redirected to nuvei.com mid-checkout on attempt #1, then to worldpay.com mid-checkout on a retry — which is both a UX disaster and leaks our routing decision to the customer.

Implementation:

1. Orchestrator gets the ACS URL, PaReq/creq, and a transaction reference from the chosen PSP.
2. Orchestrator returns to the merchant: `{status:'requires_action', action:{type:'redirect_to_url', url: 'https://3ds.routekit.io/challenge/<token>'}}`.
3. Our challenge page hosts an iframe that POSTs the creq to the issuer ACS with `termURL = https://3ds.routekit.io/return/<token>`.
4. ACS posts back to our return URL. We resolve `<token>` to the in-flight transaction, finalise the auth call to the chosen PSP with the cres, and redirect the customer to the merchant's `success_url` or `failure_url`.

This is non-trivial: it requires us to host a small set of HTML pages, manage 3DS server-side state per challenge (token, expiry, PSP, transaction), and pass a strict EMVCo 3DS2 conformance review for every PSP we add. Budget two weeks of work just for the 3DS plumbing in MVP.

Cascading after 3DS: if attempt 1 successfully authenticates with 3DS but the issuer auth declines for non-3DS reasons (e.g. velocity), we *can* retry on a second PSP and re-authenticate (3DS results are not portable across acquirers in most cases). The merchant can opt out of cross-PSP retries on 3DS-authenticated charges.

---

## 11. Compliance & Legal Posture

**We are a payment orchestrator, not a payment processor, not a PSP, not an EMI.**

We do not take possession of customer funds at any point. Money flows: customer's card → acquirer (the chosen PSP's bank) → PSP → merchant's settlement bank account. Never through us. This is the same model as Spreedly, Primer, Gr4vy, and Stripe Terminal partners.

**Under FCA's Payment Services Regulations 2017, "payment initiation" and "merchant acquiring" are regulated activities. "Forwarding card data and routing" without holding funds is not.** Specifically, the FCA exemption commonly invoked is that we provide a "technical service" to the merchant — see PSR 2017 reg. 3 and PERG 15.3 (technical service exclusion). The orchestrators in our space rely on this same carve-out. We must not:

- Hold or pool merchant funds (e.g. delayed settlement we control).
- Initiate payments on behalf of payers (PISP — that is regulated).
- Hold cardholder accounts (e-money — regulated).

We *should*:

- Register as a Data Controller with the ICO (~£60/yr).
- Get directors AML-trained and write an AML/CTF policy. We don't onboard payers, but we do onboard merchants and we want to KYC them lightly (Companies House check + UBO + sanctions screen) to avoid being a chute for shell-company merchants.
- Take out professional indemnity + cyber liability insurance at £2M aggregate from day one. ~£1,200–£2,500/yr at this scale via Hiscox / Markel.

**Becoming an FCA-authorized Payment Institution (PI / SPI / EMI)** is on the table for year 3 if we want to add settlement, payouts, marketplace splits, or stored balances. It is a 12–24 month process: Authorisation pack, ~£5k application fee, ≥ £125k initial capital (PI; SPI is lighter at £20k turnover cap), board with two qualified individuals, Compliance Officer, MLRO, ICAAP, wind-down plan. Don't start this in year 1 — it kills focus and burns runway.

---

## 12. Security & Key Management

- **Envelope encryption** for PSP credentials. KEK lives in Cloudflare's Workers Secrets / Fly's encrypted secret store; per-merchant DEK generated with libsodium (`crypto_secretbox`) on first credential save, encrypted by KEK, stored alongside the ciphertext. Rotation: rotate KEK quarterly, re-wrap DEKs in a background job. (We use libsodium / age rather than rolling our own AES-GCM because the construction matters more than the algorithm.)
- **mTLS to PSPs** wherever supported (Worldpay, Trust Payments support client certs). Pinned to PSP-supplied CA. Cert rotation tracked in calendar.
- **API authentication** for merchants: API key + per-request HMAC signature `X-Sig: t=<unix>,v1=<hmac>` over method + path + body + nonce. Replay window 5 minutes, nonce cached in Workers KV. This protects against an API key alone being enough to charge.
- **Customer-scoped keys**: every merchant's data, including their PSP credentials, is encrypted with a per-merchant DEK, so a single SQL leak doesn't reveal everyone's PSP credentials.
- **IP allowlists** on merchant accounts (optional opt-in).
- **Audit log**: append-only table `audit_events`, hash-chained (each row contains `prev_hash`), exported nightly to R2 with object-lock. Every credential read, rule change, refund, and login lands here.
- **Secret scanning** in CI (gitleaks). Pre-commit hook against `.env`.
- **Dependency policy**: Go modules pinned, `go mod verify` in CI, weekly `govulncheck`. Renovate for upgrades.
- **Threat model maintained as a doc** (STRIDE, updated each release). This is the kind of artefact a UK fintech panel will ask to see.
- **Bug bounty** via Intigriti once we have any real merchant — start invite-only, low payouts, scope limited to production.

---

## 13. Pricing

Two-line public pricing:

- **Free trial**: first £25k in routed volume, then we invoice.
- **Paid**: 0.30% of processed volume, billed monthly in arrears, no platform fee, no minimums for the first 12 months.

Above £500k/mo volume, slide to 0.20%. Above £2M/mo, custom.

Reference points:

- Spreedly: ~£400/mo + 0.05–0.15% per txn (so a £100k/mo merchant pays ~£550/mo there).
- Primer: undisclosed; reportedly £2k+/mo platform.
- Gr4vy: undisclosed; enterprise-only.

Our 0.3% looks high vs Spreedly's 0.10% but lower vs Spreedly's effective rate at SMB volumes once the £400 fee is amortised. We are explicitly cheaper than Spreedly at < £400k/mo, which is exactly our ICP.

We do **not** take a cut of the PSP's fee. The merchant's PSP contracts are direct merchant-to-PSP. We only invoice for our orchestration service. This keeps us out of MSA / settlement-services classification.

Add-ons: Reconciliation+ (deeper reporting) at £49/mo, Adaptive Routing (the bandit) bundled in v1, Anti-fraud passthrough (Sift/Seon) at cost +10%.

---

## 14. Tech Stack & Libraries

**Backend (Go on Fly.io)**

- Web framework: `chi` (lighter than Echo/Gin, plays well with stdlib `http`).
- DB: `pgx/v5` directly + `sqlc` for typed queries against Supabase Postgres. Avoid an ORM.
- Turso: `libsql-client-go`. Used only for read-heavy config (routing rules, BIN map, psp_health).
- Migrations: `goose`.
- Background jobs: `riverqueue/river` (Postgres-backed, transactional enqueue). Used for outbound webhooks, reconciliation, retries.
- HTTP client to PSPs: stdlib `http` + per-PSP retry/backoff via `cenkalti/backoff/v4`. Each PSP wrapped in its own package with a common `psp.Adapter` interface.
- Vault SDK: Basis Theory Go SDK (or hand-rolled REST — their API is small).
- Observability: OpenTelemetry → Grafana Cloud free tier (10k series, 50GB logs).
- Crypto: `golang.org/x/crypto/nacl/secretbox` (libsodium-equivalent) for envelope encryption; `filippo.io/age` for offline credential exports.
- Webhooks: pattern after Stripe — sign with HMAC, deliver via worker pool, persistent queue, HTTP 2xx = ack, anything else = retry, timeout 30s.
- Helpers: `samber/lo` sparingly. Avoid heavy generic libraries — payment code should be boring.

**Edge (Cloudflare Workers)**

- TypeScript, Hono framework.
- Workers KV for idempotency + nonce cache.
- Workers Queues for buffering webhook ingest before pushing to Fly.

**Frontend (Cloudflare Pages)**

- Next.js (static export) + Tailwind. Auth via Supabase Auth (email magic-link + TOTP). Charts via `recharts`.

**Why Turso AND Supabase?**

- Supabase Postgres = ACID, strong consistency, joins, the ledger, transactional state machine. The single source of truth.
- Turso = read-replicated config close to Workers. Routing rules are read on every charge; pulling them from London Postgres on a Sydney edge would add 200 ms. We replicate routing_rules + psp_health + BIN map to Turso and read from the nearest edge. Writes always go to Postgres.

---

## 15. Build Roadmap (12–16 weeks solo to first merchant)

Week-by-week is over-precise; phase plan is more honest. Critical-path items run in parallel with PSP underwriting paperwork that takes weeks of calendar time.

**Phase 0 (week 0, before any code).** Register UK Ltd. Open business bank. ICO registration. PI/cyber insurance bind. Send underwriting application packs to Nuvei, Trust Payments, Worldpay HR — these run in the background for 4–8 weeks. Sign up Basis Theory dev account. Sign up Supabase, Turso, Fly, Cloudflare. Write threat model v0.

**Phase 1 (weeks 1–4): skeleton.** Go orchestrator boilerplate with chi + pgx + sqlc. Schema + migrations. Merchant onboarding API and dashboard auth. Basis Theory iframe in dashboard demo. End-to-end "create token, persist payment_method" path with no PSPs yet. Single Fly region.

**Phase 2 (weeks 4–8): one PSP.** Integrate Trust Payments first (cleanest API). Auth, capture, refund, void, webhook ingest. State machine. Idempotency. Outbound webhook delivery to a test merchant endpoint. 3DS2 frictionless + challenge flow with our hosted return URL. Pass Trust's certification.

**Phase 3 (weeks 8–11): second + third PSPs and routing.** Integrate Nuvei and Worldpay HR. Rule-based routing engine. Failover on retriable declines. Reconciliation job v0 (read settlement reports). Multi-region Fly (LHR + AMS). Outbound webhook DLQ.

**Phase 4 (weeks 11–13): hardening.** SOC 2 evidence collection (Vanta). SAQ-A filed. Logging scrubbers proven. Pen test (£3–5k for a 3-day external test, scoped to the API + dashboard + 3DS pages). Threat model refresh.

**Phase 5 (weeks 13–16): first merchant trial.** Onboard one friendly CBD or vape merchant. Run a parallel "shadow" mode for 1 week (we receive copies of their txns but the merchant's existing PSP stays primary), then cutover. Daily check-in. Fix-forward.

Adaptive routing, BIN-based routing, network tokens, more PSPs — all v1 work after week 16.

---

## 16. Free-Tier Reality Check

**You will not stay on the pure free tier. Plan the spend now.**

| Component | When it stops being free | Realistic monthly cost |
|---|---|---|
| Cloudflare Workers | When you cross 100k req/day on webhooks + edge auth (week 8 with one busy merchant) | $5 |
| Fly.io | Immediately — you need 2 always-on regions. Free tier no longer covers that. | $20–40 |
| Supabase Postgres | When you need PITR + > 500MB. Day 1 once real merchant data lands. | $25 |
| Turso | Probably free indefinitely at this scale | $0 |
| Basis Theory | After the eval period, ~30 days post first real merchant | $199 |
| Postmark / Resend | Once you exceed free tier (10k/mo) — month 2 | $15 |
| Vanta (SOC 2 evidence) | Optional but you should start week 1 | $250/mo amortised |
| Pen test (one-off) | Week 13 | $4,000 one-off |
| Insurance | Day 1 | $150/mo amortised |
| Domain + Cloudflare | Day 1 | $2 |
| BIN database | When you start BIN routing (v1) | $50 |

**Recurring floor before any merchants: ~$700/mo in months 0–3, ~$900/mo from month 3 once Vanta + Basis Theory paid kick in.**

Break-even at 0.30%: ≈ £230k–£300k of routed volume per month. One serious CBD merchant gets you there.

This is materially more expensive than the other ideas in this repo. That is the price of being in payments. **Do not pretend otherwise.**

---

## 17. Go-to-Market

This is a B2B sale to operators who are already burned by Stripe. They live in specific Slack/Discord groups and trade shows. Channel order of priority:

1. **Direct outbound to UK CBD, vape, and craft-spirits brands** that are visibly running on Stripe (look for `stripe.network` headers or Stripe's checkout) — they will be frozen sooner or later. Ship a one-page "what to do when Stripe freezes you" SEO doc; rank for that exact phrase. Cold email founder + ops lead.
2. **Partnership with high-risk merchant account brokers** (e.g. Card Cutters, PayKings UK, MaxiCard). They place merchants with high-risk PSPs already; we are the orchestrator the merchant integrates *once* across all the brokers' placements. Pay 10–20% rev share on referred volume for year 1.
3. **Trade groups & events**: CBD Industry Association UK, UK Vaping Industry Association (UKVIA), Wine and Spirits Trade Association. Sponsor a small breakfast at one of their events. Cost: £2–5k.
4. **SEO**: long-tail "Stripe alternative for [vertical]" pages, written with real comparison detail (which PSPs accept which products, what the auth rates look like, what underwriting takes). Each vertical page = one piece of evergreen content with 12-month half-life.
5. **Open-source the SDKs** (TypeScript + PHP + Python wrappers around the API). Free distribution + dev-credibility.
6. **Speak at one fintech meetup in London** about payment orchestration, ideally hosted by a UK PI — recruiters and prospects in the same room.

We do not run paid ads in year 1. CAC for SMB high-risk merchants via paid is brutal; the channels above are cheaper and more credible.

---

## 18. Open Questions / Risks

- **Chargeback blame.** We are not the merchant of record. The PSP and the merchant share chargeback liability. But when chargeback rate spikes the merchant *will* email us first. Mitigation: clear contractual carve-out, dashboards that show chargeback rate per PSP, and a "chargeback prevention" pack of best practices.
- **PSP-side disputes about our routing.** A PSP whose share of volume drops because the bandit picked someone else may threaten to off-board *us* (not the merchant). Mitigation: explicit written agreement with each PSP that we are an orchestrator and free routing is permitted; never sign volume commitments.
- **FCA scope creep.** If we ever start holding funds even briefly, even for a "convenience" feature, we are now an EMI. Aggressively police this in product reviews. Document it in our internal policy.
- **Single-PSP outage taking *us* offline.** Cascading failover only helps if the merchant has multiple PSPs. If they have one, our value evaporates the day that PSP goes down. Onboarding wizard insists on at least two PSP credentials before going live.
- **Key compromise.** PSP API keys are the keys to the merchant's money. Mitigation: envelope encryption, audit log, anomaly alerting on key reads, scoped IAM in PSPs where supported, rotation runbook, breach playbook.
- **Basis Theory single point of failure.** Their downtime is our downtime for new card capture. Mitigation: keep the abstraction so we can fall back to PSP-hosted fields per-PSP if BT is down (degraded mode, no cross-PSP routing for that customer).
- **3DS conformance regressions.** Each PSP's 3DS implementation drifts. We need nightly E2E 3DS challenge tests in each PSP's sandbox or we will silently break SCA in production.
- **AML on merchants.** Even though we don't handle funds, onboarding a shell company that uses us to launder via a real PSP is a reputational disaster. Light KYB + sanctions screening at onboarding is non-negotiable.
- **Becoming "too big to be unregulated"**. There is some risk that the FCA changes the technical service exclusion or that scale alone draws scrutiny. We should monitor FCA consultations on PSR and budget for legal opinion at £1M ARR.

---

## 19. Career / Portfolio Framing

For a UK fintech panel — Wise, Revolut, Adyen UK, Form3, GoCardless, Modulr, ClearBank, Primer, Yapily, TrueLayer — this project demonstrates, concretely:

- **High-availability distributed systems**: multi-region active-active, idempotency, exactly-once semantics, circuit breakers, bandit-driven routing. These are exactly the topics in their staff/senior interviews.
- **Payment orchestration** as a domain. Hot in 2024–2026 (Primer, Gr4vy, Spreedly's growth, Adyen launching orchestration). You can speak fluently about decline reason taxonomy, 3DS2 challenge flow termURL routing, network tokens, settlement reconciliation, double-entry ledgering — all directly.
- **PCI awareness without naivete**: the choice to outsource the vault rather than build it is itself a senior-engineer signal. You can explain why SAQ-D is a non-starter for a small team and what the SAQ-A line is.
- **Regulatory navigation**: understanding the FCA technical-service exclusion vs PI authorisation, plus AML obligations, is a differentiator vs typical full-stack candidates.
- **Security engineering background applied**: envelope encryption design, threat modelling, mTLS, append-only audit log — concrete artefacts to walk through in an interview.
- **Product judgment**: niche-down to a vertical the incumbents won't serve, price below them, ship fewer PSPs but make them work. UK fintech hiring managers respond to this kind of thinking.

Bring three artefacts to interviews: the Mermaid architecture diagram, the state machine, and the threat model. That is enough to anchor a 45-minute deep-dive.
