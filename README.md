# Rapid SaaS Implementation Workspace

This repository is split into six root implementation folders so parallel agents can work without overlapping write scopes.

| Folder | Ownership |
| --- | --- |
| `01-ingress-worker/` | Cloudflare Worker webhook ingress |
| `02-router-api/` | Go queue consumer, admin API, rules, delivery |
| `03-database/` | SQLite/libSQL schema, seeds, sqlc query assets |
| `04-dashboard/` | Next.js dashboard |
| `05-infra-ci/` | Fly.io, Cloudflare, Docker, CI, deployment templates |
| `06-dev-tooling/` | Local validation, root tooling snapshots, developer scripts |

Planning and progress documentation remains in `docs/`.
