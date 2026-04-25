# Security Webhook Router Database

SQLite/libSQL schema, seed data, and sqlc query definitions for the security webhook router.

Run commands from this directory so `sqlc.yaml` paths resolve to the local `migrations/` and `queries/` directories:

```sh
cd 03-database
sqlc generate -f sqlc.yaml
```

Verify migrations and sample seed data with SQLite:

```sh
cd 03-database
sqlite3 /tmp/security_webhook_router.db \
  ".read migrations/000001_init_security_webhook_router.up.sql" \
  ".read seeds/000001_sample_security_webhook_router.sql" \
  "PRAGMA foreign_key_check;" \
  "SELECT COUNT(*) FROM tenant;" \
  "SELECT COUNT(*) FROM delivery_log;"
```

Verify the down migration removes schema objects:

```sh
cd 03-database
sqlite3 /tmp/security_webhook_router_down.db \
  ".read migrations/000001_init_security_webhook_router.up.sql" \
  ".read migrations/000001_init_security_webhook_router.down.sql" \
  "SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','trigger') AND name NOT LIKE 'sqlite_%';"
```
