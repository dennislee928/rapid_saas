# 04. iGaming Bonus Abuse & Multi-Accounting Detection API

A UK-focused, low-latency anti-fraud API that helps mid-tier iGaming operators stop bonus hunters and multi-accounters in real time. A lightweight TypeScript SDK captures device fingerprints and behavioural signals from the browser. A Cloudflare Worker scoring endpoint returns a 0-100 risk score in under 50 ms. Heavier graph-link analysis (clustering accounts that share fingerprint subsets) runs asynchronously on a Go service on Fly.io with Turso (libSQL) as the source of truth.

The codename used throughout this doc is **TiltGuard**.

---

## 1. Product Overview & Target Users

### Who the customer is

- **Mid-tier UK-licensed operators**: Tier-2/3 sportsbooks, online casinos, and bingo brands doing GBP 5M-100M GGR/year. They sit underneath Bet365/Entain/Flutter but above hobbyist white-labels. Examples of the *type* of buyer (not customer list): MrQ, Slots Magic, Mr Vegas, BetVictor (smaller verticals), BoyleSports UK arm.
- **White-label casino operators on platforms like SoftSwiss, EveryMatrix, BetConstruct, Pronet Gaming**. These platforms host hundreds of brands and the brand owner is responsible for marketing/bonus spend. They are the most price-sensitive and the most exposed to bonus abuse, because they reuse a single deposit funnel that fraudsters quickly learn to game.
- **Affiliate-driven brands** that lose 30-60% of first-deposit bonuses to fraud rings during launch promos.
- **Marketing/CRM teams inside operators**: secondary buyer. They care because abused FTD bonuses break their LTV models and CAC payback dashboards.

### Why they would buy TiltGuard over SEON / Sift / IPQS / Kount

| Vendor | Approx UK price | Weakness TiltGuard exploits |
| --- | --- | --- |
| **SEON** | ~GBP 0.05-0.12 per check, GBP 1k-3k/month minimums | Strong but pushes every customer into the full suite (email/phone/IP modules); overkill for an operator that only needs sign-up bonus fraud. Reasonably opaque scoring. |
| **Sift** | Enterprise pricing, typically USD 30k+/year minimum | Heavy onboarding, US-centric, payments-fraud DNA, slow to tune for iGaming-specific patterns (e.g., free-spin abuse). |
| **IPQualityScore (IPQS)** | ~USD 0.001-0.005 per IP/email check | Excellent IP reputation but weak browser/device fingerprinting. Customers stack IPQS *plus* something else. |
| **Kount (Equifax)** | Enterprise only | Slow integration, not aimed at white-labels. |
| **Iovation (TransUnion TruValidate)** | Enterprise only | Aging device fingerprint stack, expensive. |
| **FingerprintJS Pro / Fingerprint.com** | USD 0.0008-0.005 per identification | Excellent fingerprint, but no risk reasoning. Customer still needs to build the rules layer. |

**The TiltGuard wedge**: a single, opinionated `/score` endpoint, GBP-priced, sub-50 ms, that bundles device fingerprint + IP/ASN + behavioural + velocity into one number with reason codes - specifically tuned for iGaming sign-up/login/deposit flows. No 6-week integration. JS tag plus 1 backend call. UK data residency by default (LHR).

### What I am explicitly *not* building

- Full identity / KYC (AU10TIX, Onfido, Veriff own this).
- Payment fraud / 3DS2 risk (Ravelin, Ekata, Riskified).
- Affiliate fraud detection (different signal set - that is post-acquisition).

---

## 2. Core Features (MVP vs. v1)

### MVP (week 0-8, what gets a UK operator to sign a paid trial)

1. **`POST /v1/score`** with three call types: `signup`, `login`, `deposit`. Returns a 0-100 score, a recommendation enum (`allow`, `review`, `deny`), and up to 5 reason codes.
2. **JS SDK** (`tiltguard.min.js`, ~14 KB gzipped) that collects fingerprint and behavioural signals and produces an opaque `visit_token`. The customer's backend then calls `/v1/score` with `visit_token + account_id + event_type`.
3. **Reason codes** (machine-readable, e.g. `FP_REUSE_HIGH`, `IP_DC`, `UA_MISMATCH`, `MOUSE_LINEAR`, `WEBDRIVER_PRESENT`, `JA3_KNOWN_BAD`, `EMAIL_NEW_DOMAIN`, `VELOCITY_SIGNUP_BURST`).
4. **Reviewer dashboard (read-only MVP)**: list of high-risk events, fingerprint detail panel, linked accounts list, reason code breakdown. Built as a Next.js app on Cloudflare Pages.
5. **Customer rules**: simple JSON rules per tenant (e.g., `if score >= 70 and country = "GB" then deny`).

### v1 (week 8-16, what gets to the second paying customer)

6. **Reviewer actions**: mark as fraud / not fraud, freeze account, push back to operator via webhook.
7. **`POST /v1/feedback`** for label injection. Labels feed back into model training.
8. **Async re-scoring webhook**: when graph linking discovers a new connection, re-score affected accounts and POST to the customer's webhook.
9. **Linking explorer**: visualise the cluster around a flagged account (D3 force graph).
10. **Velocity dashboards**: signups per ASN per hour, fingerprint reuse heatmap.
11. **A/B mode** ("shadow mode"): customer can run TiltGuard for 30 days with no enforcement and see what would have been blocked. This is the killer demo.
12. **XGBoost-based scoring** layered on top of rules (gradient boosting trained on labels customers feed back).

### v2 (post-PMF, not in scope here)

- Server-side TLS JA3/JA4 capture via a dedicated edge ingest endpoint.
- Behavioural biometrics escalation (continuous typing/mouse profile during gameplay).
- Native mobile SDKs (iOS/Android), critical for sportsbook apps.
- Consortium signal sharing (anonymised hash-only fingerprint reuse across customers - real moat).

---

## 3. System Architecture

### High-level design

The hot path (sign-up/login/deposit decisioning) must complete in under 80 ms p99. It runs entirely on Cloudflare's edge with a single Turso read. The cold path (graph linking, clustering, model retraining) runs asynchronously on Fly.io in LHR.

