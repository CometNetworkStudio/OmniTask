CREATE TABLE IF NOT EXISTS `user_preferences` (
  `user_id` varchar(32) NOT NULL,
  `pinned_platforms` text NOT NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
