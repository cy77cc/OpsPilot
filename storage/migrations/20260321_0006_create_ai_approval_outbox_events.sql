-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_approval_outbox_events (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  approval_id VARCHAR(64) NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  payload_json LONGTEXT NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  retry_count INT NOT NULL DEFAULT 0,
  next_retry_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE INDEX uk_ai_approval_outbox_events_approval_event (approval_id, event_type),
  INDEX idx_ai_approval_outbox_events_queue (status, next_retry_at, created_at),
  INDEX idx_ai_approval_outbox_events_run_id (run_id),
  INDEX idx_ai_approval_outbox_events_session_id (session_id)
);

-- +migrate Down
DROP TABLE IF EXISTS ai_approval_outbox_events;
