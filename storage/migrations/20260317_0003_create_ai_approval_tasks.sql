-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_approval_tasks (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  approval_id VARCHAR(64) NOT NULL,
  checkpoint_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  run_id VARCHAR(64) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  tool_name VARCHAR(64) NOT NULL,
  tool_call_id VARCHAR(64) NOT NULL,
  arguments_json LONGTEXT NOT NULL,
  preview_json LONGTEXT NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  approved_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  disapprove_reason TEXT NULL,
  comment TEXT NULL,
  timeout_seconds INT NOT NULL DEFAULT 300,
  expires_at DATETIME(3) NULL,
  decided_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE INDEX uk_ai_approval_tasks_approval_id (approval_id),
  INDEX idx_ai_approval_tasks_checkpoint_id (checkpoint_id),
  INDEX idx_ai_approval_tasks_session_id (session_id),
  INDEX idx_ai_approval_tasks_run_id (run_id),
  INDEX idx_ai_approval_tasks_user_id (user_id),
  INDEX idx_ai_approval_tasks_status (status),
  INDEX idx_ai_approval_tasks_expires_at (expires_at),
  INDEX idx_ai_approval_tasks_created_at (created_at),
  INDEX idx_ai_approval_tasks_deleted_at (deleted_at)
);

-- +migrate Down
DROP TABLE IF EXISTS ai_approval_tasks;
