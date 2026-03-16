-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_runs (
  id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  user_message_id VARCHAR(64) NOT NULL,
  assistant_message_id VARCHAR(64) NOT NULL DEFAULT '',
  intent_type VARCHAR(32) NOT NULL DEFAULT '',
  assistant_type VARCHAR(32) NOT NULL DEFAULT '',
  risk_level VARCHAR(32) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT '',
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  error_message LONGTEXT,
  progress_summary LONGTEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_ai_runs_session_id (session_id),
  INDEX idx_ai_runs_status (status)
);

CREATE TABLE IF NOT EXISTS ai_diagnosis_reports (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  summary LONGTEXT,
  evidence_json LONGTEXT,
  root_causes_json LONGTEXT,
  recommendations_json LONGTEXT,
  generated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_ai_diagnosis_reports_run_id (run_id),
  INDEX idx_ai_diagnosis_reports_session_id (session_id)
);

-- +migrate Down
DROP TABLE IF EXISTS ai_diagnosis_reports;
DROP TABLE IF EXISTS ai_runs;
