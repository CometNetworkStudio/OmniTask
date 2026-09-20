CREATE TABLE IF NOT EXISTS `users` (
  `id` varchar(32) NOT NULL,
  `username` varchar(64) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `role` varchar(32) NOT NULL DEFAULT 'user',
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `credentials` (
  `id` varchar(32) NOT NULL,
  `user_id` varchar(32) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `cipher` blob NOT NULL,
  `nonce` blob NOT NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_credential_user_platform` (`user_id`, `platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `tasks` (
  `id` varchar(32) NOT NULL,
  `user_id` varchar(32) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `action` varchar(64) NOT NULL,
  `params` json,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `progress` bigint NOT NULL DEFAULT 0,
  `result` json,
  `error` text,
  `started_at` datetime(3) NULL,
  `finished_at` datetime(3) NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tasks_user_id` (`user_id`),
  KEY `idx_tasks_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `task_events` (
  `id` varchar(32) NOT NULL,
  `task_id` varchar(32) NOT NULL,
  `type` varchar(32) NOT NULL,
  `percent` bigint NOT NULL DEFAULT 0,
  `message` text,
  `created_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_task_events_task_id` (`task_id`),
  KEY `idx_task_events_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `questions` (
  `id` varchar(32) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `question_key` varchar(128) NOT NULL,
  `stem` text,
  `options` json,
  `answer` json,
  `source` varchar(16) NOT NULL,
  `confidence` double NOT NULL DEFAULT 0,
  `hit_count` bigint NOT NULL DEFAULT 0,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_question_platform_key` (`platform`, `question_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
