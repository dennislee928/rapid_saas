# AI Audio Stem Separation API — Implementation Plan

A pay-as-you-go web service that runs open-source audio source-separation models
(Demucs v4, MDX-Net) against user-uploaded audio and returns isolated stems
(vocals, drums, bass, other) plus optional mastering and noise reduction.

The product is hosted entirely on free tiers at MVP: Cloudflare Pages + Workers
(edge / auth / signed URLs), Cloudflare R2 (object storage), Fly.io (Go
orchestrator, control plane), Hugging Face Spaces CPU Basic (Python inference
worker), Turso (jobs + credit ledger), Stripe Checkout (credit packs).

---

## 1. Product overview & target users

### Who buys this

| Segment | Pain | Frequency of need | Willingness to pay |
|---|---|---|---|
| Indie musicians / bedroom producers | Want acapellas / instrumentals to remix or cover; can't afford iZotope RX or Lalal.ai subscription | 5–30 tracks/month, bursty | $5–$20 ad hoc |
| Podcasters | Need to clean voice from background music in interviews; remove music-bed bleed before re-cutting | 1–4 hours/week | $10–$30/month equivalent |
| YouTube / TikTok video editors | Want to keep BGM but kill commentary, or vice versa, on third-party clips | 2–10 clips/month | $5/clip |
| Music teachers / cover artists | Karaoke versions of tracks for students | Project-based | $5–$15/project |
| Audio transcription preprocessing | Boost vocal isolation before Whisper | Occasional batch | $20–$50/batch |

### Why credits beat subscriptions for this audience

1. **Demand is bursty.** A podcaster might process 20 episodes in a week, then
   nothing for two months. A monthly subscription guilts them into cancelling.
2. **Lalal.ai already trained the market on credits/minutes.** Users understand
   "$5 = 20 minutes" instantly; "$9.99/mo for 90 minutes" needs a calculator.
3. **Lower commitment lowers the trial barrier.** First-time visitors will pay
   $5 to try it. They will not enter a card for a "free trial that auto-bills".
4. **Refunds are bounded.** A failed job refunds a known credit amount; a
   subscription dispute is a chargeback for the whole month.
5. **Credits map cleanly to our cost.** Inference cost scales with audio
   minutes processed; subscription pricing forces us to absorb power-user risk.

### Differentiators vs. Lalal.ai, Moises, Vocalremover.org

- Pay-per-minute, no minimum pack, no expiry.
- Choose your model (htdemucs-ft for quality, htdemucs for speed, MDX-Net for
  vocals-only), not a black box.
- Stems delivered as a zip with original sample-rate WAV + optional FLAC.
- API access from day one — Lalal's API is enterprise-gated.

---

## 2. Core features — MVP vs. v1 vs. v2

### MVP (week 4 ship target)

Single feature, done well: **2-stem separation (vocals / instrumental)** using
`htdemucs` with `--two-stems=vocals`. This is 80% of demand, fastest to run,
smallest output (2 files instead of 4).

- Web UI: drag-drop upload, progress bar, download zip.
- Max 10 minutes of audio per job, max 50 MB upload.
- Formats in: mp3, wav, flac, m4a, ogg. Format out: wav (16-bit, original
  sample rate), zipped.
- Anonymous upload allowed, but credits require email + Stripe.
- Credit packs: $5 / $15 / $40.

### v1 (week 8)

- **4-stem separation** (vocals / drums / bass / other) via `htdemucs`.
- **6-stem** (adds guitar, piano) via `htdemucs_6s` — flagged as experimental.
- **MDX-Net option** for vocals (sometimes cleaner on pop/rap).
- **Noise reduction** preset using `demucs --segment 7 --shifts 1` plus a
  follow-up `rnnoise` pass for podcast voice tracks.
- **API endpoints** with API keys (read from credit ledger).
- **Job history** per user, 7-day retention.

### v2 (post-product/market-fit)

- **Mastering chain** using `matchering` (reference track matching) — this is
  more interesting than basic LUFS normalization.
- **Stem remixing in browser** (Web Audio API) — recombine stems before
  download.
- **Whisper transcription add-on** on the vocal stem (cheap upsell).
- **Batch / folder upload** (5+ files) at a discount.
- **GPU tier** ($X premium for 5x faster turnaround on RunPod or Replicate).

### Explicitly out of scope at MVP

- Real-time / streaming separation.
- Mobile apps.
- DAW plugins (huge support burden).
- Music generation, lyric extraction (Whisper is v2 add-on).

---

## 3. System architecture

```
                       +----------------------------+
                       |  Browser (Next.js on CF    |
                       |  Pages, Web Audio preview) |
                       +-------------+--------------+
                                     |
                  (1) POST /api/jobs (audio metadata, model)
                                     v
                   +-----------------+------------------+
                   |  Cloudflare Worker (auth, rate     |
                   |  limit, R2 signed PUT URL mint,    |
                   |  Stripe webhook receiver)          |
                   +-----------------+------------------+
                                     |
              (2) signed PUT URL     |   (4) enqueue job
                                     v
   +----------+    (3) PUT audio   +-----+        +--------------------+
   | Browser  | -----------------> | R2  |        |  Fly.io (Go)       |
   +----------+                    +-----+        |  Orchestrator:     |
                                       ^          |  - job queue       |
                                       |          |  - credit debit    |
              (6) PUT stems.zip        |          |  - HF dispatch     |
                                       |          |  - retries         |
                                       |          |  - webhook fanout  |
                                       |          +---------+----------+
                                       |                    |
                                       |       (5) POST /infer (presigned R2 GET URL,
                                       |            job_id, model, params)
                                       |                    v
                                       |          +---------+----------+
                                       |          |  HF Space (FastAPI)|
                                       |          |  Demucs / MDX-Net  |
                                       |          |  CPU Basic 16GB    |
                                       |          +---------+----------+
                                       |                    |
                                       +--------------------+
                                                            |
                                       (7) PATCH /jobs/:id status=done, output_keys
                                                            v
                              +-----------------------------+----------+
                              | Turso (jobs, credit_ledger, users)     |
                              +-----------------------------+----------+
                                                            |
                                          (8) Browser polls or Worker pushes
                                              SSE -> signed R2 GET for download
```

