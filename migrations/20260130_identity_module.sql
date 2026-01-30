-- Identity Module Migration
-- 身份认证模块数据表和菜单初始化
-- 执行时间: 2026

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 身份认证模块表
-- ============================================================

-- 身份源表
CREATE TABLE IF NOT EXISTS `identity_sources` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '身份源名称',
  `type` varchar(30) NOT NULL COMMENT '类型(wechat/dingtalk/feishu/qq/github等)',
  `icon` varchar(255) COMMENT '图标URL',
  `config` text COMMENT '配置JSON',
  `user_mapping` text COMMENT '用户属性映射',
  `auto_create_user` tinyint(1) DEFAULT 0 COMMENT '自动创建用户',
  `default_role_id` bigint unsigned DEFAULT 0 COMMENT '默认角色ID',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `sort` int DEFAULT 0 COMMENT '排序',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_enabled` (`enabled`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='身份源表';

-- SSO应用表
CREATE TABLE IF NOT EXISTS `sso_applications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '应用名称',
  `code` varchar(50) COMMENT '应用编码',
  `icon` varchar(255) COMMENT '图标URL',
  `description` varchar(500) COMMENT '应用描述',
  `category` varchar(50) COMMENT '分类(cicd/code/monitor/registry)',
  `url` varchar(500) NOT NULL COMMENT '应用URL',
  `sso_type` varchar(30) COMMENT 'SSO类型(oauth2/saml/form/token)',
  `sso_config` text COMMENT 'SSO配置JSON',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `sort` int DEFAULT 0 COMMENT '排序',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_category` (`category`),
  KEY `idx_enabled` (`enabled`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SSO应用表';

-- 用户凭证表
CREATE TABLE IF NOT EXISTS `user_credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `app_id` bigint unsigned NOT NULL COMMENT '应用ID',
  `username` varchar(100) COMMENT '应用账号',
  `password` varchar(500) COMMENT '应用密码(加密存储)',
  `extra_data` text COMMENT '额外数据JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户凭证表';

-- 应用权限表
CREATE TABLE IF NOT EXISTS `app_permissions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint unsigned NOT NULL COMMENT '应用ID',
  `subject_type` varchar(20) NOT NULL COMMENT '主体类型(user/role/dept)',
  `subject_id` bigint unsigned NOT NULL COMMENT '主体ID',
  `permission` varchar(20) DEFAULT 'access' COMMENT '权限类型',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_subject` (`subject_type`, `subject_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用权限表';

-- 用户第三方绑定表
CREATE TABLE IF NOT EXISTS `user_oauth_bindings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `source_id` bigint unsigned NOT NULL COMMENT '身份源ID',
  `source_type` varchar(30) NOT NULL COMMENT '身份源类型',
  `open_id` varchar(255) COMMENT 'OpenID',
  `union_id` varchar(255) COMMENT 'UnionID',
  `nickname` varchar(100) COMMENT '昵称',
  `avatar` varchar(500) COMMENT '头像URL',
  `extra_info` text COMMENT '额外信息JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_open_id` (`open_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户第三方绑定表';

-- 认证日志表
CREATE TABLE IF NOT EXISTS `auth_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `action` varchar(30) COMMENT '动作(login/logout/access_app)',
  `app_id` bigint unsigned COMMENT '应用ID',
  `app_name` varchar(100) COMMENT '应用名称',
  `login_type` varchar(30) COMMENT '登录类型',
  `ip` varchar(50) COMMENT 'IP地址',
  `location` varchar(100) COMMENT '地理位置',
  `user_agent` varchar(500) COMMENT 'UserAgent',
  `result` varchar(20) COMMENT '结果(success/failed)',
  `fail_reason` varchar(255) COMMENT '失败原因',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_action` (`action`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='认证日志表';

-- 用户收藏应用表
CREATE TABLE IF NOT EXISTS `user_favorite_apps` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `app_id` bigint unsigned NOT NULL COMMENT '应用ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_app` (`user_id`, `app_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_app_id` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户收藏应用表';

-- ============================================================
-- 添加身份认证模块菜单
-- ============================================================

-- 插入身份认证顶级菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `created_at`, `updated_at`)
VALUES (100, '身份认证', 'identity', 1, 0, '/identity', '', 'Key', 200, 1, 1, NOW(), NOW());

-- 插入身份认证子菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `created_at`, `updated_at`)
VALUES
  (101, '应用门户', 'identity_portal', 2, 100, '/identity/portal', 'identity/Portal', 'Grid', 1, 1, 1, NOW(), NOW()),
  (102, '身份源管理', 'identity_sources', 2, 100, '/identity/sources', 'identity/IdentitySources', 'User', 2, 1, 1, NOW(), NOW()),
  (103, '应用管理', 'identity_apps', 2, 100, '/identity/apps', 'identity/SSOApplications', 'Grid', 3, 1, 1, NOW(), NOW()),
  (104, '凭证管理', 'identity_credentials', 2, 100, '/identity/credentials', 'identity/Credentials', 'Lock', 4, 1, 1, NOW(), NOW()),
  (105, '访问策略', 'identity_permissions', 2, 100, '/identity/permissions', 'identity/Permissions', 'Key', 5, 1, 1, NOW(), NOW()),
  (106, '认证日志', 'identity_logs', 2, 100, '/identity/logs', 'identity/AuthLogs', 'Document', 6, 1, 1, NOW(), NOW());

-- 为管理员角色分配身份认证菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 100), (1, 101), (1, 102), (1, 103), (1, 104), (1, 105), (1, 106);

-- 为普通用户分配应用门户和凭证管理权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (2, 100), (2, 101), (2, 104);

SET FOREIGN_KEY_CHECKS = 1;
