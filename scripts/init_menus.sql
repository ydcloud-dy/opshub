-- 创建菜单相关表（SQLite语法）
-- 注意：如果表已存在会忽略

-- 创建菜单表
CREATE TABLE IF NOT EXISTS sys_menu (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name VARCHAR(50) NOT NULL,
  code VARCHAR(50) UNIQUE NOT NULL,
  type TINYINT NOT NULL DEFAULT 2,
  parent_id INTEGER NOT NULL DEFAULT 0,
  path VARCHAR(200),
  component VARCHAR(200),
  icon VARCHAR(100),
  sort INTEGER DEFAULT 0,
  visible TINYINT DEFAULT 1,
  status TINYINT DEFAULT 1,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建角色菜单关联表
CREATE TABLE IF NOT EXISTS sys_role_menu (
  role_id INTEGER NOT NULL,
  menu_id INTEGER NOT NULL,
  PRIMARY KEY (role_id, menu_id)
);

-- 插入基础菜单数据

-- 1. 首页/仪表盘
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('首页', 'dashboard', 2, 0, '/dashboard', 'Dashboard', 'HomeFilled', 1, 1, 1);

-- 2. 系统管理（父菜单）
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('系统管理', 'system', 1, 0, '/system', '', 'Setting', 90, 1, 1);

-- 系统管理子菜单
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('用户管理', 'users', 2, (SELECT id FROM sys_menu WHERE code = 'system'), '/users', 'system/Users', 'User', 1, 1, 1),
('角色管理', 'roles', 2, (SELECT id FROM sys_menu WHERE code = 'system'), '/roles', 'system/Roles', 'Avatar', 2, 1, 1),
('菜单管理', 'menus', 2, (SELECT id FROM sys_menu WHERE code = 'system'), '/menus', 'system/Menus', 'Menu', 3, 1, 1),
('部门信息', 'dept-info', 2, (SELECT id FROM sys_menu WHERE code = 'system'), '/dept-info', 'system/DeptInfo', 'OfficeBuilding', 4, 1, 1),
('岗位信息', 'position-info', 2, (SELECT id FROM sys_menu WHERE code = 'system'), '/position-info', 'system/PositionInfo', 'Odometer', 5, 1, 1),
('系统配置', 'system-config', 2, (SELECT id FROM sys_menu WHERE code = 'system'), '/system-config', 'system/SystemConfig', 'Tools', 6, 1, 1);

-- 3. 操作审计（父菜单）
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('操作审计', 'audit', 1, 0, '/audit', '', 'Document', 70, 1, 1);

-- 操作审计子菜单
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('操作日志', 'operation-logs', 2, (SELECT id FROM sys_menu WHERE code = 'audit'), '/audit/operation-logs', 'audit/OperationLogs', 'Document', 1, 1, 1),
('登录日志', 'login-logs', 2, (SELECT id FROM sys_menu WHERE code = 'audit'), '/audit/login-logs', 'audit/LoginLogs', 'Connection', 2, 1, 1),
('数据日志', 'data-logs', 2, (SELECT id FROM sys_menu WHERE code = 'audit'), '/audit/data-logs', 'audit/DataLogs', 'DataLine', 3, 1, 1);

-- 4. 资产管理（父菜单）
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('资产管理', 'asset', 1, 0, '/asset', '', 'FolderOpened', 30, 1, 1);

-- 资产管理子菜单
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('主机管理', 'hosts', 2, (SELECT id FROM sys_menu WHERE code = 'asset'), '/asset/hosts', 'asset/Hosts', 'Monitor', 1, 1, 1),
('凭据管理', 'credentials', 2, (SELECT id FROM sys_menu WHERE code = 'asset'), '/asset/credentials', 'asset/Credentials', 'Lock', 2, 1, 1),
('云账号管理', 'cloud-accounts', 2, (SELECT id FROM sys_menu WHERE code = 'asset'), '/asset/cloud-accounts', 'asset/CloudAccounts', 'Cloudy', 3, 1, 1),
('业务分组', 'groups', 2, (SELECT id FROM sys_menu WHERE code = 'asset'), '/asset/groups', 'asset/Groups', 'Collection', 4, 1, 1),
('数据管理', 'data', 2, (SELECT id FROM sys_menu WHERE code = 'asset'), '/asset/data', 'asset/Data', 'Files', 5, 1, 1);

-- 5. Web终端
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('Web终端', 'terminal', 2, 0, '/terminal', 'asset/Terminal', 'Monitor', 40, 1, 1);

-- 6. 个人中心
INSERT OR IGNORE INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('个人信息', 'profile', 2, 0, '/profile', 'Profile', 'UserFilled', 100, 1, 1);

-- 为角色ID=1的超级管理员分配所有菜单权限
INSERT OR IGNORE INTO sys_role_menu (role_id, menu_id)
SELECT 1, id FROM sys_menu WHERE status = 1;

-- 验证结果
SELECT
  m1.id AS parent_id,
  m1.name AS parent_name,
  m1.code AS parent_code,
  m2.id AS child_id,
  m2.name AS child_name,
  m2.path AS child_path
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.type = 1 OR (m1.type = 2 AND m1.parent_id = 0)
ORDER BY m1.sort, m2.sort;