```
                           BROWSER
                              |
                    [tiltguard.min.js SDK]
               collects fingerprint + behaviour
                              |
                  POST /collect (Cloudflare Worker)
                              |
                    writes raw event to
                  Cloudflare Queues (async)
                              |
                  returns visit_token (JWT-ish)
                              |
                              v
                   CUSTOMER BACKEND (operator)
                              |
                     POST /v1/score
                  { visit_token, account_id,
                    event_type, ip, headers }
                              |
                              v
              +--------------------------------+
              | Cloudflare Worker  /v1/score   |
              | - decode visit_token           |
              | - read fingerprint hash from   |
              |   Workers KV (hot cache)       |
              | - read tenant rules from KV    |
              | - read recent reuse counts     |
              |   from Turso (libSQL edge      |
              |   replica, LHR)                |
              | - compute rule-based score     |
              | - return JSON in <50ms p99     |
              +--------------------------------+
                              |
                              | (fire-and-forget)
                              v
                   Cloudflare Queue: "events"
                              |
                              v
              +--------------------------------+
              | Fly.io Go service (LHR)        |
              | - graph linking worker         |
              | - MinHash / SimHash compute    |
              | - cluster updates              |
              | - XGBoost re-score             |
              | - writes back to Turso primary |
              | - emits webhooks on cluster    |
              |   discovery                    |
              +--------------------------------+
                              |
                              v
                       Turso (libSQL)
                  primary in LHR, replicas at
                   the edge for read scaling
```

### Why this topology

- **Worker-first hot path**: a Worker in LHR adds ~5 ms over a direct origin. KV reads are <10 ms. Turso libSQL edge replicas read in 5-15 ms. We keep the score calculation purely synchronous and rule-based on the hot path; nothing on this path requires Go.
- **Async heavy lifting**: pairwise fingerprint comparison across millions of records is O(N) per new event with naive SQL. We push the new event onto a Cloudflare Queue, and the Go consumer on Fly.io performs MinHash bucketing and writes link edges into Turso. When a new edge crosses a confidence threshold, we emit a webhook back to the customer.
- **Fly.io for Go**: Go's small footprint fits the 256 MB free VM; we get a long-lived process for batch jobs and a place to run XGBoost inference (via `gorgonia` / ONNX runtime / leaves) without Worker CPU limits.
- **Turso (libSQL)**: 9 GB free, EU/UK regions, edge read replicas, native Go driver. Right-sized for a Year-1 ceiling of ~50M events.

### Why not put graph linking on the hot path

Graph link discovery is "find every other account whose fingerprint is at least Jaccard 0.85 to this one". Even with bucketing, that's hundreds of comparisons. We instead precompute and cache a *fingerprint-cluster ID* per device, lookup-only on the hot path. New events that don't match any existing cluster get assigned a new cluster, then the Go worker decides asynchronously if the cluster should merge with another.

---

## 4. Fingerprinting Signal Catalog

Each signal gets a weight in the rule-based scorer. Weights are learnable from feedback. The list below is what I actually plan to ship in the SDK. Anything marked *server-side* is captured at the Worker, not in JS.

