-- 将旧版“环境属于应用”关系升级为“应用选择共享环境”。
-- 本脚本仅新增字段并迁移关系，不删除旧 application_id，便于审计和回退；
-- 后端启动时会执行同等的幂等迁移并持续维护新关系。
SET NAMES utf8mb4;

DROP PROCEDURE IF EXISTS migrate_app_inventory_relationships;
DELIMITER $$
CREATE PROCEDURE migrate_app_inventory_relationships()
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_applications' AND column_name = 'environment_id'
  ) THEN
    ALTER TABLE app_inventory_applications ADD COLUMN environment_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_apps_environment (environment_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_applications' AND column_name = 'department_id'
  ) THEN
    ALTER TABLE app_inventory_applications ADD COLUMN department_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_apps_department (department_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_applications' AND column_name = 'health_checked_at'
  ) THEN
    ALTER TABLE app_inventory_applications ADD COLUMN health_checked_at datetime(3) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_applications' AND column_name = 'health_message'
  ) THEN
    ALTER TABLE app_inventory_applications ADD COLUMN health_message varchar(500) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_applications' AND column_name = 'health_source'
  ) THEN
    ALTER TABLE app_inventory_applications ADD COLUMN health_source varchar(40) NOT NULL DEFAULT 'asset-aggregation';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'environment_id'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN environment_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_domain_environment (environment_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'last_checked_at'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN last_checked_at datetime(3) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'response_time_ms'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN response_time_ms int NOT NULL DEFAULT 0;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'http_status_code'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN http_status_code int NOT NULL DEFAULT 0;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'probe_message'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN probe_message varchar(500) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'resolved_address'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN resolved_address varchar(255) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'tls_expires_at'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN tls_expires_at datetime(3) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_domains' AND column_name = 'tls_issuer'
  ) THEN
    ALTER TABLE app_inventory_domains ADD COLUMN tls_issuer varchar(255) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_resources' AND column_name = 'environment_id'
  ) THEN
    ALTER TABLE app_inventory_resources ADD COLUMN environment_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_resource_environment (environment_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_resources' AND column_name = 'host_id'
  ) THEN
    ALTER TABLE app_inventory_resources ADD COLUMN host_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_resource_host (host_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_resources' AND column_name = 'last_checked_at'
  ) THEN
    ALTER TABLE app_inventory_resources ADD COLUMN last_checked_at datetime(3) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_resources' AND column_name = 'response_time_ms'
  ) THEN
    ALTER TABLE app_inventory_resources ADD COLUMN response_time_ms int NOT NULL DEFAULT 0;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_resources' AND column_name = 'health_message'
  ) THEN
    ALTER TABLE app_inventory_resources ADD COLUMN health_message varchar(500) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_components' AND column_name = 'environment_id'
  ) THEN
    ALTER TABLE app_inventory_components ADD COLUMN environment_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_component_environment (environment_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_components' AND column_name = 'last_checked_at'
  ) THEN
    ALTER TABLE app_inventory_components ADD COLUMN last_checked_at datetime(3) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_components' AND column_name = 'response_time_ms'
  ) THEN
    ALTER TABLE app_inventory_components ADD COLUMN response_time_ms int NOT NULL DEFAULT 0;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_components' AND column_name = 'health_message'
  ) THEN
    ALTER TABLE app_inventory_components ADD COLUMN health_message varchar(500) NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_dependencies' AND column_name = 'source_environment_id'
  ) THEN
    ALTER TABLE app_inventory_dependencies ADD COLUMN source_environment_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_dependency_environment (source_environment_id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_discovery_runs' AND column_name = 'environment_id'
  ) THEN
    ALTER TABLE app_inventory_discovery_runs ADD COLUMN environment_id bigint unsigned NOT NULL DEFAULT 0, ADD KEY idx_app_inventory_discovery_environment (environment_id);
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'app_inventory_environments' AND column_name = 'application_id'
  ) THEN
    ALTER TABLE app_inventory_environments MODIFY COLUMN application_id bigint unsigned NOT NULL DEFAULT 0;

    UPDATE app_inventory_applications AS app
    JOIN (
      SELECT application_id,
             CAST(SUBSTRING_INDEX(GROUP_CONCAT(id ORDER BY FIELD(kind, 'production', 'staging', 'test', 'development'), id), ',', 1) AS UNSIGNED) AS environment_id
      FROM app_inventory_environments
      WHERE application_id > 0 AND deleted_at IS NULL
      GROUP BY application_id
    ) AS legacy ON legacy.application_id = app.id
    SET app.environment_id = legacy.environment_id
    WHERE app.environment_id = 0;
  END IF;

  UPDATE app_inventory_domains AS asset
  JOIN app_inventory_applications AS app ON app.id = asset.application_id
  SET asset.environment_id = app.environment_id
  WHERE app.environment_id > 0 AND COALESCE(asset.environment_id, 0) <> app.environment_id;

  UPDATE app_inventory_resources AS asset
  JOIN app_inventory_applications AS app ON app.id = asset.application_id
  SET asset.environment_id = app.environment_id
  WHERE app.environment_id > 0 AND COALESCE(asset.environment_id, 0) <> app.environment_id;

  UPDATE app_inventory_components AS asset
  JOIN app_inventory_applications AS app ON app.id = asset.application_id
  SET asset.environment_id = app.environment_id
  WHERE app.environment_id > 0 AND COALESCE(asset.environment_id, 0) <> app.environment_id;

  UPDATE app_inventory_dependencies AS dependency
  JOIN app_inventory_applications AS app ON app.id = dependency.source_application_id
  SET dependency.source_environment_id = app.environment_id
  WHERE app.environment_id > 0 AND COALESCE(dependency.source_environment_id, 0) <> app.environment_id;

  UPDATE app_inventory_discovery_runs AS run
  JOIN app_inventory_applications AS app ON app.id = run.application_id
  SET run.environment_id = app.environment_id
  WHERE app.environment_id > 0 AND COALESCE(run.environment_id, 0) <> app.environment_id;

  UPDATE app_inventory_applications
  SET health_status = 'unknown', health_message = '等待自动探测', health_source = 'asset-aggregation'
  WHERE health_source IS NULL OR health_source = '';
END$$
DELIMITER ;

CALL migrate_app_inventory_relationships();
DROP PROCEDURE IF EXISTS migrate_app_inventory_relationships;
