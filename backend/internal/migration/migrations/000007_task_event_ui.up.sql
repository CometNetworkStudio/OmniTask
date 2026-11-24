ALTER TABLE `task_events`
  ADD COLUMN `kind` varchar(32) NOT NULL DEFAULT 'progress',
  ADD COLUMN `level` varchar(16) NULL,
  ADD COLUMN `step` varchar(64) NULL,
  ADD COLUMN `label` varchar(128) NULL,
  ADD COLUMN `step_status` varchar(16) NULL,
  ADD COLUMN `current` bigint NOT NULL DEFAULT 0,
  ADD COLUMN `total` bigint NOT NULL DEFAULT 0,
  ADD COLUMN `unit` varchar(16) NULL,
  ADD COLUMN `stage` varchar(64) NULL,
  ADD COLUMN `key` varchar(64) NULL,
  ADD COLUMN `value` varchar(255) NULL;
