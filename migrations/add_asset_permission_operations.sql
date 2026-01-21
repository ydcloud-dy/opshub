-- 添加操作权限字段
ALTER TABLE sys_role_asset_permission
ADD COLUMN permissions INT UNSIGNED DEFAULT 1
COMMENT '操作权限位掩码：1=查看,2=编辑,4=删除,8=终端,16=文件,32=采集';

-- 为现有权限记录设置默认值（仅查看权限）
UPDATE sys_role_asset_permission
SET permissions = 1
WHERE permissions IS NULL OR permissions = 0;

-- 添加索引优化查询
CREATE INDEX idx_permissions ON sys_role_asset_permission(permissions);
