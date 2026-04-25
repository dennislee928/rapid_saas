# Aegis Adult Compliance and Anti-Piracy MVP

This directory is an MVP scaffold for the GateKeep compliance gateway and Reclaim anti-piracy workflow described in `docs/05-adult-compliance-anti-piracy.md`.

Legal and regulatory notes in this scaffold are operational placeholders for product design and customer workflow planning. They are not legal advice, compliance certification, or a representation that a tenant satisfies UK Online Safety Act, DMCA, CDPA, GDPR, Ofcom, or payment processor requirements.

## Components

- `worker/`: Cloudflare Worker skeleton for reverse-proxy age-gating, hosted verify/callback stubs, upload/hash workflow stubs, match review, and human-reviewed takedown generation.
- `services/hasher-crawler/`: Go service skeleton for perceptual-ish hashing placeholders, crawl adapters, matching thresholds, and offline tests.
- `db/migrations/`: SQLite/Turso schema for tenants, GateKeep sessions, Reclaim assets, matches, takedown notices, and audit log.
- `dashboard/`: Static dashboard stub showing the intended product surface without external services.
- `templates/`: DMCA notice template with human-review language and operational placeholders.

## Local Verification

```sh
cd 10-adult-compliance-antipiracy/worker
npm test
npm run typecheck

cd ../services/hasher-crawler
go test ./...
go run ./cmd/aegis-worker -mode once
```

## Runtime Boundaries

- GateKeep stores no identity documents and the JWT claims intentionally omit name, DOB, email, and document data.
- Reclaim defaults to human-in-the-loop takedown review. Auto-send is intentionally not implemented in the MVP scaffold.
- Upload handlers are stubs: the Worker computes a SHA-256 content hash and returns the next workflow state, but does not persist content.
- Crawler targets are placeholders and must be curated by a human before production use.
- KYC provider callbacks are stubbed with HMAC verification and a synthetic success payload for local testing.

## Environment

Copy `worker/.env.example` for local Worker development and configure Cloudflare secrets for production.

Required Worker variables:

- `AEGIS_ENV`: `local`, `staging`, or `production`.
- `PLATFORM_HMAC_SECRET`: signing secret for local callback simulation and state tokens.
- `DEFAULT_TENANT_SECRET`: development-only tenant JWT secret.
- `POSTMARK_API_TOKEN`: optional; omitted in MVP tests because takedowns are rendered for review only.

