ALTER TABLE `users`
  ADD COLUMN `terms_version` varchar(32) NOT NULL DEFAULT '' AFTER `enabled`,
  ADD COLUMN `terms_accepted_at` datetime(3) NULL AFTER `terms_version`;
