-- 为 OpenAI-compatible 推理模型增加可选推理强度配置。
ALTER TABLE `ai_providers`
  ADD COLUMN `reasoning_effort` varchar(20) DEFAULT NULL COMMENT '推理强度' AFTER `timeout`;
