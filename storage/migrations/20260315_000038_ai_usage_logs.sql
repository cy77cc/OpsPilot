-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_usage_logs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  trace_id VARCHAR(36) NOT NULL,
  session_id VARCHAR(36) NOT NULL,
  plan_id VARCHAR(36) NOT NULL,
  turn_id VARCHAR(36) DEFAULT '',
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,

  scene VARCHAR(64) NOT NULL DEFAULT '',
  operation VARCHAR(32) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT '',

  prompt_tokens INT NOT NULL DEFAULT 0,
  completion_tokens INT NOT NULL DEFAULT 0,
  total_tokens INT NOT NULL DEFAULT 0,
  estimated_cost_usd DECIMAL(10,6) NOT NULL DEFAULT 0,
  model_name VARCHAR(128) NOT NULL DEFAULT '',

  duration_ms INT NOT NULL DEFAULT 0,
  first_token_ms INT NOT NULL DEFAULT 0,
  tokens_per_second DECIMAL(10,2) NOT NULL DEFAULT 0,

  approval_count INT NOT NULL DEFAULT 0,
  approval_status VARCHAR(32) NOT NULL DEFAULT 'none',
  approval_wait_ms INT NOT NULL DEFAULT 0,

  tool_call_count INT NOT NULL DEFAULT 0,
  tool_error_count INT NOT NULL DEFAULT 0,

  error_type VARCHAR(64) NOT NULL DEFAULT '',
  error_message TEXT,

  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  KEY idx_ai_usage_logs_session_id (session_id),
  KEY idx_ai_usage_logs_user_created (user_id, created_at),
  KEY idx_ai_usage_logs_scene (scene),
  KEY idx_ai_usage_logs_status (status),
  KEY idx_ai_usage_logs_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 请求级使用日志汇总表';

-- +migrate Down
DROP TABLE IF EXISTS ai_usage_logs;
