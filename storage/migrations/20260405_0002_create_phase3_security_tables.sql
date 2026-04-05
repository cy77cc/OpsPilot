-- +migrate Up
CREATE TABLE IF NOT EXISTS admission_policies (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  cluster_id BIGINT UNSIGNED NOT NULL,
  policy_name VARCHAR(191) NOT NULL,
  version VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  content_json LONGTEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_admission_policies_cluster_name (cluster_id, policy_name)
);

CREATE TABLE IF NOT EXISTS admission_exemptions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  cluster_id BIGINT UNSIGNED NOT NULL,
  scope_type VARCHAR(32) NOT NULL,
  scope_ref VARCHAR(255) NOT NULL,
  reason LONGTEXT NOT NULL,
  approval_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_admission_exemptions_cluster_scope (cluster_id, scope_type, scope_ref)
);

CREATE TABLE IF NOT EXISTS image_scan_reports (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  cluster_id BIGINT UNSIGNED NOT NULL,
  image_digest VARCHAR(255) NOT NULL,
  scanner VARCHAR(32) NOT NULL,
  severity_summary_json LONGTEXT NOT NULL,
  sbom_ref VARCHAR(255) NOT NULL DEFAULT '',
  policy_decision VARCHAR(32) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_image_scan_reports_cluster_digest (cluster_id, image_digest)
);

CREATE TABLE IF NOT EXISTS gitops_app_releases (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  cluster_id BIGINT UNSIGNED NOT NULL,
  app_name VARCHAR(191) NOT NULL,
  environment VARCHAR(32) NOT NULL,
  git_revision VARCHAR(128) NOT NULL,
  sync_result VARCHAR(32) NOT NULL,
  rollback_ref VARCHAR(128) NOT NULL DEFAULT '',
  audit_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_gitops_app_releases_cluster_app (cluster_id, app_name)
);

CREATE TABLE IF NOT EXISTS runtime_security_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  cluster_id BIGINT UNSIGNED NOT NULL,
  namespace VARCHAR(191) NOT NULL DEFAULT '',
  workload VARCHAR(191) NOT NULL DEFAULT '',
  rule_id VARCHAR(191) NOT NULL DEFAULT '',
  severity VARCHAR(32) NOT NULL,
  source VARCHAR(32) NOT NULL,
  raw_payload_json LONGTEXT NOT NULL,
  dispose_status VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_runtime_security_events_cluster_severity (cluster_id, severity),
  KEY idx_runtime_security_events_cluster_rule (cluster_id, rule_id)
);

CREATE TABLE IF NOT EXISTS runtime_disposal_actions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id BIGINT UNSIGNED NOT NULL,
  action VARCHAR(64) NOT NULL,
  mode VARCHAR(32) NOT NULL,
  approval_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  audit_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  result VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_runtime_disposal_actions_event (event_id)
);

-- +migrate Down
DROP TABLE IF EXISTS runtime_disposal_actions;
DROP TABLE IF EXISTS runtime_security_events;
DROP TABLE IF EXISTS gitops_app_releases;
DROP TABLE IF EXISTS image_scan_reports;
DROP TABLE IF EXISTS admission_exemptions;
DROP TABLE IF EXISTS admission_policies;
