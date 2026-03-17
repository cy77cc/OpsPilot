-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_chat_sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  scene VARCHAR(32) NOT NULL DEFAULT 'ai',
  title VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ai_chat_sessions_user_scene_updated (user_id, scene, updated_at),
  INDEX idx_ai_chat_sessions_user_id (user_id),
  INDEX idx_ai_chat_sessions_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_chat_messages (
  id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  session_id_num INT NOT NULL DEFAULT 0,
  role VARCHAR(16) NOT NULL DEFAULT 'assistant',
  content LONGTEXT NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'done',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_ai_chat_messages_session_seq (session_id, session_id_num),
  INDEX idx_ai_chat_messages_session_created (session_id, created_at),
  INDEX idx_ai_chat_messages_session_role (session_id, role),
  INDEX idx_ai_chat_messages_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_runs (
  id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  user_message_id VARCHAR(64) NOT NULL,
  assistant_message_id VARCHAR(64) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'running',
  assistant_type VARCHAR(64) NULL,
  intent_type VARCHAR(32) NULL,
  progress_summary TEXT NULL,
  risk_level VARCHAR(16) NULL,
  trace_id VARCHAR(128) NULL,
  error_message TEXT NULL,
  trace_json LONGTEXT NOT NULL,
  started_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  finished_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ai_runs_session_id (session_id),
  INDEX idx_ai_runs_user_message_id (user_message_id),
  INDEX idx_ai_runs_assistant_message_id (assistant_message_id),
  INDEX idx_ai_runs_status_created (status, created_at),
  INDEX idx_ai_runs_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_diagnosis_reports (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  summary TEXT NOT NULL,
  report_json LONGTEXT NULL,
  evidence_json LONGTEXT NULL,
  root_causes_json LONGTEXT NULL,
  recommendations_json LONGTEXT NULL,
  risk_level VARCHAR(16) NULL,
  generated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_ai_diagnosis_reports_run_id (run_id),
  INDEX idx_ai_diagnosis_reports_session_created (session_id, created_at),
  INDEX idx_ai_diagnosis_reports_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_scene_prompts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  scene VARCHAR(32) NOT NULL,
  prompt_text TEXT NOT NULL,
  display_order INT NOT NULL DEFAULT 0,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ai_scene_prompts_scene_active_order (scene, is_active, display_order),
  INDEX idx_ai_scene_prompts_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_scene_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  scene VARCHAR(32) NOT NULL,
  description TEXT NULL,
  constraints_json LONGTEXT NULL,
  allowed_tools_json LONGTEXT NULL,
  blocked_tools_json LONGTEXT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_ai_scene_configs_scene (scene),
  INDEX idx_ai_scene_configs_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_trace_spans (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NULL,
  session_id VARCHAR(64) NULL,
  scene VARCHAR(32) NULL,
  status VARCHAR(16) NULL,
  model_name VARCHAR(128) NULL,
  tokens BIGINT NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  start_time DATETIME(3) NOT NULL,
  end_time DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ai_trace_spans_run_id (run_id),
  INDEX idx_ai_trace_spans_session_id (session_id),
  INDEX idx_ai_trace_spans_scene (scene),
  INDEX idx_ai_trace_spans_status (status),
  INDEX idx_ai_trace_spans_start_time (start_time),
  INDEX idx_ai_trace_spans_deleted_at (deleted_at)
);

CREATE TABLE IF NOT EXISTS ai_usage_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id VARCHAR(64) NULL,
  session_id VARCHAR(64) NULL,
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  scene VARCHAR(32) NULL,
  status VARCHAR(16) NULL,
  prompt_tokens BIGINT NOT NULL DEFAULT 0,
  completion_tokens BIGINT NOT NULL DEFAULT 0,
  total_tokens BIGINT NOT NULL DEFAULT 0,
  estimated_cost_usd DECIMAL(12,6) NOT NULL DEFAULT 0,
  first_token_ms BIGINT NOT NULL DEFAULT 0,
  tokens_per_second DECIMAL(12,4) NOT NULL DEFAULT 0,
  approval_count BIGINT NOT NULL DEFAULT 0,
  approval_status VARCHAR(16) NOT NULL DEFAULT '',
  tool_call_count BIGINT NOT NULL DEFAULT 0,
  tool_error_count BIGINT NOT NULL DEFAULT 0,
  metadata_json LONGTEXT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ai_usage_logs_run_id (run_id),
  INDEX idx_ai_usage_logs_session_id (session_id),
  INDEX idx_ai_usage_logs_user_id (user_id),
  INDEX idx_ai_usage_logs_scene (scene),
  INDEX idx_ai_usage_logs_status (status),
  INDEX idx_ai_usage_logs_created_at (created_at),
  INDEX idx_ai_usage_logs_deleted_at (deleted_at)
);

-- +migrate Down
DROP TABLE IF EXISTS ai_usage_logs;
DROP TABLE IF EXISTS ai_trace_spans;
DROP TABLE IF EXISTS ai_scene_configs;
DROP TABLE IF EXISTS ai_scene_prompts;
DROP TABLE IF EXISTS ai_diagnosis_reports;
DROP TABLE IF EXISTS ai_runs;
DROP TABLE IF EXISTS ai_chat_messages;
DROP TABLE IF EXISTS ai_chat_sessions;
