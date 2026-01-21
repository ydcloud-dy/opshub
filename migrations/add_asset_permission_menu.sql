-- 添加资产权限配置菜单
INSERT INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status, created_at, updated_at)
SELECT '权限配置', 'asset_permission', 2, id, '/asset/permissions', 'views/asset/AssetPermission.vue', 'Lock', 6, 1, 1, NOW(), NOW()
FROM sys_menu WHERE name = '资产管理' AND type = 1;

-- 将菜单权限分配给超级管理员角色
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id
FROM sys_role r, sys_menu m
WHERE r.code = 'admin'
  AND m.code = 'asset_permission'
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.role_id = r.id AND rm.menu_id = m.id
  );
