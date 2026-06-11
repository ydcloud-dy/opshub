-- Copyright (c) 2026 YDCloud
--
-- Permission is hereby granted, free of charge, to any person obtaining a copy of
-- this software and associated documentation files (the "Software"), to deal in
-- the Software without restriction, including without limitation the rights to
-- use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
-- the Software, and to permit persons to whom the Software is furnished to do so,
-- subject to the following conditions:
--
-- The above copyright notice and this permission notice shall be included in all
-- copies or substantial portions of the Software.
--
-- THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
-- IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
-- FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
-- COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
-- IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
-- CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

-- OpsHub Database Initialization Script
-- 创建数据库的所有必要表和初始化数据
-- 执行前请确保数据库已创建: CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 1. RBAC 系统表
-- ============================================================
--
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
  `source` varchar(20) DEFAULT 'local' COMMENT '用户来源 local:本地 ldap:LDAP',
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

-- 系统配置表
CREATE TABLE IF NOT EXISTS `sys_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(100) NOT NULL COMMENT '配置键',
  `value` text COMMENT '配置值',
  `type` varchar(20) DEFAULT 'string' COMMENT '配置类型(string/int/bool/json)',
  `group` varchar(50) COMMENT '配置分组(basic/security)',
  `remark` varchar(200) COMMENT '备注说明',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_key` (`key`),
  KEY `idx_group` (`group`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户登录失败记录表
CREATE TABLE IF NOT EXISTS `sys_user_login_attempt` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `fail_count` int DEFAULT 0 COMMENT '失败次数',
  `last_fail_at` datetime COMMENT '最后失败时间',
  `locked_until` datetime COMMENT '锁定截止时间',
  PRIMARY KEY (`id`),
  KEY `idx_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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
  `agent_id` varchar(80) COMMENT 'Agent唯一标识',
  `agent_version` varchar(50) COMMENT 'Agent版本',
  `agent_status` varchar(20) COMMENT 'Agent状态 online/offline/pending',
  `agent_last_seen` datetime COMMENT 'Agent最后心跳时间',
  `agent_last_collect_at` datetime COMMENT 'Agent最后采集时间',
  `agent_token_hash` varchar(128) COMMENT 'Agent认证Token哈希',
  `agent_install_token_hash` varchar(128) COMMENT 'Agent安装注册码哈希',
  `agent_install_token_expires_at` datetime COMMENT 'Agent安装注册码过期时间',
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
  KEY `idx_hosts_agent_id` (`agent_id`),
  KEY `idx_hosts_agent_install_token_hash` (`agent_install_token_hash`),
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
-- 12. SSL证书管理插件
-- ============================================================

-- SSL证书表
CREATE TABLE IF NOT EXISTS `ssl_certificates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '证书名称',
  `domain` varchar(255) NOT NULL COMMENT '主域名',
  `san_domains` text COMMENT 'SAN域名(JSON数组)',
  `acme_email` varchar(255) COMMENT 'ACME注册邮箱',
  `ca_provider` varchar(20) COMMENT 'CA提供商: letsencrypt/zerossl/google/buypass',
  `key_algorithm` varchar(20) COMMENT '密钥算法: rsa2048/rsa3072/rsa4096/ec256/ec384',
  `source_type` varchar(20) COMMENT '证书来源: acme/aliyun/manual',
  `cloud_account_id` bigint unsigned DEFAULT NULL COMMENT '云账号ID',
  `cloud_cert_id` varchar(100) COMMENT '云厂商证书ID',
  `certificate` text COMMENT '证书PEM',
  `private_key` text COMMENT '私钥PEM(加密)',
  `cert_chain` text COMMENT '证书链',
  `issuer` varchar(255) COMMENT '签发机构',
  `not_before` datetime COMMENT '生效时间',
  `not_after` datetime COMMENT '过期时间',
  `fingerprint` varchar(100) COMMENT '指纹',
  `status` varchar(20) DEFAULT 'pending' COMMENT '状态: pending/active/expiring/expired/error',
  `auto_renew` tinyint(1) DEFAULT 1 COMMENT '自动续期',
  `renew_days_before` int DEFAULT 30 COMMENT '提前续期天数',
  `dns_provider_id` bigint unsigned DEFAULT NULL COMMENT 'DNS服务商ID',
  `last_renew_at` datetime COMMENT '最后续期时间',
  `last_error` text COMMENT '最后错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_certificates_deleted_at` (`deleted_at`),
  KEY `idx_ssl_certificates_domain` (`domain`),
  KEY `idx_ssl_certificates_not_after` (`not_after`),
  KEY `idx_ssl_certificates_cloud_account_id` (`cloud_account_id`),
  KEY `idx_ssl_certificates_dns_provider_id` (`dns_provider_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSL DNS服务商配置表
CREATE TABLE IF NOT EXISTS `ssl_dns_providers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '名称',
  `provider` varchar(50) NOT NULL COMMENT 'DNS服务商类型: aliyun/cloudflare/huawei/aws_route53',
  `config` text NOT NULL COMMENT '配置JSON(加密)',
  `email` varchar(255) COMMENT '联系邮箱',
  `phone` varchar(50) COMMENT '联系电话',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `last_test_at` datetime COMMENT '最后测试时间',
  `last_test_ok` tinyint(1) DEFAULT 0 COMMENT '最后测试结果',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_dns_providers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSL部署配置表
CREATE TABLE IF NOT EXISTS `ssl_deploy_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `certificate_id` bigint unsigned NOT NULL COMMENT '关联证书ID',
  `name` varchar(100) NOT NULL COMMENT '配置名称',
  `deploy_type` varchar(20) NOT NULL COMMENT '部署类型: nginx_ssh/k8s_secret',
  `target_config` text NOT NULL COMMENT '目标配置JSON',
  `auto_deploy` tinyint(1) DEFAULT 1 COMMENT '续期后自动部署',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `last_deploy_at` datetime COMMENT '最后部署时间',
  `last_deploy_ok` tinyint(1) DEFAULT 0 COMMENT '最后部署结果',
  `last_error` text COMMENT '最后错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_deploy_configs_deleted_at` (`deleted_at`),
  KEY `idx_ssl_deploy_configs_certificate_id` (`certificate_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSL续期任务表
CREATE TABLE IF NOT EXISTS `ssl_renew_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `certificate_id` bigint unsigned NOT NULL COMMENT '关联证书ID',
  `task_type` varchar(20) NOT NULL COMMENT '任务类型: issue/renew/deploy',
  `status` varchar(20) DEFAULT 'pending' COMMENT '状态: pending/running/success/failed',
  `trigger_type` varchar(20) NOT NULL COMMENT '触发类型: auto/manual',
  `started_at` datetime COMMENT '开始时间',
  `finished_at` datetime COMMENT '完成时间',
  `error_message` text COMMENT '错误信息',
  `result` text COMMENT '结果JSON',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_renew_tasks_deleted_at` (`deleted_at`),
  KEY `idx_ssl_renew_tasks_certificate_id` (`certificate_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 8. 统一认证模块表 (Identity)
-- ============================================================

-- 身份源表（第三方登录配置）
CREATE TABLE IF NOT EXISTS `identity_sources` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '身份源名称',
  `type` varchar(30) NOT NULL COMMENT '类型(wechat/dingtalk/feishu/qq/github/ldap/oidc/saml)',
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
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSO应用表
CREATE TABLE IF NOT EXISTS `sso_applications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '应用名称',
  `code` varchar(50) COMMENT '应用编码',
  `icon` varchar(255) COMMENT '图标URL',
  `description` varchar(500) COMMENT '应用描述',
  `category` varchar(50) COMMENT '分类(cicd/code/monitor/registry/other)',
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
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户凭证表（存储用户在各应用的账号密码）
CREATE TABLE IF NOT EXISTS `user_credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `app_id` bigint unsigned NOT NULL COMMENT '应用ID',
  `username` varchar(100) COMMENT '应用账号',
  `password` varchar(500) COMMENT '应用密码(AES加密存储)',
  `extra_data` text COMMENT '额外数据JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 应用权限表（控制用户/角色/部门对应用的访问权限）
CREATE TABLE IF NOT EXISTS `app_permissions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint unsigned NOT NULL COMMENT '应用ID',
  `subject_type` varchar(20) NOT NULL COMMENT '主体类型(user/role/dept)',
  `subject_id` bigint unsigned NOT NULL COMMENT '主体ID',
  `permission` varchar(20) DEFAULT 'access' COMMENT '权限类型(access/admin)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_subject` (`subject_type`, `subject_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户OAuth绑定表（用户与第三方账号的绑定关系）
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
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 认证日志表
CREATE TABLE IF NOT EXISTS `auth_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `action` varchar(30) COMMENT '动作(login/logout/access_app)',
  `app_id` bigint unsigned DEFAULT 0 COMMENT '应用ID',
  `app_name` varchar(100) COMMENT '应用名称',
  `login_type` varchar(30) COMMENT '登录类型(password/oauth/ldap)',
  `ip` varchar(50) COMMENT 'IP地址',
  `location` varchar(100) COMMENT '地理位置',
  `user_agent` varchar(500) COMMENT 'UserAgent',
  `result` varchar(20) COMMENT '结果(success/failed)',
  `fail_reason` varchar(255) COMMENT '失败原因',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_action` (`action`),
  KEY `idx_result` (`result`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- OAuth状态表（CSRF防护）
CREATE TABLE IF NOT EXISTS `oauth_states` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `state` varchar(64) NOT NULL COMMENT '状态码',
  `provider` varchar(30) NOT NULL COMMENT '提供商类型',
  `redirect_url` varchar(500) COMMENT '回调后重定向URL',
  `action` varchar(20) DEFAULT 'login' COMMENT '操作类型(login/bind)',
  `user_id` bigint unsigned DEFAULT 0 COMMENT '用户ID(绑定操作时使用)',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_state` (`state`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- OAuth2授权码表（OpsHub作为OAuth2服务端）
CREATE TABLE IF NOT EXISTS `oauth2_authorization_codes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL COMMENT '授权码',
  `client_id` varchar(100) NOT NULL COMMENT '客户端ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `scope` text COMMENT '授权范围',
  `redirect_uri` varchar(500) COMMENT '重定向URI',
  `code_challenge` varchar(128) COMMENT 'PKCE挑战码',
  `code_challenge_method` varchar(10) COMMENT 'PKCE方法(S256/plain)',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `used` tinyint(1) DEFAULT 0 COMMENT '是否已使用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_client_id` (`client_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- OAuth2访问令牌表
CREATE TABLE IF NOT EXISTS `oauth2_access_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token_hash` varchar(64) NOT NULL COMMENT '令牌哈希',
  `client_id` varchar(100) NOT NULL COMMENT '客户端ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `scope` text COMMENT '授权范围',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_client_id` (`client_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- OAuth2刷新令牌表
CREATE TABLE IF NOT EXISTS `oauth2_refresh_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token_hash` varchar(64) NOT NULL COMMENT '令牌哈希',
  `access_token_id` bigint unsigned NOT NULL COMMENT '关联的访问令牌ID',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `revoked` tinyint(1) DEFAULT 0 COMMENT '是否已撤销',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_access_token_id` (`access_token_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MFA设置表（双因素认证）
CREATE TABLE IF NOT EXISTS `mfa_settings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `totp_enabled` tinyint(1) DEFAULT 0 COMMENT 'TOTP是否启用',
  `totp_secret` varchar(255) COMMENT 'TOTP密钥(加密存储)',
  `totp_verified` tinyint(1) DEFAULT 0 COMMENT 'TOTP是否已验证',
  `backup_codes` text COMMENT '备用码(JSON数组,加密存储)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MFA日志表
CREATE TABLE IF NOT EXISTS `sys_mfa_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `mfa_type` varchar(20) COMMENT 'MFA类型',
  `action` varchar(20) NOT NULL COMMENT '操作(verify/enable/disable/setup)',
  `ip_address` varchar(45) COMMENT 'IP地址',
  `user_agent` varchar(500) COMMENT '用户代理',
  `success` tinyint(1) DEFAULT 0 COMMENT '是否成功',
  `message` varchar(255) COMMENT '消息',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MFA信任设备表
CREATE TABLE IF NOT EXISTS `mfa_trusted_devices` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `device_token` varchar(255) NOT NULL COMMENT '设备令牌(哈希存储)',
  `device_name` varchar(255) COMMENT '设备名称(User-Agent)',
  `ip_address` varchar(45) COMMENT 'IP地址',
  `last_verified_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '最后验证时间',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_device_token` (`device_token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MFA挑战表
CREATE TABLE IF NOT EXISTS `mfa_challenges` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `token` varchar(64) NOT NULL COMMENT '挑战令牌',
  `type` varchar(20) NOT NULL COMMENT '类型(login/action)',
  `attempts` int DEFAULT 0 COMMENT '尝试次数',
  `verified` tinyint(1) DEFAULT 0 COMMENT '是否已验证',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token` (`token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- LDAP同步任务表
CREATE TABLE IF NOT EXISTS `ldap_sync_jobs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '身份源ID',
  `status` varchar(20) NOT NULL COMMENT '状态(pending/running/completed/failed)',
  `total_users` int DEFAULT 0 COMMENT '总用户数',
  `synced_users` int DEFAULT 0 COMMENT '已同步用户数',
  `failed_users` int DEFAULT 0 COMMENT '失败用户数',
  `error_message` text COMMENT '错误信息',
  `started_at` datetime COMMENT '开始时间',
  `completed_at` datetime COMMENT '完成时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 7. Nginx 日志分析插件表
-- ============================================================

-- Nginx 数据源配置表
CREATE TABLE IF NOT EXISTS `nginx_sources` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '数据源名称',
  `type` varchar(20) NOT NULL COMMENT '数据源类型 host/k8s_ingress',
  `description` varchar(500) COMMENT '描述',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `host_id` bigint unsigned COMMENT '主机ID(host类型)',
  `log_path` varchar(500) COMMENT '日志路径(host类型)',
  `log_format` varchar(50) DEFAULT 'combined' COMMENT '日志格式',
  `cluster_id` bigint unsigned COMMENT 'K8s集群ID(k8s_ingress类型)',
  `namespace` varchar(100) COMMENT 'K8s命名空间',
  `ingress_name` varchar(100) COMMENT 'Ingress名称',
  `k8s_pod_selector` varchar(200) COMMENT 'Pod标签选择器',
  `k8s_container_name` varchar(100) COMMENT '容器名称',
  `log_format_config` text COMMENT '自定义日志格式配置',
  `geo_enabled` tinyint DEFAULT 1 COMMENT '是否启用地理位置解析',
  `session_enabled` tinyint DEFAULT 0 COMMENT '是否启用会话跟踪',
  `collect_interval` int DEFAULT 60 COMMENT '采集间隔(秒)',
  `retention_days` int DEFAULT 30 COMMENT '数据保留天数',
  `last_collect_at` datetime COMMENT '最后采集时间',
  `last_collect_logs` bigint DEFAULT 0 COMMENT '最后采集日志数',
  `last_error` varchar(500) COMMENT '最后错误信息',
  `last_file_size` bigint DEFAULT 0 COMMENT '上次文件大小',
  `last_file_offset` bigint DEFAULT 0 COMMENT '上次读取偏移量',
  `last_file_inode` bigint unsigned DEFAULT 0 COMMENT '文件inode',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_host_id` (`host_id`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx IP 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_ip` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ip_address` varchar(50) NOT NULL COMMENT 'IP地址',
  `country` varchar(50) COMMENT '国家',
  `province` varchar(50) COMMENT '省份',
  `city` varchar(50) COMMENT '城市',
  `isp` varchar(100) COMMENT '运营商',
  `is_bot` tinyint DEFAULT 0 COMMENT '是否机器人',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ip_address` (`ip_address`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx URL 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_url` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `url_hash` varchar(64) NOT NULL COMMENT 'URL哈希',
  `url_path` varchar(2000) COMMENT 'URL路径',
  `url_normalized` varchar(500) COMMENT '规范化路径',
  `host` varchar(255) COMMENT '主机名',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_url_hash` (`url_hash`),
  KEY `idx_url_normalized` (`url_normalized`),
  KEY `idx_host` (`host`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx Referer 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_referer` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `referer_hash` varchar(64) NOT NULL COMMENT 'Referer哈希',
  `referer_url` varchar(2000) COMMENT 'Referer URL',
  `referer_domain` varchar(255) COMMENT 'Referer域名',
  `referer_type` varchar(20) COMMENT '来源类型 direct/search/social/other',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_referer_hash` (`referer_hash`),
  KEY `idx_referer_domain` (`referer_domain`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx User-Agent 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_user_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ua_hash` varchar(64) NOT NULL COMMENT 'UA哈希',
  `user_agent` varchar(500) COMMENT 'User-Agent',
  `browser` varchar(50) COMMENT '浏览器',
  `browser_version` varchar(20) COMMENT '浏览器版本',
  `os` varchar(50) COMMENT '操作系统',
  `os_version` varchar(20) COMMENT '系统版本',
  `device_type` varchar(20) COMMENT '设备类型 desktop/mobile/tablet/bot',
  `is_bot` tinyint DEFAULT 0 COMMENT '是否机器人',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ua_hash` (`ua_hash`),
  KEY `idx_browser` (`browser`),
  KEY `idx_os` (`os`),
  KEY `idx_device_type` (`device_type`),
  KEY `idx_is_bot` (`is_bot`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 访问日志事实表 (星型模型)
CREATE TABLE IF NOT EXISTS `nginx_fact_access_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `timestamp` datetime NOT NULL COMMENT '访问时间',
  `ip_id` bigint unsigned COMMENT 'IP维度ID',
  `url_id` bigint unsigned COMMENT 'URL维度ID',
  `referer_id` bigint unsigned COMMENT 'Referer维度ID',
  `ua_id` bigint unsigned COMMENT 'UA维度ID',
  `method` varchar(20) COMMENT '请求方法',
  `protocol` varchar(50) COMMENT '协议',
  `status` int COMMENT '状态码',
  `body_bytes_sent` bigint COMMENT '响应大小',
  `request_time` decimal(10,3) COMMENT '请求耗时',
  `upstream_time` decimal(10,3) COMMENT '上游耗时',
  `ingress_name` varchar(100) COMMENT 'Ingress名称',
  `service_name` varchar(100) COMMENT '服务名称',
  `pod_name` varchar(100) COMMENT 'Pod名称',
  `is_pv` tinyint DEFAULT 1 COMMENT '是否页面访问',
  `session_id` varchar(64) COMMENT '会话ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_time` (`source_id`, `timestamp`),
  KEY `idx_ip_id` (`ip_id`),
  KEY `idx_url_id` (`url_id`),
  KEY `idx_referer_id` (`referer_id`),
  KEY `idx_ua_id` (`ua_id`),
  KEY `idx_method` (`method`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 访问日志表 (兼容旧版，扁平化存储)
CREATE TABLE IF NOT EXISTS `nginx_access_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `timestamp` datetime NOT NULL COMMENT '访问时间',
  `remote_addr` varchar(50) COMMENT '客户端IP',
  `remote_user` varchar(100) COMMENT '远程用户',
  `request` varchar(2000) COMMENT '请求行',
  `method` varchar(20) COMMENT '请求方法',
  `uri` varchar(1000) COMMENT '请求URI',
  `protocol` varchar(50) COMMENT '协议',
  `status` int COMMENT '状态码',
  `body_bytes_sent` bigint COMMENT '响应大小',
  `http_referer` varchar(1000) COMMENT 'Referer',
  `http_user_agent` varchar(500) COMMENT 'User-Agent',
  `request_time` decimal(10,3) COMMENT '请求耗时',
  `upstream_time` decimal(10,3) COMMENT '上游耗时',
  `host` varchar(255) COMMENT '主机名',
  `country` varchar(50) COMMENT '国家',
  `province` varchar(50) COMMENT '省份',
  `city` varchar(50) COMMENT '城市',
  `isp` varchar(100) COMMENT '运营商',
  `browser` varchar(50) COMMENT '浏览器',
  `browser_version` varchar(20) COMMENT '浏览器版本',
  `os` varchar(50) COMMENT '操作系统',
  `os_version` varchar(20) COMMENT '系统版本',
  `device_type` varchar(20) COMMENT '设备类型',
  `ingress_name` varchar(100) COMMENT 'Ingress名称',
  `service_name` varchar(100) COMMENT '服务名称',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_time` (`source_id`, `timestamp`),
  KEY `idx_source_ip` (`source_id`, `remote_addr`),
  KEY `idx_source_status` (`source_id`, `status`),
  KEY `idx_source_country` (`source_id`, `country`),
  KEY `idx_source_device` (`source_id`, `device_type`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 小时聚合统计表
CREATE TABLE IF NOT EXISTS `nginx_agg_hourly` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `hour` datetime NOT NULL COMMENT '小时时间点',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `pv_count` bigint DEFAULT 0 COMMENT 'PV数',
  `unique_ips` bigint DEFAULT 0 COMMENT '独立IP数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `max_response_time` decimal(10,3) DEFAULT 0 COMMENT '最大响应时间',
  `min_response_time` decimal(10,3) DEFAULT 0 COMMENT '最小响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `method_distribution` text COMMENT '方法分布JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source_hour` (`source_id`, `hour`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 日聚合统计表
CREATE TABLE IF NOT EXISTS `nginx_agg_daily` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `date` date NOT NULL COMMENT '日期',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `pv_count` bigint DEFAULT 0 COMMENT 'PV数',
  `unique_ips` bigint DEFAULT 0 COMMENT '独立IP数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `max_response_time` decimal(10,3) DEFAULT 0 COMMENT '最大响应时间',
  `min_response_time` decimal(10,3) DEFAULT 0 COMMENT '最小响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `top_urls` text COMMENT 'Top URL JSON',
  `top_ips` text COMMENT 'Top IP JSON',
  `top_referers` text COMMENT 'Top Referer JSON',
  `top_countries` text COMMENT 'Top 国家 JSON',
  `top_browsers` text COMMENT 'Top 浏览器 JSON',
  `top_devices` text COMMENT 'Top 设备 JSON',
  `hourly_traffic` text COMMENT '每小时流量分布 JSON',
  `method_distribution` text COMMENT '方法分布 JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source_date` (`source_id`, `date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 日统计表 (兼容旧版)
CREATE TABLE IF NOT EXISTS `nginx_daily_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `date` date NOT NULL COMMENT '日期',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `unique_visitors` bigint DEFAULT 0 COMMENT '独立访客数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `top_ur_is` text COMMENT 'Top URI JSON',
  `top_i_ps` text COMMENT 'Top IP JSON',
  `top_referers` text COMMENT 'Top Referer JSON',
  `top_user_agents` text COMMENT 'Top UA JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_date` (`date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 小时统计表 (兼容旧版)
CREATE TABLE IF NOT EXISTS `nginx_hourly_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `hour` datetime NOT NULL COMMENT '小时时间点',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `unique_visitors` bigint DEFAULT 0 COMMENT '独立访客数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_hour` (`hour`)
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
-- 注意：插件菜单（kubernetes、monitor、task、nginx、ssl-cert、test）不在此处定义
-- 插件菜单由后端在插件启用时自动同步到数据库
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `created_at`, `updated_at`)
VALUES
  -- ========== 顶级菜单 ==========
  (10, '仪表盘', 'dashboard', 1, 0, '/dashboard', '', 'HomeFilled', 0, 1, 1, NOW(), NOW()),
  (15, '资产管理', 'asset-management', 1, 0, '/asset', '', 'Coin', 1, 1, 1, NOW(), NOW()),
  -- (90, '身份认证', 'identity', 1, 0, '/identity', '', 'Key', 2, 1, 1, NOW(), NOW()),  -- 身份认证模块暂不开放
  (23, '操作审计', 'audit', 1, 0, '/audit', '', 'Document', 50, 1, 1, NOW(), NOW()),
  (30, '插件管理', 'plugin', 1, 0, '/plugin', '', 'Grid', 80, 1, 1, NOW(), NOW()),
  (1, '系统管理', 'system', 1, 0, '', '', 'Setting', 100, 1, 1, NOW(), NOW()),
  (29, '个人信息', 'profile', 2, 0, '/profile', 'Profile', 'UserFilled', 100, 0, 1, NOW(), NOW()),

  -- ========== 系统管理子菜单 (parent_id=1) ==========
  (2, '用户管理', 'users', 2, 1, '/users', 'system/Users', 'User', 1, 1, 1, NOW(), NOW()),
  (3, '角色管理', 'roles', 2, 1, '/roles', 'system/Roles', 'UserFilled', 2, 1, 1, NOW(), NOW()),
  (5, '菜单管理', 'menus', 2, 1, '/menus', 'system/Menus', 'Menu', 4, 1, 1, NOW(), NOW()),
  (11, '部门信息', 'dept-info', 2, 1, '/dept-info', 'system/DeptInfo', 'OfficeBuilding', 5, 1, 1, NOW(), NOW()),
  (12, '岗位信息', 'position-info', 2, 1, '/position-info', 'system/PositionInfo', 'Avatar', 6, 1, 1, NOW(), NOW()),
  (13, '系统配置', 'system-config', 2, 1, '/system-config', 'system/SystemConfig', 'Setting', 7, 1, 1, NOW(), NOW()),

  -- ========== 身份认证子菜单 (parent_id=90) - 身份认证模块暂不开放 ==========
  -- (91, '身份源管理', 'identity_sources', 2, 90, '/identity/sources', 'identity/IdentitySources', 'User', 1, 1, 1, NOW(), NOW()),
  -- (92, '应用管理', 'identity_apps', 2, 90, '/identity/apps', 'identity/SSOApplications', 'Grid', 2, 1, 1, NOW(), NOW()),
  -- (93, '凭证管理', 'identity_credentials', 2, 90, '/identity/credentials', 'identity/Credentials', 'Lock', 3, 1, 1, NOW(), NOW()),
  -- (94, '访问策略', 'identity_permissions', 2, 90, '/identity/permissions', 'identity/Permissions', 'Key', 4, 1, 1, NOW(), NOW()),
  -- (95, '认证日志', 'identity_logs', 2, 90, '/identity/logs', 'identity/AuthLogs', 'Document', 5, 1, 1, NOW(), NOW()),
  -- (96, '应用门户', 'identity_portal', 2, 90, '/identity/portal', 'identity/Portal', 'Menu', 6, 1, 1, NOW(), NOW()),

  -- ========== 资产管理子菜单 (parent_id=15) ==========
  (16, '主机管理', 'host-management', 2, 15, '/asset/hosts', 'asset/Hosts', 'Monitor', 1, 1, 1, NOW(), NOW()),
  (66, 'Agent管理', 'asset-agent-management', 2, 15, '/asset/agents', 'asset/Agents', 'Connection', 2, 1, 1, NOW(), NOW()),
  (19, '凭据管理', 'asset:credentials', 3, 15, '/asset/credentials', 'asset/Credentials', 'Lock', 3, 1, 1, NOW(), NOW()),
  (17, '业务分组', 'business-group', 2, 15, '/asset/groups', 'asset/Groups', 'Collection', 4, 1, 1, NOW(), NOW()),
  (27, '云账号管理', 'cloud-accounts', 2, 15, '/asset/cloud-accounts', 'asset/CloudAccounts', 'Cloudy', 5, 1, 1, NOW(), NOW()),
  (34, '终端审计', 'asset_terminal_audit', 2, 15, '/asset/terminal-audit', '', 'View', 5, 1, 1, NOW(), NOW()),
  (65, '权限配置', 'asset_permission', 2, 15, '/asset/permissions', 'views/asset/AssetPermission.vue', 'Lock', 6, 1, 1, NOW(), NOW()),

  -- ========== 操作审计子菜单 (parent_id=23) ==========
  (24, '操作日志', 'operation-logs', 2, 23, '/audit/operation-logs', 'audit/OperationLogs', 'Document', 1, 1, 1, NOW(), NOW()),
  (25, '登录日志', 'login-logs', 2, 23, '/audit/login-logs', 'audit/LoginLogs', 'CircleCheck', 2, 1, 1, NOW(), NOW()),

  -- ========== 插件管理子菜单 (parent_id=30) ==========
  (32, '插件列表', 'plugin-list', 2, 30, '/plugin/list', 'plugin/PluginList', 'Grid', 1, 1, 1, NOW(), NOW()),
  (33, '插件安装', 'plugin-install', 2, 30, '/plugin/install', 'plugin/PluginInstall', 'Upload', 2, 1, 1, NOW(), NOW());

-- 为管理员角色分配所有菜单权限（不包括插件菜单，插件菜单权限在插件启用后单独分配）
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 1), (1, 2), (1, 3), (1, 5), (1, 10), (1, 11), (1, 12), (1, 13), (1, 15), (1, 16), (1, 17), (1, 19),
  (1, 23), (1, 24), (1, 25), (1, 27), (1, 29), (1, 30), (1, 32), (1, 33), (1, 34), (1, 65), (1, 66);
  -- 身份认证模块暂不开放
  -- (1, 90), (1, 91), (1, 92), (1, 93), (1, 94), (1, 95), (1, 96);

-- 为普通用户角色分配基础菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (2, 10), (2, 15), (2, 16), (2, 17), (2, 19), (2, 27), (2, 34), (2, 65), (2, 66),
  (2, 23), (2, 24), (2, 25);
  -- 身份认证模块暂不开放
  -- (2, 90), (2, 92), (2, 93), (2, 96);

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
  ('task', 1, NOW(), NOW()),
  ('ssl-cert', 1, NOW(), NOW()),
  ('nginx', 1, NOW(), NOW());

-- 插入默认系统配置
INSERT INTO `sys_config` (`key`, `value`, `type`, `group`, `remark`, `created_at`, `updated_at`)
VALUES
  -- 基础配置
  ('system_name', 'OpsHub', 'string', 'basic', '系统名称', NOW(), NOW()),
  ('system_logo', '', 'string', 'basic', '系统Logo路径', NOW(), NOW()),
  ('system_description', '运维管理平台', 'string', 'basic', '系统描述', NOW(), NOW()),
  -- 安全配置
  ('password_min_length', '8', 'int', 'security', '密码最小长度', NOW(), NOW()),
  ('session_timeout', '3600', 'int', 'security', 'Session超时时间(秒)', NOW(), NOW()),
  ('enable_captcha', 'true', 'bool', 'security', '是否开启验证码', NOW(), NOW()),
  ('max_login_attempts', '5', 'int', 'security', '最大登录失败次数', NOW(), NOW()),
  ('lockout_duration', '300', 'int', 'security', '账户锁定时间(秒)', NOW(), NOW()),
  -- MFA配置
  ('mfa_enabled', 'false', 'bool', 'security', '是否启用MFA功能', NOW(), NOW()),
  ('mfa_enforced', 'false', 'bool', 'security', '是否强制所有用户启用MFA', NOW(), NOW()),
  ('mfa_type', 'totp', 'string', 'security', 'MFA类型(totp)', NOW(), NOW()),
  ('mfa_skip_duration', '2592000', 'int', 'security', 'MFA记住设备时长(秒)', NOW(), NOW());

SET FOREIGN_KEY_CHECKS = 1;

-- 创建默认的admin用户
-- 密码: 123456
-- 警告: 生产环境请立即修改默认密码!
INSERT INTO `sys_user` (`id`, `username`, `password`, `real_name`, `email`, `status`, `department_id`, `created_at`, `updated_at`)
VALUES (1, 'admin', '$2a$10$RLkgoedTSa0dYj3ujbXMcunSED3c6GLvfdKYsmpz0l0YFZbVrSBqW', '系统管理员', 'admin@opshub.io', 1, 1, NOW(), NOW());

-- 关联admin用户到admin角色
INSERT INTO `sys_user_role` (`user_id`, `role_id`) VALUES (1, 1);
