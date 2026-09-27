CREATE TABLE IF NOT EXISTS `credential_access_logs` (
  `id` varchar(32) NOT NULL,
  `user_id` varchar(32) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `purpose` varchar(16) NOT NULL,
  `ref_id` varchar(32) NULL,
  `created_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cal_user_created` (`user_id`, `created_at`),
  KEY `idx_cal_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
