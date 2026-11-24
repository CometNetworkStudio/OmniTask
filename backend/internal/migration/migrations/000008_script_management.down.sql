DROP TABLE IF EXISTS `script_install_jobs`;

ALTER TABLE `scripts`
  DROP COLUMN `ref`,
  DROP COLUMN `commit_sha`,
  DROP COLUMN `manifest_hash`,
  DROP COLUMN `last_error`;
