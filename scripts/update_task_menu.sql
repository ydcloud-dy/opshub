-- 删除旧的任务中心相关菜单
DELETE FROM sys_menu WHERE code IN ('task-jobs', 'task-templates', 'task-ansible');

-- 重新加载菜单配置（需要重启后端服务使其生效）
-- 或者手动添加新菜单
INSERT INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status) VALUES
('执行任务', 'task-execute', 2, (SELECT id FROM sys_menu WHERE code = 'task' LIMIT 1), '/task/execute', 'task/Execute', 'VideoPlay', 1, 1, 1),
('文件分发', 'task-file-distribution', 2, (SELECT id FROM sys_menu WHERE code = 'task' LIMIT 1), '/task/file-distribution', 'task/FileDistribution', 'FolderOpened', 3, 1, 1)
ON DUPLICATE KEY UPDATE name=VALUES(name), path=VALUES(path), component=VALUES(component), icon=VALUES(icon);
