-- ============================================================
-- 智能运维内置模块
-- ============================================================

CREATE TABLE IF NOT EXISTS `ai_providers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '配置名称',
  `provider` varchar(50) NOT NULL DEFAULT 'openai-compatible' COMMENT '供应商类型',
  `base_url` varchar(500) NOT NULL COMMENT 'API地址',
  `api_key` varchar(1000) DEFAULT NULL COMMENT 'API Key',
  `model` varchar(100) NOT NULL COMMENT '模型名称',
  `temperature` decimal(4,2) DEFAULT 0.20 COMMENT '温度',
  `max_tokens` int DEFAULT 8192 COMMENT '最大输出Token',
  `timeout` int DEFAULT 180 COMMENT '超时时间秒',
  `reasoning_effort` varchar(20) DEFAULT NULL COMMENT '推理强度',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `is_default` tinyint(1) DEFAULT 0 COMMENT '是否默认',
  `remark` varchar(500) DEFAULT NULL COMMENT '备注',
  `last_test_at` datetime(3) DEFAULT NULL COMMENT '最后测试时间',
  `last_test_msg` varchar(500) DEFAULT NULL COMMENT '最后测试结果',
  PRIMARY KEY (`id`),
  KEY `idx_ai_providers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI模型供应商配置';

CREATE TABLE IF NOT EXISTS `ai_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户ID',
  `username` varchar(100) DEFAULT NULL COMMENT '用户名',
  `title` varchar(200) DEFAULT NULL COMMENT '会话标题',
  `type` varchar(50) DEFAULT 'chat' COMMENT '会话类型',
  `status` varchar(30) DEFAULT 'active' COMMENT '状态',
  `summary` longtext COMMENT '摘要',
  PRIMARY KEY (`id`),
  KEY `idx_ai_sessions_deleted_at` (`deleted_at`),
  KEY `idx_ai_sessions_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI会话';

CREATE TABLE IF NOT EXISTS `ai_messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `session_id` bigint unsigned DEFAULT NULL COMMENT '会话ID',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户ID',
  `role` varchar(30) NOT NULL COMMENT '角色',
  `content` longtext COMMENT '内容',
  `model` varchar(100) DEFAULT NULL COMMENT '模型',
  `status` varchar(30) DEFAULT 'success' COMMENT '状态',
  `tokens_in` int DEFAULT 0 COMMENT '输入Token',
  `tokens_out` int DEFAULT 0 COMMENT '输出Token',
  `latency_ms` bigint DEFAULT 0 COMMENT '耗时毫秒',
  `error` text COMMENT '错误信息',
  `context_ref` longtext COMMENT '上下文引用JSON',
  PRIMARY KEY (`id`),
  KEY `idx_ai_messages_deleted_at` (`deleted_at`),
  KEY `idx_ai_messages_session_id` (`session_id`),
  KEY `idx_ai_messages_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI消息';

CREATE TABLE IF NOT EXISTS `ai_tool_calls` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `session_id` bigint unsigned DEFAULT NULL COMMENT '会话ID',
  `message_id` bigint unsigned DEFAULT NULL COMMENT '消息ID',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户ID',
  `tool_name` varchar(100) NOT NULL COMMENT '工具名称',
  `params` longtext COMMENT '参数JSON',
  `result` longtext COMMENT '结果JSON',
  `status` varchar(30) DEFAULT 'success' COMMENT '状态',
  `latency_ms` bigint DEFAULT 0 COMMENT '耗时毫秒',
  `error` text COMMENT '错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_ai_tool_calls_deleted_at` (`deleted_at`),
  KEY `idx_ai_tool_calls_session_id` (`session_id`),
  KEY `idx_ai_tool_calls_message_id` (`message_id`),
  KEY `idx_ai_tool_calls_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI工具调用';

CREATE TABLE IF NOT EXISTS `ai_diagnosis_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户ID',
  `username` varchar(100) DEFAULT NULL COMMENT '用户名',
  `session_id` bigint unsigned DEFAULT NULL COMMENT '会话ID',
  `object_type` varchar(50) DEFAULT NULL COMMENT '对象类型',
  `cluster_id` bigint unsigned DEFAULT NULL COMMENT '集群ID',
  `namespace` varchar(120) DEFAULT NULL COMMENT '命名空间',
  `object_name` varchar(200) DEFAULT NULL COMMENT '对象名称',
  `container` varchar(200) DEFAULT NULL COMMENT '容器名称',
  `status` varchar(30) DEFAULT 'success' COMMENT '状态',
  `conclusion` longtext COMMENT '诊断结论',
  `evidence_json` longtext COMMENT '证据JSON',
  `suggestion` longtext COMMENT '处理建议',
  `error` text COMMENT '错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_ai_diagnosis_tasks_deleted_at` (`deleted_at`),
  KEY `idx_ai_diagnosis_tasks_user_id` (`user_id`),
  KEY `idx_ai_diagnosis_tasks_session_id` (`session_id`),
  KEY `idx_ai_diagnosis_tasks_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI诊断任务';

-- 本版本暂时关闭智能运维菜单入口，仅保留表结构和接口能力。
DELETE FROM `sys_role_menu`
WHERE `menu_id` IN (
  SELECT `id` FROM `sys_menu`
  WHERE (`code` = 'aiops' OR `code` LIKE 'aiops-%')
);

UPDATE `sys_menu`
SET `visible` = 0, `status` = 0, `deleted_at` = NOW(), `updated_at` = NOW()
WHERE (`code` = 'aiops' OR `code` LIKE 'aiops-%')
  AND `deleted_at` IS NULL;
