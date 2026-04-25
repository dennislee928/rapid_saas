PRAGMA foreign_keys = ON;

CREATE TABLE tenant (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  plan TEXT NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'starter', 'pro')),
  stripe_cust_id TEXT,
  created_at INTEGER NOT NULL
);

CREATE TABLE user (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  clerk_user_id TEXT,
  role TEXT NOT NULL DEFAULT 'owner' CHECK (role IN ('owner', 'admin', 'member')),
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_user_tenant ON user(tenant_id);

CREATE TABLE api_key (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  prefix TEXT NOT NULL,
  hash TEXT NOT NULL,
  scopes TEXT NOT NULL DEFAULT 'read,write',
  last_used_at INTEGER,
  revoked_at INTEGER,
  created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_api_key_prefix ON api_key(prefix);
CREATE INDEX idx_api_key_tenant ON api_key(tenant_id);

CREATE TABLE endpoint (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  source_preset TEXT CHECK (
    source_preset IS NULL OR source_preset IN ('github', 'stripe', 'slack', 'crowdstrike', 'sentinelone', 'wazuh', 'sentry', 'generic')
  ),
  signing_secret TEXT,
  signing_header TEXT,
  signing_algo TEXT CHECK (signing_algo IS NULL OR signing_algo IN ('sha256', 'sha1')),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_endpoint_tenant ON endpoint(tenant_id);
CREATE INDEX idx_endpoint_tenant_enabled ON endpoint(tenant_id, enabled, created_at DESC);

CREATE TABLE destination (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('slack', 'discord', 'http', 'email', 'pagerduty', 'splunk_hec', 'elastic_bulk', 'sentinel_http')),
  name TEXT NOT NULL,
  config_json TEXT NOT NULL CHECK (json_valid(config_json)),
  secret_ref TEXT,
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_destination_tenant ON destination(tenant_id);

CREATE TABLE rule (
  id TEXT PRIMARY KEY,
  endpoint_id TEXT NOT NULL REFERENCES endpoint(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  name TEXT NOT NULL,
  filter_jsonlogic TEXT CHECK (filter_jsonlogic IS NULL OR json_valid(filter_jsonlogic)),
  transform_kind TEXT NOT NULL CHECK (transform_kind IN ('passthrough', 'template', 'preset')),
  transform_body TEXT,
  destination_id TEXT REFERENCES destination(id),
  on_match TEXT NOT NULL DEFAULT 'forward' CHECK (on_match IN ('forward', 'drop', 'continue')),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  created_at INTEGER NOT NULL,
  CHECK (on_match = 'drop' OR destination_id IS NOT NULL)
);
CREATE UNIQUE INDEX idx_rule_endpoint_position ON rule(endpoint_id, position);
CREATE INDEX idx_rule_endpoint_enabled ON rule(endpoint_id, enabled, position);
CREATE INDEX idx_rule_destination ON rule(destination_id);

CREATE TABLE filter_list (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('ip', 'cidr', 'string', 'regex')),
  source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'spamhaus_drop', 'tor_exit')),
  refreshed_at INTEGER
);
CREATE INDEX idx_filter_list_tenant ON filter_list(tenant_id);

CREATE TABLE filter_list_item (
  list_id TEXT NOT NULL REFERENCES filter_list(id) ON DELETE CASCADE,
  value TEXT NOT NULL,
  PRIMARY KEY (list_id, value)
);

CREATE TABLE delivery_log (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  endpoint_id TEXT NOT NULL REFERENCES endpoint(id) ON DELETE CASCADE,
  rule_id TEXT REFERENCES rule(id) ON DELETE SET NULL,
  destination_id TEXT REFERENCES destination(id) ON DELETE SET NULL,
  status TEXT NOT NULL CHECK (status IN ('queued', 'delivered', 'failed', 'dropped', 'dlq')),
  attempt INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
  http_status INTEGER,
  latency_ms INTEGER CHECK (latency_ms IS NULL OR latency_ms >= 0),
  error TEXT,
  request_hash TEXT,
  request_size INTEGER CHECK (request_size IS NULL OR request_size >= 0),
  received_at INTEGER NOT NULL,
  delivered_at INTEGER
);
CREATE INDEX idx_dlog_tenant_time ON delivery_log(tenant_id, received_at DESC);
CREATE INDEX idx_dlog_endpoint_time ON delivery_log(endpoint_id, received_at DESC);
CREATE INDEX idx_dlog_tenant_endpoint_time ON delivery_log(tenant_id, endpoint_id, received_at DESC);
CREATE INDEX idx_dlog_tenant_status_time ON delivery_log(tenant_id, status, received_at DESC);
CREATE INDEX idx_dlog_destination_time ON delivery_log(destination_id, received_at DESC);
CREATE INDEX idx_dlog_request_hash ON delivery_log(tenant_id, request_hash, received_at DESC);
CREATE INDEX idx_dlog_retention_received_at ON delivery_log(received_at);
CREATE INDEX idx_dlog_retention_status_time ON delivery_log(status, received_at);

CREATE TABLE usage_counter (
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  bucket_hour INTEGER NOT NULL,
  ingressed INTEGER NOT NULL DEFAULT 0 CHECK (ingressed >= 0),
  forwarded INTEGER NOT NULL DEFAULT 0 CHECK (forwarded >= 0),
  failed INTEGER NOT NULL DEFAULT 0 CHECK (failed >= 0),
  PRIMARY KEY (tenant_id, bucket_hour)
);
CREATE INDEX idx_usage_counter_bucket_hour ON usage_counter(bucket_hour);

CREATE TABLE dlq (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  endpoint_id TEXT NOT NULL REFERENCES endpoint(id) ON DELETE CASCADE,
  rule_id TEXT REFERENCES rule(id) ON DELETE SET NULL,
  destination_id TEXT REFERENCES destination(id) ON DELETE SET NULL,
  payload_b64 TEXT NOT NULL,
  last_error TEXT,
  attempts INTEGER NOT NULL CHECK (attempts >= 0),
  parked_at INTEGER NOT NULL
);
CREATE INDEX idx_dlq_tenant ON dlq(tenant_id, parked_at DESC);
CREATE INDEX idx_dlq_endpoint ON dlq(endpoint_id, parked_at DESC);
CREATE INDEX idx_dlq_retention_parked_at ON dlq(parked_at);
