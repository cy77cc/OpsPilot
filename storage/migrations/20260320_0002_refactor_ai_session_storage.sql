-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_run_events (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  seq INT NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  agent_name VARCHAR(64) DEFAULT '',
  tool_call_id VARCHAR(64) DEFAULT '',
  payload_json LONGTEXT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ai_run_events_run_seq (run_id, seq),
  INDEX idx_ai_run_events_session_created (session_id, created_at),
  INDEX idx_ai_run_events_tool_call_id (tool_call_id),
  INDEX idx_ai_run_events_run_type (run_id, event_type)
);

CREATE TABLE IF NOT EXISTS ai_run_projections (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  version INT NOT NULL DEFAULT 1,
  status VARCHAR(32) NOT NULL,
  projection_json LONGTEXT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ai_run_projections_run_id (run_id),
  INDEX idx_ai_run_projections_session_id (session_id)
);

CREATE TABLE IF NOT EXISTS ai_run_contents (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  content_kind VARCHAR(32) NOT NULL,
  encoding VARCHAR(16) NOT NULL,
  summary_text VARCHAR(500) DEFAULT '',
  body_text LONGTEXT NULL,
  body_json LONGTEXT NULL,
  size_bytes BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_ai_run_contents_run_id (run_id),
  INDEX idx_ai_run_contents_session_id (session_id),
  INDEX idx_ai_run_contents_kind (content_kind)
);

SET @drop_runtime_json_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_chat_messages'
        AND column_name = 'runtime_json'
    ),
    'ALTER TABLE ai_chat_messages DROP COLUMN runtime_json',
    'SELECT 1'
  )
);
PREPARE drop_runtime_json_stmt FROM @drop_runtime_json_sql;
EXECUTE drop_runtime_json_stmt;
DEALLOCATE PREPARE drop_runtime_json_stmt;

-- +migrate Down
SET @add_runtime_json_sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'ai_chat_messages'
        AND column_name = 'runtime_json'
    ),
    'SELECT 1',
    'ALTER TABLE ai_chat_messages ADD COLUMN runtime_json LONGTEXT NULL AFTER status'
  )
);
PREPARE add_runtime_json_stmt FROM @add_runtime_json_sql;
EXECUTE add_runtime_json_stmt;
DEALLOCATE PREPARE add_runtime_json_stmt;
DROP TABLE IF EXISTS ai_run_contents;
DROP TABLE IF EXISTS ai_run_projections;
DROP TABLE IF EXISTS ai_run_events;
