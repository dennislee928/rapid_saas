PRAGMA foreign_keys = ON;

CREATE TABLE tenants (
  id TEXT PRIMARY KEY,
  slug TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  plan TEXT NOT NULL CHECK (plan IN ('reclaim_starter','reclaim_pro','gatekeep_starter','gatekeep_pro','combined')),
  created_at INTEGER NOT NULL,
  jwt_secret_enc BLOB NOT NULL,
  jwt_kid TEXT NOT NULL,
  jwt_secret_prev BLOB,
  jwt_kid_prev TEXT,
  reverify_days INTEGER NOT NULL DEFAULT 365,
  geo_rules_json TEXT
);

CREATE TABLE protected_assets (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('image','video')),
  label TEXT,
  phash BLOB,
  video_meta_json TEXT,
  uploaded_at INTEGER NOT NULL,
  source_deleted INTEGER NOT NULL DEFAULT 0 CHECK (source_deleted IN (0, 1))
);

CREATE INDEX idx_assets_tenant ON protected_assets(tenant_id);

CREATE TABLE monitored_sites (
  id TEXT PRIMARY KEY,
  hostname TEXT NOT NULL UNIQUE,
  category TEXT,
  scrape_strategy TEXT NOT NULL CHECK (scrape_strategy IN ('http_fetch','playwright_stealth','flaresolverr','manual')),
  abuse_email TEXT,
  rate_limit_rps REAL NOT NULL DEFAULT 1.0,
  last_crawled_at INTEGER,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
);

CREATE TABLE matches (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  asset_id TEXT NOT NULL REFERENCES protected_assets(id) ON DELETE CASCADE,
  site_id TEXT NOT NULL REFERENCES monitored_sites(id),
  candidate_url TEXT NOT NULL,
  match_score REAL NOT NULL CHECK (match_score >= 0 AND match_score <= 1),
  hamming INTEGER,
  candidate_phash BLOB,
  status TEXT NOT NULL CHECK (status IN ('new','reviewing','approved','rejected','sent')),
  detected_at INTEGER NOT NULL,
  reviewed_at INTEGER,
  reviewed_by TEXT
);

CREATE INDEX idx_matches_tenant_status ON matches(tenant_id, status);

CREATE TABLE takedown_notices (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  rendered_body TEXT NOT NULL,
  to_address TEXT NOT NULL,
  from_address TEXT NOT NULL,
  postmark_msg_id TEXT,
  status TEXT NOT NULL CHECK (status IN ('queued','sent','bounced','acknowledged','removed','declined','escalated')),
  sent_at INTEGER,
  responded_at INTEGER,
  removed_at INTEGER,
  google_request_id TEXT
);

CREATE TABLE kyc_sessions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  vendor TEXT NOT NULL,
  vid_hash TEXT NOT NULL,
  age_passed INTEGER NOT NULL CHECK (age_passed IN (0, 1)),
  geo_country TEXT,
  device_hash TEXT,
  verified_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  revoked INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0, 1))
);

CREATE INDEX idx_kyc_tenant_expires ON kyc_sessions(tenant_id, expires_at);

CREATE TABLE audit_log (
  id TEXT PRIMARY KEY,
  tenant_id TEXT REFERENCES tenants(id) ON DELETE SET NULL,
  event TEXT NOT NULL,
  actor TEXT,
  meta_json TEXT,
  occurred_at INTEGER NOT NULL
);

CREATE INDEX idx_audit_tenant_time ON audit_log(tenant_id, occurred_at);