| # | Signal | What it captures | Stability | Evasion | Counter-evasion |
| - | ------ | ---------------- | --------- | ------- | --------------- |
| 1 | **Canvas fingerprint** | SHA-256 of pixel output of a hidden 2D canvas drawing a fixed string with mixed fonts, gradients, emoji | Very high across sessions on same device/GPU | Canvas Defender / Brave shields randomise pixels. AntiDetect/Multilogin spoof to a "popular" hash. | Detect *instability* across sessions on the same device (legit users have stable canvas hash; spoofers' hashes change every session). Maintain a deny-list of known anti-detect canvas hashes. |
| 2 | **WebGL renderer + vendor** | `WEBGL_debug_renderer_info` UNMASKED_RENDERER and UNMASKED_VENDOR strings, plus a hashed render of a 3D scene | High | Some browsers redact (Firefox, Brave). Spoofers fake to "ANGLE (Intel, Intel(R) UHD Graphics ...)". | Cross-check with `navigator.gpu.requestAdapter()` if available; mismatch with claimed renderer = high entropy fraud signal. |
| 3 | **AudioContext fingerprint** | Compute a hash of an OfflineAudioContext oscillator output | High; varies subtly by hardware/OS | Anti-fingerprint extensions add audio noise. | Look for *impossible* values (audio fingerprint that never appears in our population) and for browsers reporting AudioContext support but throwing on actual sampling. |
| 4 | **Font enumeration** | List of installed fonts via measuring text width/height with each candidate font (Flash-style trick); on Chrome use the Local Font Access API where granted | Medium-high | Fingerprint browsers ship a curated short list. | We MinHash the font set. A list that exactly matches the default macOS/Windows install with no third-party fonts (no Adobe, no Office, no game fonts) is suspicious for an adult sports bettor demographic. |
| 5 | **Screen + device metrics** | screen.width/height, devicePixelRatio, window.outerHeight - innerHeight (chrome height), color depth | Medium | Easy to spoof. | Combine with `screen.availWidth - screen.width` (taskbar offset) which is harder to fake consistently. |
| 6 | **Timezone + language + locale** | Intl.DateTimeFormat().resolvedOptions().timeZone, navigator.language, navigator.languages | Medium | Spoofable but often forgotten. | Compare with IP geo: GB IP claiming `Asia/Shanghai` timezone is a near-perfect proxy hunter signal. |
| 7 | **navigator.hardwareConcurrency / deviceMemory** | Logical CPU cores; RAM bucket | Medium | Spoofable in modern anti-detect browsers. | Cross-check with WebGL renderer (a "GeForce RTX 4090" reporting `hardwareConcurrency: 2` is impossible). |
| 8 | **BatteryManager** | charging state, level | Low (deprecated/removed in modern Chrome 103+, Firefox; still in some Safari/older mobile) | n/a | **Note**: do not rely on this; flagged for removal. Use only as a corroborating low-weight signal where present. |
| 9 | **TLS JA3/JA4** (server-side) | JA3 / JA4 hash of the TLS ClientHello at our edge | Very high; per-browser-version | Custom TLS stacks (curl-impersonate, Go bots) mimic Chrome but rarely exact. JA4 is harder to spoof than JA3. | Maintain a JA3/JA4 reputation list. Mismatch between User-Agent (claims Chrome 124) and JA3 (curl-impersonate-110 signature) is a near-certain bot. |
| 10 | **Mouse trajectory entropy** | Capture mouseMove deltas over the page; compute jerk (3rd derivative), straight-line ratio, dwell on form fields | High behavioural signal | Headless browsers with `puppeteer-extra-plugin-stealth` add fake mouse movement; the curves are too smooth (cubic Bezier). | Compute the variance of inter-event timing. Real humans have heavy-tailed timing; bots have near-Gaussian. |
| 11 | **Typing cadence** | Inter-keystroke interval distribution on the email/password fields | High behavioural | Replays of recorded human typing. | Look at backspace rate, copy-paste detection (`paste` event), and field focus order. |
| 12 | **Headless detection** | `navigator.webdriver === true`, missing `window.chrome.runtime`, `navigator.plugins.length === 0`, `navigator.languages === []`, presence of `__nightmare`, `_phantom`, CDP-specific globals | Hard signal when present | Stealth plugins patch all of these. | Run *positive* checks: e.g. `Notification.permission` should not be `"denied"` by default in headless; `navigator.permissions.query({name: "notifications"})` returning `"denied"` while permission is `"default"` is a known Chromium-headless bug. |
| 13 | **WebRTC local IP leak** | mDNS/STUN candidate gathering reveals local network IP | Medium (often blocked now) | mDNS hashing introduced in Chrome 76+ blocks raw local IPs. | Use *number of candidates*; VPNs/proxies typically yield 0 or only relay candidates. |
| 14 | **Storage estimates** | navigator.storage.estimate() (quota and usage) | Low-medium | Easy to spoof but rarely is. | Fresh installs of fingerprint browsers consistently report a near-zero usage; combined with old-account_id is suspicious. |
| 15 | **TLS-level header order + HTTP/2 frame fingerprint** (server-side) | The order of HTTP/2 SETTINGS frames and HPACK header order | Very high | Almost no spoofer gets this right yet. | Hash the order, store per-tenant top-10 list of OK orders, anything new becomes a soft signal. |
| 16 | **IP / ASN reputation** (server-side) | ASN, datacenter flag, residential proxy lists, recent abuse | High | Residential proxy networks (Bright Data, etc.) bypass ASN flag. | Combine with timezone mismatch and JA3 to detect residential proxy users; the IP looks clean, but the JA3 + locale combo betrays them. |

### Implementation note

The SDK should *not* compute scores client-side. It produces signals, hashes the heavy ones (Canvas, font set, audio) on-device, and ships the hashes plus raw small fields up. The scoring decision is server-side. The SDK exposes only `tiltguard.collect()` and returns a short-lived JWT-style `visit_token` (signed by the Worker).

---

## 5. Risk Scoring Model

### Day-1: rule-based weighted features

Each event yields a vector of feature flags and continuous values. We compute:

```
raw_score = sum( weight_i * normalised_value_i )
score    = clamp(0, 100, sigmoid(raw_score) * 100)
```

Initial weights (tuned by hand from prior fraud research; will be re-fit once we have labels):

| Feature | Weight | Rationale |
| ------- | -----: | --------- |
| Fingerprint hash exact match against a different account in last 30 days | +35 | Strongest single signal |
| Fingerprint MinHash Jaccard >= 0.85 against different account | +25 | Catches near-matches |
| IP is on a known datacenter ASN | +15 | Easy bonus hunter tell |
| IP is on a residential proxy list (Spur, IPQS feed) | +20 | Stronger than datacenter |
| JA3 in known-bad list | +25 | Almost always a bot |
| Headless detection positive | +30 | Hard fail |
| Mouse trajectory near-Gaussian or absent | +15 | Bot or scripted user |
| Timezone vs IP-geo mismatch | +12 | Proxy hunter |
| UA major version vs JA3 mismatch | +20 | Spoofed UA |
| `webdriver === true` | +40 | Hard fail (close to deny) |
| Email at disposable domain (mailcheck list) | +10 | Soft signal |
| Velocity: >5 signups from same fingerprint in 24h | +25 | Bonus farming |
| Velocity: >3 signups from same /24 IPv4 in 1h | +15 | Cluster |

Recommendation thresholds (tenant-overridable):

- `score < 30` -> `allow`
- `30 <= score < 70` -> `review` (operator routes to manual review queue or delays bonus credit)
- `score >= 70` -> `deny`

### v1: gradient boosting (XGBoost)

Once a customer has been live for ~30 days and has fed back at least ~2,000 labels (`fraud` / `not_fraud`), we train a per-tenant XGBoost binary classifier on the same features plus interaction features.

We keep the rule-based score as a fallback and as a *feature into* XGBoost. Output is calibrated with isotonic regression so the 0-100 number is interpretable as approximate fraud probability.

### Bootstrapping when we have zero labels

This is the single hardest operational problem. Strategies:

1. **Synthetic adversarial set**: spin up a test farm with Multilogin, Linken Sphere, Kameleo, GoLogin, plus headless puppeteer-stealth, against my own honeypot signup form. Generate ~50k labelled-fraud events. This becomes the *positive* class for the seed model.
2. **Self-supervised cluster labels**: any cluster with >=10 accounts sharing the same fingerprint hash within 30 days is auto-labelled fraud (false-positive risk acceptable - shared family device is rare beyond 3-4 accounts at a single operator).
3. **Public datasets**: Kaggle's IEEE-CIS fraud dataset and the limited iGaming-relevant features in PaySim. Not great but lets us calibrate sklearn pipelines.
4. **Customer cold-start template**: a default model derived from synthetic + auto-labelled clusters, served until the customer has enough volume to fit their own.
5. **Shadow-mode (week 1-30 of any new customer)**: TiltGuard scores everything but blocks nothing. Operator side-by-sides with their existing manual review outcomes; we collect labels for free.

---

## 6. Linking Algorithm

### Goal

Given a new fingerprint vector for `account_X`, find every existing account whose fingerprint is "close enough" that they likely share a device. Output: edges in an account graph with confidence scores. Connected components (clusters) become *device groups*, often 1:1 with a real human or a fraud ring.

### Approach

1. **Compute a MinHash signature** of the high-entropy multi-valued signals: font list, plugin list, mime types, supported audio/video codecs, browser feature set. Use 128 hash functions (`twmb/murmur3` in Go), 16 bands of 8 rows for LSH.
2. **Compute SimHash** of categorical fingerprint signals (UA family, OS, GPU vendor, color depth bucket, screen resolution bucket, timezone) for fast Hamming-distance bucket lookup.
3. **Bucket lookup**: at write time, we insert each MinHash band into a Turso table indexed on `(band_idx, band_hash)`. New event lookups read O(16) rows per band -> typical candidate set of 5-50 accounts.
4. **Pairwise rescoring**: for each candidate, compute true Jaccard on the original sets and a weighted similarity over the rest of the fingerprint vector. We use a learned threshold (start at 0.82) for "same device".
5. **Edge insertion** with confidence (Jaccard + behavioural agreement). Edges live in `links` table.
6. **Cluster maintenance**: union-find (Tarjan offline; in code, `golang-set/v2` with a parent map flushed periodically) over edges with confidence >= 0.6. The cluster ID is denormalised onto every event so the hot path can read it in one join.

### Why probabilistic record linkage (not exact match)

Real devices change state daily: fonts get added, browsers update, GPU drivers update, screen resolution changes when a user docks a laptop. Exact-hash matching gives us false negatives (~25% miss rate by my synthetic tests). Jaccard on the underlying multi-sets recovers ~95% of true reuse.

### Why SQL (Turso) and not a graph DB

For our scale (target Year-1: ~5M devices, ~50M events, ~10M edges), well-indexed SQL on Turso is faster and cheaper than spinning up Neo4j / DGraph. The queries we need are:

- "Find all accounts in the same cluster as account X" -> indexed lookup on `cluster_id`.
- "Show me the edges of cluster C with confidence >= 0.7" -> indexed range scan.
- "When this new event arrives, which existing clusters share at least one MinHash band?" -> indexed lookup on `(band_idx, band_hash)`.

None of these are recursive or deep traversal. If we ever need 3+ hops with weight propagation (consortium-level cross-tenant clustering), we can shadow-replicate to Neo4j Aura free tier.

If the customer specifically wants graph queries via SQL, we keep Supabase Postgres in our pocket as an alternative; recursive CTEs handle most graph queries at our scale.

---

## 7. Data Model

Turso / libSQL (SQLite dialect). Conventions: ULIDs for IDs, all timestamps UTC integer epoch millis.

```sql
-- Tenants (operators)
CREATE TABLE tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    api_key_hash TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT 'lhr',
    plan TEXT NOT NULL DEFAULT 'starter',
    created_at INTEGER NOT NULL
);

-- Each browser visit; immutable
CREATE TABLE events (
    id TEXT PRIMARY KEY,                -- ULID
    tenant_id TEXT NOT NULL,
    visit_token TEXT NOT NULL,
    account_id TEXT,                    -- operator's account id, may be NULL for pre-signup
    event_type TEXT NOT NULL,           -- signup | login | deposit | custom
    device_id TEXT NOT NULL,            -- our derived device id
    cluster_id TEXT,                    -- denormalised for hot-path reads
    fingerprint_hash TEXT NOT NULL,     -- exact full-vector SHA256
    minhash BLOB NOT NULL,              -- 128 x uint32
    simhash INTEGER NOT NULL,           -- 64-bit
    ip TEXT NOT NULL,
    asn INTEGER,
    country TEXT,
    ja3 TEXT,
    ja4 TEXT,
    user_agent TEXT,
    score INTEGER,                      -- final 0-100
    recommendation TEXT,                -- allow | review | deny
    reason_codes TEXT,                  -- JSON array
    created_at INTEGER NOT NULL
);
CREATE INDEX idx_events_tenant_created ON events(tenant_id, created_at DESC);
CREATE INDEX idx_events_device ON events(tenant_id, device_id);
CREATE INDEX idx_events_account ON events(tenant_id, account_id);
CREATE INDEX idx_events_cluster ON events(tenant_id, cluster_id);

-- Devices: one row per derived device fingerprint
CREATE TABLE devices (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    cluster_id TEXT,
    first_seen INTEGER NOT NULL,
    last_seen INTEGER NOT NULL,
    seen_count INTEGER NOT NULL DEFAULT 1,
    fingerprint_hash TEXT NOT NULL,
    canvas_hash TEXT,
    webgl_hash TEXT,
    audio_hash TEXT,
    fonts_minhash BLOB,
    UNIQUE(tenant_id, fingerprint_hash)
);

-- Operator accounts (mirror, lazily created when first scored)
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,                -- operator's account id, scoped per tenant
    tenant_id TEXT NOT NULL,
    cluster_id TEXT,
    devices_count INTEGER DEFAULT 0,
    risk_max INTEGER DEFAULT 0,
    flagged_at INTEGER,
    PRIMARY KEY (tenant_id, id)
) WITHOUT ROWID;

-- Edges between accounts (and between devices via the same accounts)
CREATE TABLE links (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    source_account_id TEXT NOT NULL,
    target_account_id TEXT NOT NULL,
    via TEXT NOT NULL,                  -- 'device' | 'fingerprint_minhash' | 'ip_asn' | 'payment'
    confidence REAL NOT NULL,           -- 0..1
    evidence TEXT,                      -- JSON
    created_at INTEGER NOT NULL,
    UNIQUE(tenant_id, source_account_id, target_account_id, via)
);
CREATE INDEX idx_links_source ON links(tenant_id, source_account_id);

-- LSH bucket index for MinHash
CREATE TABLE minhash_bands (
    tenant_id TEXT NOT NULL,
    band_idx INTEGER NOT NULL,
    band_hash INTEGER NOT NULL,
    device_id TEXT NOT NULL,
    PRIMARY KEY (tenant_id, band_idx, band_hash, device_id)
);

-- Reason code registry (for i18n + dashboard)
CREATE TABLE reason_codes (
    code TEXT PRIMARY KEY,
    severity INTEGER NOT NULL,
    description TEXT NOT NULL
);

-- Per-tenant rule overrides
CREATE TABLE customer_rules (
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    rule_json TEXT NOT NULL,            -- compiled to a small DSL
    enabled INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (tenant_id, name)
);

-- Labels from /v1/feedback
CREATE TABLE feedback (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    event_id TEXT,
    account_id TEXT,
    label TEXT NOT NULL,                -- fraud | not_fraud | unknown
    note TEXT,
    created_at INTEGER NOT NULL
);

-- Webhooks
CREATE TABLE webhook_deliveries (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    url TEXT NOT NULL,
    payload TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_status INTEGER,
    next_attempt_at INTEGER,
    delivered_at INTEGER
);
```

---

## 8. API Contracts

### `POST /v1/score`

Authenticated with `Authorization: Bearer <tenant_api_key>`.

```json
{
  "visit_token": "eyJhbGciOi...",
  "account_id": "op_acct_5f3b...",
  "event_type": "signup",
  "context": {
    "promo_id": "FTD200",
    "deposit_amount_minor": 2000,
    "deposit_currency": "GBP",
    "email_hash": "sha256:6ad...",
    "phone_hash": "sha256:a1c..."
  },
  "client": {
    "ip": "82.31.44.99",
    "user_agent": "Mozilla/5.0 ...",
    "accept_language": "en-GB,en;q=0.9"
  }
}
```

Response (typical 25-45 ms):

```json
{
  "score": 78,
  "recommendation": "deny",
  "reason_codes": [
    {"code": "FP_REUSE_HIGH", "weight": 35, "detail": "Fingerprint matches 4 accounts in last 30d"},
    {"code": "JA3_ANOMALY", "weight": 18, "detail": "JA3 e7d705a3286e... not seen on this UA family"},
    {"code": "TZ_GEO_MISMATCH", "weight": 12, "detail": "tz=Asia/Shanghai, ip_country=GB"}
  ],
  "device_id": "dev_01HX...",
  "cluster_id": "clu_01HX...",
  "linked_account_count": 4,
  "request_id": "req_01HX...",
  "latency_ms": 31
}
```

### `POST /v1/link`

Used by reviewers / batch tooling to ask: who is this account linked to?

Request:
```json
{ "account_id": "op_acct_5f3b..." }
```

Response:
```json
{
  "account_id": "op_acct_5f3b...",
  "cluster_id": "clu_01HX...",
  "linked_accounts": [
    {"account_id": "op_acct_a1...", "confidence": 0.94, "via": ["device", "ip_asn"]},
    {"account_id": "op_acct_99...", "confidence": 0.71, "via": ["fingerprint_minhash"]}
  ],
  "evidence_summary": {
    "shared_canvas_hash": true,
    "shared_webgl": true,
    "ip_overlap_24h": 2,
    "minhash_jaccard": 0.91
  }
}
```

### `POST /v1/feedback`

Customer marks a past event/account as fraud or not-fraud. Feeds the model.

```json
{
  "event_id": "evt_01HX...",
  "account_id": "op_acct_5f3b...",
  "label": "fraud",
  "note": "Chargeback after FTD bonus claimed; same KYC docs as 3 other accounts."
}
```

Response: `202 Accepted`.

### Webhook `POST -> {customer_url}` on async re-scoring

```json
{
  "type": "tiltguard.cluster_updated",
  "occurred_at": 1745596800000,
  "tenant_id": "ten_01HX...",
  "cluster_id": "clu_01HX...",
  "delta": {
    "added_accounts": ["op_acct_8d...", "op_acct_b2..."],
    "new_score_max": 88
  },
  "recommendation": "freeze_bonus",
  "signature": "sha256=..."
}
```

Signature is HMAC-SHA256 over the raw body using a tenant webhook secret.

### Errors

Standard problem+json. Hot path returns `200` even for `deny` decisions; only auth, rate-limit, and validation errors are non-2xx. Operators must not depend on HTTP status to decide allow/deny.

---

## 9. Privacy & Compliance

UK GDPR + the UK Gambling Commission (UKGC) rules, primarily LCCP (Licence Conditions and Codes of Practice) Social Responsibility Code, plus the 2024 Customer Interaction guidance.

### Data minimisation

- The SDK never collects PII (no email, no phone, no name). The customer's backend hashes any PII (SHA-256 with a tenant pepper) before sending.
- Fingerprint hashes are stored, not the raw values. We retain the raw font list and Canvas pixel hash for 30 days for debugging, then prune.
- IP is stored truncated to /24 (IPv4) or /48 (IPv6) after 7 days for trend analytics. Full IP retained 7 days only.

### Lawful basis

Article 6(1)(f) GDPR - **legitimate interest** - "fraud prevention" (Recital 47 explicitly recognises this). Backed by:

- A documented Legitimate Interest Assessment (LIA) provided to customers as a template.
- A standard processor DPA (data processing agreement) ready to sign with each tenant; TiltGuard is the *processor*, the operator is the *controller*.

### DPIA (Data Protection Impact Assessment)

Required because we do "systematic monitoring on a large scale" and process "data revealing behaviour" of individuals. Template DPIA shipped with onboarding, covering:

- Purpose: prevent bonus abuse/multi-accounting; reduce financial harm.
- Data categories: technical telemetry, no special-category data, no payment data.
- Risks: incorrect scoring leading to denial of service; mitigation: review tier between 30-70.
- Subject rights: right to object, right to access, right to rectification handled via the operator (not direct).

### Retention

- Hot events table: 90 days.
- Aggregated device+cluster records: 24 months.
- Feedback labels: 36 months (model training).
- Hard delete on tenant offboarding within 30 days.

### Data residency

- Turso primary in `lhr` (London Heathrow). Replicas in `lhr` and `cdg` only - no US replicas by default.
- Fly.io app pinned to `lhr` and `mad` (Madrid as failover).
- Cloudflare Workers run globally but do not write durable state outside Workers KV (which has regional metadata; we keep only ephemeral session data there).
- For customers requiring strict UK-only, we offer a `region: gb-only` flag that disables `cdg` and `mad` failovers (paid tier).

### UKGC considerations

- TiltGuard is *not* a regulated activity (we are a B2B technology provider, not an operator). However, the UKGC Source of Funds + Customer Interaction policies require operators to have anti-fraud controls; TiltGuard helps them evidence those controls.
- We will publish a **transparency report** quarterly: aggregate denial rates, average score distribution, false-positive rate from feedback. Useful for UKGC inspections customers go through.
- We must not be *the* decision-maker for self-exclusion or affordability checks. TiltGuard explicitly disclaims those use cases (UKGC guidance prefers human-in-the-loop for player-affecting decisions).

### Documentation we ship

- DPA template
- LIA template (operator fills in, we provide the security/processing portion)
- Sub-processor list (Cloudflare, Fly.io, Turso, Spur, IPQS)
- Pen test report (annual; year 1 use a UK CREST-accredited firm; ~GBP 6-10k)

---

## 10. Latency Budget

Target: **p50 < 30 ms, p99 < 80 ms**, measured server-side from `/v1/score` request entering the Worker to response leaving the Worker.

End-to-end from the operator's UK datacentre to TiltGuard and back:

```
[Operator BE in LHR]
   |  ~1 ms  TLS handshake (reused, HTTP/2 keepalive)
   v
[Cloudflare Edge LHR]
   |  ~3 ms  TLS terminate, route to Worker
   v
[Worker /v1/score]
   |  +2 ms  parse + auth (api key hash table in KV; cached on isolate)
   |  +5 ms  decode visit_token (HMAC verify)
   |  +6 ms  Workers KV: tenant rules + tenant hot device cache (one read)
   |  +12 ms Turso libSQL edge replica (one read: events count + cluster size)
   |  +3 ms  rule evaluation (~50 features, integer math)
   |  +2 ms  serialise response, log to Tail Worker
   v
[Worker -> CF Edge -> Operator]  ~1 ms
   total Worker time: ~30 ms
   total wall time:    ~35 ms
```

**Async path (not on the budget):**
```
Worker -> Cloudflare Queue   (~5 ms, fire-and-forget)
Queue  -> Fly.io LHR Go svc  (~50-200 ms processing + DB writes)
```

### Where we save when the budget is tight

- **No Turso read at all on second visit**: the device's MinHash bucket info is cached in Workers KV keyed on `device_id` for 60 s. Hit rate after warm-up: ~70%.
- **No KV read for tenant rules**: rules are bundled into the Worker isolate's in-memory cache on cold start (Workers Durable Object holds the latest version, isolates pull at boot).
- **No JSON parsing on hot path**: we accept request as JSON but the response builder uses pre-templated strings.

### What blows the budget

- Turso write on the hot path. **Do not write inline** - all writes go to the Queue and a Worker consumer.
- DNS for outbound third-party calls. **Do not call IPQS/Spur on the hot path** - we run a daily snapshot pulled into Workers KV. The hot path consults KV.

---

## 11. Anti-Evasion Roadmap

Once attackers know TiltGuard is in use (and they will, within weeks - bonus hunters compare notes on Telegram and SBC forums), they will probe and try to fingerprint our SDK. The mitigations:

### Phase 1: SDK obfuscation (week 0-8)

- The SDK is built per-customer with Cloudflare Worker on-the-fly per-tenant code generation. Each tenant gets a different variable layout, function-name hash, and signal-collection order.
- The SDK ships as a 1x1 invisible iframe that posts a `MessagePort` back to the parent. Removing the script tag does not remove the collection (iframe is loaded from the `tiltguard.io` worker domain).
- Critical detection logic (webdriver checks) runs *after* a 50-200 ms random delay, breaking attackers' synchronous bypass shims.

### Phase 2: signal rotation (week 8-16)

- Every 14 days the Worker pushes a config change via Durable Object that disables some signals and enables others (we always collect more than we use). Attackers who patch their fingerprint browser to fake "Canvas + WebGL" miss the week we rely on AudioContext + JA3.
- The list of "signals in use today" is itself encrypted client-side and only decrypted on the Worker.

### Phase 3: server-side TLS fingerprinting (week 12+)

- Run a small Go ingest service on Fly.io behind a dedicated subdomain `t.tiltguard.io`. The SDK pings that endpoint. We capture the TLS ClientHello (ja3/ja4) and HTTP/2 frame order from the connection. The score endpoint joins on this server-side signal that the attacker cannot fake from JS.
- We use `dreadl0ck/tlsx` or `open-ch/ja3` in Go to parse the ClientHello. JA4 via `FoxIO-LLC/ja4`.

### Phase 4: behavioural biometrics escalation (post-PMF)

- Continuous mouse/typing capture during gameplay, not just signup. A fraud ring that solves the signup gauntlet often slips on the gameplay biometrics (a real first-time slot player has different timing rhythms than the third account on the same human).

### Phase 5: consortium signals (year 2)

- Anonymised, hashed fingerprint reuse counts shared across customers (with explicit opt-in and a bloom-filter privacy layer). This is the moat - one operator alone cannot see that a fingerprint hit 17 brands; the consortium sees it instantly.

---

## 12. Pricing & Metering

Per-API-call tiered pricing in GBP, matching how SEON and IPQS sell into UK iGaming.

| Tier | Price | Volume | Min monthly | Notes |
| ---- | ----- | ------ | ----------- | ----- |
| **Pilot** | Free | 5,000 calls/mo | 0 | 30-day trial; full features incl. shadow mode |
| **Starter** | GBP 49/mo + GBP 0.04 per call after 5k | up to 50k calls/mo | GBP 49 | small white-label brands |
| **Growth** | GBP 299/mo incl. 25k calls, GBP 0.025 per call after | up to 250k | GBP 299 | mid-tier, the sweet spot |
| **Scale** | GBP 999/mo incl. 100k calls, GBP 0.015 per call after | up to 2M | GBP 999 | tier-2 operators |
| **Enterprise** | Custom (target GBP 4-12k/mo) | unlimited + SLA + dedicated tenancy | n/a | Contracted; UK-only residency, 99.95% SLA |

### Reference points (publicly inferable from customer reports / sales calls)

- SEON: ~GBP 0.05-0.12 per check at GBP 1-3k/month minimum.
- IPQS: ~GBP 0.001-0.004 per IP-only check, but the equivalent device-fingerprint product is closer to GBP 0.02-0.05.
- FingerprintJS Pro: from USD 0.0008 per identification at the lowest tier.
- Sift: typically GBP 25-40k+/year minimum for iGaming.

We deliberately undercut SEON by 40-60% on per-call price for the mid-tier, and we are on par with FingerprintJS Pro at the entry point but ship the rules engine that FingerprintJS makes you build yourself.

### Metering implementation

- Worker increments a Durable Object counter per tenant per minute.
- Daily rollup writes to Turso `usage_daily`.
- Soft limits trigger 429 with `Retry-After`; hard limits triggered only at +20% of plan to avoid disrupting an operator mid-promo.
- Stripe invoice generated monthly; UK VAT handled via Stripe Tax.

### Annual contracts

Enterprise tier signs annual; pay 12x monthly upfront for a 15% discount. Helps cashflow and lets us commit to UK-only residency.

---

## 13. Tech Stack & Libraries

### SDK (browser, TypeScript)

- TypeScript + Vite for the build.
- `tsup` for SDK bundling, output ESM + UMD, 14 KB gzipped target.
- Inspired (in spirit) by `fingerprintjs/fingerprintjs` v3 open-source edition; we do not vendor it - reimplemented to avoid AGPL concerns and to control signal rotation.
- `comlink` for the iframe bridge.
- Build per-tenant via a Cloudflare Worker that emits a deterministically-shuffled bundle.

### Cloudflare Worker (hot path)

- Workers TypeScript + `wrangler`.
- `hono` web framework (fast, tiny, has KV/DO bindings as first-class).
- Routing via Worker subdomains: `api.tiltguard.io` for `/v1/score`, `cdn.tiltguard.io` for the SDK, `t.tiltguard.io` for the TLS-capture beacon.
- Workers KV for: api-key cache, tenant rules cache, IPQS/Spur snapshot, hot device cache.
- Cloudflare Queues for the events firehose.
- Durable Objects for per-tenant rate limit counters.
- `@libsql/client` (libsql HTTP) for Turso reads from the Worker.

### Go service (Fly.io, async path)

- Go 1.22+, single binary.
- `chi` (`go-chi/chi/v5`) for HTTP routing.
- `tursodatabase/libsql-client-go` or pure SQLite via `mattn/go-sqlite3` against a local Turso replica (libsql server embedded mode).
- `twmb/murmur3` for MinHash hashing.
- `ekzhu/minhash-lsh` for the LSH bucket implementation (Go port; we may fork).
- `dreadl0ck/tlsx` or `salesforce/ja3` for JA3 fingerprinting.
- `FoxIO-LLC/ja4` (Go bindings) for JA4.
- `dgraph-io/sroar` or `RoaringBitmap/roaring` for compact device-set bitmaps in cluster math.
- `prometheus/client_golang` + Fly.io Prometheus integration for metrics.
- `slog` (stdlib) + `lmittmann/tint` for human logs in dev, JSON for prod.
- `riverqueue/river` (Postgres-only, so swap for) -> custom queue worker reading from Cloudflare Queues via the HTTP pull API.
- For XGBoost inference in Go: `dmitryikh/leaves` (reads native XGBoost models, no CGo needed). Training in Python (`xgboost` + `scikit-learn`), export to JSON, load in Go.

### Dashboard (Cloudflare Pages)

- Next.js 14 (App Router) statically exported.
- `shadcn/ui` + Tailwind.
- Auth: Clerk free tier or Supabase Auth (only for the dashboard, not the API).
- D3 force graph for cluster explorer.

### Infra glue

- Terraform for Cloudflare + Fly.io + Turso provisioning.
- GitHub Actions for CI; tests run on pushes, deploys on tag.
- Sentry self-hosted on Fly.io for error reporting (free tier suffices).
- Plausible Analytics for dashboard usage metrics.

---

## 14. Build Roadmap (8-12 weeks solo)

**Week 1**: SDK skeleton (Canvas, WebGL, font enumeration, screen, timezone, hardwareConcurrency, webdriver). Cloudflare Worker `/v1/score` returning a stub score. Turso schema. Tenant signup flow.

**Week 2**: Risk scoring rule engine. KV caching. Wire SDK -> Worker -> Turso read path. Hit p99 < 80 ms on a synthetic load test.

**Week 3**: Cloudflare Queue + Fly.io Go consumer. Write events. Compute MinHash. Build LSH index in Turso. Manual cluster verification on synthetic data.

**Week 4**: Linking algorithm complete; cluster IDs flow back to events table. Reason codes registry. JA3/JA4 capture via `t.tiltguard.io` beacon.

**Week 5**: Build the synthetic adversarial test farm (Multilogin trial, GoLogin trial, puppeteer-stealth). Run 50k synthetic signups. Tune weights. Hit > 90% recall on synthetic fraud, < 3% FPR on synthetic legit.

**Week 6**: Reviewer dashboard MVP (event list, fingerprint detail, cluster explorer). Auth. Stripe billing.

**Week 7**: `/v1/feedback` + webhook delivery. Documentation site (Mintlify or Nextra on Pages). DPA, LIA, DPIA templates.

**Week 8**: First demo to a UK operator (warm intro via LinkedIn / SBC London follow-up). Shadow-mode integration guide. Onboarding script.

**Week 9-10**: Pilot with first operator (free tier, shadow mode). Iterate on false positives. Build velocity rules.

**Week 11**: XGBoost training pipeline (Python, exported model loaded in Go via `leaves`). A/B between rule-based and XGBoost.

**Week 12**: Pen-test prep + first paid plan signed.

If anything slips, the cuttable items are: XGBoost (rule-based ships first), JA3/JA4 (can wait), reviewer dashboard polish (week 6 can be MVP-ugly).

---

## 15. Free-Tier Risk & Scaling Triggers

| Resource | Free limit | Break point | Action |
| -------- | ---------- | ----------- | ------ |
| Cloudflare Workers | 100k requests/day, 10 ms CPU/req | ~2-3M `/v1/score` calls/month | Move to Workers Paid (USD 5/month, 10M req incl., +USD 0.50/M after) |
| Workers KV | 100k reads/day free, 1k writes/day free | ~30k events/day | Workers Paid covers 10M reads/M; we keep KV for caches only |
| Cloudflare Queues | 100k operations/month free | ~3k events/day | Paid: USD 0.40/M ops |
| Turso | 9 GB storage, 1B row reads/month, 25M row writes/month free | Row reads usually OK; row writes hit first at ~20M events/M | Scaler plan USD 29/month |
| Fly.io | 3x shared-cpu-1x 256MB free | CPU saturates above ~50 events/sec sustained (MinHash compute) | Scale to dedicated-cpu-1x 1GB at ~USD 5-10/month |
| Cloudflare Pages | unlimited static | n/a | n/a |

### Hard signals to watch

- Worker CPU ms p99 climbing above 8 ms = we are about to hit the 10 ms cap; profile and move work async.
- Turso write QPS > 200/s = batch writes, switch to bulk insert.
- Queue backlog > 30 s = scale Fly.io Go workers (separate concurrency-1 machines).
- Workers KV read p99 > 15 ms = colocate the data in the Worker isolate (DO-backed cache) instead.

### Cost ceiling guard

The product remains profitable as long as **avg infra cost per call < GBP 0.005**. Above that, the Starter tier loses money. We monitor a `cost_per_call_p50` metric and alert when > GBP 0.003.

---

## 16. Go-To-Market

### Channels (in priority order)

1. **SBC Summit Barcelona (Sept) + ICE London (Feb)**. ICE is the world's biggest iGaming event; SBC is the second. A GBP 1,500 visitor pass + 2 days of meetings is the single highest ROI activity. Don't buy a booth; pre-book 25 meetings via the SBC matchmaking app.
2. **iGaming Business Slack + LinkedIn groups**: GamCare, Gambling Insider, EGR Global. Post the *quarterly transparency report* as content. Comment thoughtfully on UKGC announcements.
3. **White-label platform integrators**: SoftSwiss, EveryMatrix, BetConstruct, Pronet Gaming - these companies host tens of brands and resell technology to them. Land an integration deal with one ($DEAL_VALUE = 20 brands at GBP 299/mo each). Cold-email their CTO/Head of Risk.
4. **Affiliates and consultants**: people like Steve Donoughue, Dan Iliovici, the Right2Bet network. Pay them a referral cut (15% first-year revenue).
5. **Open-source SDK**: open-source the *base* fingerprint SDK (MIT) without the scoring backend. Hacker News, GitHub stars, dev-tools awareness; pulls in the dev-led smaller operators.
6. **Content**: a UK-iGaming-fraud blog. Topics: "Why your FTD bonus abuse rate is 18% higher than you think", "JA4 vs JA3 for bot detection in 2026", "What UKGC's Customer Interaction Guidance means for fraud teams". Each post is 1,500-2,500 words, targeted at Heads of Risk.
7. **LinkedIn outbound** to "Head of Fraud", "Head of Risk", "Head of Payments" titles at UK-licensed operators with 50-500 employees. ~120 personalised messages a week.

### Sales motion

- Free 30-day shadow-mode pilot. No procurement needed (under GBP 1k/month threshold lets most operators sign without legal).
- Convert via the side-by-side report: "in 30 days, TiltGuard would have stopped GBP X of bonus abuse you paid out." If X > 10x annual contract value, it sells itself.
- Annual contract via DocuSign; standard MSA + DPA we ship.

### Pricing anchors in conversation

Lead with "GBP 299/month gets you up to 250k checks - same as ~50k SEON checks at their lower tier." Operators benchmark on price per check; this framing wins.

---

## 17. Open Questions / Risks

### UKGC licensing implications

- TiltGuard is a B2B technology vendor; it does not need a UKGC operating licence under the Gambling Act 2005. But: if our scoring drives *automated* customer-affecting decisions (denying bonuses, freezing accounts), the UKGC's Customer Interaction guidance prefers a documented human-in-the-loop. We will require operators to sign that they retain final decision rights, and we recommend the `review` middle band for ambiguous scores. Ambiguity to resolve: whether the UKGC's "Software Licence" applies if our SDK runs inside the operator's gaming session - probably not (we are not "gambling software" per the technical definition), but we will get a written legal opinion before going live with the first paid customer (~GBP 2-4k from a UK gaming-law firm like Harris Hagan or Wiggin).

### False positive cost

- A real punter denied a bonus and forced into manual review is a churned customer. The UK iGaming customer LTV is ~GBP 200-500. A single 1% FPR on a 100k-call month is ~1,000 angry users -> GBP 200k-500k of LTV at risk for the operator. Our threshold defaults must err toward `review`, not `deny`. This is also the strongest reason to ship shadow-mode first.

### Competitor moats and our counter

- SEON has 6+ years of fraud data across thousands of customers - their model is better trained. **Counter**: we win on price, latency, UK residency, and iGaming-tuned signals; we accept being a #2 in raw model accuracy until we accumulate data.
- FingerprintJS Pro has the best-in-class fingerprint stability. **Counter**: we are the rules-and-decision layer they don't ship, and we are GBP-priced for UK SMB.
- The big risk: SEON or FingerprintJS launches an explicit iGaming SKU at our price. **Counter**: lock in white-label platform partnerships early and build the consortium signal moat.

### Operational risks

- Cloudflare Queue downtime breaks the async path; the hot path keeps working but cluster updates pause. We measure backlog and degrade gracefully (feature-flag off cluster reads if backlog > 5 min).
- Turso edge replica staleness: edge replicas can be up to a few seconds behind. Acceptable for our use case; we do not require linearisable reads.
- The synthetic adversarial test farm (Multilogin, GoLogin) may itself be a UKGC compliance question if used to test against live operators. We will only run it against our own honeypot; never against a customer environment without a signed pen-test contract.

### Open product questions

- Do we offer per-event pricing or per-MAU pricing? Per-event matches the iGaming sales motion better, but per-MAU might be stickier. Default per-event for v1.
- Do we charge the customer for `review` outcomes the same as `allow`/`deny`? Yes, all `/v1/score` calls are billable. Easy to explain; matches SEON.
- Do we expose the raw fingerprint to the operator? No, only derived signals + reason codes. This protects our IP and reduces operator GDPR scope.

---

## 18. Career / Portfolio Framing

This project demonstrably proves to a UK iGaming or fintech hiring panel:

- **Low-latency systems engineering**: a real p99 < 80 ms global edge endpoint backed by an async Go pipeline. Few candidates can talk concretely about Worker CPU budgets, KV read latency, and Turso edge replica staleness in the same breath. This is exactly what Bet365, Sky Betting, Flutter, Entain, and the larger fintechs (Wise, Revolut, Monzo) interview for at the senior backend level.
- **Device fingerprinting depth**: actual, named signals (Canvas, WebGL, AudioContext, font MinHash, JA3/JA4) with evasion and counter-evasion analysis. This is at the level of a SEON or Iovation senior engineer JD.
- **Anti-fraud / risk scoring**: rule engines, weighted features, calibrated XGBoost, the bootstrap-with-no-labels problem. Maps to Job titles like "Senior Engineer, Fraud Platform" at Wise, Revolut, Bet365 Risk Engineering, Flutter Group Risk.
- **GDPR + UKGC literacy**: lawful basis selection, DPIA template, retention policy, UK data residency. These compliance questions come up in *every* UK fintech/iGaming interview, and most engineers fumble them. Showing a finished DPIA template and a UKGC-aware customer interaction model is the single most differentiating soft signal.
- **Full-stack delivery**: Cloudflare edge (Workers + KV + Queues + Pages), Go on Fly.io, libSQL/Turso, TypeScript SDK, Stripe billing, Terraform, GitHub Actions. Rare to see one candidate ship all of these.
- **Security-engineering instinct**: per-tenant SDK obfuscation, signal rotation, JA3/JA4 capture, adversarial testing with real fingerprint browsers. This is what a UK operator's CISO wants to see in a Head of Engineering or Senior Backend hire.

For interviews, the talking points are:

1. *Walk me through the architecture* -> hot path vs cold path, why Workers, why Turso, why MinHash bucketing.
2. *How did you handle no labels at the start?* -> synthetic adversarial farm, self-supervised cluster labels, shadow mode.
3. *What's the latency budget and how did you defend it?* -> 30 ms target, KV caches, no inline writes, Queue async.
4. *What's the GDPR story?* -> Article 6(1)(f), DPIA, retention, residency, no PII in fingerprints.
5. *Where would this break if a real adversary targeted it?* -> SDK rotation, JA3/JA4 server-side, behavioural escalation, consortium signals.

The portfolio one-liner: *"I built a sub-50 ms anti-fraud API on Cloudflare Workers and Go that detects bonus abuse for UK iGaming operators, with a UKGC-compliant data model and a learned linking algorithm that clusters multi-account rings."*
