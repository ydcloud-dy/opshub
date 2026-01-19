-- 添加主机类型字段
ALTER TABLE `hosts` ADD COLUMN `type` VARCHAR(20) NOT NULL DEFAULT 'self' COMMENT '主机类型 self:自建 cloud:云主机' AFTER `group_id`;
ALTER TABLE `hosts` ADD COLUMN `cloud_provider` VARCHAR(50) COMMENT '云厂商 aliyun/tencent/aws' AFTER `type`;
ALTER TABLE `hosts` ADD COLUMN `cloud_instance_id` VARCHAR(100) COMMENT '云实例ID' AFTER `cloud_provider`;
ALTER TABLE `hosts` ADD COLUMN `cloud_account_id` INT UNSIGNED COMMENT '云账号ID' AFTER `cloud_instance_id`;
