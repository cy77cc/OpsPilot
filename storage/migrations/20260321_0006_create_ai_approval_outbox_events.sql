-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_approval_outbox_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  approval_id VARCHAR(64) NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  payload_json LONGTEXT NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  retry_count INT NOT NULL DEFAULT 0,
  next_retry_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_ai_approval_outbox_events_approval_event (approval_id, event_type),
  KEY idx_ai_approval_outbox_events_status_next_retry_created (status, next_retry_at, created_at),
  KEY idx_ai_approval_outbox_events_run_id (run_id),
  KEY idx_ai_approval_outbox_events_session_id (session_id)
);

-- +migrate Down
DROP TABLE IF EXISTS ai_approval_outbox_events;
