-- +migrate Up
SET @add_client_request_id_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_runs'
        AND column_name = 'client_request_id'
    ),
    'SELECT 1',
    'ALTER TABLE ai_runs ADD COLUMN client_request_id VARCHAR(64) NOT NULL DEFAULT '''' AFTER session_id'
  )
);
PREPARE add_client_request_id_stmt FROM @add_client_request_id_sql;
EXECUTE add_client_request_id_stmt;
DEALLOCATE PREPARE add_client_request_id_stmt;

SET @add_last_event_at_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_runs'
        AND column_name = 'last_event_at'
    ),
    'SELECT 1',
    'ALTER TABLE ai_runs ADD COLUMN last_event_at DATETIME(3) NULL AFTER started_at'
  )
);
PREPARE add_last_event_at_stmt FROM @add_last_event_at_sql;
EXECUTE add_last_event_at_stmt;
DEALLOCATE PREPARE add_last_event_at_stmt;

UPDATE ai_runs
SET client_request_id = id
WHERE COALESCE(client_request_id, '') = '';

SET @add_session_request_index_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_runs'
        AND index_name = 'uk_ai_runs_session_request'
    ),
    'SELECT 1',
    'ALTER TABLE ai_runs ADD UNIQUE KEY uk_ai_runs_session_request (session_id, client_request_id)'
  )
);
PREPARE add_session_request_index_stmt FROM @add_session_request_index_sql;
EXECUTE add_session_request_index_stmt;
DEALLOCATE PREPARE add_session_request_index_stmt;

-- +migrate Down
SET @drop_session_request_index_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_runs'
        AND index_name = 'uk_ai_runs_session_request'
    ),
    'DROP INDEX uk_ai_runs_session_request ON ai_runs',
    'SELECT 1'
  )
);
PREPARE drop_session_request_index_stmt FROM @drop_session_request_index_sql;
EXECUTE drop_session_request_index_stmt;
DEALLOCATE PREPARE drop_session_request_index_stmt;

SET @drop_last_event_at_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_runs'
        AND column_name = 'last_event_at'
    ),
    'ALTER TABLE ai_runs DROP COLUMN last_event_at',
    'SELECT 1'
  )
);
PREPARE drop_last_event_at_stmt FROM @drop_last_event_at_sql;
EXECUTE drop_last_event_at_stmt;
DEALLOCATE PREPARE drop_last_event_at_stmt;

SET @drop_client_request_id_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_runs'
        AND column_name = 'client_request_id'
    ),
    'ALTER TABLE ai_runs DROP COLUMN client_request_id',
    'SELECT 1'
  )
);
PREPARE drop_client_request_id_stmt FROM @drop_client_request_id_sql;
EXECUTE drop_client_request_id_stmt;
DEALLOCATE PREPARE drop_client_request_id_stmt;
