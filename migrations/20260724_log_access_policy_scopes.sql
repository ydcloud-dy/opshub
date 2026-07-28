ALTER TABLE log_access_policies
  ADD COLUMN IF NOT EXISTS scope_mode VARCHAR(32) NOT NULL DEFAULT 'all' AFTER library_item_pattern;

UPDATE log_access_policies
SET scope_mode = 'all'
WHERE scope_mode IS NULL OR scope_mode = '';

CREATE TABLE IF NOT EXISTS log_access_policy_scopes (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  access_policy_id BIGINT UNSIGNED NOT NULL,
  collection_policy_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_access_policy_scope (access_policy_id, collection_policy_id),
  KEY idx_log_access_policy_scopes_access_policy_id (access_policy_id),
  KEY idx_log_access_policy_scopes_collection_policy_id (collection_policy_id)
);
