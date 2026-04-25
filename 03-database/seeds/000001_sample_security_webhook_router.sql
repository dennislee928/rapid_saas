PRAGMA foreign_keys = ON;

INSERT INTO tenant (id, name, plan, stripe_cust_id, created_at) VALUES
  ('ten_01HW0000000000000000000000', 'Aegis SOC Demo', 'starter', 'cus_demo_aegis', 1714000000000),
  ('ten_01HW0000000000000000000001', 'Free SecOps Lab', 'free', NULL, 1714003600000);

INSERT INTO user (id, tenant_id, email, clerk_user_id, role, created_at) VALUES
  ('usr_01HW0000000000000000000000', 'ten_01HW0000000000000000000000', 'owner@aegis.example', 'user_demo_aegis_owner', 'owner', 1714000001000),
  ('usr_01HW0000000000000000000001', 'ten_01HW0000000000000000000001', 'owner@lab.example', 'user_demo_lab_owner', 'owner', 1714003601000);

INSERT INTO api_key (id, tenant_id, prefix, hash, scopes, last_used_at, revoked_at, created_at) VALUES
  ('key_01HW0000000000000000000000', 'ten_01HW0000000000000000000000', 'whk_live_dem', 'argon2id$demo-live-hash', 'read,write', 1714007200000, NULL, 1714000002000),
  ('key_01HW0000000000000000000001', 'ten_01HW0000000000000000000001', 'whk_test_dem', 'argon2id$demo-test-hash', 'read,write', NULL, NULL, 1714003602000);

INSERT INTO endpoint (id, tenant_id, name, source_preset, signing_secret, signing_header, signing_algo, enabled, created_at) VALUES
  ('ep_01HW0000000000000000000000', 'ten_01HW0000000000000000000000', 'CrowdStrike production alerts', 'crowdstrike', 'secretbox:v1:demo-encrypted-signing-secret', 'X-Signature', 'sha256', 1, 1714000003000),
  ('ep_01HW0000000000000000000001', 'ten_01HW0000000000000000000000', 'GitHub Advanced Security', 'github', 'secretbox:v1:demo-github-signing-secret', 'X-Hub-Signature-256', 'sha256', 1, 1714000004000),
  ('ep_01HW0000000000000000000002', 'ten_01HW0000000000000000000001', 'Generic test ingress', 'generic', NULL, NULL, NULL, 1, 1714003603000);

INSERT INTO destination (id, tenant_id, kind, name, config_json, secret_ref, created_at) VALUES
  ('dest_01HW00000000000000000000', 'ten_01HW0000000000000000000000', 'slack', 'SOC Slack channel', '{"url":"secretbox:v1:demo-slack-url","headers":{"Content-Type":"application/json"}}', NULL, 1714000005000),
  ('dest_01HW00000000000000000001', 'ten_01HW0000000000000000000000', 'http', 'Archive webhook', '{"url":"https://archive.example.test/webhooks/security","headers":{"X-Archive-Token":"secretbox:v1:demo-token"}}', NULL, 1714000006000),
  ('dest_01HW00000000000000000002', 'ten_01HW0000000000000000000001', 'discord', 'Lab Discord', '{"url":"secretbox:v1:demo-discord-url"}', NULL, 1714003604000);

INSERT INTO rule (id, endpoint_id, position, name, filter_jsonlogic, transform_kind, transform_body, destination_id, on_match, enabled, created_at) VALUES
  ('rule_01HW0000000000000000000', 'ep_01HW0000000000000000000000', 10, 'Drop known scanner sources', '{"in_list":[{"var":"DeviceExternalIP"},"list_01HW000000000000000000"]}', 'passthrough', NULL, NULL, 'drop', 1, 1714000007000),
  ('rule_01HW0000000000000000001', 'ep_01HW0000000000000000000000', 20, 'Critical and high to Slack', '{"in":[{"var":"Severity"},["Critical","High"]]}', 'template', '{{ template "slack_blocks" . }}', 'dest_01HW00000000000000000000', 'forward', 1, 1714000008000),
  ('rule_01HW0000000000000000002', 'ep_01HW0000000000000000000000', 30, 'Archive everything else', NULL, 'passthrough', NULL, 'dest_01HW00000000000000000001', 'continue', 1, 1714000009000),
  ('rule_01HW0000000000000000003', 'ep_01HW0000000000000000000002', 10, 'Lab forward all', NULL, 'passthrough', NULL, 'dest_01HW00000000000000000002', 'forward', 1, 1714003605000);

INSERT INTO filter_list (id, tenant_id, name, kind, source, refreshed_at) VALUES
  ('list_01HW000000000000000000', 'ten_01HW0000000000000000000000', 'Known scanners', 'ip', 'manual', 1714000010000);

INSERT INTO filter_list_item (list_id, value) VALUES
  ('list_01HW000000000000000000', '198.51.100.23'),
  ('list_01HW000000000000000000', '203.0.113.42');

INSERT INTO delivery_log (id, tenant_id, endpoint_id, rule_id, destination_id, status, attempt, http_status, latency_ms, error, request_hash, request_size, received_at, delivered_at) VALUES
  ('evt_01HW0000000000000000000', 'ten_01HW0000000000000000000000', 'ep_01HW0000000000000000000000', 'rule_01HW0000000000000000001', 'dest_01HW00000000000000000000', 'delivered', 1, 200, 132, NULL, 'sha256:demo-delivered', 4096, 1714007200000, 1714007200132),
  ('evt_01HW0000000000000000001', 'ten_01HW0000000000000000000000', 'ep_01HW0000000000000000000000', 'rule_01HW0000000000000000000', NULL, 'dropped', 0, NULL, NULL, 'matched_drop_rule', 'sha256:demo-dropped', 2048, 1714007260000, NULL),
  ('evt_01HW0000000000000000002', 'ten_01HW0000000000000000000001', 'ep_01HW0000000000000000000002', 'rule_01HW0000000000000000003', 'dest_01HW00000000000000000002', 'failed', 2, 429, 98, 'destination_rate_limited', 'sha256:demo-failed', 1024, 1714007320000, NULL);

INSERT INTO usage_counter (tenant_id, bucket_hour, ingressed, forwarded, failed) VALUES
  ('ten_01HW0000000000000000000000', 476113, 2, 1, 0),
  ('ten_01HW0000000000000000000001', 476113, 1, 0, 1);

INSERT INTO dlq (id, tenant_id, endpoint_id, rule_id, destination_id, payload_b64, last_error, attempts, parked_at) VALUES
  ('dlq_01HW0000000000000000000', 'ten_01HW0000000000000000000001', 'ep_01HW0000000000000000000002', 'rule_01HW0000000000000000003', 'dest_01HW00000000000000000002', 'H4sIAAAAAAAC/6tWSkksSVSyUjI0MlbSUQooys9WslIqSa0oUbIyNDJWqgUAAP//AQAA//8=', 'destination_rate_limited', 5, 1714007920000);
