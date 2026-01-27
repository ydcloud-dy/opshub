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

-- 用户K8s角色绑定表
CREATE TABLE IF NOT EXISTS `k8s_user_role_bindings` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `cluster_id` BIGINT UNSIGNED NOT NULL COMMENT '集群ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '平台用户ID',
  `role_name` VARCHAR(255) NOT NULL COMMENT 'K8s角色名称',
  `role_namespace` VARCHAR(255) DEFAULT '' COMMENT '角色命名空间(空表示集群角色)',
  `role_type` VARCHAR(50) NOT NULL COMMENT '角色类型: ClusterRole/Role',
  `bound_by` BIGINT UNSIGNED NOT NULL COMMENT '操作人ID',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY `uk_cluster_user_role` (`cluster_id`, `user_id`, `role_name`, `role_namespace`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_role` (`cluster_id`, `role_name`, `role_namespace`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户K8s角色绑定表';
