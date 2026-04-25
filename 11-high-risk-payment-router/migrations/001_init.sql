-- RouteKit MVP schema. Compatible with Postgres; simple enough for SQLite-style local ports.
-- Token-only model: no PAN, CVV, or raw card fields exist in this schema.

CREATE TABLE IF NOT EXISTS merchants (
  id TEXT PRIMARY KEY,
  legal_name TEXT NOT NULL,
  country TEXT NOT NULL,
  mcc TEXT NOT NULL,
  settlement_currency CHAR(3) NOT NULL,
  api_key_hash TEXT NOT NULL,
  hmac_secret_encrypted TEXT NOT NULL,
  webhook_signing_secret_encrypted TEXT NOT NULL,
  pci_attestation_signed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS psp_credentials (
  id TEXT PRIMARY KEY,
  merchant_id TEXT NOT NULL REFERENCES merchants(id),
  psp_code TEXT NOT NULL,
  credential_blob BLOB NOT NULL,
  kek_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('active','disabled','underwriting')),
  added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS routing_rules (
  id TEXT PRIMARY KEY,
  merchant_id TEXT NOT NULL REFERENCES merchants(id),
  priority INTEGER NOT NULL,
  predicate TEXT NOT NULL,
  action TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payment_methods (
  id TEXT PRIMARY KEY,
  merchant_id TEXT NOT NULL REFERENCES merchants(id),
  vault_token TEXT NOT NULL,
  bin6 TEXT,
  last4 TEXT,
  brand TEXT,
  expiry_month INTEGER,
  expiry_year INTEGER,
  customer_ref TEXT,
  network_token_status TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS transactions (
  id TEXT PRIMARY KEY,
  merchant_id TEXT NOT NULL REFERENCES merchants(id),
  idempotency_key TEXT NOT NULL,
  payment_method_id TEXT REFERENCES payment_methods(id),
  amount_minor BIGINT NOT NULL,
  currency CHAR(3) NOT NULL,
  state TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  current_psp TEXT,
  current_psp_txn_id TEXT,
  decline_reason_normalised TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (merchant_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS transaction_attempts (
  id TEXT PRIMARY KEY,
  transaction_id TEXT NOT NULL REFERENCES transactions(id),
  attempt_no INTEGER NOT NULL,
  psp_code TEXT NOT NULL,
  psp_txn_id TEXT,
  request_blob TEXT NOT NULL,
  response_blob TEXT NOT NULL,
  success BOOLEAN NOT NULL,
  latency_ms INTEGER NOT NULL,
  decline_code_raw TEXT,
  decline_code_normalised TEXT,
  started_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS webhooks_in (
  id TEXT PRIMARY KEY,
  psp_code TEXT NOT NULL,
  signature_verified BOOLEAN NOT NULL,
  event_id TEXT NOT NULL,
  raw_body BLOB NOT NULL,
  headers TEXT NOT NULL,
  processed_at TIMESTAMP,
  status TEXT NOT NULL,
  UNIQUE (psp_code, event_id)
);

CREATE TABLE IF NOT EXISTS webhooks_out (
  id TEXT PRIMARY KEY,
  merchant_id TEXT NOT NULL REFERENCES merchants(id),
  transaction_id TEXT NOT NULL REFERENCES transactions(id),
  event_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMP,
  delivered_at TIMESTAMP,
  status TEXT NOT NULL CHECK (status IN ('pending','delivered','dead'))
);

CREATE TABLE IF NOT EXISTS ledger_entries (
  id INTEGER PRIMARY KEY,
  transaction_id TEXT NOT NULL REFERENCES transactions(id),
  attempt_id TEXT REFERENCES transaction_attempts(id),
  account TEXT NOT NULL,
  direction CHAR(2) NOT NULL CHECK (direction IN ('DR','CR')),
  amount_minor BIGINT NOT NULL,
  currency CHAR(3) NOT NULL,
  occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  source TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS psp_health (
  psp_code TEXT NOT NULL,
  region TEXT NOT NULL,
  ewma_auth_rate REAL NOT NULL,
  p95_latency_ms INTEGER NOT NULL,
  circuit_state TEXT NOT NULL CHECK (circuit_state IN ('closed','open','half_open')),
  window_start TIMESTAMP NOT NULL,
  window_end TIMESTAMP NOT NULL,
  PRIMARY KEY (psp_code, region)
);

