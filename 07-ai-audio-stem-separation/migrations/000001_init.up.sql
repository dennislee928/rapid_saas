CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    stripe_cust TEXT UNIQUE,
    api_key_hash TEXT,
    created_at INTEGER NOT NULL,
    deleted_at INTEGER
);

CREATE TABLE credit_ledger (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    delta INTEGER NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('purchase', 'consume', 'refund', 'grant', 'expiry')),
    job_id TEXT,
    stripe_event_id TEXT,
    note TEXT,
    created_at INTEGER NOT NULL,
    UNIQUE(stripe_event_id)
);

CREATE INDEX idx_ledger_user ON credit_ledger(user_id, created_at);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    idempotency_key TEXT,
    model TEXT NOT NULL,
    params_json TEXT NOT NULL,
    input_r2_key TEXT NOT NULL,
    input_bytes INTEGER NOT NULL,
    input_secs REAL NOT NULL,
    cost_credits INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'dispatched', 'done', 'failed', 'failed_lost', 'cancelled')),
    error_code TEXT,
    output_r2_keys TEXT,
    output_zip_key TEXT,
    output_ttl_at INTEGER,
    space_url TEXT,
    started_at INTEGER,
    finished_at INTEGER,
    last_heartbeat INTEGER,
    created_at INTEGER NOT NULL,
    UNIQUE(user_id, idempotency_key)
);

CREATE INDEX idx_jobs_status ON jobs(status, created_at);
CREATE INDEX idx_jobs_user ON jobs(user_id, created_at DESC);

CREATE TABLE pricing_rules (
    id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    multiplier REAL NOT NULL,
    base_per_sec INTEGER NOT NULL,
    effective_from INTEGER NOT NULL,
    effective_to INTEGER
);

CREATE TABLE webhook_outbox (
    id TEXT PRIMARY KEY,
    target_url TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_try_at INTEGER NOT NULL,
    delivered_at INTEGER,
    created_at INTEGER NOT NULL
);

CREATE TABLE auth_tokens (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    consumed_at INTEGER,
    created_at INTEGER NOT NULL
);

INSERT INTO pricing_rules (id, model, multiplier, base_per_sec, effective_from)
VALUES
    ('price_htdemucs_v1', 'htdemucs', 1.0, 833, 0),
    ('price_htdemucs_ft_v1', 'htdemucs_ft', 2.0, 833, 0),
    ('price_htdemucs_6s_v1', 'htdemucs_6s', 1.4, 833, 0),
    ('price_mdxnet_vocal_v1', 'mdxnet_vocal', 1.0, 833, 0);

