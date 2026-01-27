# OpsHub 数据库初始化指南

<p align="center">
  <img src="https://img.shields.io/badge/Database-MySQL%208.0+-4479A1?style=flat&logo=mysql" alt="MySQL">
  <img src="https://img.shields.io/badge/Tables-32-blue?style=flat" alt="Tables">
  <img src="https://img.shields.io/badge/Charset-utf8mb4-green?style=flat" alt="Charset">
</p>

---

## 概述

本文档介绍如何为 OpsHub 项目初始化数据库。所有必要的表结构和初始化数据都包含在 `migrations/init.sql` 文件中。

---

## 快速开始

### 1. 创建数据库

```bash
mysql -u root -p -e "CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 2. 执行初始化脚本

```bash
mysql -u root -p opshub < migrations/init.sql
```

### 3. 验证初始化

```sql
USE opshub;
SHOW TABLES;
-- 应该看到 32 个表
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'opshub';
```

---

## 表结构概览

### 总计：32 张表

---

### 系统核心表 (11 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `sys_user` | 用户表 | username, password, real_name, email, status |
| `sys_role` | 角色表 | name, code, description, status |
| `sys_department` | 部门表 | name, code, parent_id, dept_type |
| `sys_menu` | 菜单表 | name, code, type, parent_id, path, component |
| `sys_position` | 职位表 | post_name, post_code, post_status |
| `sys_user_role` | 用户-角色关联 | user_id, role_id |
| `sys_role_menu` | 角色-菜单关联 | role_id, menu_id |
| `sys_user_position` | 用户-职位关联 | user_id, position_id |
| `sys_operation_log` | 操作审计日志 | user_id, module, action, method, path |
| `sys_login_log` | 登录审计日志 | user_id, login_type, login_status, ip |
| `sys_data_log` | 数据变更日志 | user_id, table_name, action, old_data, new_data |

---

### 资产管理表 (6 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `asset_group` | 资产分组 | name, code, parent_id, description |
| `credentials` | 访问凭证 | name, type, username, password, private_key |
| `hosts` | 主机/服务器 | name, ip, port, ssh_user, credential_id, status |
| `cloud_accounts` | 云账户 | name, provider, access_key, secret_key |
| `sys_role_asset_permission` | 角色资产权限 | role_id, asset_group_id, host_ids, permissions |
| `ssh_terminal_sessions` | SSH 终端会话 | host_id, user_id, recording_path, duration |

---

### 任务管理表 (3 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `job_templates` | 任务模板 | name, code, content, category, platform, timeout |
| `job_tasks` | 任务执行记录 | name, template_id, task_type, status, result |
| `ansible_tasks` | Ansible 任务 | name, playbook_content, inventory, status |

---

### Kubernetes 表 (5 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `k8s_clusters` | Kubernetes 集群 | name, api_endpoint, kube_config, version, status |
| `k8s_user_kube_configs` | 用户 kubeconfig | cluster_id, user_id, service_account, namespace |
| `k8s_user_role_bindings` | 用户角色绑定 | cluster_id, user_id, role_name, role_type |
| `k8s_cluster_inspections` | 集群巡检记录 | cluster_id, status, score, report_data |
| `k8s_terminal_sessions` | K8S 终端会话 | cluster_id, pod_name, container_name, recording_path |

---

### 监控告警表 (6 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `domain_monitors` | 域名监控 | domain, status, ssl_valid, ssl_expiry, response_time |
| `alert_configs` | 告警配置 | name, alert_type, enabled, threshold |
| `alert_channels` | 告警通道 | name, channel_type, enabled, config |
| `alert_receivers` | 告警接收人 | name, email, phone, wechat_id, dingtalk_id |
| `alert_receiver_channels` | 接收人-通道关联 | receiver_id, channel_id, config |
| `alert_logs` | 告警日志 | alert_type, domain, status, message, sent_at |

---

### 插件管理表 (1 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `plugin_states` | 插件状态 | name, enabled |

---

## 初始化数据

初始化脚本包含以下基础数据：

### 1. 默认部门

| ID | 名称 | 编码 | 类型 |
|:---|:-----|:-----|:-----|
| 1 | 总公司 | head | 公司 |

### 2. 默认角色

| ID | 名称 | 编码 | 说明 |
|:---|:-----|:-----|:-----|
| 1 | 管理员 | admin | 拥有所有权限 |
| 2 | 普通用户 | user | 基本操作权限 |

### 3. 默认菜单 (44 个)

包括以下功能模块：

| 模块 | 子菜单 |
|:-----|:-------|
| 仪表盘 | - |
| 资产管理 | 主机管理、凭据管理、业务分组、云账号管理、终端审计、权限配置 |
| 操作审计 | 操作日志、登录日志 |
| 插件管理 | 插件列表、插件安装 |
| 容器管理 | 集群管理、节点管理、命名空间、工作负载、网络管理、配置管理、存储管理、访问控制、终端审计、应用诊断、集群巡检 |
| 监控中心 | 域名监控、告警通道、告警接收人、告警日志 |
| 任务中心 | 任务模板、执行任务、文件分发 |
| 系统管理 | 用户管理、角色管理、菜单管理、部门信息、岗位信息、系统配置 |
| 个人信息 | - |

### 4. 默认用户

| 用户名 | 密码 | 角色 | 邮箱 |
|:-------|:-----|:-----|:-----|
| admin | 123456 | 管理员 | admin@opshub.io |

> ⚠️ **重要**: 生产环境请立即修改默认密码！

### 5. 默认插件状态

| 插件名称 | 状态 |
|:---------|:-----|
| kubernetes | 启用 |
| monitor | 启用 |
| task | 启用 |

---

## 数据库配置

确保 `config/config.yaml` 中的数据库配置正确：

```yaml
database:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  database: opshub
  username: root
  password: "your-password"
  charset: utf8mb4
  max_idle_conns: 10
  max_open_conns: 100
