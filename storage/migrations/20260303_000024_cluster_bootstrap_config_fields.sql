-- +migrate Up
SET @tbl_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks'
);

-- baseline columns compatible with 20260226_000010
SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'cluster_id'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN cluster_id BIGINT UNSIGNED NULL',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND INDEX_NAME = 'idx_cluster_bootstrap_cluster_id'
);
SET @sql := IF(@tbl_exists = 1 AND @idx_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD INDEX idx_cluster_bootstrap_cluster_id (cluster_id)',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'k8s_version'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN k8s_version VARCHAR(32) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'pod_cidr'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN pod_cidr VARCHAR(32) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'service_cidr'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN service_cidr VARCHAR(32) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'steps_json'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN steps_json LONGTEXT',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- config/diagnostic columns
SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'version_channel'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN version_channel VARCHAR(32) NOT NULL DEFAULT ''stable-1''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'repo_mode'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN repo_mode VARCHAR(16) NOT NULL DEFAULT ''online''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'repo_url'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN repo_url VARCHAR(512) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'image_repository'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN image_repository VARCHAR(256) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'endpoint_mode'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN endpoint_mode VARCHAR(16) NOT NULL DEFAULT ''nodeIP''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'control_plane_endpoint'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN control_plane_endpoint VARCHAR(256) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'vip_provider'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN vip_provider VARCHAR(32) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'etcd_mode'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN etcd_mode VARCHAR(16) NOT NULL DEFAULT ''stacked''',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'external_etcd_json'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN external_etcd_json LONGTEXT',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'resolved_config_json'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN resolved_config_json LONGTEXT',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = 'diagnostics_json'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE cluster_bootstrap_tasks ADD COLUMN diagnostics_json LONGTEXT',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +migrate Down
SET @tbl_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks'
);

-- drop index if exists
SET @idx_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND INDEX_NAME = 'idx_cluster_bootstrap_cluster_id'
);
SET @sql := IF(@tbl_exists = 1 AND @idx_exists = 1,
  'ALTER TABLE cluster_bootstrap_tasks DROP INDEX idx_cluster_bootstrap_cluster_id',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- drop columns safely one by one
SET @drop_col := 'diagnostics_json';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'resolved_config_json';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'external_etcd_json';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'etcd_mode';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'vip_provider';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'control_plane_endpoint';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'endpoint_mode';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'image_repository';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'repo_url';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'repo_mode';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'version_channel';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'steps_json';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'service_cidr';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'pod_cidr';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'k8s_version';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_col := 'cluster_id';
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cluster_bootstrap_tasks' AND COLUMN_NAME = @drop_col);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 1, CONCAT('ALTER TABLE cluster_bootstrap_tasks DROP COLUMN ', @drop_col), 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
