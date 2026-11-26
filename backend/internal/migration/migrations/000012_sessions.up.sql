CREATE TABLE IF NOT EXISTS `sessions` (
  `id` varchar(32) NOT NULL,
  `user_id` varchar(32) NOT NULL,
  `token_hash` char(64) NOT NULL,
  `ip` varchar(64) NOT NULL DEFAULT '',
  `user_agent` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) NULL,
  `last_seen_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_sessions_token` (`token_hash`),
  KEY `idx_sessions_user` (`user_id`),
  KEY `idx_sessions_expires` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `login_attempts` (
  `id` varchar(32) NOT NULL,
  `username` varchar(64) NOT NULL,
  `ip` varchar(64) NOT NULL DEFAULT '',
  `user_agent` varchar(255) NOT NULL DEFAULT '',
  `success` tinyint(1) NOT NULL DEFAULT 0,
  `reason` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_login_attempts_user_time` (`username`, `created_at`),
  KEY `idx_login_attempts_time` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX `idx_tasks_user_created` ON `tasks` (`user_id`, `created_at`);
