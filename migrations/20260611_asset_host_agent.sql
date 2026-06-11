-- Add Go Agent collection metadata to asset hosts.

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_id'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_id` varchar(80) NULL COMMENT ''Agent唯一标识'' AFTER `last_seen`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_version'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_version` varchar(50) NULL COMMENT ''Agent版本'' AFTER `agent_id`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_status'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_status` varchar(20) NULL COMMENT ''Agent状态 online/offline/pending'' AFTER `agent_version`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_last_seen'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_last_seen` datetime NULL COMMENT ''Agent最后心跳时间'' AFTER `agent_status`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_last_collect_at'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_last_collect_at` datetime NULL COMMENT ''Agent最后采集时间'' AFTER `agent_last_seen`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_token_hash'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_token_hash` varchar(128) NULL COMMENT ''Agent认证Token哈希'' AFTER `agent_last_collect_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_install_token_hash'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_install_token_hash` varchar(128) NULL COMMENT ''Agent安装注册码哈希'' AFTER `agent_token_hash`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND COLUMN_NAME = 'agent_install_token_expires_at'
);
SET @ddl := IF(@column_exists = 0, 'ALTER TABLE `hosts` ADD COLUMN `agent_install_token_expires_at` datetime NULL COMMENT ''Agent安装注册码过期时间'' AFTER `agent_install_token_hash`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists := (
  SELECT COUNT(1)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND INDEX_NAME = 'idx_hosts_agent_id'
);
SET @ddl := IF(@index_exists = 0, 'ALTER TABLE `hosts` ADD INDEX `idx_hosts_agent_id` (`agent_id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists := (
  SELECT COUNT(1)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hosts'
    AND INDEX_NAME = 'idx_hosts_agent_install_token_hash'
);
SET @ddl := IF(@index_exists = 0, 'ALTER TABLE `hosts` ADD INDEX `idx_hosts_agent_install_token_hash` (`agent_install_token_hash`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
