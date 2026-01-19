-- 添加云账号管理菜单
-- 注意：请先启动应用，让GORM创建 sys_menu 表后再执行此脚本

-- 1. 创建资产管理父菜单（如果不存在）
INSERT INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status, created_at, updated_at)
VALUES ('资产管理', 'asset', 1, 0, '/asset', '', 'FolderOpened', 30, 1, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE name = '资产管理';

-- 2. 获取资产管理菜单ID
SELECT id INTO @asset_menu_id FROM sys_menu WHERE code = 'asset' LIMIT 1;

-- 3. 插入云账号管理子菜单
INSERT INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status, created_at, updated_at)
VALUES ('云账号管理', 'cloud-accounts', 2, @asset_menu_id, '/asset/cloud-accounts', 'asset/CloudAccounts', 'Cloudy', 4, 1, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  name = '云账号管理',
  path = '/asset/cloud-accounts',
  component = 'asset/CloudAccounts',
  icon = 'Cloudy',
  sort = 4;

-- 4. 为超级管理员角色分配菜单权限
-- 注意：将1替换为实际的超级管理员role_id
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 1, id FROM sys_menu WHERE code = 'cloud-accounts'
ON DUPLICATE KEY UPDATE role_id = role_id;

-- 5. 验证结果
SELECT
  m1.id AS parent_id,
  m1.name AS parent_name,
  m1.code AS parent_code,
  m2.id AS child_id,
  m2.name AS child_name,
  m2.code AS child_code,
  m2.path AS child_path
FROM sys_menu m1
INNER JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.code = 'asset' AND m2.code = 'cloud-accounts';
