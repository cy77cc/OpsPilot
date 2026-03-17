-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_checkpoints (
  checkpoint_id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NULL,
  run_id VARCHAR(64) NULL,
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  scene VARCHAR(32) NULL,
  payload LONGBLOB NOT NULL,
  expires_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ai_checkpoints_session_id (session_id),
  INDEX idx_ai_checkpoints_run_id (run_id),
  INDEX idx_ai_checkpoints_user_id (user_id),
  INDEX idx_ai_checkpoints_scene (scene),
  INDEX idx_ai_checkpoints_expires_at (expires_at),
  INDEX idx_ai_checkpoints_deleted_at (deleted_at)
);

-- +migrate Down
DROP TABLE IF EXISTS ai_checkpoints;
