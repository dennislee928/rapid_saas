# 05 — UK Adult/Creator Economy Compliance & Anti-Piracy Middleware

> Codename: **GateKeep** (Compliance Gateway) and **Reclaim** (Anti-Piracy Bot).
> Two products under one platform brand: provisional working name **Aegis.adult** (or a more neutral "Aegis Trust" if registrar/payment partners get squeamish about the TLD).

---

## 1. Product overview & target users

The UK Online Safety Act (OSA), enforced by Ofcom from 25 July 2025 for "highly effective age assurance" on Part 5 services, has put a regulatory gun to the head of every adult site that serves UK traffic. Fines are the greater of £18m or 10% of global turnover, and Ofcom has already opened formal investigations into multiple non-compliant operators. At the same time, the creator economy on OnlyFans/Fansly continues to bleed revenue to leak forums and tube sites — industry estimates (CAM4/StripChat antipiracy disclosures, Bayou Watch reports) put creator losses at 20–30% of net potential revenue.

There are two distinct buyer segments and they share a structural problem: **they are not engineers.**

- **Solo creators on OnlyFans, Fansly, ManyVids, Loyalfans, JustForFans.** They sell content directly. Their pain is leaks. They will pay a few hundred pounds a month if takedowns are visibly working. They do not run their own websites and do not need a gateway — but their *agencies* do (see below).
- **OnlyFans agencies / management firms.** A surprisingly professionalised industry, especially in the UK (London, Manchester) and US. Agencies manage 30–300 creators each. They are the highest-LTV customer because they aggregate demand and are repeat buyers across creators.
- **Small UK adult sites, indie studios, fetish niche operators, paysite networks.** Run by 1–3 person teams, often non-technical owners with a contracted dev. They need the Compliance Gateway *yesterday* because Ofcom enforcement is here. Most are currently either (a) ignoring the law, (b) blocking UK IPs entirely, or (c) bolting on Yoti/Persona via a half-broken iframe and praying.
- **Cam studios and clip sites (whitelabel / Modelcentro / AdmireMe alumni).** Compliance buyer.

**Why they buy from us instead of going to Yoti/Persona/Veriff directly.** This is the core question and the answer is the wedge:

1. **The KYC vendors sell APIs, not solutions.** A small site does not have an engineer who can wrap Yoti's session API, manage JWT issuance, set up cookie scoping across subdomains, build a fallback flow for when the iframe is blocked, handle re-verification, or implement geo-rules so they don't burn money verifying US visitors. We sell the *integration*, deployable in 10 minutes via a CNAME or Worker route.
2. **One platform, two needs.** They also want takedowns. Doing both from one vendor with one invoice and one dashboard is an obvious pitch.
3. **Edge deployment means latency-free.** A Cloudflare-Worker reverse proxy adds ~5ms vs. an origin-side middleware that adds 200–500ms because the AV check has to round-trip to the KYC vendor before any HTML loads.
4. **Compliance documentation.** We give them an Ofcom-evidence pack (logs, configuration audit, retention policy) bundled with the gateway. That is the actual deliverable Ofcom will ask for.
5. **GDPR posture.** We do *not* store ID documents. The KYC vendor does. This is the only safe architecture for the buyer.

---

## 2. Two products, one platform — or split?

**Recommendation: ship the Anti-Piracy Bot (Reclaim) first as a standalone product, and add the Compliance Gateway (GateKeep) as product #2 in months 4–6.**

Argument:

- **Reclaim has a faster sales cycle and zero regulatory exposure.** Selling DMCA takedowns to individual creators is a $99–499/mo impulse purchase. They will swipe a card after a 10-minute demo. The Compliance Gateway is a regulated-vendor sale: the buyer wants SOC2-lite documentation, references, an SLA, and probably wants you on a video call with their lawyer. That is months 6+ work.
- **Reclaim has a cleaner compliance story for *you*.** You are providing IP-protection automation. No KYC-vendor relationship, no data-residency contract, no Ofcom touchpoint. You can take Stripe payments through (probably) — see §9 — using a careful merchant category description. The Compliance Gateway forces you into adult-vertical payment processors immediately.
- **Reclaim builds the customer list that you sell GateKeep into.** Creators have agencies. Agencies have studios. Studios run sites. The same logo on the Reclaim dashboard is the lead source for GateKeep.
- **GateKeep needs a real legal review before launch.** Ofcom's "highly effective age assurance" criteria (technical accuracy, robustness, reliability, fairness) require you to make claims you should not make without a solicitor signing off. Reclaim's DMCA workflow is well-trodden law.

Counter-argument considered and rejected: "Compliance Gateway has a regulatory deadline tailwind — go where the urgency is." True, but urgency without product-market polish equals churn. A buggy AV gateway that fails open is a £18m problem for the customer and a career-ending event for you.

**Platform packaging.** They share a single dashboard, single tenant model, single billing. A customer can subscribe to either or both. Treat them as one product internally for the data model and admin UI, but two separately purchasable SKUs.

---

## 3. Core features (MVP vs. v1)

### Compliance Gateway (GateKeep)

**MVP (week 8 target):**
- Cloudflare Worker reverse proxy. Customer points a CNAME or runs the Worker route on their existing CF zone.
- AV check via a single KYC vendor (recommend Persona for MVP — see §5).
- Signed JWT cookie session: HS256 (with a clear migration path to ES256), 30-day default expiry, host-only, `Secure`, `HttpOnly`, `SameSite=Lax`.
- Verification page: a hosted `/verify` route on `<customer>.gatekeep.aegis.adult` with the KYC iframe and post-verify callback.
- Strict CSP injection on HTML responses for the customer's pages (configurable).
- Bot blocking: drop requests with no `Accept` header, no UA, or matching the CF managed bot list.
- Per-tenant config in Turso: redirect path, allowed origins, JWT secret (per tenant), cookie name.
- Admin dashboard: Cloudflare Pages + Next.js. Shows verification volume, success rate, failure reasons.

