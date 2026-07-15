CREATE TABLE IF NOT EXISTS log_collection_policies (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  source_mode VARCHAR(20) NOT NULL DEFAULT 'host',
  description VARCHAR(500) DEFAULT '',
  paths TEXT NOT NULL,
  source_options TEXT,
  parser_type VARCHAR(32) NOT NULL DEFAULT 'raw',
  parser_config TEXT,
  multiline_config TEXT,
  filter_config TEXT,
  mask_config TEXT,
  metadata_config TEXT,
  retention_days INT NOT NULL DEFAULT 30,
  wal_config TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'draft',
  version BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_by BIGINT UNSIGNED DEFAULT 0,
  updated_by BIGINT UNSIGNED DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_collection_policies_name (name),
  KEY idx_log_collection_policies_source_mode (source_mode),
  KEY idx_log_collection_policies_status (status)
);

CREATE TABLE IF NOT EXISTS log_policy_targets (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  policy_id BIGINT UNSIGNED NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id BIGINT UNSIGNED NOT NULL,
  namespace VARCHAR(128) NOT NULL DEFAULT '',
  workload_kind VARCHAR(64) NOT NULL DEFAULT '',
  workload_name VARCHAR(255) NOT NULL DEFAULT '',
  label_selector VARCHAR(1000) DEFAULT '',
  container_include TEXT,
  container_exclude TEXT,
  created_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_policy_target (policy_id, target_type, target_id, namespace, workload_kind, workload_name),
  KEY idx_log_policy_targets_policy_id (policy_id),
  KEY idx_log_policy_targets_target_id (target_id)
);

CREATE TABLE IF NOT EXISTS log_policy_revisions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  policy_id BIGINT UNSIGNED NOT NULL,
  version BIGINT UNSIGNED NOT NULL,
  content LONGTEXT NOT NULL,
  checksum VARCHAR(64) NOT NULL,
  change_summary VARCHAR(500) DEFAULT '',
  created_by BIGINT UNSIGNED DEFAULT 0,
  created_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_policy_revision (policy_id, version),
  KEY idx_log_policy_revisions_policy_id (policy_id)
);

CREATE TABLE IF NOT EXISTS log_collector_assignments (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  instance_id VARCHAR(120) NOT NULL,
  policy_id BIGINT UNSIGNED NOT NULL,
  policy_version BIGINT UNSIGNED NOT NULL,
  desired_state VARCHAR(20) NOT NULL DEFAULT 'active',
  apply_status VARCHAR(20) NOT NULL DEFAULT 'pending',
  applied_at DATETIME(3) NULL,
  last_error TEXT,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_assignment (instance_id, policy_id),
  KEY idx_log_assignments_instance_id (instance_id),
  KEY idx_log_assignments_policy_id (policy_id),
  KEY idx_log_assignments_desired_state (desired_state),
  KEY idx_log_assignments_apply_status (apply_status)
);

ALTER TABLE log_collector_instances
  ADD COLUMN IF NOT EXISTS agent_id VARCHAR(80) DEFAULT '',
  ADD COLUMN IF NOT EXISTS mode VARCHAR(20) DEFAULT '',
  ADD COLUMN IF NOT EXISTS host_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS cluster_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS config_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_ingest_at DATETIME(3) NULL,
  ADD COLUMN IF NOT EXISTS wal_bytes BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS input_eps DOUBLE NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS output_eps DOUBLE NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS dropped_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS retry_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS reload_generation BIGINT UNSIGNED NOT NULL DEFAULT 0;
