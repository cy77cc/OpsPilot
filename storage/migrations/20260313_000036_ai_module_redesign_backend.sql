-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_scene_configs (
  scene VARCHAR(128) PRIMARY KEY,
  name VARCHAR(128) NOT NULL DEFAULT '',
  description TEXT,
  constraints_json LONGTEXT,
  allowed_tools_json LONGTEXT,
  blocked_tools_json LONGTEXT,
  examples_json LONGTEXT,
  approval_config_json LONGTEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ai_approvals (
  id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL DEFAULT '',
  plan_id VARCHAR(64) NOT NULL DEFAULT '',
  step_id VARCHAR(64) NOT NULL DEFAULT '',
  checkpoint_id VARCHAR(128) NOT NULL DEFAULT '',
  approval_key VARCHAR(128) NOT NULL,
  request_user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  reviewer_user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  tool_name VARCHAR(128) NOT NULL,
  tool_display_name VARCHAR(128) NOT NULL DEFAULT '',
  tool_mode VARCHAR(32) NOT NULL DEFAULT 'readonly',
  risk_level VARCHAR(16) NOT NULL DEFAULT 'low',
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  scene VARCHAR(128) NOT NULL DEFAULT '',
  summary TEXT,
  reason VARCHAR(255) NOT NULL DEFAULT '',
  params_json LONGTEXT,
  preview_json LONGTEXT,
  execution_id VARCHAR(64) NOT NULL DEFAULT '',
  approved_at TIMESTAMP NULL,
  rejected_at TIMESTAMP NULL,
  expires_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_ai_approvals_key (approval_key),
  KEY idx_ai_approvals_session_plan_step (session_id, plan_id, step_id),
  KEY idx_ai_approvals_checkpoint (checkpoint_id),
  KEY idx_ai_approvals_status (status),
  KEY idx_ai_approvals_tool (tool_name),
  KEY idx_ai_approvals_user_created (request_user_id, created_at)
);

CREATE TABLE IF NOT EXISTS ai_executions (
  id VARCHAR(64) PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL DEFAULT '',
  plan_id VARCHAR(64) NOT NULL DEFAULT '',
  step_id VARCHAR(64) NOT NULL DEFAULT '',
  checkpoint_id VARCHAR(128) NOT NULL DEFAULT '',
  approval_id VARCHAR(64) NOT NULL DEFAULT '',
  request_user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  tool_name VARCHAR(128) NOT NULL,
  tool_mode VARCHAR(32) NOT NULL DEFAULT 'readonly',
  risk_level VARCHAR(16) NOT NULL DEFAULT 'low',
  scene VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  params_json LONGTEXT,
  result_json LONGTEXT,
  metadata_json LONGTEXT,
  error_message TEXT,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  prompt_tokens BIGINT NOT NULL DEFAULT 0,
  completion_tokens BIGINT NOT NULL DEFAULT 0,
  total_tokens BIGINT NOT NULL DEFAULT 0,
  estimated_cost_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
  started_at TIMESTAMP NULL,
  finished_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_ai_executions_session_plan_step (session_id, plan_id, step_id),
  KEY idx_ai_executions_checkpoint (checkpoint_id),
  KEY idx_ai_executions_approval (approval_id),
  KEY idx_ai_executions_status (status),
  KEY idx_ai_executions_tool (tool_name),
  KEY idx_ai_executions_user_created (request_user_id, created_at)
);

INSERT INTO ai_scene_configs (scene, name, description, constraints_json, allowed_tools_json, blocked_tools_json, examples_json, approval_config_json)
VALUES
  ('global', 'Global Assistant', 'Default cross-domain assistant scene', '["Prefer readonly investigation before mutating actions"]', '[]', '[]', '["Summarize the current issue","List recent operational changes"]', '{"default_policy":{"require_approval_for":["medium","high"]}}'),
  ('host', 'Host Operations', 'Host and node operational support', '["High-risk host changes require approval"]', '["host_list_inventory","host_exec","host_exec_by_target","host_batch_exec_preview","host_batch_exec_apply"]', '[]', '["Check host CPU and memory usage","Preview restarting nginx on selected hosts"]', '{"default_policy":{"require_approval_for":["medium","high"],"require_for_all_mutating":true}}'),
  ('service', 'Service Operations', 'Service status, deployment and runtime assistance', '["Preview deployments before applying them"]', '["service_status","service_status_by_target","service_deploy_preview","service_deploy_apply","service_catalog_list"]', '[]', '["Check service status by name","Preview a service deployment"]', '{"default_policy":{"require_approval_for":["medium","high"],"require_for_all_mutating":true}}')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  constraints_json = VALUES(constraints_json),
  allowed_tools_json = VALUES(allowed_tools_json),
  blocked_tools_json = VALUES(blocked_tools_json),
  examples_json = VALUES(examples_json),
  approval_config_json = VALUES(approval_config_json);

-- +migrate Down
DROP TABLE IF EXISTS ai_executions;
DROP TABLE IF EXISTS ai_approvals;
DROP TABLE IF EXISTS ai_scene_configs;
