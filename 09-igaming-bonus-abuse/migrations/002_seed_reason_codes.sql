INSERT OR IGNORE INTO reason_codes (code, severity, description) VALUES
('FP_REUSE_HIGH', 5, 'Fingerprint matches multiple accounts in the last 30 days.'),
('CLUSTER_REUSE', 5, 'Async graph cluster has multiple linked accounts.'),
('IP_DC', 3, 'Client IP is marked as datacenter hosting.'),
('IP_RES_PROXY', 4, 'Client IP is marked as residential proxy.'),
('UA_MISMATCH', 3, 'User-Agent does not match observed transport/browser signals.'),
('MOUSE_LINEAR', 2, 'Mouse movement is absent or unusually linear.'),
('WEBDRIVER_PRESENT', 5, 'Browser reports navigator.webdriver.'),
('JA3_KNOWN_BAD', 5, 'JA3 fingerprint is on a known-bad list.'),
('EMAIL_NEW_DOMAIN', 2, 'Email domain is disposable or newly observed.'),
('VELOCITY_SIGNUP_BURST', 4, 'Too many signups from one fingerprint in a short window.'),
('IP24_VELOCITY', 3, 'Too many signups from an IP bucket in a short window.'),
('TZ_GEO_MISMATCH', 3, 'Browser timezone does not match IP geolocation.'),
('COUNTRY_RULE_DENY', 4, 'Tenant rule flags the client country.');
