-- Add Agent management menu under Asset Management.

INSERT INTO `sys_menu` (`name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `created_at`, `updated_at`)
SELECT 'Agent管理', 'asset-agent-management', 2, m.id, '/asset/agents', 'asset/Agents', 'Connection', 2, 1, 1, NOW(), NOW()
FROM `sys_menu` m
WHERE m.code = 'asset-management'
  AND NOT EXISTS (
    SELECT 1 FROM `sys_menu` x WHERE x.code = 'asset-agent-management' AND x.deleted_at IS NULL
  )
LIMIT 1;

UPDATE `sys_menu`
SET `name` = 'Agent管理',
    `path` = '/asset/agents',
    `component` = 'asset/Agents',
    `icon` = 'Connection',
    `sort` = 2,
    `visible` = 1,
    `status` = 1,
    `updated_at` = NOW()
WHERE `code` = 'asset-agent-management'
  AND `deleted_at` IS NULL;

INSERT IGNORE INTO `sys_role_menu` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `sys_role` r
JOIN `sys_menu` m ON m.code = 'asset-agent-management' AND m.deleted_at IS NULL
WHERE r.code IN ('admin', 'user')
  AND r.deleted_at IS NULL;
