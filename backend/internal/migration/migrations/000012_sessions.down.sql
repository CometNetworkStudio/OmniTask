DROP TABLE IF EXISTS `sessions`;
DROP TABLE IF EXISTS `login_attempts`;
ALTER TABLE `tasks` DROP INDEX `idx_tasks_user_created`;
