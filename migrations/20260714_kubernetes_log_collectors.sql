CREATE TABLE IF NOT EXISTS log_cluster_collector_credentials (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id BIGINT UNSIGNED NOT NULL,
  token_hash VARCHAR(64) NOT NULL,
  token_hint VARCHAR(16) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  rotated_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_log_cluster_collector_credentials_cluster_id (cluster_id),
  KEY idx_log_cluster_collector_credentials_status (status)
);

ALTER TABLE log_collector_instances
  MODIFY COLUMN instance_id VARCHAR(160) NOT NULL,
  ADD COLUMN IF NOT EXISTS pod_name VARCHAR(255) DEFAULT '',
  ADD COLUMN IF NOT EXISTS namespace VARCHAR(120) DEFAULT '',
  ADD COLUMN IF NOT EXISTS node_name VARCHAR(255) DEFAULT '';

ALTER TABLE log_collector_assignments
  MODIFY COLUMN instance_id VARCHAR(160) NOT NULL;
