-- 应用资产中心第一阶段表结构。
-- 后端插件启动时也会执行 GORM AutoMigrate；此文件用于离线迁移/审查。
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_applications (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL,
  updated_at datetime(3) NULL,
  deleted_at datetime(3) NULL,
  code varchar(80) NOT NULL,
  name varchar(120) NOT NULL,
  description varchar(1000) NULL,
  environment_id bigint unsigned NOT NULL DEFAULT 0,
  owner_name varchar(120) NULL,
  owner_user_id bigint unsigned NOT NULL DEFAULT 0,
  department_id bigint unsigned NOT NULL DEFAULT 0,
  team varchar(120) NULL,
  criticality varchar(20) NOT NULL DEFAULT 'medium',
  status varchar(20) NOT NULL DEFAULT 'active',
  lifecycle varchar(20) NOT NULL DEFAULT 'production',
  health_status varchar(20) NOT NULL DEFAULT 'unknown',
  health_checked_at datetime(3) NULL,
  health_message varchar(500) NULL,
  health_source varchar(40) NOT NULL DEFAULT 'asset-aggregation',
  repository_url varchar(500) NULL,
  documentation_url varchar(500) NULL,
  language varchar(80) NULL,
  tags text NULL,
  created_by bigint unsigned NOT NULL DEFAULT 0,
  updated_by bigint unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (id), KEY idx_app_inventory_apps_code (code), KEY idx_app_inventory_apps_deleted (deleted_at),
  KEY idx_app_inventory_apps_environment (environment_id), KEY idx_app_inventory_apps_department (department_id),
  KEY idx_app_inventory_apps_status (status), KEY idx_app_inventory_apps_health (health_status), KEY idx_app_inventory_apps_health_checked (health_checked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_environments (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL, updated_at datetime(3) NULL, deleted_at datetime(3) NULL,
  code varchar(40) NOT NULL, name varchar(80) NOT NULL,
  kind varchar(20) NOT NULL DEFAULT 'production', region varchar(100) NULL,
  status varchar(20) NOT NULL DEFAULT 'active', description varchar(500) NULL,
  created_by bigint unsigned NOT NULL DEFAULT 0, updated_by bigint unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (id), KEY idx_app_inventory_env_code (code), KEY idx_app_inventory_env_kind (kind), KEY idx_app_inventory_env_status (status), KEY idx_app_inventory_env_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_domains (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL, updated_at datetime(3) NULL, deleted_at datetime(3) NULL,
  application_id bigint unsigned NOT NULL, environment_id bigint unsigned NOT NULL DEFAULT 0,
  domain varchar(255) NOT NULL, protocol varchar(10) NOT NULL DEFAULT 'https', port int NOT NULL DEFAULT 443,
  path varchar(255) NOT NULL DEFAULT '/', dns_provider varchar(80) NULL, certificate_id bigint unsigned NOT NULL DEFAULT 0,
  is_primary tinyint(1) NOT NULL DEFAULT 0, status varchar(20) NOT NULL DEFAULT 'unknown', source varchar(20) NOT NULL DEFAULT 'manual', description varchar(500) NULL,
  last_checked_at datetime(3) NULL, response_time_ms int NOT NULL DEFAULT 0, http_status_code int NOT NULL DEFAULT 0,
  probe_message varchar(500) NULL, resolved_address varchar(255) NULL, tls_expires_at datetime(3) NULL, tls_issuer varchar(255) NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_domain_app (application_id), KEY idx_app_inventory_domain_environment (environment_id),
  KEY idx_app_inventory_domain_name (domain), KEY idx_app_inventory_domain_cert (certificate_id), KEY idx_app_inventory_domain_status (status),
  KEY idx_app_inventory_domain_checked (last_checked_at), KEY idx_app_inventory_domain_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_resources (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL, updated_at datetime(3) NULL, deleted_at datetime(3) NULL,
  application_id bigint unsigned NOT NULL, environment_id bigint unsigned NOT NULL DEFAULT 0,
  kind varchar(40) NOT NULL, name varchar(180) NOT NULL, address varchar(500) NULL, port int NOT NULL DEFAULT 0,
  host_id bigint unsigned NOT NULL DEFAULT 0, cluster_id bigint unsigned NOT NULL DEFAULT 0, namespace varchar(120) NULL,
  external_id varchar(500) NULL, credential_id bigint unsigned NOT NULL DEFAULT 0, status varchar(20) NOT NULL DEFAULT 'unknown',
  source varchar(20) NOT NULL DEFAULT 'manual', metadata longtext NULL, description varchar(500) NULL, last_synced_at datetime(3) NULL,
  last_checked_at datetime(3) NULL, response_time_ms int NOT NULL DEFAULT 0, health_message varchar(500) NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_resource_app (application_id), KEY idx_app_inventory_resource_environment (environment_id),
  KEY idx_app_inventory_resource_kind (kind), KEY idx_app_inventory_resource_host (host_id), KEY idx_app_inventory_resource_cluster (cluster_id),
  KEY idx_app_inventory_resource_status (status), KEY idx_app_inventory_resource_checked (last_checked_at), KEY idx_app_inventory_resource_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_components (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL, updated_at datetime(3) NULL, deleted_at datetime(3) NULL,
  application_id bigint unsigned NOT NULL, environment_id bigint unsigned NOT NULL DEFAULT 0,
  category varchar(30) NOT NULL, type varchar(60) NOT NULL, name varchar(150) NOT NULL, address varchar(500) NULL, port int NOT NULL DEFAULT 0,
  database_name varchar(120) NULL, version varchar(80) NULL, credential_id bigint unsigned NOT NULL DEFAULT 0, tls_enabled tinyint(1) NOT NULL DEFAULT 0,
  status varchar(20) NOT NULL DEFAULT 'unknown', source varchar(20) NOT NULL DEFAULT 'manual', metadata longtext NULL, description varchar(500) NULL,
  last_checked_at datetime(3) NULL, response_time_ms int NOT NULL DEFAULT 0, health_message varchar(500) NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_component_app (application_id), KEY idx_app_inventory_component_environment (environment_id),
  KEY idx_app_inventory_component_category (category), KEY idx_app_inventory_component_credential (credential_id), KEY idx_app_inventory_component_status (status),
  KEY idx_app_inventory_component_checked (last_checked_at), KEY idx_app_inventory_component_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_dependencies (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL, updated_at datetime(3) NULL, deleted_at datetime(3) NULL,
  source_application_id bigint unsigned NOT NULL, source_environment_id bigint unsigned NOT NULL DEFAULT 0,
  target_application_id bigint unsigned NOT NULL DEFAULT 0, target_component_id bigint unsigned NOT NULL DEFAULT 0, target_resource_id bigint unsigned NOT NULL DEFAULT 0,
  target_name varchar(180) NULL, relation_type varchar(30) NOT NULL, protocol varchar(30) NULL, endpoint varchar(500) NULL, port int NOT NULL DEFAULT 0,
  criticality varchar(20) NOT NULL DEFAULT 'medium', status varchar(20) NOT NULL DEFAULT 'active', description varchar(500) NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_dep_source (source_application_id), KEY idx_app_inventory_dependency_environment (source_environment_id),
  KEY idx_app_inventory_dep_target_app (target_application_id), KEY idx_app_inventory_dep_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_credentials (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at datetime(3) NULL, updated_at datetime(3) NULL, deleted_at datetime(3) NULL,
  name varchar(120) NOT NULL, kind varchar(40) NOT NULL, username varchar(255) NULL, secret_ciphertext longtext NOT NULL,
  key_version varchar(32) NOT NULL, scope varchar(20) NOT NULL DEFAULT 'private', status varchar(20) NOT NULL DEFAULT 'active', description varchar(500) NULL,
  owner_user_id bigint unsigned NOT NULL, last_rotated_at datetime(3) NULL, expires_at datetime(3) NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_cred_kind (kind), KEY idx_app_inventory_cred_owner (owner_user_id), KEY idx_app_inventory_cred_expiry (expires_at), KEY idx_app_inventory_cred_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_credential_grants (
  id bigint unsigned NOT NULL AUTO_INCREMENT, created_at datetime(3) NULL, updated_at datetime(3) NULL,
  credential_id bigint unsigned NOT NULL, subject_type varchar(20) NOT NULL, subject_id bigint unsigned NOT NULL,
  permissions int unsigned NOT NULL DEFAULT 1, created_by bigint unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (id), KEY idx_app_inventory_grant_credential (credential_id), KEY idx_app_inventory_grant_subject (subject_type, subject_id),
  UNIQUE KEY uidx_app_inventory_credential_subject (credential_id, subject_type, subject_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_secret_audits (
  id bigint unsigned NOT NULL AUTO_INCREMENT, credential_id bigint unsigned NOT NULL, user_id bigint unsigned NOT NULL,
  username varchar(80) NULL, action varchar(30) NOT NULL, success tinyint(1) NOT NULL DEFAULT 0, reason varchar(500) NULL,
  ip varchar(80) NULL, user_agent varchar(500) NULL, created_at datetime(3) NOT NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_audit_credential (credential_id), KEY idx_app_inventory_audit_user (user_id), KEY idx_app_inventory_audit_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_inventory_discovery_runs (
  id bigint unsigned NOT NULL AUTO_INCREMENT, source_type varchar(30) NOT NULL, source_id bigint unsigned NOT NULL,
  application_id bigint unsigned NOT NULL, environment_id bigint unsigned NOT NULL DEFAULT 0, namespace varchar(120) NULL,
  selector varchar(500) NULL, status varchar(20) NOT NULL, resource_count int NOT NULL DEFAULT 0, domain_count int NOT NULL DEFAULT 0,
  error_message text NULL, created_by bigint unsigned NOT NULL DEFAULT 0, started_at datetime(3) NOT NULL, finished_at datetime(3) NULL,
  PRIMARY KEY (id), KEY idx_app_inventory_discovery_source (source_type, source_id), KEY idx_app_inventory_discovery_app (application_id),
  KEY idx_app_inventory_discovery_environment (environment_id), KEY idx_app_inventory_discovery_started (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
