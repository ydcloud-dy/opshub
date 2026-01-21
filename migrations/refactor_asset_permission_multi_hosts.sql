-- 重构资产权限表以支持单条记录存储多个主机
-- 此迁移文件用于将原有的一条主机一条记录的设计改为单条记录存储多个主机

-- 第一步：添加新字段 host_ids (JSON格式) 用于存储多个主机ID
ALTER TABLE sys_role_asset_permission
ADD COLUMN host_ids JSON COMMENT '指定的主机ID列表（JSON格式）' AFTER host_id;

-- 第二步：迁移现有数据
-- 如果host_id不为NULL，则将其迁移到host_ids
UPDATE sys_role_asset_permission
SET host_ids = JSON_ARRAY(host_id)
WHERE host_id IS NOT NULL;

-- 第三步：删除旧的host_id字段
ALTER TABLE sys_role_asset_permission
DROP COLUMN host_id;

-- 第四步：添加唯一索引，确保同一角色对同一分组只有一条记录
ALTER TABLE sys_role_asset_permission
ADD UNIQUE INDEX idx_unique_role_group ON sys_role_asset_permission(role_id, asset_group_id, deleted_at);

-- 第五步：删除旧的索引
DROP INDEX idx_host ON sys_role_asset_permission;
