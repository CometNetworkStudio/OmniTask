ALTER TABLE `credentials`
  DROP INDEX `uk_credential_user_platform`,
  ADD INDEX `idx_credential_user_platform` (`user_id`, `platform`),
  ADD COLUMN `label` varchar(64) NULL,
  ADD COLUMN `preview` varchar(128) NULL,
  ADD COLUMN `is_default` tinyint(1) NOT NULL DEFAULT 0,
  ADD COLUMN `archived_at` datetime(3) NULL;

ALTER TABLE `platform_caches`
  DROP INDEX `uk_platform_cache`,
  ADD COLUMN `credential_id` varchar(32) NOT NULL DEFAULT '',
  ADD UNIQUE KEY `uk_platform_cache` (`user_id`, `platform`, `credential_id`);

ALTER TABLE `tasks`
  ADD COLUMN `credential_id` varchar(32) NULL,
  ADD INDEX `idx_tasks_credential` (`credential_id`);

ALTER TABLE `credential_access_logs`
  ADD COLUMN `credential_id` varchar(32) NULL;

CREATE TABLE IF NOT EXISTS `platform_configs` (
  `platform_id` varchar(64) NOT NULL,
  `max_credentials` int NOT NULL DEFAULT 3,
  `default_credential_type` varchar(16) NULL,
  `preview_fields` json,
  `token_head_len` int NOT NULL DEFAULT 6,
  `token_tail_len` int NOT NULL DEFAULT 4,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`platform_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
