-- +migrate Up
SET @tbl_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters'
);

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'source'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE clusters ADD COLUMN source VARCHAR(32) NOT NULL DEFAULT ''platform_managed'' AFTER type',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'credential_id'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE clusters ADD COLUMN credential_id BIGINT UNSIGNED NULL AFTER auth_method',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'k8s_version'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE clusters ADD COLUMN k8s_version VARCHAR(32) NOT NULL DEFAULT '''' AFTER credential_id',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'pod_cidr'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE clusters ADD COLUMN pod_cidr VARCHAR(32) NOT NULL DEFAULT '''' AFTER k8s_version',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'service_cidr'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE clusters ADD COLUMN service_cidr VARCHAR(32) NOT NULL DEFAULT '''' AFTER pod_cidr',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'last_sync_at'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists = 0,
  'ALTER TABLE clusters ADD COLUMN last_sync_at TIMESTAMP NULL DEFAULT NULL AFTER management_mode',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND INDEX_NAME = 'idx_cluster_source'
);
SET @sql := IF(@tbl_exists = 1 AND @idx_exists = 0,
  'CREATE INDEX idx_cluster_source ON clusters (source)',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND INDEX_NAME = 'idx_cluster_credential_id'
);
SET @sql := IF(@tbl_exists = 1 AND @idx_exists = 0,
  'CREATE INDEX idx_cluster_credential_id ON clusters (credential_id)',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +migrate Down
SET @idx_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND INDEX_NAME = 'idx_cluster_credential_id'
);
SET @sql := IF(@idx_exists > 0,
  'DROP INDEX idx_cluster_credential_id ON clusters',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND INDEX_NAME = 'idx_cluster_source'
);
SET @sql := IF(@idx_exists > 0,
  'DROP INDEX idx_cluster_source ON clusters',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @tbl_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters'
);
SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'last_sync_at'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists > 0,
  'ALTER TABLE clusters DROP COLUMN last_sync_at',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'service_cidr'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists > 0,
  'ALTER TABLE clusters DROP COLUMN service_cidr',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'pod_cidr'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists > 0,
  'ALTER TABLE clusters DROP COLUMN pod_cidr',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'k8s_version'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists > 0,
  'ALTER TABLE clusters DROP COLUMN k8s_version',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'credential_id'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists > 0,
  'ALTER TABLE clusters DROP COLUMN credential_id',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clusters' AND COLUMN_NAME = 'source'
);
SET @sql := IF(@tbl_exists = 1 AND @col_exists > 0,
  'ALTER TABLE clusters DROP COLUMN source',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
