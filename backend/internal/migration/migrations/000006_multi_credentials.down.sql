DROP TABLE IF EXISTS `platform_configs`;

ALTER TABLE `credential_access_logs` DROP COLUMN `credential_id`;

ALTER TABLE `tasks`
  DROP INDEX `idx_tasks_credential`,
  DROP COLUMN `credential_id`;

ALTER TABLE `platform_caches`
  DROP INDEX `uk_platform_cache`,
  DROP COLUMN `credential_id`,
  ADD UNIQUE KEY `uk_platform_cache` (`user_id`, `platform`);

ALTER TABLE `credentials`
  DROP INDEX `idx_credential_user_platform`,
  DROP COLUMN `archived_at`,
  DROP COLUMN `is_default`,
  DROP COLUMN `preview`,
  DROP COLUMN `label`,
  ADD UNIQUE KEY `uk_credential_user_platform` (`user_id`, `platform`);