### Why each hop exists

- **Worker mints signed PUTs, not Fly.io.** Cloudflare gives 100k free Worker
  requests/day at edge latency; routing every upload through Fly's 256MB VM
  would saturate it. Worker also validates the user's credit balance before
  issuing the signed URL — fail fast, before bytes move.
- **Upload goes browser → R2 directly.** The audio file never touches the Go
  VM, the Worker, or the HF Space until inference. Fly's egress is metered;
  R2 egress to Cloudflare-attached compute is free.
- **Go orchestrator on Fly does dispatch, not inference.** 256 MB is plenty
  for a queue + libsql client + outbound HTTPS. Inference RAM peak for
  htdemucs on a 4-minute song is ~6 GB, which is why it lives on HF.
- **HF Space pulls audio via presigned R2 GET.** Avoids streaming 50 MB
  through the Go VM. The Space writes stems back via presigned R2 PUT.
- **Status updates go from HF Space → Go via signed callback** (HMAC of
  job_id + timestamp). The Worker is path-of-least-resistance for the
  browser to poll, but the source of truth is Turso, written by Go.
- **Stripe webhooks land on the Worker**, which writes a `credits_added`
  ledger row through the Go API (Worker has Turso token but only for read
  paths; writes go through Go for ledger invariants).

---

## 4. Model choice for MVP

### Candidates considered

| Model | Quality (SDR) | License | Peak RAM (4 min stereo) | CPU runtime per min audio (2 vCPU) | Notes |
|---|---|---|---|---|---|
| Demucs v4 `htdemucs` | ~9.0 dB SDR | MIT | ~5–6 GB | ~30–45 s | Default. Hybrid time/spectral. |
| Demucs v4 `htdemucs_ft` | ~9.2 dB SDR | MIT | ~6 GB | ~90–120 s (4x shifts) | Fine-tuned, slower. |
| Demucs v4 `htdemucs_6s` | ~8.5 dB SDR (overall) | MIT | ~7 GB | ~50–70 s | 6-stem variant. |
| MDX-Net (UVR) | ~9.5 dB SDR (vocals) | MIT (model weights vary) | ~3–4 GB | ~25–35 s | Vocals-only is excellent. |
| Spleeter | ~6.5 dB SDR | MIT | ~2 GB | ~10–15 s | Older, audibly worse. |

### Pick for MVP: `htdemucs` (Demucs v4, default checkpoint)

Reasons:

1. **Quality/speed sweet spot.** htdemucs averages ~9.0 dB SDR on MUSDB18
   while running ~30–45 seconds per minute of audio on 2 vCPU — i.e., a
   4-minute song processes in ~2 minutes wall-clock. That keeps a free-tier
   cold-started Space delivering results in under 4 minutes total.
2. **MIT license** on both code and weights — no commercial use restrictions.
3. **RAM headroom on 16 GB.** Peak ~6 GB leaves room for ffmpeg, the FastAPI
   process, and concurrent ZIP packing.
4. **Two-stems mode** (`--two-stems=vocals`) halves output size and hides the
   complexity of 4 stems for the MVP UI.
5. **Same binary later supports 4-stem and 6-stem** in v1 — no model swap.

### Concrete CLI invocation on the HF Space

```bash
demucs \
  --name htdemucs \
  --two-stems=vocals \
  --out /tmp/jobs/$JOB_ID \
  --segment 7 \
  --jobs 1 \
  --device cpu \
  /tmp/jobs/$JOB_ID/input.wav
```

For 4-stem v1: drop `--two-stems`. For "max quality" tier: switch `--name
htdemucs_ft` and `--shifts 2` (charged at 2x credits because runtime ~3x).

`--segment 7` keeps memory bounded by processing 7-second windows; without
it RAM can spike past 8 GB on long tracks. `--jobs 1` because the Space has
2 vCPU and the Python wrapper uses one for I/O — letting Demucs spawn more
workers thrashes.

### Why not MDX-Net at MVP

It's marginally better at vocals isolation but ships as a zoo of community
checkpoints with mixed licensing. We add it in v1 as an optional model and
clearly attribute the checkpoint we ship with.

### Why not Spleeter

Quality gap is audible. Users who already tried Vocalremover.org (Spleeter
under the hood) come to us specifically because Spleeter wasn't good enough.

---

## 5. Cold-start & queue design

### The HF free-tier sleep problem

HF Spaces on CPU Basic sleep after 48 hours of inactivity by default;
practically the wake delay is 30–90 seconds (Docker image pull + Python
import + model load). A user submitting the first job after a quiet night
sees 60 s of "starting model" before any progress.

