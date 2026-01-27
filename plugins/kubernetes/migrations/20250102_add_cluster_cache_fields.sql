-- Copyright (c) 2026 YDCloud
--
-- Permission is hereby granted, free of charge, to any person obtaining a copy of
-- this software and associated documentation files (the "Software"), to deal in
-- the Software without restriction, including without limitation the rights to
-- use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
-- the Software, and to permit persons to whom the Software is furnished to do so,
-- subject to the following conditions:
--
-- The above copyright notice and this permission notice shall be included in all
-- copies or substantial portions of the Software.
--
-- THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
-- IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
-- FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
-- COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
-- IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
-- CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

-- 添加集群缓存字段，用于存储同步后的集群状态信息
-- 这样列表查询可以直接从数据库读取，无需实时连接 K8s API

-- 添加节点数量缓存字段
ALTER TABLE `k8s_clusters`
ADD COLUMN `node_count` INT DEFAULT 0 COMMENT '节点数量（缓存）' AFTER `description`;

-- 添加状态同步时间字段
ALTER TABLE `k8s_clusters`
ADD COLUMN `status_synced_at` TIMESTAMP NULL DEFAULT NULL COMMENT '状态最后同步时间' AFTER `node_count`;

-- 添加 Pod 数量缓存字段
ALTER TABLE `k8s_clusters`
ADD COLUMN `pod_count` INT DEFAULT 0 COMMENT 'Pod数量（缓存）' AFTER `node_count`;

-- 为已有数据初始化默认值
UPDATE `k8s_clusters` SET `node_count` = 0, `pod_count` = 0 WHERE `node_count` IS NULL;
