# AI Audio Stem Separation MVP

Practical monorepo slice for the MVP described in `docs/02-ai-audio-stem-separation.md`.

This scaffold is intentionally safe by default:

- No external services are required for tests.
- Upload, download, Stripe, R2, Turso, and Hugging Face calls are stubbed behind local interfaces.
- The inference worker defaults to `SAFE_STUB_MODE=true`, so no Demucs model is downloaded and no audio command is executed unless explicitly enabled.

## Layout

- `worker/` - Cloudflare Worker-style API for upload URL minting, job forwarding, download URL stubs, Stripe webhook stubs, and warm cron.
- `orchestrator/` - Go control plane for job creation, credit debits/refunds, status transitions, and Hugging Face dispatch skeleton.
- `space/` - FastAPI-compatible inference service skeleton for ffprobe/ffmpeg/Demucs orchestration with safe stub mode.
- `migrations/` - SQLite/Turso-compatible schema migrations.
- `dashboard/` - Minimal static upload/status UI that can run from any static server.

## Local Verification

```bash
cd worker && npm test
cd orchestrator && go test ./...
cd space && python3 -m unittest discover -s app -p '*_test.py'
```

## Environment

Copy `.env.example` into each service as needed. MVP tests do not require these values.

Production integrations expected later:

- Cloudflare R2 signed URLs and lifecycle policies
- Turso/libSQL client for durable jobs and ledger writes
- Stripe webhook signature verification
- Resend magic links
- Hugging Face Space or RunPod inference endpoint

