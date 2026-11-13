-- OpenTermius Initial Schema Rollback
-- Down Migration

DROP TABLE IF EXISTS session_logs;
DROP TABLE IF EXISTS vault_items;
DROP TABLE IF EXISTS vault_members;
DROP TABLE IF EXISTS vaults;
DROP TABLE IF EXISTS users;
