-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_runs (
  id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  user_message_id VARCHAR(64) NOT NULL,
  assistant_message_id VARCHAR(64) DEFAULT NULL,
  intent_type VARCHAR(32) NOT NULL DEFAULT '',
  assistant_type VARCHAR(32) NOT NULL DEFAULT '',
  risk_level VARCHAR(16) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT '',
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  error_message TEXT NOT NULL,
  started_at DATETIME(6),
  finished_at DATETIME(6),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ai_runs_status (status),
  KEY idx_ai_runs_session_status (session_id, status),
  KEY idx_ai_runs_trace_id (trace_id),
  KEY idx_ai_runs_user_message_id (user_message_id),
  KEY idx_ai_runs_assistant_message_id (assistant_message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Phase 1 AI run lifecycle table';

CREATE TABLE IF NOT EXISTS ai_diagnosis_reports (
  id VARCHAR(64) NOT NULL,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  summary LONGTEXT NOT NULL,
  impact_scope LONGTEXT NOT NULL,
  suspected_root_causes LONGTEXT NOT NULL,
  evidence LONGTEXT NOT NULL,
  recommendations LONGTEXT NOT NULL,
  raw_tool_refs LONGTEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ai_diagnosis_reports_run_id (run_id),
  KEY idx_ai_diagnosis_reports_session_id (session_id),
  KEY idx_ai_diagnosis_reports_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Phase 1 AI diagnosis reports table';

-- +migrate Down
DROP TABLE IF EXISTS ai_diagnosis_reports;
DROP TABLE IF EXISTS ai_runs;
