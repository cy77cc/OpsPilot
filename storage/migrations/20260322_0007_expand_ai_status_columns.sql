-- +migrate Up
ALTER TABLE ai_runs
  MODIFY COLUMN status VARCHAR(32) NOT NULL DEFAULT 'running';

ALTER TABLE ai_trace_spans
  MODIFY COLUMN status VARCHAR(32) NULL;

ALTER TABLE ai_usage_logs
  MODIFY COLUMN status VARCHAR(32) NULL;

-- +migrate Down
UPDATE ai_runs SET status = LEFT(status, 16) WHERE CHAR_LENGTH(status) > 16;
UPDATE ai_trace_spans SET status = LEFT(status, 16) WHERE status IS NOT NULL AND CHAR_LENGTH(status) > 16;
UPDATE ai_usage_logs SET status = LEFT(status, 16) WHERE status IS NOT NULL AND CHAR_LENGTH(status) > 16;

ALTER TABLE ai_runs
  MODIFY COLUMN status VARCHAR(16) NOT NULL DEFAULT 'running';

ALTER TABLE ai_trace_spans
  MODIFY COLUMN status VARCHAR(16) NULL;

ALTER TABLE ai_usage_logs
  MODIFY COLUMN status VARCHAR(16) NULL;
