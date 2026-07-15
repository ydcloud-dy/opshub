ALTER TABLE log_collection_policies
  ADD COLUMN IF NOT EXISTS retention_policy_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER metadata_config,
  ADD COLUMN IF NOT EXISTS retention_config TEXT AFTER retention_days;

CREATE TABLE IF NOT EXISTS log_retention_policies (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  description VARCHAR(500) DEFAULT '',
  storage_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  default_days INT NOT NULL DEFAULT 30,
  level_days TEXT,
  priority INT NOT NULL DEFAULT 100,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_retention_policies_name (name),
  KEY idx_log_retention_policies_storage_id (storage_id),
  KEY idx_log_retention_policies_enabled (enabled)
);

ALTER TABLE log_access_policies
  ADD COLUMN IF NOT EXISTS name VARCHAR(120) NOT NULL DEFAULT '' AFTER id,
  ADD COLUMN IF NOT EXISTS description VARCHAR(500) DEFAULT '' AFTER name,
  ADD COLUMN IF NOT EXISTS created_by BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER enabled,
  ADD COLUMN IF NOT EXISTS updated_by BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER created_by;