```

---

## 常见操作

### 重置数据库

```bash
# 删除并重建数据库
mysql -u root -p -e "DROP DATABASE IF EXISTS opshub; CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 重新导入初始化脚本
mysql -u root -p opshub < migrations/init.sql
```

### 备份数据库

```bash
# 完整备份
mysqldump -u root -p opshub > opshub_backup_$(date +%Y%m%d).sql

# 仅备份数据（不含结构）
mysqldump -u root -p --no-create-info opshub > opshub_data_$(date +%Y%m%d).sql
```

### 恢复数据库

```bash
mysql -u root -p opshub < opshub_backup.sql
```

### 重置管理员密码

```sql
-- 密码: 123456 (bcrypt 加密)
UPDATE sys_user
SET password='$2a$10$RLkgoedTSa0dYj3ujbXMcunSED3c6GLvfdKYsmpz0l0YFZbVrSBqW'
WHERE username='admin';
```

### 添加新用户

```sql
-- 插入用户
INSERT INTO sys_user (username, password, real_name, email, status, department_id, created_at, updated_at)
VALUES ('newuser', '$2a$10$xxx', '新用户', 'new@example.com', 1, 1, NOW(), NOW());

-- 获取新用户ID
SET @user_id = LAST_INSERT_ID();

-- 分配角色（普通用户）
INSERT INTO sys_user_role (user_id, role_id) VALUES (@user_id, 2);
```

---

## 数据类型说明

| 类型 | 用途 | 说明 |
|:-----|:-----|:-----|
| `bigint unsigned` | 主键/外键 | 支持更大的 ID 范围 |
| `varchar(n)` | 字符串 | 根据实际需求设定长度 |
| `text` | 中等文本 | JSON 数据、配置等 |
| `longtext` | 大文本 | 脚本内容、报告数据 |
| `json` | JSON 数据 | MySQL 5.7+ 原生支持 |
| `tinyint` | 状态标志 | 0/1 布尔值或枚举 |
| `datetime` | 时间戳 | 创建/更新时间 |

---

## 索引说明

所有关键字段都已创建索引：

| 索引类型 | 用途 |
|:---------|:-----|
| 主键索引 | 每张表的 `id` 字段 |
| 唯一索引 | 用户名、角色编码等唯一字段 |
| 普通索引 | 状态、时间戳等查询字段 |
| 外键索引 | 关联表的外键字段 |

---

## 性能优化建议

### 1. 定期清理日志

```sql
-- 删除 30 天前的操作日志
DELETE FROM sys_operation_log WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);

-- 删除 30 天前的登录日志
DELETE FROM sys_login_log WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);

-- 删除 30 天前的告警日志
DELETE FROM alert_logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
```

### 2. 优化表

```sql
OPTIMIZE TABLE sys_operation_log;
OPTIMIZE TABLE sys_login_log;
OPTIMIZE TABLE alert_logs;
```

### 3. 分析表

```sql
ANALYZE TABLE sys_user;
ANALYZE TABLE hosts;
ANALYZE TABLE k8s_clusters;
```

---

## 常见问题

### Q: 导入脚本报外键约束错误？

**A:** 临时禁用外键检查：

```sql
SET FOREIGN_KEY_CHECKS = 0;
SOURCE migrations/init.sql;
SET FOREIGN_KEY_CHECKS = 1;
```

### Q: 字符集问题导致乱码？

**A:** 确保数据库和连接都使用 `utf8mb4`：

```sql
ALTER DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### Q: 自动迁移失败？

**A:** 应用启动时 GORM 会自动迁移表结构。如果失败：

1. 查看应用日志获取详细错误
2. 检查数据库连接配置
3. 确保数据库用户有建表权限
4. 必要时手动执行 init.sql

### Q: 如何查看表结构？

```sql
-- 查看所有表
SHOW TABLES;

-- 查看表结构
DESCRIBE sys_user;

-- 查看建表语句
SHOW CREATE TABLE sys_user;
```

---

## 相关文档

- [OpsHub 主文档](../README.md)
- [部署指南](../docs/deployment.md)
- [Kubernetes 插件](../docs/plugins/kubernetes.md)
- [任务中心插件](../docs/plugins/task.md)
- [监控中心插件](../docs/plugins/monitor.md)

---

## 支持

如遇到问题，请：

1. 查看应用日志获取详细错误信息
2. 检查 MySQL 服务状态和连接配置
3. 提交 Issue: [GitHub Issues](https://github.com/ydcloud-dy/opshub/issues)
