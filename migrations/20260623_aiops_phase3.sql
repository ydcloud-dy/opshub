-- ============================================================
-- 智能运维：告警根因分析
-- ============================================================

CREATE TABLE IF NOT EXISTS `ai_root_cause_analyses` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户ID',
  `username` varchar(100) DEFAULT NULL COMMENT '用户名',
  `alert_event_id` bigint unsigned DEFAULT NULL COMMENT '告警事件ID',
  `rule_id` bigint unsigned DEFAULT NULL COMMENT '告警规则ID',
  `rule_name` varchar(160) DEFAULT NULL COMMENT '规则名称',
  `severity` varchar(30) DEFAULT NULL COMMENT '严重级别',
  `state` varchar(30) DEFAULT NULL COMMENT '告警状态',
  `summary` longtext COMMENT '摘要',
  `root_cause` longtext COMMENT '根因分析',
  `evidence_json` longtext COMMENT '证据JSON',
  `suggestion` longtext COMMENT '处理建议',
  `model` varchar(100) DEFAULT NULL COMMENT '模型',
  `fallback` tinyint(1) DEFAULT 0 COMMENT '是否本地兜底',
  `status` varchar(30) DEFAULT 'success' COMMENT '状态',
  PRIMARY KEY (`id`),
  KEY `idx_ai_root_cause_analyses_deleted_at` (`deleted_at`),
  KEY `idx_ai_root_cause_analyses_user_id` (`user_id`),
  KEY `idx_ai_root_cause_analyses_alert_event_id` (`alert_event_id`),
  KEY `idx_ai_root_cause_analyses_rule_id` (`rule_id`),
  KEY `idx_ai_root_cause_analyses_severity` (`severity`),
  KEY `idx_ai_root_cause_analyses_state` (`state`),
  KEY `idx_ai_root_cause_analyses_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI告警根因分析';

-- 本版本暂时关闭智能运维菜单入口，仅保留告警分析表结构。
DELETE FROM `sys_role_menu`
WHERE `menu_id` IN (
  SELECT `id` FROM `sys_menu`
  WHERE (`code` = 'aiops' OR `code` LIKE 'aiops-%')
);

UPDATE `sys_menu`
SET `visible` = 0, `status` = 0, `deleted_at` = NOW(), `updated_at` = NOW()
WHERE (`code` = 'aiops' OR `code` LIKE 'aiops-%')
  AND `deleted_at` IS NULL;
