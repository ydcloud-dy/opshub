-- 重构资产权限表以支持单条记录存储多个主机
-- 此迁移文件用于将原有的一条主机一条记录的设计改为单条记录存储多个主机

-- 注意：host_ids 字段应该已经存在于表中，此文件主要用于添加唯一索引约束

-- 步骤1：清理重复数据（如果存在）
-- 每个 role_id + asset_group_id 组合只保留一条记录
CREATE TEMPORARY TABLE temp_max_ids AS
SELECT MAX(id) as max_id FROM sys_role_asset_permission
GROUP BY role_id, asset_group_id;

DELETE FROM sys_role_asset_permission
WHERE id NOT IN (SELECT max_id FROM temp_max_ids);

-- 步骤2：添加唯一索引确保同一角色对同一分组只有一条记录
ALTER TABLE sys_role_asset_permission
ADD CONSTRAINT idx_unique_role_group UNIQUE (role_id, asset_group_id);

-- 步骤3：验证迁移成功
-- 表结构应该包含：id, created_at, updated_at, deleted_at, role_id, asset_group_id, host_ids(JSON), permissions


