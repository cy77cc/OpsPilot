ALTER TABLE ai_runs
  ADD COLUMN client_request_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN last_event_at DATETIME NULL,
  ADD UNIQUE KEY uk_ai_runs_session_request (session_id, client_request_id);

UPDATE ai_runs
SET client_request_id = id
WHERE client_request_id = '';
