-- ============================================================
-- 临时关闭智能运维菜单入口
-- ============================================================
-- 保留 AIOps 相关表和后端能力，仅移除侧边栏/菜单管理中的菜单入口。

DELETE FROM `sys_role_menu`
WHERE `menu_id` IN (
  SELECT `id` FROM `sys_menu`
  WHERE (`code` = 'aiops' OR `code` LIKE 'aiops-%')
);

UPDATE `sys_menu`
SET `visible` = 0, `status` = 0, `deleted_at` = NOW(), `updated_at` = NOW()
WHERE (`code` = 'aiops' OR `code` LIKE 'aiops-%')
  AND `deleted_at` IS NULL;
