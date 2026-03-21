-- +migrate Up
SET @ai_run_contents_exists = (
  SELECT COUNT(1)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'ai_run_contents'
);

SET @ai_run_contents_charset_sql = (
  SELECT IF(
    @ai_run_contents_exists > 0,
    'ALTER TABLE ai_run_contents CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci',
    'SELECT 1'
  )
);

PREPARE ai_run_contents_charset_stmt FROM @ai_run_contents_charset_sql;
EXECUTE ai_run_contents_charset_stmt;
DEALLOCATE PREPARE ai_run_contents_charset_stmt;

-- +migrate Down
SET @ai_run_contents_exists = (
  SELECT COUNT(1)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'ai_run_contents'
);

SET @ai_run_contents_charset_rollback_sql = (
  SELECT IF(
    @ai_run_contents_exists > 0,
    'ALTER TABLE ai_run_contents CONVERT TO CHARACTER SET utf8 COLLATE utf8_general_ci',
    'SELECT 1'
  )
);

PREPARE ai_run_contents_charset_rollback_stmt FROM @ai_run_contents_charset_rollback_sql;
EXECUTE ai_run_contents_charset_rollback_stmt;
DEALLOCATE PREPARE ai_run_contents_charset_rollback_stmt;
