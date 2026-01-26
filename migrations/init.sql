-- OpsHub Database Initialization Script
-- 创建数据库的所有必要表和初始化数据
-- 执行前请确保数据库已创建: CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 1. RBAC 系统表
-- ============================================================

-- 用户表
CREATE TABLE IF NOT EXISTS `sys_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password` varchar(255) NOT NULL COMMENT '密码',
  `real_name` varchar(50) COMMENT '真实姓名',
  `email` varchar(100) COMMENT '邮箱',
  `phone` varchar(20) COMMENT '手机号',
  `avatar` varchar(255) COMMENT '头像',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `department_id` bigint unsigned DEFAULT 0 COMMENT '部门ID',
  `bio` text COMMENT '个人简介',
  `last_login_at` datetime COMMENT '最后登录时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username_deleted` (`username`, `deleted_at`),
  KEY `idx_department_id` (`department_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色表
CREATE TABLE IF NOT EXISTS `sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '角色名称',
  `code` varchar(50) NOT NULL COMMENT '角色编码',
  `description` varchar(200) COMMENT '角色描述',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`, `deleted_at`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_sort` (`sort`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 部门表
CREATE TABLE IF NOT EXISTS `sys_department` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '部门名称',
  `code` varchar(50) COMMENT '部门编码',
  `parent_id` bigint unsigned DEFAULT 0 COMMENT '父部门ID',
  `dept_type` tinyint DEFAULT 3 COMMENT '部门类型 1:公司 2:中心 3:部门',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_dept_type` (`dept_type`),
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 菜单表
CREATE TABLE IF NOT EXISTS `sys_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '菜单名称',
  `code` varchar(50) COMMENT '菜单编码',
  `type` tinyint COMMENT '菜单类型 1:目录 2:菜单 3:按钮',
  `parent_id` bigint unsigned DEFAULT 0 COMMENT '父菜单ID',
  `path` varchar(200) COMMENT '路由路径',
  `component` varchar(200) COMMENT '组件路径',
  `icon` varchar(100) COMMENT '图标',
  `sort` int DEFAULT 0 COMMENT '排序',
  `visible` tinyint DEFAULT 1 COMMENT '是否显示 1:显示 0:隐藏',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_type` (`type`),
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 职位表
CREATE TABLE IF NOT EXISTS `sys_position` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `post_name` varchar(50) NOT NULL COMMENT '职位名称',
  `post_code` varchar(50) NOT NULL COMMENT '职位编码',
  `post_status` tinyint DEFAULT 1 COMMENT '职位状态 1:启用 2:禁用',
  `remark` varchar(200) COMMENT '备注',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_post_code` (`post_code`, `deleted_at`),
  KEY `idx_post_status` (`post_status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户-角色关联表
CREATE TABLE IF NOT EXISTS `sys_user_role` (
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`user_id`, `role_id`),
  KEY `idx_role_id` (`role_id`),
  CONSTRAINT `fk_user_role_user` FOREIGN KEY (`user_id`) REFERENCES `sys_user` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_role_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色-菜单关联表
CREATE TABLE IF NOT EXISTS `sys_role_menu` (
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `menu_id` bigint unsigned NOT NULL COMMENT '菜单ID',
  PRIMARY KEY (`role_id`, `menu_id`),
  KEY `idx_menu_id` (`menu_id`),
  CONSTRAINT `fk_role_menu_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_menu_menu` FOREIGN KEY (`menu_id`) REFERENCES `sys_menu` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户-职位关联表
CREATE TABLE IF NOT EXISTS `sys_user_position` (
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `position_id` bigint unsigned NOT NULL COMMENT '职位ID',
  PRIMARY KEY (`user_id`, `position_id`),
  KEY `idx_position_id` (`position_id`),
  CONSTRAINT `fk_user_position_user` FOREIGN KEY (`user_id`) REFERENCES `sys_user` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_position_position` FOREIGN KEY (`position_id`) REFERENCES `sys_position` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 2. 审计日志表
-- ============================================================

-- 操作审计日志表
CREATE TABLE IF NOT EXISTS `sys_operation_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `real_name` varchar(50) COMMENT '真实姓名',
  `module` varchar(50) COMMENT '操作模块',
  `action` varchar(50) COMMENT '操作动作',
  `description` varchar(200) COMMENT '操作描述',
  `method` varchar(10) COMMENT '请求方法',
  `path` varchar(200) COMMENT '请求路径',
  `params` text COMMENT '请求参数',
  `status` int COMMENT '响应状态码',
  `error_msg` text COMMENT '错误信息',
  `cost_time` bigint COMMENT '耗时(毫秒)',
  `ip` varchar(50) COMMENT '客户端IP',
  `user_agent` varchar(500) COMMENT '用户代理',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_username` (`username`),
  KEY `idx_action` (`action`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 登录审计日志表
CREATE TABLE IF NOT EXISTS `sys_login_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `real_name` varchar(50) COMMENT '真实姓名',
  `login_type` varchar(20) COMMENT '登录类型',
  `login_status` varchar(20) COMMENT '登录状态',
  `login_time` datetime COMMENT '登录时间',
  `logout_time` datetime COMMENT '登出时间',
  `ip` varchar(50) COMMENT '登录IP',
  `location` varchar(100) COMMENT '登录地点',
  `user_agent` varchar(500) COMMENT '用户代理',
  `fail_reason` varchar(200) COMMENT '失败原因',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_username` (`username`),
  KEY `idx_login_time` (`login_time`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 数据变更审计日志表
CREATE TABLE IF NOT EXISTS `sys_data_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `real_name` varchar(50) COMMENT '真实姓名',
  `table_name` varchar(50) COMMENT '操作表名',
  `record_id` bigint unsigned COMMENT '记录ID',
  `action` varchar(20) COMMENT '操作类型',
  `old_data` longtext COMMENT '旧数据',
  `new_data` longtext COMMENT '新数据',
  `diff_fields` text COMMENT '变更字段',
  `ip` varchar(50) COMMENT '客户端IP',
  `user_agent` varchar(500) COMMENT '用户代理',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_table_name` (`table_name`),
  KEY `idx_record_id` (`record_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 3. 资产管理表
-- ============================================================

-- 资产组表
CREATE TABLE IF NOT EXISTS `asset_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '组名称',
  `code` varchar(50) COMMENT '组编码',
  `parent_id` bigint unsigned DEFAULT 0 COMMENT '父组ID',
  `description` varchar(500) COMMENT '描述',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 凭证表
CREATE TABLE IF NOT EXISTS `credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '凭证名称',
  `type` varchar(20) NOT NULL COMMENT '凭证类型 password/key',
  `username` varchar(100) COMMENT '用户名',
  `password` varchar(500) COMMENT '密码(加密)',
  `private_key` text COMMENT '私钥(加密)',
  `passphrase` varchar(500) COMMENT '私钥密码(加密)',
  `description` varchar(500) COMMENT '描述',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 主机表
CREATE TABLE IF NOT EXISTS `hosts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '主机名称',
  `group_id` bigint unsigned COMMENT '所属组ID',
  `type` varchar(20) DEFAULT 'self' COMMENT '主机类型 self:自建 cloud:云实例',
  `cloud_provider` varchar(50) COMMENT '云厂商',
  `cloud_instance_id` varchar(100) COMMENT '云实例ID',
  `cloud_account_id` bigint unsigned COMMENT '云账户ID',
  `ssh_user` varchar(50) NOT NULL COMMENT 'SSH用户',
  `ip` varchar(50) NOT NULL COMMENT 'IP地址',
  `port` int DEFAULT 22 COMMENT 'SSH端口',
  `credential_id` bigint unsigned COMMENT '凭证ID',
  `tags` varchar(500) COMMENT '标签',
  `description` varchar(500) COMMENT '描述',
  `status` tinyint DEFAULT -1 COMMENT '状态 1:在线 0:离线 -1:未知',
  `last_seen` datetime COMMENT '最后看到时间',
  `os` varchar(100) COMMENT '操作系统',
  `kernel` varchar(100) COMMENT '内核版本',
  `arch` varchar(50) COMMENT '架构',
  `cpu_info` text COMMENT 'CPU信息',
  `cpu_cores` int COMMENT 'CPU核心数',
  `cpu_usage` float COMMENT 'CPU使用率',
  `memory_total` bigint COMMENT '总内存',
  `memory_used` bigint COMMENT '已用内存',
  `memory_usage` float COMMENT '内存使用率',
  `disk_total` bigint COMMENT '总磁盘',
  `disk_used` bigint COMMENT '已用磁盘',
  `disk_usage` float COMMENT '磁盘使用率',
  `uptime` varchar(100) COMMENT '运行时间',
  `hostname` varchar(100) COMMENT '主机名',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_ip` (`ip`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_hosts_group` FOREIGN KEY (`group_id`) REFERENCES `asset_group` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 云账户表
CREATE TABLE IF NOT EXISTS `cloud_accounts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '账户名称',
  `provider` varchar(50) NOT NULL COMMENT '云厂商',
  `access_key` varchar(200) NOT NULL COMMENT 'AccessKey',
  `secret_key` varchar(500) NOT NULL COMMENT 'SecretKey',
  `region` varchar(100) COMMENT '默认地域',
  `description` varchar(500) COMMENT '描述',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_provider` (`provider`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色资产权限表
CREATE TABLE IF NOT EXISTS `sys_role_asset_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `asset_group_id` bigint unsigned NOT NULL COMMENT '资产组ID',
  `host_ids` json COMMENT '主机ID列表',
  `permissions` int unsigned DEFAULT 63 COMMENT '权限位',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_asset` (`role_id`, `asset_group_id`, `deleted_at`),
  KEY `idx_asset_group_id` (`asset_group_id`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_role_asset_perm_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_asset_perm_group` FOREIGN KEY (`asset_group_id`) REFERENCES `asset_group` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSH终端会话记录表（资产管理-终端审计）
CREATE TABLE IF NOT EXISTS `ssh_terminal_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `host_id` bigint unsigned NOT NULL COMMENT '主机ID',
  `host_name` varchar(100) COMMENT '主机名称',
  `host_ip` varchar(50) COMMENT '主机IP',
  `user_id` bigint unsigned NOT NULL COMMENT '操作用户ID',
  `username` varchar(100) COMMENT '用户名',
  `recording_path` varchar(500) COMMENT '录制文件路径',
  `duration` int COMMENT '会话时长(秒)',
  `file_size` bigint COMMENT '文件大小(字节)',
  `status` varchar(20) DEFAULT 'recording' COMMENT '会话状态 recording/completed/failed',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_host_id` (`host_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 4. 任务管理表 (Task Plugin)
-- ============================================================

-- 任务模板表
CREATE TABLE IF NOT EXISTS `job_templates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '模板名称',
  `code` varchar(100) NOT NULL COMMENT '模板编码',
  `description` text COMMENT '模板描述',
  `content` longtext NOT NULL COMMENT '模板内容',
  `variables` json COMMENT '变量定义',
  `category` varchar(50) NOT NULL COMMENT '分类 script/ansible/module',
  `platform` varchar(50) COMMENT '平台 linux/windows',
  `timeout` int DEFAULT 300 COMMENT '超时时间(秒)',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 0:禁用 1:启用',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_category` (`category`),
  KEY `idx_sort` (`sort`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 任务执行表
CREATE TABLE IF NOT EXISTS `job_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '任务名称',
  `template_id` bigint unsigned COMMENT '模板ID',
  `task_type` varchar(50) NOT NULL COMMENT '任务类型 manual/ansible/cron',
  `status` varchar(50) DEFAULT 'pending' COMMENT '状态 pending/running/success/failed',
  `target_hosts` text COMMENT '目标主机列表(JSON)',
  `parameters` json COMMENT '执行参数',
  `execute_time` datetime COMMENT '执行时间',
  `result` json COMMENT '执行结果',
  `error_message` text COMMENT '错误信息',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_template_id` (`template_id`),
  KEY `idx_task_type` (`task_type`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Ansible任务表
CREATE TABLE IF NOT EXISTS `ansible_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '任务名称',
  `playbook_content` longtext COMMENT 'Playbook内容',
  `playbook_path` varchar(500) COMMENT 'Playbook路径',
  `inventory` text COMMENT '清单(JSON)',
  `extra_vars` json COMMENT '额外变量',
  `tags` varchar(500) COMMENT '标签',
  `fork` int DEFAULT 5 COMMENT '并发数',
  `timeout` int DEFAULT 600 COMMENT '超时时间(秒)',
  `verbose` varchar(20) DEFAULT 'v' COMMENT '日志级别',
  `status` varchar(50) DEFAULT 'pending' COMMENT '状态 pending/running/success/failed/cancelled',
  `last_run_time` datetime COMMENT '最后执行时间',
  `last_run_result` json COMMENT '最后执行结果',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 5. Kubernetes 插件表
-- ============================================================

-- Kubernetes集群表
CREATE TABLE IF NOT EXISTS `k8s_clusters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '集群名称',
  `alias` varchar(100) COMMENT '集群别名',
  `api_endpoint` varchar(500) NOT NULL COMMENT 'API地址',
  `kube_config` text NOT NULL COMMENT 'kubeconfig(加密)',
  `version` varchar(50) COMMENT 'K8S版本',
  `status` int DEFAULT 1 COMMENT '状态 1:正常 2:连接失败 3:不可用',
  `region` varchar(100) COMMENT '地域',
  `provider` varchar(50) COMMENT '云厂商',
  `description` varchar(500) COMMENT '描述',
  `created_by` bigint unsigned COMMENT '创建者ID',
  `node_count` int DEFAULT 0 COMMENT '节点数',
  `pod_count` int DEFAULT 0 COMMENT 'Pod数',
  `status_synced_at` datetime COMMENT '状态同步时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_status` (`status`),
  KEY `idx_provider` (`provider`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户kubeconfig表
CREATE TABLE IF NOT EXISTS `k8s_user_kube_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `service_account` varchar(255) NOT NULL COMMENT 'ServiceAccount名称',
  `namespace` varchar(255) DEFAULT 'default' COMMENT '命名空间',
  `is_active` tinyint DEFAULT 1 COMMENT '是否激活',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` datetime COMMENT '撤销时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cluster_user_sa` (`cluster_id`, `user_id`, `service_account`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户K8S角色绑定表
CREATE TABLE IF NOT EXISTS `k8s_user_role_bindings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `role_name` varchar(255) NOT NULL COMMENT '角色名称',
  `role_namespace` varchar(255) DEFAULT '' COMMENT '命名空间(空=ClusterRole)',
  `role_type` varchar(50) NOT NULL COMMENT '角色类型 ClusterRole/Role',
  `bound_by` bigint unsigned NOT NULL COMMENT '绑定者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cluster_user_role` (`cluster_id`, `user_id`, `role_name`, `role_namespace`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 集群巡检记录表
CREATE TABLE IF NOT EXISTS `k8s_cluster_inspections` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `cluster_name` varchar(100) COMMENT '集群名称',
  `status` varchar(20) COMMENT '状态 running/completed/failed',
  `score` int COMMENT '健康评分',
  `check_count` int COMMENT '检查项总数',
  `pass_count` int COMMENT '通过项数',
  `warning_count` int COMMENT '警告项数',
  `fail_count` int COMMENT '失败项数',
  `duration` int COMMENT '耗时(秒)',
  `report_data` longtext COMMENT '巡检报告',
  `user_id` bigint unsigned COMMENT '执行者ID',
  `start_time` datetime COMMENT '开始时间',
  `end_time` datetime COMMENT '结束时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 终端会话记录表
CREATE TABLE IF NOT EXISTS `k8s_terminal_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `cluster_name` varchar(100) COMMENT '集群名称',
  `namespace` varchar(100) NOT NULL COMMENT '命名空间',
  `pod_name` varchar(200) NOT NULL COMMENT 'Pod名称',
  `container_name` varchar(100) NOT NULL COMMENT '容器名称',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `username` varchar(100) COMMENT '用户名',
  `recording_path` varchar(500) NOT NULL COMMENT '录制文件路径',
  `duration` int COMMENT '会话时长(秒)',
  `file_size` bigint COMMENT '文件大小(字节)',
  `status` varchar(20) DEFAULT 'completed' COMMENT '状态',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_namespace` (`namespace`),
  KEY `idx_pod_name` (`pod_name`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 6. 监控插件表
-- ============================================================

-- 域名监控表
CREATE TABLE IF NOT EXISTS `domain_monitors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `domain` varchar(255) NOT NULL COMMENT '监控域名',
  `status` varchar(20) DEFAULT 'unknown' COMMENT '状态',
  `response_time` int DEFAULT 0 COMMENT '响应时间(ms)',
  `ssl_valid` tinyint DEFAULT 0 COMMENT 'SSL是否有效',
  `ssl_expiry` datetime COMMENT 'SSL过期时间',
  `check_interval` int DEFAULT 300 COMMENT '检查间隔(秒)',
  `enable_ssl` tinyint DEFAULT 1 COMMENT '是否启用SSL检查',
  `enable_alert` tinyint DEFAULT 0 COMMENT '是否启用告警',
  `last_check` datetime COMMENT '最后检查时间',
  `next_check` datetime COMMENT '下次检查时间',
  `alert_config_id` bigint unsigned COMMENT '告警配置ID',
  `response_threshold` int DEFAULT 1000 COMMENT '响应时间阈值(ms)',
  `ssl_expiry_days` int DEFAULT 30 COMMENT '证书过期天数阈值',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_domain` (`domain`),
  KEY `idx_status` (`status`),
  KEY `idx_next_check` (`next_check`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警配置表
CREATE TABLE IF NOT EXISTS `alert_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '告警名称',
  `alert_type` varchar(20) NOT NULL COMMENT '告警类型',
  `enabled` tinyint DEFAULT 1 COMMENT '是否启用',
  `threshold` int COMMENT '阈值',
  `domain_monitor_id` bigint unsigned COMMENT '域名监控ID',
  `enable_email` tinyint DEFAULT 0 COMMENT '邮件告警',
  `enable_webhook` tinyint DEFAULT 0 COMMENT 'Webhook告警',
  `enable_wechat` tinyint DEFAULT 0 COMMENT '企业微信告警',
  `enable_dingtalk` tinyint DEFAULT 0 COMMENT '钉钉告警',
  `enable_feishu` tinyint DEFAULT 0 COMMENT '飞书告警',
  `enable_system_msg` tinyint DEFAULT 0 COMMENT '系统消息告警',
  `alert_interval` int DEFAULT 600 COMMENT '告警间隔(秒)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_alert_type` (`alert_type`),
  KEY `idx_domain_monitor_id` (`domain_monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警渠道表
CREATE TABLE IF NOT EXISTS `alert_channels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '渠道名称',
  `channel_type` varchar(20) NOT NULL COMMENT '渠道类型',
  `enabled` tinyint DEFAULT 1 COMMENT '是否启用',
  `config` text COMMENT '渠道配置(JSON)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_channel_type` (`channel_type`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警接收人表
CREATE TABLE IF NOT EXISTS `alert_receivers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '接收人名称',
  `email` varchar(100) COMMENT '邮箱',
  `phone` varchar(20) COMMENT '电话',
  `wechat_id` varchar(100) COMMENT '企业微信ID',
  `dingtalk_id` varchar(100) COMMENT '钉钉ID',
  `feishu_id` varchar(100) COMMENT '飞书ID',
  `user_id` bigint unsigned COMMENT '关联用户ID',
  `enable_email` tinyint DEFAULT 1 COMMENT '启用邮件',
  `enable_webhook` tinyint DEFAULT 0 COMMENT '启用webhook',
  `enable_wechat` tinyint DEFAULT 0 COMMENT '启用企业微信',
  `enable_dingtalk` tinyint DEFAULT 0 COMMENT '启用钉钉',
  `enable_feishu` tinyint DEFAULT 0 COMMENT '启用飞书',
  `enable_system_msg` tinyint DEFAULT 1 COMMENT '启用系统消息',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警接收人-渠道关联表
CREATE TABLE IF NOT EXISTS `alert_receiver_channels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `receiver_id` bigint unsigned NOT NULL COMMENT '接收人ID',
  `channel_id` bigint unsigned NOT NULL COMMENT '渠道ID',
  `config` text COMMENT '渠道特定配置',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_receiver_channel` (`receiver_id`, `channel_id`),
  KEY `idx_receiver_id` (`receiver_id`),
  KEY `idx_channel_id` (`channel_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警日志表
CREATE TABLE IF NOT EXISTS `alert_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `alert_type` varchar(50) NOT NULL COMMENT '告警类型',
  `domain_monitor_id` bigint unsigned NOT NULL COMMENT '监控ID',
  `domain` varchar(255) NOT NULL COMMENT '域名',
  `status` varchar(20) NOT NULL COMMENT '发送状态',
  `message` text COMMENT '告警消息',
  `channel_type` varchar(20) COMMENT '渠道类型',
  `error_msg` text COMMENT '错误信息',
  `sent_at` datetime COMMENT '发送时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_alert_type` (`alert_type`),
  KEY `idx_domain_monitor_id` (`domain_monitor_id`),
  KEY `idx_sent_at` (`sent_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 初始化数据
-- ============================================================

-- 插入默认部门
INSERT INTO `sys_department` (`id`, `name`, `code`, `parent_id`, `dept_type`, `sort`, `status`, `created_at`, `updated_at`)
VALUES (1, '总公司', 'head', 0, 1, 0, 1, NOW(), NOW());

-- 插入默认角色
INSERT INTO `sys_role` (`id`, `name`, `code`, `description`, `sort`, `status`, `created_at`, `updated_at`)
VALUES
  (1, '管理员', 'admin', '系统管理员，拥有所有权限', 0, 1, NOW(), NOW()),
  (2, '普通用户', 'user', '普通用户，具有基本操作权限', 1, 1, NOW(), NOW());

-- 插入默认菜单（从当前数据库导出的完整菜单结构）
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `created_at`, `updated_at`)
VALUES
  -- ========== 顶级菜单 ==========
  (10, '仪表盘', 'dashboard', 1, 0, '/dashboard', '', 'HomeFilled', 0, 1, 1, NOW(), NOW()),
  (15, '资产管理', 'asset-management', 1, 0, '/asset', '', 'Coin', 1, 1, 1, NOW(), NOW()),
  (23, '操作审计', 'audit', 1, 0, '/audit', '', 'Document', 50, 1, 1, NOW(), NOW()),
  (30, '插件管理', 'plugin', 1, 0, '/plugin', '', 'Grid', 80, 1, 1, NOW(), NOW()),
  (42, '监控中心', '_monitor', 1, 0, '/monitor', '', 'Monitor', 80, 1, 1, NOW(), NOW()),
  (61, '任务中心', '_task', 1, 0, '/task', '', 'Grid', 90, 1, 1, NOW(), NOW()),
  (1, '系统管理', 'system', 1, 0, '', '', 'Setting', 100, 1, 1, NOW(), NOW()),
  (29, '个人信息', 'profile', 2, 0, '/profile', 'Profile', 'UserFilled', 100, 0, 1, NOW(), NOW()),
  (36, '容器管理', '_kubernetes', 1, 0, '/kubernetes', '', 'Platform', 100, 1, 1, NOW(), NOW()),

  -- ========== 系统管理子菜单 (parent_id=1) ==========
  (2, '用户管理', 'users', 2, 1, '/users', 'system/Users', 'User', 1, 1, 1, NOW(), NOW()),
  (3, '角色管理', 'roles', 2, 1, '/roles', 'system/Roles', 'UserFilled', 2, 1, 1, NOW(), NOW()),
  (5, '菜单管理', 'menus', 2, 1, '/menus', 'system/Menus', 'Menu', 4, 1, 1, NOW(), NOW()),
  (11, '部门信息', 'dept-info', 2, 1, '/dept-info', 'system/DeptInfo', 'OfficeBuilding', 5, 1, 1, NOW(), NOW()),
  (12, '岗位信息', 'position-info', 2, 1, '/position-info', 'system/PositionInfo', 'Avatar', 6, 1, 1, NOW(), NOW()),
  (13, '系统配置', 'system-config', 2, 1, '/system-config', 'system/SystemConfig', 'Setting', 7, 1, 1, NOW(), NOW()),

  -- ========== 资产管理子菜单 (parent_id=15) ==========
  (16, '主机管理', 'host-management', 2, 15, '/asset/hosts', 'asset/Hosts', 'Monitor', 1, 1, 1, NOW(), NOW()),
  (19, '凭据管理', 'asset:credentials', 3, 15, '/asset/credentials', 'asset/Credentials', 'Lock', 2, 1, 1, NOW(), NOW()),
  (17, '业务分组', 'business-group', 2, 15, '/asset/groups', 'asset/Groups', 'Collection', 3, 1, 1, NOW(), NOW()),
  (27, '云账号管理', 'cloud-accounts', 2, 15, '/asset/cloud-accounts', 'asset/CloudAccounts', 'Cloudy', 5, 1, 1, NOW(), NOW()),
  (34, '终端审计', 'asset_terminal_audit', 2, 15, '/asset/terminal-audit', '', 'View', 5, 1, 1, NOW(), NOW()),
  (65, '权限配置', 'asset_permission', 2, 15, '/asset/permissions', 'views/asset/AssetPermission.vue', 'Lock', 6, 1, 1, NOW(), NOW()),

  -- ========== 操作审计子菜单 (parent_id=23) ==========
  (24, '操作日志', 'operation-logs', 2, 23, '/audit/operation-logs', 'audit/OperationLogs', 'Document', 1, 1, 1, NOW(), NOW()),
  (25, '登录日志', 'login-logs', 2, 23, '/audit/login-logs', 'audit/LoginLogs', 'CircleCheck', 2, 1, 1, NOW(), NOW()),

  -- ========== 插件管理子菜单 (parent_id=30) ==========
  (32, '插件列表', 'plugin-list', 2, 30, '/plugin/list', 'plugin/PluginList', 'Grid', 1, 1, 1, NOW(), NOW()),
  (33, '插件安装', 'plugin-install', 2, 30, '/plugin/install', 'plugin/PluginInstall', 'Upload', 2, 1, 1, NOW(), NOW()),

  -- ========== 容器管理子菜单 (parent_id=36) ==========
  (69, '集群管理', 'kubernetes_clusters', 2, 36, '/kubernetes/clusters', '', 'Connection', 1, 1, 1, NOW(), NOW()),
  (70, '节点管理', 'kubernetes_nodes', 2, 36, '/kubernetes/nodes', '', 'Monitor', 2, 1, 1, NOW(), NOW()),
  (71, '命名空间', 'kubernetes_namespaces', 2, 36, '/kubernetes/namespaces', '', 'FolderOpened', 3, 1, 1, NOW(), NOW()),
  (72, '工作负载', 'kubernetes_workloads', 2, 36, '/kubernetes/workloads', '', 'Grid', 4, 1, 1, NOW(), NOW()),
  (73, '网络管理', 'kubernetes_network', 2, 36, '/kubernetes/network', '', 'Connection', 5, 1, 1, NOW(), NOW()),
  (74, '配置管理', 'kubernetes_config', 2, 36, '/kubernetes/config', '', 'Tools', 6, 1, 1, NOW(), NOW()),
  (75, '存储管理', 'kubernetes_storage', 2, 36, '/kubernetes/storage', '', 'Files', 7, 1, 1, NOW(), NOW()),
  (76, '访问控制', 'kubernetes_access', 2, 36, '/kubernetes/access', '', 'Lock', 8, 1, 1, NOW(), NOW()),
  (77, '终端审计', 'kubernetes_audit', 2, 36, '/kubernetes/audit', '', 'Monitor', 9, 1, 1, NOW(), NOW()),
  (85, '应用诊断', 'kubernetes_application_diagnosis', 2, 36, '/kubernetes/application-diagnosis', '', 'Grid', 10, 1, 1, NOW(), NOW()),
  (86, '集群巡检', 'kubernetes_cluster_inspection', 2, 36, '/kubernetes/cluster-inspection', '', 'Grid', 11, 1, 1, NOW(), NOW()),

  -- ========== 监控中心子菜单 (parent_id=42) ==========
  (78, '域名监控', 'monitor_domain', 2, 42, '/monitor/domain', '', 'Monitor', 1, 1, 1, NOW(), NOW()),
  (79, '告警通道', 'monitor_alert_channels', 2, 42, '/monitor/alert-channels', '', 'Grid', 2, 1, 1, NOW(), NOW()),
  (80, '告警接收人', 'monitor_alert_receivers', 2, 42, '/monitor/alert-receivers', '', 'User', 3, 1, 1, NOW(), NOW()),
  (81, '告警日志', 'monitor_alert_logs', 2, 42, '/monitor/alert-logs', '', 'Document', 4, 1, 1, NOW(), NOW()),

  -- ========== 任务中心子菜单 (parent_id=61) ==========
  (82, '任务模板', 'task_templates', 2, 61, '/task/templates', '', 'Document', 1, 1, 1, NOW(), NOW()),
  (83, '执行任务', 'task_execute', 2, 61, '/task/execute', '', 'Tools', 2, 1, 1, NOW(), NOW()),
  (84, '文件分发', 'task_file_distribution', 2, 61, '/task/file-distribution', '', 'Files', 3, 1, 1, NOW(), NOW());

-- 为管理员角色分配所有菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 1), (1, 2), (1, 3), (1, 5), (1, 10), (1, 11), (1, 12), (1, 13), (1, 15), (1, 16), (1, 17), (1, 19),
  (1, 23), (1, 24), (1, 25), (1, 27), (1, 29), (1, 30), (1, 32), (1, 33), (1, 34), (1, 36),
  (1, 42), (1, 61), (1, 65), (1, 69), (1, 70), (1, 71), (1, 72), (1, 73), (1, 74), (1, 75), (1, 76), (1, 77),
  (1, 78), (1, 79), (1, 80), (1, 81), (1, 82), (1, 83), (1, 84), (1, 85), (1, 86);

-- 为普通用户角色分配基础菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (2, 10), (2, 15), (2, 16), (2, 17), (2, 19), (2, 27), (2, 34), (2, 65),
  (2, 23), (2, 24), (2, 25), (2, 36), (2, 42), (2, 61);

-- ============================================================
-- 11. 插件状态表
-- ============================================================

-- 插件状态表（用于记录插件启用/禁用状态）
CREATE TABLE IF NOT EXISTS `plugin_states` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '插件名称',
  `enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否启用 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 默认启用所有内置插件
INSERT INTO `plugin_states` (`name`, `enabled`, `created_at`, `updated_at`)
VALUES
  ('kubernetes', 1, NOW(), NOW()),
  ('monitor', 1, NOW(), NOW()),
  ('task', 1, NOW(), NOW());

SET FOREIGN_KEY_CHECKS = 1;

-- 创建默认的admin用户
-- 密码: 123456 (需要前端加密后的值，这里需要根据实际的密码加密方式来设置)
-- 警告: 生产环境请立即修改默认密码!
INSERT INTO `sys_user` (`id`, `username`, `password`, `real_name`, `email`, `status`, `department_id`, `created_at`, `updated_at`)
VALUES (1, 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMye4RjIvjQaY8FiKbLsxI0W.6.rPfELDci', '系统管理员', 'admin@opshub.io', 1, 1, NOW(), NOW());

-- 关联admin用户到admin角色
INSERT INTO `sys_user_role` (`user_id`, `role_id`) VALUES (1, 1);
