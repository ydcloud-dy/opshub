-- ============================================================
-- 移除服务运营模块菜单
-- ============================================================
-- 该脚本只清理菜单和角色授权关系，不 DROP 应用中心历史数据表，避免升级时误删已有数据。

SET @service_ops_codes := '"service-operations","applications","application-services","application-topology","application-observability","application-dependencies","incidents","incident-active","incident-history","incident-reviews","incident-actions","changes","change-events","change-sources","change-webhooks","runbooks","runbook-list","runbook-executions","runbook-commands","health-checks","health-hosts","health-kubernetes","health-capacity","health-backups"';

DELETE rm
FROM `sys_role_menu` rm
JOIN `sys_menu` m ON m.`id` = rm.`menu_id`
WHERE FIND_IN_SET(m.`code`, REPLACE(@service_ops_codes, '"', '')) > 0;

UPDATE `sys_menu`
SET `visible` = 0,
    `status` = 0,
    `deleted_at` = COALESCE(`deleted_at`, NOW()),
    `updated_at` = NOW()
WHERE FIND_IN_SET(`code`, REPLACE(@service_ops_codes, '"', '')) > 0;

SET @has_module_feature_flags := (
  SELECT COUNT(*)
  FROM information_schema.TABLES
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'module_feature_flags'
);

SET @cleanup_feature_flags_sql := IF(
  @has_module_feature_flags > 0,
  'UPDATE `module_feature_flags` SET `enabled` = 0, `deleted_at` = COALESCE(`deleted_at`, NOW()), `updated_at` = NOW() WHERE `module_code` IN (''applications'', ''incidents'', ''changes'', ''runbooks'', ''health-checks'')',
  'SELECT 1'
);

PREPARE cleanup_feature_flags_stmt FROM @cleanup_feature_flags_sql;
EXECUTE cleanup_feature_flags_stmt;
DEALLOCATE PREPARE cleanup_feature_flags_stmt;