**v1 (month 4–6):**
- Re-verification triggers (configurable cadence, e.g. annual; or session-fingerprint-mismatch triggered).
- Geo-rules: only intercept UK visitors (CF country header). Optional country allow/deny lists. This alone halves the customer's verification spend.
- Multi-vendor support (Persona + Yoti + Veriff) with per-tenant choice and automatic failover if primary vendor returns 5xx.
- Hardened JWT with key rotation (JWKS endpoint, `kid` rotation every 30 days).
- Device-binding token to detect cookie theft (a separate `gk_device` cookie containing a hash of `UA + Accept-Language`).
- Audit log export for Ofcom (CSV + signed JSON).
- A/B testing of verification UX flows.
- Self-serve onboarding wizard that auto-detects the customer's CF zone via API.

### Anti-Piracy Bot (Reclaim)

**MVP (week 6 target):**
- Customer uploads images/videos via dashboard. Files are hashed client-side (where possible) or server-side, then deleted from our blob store within 24h. We keep only hashes.
- pHash (perceptual hash) for images. Video gets sampled at 1fps and each keyframe hashed; whole-video hash is the median.
- Curated crawl list: 8 sites for MVP. Recommended starter set (verify each is appropriate at build time, list curated by checking current leak-aggregator landscape): a major reddit-clone forum, two tube sites known for OF leaks, two image-board style sites, two telegram-channel-archive frontends, one mainstream tube site that hosts user uploads. Avoid sites that are pure CSAM-adjacent — those go to NCMEC, not us.
- Hourly crawl per asset against indexed page-image hashes.
- DMCA email generation from a UK + US safe-harbour-compliant template, sent via Postmark. Customer reviews and clicks "send" (human-in-the-loop is the MVP default).
- Match dashboard with thumbnails (matched URL + similarity score + timestamp).

