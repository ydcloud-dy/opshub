-- 提升 AI 助手长回答的默认输出长度和模型等待时间。
ALTER TABLE `ai_providers`
  MODIFY COLUMN `max_tokens` int DEFAULT 8192 COMMENT '最大输出Token',
  MODIFY COLUMN `timeout` int DEFAULT 180 COMMENT '超时时间秒';

UPDATE `ai_providers`
SET
  `max_tokens` = CASE WHEN `max_tokens` IS NULL OR `max_tokens` IN (0, 2048) THEN 8192 ELSE `max_tokens` END,
  `timeout` = CASE WHEN `timeout` IS NULL OR `timeout` IN (0, 60) THEN 180 ELSE `timeout` END
WHERE `deleted_at` IS NULL;
