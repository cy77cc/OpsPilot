CREATE TABLE IF NOT EXISTS admission_policies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  cluster_id INTEGER NOT NULL,
  policy_name VARCHAR(191) NOT NULL,
  version VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  content_json TEXT NOT NULL DEFAULT '{}',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admission_policies_cluster_name
  ON admission_policies(cluster_id, policy_name);

CREATE TABLE IF NOT EXISTS admission_exemptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  cluster_id INTEGER NOT NULL,
  scope_type VARCHAR(32) NOT NULL,
  scope_ref VARCHAR(255) NOT NULL,
  reason TEXT NOT NULL,
  approval_id INTEGER NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admission_exemptions_cluster_scope
  ON admission_exemptions(cluster_id, scope_type, scope_ref);

CREATE TABLE IF NOT EXISTS image_scan_reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  cluster_id INTEGER NOT NULL,
  image_digest VARCHAR(255) NOT NULL,
  scanner VARCHAR(32) NOT NULL,
  severity_summary_json TEXT NOT NULL DEFAULT '{}',
  sbom_ref VARCHAR(255) NOT NULL DEFAULT '',
  policy_decision VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_image_scan_reports_cluster_digest
  ON image_scan_reports(cluster_id, image_digest);

CREATE TABLE IF NOT EXISTS gitops_app_releases (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  cluster_id INTEGER NOT NULL,
  app_name VARCHAR(191) NOT NULL,
  environment VARCHAR(32) NOT NULL,
  git_revision VARCHAR(128) NOT NULL,
  sync_result VARCHAR(32) NOT NULL,
  rollback_ref VARCHAR(128) NOT NULL DEFAULT '',
  audit_id INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gitops_app_releases_cluster_app
  ON gitops_app_releases(cluster_id, app_name);

CREATE TABLE IF NOT EXISTS runtime_security_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  cluster_id INTEGER NOT NULL,
  namespace VARCHAR(191) NOT NULL DEFAULT '',
  workload VARCHAR(191) NOT NULL DEFAULT '',
  rule_id VARCHAR(191) NOT NULL DEFAULT '',
  severity VARCHAR(32) NOT NULL,
  source VARCHAR(32) NOT NULL,
  raw_payload_json TEXT NOT NULL DEFAULT '{}',
  dispose_status VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_security_events_cluster_severity
  ON runtime_security_events(cluster_id, severity);

CREATE INDEX IF NOT EXISTS idx_runtime_security_events_cluster_rule
  ON runtime_security_events(cluster_id, rule_id);

CREATE TABLE IF NOT EXISTS runtime_disposal_actions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id INTEGER NOT NULL,
  action VARCHAR(64) NOT NULL,
  mode VARCHAR(32) NOT NULL,
  approval_id INTEGER NOT NULL DEFAULT 0,
  audit_id INTEGER NOT NULL DEFAULT 0,
  result VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_disposal_actions_event
  ON runtime_disposal_actions(event_id);
