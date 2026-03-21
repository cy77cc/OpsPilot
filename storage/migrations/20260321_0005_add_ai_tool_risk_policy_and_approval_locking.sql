-- +migrate Up
CREATE TABLE IF NOT EXISTS ai_tool_risk_policies (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tool_name VARCHAR(64) NOT NULL,
  scene VARCHAR(32) NULL,
  command_class VARCHAR(32) NULL,
  argument_rules LONGTEXT NULL,
  approval_required TINYINT(1) NOT NULL DEFAULT 0,
  risk_level VARCHAR(16) NOT NULL DEFAULT 'medium',
  priority INT NOT NULL DEFAULT 0,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  policy_version VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_ai_tool_risk_policies_tool_enabled (tool_name, enabled)
);

ALTER TABLE ai_approval_tasks
  ADD COLUMN lock_expires_at DATETIME(3) NULL AFTER expires_at,
  ADD COLUMN matched_rule_id BIGINT UNSIGNED NULL AFTER lock_expires_at,
  ADD COLUMN policy_version VARCHAR(64) NULL AFTER matched_rule_id,
  ADD COLUMN decision_source VARCHAR(32) NULL AFTER policy_version;

ALTER TABLE ai_approval_tasks
  ADD INDEX idx_ai_approval_tasks_lock_expires_at (lock_expires_at),
  ADD INDEX idx_ai_approval_tasks_matched_rule_id (matched_rule_id);

-- +migrate Down
ALTER TABLE ai_approval_tasks
  DROP INDEX idx_ai_approval_tasks_matched_rule_id,
  DROP INDEX idx_ai_approval_tasks_lock_expires_at;

ALTER TABLE ai_approval_tasks
  DROP COLUMN decision_source,
  DROP COLUMN policy_version,
  DROP COLUMN matched_rule_id,
  DROP COLUMN lock_expires_at;

DROP TABLE IF EXISTS ai_tool_risk_policies;
