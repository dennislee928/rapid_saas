PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_rule_destination_same_tenant_update;
DROP TRIGGER IF EXISTS trg_rule_destination_same_tenant_insert;
DROP TABLE IF EXISTS dlq;
DROP TABLE IF EXISTS usage_counter;
DROP TABLE IF EXISTS delivery_log;
DROP TABLE IF EXISTS filter_list_item;
DROP TABLE IF EXISTS filter_list;
DROP TABLE IF EXISTS rule;
DROP TABLE IF EXISTS destination;
DROP TABLE IF EXISTS endpoint;
DROP TABLE IF EXISTS api_key;
DROP TABLE IF EXISTS user;
DROP TABLE IF EXISTS tenant;
