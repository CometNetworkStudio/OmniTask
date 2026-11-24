ALTER TABLE `credentials`
  ADD COLUMN `key_version` int NOT NULL DEFAULT 1,
  ADD COLUMN `last_used_at` datetime(3) NULL;