### Strategy

1. **Don't fight cold starts on free tier — surface them.** The UI shows a
   "Warming up the model (first job after idle takes ~60 s extra)" status
   pulled from Go. We honestly tell the user the cost of free-tier hosting.

2. **Cron warm-ping.** Cloudflare Worker cron trigger every 30 minutes during
   the user's active hours (06:00–02:00 UTC by default; configurable later)
   hits `GET /healthz` on the Space. This keeps the Space hot during peak
   hours without burning the free tier 24/7.
   - Free Worker cron triggers: yes, included.
   - HF Space CPU time: ping is < 50 ms; negligible.

3. **Pre-load model on Space boot.** FastAPI app does
   `pretrained.get_model('htdemucs')` at import time, not at first request.
   This moves the 20-second weight load into the cold start window where the
   user is already waiting on the Docker image, instead of adding another 20 s
   on top.

4. **Queue on Fly.io.** A simple in-memory + Turso-backed queue:
   - On `POST /api/jobs`, Go writes `jobs(status='queued')` and pushes to an
     in-process buffered channel (size 64).
   - A worker goroutine pulls and dispatches to HF (one in flight per Space
     instance — the Space is single-process).
   - On Go restart, the worker re-reads `WHERE status IN ('queued',
     'dispatched') ORDER BY created_at` from Turso and reseeds the channel.

5. **Concurrency limit = 1 job at a time on free Space.** htdemucs peaks at
   ~6 GB RAM; running two in parallel risks OOM. Surface estimated wait in
   the UI ("3 jobs ahead, ~6 minutes").

6. **Retry/timeout policy:**
   - HTTP timeout from Go to Space: 600 s (10 min) — long enough for
     10-minute audio with 2x shifts on slow Space.
   - Retry once on 5xx after 5 s backoff. Don't retry on 4xx.
   - On final failure: refund credits to ledger atomically, mark
     `jobs.status='failed'`, surface error code to user.
   - Crashed/lost jobs (Space dies mid-inference): heartbeat from Space
     every 10 s; Go marks `dispatched` jobs with no heartbeat for 90 s as
     `failed_lost` and refunds.

