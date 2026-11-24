ALTER TABLE `scripts`
  ADD COLUMN `ref` varchar(128) NOT NULL DEFAULT '',
  ADD COLUMN `commit_sha` varchar(64) NOT NULL DEFAULT '',
  ADD COLUMN `manifest_hash` varchar(64) NOT NULL DEFAULT '',
  ADD COLUMN `last_error` text;

CREATE TABLE IF NOT EXISTS `script_install_jobs` (
  `id` varchar(32) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'running',
  `stage` varchar(32) NOT NULL DEFAULT 'queued',
  `percent` bigint NOT NULL DEFAULT 0,
  `message` text,
  `error` text,
  `script_id` varchar(64) NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_script_install_jobs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
