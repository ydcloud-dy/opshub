-- Cloud hosts are collected only through SSH. Clear stale Agent bindings/tokens
-- for historical cloud assets so the UI and backend rules stay consistent.

UPDATE `hosts`
SET
  `agent_id` = '',
  `agent_version` = '',
  `agent_status` = '',
  `agent_last_seen` = NULL,
  `agent_last_collect_at` = NULL,
  `agent_token_hash` = '',
  `agent_install_token_hash` = '',
  `agent_install_token_expires_at` = NULL
WHERE
  `type` = 'cloud'
  OR COALESCE(`cloud_provider`, '') <> ''
  OR COALESCE(`cloud_instance_id`, '') <> ''
  OR COALESCE(`cloud_account_id`, 0) > 0;