7. **Hard limits:**
   - Max audio length MVP: **10 minutes** (file rejected at Worker, before
     R2 PUT).
   - Max file size: 50 MB.
   - Per-user concurrency: 1 in-flight job; subsequent submissions queue
     (don't 429).

---

## 6. Data model (Turso / libSQL)

```sql
-- Users (minimal at MVP; expand if Supabase Auth comes in)
CREATE TABLE users (
    id            TEXT PRIMARY KEY,        -- ULID
    email         TEXT UNIQUE NOT NULL,
    stripe_cust   TEXT UNIQUE,
    api_key_hash  TEXT,                    -- SHA-256 of API key (v1)
    created_at    INTEGER NOT NULL,        -- unix seconds
    deleted_at    INTEGER
);

-- Immutable credit ledger. Balance = SUM(delta) over user_id.
-- NEVER UPDATE OR DELETE rows. Refunds are positive entries with a kind.
CREATE TABLE credit_ledger (
    id              TEXT PRIMARY KEY,      -- ULID
    user_id         TEXT NOT NULL REFERENCES users(id),
    delta           INTEGER NOT NULL,      -- in millicredits (1 credit = 1000)
    kind            TEXT NOT NULL,         -- 'purchase' | 'consume' | 'refund' | 'grant' | 'expiry'
    job_id          TEXT,                  -- nullable, for consume/refund
    stripe_event_id TEXT,                  -- idempotency for purchases
    note            TEXT,
    created_at      INTEGER NOT NULL,
    UNIQUE(stripe_event_id)
);
CREATE INDEX idx_ledger_user ON credit_ledger(user_id, created_at);

-- Jobs. Mutable: status transitions queued -> dispatched -> done|failed.
CREATE TABLE jobs (
    id              TEXT PRIMARY KEY,           -- ULID
    user_id         TEXT REFERENCES users(id),  -- nullable for anon (we don't allow anon paid jobs at MVP)
    idempotency_key TEXT,                       -- client-supplied, scoped per-user
    model           TEXT NOT NULL,              -- 'htdemucs' | 'htdemucs_ft' | 'htdemucs_6s' | 'mdxnet_vocal'
    params_json     TEXT NOT NULL,              -- {"two_stems":"vocals","shifts":1,"segment":7}
    input_r2_key    TEXT NOT NULL,              -- e.g. 'in/2026/04/JOB_ID.wav'
    input_bytes     INTEGER NOT NULL,
    input_secs      REAL NOT NULL,              -- after ffprobe
    cost_credits    INTEGER NOT NULL,           -- in millicredits, debited at enqueue
    status          TEXT NOT NULL,              -- queued|dispatched|done|failed|failed_lost|cancelled
    error_code      TEXT,
    output_r2_keys  TEXT,                       -- JSON array of keys
    output_zip_key  TEXT,                       -- packed zip key
    output_ttl_at   INTEGER,                    -- unix seconds; cleanup cron deletes after
    space_url       TEXT,                       -- which Space handled it (multi-Space later)
    started_at      INTEGER,
    finished_at     INTEGER,
    last_heartbeat  INTEGER,
    created_at      INTEGER NOT NULL,
    UNIQUE(user_id, idempotency_key)
);
CREATE INDEX idx_jobs_status ON jobs(status, created_at);
CREATE INDEX idx_jobs_user ON jobs(user_id, created_at DESC);

-- Pricing rules (versioned, so old jobs can be re-priced for refunds correctly)
CREATE TABLE pricing_rules (
    id            TEXT PRIMARY KEY,
    model         TEXT NOT NULL,
    multiplier    REAL NOT NULL,    -- e.g. htdemucs=1.0, htdemucs_ft=2.0, 6s=1.4
    base_per_sec  INTEGER NOT NULL, -- millicredits per audio second
    effective_from INTEGER NOT NULL,
    effective_to   INTEGER          -- nullable = current
);

-- Webhook outbox for reliable Stripe and customer-facing notifications
CREATE TABLE webhook_outbox (
    id           TEXT PRIMARY KEY,
    target_url   TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    next_try_at  INTEGER NOT NULL,
    delivered_at INTEGER,
    created_at   INTEGER NOT NULL
);
```

### Invariants enforced in Go

- A consume row is written **before** dispatching to HF, in the same Turso
  transaction as `jobs INSERT`. If HF dispatch fails immediately, we write a
  refund row.
- `SUM(delta) >= 0` for any user: enforced by reading current balance inside
  the consume transaction and rejecting if insufficient. (libSQL doesn't
  support `CHECK` across rows, so this lives in app code with a serializable
  retry on conflict.)
- Idempotency: `(user_id, idempotency_key)` unique. Re-submitting the same
  key returns the existing job.

---

## 7. Credit economics

### What does CPU inference actually cost us on free tier?

At MVP: **$0 marginal cost per job** while we stay under HF Spaces free tier.
The relevant question is: when do we get kicked off, and what's the upgrade?

HF Spaces free CPU Basic limits (as of late 2025):
- 16 GB RAM, 2 vCPU.
- Sleeps after inactivity.
- No hard CPU-hour cap on free tier, but Spaces with sustained high traffic
  may be throttled / asked to upgrade.

### The pricing math

Target: **$5 / 1000 credits**, where **1 minute of audio at htdemucs = 50
credits** (so $5 ≈ 20 minutes of audio, matching the spec).

Pricing rule: `base_per_sec = 833 millicredits` for htdemucs (i.e., 50
credits per 60 s = 0.833 credits/sec, encoded as 833 of 1/1000 credit).
Multipliers:

| Model | Multiplier | Effective $/min |
|---|---|---|
| htdemucs (2-stem) | 1.0x | $0.25/min |
| htdemucs (4-stem) | 1.0x | $0.25/min |
| htdemucs_ft (max quality) | 2.0x | $0.50/min |
| htdemucs_6s | 1.4x | $0.35/min |
| MDX-Net vocal | 1.0x | $0.25/min |
| Mastering pass (v2) | 0.5x add-on | +$0.125/min |

### Does it work on free tier?

Yes, trivially, while we are zero-cost. Stripe takes 2.9% + $0.30 per
transaction. On a $5 pack:
- Stripe fee: ~$0.45
- Net: ~$4.55
- R2 storage: 10 GB free (we delete inputs after 1 h, outputs after 24 h
  default, 7 days for paying users)
- R2 egress to Cloudflare: free
- Turso: 10k+ free rows
- Worker requests: < 100k/day free
- Fly.io: 256 MB VM free
- HF Space: free until throttled

So at sub-1000 paying users/month we're net-positive on every $5 pack.

### When does free tier break, and what's the upgrade?

**Trigger 1: HF Space throttled / kicked.** This happens when sustained
traffic suggests commercial use. Upgrade path: HF Spaces CPU Upgrade ($0.03/h,
~$22/month always-on, 8 vCPU). At that point we can run 2 concurrent jobs.

**Trigger 2: 16 GB RAM ceiling on big jobs.** A 30-minute song with shifts=2
at 6-stem can OOM. Upgrade: same $22/month tier has 32 GB.

**Trigger 3: CPU is too slow.** Power users complain about 8-minute waits
for 4-minute songs. Move to **Replicate** (Demucs is on Replicate at ~$0.0023/sec
of GPU time, ~$0.10–$0.15 for a 4-minute song). At our $1.00 revenue per
4-min job, we still net ~$0.85. Or **RunPod Serverless** with our own
container, ~$0.0004/sec of A4000 — 4-minute song processes in ~30 s for
~$0.012, giving us 98% margin but with cold-start spikes.

**Break-even back-of-envelope:** if free tier dies and we move to Replicate
GPU at $0.10/job avg, then $5 pack with 4 jobs of 4 min each = $4.55 net –
$0.40 inference – ~$0.05 misc = **~$4.10 margin**. Still works; pricing
doesn't need to change.

---

## 8. File handling

### Upload

- Browser does a **multipart upload directly to R2** using a Worker-minted
  presigned URL. For files < 50 MB at MVP we use a single PUT (R2 supports
  up to 5 GB single PUT). Multipart-chunked uploads come in v1 when we lift
  the cap to 200 MB.
- Format probe happens **on the Space**, not the browser, because clients
  lie. First step of the inference pipeline:
  ```bash
  ffprobe -v error -show_format -show_streams -of json input.{ext}
  ```
  Reject if duration > advertised, sample rate < 8 kHz, channels > 2, or
  codec is not in our allow-list.

### Format conversion (ffmpeg)

The Demucs pipeline needs WAV. ffmpeg pipeline in the Space:

```bash
# Normalize to 44.1 kHz stereo 16-bit PCM, mono -> stereo upmix if needed
ffmpeg -y -i input.{ext} \
  -ac 2 -ar 44100 -sample_fmt s16 \
  -af "aresample=resampler=soxr" \
  /tmp/jobs/$JOB_ID/input.wav
```

For very-long inputs (v1, > 10 min), we'll consider splitting at silences
with `silencedetect`, processing chunks, and rejoining — but only if users
ask for it. Demucs handles long inputs natively with `--segment`.

### Output packaging

Demucs writes to `out/htdemucs/<input_stem>/{vocals,no_vocals}.wav`. We:

1. Optionally transcode to FLAC for downloaders who want lossless smaller
   files: `ffmpeg -i vocals.wav -compression_level 5 vocals.flac` (about
   60% of WAV size).
2. Pack: `cd out && zip -0 stems.zip vocals.wav no_vocals.wav` (zip
   `-0` because WAVs are already incompressible; saves CPU).
3. Upload zip to R2 with key `out/<yyyy>/<mm>/<job_id>/stems.zip`.

### Signed download URLs

Worker mints presigned R2 GET with **TTL = 1 hour** by default (24 h on
explicit "extend link" click for paying users). Each download is logged for
abuse monitoring, but we don't watermark MVP outputs.

### Retention

- Inputs: deleted after **1 hour** from successful inference (kept longer
  only if job is still in-flight).
- Outputs: deleted after **24 hours** for free-tier or first-time users;
  **7 days** for users with active credits.
- A nightly Worker cron walks `jobs WHERE output_ttl_at < now()` and issues
  R2 deletes, then nulls out keys.

---

## 9. Payments

### Stripe Checkout (not Paddle)

Stripe is faster to integrate and the audience is mostly US/UK/EU
hobbyists. Paddle's Merchant-of-Record is overkill for $5 packs and adds
~5% friction to margins.

### Pack SKUs

| Pack | Price | Credits | Effective $/min on htdemucs |
|---|---|---|---|
| Starter | $5 | 1000 | $0.25/min |
| Creator | $15 | 3500 | $0.214/min (12.5% bonus) |
| Pro | $40 | 10000 | $0.20/min (20% bonus) |

Each is a one-time `Price` in Stripe with `metadata.credits` set; the
webhook handler reads metadata, not price ID, so we can rotate prices
without touching code.

### Webhook flow

1. Stripe webhook lands at Worker: `POST /api/stripe/webhook`.
2. Worker verifies signature, then forwards to Go: `POST /internal/stripe/event`
   with HMAC.
3. Go handles `checkout.session.completed`:
   - Idempotency check: `SELECT ... FROM credit_ledger WHERE
     stripe_event_id = ?`. If found, no-op return 200.
   - Begin transaction:
     - INSERT into `credit_ledger` (delta = +pack_credits, kind='purchase',
       stripe_event_id = event.id).
     - UPSERT user by email if not exists.
   - Commit.
4. Send confirmation email (Resend free tier).

### Refund / dispute flow

- **Job failure refund**: automatic, at the moment Go marks
  `jobs.status='failed'`. Writes a `kind='refund'` row equal to the
  `cost_credits`. User sees credits restored within seconds.
- **User-requested refund** (within 7 days, unused credits): manual at MVP
  via Stripe dashboard; a Stripe `charge.refunded` webhook triggers Go to
  insert a negative `kind='refund'` ledger row equal to credits remaining
  (not used). Used credits are non-refundable per ToS.
- **Dispute / chargeback**: Stripe `charge.dispute.created` webhook flips
  the user's `users.deleted_at` (soft suspend) and queues an admin alert.
  Don't auto-decide; review and respond via Stripe portal.

---

## 10. Auth & abuse

### MVP auth: magic links via Resend

- `POST /api/auth/email` — Worker generates a one-time token (32 bytes,
  base64url), stores hash in Turso `auth_tokens` with 15-minute expiry,
  sends email via Resend (free tier: 100 emails/day, 3000/month).
- Click link → Worker validates token → sets `session` cookie (signed JWT,
  HS256, 30-day TTL).
- No password storage, no OAuth dance for MVP.

### v1 auth

Add Google OAuth via Supabase Auth if we hit Resend's daily ceiling or
users complain about magic-link friction. Supabase Auth free tier is
50k MAU.

### Abuse vectors

| Vector | Defense |
|---|---|
| Free-credit farming via burner emails | No free credits at MVP. Optional 1-time 60-second free trial sample per IP (locked to a fixed sample track we provide) |
| Massive uploads to clog the Space | Reject > 50 MB / > 10 min at Worker before R2 PUT |
| Replay of signed download URLs | TTL = 1 hour, R2 returns 403 after |
| API abuse via leaked API key (v1) | Hash storage + per-key rate limit (Worker durable object), revocation in dashboard |
| Card testing on Stripe | Stripe Radar default rules; minimum $5 charge |
| DDoS upload spam | Cloudflare WAF + Turnstile on the upload form |

### Copyright / DMCA hygiene

- Terms of Service explicitly say users must own rights or have
  authorization for uploaded content. We don't verify, but logging
  upload IPs + email + Stripe identity gives us repudiation.
- **Output retention is short on purpose**: 24 hours default. We are not a
  music distribution platform; we are a one-shot processor. Short
  retention reduces our exposure to "you hosted my track" complaints.
- Provide a public DMCA agent email (`dmca@<domain>`) and a takedown form.
  Auto-purge inputs and outputs for any matching `job_id` within 1 hour.
- We do not log audio fingerprints (no Audible Magic / etc.) at MVP — we'd
  need a license. We rely on retention + ToS instead.
- For paid users we keep a hash of input filename and email, but not the
  audio.

---

## 11. Tech stack — concrete

### Frontend (Cloudflare Pages)

- **Next.js 14 App Router**, deployed via `@cloudflare/next-on-pages`.
- **Tailwind + shadcn/ui** for speed.
- **Upload**: `@cloudflare/workers-types` for direct Worker calls;
  `react-dropzone` + native `fetch` PUT for the upload itself.
- **State**: TanStack Query for job polling.
- **Auth**: cookie-based session, no client-side auth library.

### Edge (Cloudflare Workers)

- TypeScript, **Hono** router (small, edge-friendly).
- Endpoints:
  - `POST /api/upload-url` — mints R2 PUT URL after auth + balance check
  - `POST /api/jobs` — forwards job spec to Go
  - `GET /api/jobs/:id` — reads Turso (read replica) directly
  - `GET /api/download/:job_id/:asset` — issues R2 GET signed URL
  - `POST /api/stripe/webhook` — Stripe receiver, forwards to Go
  - `POST /api/auth/email`, `GET /api/auth/verify`
- **Cron**: `*/30 * * * *` — warm-ping HF Space.
- **Turnstile**: on signup + paid checkout.

### Backend orchestrator (Fly.io, Go)

- **Go 1.22**, single binary, deployed via `flyctl` (Dockerfile from
  `golang:alpine` minimal).
- Libraries:
  - HTTP: `chi` router (light) or stdlib `net/http`.
  - Turso: `github.com/tursodatabase/libsql-client-go`.
  - Validation: `go-playground/validator`.
  - ULID: `oklog/ulid/v2`.
  - HMAC: stdlib.
  - Stripe: `stripe/stripe-go/v76`.
- Process model: single binary, two goroutine pools:
  - HTTP server (chi) handling Worker-forwarded requests.
  - Worker pool (1 worker at MVP because Space is single-flight) draining
    a buffered channel reseeded from Turso on boot.
- Health endpoint `/healthz` returns build SHA + queue depth.
- Logs: stdout JSON (`zerolog`), Fly tails them.

### Inference (Hugging Face Space)

- Public Space, Docker SDK (custom Dockerfile, not Gradio default).
- Base image: `python:3.11-slim` + system `ffmpeg`.
- App: **FastAPI**, single endpoint `POST /infer` taking a small JSON:
  ```json
  {
    "job_id": "01HX...",
    "input_url": "https://r2-presigned.../input.wav",
    "output_put_url_template": "https://r2-presigned.../{stem}.wav",
    "model": "htdemucs",
    "params": {"two_stems": "vocals", "segment": 7, "shifts": 1},
    "callback_url": "https://orch.fly.dev/internal/jobs/01HX.../complete",
    "hmac_sig": "..."
  }
  ```
- App layout:
  ```
  /app
    main.py        # FastAPI app, single /infer + /healthz
    pipeline.py    # download -> ffprobe -> ffmpeg -> demucs -> zip -> upload
    callbacks.py   # heartbeat + completion POST with HMAC
    Dockerfile
    requirements.txt  # demucs, fastapi, uvicorn, httpx, soundfile
  ```
- Demucs invoked as a subprocess (not via Python API) because the CLI is
  the most-tested code path and we want easy `--segment`/`--shifts`/`--name`
  knobs without re-implementing argument parsing.
- Heartbeat: a background `asyncio.create_task` that POSTs `/heartbeat`
  every 10 seconds while the subprocess runs.

### Database (Turso)

- One primary in `iad`, replicas in `lhr` (London) and `nrt` (Tokyo) for
  Worker reads. Writes always go through Go.
- Migrations: `goose` (Go-native), checked into the orchestrator repo.
- `litestream`-style backups not needed; Turso handles point-in-time.

### Storage (Cloudflare R2)

- Buckets: `audio-in` (1 h lifecycle), `audio-out` (24 h / 7 d lifecycle).
- Lifecycle rules expressed in R2 config, not application code.
- CORS allows direct PUT from `https://<our-domain>` only.

### Email

- **Resend** for transactional (magic links, receipts, job-done
  notifications).

### Monitoring

- **Better Stack** free tier (Logtail) for log aggregation.
- **Sentry** free tier for errors (browser + Go).
- **Healthchecks.io** free tier pings the warm-cron and the Fly health
  endpoint every 5 min.

---

## 12. Build roadmap (solo, 4–6 weeks part-time)

Assumes ~15 hours/week.

### Week 1 — foundations & inference proof

- Repo skeleton: `web/` (Next.js), `worker/` (CF Worker), `orch/` (Go),
  `space/` (FastAPI/Demucs), `infra/` (Terraform-lite or just docs).
- Stand up HF Space with a hard-coded local input → demucs → output
  written to `/tmp`, reachable on a public URL.
- Verify htdemucs runs on CPU Basic with a 4-minute song. Time it.
  Record peak RAM with `/usr/bin/time -v` so we know our headroom.
- Deliverable: curl `POST /infer` with a public-mp3 URL, get back a zip
  URL hosted on HF tmpfiles. Quality check on three reference tracks
  (rap, rock, EDM).

### Week 2 — storage & orchestrator

- Cloudflare R2 buckets, lifecycle rules, CORS.
- Worker: `/api/upload-url` + `/api/download/:id` endpoints; basic
  Hono skeleton; KV for short-lived idempotency tokens.
- Go orchestrator on Fly.io: Turso schema, `POST /api/jobs`,
  `GET /api/jobs/:id`, dispatcher goroutine, HMAC client to Space.
- Connect end-to-end with **anonymous, free** jobs (no payments yet).
  Manual smoke test: upload mp3 in browser → see vocals.zip download.

### Week 3 — credits & payments

- Stripe products & test-mode webhook.
- `credit_ledger` table + balance read path.
- Charge before dispatch; refund on failure.
- Magic-link auth via Resend.
- Pricing rules in DB; implement `cost_credits` calc from
  `input_secs * pricing.base_per_sec * model.multiplier`.

### Week 4 — UI polish & MVP launch

- Next.js front page: drag-drop, progress, queue position, download.
- Account page: balance, history, "buy more credits".
- Pricing page (clear, no dark patterns).
- ToS, Privacy, DMCA agent listing.
- Cron warm-ping live.
- Sentry hooked up. Production Stripe keys.
- **Soft launch on r/podcasting + Twitter/X. Get first 20 users.**

### Week 5 — v1 features

- 4-stem mode (`htdemucs` no `--two-stems`).
- 6-stem mode (`htdemucs_6s`) behind a "experimental" tag.
- MDX-Net vocal model added; user picks model in UI.
- Job retention extended to 7 days for paying users.

### Week 6 — API & ProductHunt

- API keys for paying users.
- Public OpenAPI spec at `/docs`.
- Postman collection.
- ProductHunt launch with a demo track and side-by-side audio comparison
  vs Lalal.ai (using their free 1-minute trial).

### Weeks 7–8 — buffer / mastering / scale prep

- Watch for HF throttling. If hit, port the Space Dockerfile to RunPod
  Serverless behind a feature flag.
- Add `matchering`-based mastering (v2 feature) if user requests
  justify it.
- Whisper transcription add-on on the vocal stem (small, cheap).

---

## 13. Free-tier risk & scaling triggers

### Risks ranked

1. **HF Spaces sleep causes UX failure.** First-job-after-idle = 60 s
   spinner. **Mitigation**: warm-cron during peak hours, honest UI
   messaging. **Trigger to upgrade**: > 5% of jobs experience cold-start
   delays per our metric, or > 20 paid users/day.

2. **HF throttles for "commercial use".** Their ToS allows free Spaces
   but reserves the right to throttle high-traffic ones. **Trigger**:
   first throttling notice, or sustained > 100 jobs/day. **Action**: in
   24 hours, redeploy the Space's Dockerfile to **RunPod Serverless**
   (CPU pod first, ~$0.0001/s; switch to A4000 GPU later for ~10x
   speedup at ~4x cost). Keep HF as a fallback in Go's dispatcher.

3. **Single-Space single-flight is the bottleneck.** **Trigger**: avg
   queue depth > 3 for 24 h. **Action**: spin up second HF Space (same
   container, different URL), Go round-robins.

4. **Fly.io 256 MB ceiling.** Almost certainly fine for control plane,
   but a buffered channel of 64 + libsql + chi can creep. **Trigger**:
   memory at > 200 MB sustained. **Action**: bump Fly to
   `shared-cpu-1x@512MB` for $1.94/mo.

5. **Turso row count.** Each job writes ~3 ledger rows + 1 job row + N
   webhook rows. At 1000 jobs/day = ~5000 rows/day = ~150k/month. Free
   tier is 9 GB storage and ~10k row reads/day on the cheap tier — read
   billions on standard. **Trigger**: storage > 80%. **Action**: archive
   completed jobs older than 90 days to R2 as JSONL.

6. **Worker request count.** 100k/day free. At our scale, every job
   triggers ~10 Worker calls (upload-url, jobs POST, polls, download).
   1000 jobs/day = 10k calls/day; we are fine until 10k jobs/day.

7. **R2 storage / egress.** Inputs auto-delete after 1 h; outputs after
   24 h. At 1000 jobs/day x avg 10 MB output = 10 GB resident at any
   moment, well within the 10 GB free tier. R2 egress to CF-attached
   compute is free; user downloads count against the 1M-Class-A-ops
   tier (also free for our usage).

8. **Stripe Radar false positives.** Hobbyist musician tries to pay
   from a sketchy IP, gets blocked. Acceptable cost.

### Move-off-free-tier checklist (when revenue justifies)

- HF Space → RunPod Serverless GPU (A4000, on-demand): ~10x speed at ~$0.012/job.
- Fly.io → keep, just increase RAM.
- Turso → keep, paid tier is cheap.
- R2 → keep, $0.015/GB after free tier.
- Resend → upgrade to $20/mo for 50k/mo emails when we scale auth.
- **Stop being free-tier-first**: total bill at ~10k jobs/month is ~$80,
  which is ~$2.5k MRR break-even — easy at the unit economics above.

---

## 14. Go-to-market

### Wedge against incumbents

| Competitor | Their weakness | Our angle |
|---|---|---|
| Lalal.ai | Subscription + per-pack pricing; closed model; no API for prosumers | Pure pay-as-you-go, no expiry, no tier locks; API on day one of v1 |
| Moises.ai | App-centric, $4–10/mo subscriptions, mobile-first | Web + API for builders; cheaper for occasional use |
| Vocalremover.org | Free but Spleeter quality (audibly worse) | Demucs v4 quality at the same low price-per-minute |
| UVR (desktop) | Local install, technical setup, no batch sharing | Web-based, share download links, runs on any device |
| iZotope RX (pro) | $399 license, overkill for a vocal removal | $5 buys exactly what 90% need |

### Launch surfaces (in order)

1. **r/podcasting** — post a free 60-second sample tool branded as
   "free podcast voice cleaner" (loss-leader). Convert via "want to
   do longer? $5 = 20 min."
2. **r/WeAreTheMusicMakers** + **r/edmproduction** — post side-by-side
   audio comparisons (ours vs. Spleeter vs. Lalal trial). Music
   producers care about quality demos, not features lists.
3. **r/podcasting** + **r/NewTubers** — different angle: "remove
   background music from interviews". This is a real recurring pain.
4. **ProductHunt** — week 6 launch. Bring a real demo, not a landing
   page. Pre-launch teaser to email list.
5. **Hacker News** — Show HN around v1 (4-stem + API). Audience cares
   about the open-source stack story (Demucs + Go + free-tier
   architecture); lead with the architecture, not the feature.
6. **YouTube tutorials** — make 3 short tutorials (cover acapella, EDM
   remix bass extraction, podcast voice cleanup). SEO long-tail.
7. **API/dev outreach (v1+)** — n8n integrations, Zapier, Bubble.io
   plugins. Builders bring traffic without competing.

### Honest first-30-days target

- 200 free 60-second trial uses
- 30 paying users (15% conversion of triers)
- Avg pack: $5 → $150 first-month revenue
- Realistic: not life-changing, but enough signal to invest week 7+.

---

## 15. Open questions & risks

### Legal

- **Q: Do we need to display a copyright disclaimer at every upload?**
  Probably yes; mirror Lalal.ai/Moises ToS wording on the upload page
  itself, not just buried in legal pages. Consult a lawyer before
  paid launch if budget allows.
- **Q: GDPR — do we store EU user emails? When?** Yes (paying users),
  so DPA + privacy policy needed. SES/Resend are fine processors.
- **Q: Music industry takedown campaigns.** Unlikely to target us at
  small scale, but Lalal has been around since 2019 unscathed; stem
  separation is widely accepted. Big risk: an artist Twitter-storm.
  Have a public takedown form and respond in 24 h.

### Quality

- **Q: How do we handle "stem quality is bad on this track"
  complaints?** First line: re-run with `htdemucs_ft` and `--shifts 2`
  for free as a goodwill credit. Second line: refund. Don't promise
  perfection in marketing copy.
- **Q: Should we do automated quality scoring?** Could compute SDR
  vs. the input mix as a sanity check, but the values are not
  meaningful to users. Skip at MVP.

### Operational

- **Q: One HF Space or many?** One at MVP for simplicity. Multi-Space
  load balancing waits for queue-depth alarm.
- **Q: How do we test the Space's Demucs quality before promoting a
  new model?** Maintain a 5-track regression set in the repo; on every
  Space deploy, run inference, hash outputs, diff against pinned
  golden hashes. Tolerate small float diffs but flag big regressions.
- **Q: GPU upgrade path** — Replicate is easiest (managed, pay-per-sec),
  but they charge per-second of cold-started container. RunPod
  Serverless with our own image gives better economics but requires
  cold-start tolerance. Plan: have a working RunPod image ready by
  week 8 even if we don't deploy it; switching is a config flag in Go.

### Strategic

- **Q: What if Demucs v5 lands and is twice as good?** Drop-in upgrade;
  same CLI; we benefit immediately. This is the value of building on
  open-source.
- **Q: What if HF deprecates free tier?** 30-day migration; RunPod /
  Replicate fallback; no architectural change required because Go
  already abstracts the inference target via a `SpaceClient` interface.
- **Q: Subscription hybrid?** Probably yes at v2: a $9/month plan for
  power users (e.g., 60 minutes included + 20% off overage). But not
  before there's evidence anyone wants it. Credits-only at MVP and v1.

---

## Appendix A — Cost summary at three traffic levels

| Metric | 100 jobs/mo | 1000 jobs/mo | 10000 jobs/mo |
|---|---|---|---|
| Avg job length | 4 min | 4 min | 4 min |
| Audio-min/mo | 400 | 4000 | 40000 |
| Revenue (avg $0.25/min) | $100 | $1000 | $10000 |
| Stripe fees (~6%) | $6 | $60 | $600 |
| HF Space | $0 | $0 (or upgrade $22) | upgrade or move ($150–$300) |
| Fly.io | $0 | $0 | $5 |
| Turso | $0 | $0 | $0–$29 |
| R2 | $0 | $0 | ~$5 |
| Resend | $0 | $0 | $20 |
| **Net** | **~$94** | **~$940** (–$22 if upgraded) | **~$8800** |

All numbers approximate. The architecture intentionally has no scaling
cliff: every component has a clear $/scale upgrade path, and inference
(the only non-trivial cost) can be moved between providers in hours
without app changes.
