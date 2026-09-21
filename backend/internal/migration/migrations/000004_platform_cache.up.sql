CREATE TABLE IF NOT EXISTS `platform_caches` (
  `id` varchar(32) NOT NULL,
  `user_id` varchar(32) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `items` json,
  `refreshed_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_cache` (`user_id`, `platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
