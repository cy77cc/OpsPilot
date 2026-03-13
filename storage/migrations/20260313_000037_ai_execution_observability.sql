-- +migrate Up
ALTER TABLE ai_executions
  ADD COLUMN IF NOT EXISTS metadata_json LONGTEXT AFTER result_json,
  ADD COLUMN IF NOT EXISTS duration_ms BIGINT NOT NULL DEFAULT 0 AFTER error_message,
  ADD COLUMN IF NOT EXISTS prompt_tokens BIGINT NOT NULL DEFAULT 0 AFTER duration_ms,
  ADD COLUMN IF NOT EXISTS completion_tokens BIGINT NOT NULL DEFAULT 0 AFTER prompt_tokens,
  ADD COLUMN IF NOT EXISTS total_tokens BIGINT NOT NULL DEFAULT 0 AFTER completion_tokens,
  ADD COLUMN IF NOT EXISTS estimated_cost_usd DECIMAL(20,8) NOT NULL DEFAULT 0 AFTER total_tokens;

-- +migrate Down
ALTER TABLE ai_executions
  DROP COLUMN estimated_cost_usd,
  DROP COLUMN total_tokens,
  DROP COLUMN completion_tokens,
  DROP COLUMN prompt_tokens,
  DROP COLUMN duration_ms,
  DROP COLUMN metadata_json;
