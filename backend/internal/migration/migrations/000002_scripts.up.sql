CREATE TABLE IF NOT EXISTS `scripts` (
  `id` varchar(64) NOT NULL,
  `name` varchar(128) NOT NULL DEFAULT '',
  `version` varchar(32) NOT NULL DEFAULT '',
  `source_url` varchar(512) NOT NULL DEFAULT '',
  `manifest` json,
  `enabled` tinyint(1) NOT NULL DEFAULT 1,
  `status` varchar(32) NOT NULL DEFAULT 'installed',
  `installed_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