**v1 (month 4–6):**
- ML-based partial-match (CLIP embeddings on Hugging Face Space) for cropped, watermarked, re-encoded leaks.
- Video matching with TMK+PDQF (Facebook's open-source, the actual reference for video CSAM matching, perfect for legitimate use here) on the Hugging Face Space.
- Auto-send mode (customer pre-authorises, no human review).
- Google deindexing requests via the Removals API (Lumen-style submission).
- Bing/Yandex deindex requests.
- Takedown reporting analytics: median time-to-removal per site, sites with worst response, recovered revenue estimate.
- Counter-notice handling workflow (when a host pushes back).
- Subreddit/Discord webhook integrations (notify customer in their channel).
- API for agencies to bulk-onboard creator catalogues.

---

## 4. System architecture

### 4.1 Compliance Gateway (proxy path)

```
                                 [ Visitor ]
                                     |
                                     v
                  +--------------------------------------+
                  | Cloudflare Worker (edge, ~300 PoPs)  |
                  |  - Read gk_session cookie            |
                  |  - Verify JWT (jose, HS256)          |
                  |  - On miss: redirect to /verify      |
                  |  - On hit: proxy to origin           |
                  |  - Inject CSP header                 |
                  +--------------------------------------+
                     |               |                |
            (verified)|         (no JWT)|        (config read)
                     v               v                v
            +-----------------+ +------------+ +-----------------+
            | Customer origin | | /verify    | | Turso (edge)    |
            | (their site)    | | hosted UI  | |  - tenants      |
            +-----------------+ |  iframes   | |  - jwt secrets  |
                                |  KYC vendor| |  - geo rules    |
                                +------------+ +-----------------+
                                       |
                                       v
                                +-----------------+
                                | KYC vendor API  |
                                | (Persona/Yoti)  |
                                +-----------------+
                                       |
                                       v   (webhook on success)
                                +-----------------+
                                | Worker callback |
                                |  - verify HMAC  |
                                |  - mint JWT     |
                                |  - set cookie   |
                                |  - 302 to origin|
                                +-----------------+
```

**Why the proxy MUST be edge-resident:** every request to the customer's site needs the cookie check. Doing this at an origin server adds 100–300ms per request (UK→customer-origin RTT) and forces the customer to either change DNS to point at us (a single chokepoint we must keep up) or run our middleware in their stack (which we said the customer cannot do). Cloudflare Workers run in the same PoP as the user, the cookie check is sub-millisecond, and the customer keeps their existing origin. The Worker is also where bot blocking, CSP injection, and rate limiting are cheapest.

### 4.2 Anti-Piracy pipeline

```
   [ Customer Dashboard (CF Pages, Next.js) ]
                  |
                  | upload assets
                  v
   +------------------------------+
   | Worker: receive upload       |
   |  - stream to R2 temp bucket  |
   |  - enqueue hash job          |
   +------------------------------+
                  |
                  v
   +------------------------------+        +-------------------+
   | Fly.io Go service: Hasher    |------->| Turso             |
   |  - pHash (image)             |        |  - protected      |
   |  - frame-sample + pHash (vid)|        |    assets         |
   |  - delete original from R2   |        |  - hashes         |
   +------------------------------+        +-------------------+
                                                    ^
                                                    |
   +------------------------------+                 |
   | Fly.io Go service: Crawler   |-----------------+
   |  - per-target adapters       |
   |  - residential proxy pool    |       (writes matches)
   |  - polite scrape             |
   |  - extract image URLs        |
   |  - fetch + pHash candidates  |
   |  - Hamming-distance match    |
   +------------------------------+
                  |
                  v (matches above threshold)
   +------------------------------+
   | HF Space (Python, optional)  |
   |  - CLIP embedding match      |
   |  - TMK+PDQF video match      |
   |  - return confidence score   |
   +------------------------------+
                  |
                  v
   +------------------------------+
   | Worker: takedown generator   |
   |  - render DMCA template      |
   |  - resolve host abuse@       |
   |  - send via Postmark         |
   |  - track DSN                 |
   +------------------------------+
                  |
                  v
              [ Customer dashboard alerts ]
```

The Hasher and Crawler are on Fly.io because they are persistent workers with predictable bandwidth needs that would melt the Worker free tier. The HF Space handles only the heavy ML (CLIP, TMK+PDQF) and is woken on demand. Turso stores hashes in libSQL, accessible from both the Worker and the Go services with low latency (it replicates to the edge).

---

## 5. Compliance Gateway design (deep dive)

### 5.1 Exact request flow

```
1. GET https://customer.com/some-page
2. CF Worker intercepts. Reads cookie `gk_session`.
3. If cookie present:
     - Verify JWT signature with tenant's secret (jose)
     - Check exp, iat, iss, aud, kid
     - If valid → fetch origin, inject CSP, return response
     - If invalid → fall to step 4
4. If cookie missing/invalid:
     - 302 to https://verify.gatekeep.aegis.adult/start?return=<encoded-original-url>&tenant=<id>&state=<csrf-token>
     - state cookie set with httpOnly+SameSite=Strict (3-min expiry)
5. /verify renders an iframe loading the KYC vendor's hosted flow.
6. User completes KYC with vendor (selfie + ID, OR estimation, OR digital ID wallet).
7. Vendor calls our webhook https://verify.gatekeep.aegis.adult/callback/<tenant>.
8. Worker:
     - Verify webhook signature (vendor HMAC)
     - Check state cookie matches
     - Read verification result (verified=true, age_over_18=true, verification_id=xxx)
     - Mint JWT (see 5.2)
     - Set Set-Cookie: gk_session=<jwt>; Domain=customer.com; Path=/; Secure; HttpOnly; SameSite=Lax; Max-Age=2592000
     - 302 to original return URL
9. Subsequent requests sail through with the cookie.
```

### 5.2 JWT contents — no PII

```json
{
  "iss": "gatekeep.aegis.adult",
  "aud": "tenant_abc123",
  "sub": "session_<random_uuid>",
  "iat": 1714000000,
  "exp": 1716592000,
  "kid": "k_2026_04",
  "vrf": "persona",
  "vid_hash": "sha256(verification_id + tenant_secret)",
  "ag": true,
  "geo": "GB",
  "v": 1
}
```

- No name, no DOB, no email, no document data.
- `vid_hash` lets us prove later "this session was verified" without storing the verification_id reversibly. The vendor keeps the source of truth.
- `ag` is the only assertion the customer cares about: age-gate passed.
- `kid` enables key rotation.

**Signing key rotation.** HS256 secrets are 32-byte random, generated per tenant on onboarding and stored in Turso (encrypted at rest with a master KMS-style key held in Cloudflare Workers secrets). Every 30 days a new key is generated, the old one stays valid for verification for another 30 days (overlap window), then is deleted. The Worker keeps both old+new in memory and tries them in order. v1 migrates to ES256 with JWKS at `https://verify.gatekeep.aegis.adult/.well-known/jwks/<tenant>.json` so customers can verify tokens themselves if they want defence in depth.

**Cookie scope.** Host-only on the customer's apex (`Domain` attribute omitted, so `Domain=customer.com` — not `.customer.com` which would leak to subdomains). `Secure`, `HttpOnly`, `SameSite=Lax`. Lax (not Strict) because the post-callback 302 is cross-site and Strict would drop the cookie.

**CSRF on callback.** A `gk_state` cookie is set at /verify with `SameSite=Strict, HttpOnly, 3-min expiry`. The callback must receive a matching `state` query param signed with the tenant secret. Without this, an attacker who knows a tenant ID could pre-mint sessions.

### 5.3 KYC vendor comparison

| Vendor | Per-verification | Ofcom-recognised¹ | API quality | UK trust signal | Verdict |
|---|---|---|---|---|---|
| **Persona** | ~$1.00–2.50 (volume tiered) | Indirectly (through customer's own assessment); not on Ofcom's named list | Excellent — clean REST, hosted flow, webhooks, sandbox is good | Mid — US company, GDPR compliant | **Recommended for MVP.** Best DX, fastest integration. |
| **Yoti** | ~£0.70 ID + £0.20 age estimation | Yes — Yoti is the canonical UK age-assurance brand and is named in Ofcom guidance | Good — but more enterprise-y, slower onboarding | High — UK company, brand UK customers recognise | Add as second vendor in v1; required for the high-trust enterprise tier. |
| **Veriff** | ~€1.50–3.50 | Yes for ID-based assurance | Excellent — Estonian, used by major fintechs | Mid — Estonian but UK-active | Add in v1 for redundancy. |
| **OneID** | Per-verification, bank-grade ~£0.50 | Yes — uses UK bank Open Banking IDV | Decent | High — UK bank network | Niche but compelling. Cheap, fast, but only works for users with UK banks. Add as a sub-flow in v1. |
| **Onfido** | ~£1.50–3 | Yes | Mature but legacy DX | High | Skip unless customer demands. |

¹ "Ofcom-recognised" is informal — Ofcom does not certify vendors. The OSA requires the *service* to perform "highly effective age assurance"; vendors publish reports against Ofcom's criteria. We document which vendor we use and the customer's compliance posture in the evidence pack.

**Strategy:** ship with Persona only (best DX, fastest path to working product). Add Yoti by month 3 because UK enterprise will ask for it by name. Add Veriff and OneID for failover and pricing flexibility.

### 5.4 Failure modes

- **Vendor down (5xx, timeout > 10s).** v1 fails over to the next configured vendor. MVP shows a polite "verification temporarily unavailable, please retry in 5 minutes" page and increments a Prometheus-style counter. We do *not* fail open. Customers signed an SLA that promises this.
- **Re-verification cadence.** Default 365 days (matches OSA expectation of periodic re-assurance). Configurable down to 30 days for highly regulated tenants. Re-verification triggers on: cookie expiry, IP geo change of more than 1000km, device fingerprint mismatch, user explicit logout, customer-pushed force-reverify.
- **Cookie theft / age-gate bypass.** This is the hardest threat. The cookie is `HttpOnly` so JS can't read it. But malware on the user's machine can steal cookies wholesale (Stealc, RedLine). Mitigations:
  - Device-binding cookie (`gk_device` = HMAC(UA + Accept-Lang + IP_/24_subnet + tenant_secret)). On JWT verify, also recompute and compare device hash. Mismatch → force re-verification. We tolerate a /24 subnet change to reduce false positives from mobile NAT.
  - Optional `mfa` claim for high-trust tenants requiring re-verification on every new device.
  - JWT short-lifetime mode for paranoid tenants (e.g. 24h) at the cost of more verification spend.
  - Honesty: cookie theft for adult-content age-gating is a well-known open problem. We document it in the threat model and tell tenants what we cannot prevent.
- **User abandons mid-flow.** /verify times out after 30 minutes. Logged but not punished — they just retry.
- **Underage user passes vendor flow.** Vendor liability primarily, but we forward the verification result as-is. We store hashed verification_id so we can purge sessions on vendor's request (e.g. if vendor revokes a verification post-hoc).
- **Tenant compromise.** If a tenant's signing secret leaks, they can self-revoke by rotating in dashboard. Forces every user to re-verify. Single-button rotation.

---

## 6. Anti-Piracy pipeline design (deep dive)

### 6.1 Hashing strategy

- **Images: pHash (DCT-based, 64-bit).** Cheap, robust to scaling/quality changes, weak to crops > 30%. Primary signal.
- **Images v1: pHash + CLIP embedding.** Add CLIP (open_clip ViT-B/32) on the HF Space for any candidate scoring 12–20 Hamming distance (the "maybe" zone). CLIP catches crops, watermarks, mirror flips.
- **Video MVP: keyframe pHash sampling.** Decode at 1fps with ffmpeg, pHash each frame. Match by counting frames whose pHash falls within Hamming ≤ 8 of any frame in the reference; threshold "matched" if ≥ 30% of reference frames matched in candidate.
- **Video v1: TMK+PDQF.** Facebook's video matching pair, open-source, used in production at scale by Meta and NCMEC. Two-stage: TMK for initial filter, PDQF for confirmation. Run on HF Space (it requires Python and OpenCV/FFmpeg). Output: a similarity score 0–1.
- **NeuralHash-style options.** Apple's NeuralHash is closed and was shown to be weak. SimCLR-trained perceptual encoders are stronger — but the engineering cost is high and the false-positive cost is also high (see 6.3). Defer past v1.
- **Per-frame vs whole-video.** Whole-video hashing (e.g. videohash library) is fast but useless for partial leaks (5 minutes of a 60-min stream). We do per-frame fingerprinting against a shingle index.

### 6.2 Crawler design

**Target site list (curated, ~30 sites in v1, 8 in MVP).** Categories:

1. Major adult tube sites with user-uploads (xHamster, SpankBang, etc. — they have proper DMCA agents, fast turnaround).
2. Reddit-clone leak forums (these come and go; maintain a hot-list).
3. Image board / ranchan-style sites that host OF screenshot dumps.
4. Telegram-channel mirrors and discord-leak archive sites.
5. Smaller paysite-leak hosts (the long tail; this is where automation matters most).

**Polite scraping.** robots.txt for adult sites is a polite fiction — most either have no robots.txt or have one written by SEO consultants that doesn't reflect operator intent. Our policy: respect robots.txt where it exists, but document that we are a rights-holder agent acting under DMCA §512 / UK CDPA §97A which legally permits identification of infringing material. Rate-limit to 2 req/s per site, randomised; backoff on 429/503; stop entirely on a polite request from the operator.

**Rotating residential proxies.** The big tube sites Cloudflare-front their content. Free tier won't work — we need residential proxies (BrightData, Oxylabs, Smartproxy). Cost: ~$5–8/GB. Budget assumption: each crawl burns ~50MB, hourly crawls of 8 sites = 9.6GB/day = ~$50/day at retail. Mitigations:
  - Cache aggressively: index page → hash, only re-fetch changed images.
  - Use datacenter IPs first, fall back to residential only on block.
  - Pass cost through to customer: each tenant gets N crawls/day included in plan, more = add-on.

**Cloudflare-bypass strategy.** We do NOT bypass Cloudflare maliciously. Stack:
  - Use a real headless browser (Playwright with stealth plugin) for sites with serious anti-bot.
  - Use FlareSolverr or curl-impersonate for medium difficulty.
  - For sites where we get persistently blocked, document and notify the customer that this site is uncrawlable; offer a manual-submit endpoint where a human flags URLs for takedown.
  - Never use stolen credentials, never login as a user, never circumvent paywalls.

### 6.3 Matching threshold tuning

- **pHash Hamming distance:** ≤ 6 is "match" (auto-actionable in v1), 7–12 is "review", > 12 is "no match". These thresholds are tuned against the OF-watermark dataset assumption of ~30% pixel modification.
- **CLIP cosine similarity:** ≥ 0.95 is "match", 0.88–0.95 is "review".
- **Video TMK+PDQF score:** ≥ 0.8 match, 0.65–0.8 review.

**False positive cost is asymmetric and severe.** A wrongful DMCA against a non-infringing site means:
  - Reputational harm to the customer (the actual claimant).
  - Potential §512(f) liability in the US (knowing material misrepresentation).
  - UK common-law tort liability for malicious falsehood.
  - Our platform getting added to a "DMCA abuse" Lumen-tracking blocklist.

Therefore: MVP is **human-in-the-loop by default**. The customer sees the match with side-by-side preview and clicks "send takedown". v1's auto-send mode is opt-in, requires the customer to e-sign an agency authorisation, and only fires on Hamming ≤ 4 (tighter than manual threshold) with two independent confirmations (pHash + CLIP).

### 6.4 DMCA generation

Template structure (compliant with US DMCA §512(c)(3) and UK CDPA §97A take-down practice):

1. Identification of copyrighted work (description + reference URL on customer's site).
2. Identification of infringing material (full URL + screenshot + match score).
3. Contact info: rights holder name (or pseudonym + designated agent contact), email, phone.
4. Good-faith belief statement: "I have a good-faith belief that the use of the material described above is not authorised by the copyright owner, its agent, or the law."
5. Accuracy statement: "The information in this notification is accurate."
6. Sworn statement under penalty of perjury (US): "Under penalty of perjury, I swear that I am the copyright owner or am authorised to act on behalf of the owner."
7. Physical or electronic signature (we use typed name + customer's authorisation token).

**Agent designation.** Some hosts in the US require the rights-holder to have a registered DMCA agent at the Copyright Office (Section 512(c)(2)). We are *not* an agent — we are a notification automation. The notice goes out *from* the customer (with our infrastructure as the sending mechanism). Email From: header is `takedowns@<customer-slug>.reclaim.aegis.adult`, Reply-To: their real address.

**Sending.** Postmark for transactional. Postmark is more permissive than SES for this content category (still ask sales). For high-volume, fall back to a self-managed Postfix on Fly.io with proper SPF/DKIM/DMARC. Custom From-domain per tenant with verified SPF.

**DSN parsing.** Inbound webhook from Postmark on bounces and replies. Parse common abuse@ auto-responses (template detection: "We have received your DMCA notice", ticket numbers extracted via regex). Update notice status: sent → acknowledged → removed/declined/expired.

**Escalation to deindex requests.** If the host doesn't respond in 14 days (configurable per host), auto-fire a Google Search Console DMCA-removal request via the Lumen-style submission endpoint. Bing equivalent. Yandex equivalent.

---

## 7. Data model (Turso / libSQL)

```sql
-- Multi-tenant root
CREATE TABLE tenants (
  id              TEXT PRIMARY KEY,
  slug            TEXT UNIQUE NOT NULL,
  name            TEXT NOT NULL,
  email           TEXT NOT NULL,
  plan            TEXT NOT NULL CHECK (plan IN ('reclaim_starter','reclaim_pro','gatekeep_starter','gatekeep_pro','combined')),
  created_at      INTEGER NOT NULL,
  jwt_secret_enc  BLOB NOT NULL,         -- encrypted with platform master key
  jwt_kid         TEXT NOT NULL,
  jwt_secret_prev BLOB,                  -- for rotation overlap
  jwt_kid_prev    TEXT,
  reverify_days   INTEGER DEFAULT 365,
  geo_rules_json  TEXT                   -- e.g. {"verify_only":["GB"]}
);

-- Reclaim
CREATE TABLE protected_assets (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  kind            TEXT NOT NULL CHECK (kind IN ('image','video')),
  label           TEXT,
  phash           BLOB,                   -- 64-bit for image, NULL for video
  video_meta_json TEXT,                   -- frame hashes, duration, etc.
  uploaded_at     INTEGER NOT NULL,
  source_deleted  INTEGER NOT NULL DEFAULT 0  -- 1 once we've purged the original from R2
);
CREATE INDEX idx_assets_tenant ON protected_assets(tenant_id);

CREATE TABLE monitored_sites (
  id              TEXT PRIMARY KEY,
  hostname        TEXT NOT NULL UNIQUE,
  category        TEXT,                   -- tube/forum/image_board/telegram_mirror
  scrape_strategy TEXT NOT NULL,          -- 'http_fetch' | 'playwright_stealth' | 'flaresolverr'
  abuse_email     TEXT,                   -- discovered or manually set
  rate_limit_rps  REAL NOT NULL DEFAULT 1.0,
  last_crawled_at INTEGER,
  active          INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE matches (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  asset_id        TEXT NOT NULL REFERENCES protected_assets(id),
  site_id         TEXT NOT NULL REFERENCES monitored_sites(id),
  candidate_url   TEXT NOT NULL,
  match_score     REAL NOT NULL,          -- 0..1 normalized
  hamming         INTEGER,                -- pHash distance if applicable
  candidate_phash BLOB,
  status          TEXT NOT NULL CHECK (status IN ('new','reviewing','approved','rejected','sent')),
  detected_at     INTEGER NOT NULL,
  reviewed_at     INTEGER,
  reviewed_by     TEXT
);
CREATE INDEX idx_matches_tenant_status ON matches(tenant_id, status);

CREATE TABLE takedown_notices (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  match_id        TEXT NOT NULL REFERENCES matches(id),
  rendered_body   TEXT NOT NULL,
  to_address      TEXT NOT NULL,
  from_address    TEXT NOT NULL,
  postmark_msg_id TEXT,
  status          TEXT NOT NULL CHECK (status IN ('queued','sent','bounced','acknowledged','removed','declined','escalated')),
  sent_at         INTEGER,
  responded_at    INTEGER,
  removed_at      INTEGER,
  google_request_id TEXT
);

-- GateKeep
CREATE TABLE kyc_sessions (
  id              TEXT PRIMARY KEY,       -- session_<uuid>, this is the JWT sub
  tenant_id       TEXT NOT NULL REFERENCES tenants(id),
  vendor          TEXT NOT NULL,          -- persona/yoti/veriff/oneid
  vid_hash        TEXT NOT NULL,          -- sha256(vendor_verification_id + tenant_secret)
  age_passed      INTEGER NOT NULL,
  geo_country     TEXT,
  device_hash     TEXT,
  verified_at     INTEGER NOT NULL,
  expires_at      INTEGER NOT NULL,
  revoked         INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_kyc_tenant_expires ON kyc_sessions(tenant_id, expires_at);

CREATE TABLE audit_log (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT,
  event           TEXT NOT NULL,          -- jwt_minted, key_rotated, takedown_sent, etc.
  actor           TEXT,
  meta_json       TEXT,
  occurred_at     INTEGER NOT NULL
);
CREATE INDEX idx_audit_tenant_time ON audit_log(tenant_id, occurred_at);
```

**Crucial property of the KYC table: zero PII.** No name, no DOB, no email, no document number. The vendor holds all of that. We hold a hash of their verification_id, the boolean result, and an expiry. If the user invokes their GDPR right to erasure with us, we can delete the row in milliseconds and have nothing else to expose.

---

## 8. Privacy & compliance

**UK Online Safety Act timeline.**
- 25 July 2025: Part 5 (Pornography Provider) age-assurance duties enforceable.
- 17 March 2025: Children's Access Assessment deadline for Part 3 services.
- Ofcom continues issuing guidance under "Highly Effective Age Assurance" (HEAA) criteria. Vendor compliance is assessed on: technical accuracy, robustness, reliability, fairness.
- Ongoing: Ofcom illegal-content codes, transparency reporting requirements (annual).

**Ofcom HEAA principles we must address in customer documentation:**
1. The age assurance must be technically accurate (we let the customer pick a HEAA-conformant vendor).
2. Robust against bypass (we document JWT/cookie design; no client-side trust).
3. Reliable in normal use (we publish vendor uptime SLAs).
4. Fair to all users (we disclose vendor accuracy across demographics — Yoti publishes this; we surface it to customers).

**GDPR data minimisation for KYC.**
- We do NOT receive ID document images. The KYC vendor's iframe sends them directly to the vendor.
- We receive only the verification result (age_passed: true, verification_id: vendor-side opaque token).
- Even the verification_id we hash before storing.
- Lawful basis: Article 6(1)(c) — legal obligation (the customer's OSA duty), and Article 6(1)(f) — legitimate interest (the customer's business need, balanced against the user's rights and freedoms; clearly necessary for the legal-obligation case anyway).
- Article 9 special categories: we do not process biometric data ourselves. The vendor does.
- DPIA: we publish a template DPIA the customer can adopt.

**Data residency.** Turso deployed to LHR (London) primary region for any tenant whose users are UK. Cloudflare Workers run at edge but we set up "data localisation suite"-style policies for any customer whose data we hold. Postmark is US-hosted; we use their EU region for tenants with strict residency.

**ICO registration.** We register as a data controller (£40–£60/yr). Each customer is also a controller of their user data; we are a processor for them. We sign DPAs (data processing agreements) per tenant — template provided in the dashboard.

**Age-assurance principles language.** We never describe ourselves as "age-verifying users." We provide infrastructure that mediates the customer's age-assurance flow. The customer is the responsible service in OSA terms.

---

## 9. Legal posture

**You are a B2B technology vendor, not an adult publisher.** Marketing language must reflect this:
- "Compliance and content protection infrastructure for regulated digital businesses."
- Never use porn-vertical buzzwords on the marketing site or in any material a payment processor can see.
- The dashboard, after login, can be more explicit. The public marketing site stays sober.

**T&Cs must be explicit on:**
- We do not host, serve, mediate, monetise, or curate adult content.
- Customer is solely responsible for their own content and OSA compliance posture.
- Customer warrants they own/control rights to assets uploaded to Reclaim.
- Customer indemnifies us for any DMCA misuse claim arising from their use of the service.
- We have a takedown-policy for our own platform if anyone alleges abuse.
- No NCII content (non-consensual intimate imagery) protection unless customer signs a separate verified-victim flow (see §16).

**Banking and Stripe risk.** Stripe will almost certainly flag and probably terminate. Their adult-vertical policy has been tightening since the 2020 Pornhub event. Realistic options:
- **Paddle as Merchant of Record.** Paddle accepts adult-adjacent B2B SaaS more readily because *they* hold the merchant relationship. Risk: they can also drop you. Acceptable for MVP.
- **Lemon Squeezy** — similar profile, MoR, slightly more permissive but smaller. Backup option.
- **Specialist high-risk processors** (CCBill, Segpay, RocketGate). They are built for adult and won't churn you over the vertical. Higher fees (4.5–7%) and slower payouts. Production option for v1.
- **B2B invoice + bank transfer.** For larger customers (agencies, paysites), straight invoice billing in GBP via a UK Ltd. business account. A challenger bank like Tide or Starling will open the account if you describe the business carefully ("compliance technology / IP-protection automation"); some traditional banks will refuse outright.

Recommendation: launch Reclaim on Paddle. Migrate to a hybrid (Paddle for self-serve creators, invoice for agencies and enterprise) at >£10k MRR.

**Company structure.** Form a UK Ltd. (cheap, fast). Get PI insurance (~£500–800/yr for a small tech vendor) and cyber liability (~£700/yr). Both are pre-launch musts.

---

## 10. Pricing

### Reclaim (Anti-Piracy) — recommended ship-first

| Plan | Monthly | Assets | Crawls/day | Takedowns | Notes |
|---|---|---|---|---|---|
| **Free trial** | £0 | 5 | 1 | 5 (manual review) | 14 days |
| **Creator** | £79 | 50 | 4 | 100 | Solo OF creator |
| **Creator+** | £179 | 200 | 12 | unlimited | Top-tier creator |
| **Agency** | £499 | 1,500 | 24 | unlimited | Up to 30 creators |
| **Agency+** | £1,299 | 5,000 | 24 + custom | unlimited + auto-send | API access, white-label |

Add-on: extra residential-proxy bandwidth at £20 / 5GB. Google deindexing requests at £1.50 each over plan limit.

UK market reference points: Rulta charges £109/mo for 100 takedowns; BranditScan £149/mo Pro; Veriverify £100+. Our positioning is similar pricing with substantially better automation and an explicit roadmap to the Compliance Gateway.

### GateKeep (Compliance Gateway) — second product

Hybrid: pass-through verification cost + platform fee.

| Plan | Monthly platform fee | Per-verification | Verifications included |
|---|---|---|---|
| **Starter** | £149 | £0.95 | 100 |
| **Growth** | £499 | £0.85 | 1,000 |
| **Scale** | £1,499 | £0.70 | 5,000 |
| **Enterprise** | from £4,000 | negotiated | custom |

Pass-through reflects vendor cost (we add ~£0.20 margin per verification). Platform fee covers integration, dashboard, audit-log retention, support, multi-vendor failover, evidence pack.

Mid-tier UK adult site doing ~10k UK uniques/month, re-verifying annually with smart geo-rules turned on, will spend roughly £600–900/mo with us — vs. the cost of an in-house engineer or an enterprise KYC contract starting at £15k/yr. The pitch writes itself.

---

## 11. Tech stack & libraries

**Edge / Worker:**
- Cloudflare Workers (TypeScript), `@cloudflare/workers-types`
- `jose` for JWT signing/verification
- `itty-router` for routing inside Worker
- Cloudflare R2 for temporary upload blob storage
- Cloudflare Workers KV for hot tenant config cache (Turso is source of truth)

**Backend services (Fly.io):**
- Go 1.22, single binary per service (Hasher, Crawler, Notifier)
- `corona10/goimagehash` for pHash
- `u2takey/ffmpeg-go` wrapper, ffmpeg binary in container, for video frame sampling
- `chromedp` for headless Chrome where Playwright not needed; `playwright-go` for hard sites
- `colly` as polite scraping framework
- `tursodatabase/libsql-client-go` for DB
- `mailgun/raymond` (or stdlib text/template) for DMCA email rendering
- `mrz1836/postmark` for sending

**ML / Heavy compute (Hugging Face Space):**
- Python 3.11, FastAPI
- `open_clip_torch` for CLIP embeddings
- ThreatExchange/`tmkpdqf` (Facebook reference impl) for video matching
- Pillow, OpenCV

**Frontend (Cloudflare Pages):**
- Next.js 14 (app router) static export
- Tailwind + shadcn/ui
- React Query
- Auth: Supabase Auth (drop-in for the dashboard's user accounts; tenants are a layer above)
- Charts: Recharts
- Stripe/Paddle JS for billing

**Infra & ops:**
- GitHub Actions for CI/CD (Wrangler deploys for Workers, flyctl for Fly, HF API for Space)
- Sentry free tier for error tracking
- Better Stack / Logflare for log aggregation (free tier OK for MVP)
- Trivy for container scanning (you have a security background, demonstrate it)

---

## 12. Build roadmap (12 weeks solo)

Sequencing legal/payments alongside engineering matters more than the engineering itself. Many adult-tech products died at the payments step.

**Week 0 (pre-build, parallel to setup):** Form UK Ltd. Open Tide/Starling business account. Paddle application submitted. PI + cyber insurance quotes. Solicitor consult (1hr, £200) on T&Cs and DMCA misuse risk.

**Week 1: Foundations.** Repo, monorepo with apps/{worker,frontend,services}/. Turso schema migrated. Auth in dashboard (Supabase). Stripe-or-Paddle integration scaffold.

**Week 2: Reclaim ingestion.** Asset upload UI. Hasher Go service deployed to Fly.io. R2 ingestion + 24h purge job. pHash + frame-sample working end-to-end. Demo: upload a video, see hash row in DB.

**Week 3: Reclaim crawler.** Crawler Go service, two target adapters. Residential proxy integration via env var (start with $20 BrightData credits to test). Match writing.

**Week 4: Reclaim takedowns.** DMCA template, render, Postmark integration with custom SPF/DKIM-set sending domain. Manual-review dashboard (match list, side-by-side preview, send button). DSN webhook parsing.

**Week 5: Reclaim hardening + 8 site adapters.** Onboard 6 more crawl targets. Add basic threshold tuning UI. Implement notice status state machine.

**Week 6: Reclaim launch.** Public landing page, pricing page, T&Cs (solicitor-reviewed), Paddle live, Sentry live. Soft-launch to 5 hand-picked beta creators (free for 2 months in exchange for feedback + permission to use logo).

**Week 7: GateKeep — proxy core.** Worker reverse proxy. JWT mint/verify with `jose`. Cookie set with correct attributes. Tenant config in Turso. /verify hosted page wireframe.

**Week 8: GateKeep — Persona integration.** Persona iframe in /verify. Webhook callback. State CSRF protection. End-to-end: visit fake-customer site, get redirected, verify, return, see content. Geo-rules MVP (UK-only).

**Week 9: GateKeep — onboarding + dashboard.** Customer onboarding wizard (paste CF API token, auto-create Worker route). Tenant admin dashboard: verifications/day, success rate, revoke session, force-rotate key. Audit log export.

**Week 10: GateKeep — Yoti as second vendor + failover.** Add Yoti adapter. Multi-vendor failover. Re-verification cadence config.

**Week 11: Combined platform.** Single login, combined billing in Paddle/Stripe, cross-product navigation. Polish. Performance test the Worker (CPU ms budget).

**Week 12: GateKeep launch + first paid customer.** Outreach push (see §14). Solicitor reviews GateKeep T&Cs and OSA evidence pack. Live.

If anything slips, drop GateKeep v1 features (Yoti, multi-vendor failover) — keep only Persona for launch.

---

## 13. Free-tier risk & scaling triggers

**Cloudflare Workers free tier:** 100k requests/day, 10ms CPU per request. AV cookie check is ~1ms CPU. The proxy itself adds ~5–8ms (Turso lookup is the slow part). At 100k req/day per customer we are still inside CPU budget. **Trigger to leave free tier:** when one customer crosses 80k req/day, move them to Workers Paid ($5/mo + $0.50/M req). Easy.

**Cloudflare R2:** 10GB free, 1M Class A ops, 10M Class B ops free per month. Fingerprint upload temporary storage easily fits in free tier.

**Turso free:** 9GB, 1B reads/mo, 25M writes/mo. We will not approach these.

**Fly.io free:** 3 shared-CPU 256MB VMs. Hasher + Crawler + Notifier = 3 VMs. Tight. **Trigger to leave free tier:** when crawler bandwidth or hasher CPU hits sustained 80%. Likely at 50–100 customers. Then move to Hobby (~$20–40/mo for the three).

**Hugging Face Space free:** 16GB RAM, 2 vCPU, sleeps after inactivity (cold start ~30s). Acceptable for v1 ML-match (asynchronous, customer waits 30s for the first match in a session). **Trigger to leave free:** any customer with auto-send ML matching (latency-sensitive). Move to HF Pro ($9/mo) or self-host on Fly with a GPU once revenue justifies (>£3k MRR).

**Residential proxy bandwidth:** this is the single biggest cost variable and lives outside free tiers entirely. Budget: $50–200/month at MVP scale. Pass through in pricing.

**Postmark:** 100 emails/mo free. Trial only. Move to $15/mo (10k emails) before launch.

**Master scaling trigger:** when MRR hits £2k. At that point all paid-tier moves combined cost ~£100–150/mo. Margin still 90%+.

---

## 14. Go-to-market

### Reclaim (creator / agency)

- **OnlyFans agencies first.** They are the highest-leverage channel. A 50-creator agency is worth 50 individual creators. Cold outreach via LinkedIn (London/Manchester agency owners are findable). Offer 30 days free + 30% recurring referral.
- **Creator subreddits with care.** r/CreatorsAdvice, r/onlyfansadvice. Do NOT spam. Answer help-with-leak posts genuinely, link only if asked. The subreddit mods will ban shillage immediately.
- **Discord servers.** OnlyFans creator-help servers, "OFTV" community, "creator coalition" type communities. Long-term presence, not blast outreach.
- **Twitter/X.** A handful of UK adult-creator influencers run paid newsletters about business-of-OnlyFans. £200–500 per sponsored mention; small audience but extreme conversion.
- **Trade press.** XBIZ, AVN. £1k–3k for sponsored editorial; covers UK and US adult industry. Worth it once for launch announcement.
- **SEO**: long-tail "how to send a DMCA for [site name] OnlyFans leak" content. Slow but compounds.

### GateKeep (sites and studios)

- **UK Adult Producers Trade Association (UKAP).** Direct outreach to membership.
- **Cam-model directories and paysite-network forums.** AVN forum, ADT, GFY (Adult Webmaster). Old-school but where small operators still hang out.
- **Direct outreach to Ofcom-investigation list.** Public list of investigated services. Not tactful to lead with "you're being investigated" — but a quiet, helpful first email offering a 30-day pilot has high conversion.
- **Solicitor partnerships.** Two or three UK media-and-tech-law firms advise adult clients on OSA compliance. Refer-and-be-referred.

### Founder positioning

Speak at a UK security or edge-computing meetup (DevOps Exchange, BSides, OWASP London) about edge-resident compliance. Not about adult specifically — about *technical* problems of edge JWT and Cloudflare Workers and KYC iframes. This builds the credibility for the UK-jobs portfolio piece (§17).

---

## 15. Open questions / risks

- **Stripe/Paddle drops you.** Likelihood: medium. Mitigation: parallel onboarding with Lemon Squeezy and a high-risk processor (CCBill) so a 24-hour pivot is possible. Always export customer billing data nightly.
- **Reputational risk to founder.** Adult-adjacent work is on your CV (or visibly on your GitHub) for hiring panels at conservative employers. Mitigation: brand the company as compliance/IP-protection infrastructure ("Aegis Trust"), not adult. Keep the parent company name off the public-facing creator marketing if needed via a trading-as.
- **DMCA abuse complaints.** A bad actor could use Reclaim to target legitimate critique/journalism. Mitigation: §16 guardrails enforced in code (not just policy). Maintain a do-not-target list of well-known critique/news outlets.
- **False-positive takedowns harming non-pirate sites.** A wrongful notice can attract §512(f) liability and Lumen-listing as a DMCA-abuser. Mitigation: human-in-the-loop default; tight thresholds; quarterly false-positive audit.
- **OSA evolves.** Ofcom is iterating. New HEAA criteria might invalidate our default vendor mix. Mitigation: vendor-agnostic architecture, multi-vendor support in v1, designated person tracking Ofcom guidance updates.
- **CSAM exposure during crawling.** Adult crawl targets are unfortunately also vectors for illegal content. Mitigation: PhotoDNA / Microsoft's free hash list integrated into the crawler — if a crawled image hits a known-CSAM hash, drop it, log, report to NCMEC/IWF. Never store. This is a non-negotiable code-level guardrail.
- **GDPR DPA management overhead.** At 100 customers, signing 100 DPAs is a real time-suck. Mitigation: click-through DPA on signup, mirroring Stripe's approach.
- **Cookie-bypass at scale.** Determined teen-bypass communities exist. We can only do "highly effective" not "perfect." Document the threat model and don't oversell.
- **Banking shutdown.** Tide/Starling can also drop. Have a backup at Revolut Business + a secondary at a specialist like Cashplus.

---

## 16. Ethical guardrails — what we will NOT do

- **No NCII (non-consensual intimate imagery) takedown service.** Or rather: we will not accept arbitrary uploads claiming NCII without a verified-victim consent flow. NCII takedowns have a different legal regime (Revenge Porn Helpline in the UK, UK Online Safety Act §66B-D for sharing offences). We refer NCII victims to StopNCII.org and the Revenge Porn Helpline; we do not compete with those services and we do not allow our platform to be used as a circumvention of them. A v1 verified-victim flow could be added in partnership with the Revenge Porn Helpline if they want it; not on our roadmap unilaterally.
- **No targeting of legal critique, journalism, or commentary.** A do-not-target list of well-known news/critique outlets is hard-coded. Customer cannot override.
- **No DMCA notices on fair-use content** (review, criticism, parody) — best-effort detection in human review, customer attestation in auto-send mode, terminate accounts that abuse this.
- **No CSAM. Any contact with CSAM** during crawling is reported to NCMEC (US) and IWF (UK) and the original is destroyed. We do not store. This is engineering-enforced via PhotoDNA hash matching at ingestion.
- **No targeting based on sexual orientation, identity, race, or political alignment.** If a tenant tries to enumerate target sites that are obviously hate-based, the account is closed.
- **No verification bypass on request.** Even for customers. Even for influencers who "just need to test." Document this in T&Cs.
- **No data sale, no advertising, no analytics resale.** The KYC table has no PII; we couldn't sell it if we wanted to. The audit log is for the tenant only.
- **A clear, public refusal policy.** Linked from the homepage footer.

---

## 17. Career / portfolio framing

For a UK security or edge-computing hiring panel, this project demonstrates:

- **Edge computing competence.** A production-grade Cloudflare Worker with JWT issuance, cookie security, CSP injection, multi-tenant config caching, and key rotation is a compelling artifact. Walk a panel through the request flow and they will be impressed.
- **Security engineering judgement.** Cookie-theft threat modelling, device-binding tokens, JWT key rotation overlap, CSRF on third-party callbacks, PhotoDNA-at-ingestion for CSAM avoidance — all show senior-level judgement, not just feature implementation.
- **Privacy-by-design and GDPR fluency.** Data minimisation in the kyc_sessions schema, no PII anywhere on the platform, DPA flow, Article 28 processor relationship — all documented and shippable.
- **Regulatory engineering.** OSA implementation is a current-event topic the UK fintech/tech industry cares deeply about. Speaking precisely about HEAA criteria, Ofcom guidance, and the technical/legal interface differentiates from generalist full-stack candidates.
- **Automation / SOAR-adjacent thinking.** The Reclaim pipeline (crawl → match → notice → DSN → escalation → deindex) is a SOAR playbook in everything but name. UK financial institutions and security vendors hiring for SOC tooling will recognise this pattern instantly.
- **Distributed systems pragmatism.** Choosing Workers vs. Fly vs. HF Spaces appropriately, and being able to defend the choice with cost, latency, and capability arguments, is exactly what a staff/principal interview probes.
- **Solo end-to-end delivery.** Repo + production system + paying customers + legal posture, executed alone, in 12 weeks. This is the strongest possible signal for a senior IC role.

The version of this you put on a CV is "Architected and shipped a multi-tenant compliance platform for UK regulated digital services, including edge-resident JWT authentication on Cloudflare Workers, automated DMCA workflow with perceptual hashing, and an Ofcom-aligned age-assurance integration pattern." The adult vertical is a footnote — the technology is the headline.

---

*End of plan.*
