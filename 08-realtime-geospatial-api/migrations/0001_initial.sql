CREATE TABLE IF NOT EXISTS tenants (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  plan TEXT NOT NULL DEFAULT 'free',
  ccu_cap INTEGER NOT NULL DEFAULT 200,
  write_rps_cap INTEGER NOT NULL DEFAULT 100,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id),
  prefix TEXT NOT NULL,
  hash TEXT NOT NULL,
  scopes TEXT NOT NULL,
  last_used_at INTEGER,
  revoked_at INTEGER,
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);

CREATE TABLE IF NOT EXISTS groups (
  id TEXT NOT NULL,
  tenant_id TEXT NOT NULL REFERENCES tenants(id),
  retention_ttl INTEGER NOT NULL DEFAULT 60,
  history_enabled INTEGER NOT NULL DEFAULT 0,
  history_days INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS history_chunks (
  tenant_id TEXT NOT NULL,
  group_id TEXT NOT NULL,
  day TEXT NOT NULL,
  r2_key TEXT NOT NULL,
  bytes INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, group_id, day)
);

CREATE TABLE IF NOT EXISTS usage_buckets (
  tenant_id TEXT NOT NULL,
  bucket_minute INTEGER NOT NULL,
  peak_ccu INTEGER NOT NULL,
  publish_count INTEGER NOT NULL,
  egress_bytes INTEGER NOT NULL,
  PRIMARY KEY (tenant_id, bucket_minute)
);
