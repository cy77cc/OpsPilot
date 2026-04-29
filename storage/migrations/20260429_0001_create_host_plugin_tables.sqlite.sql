-- +migrate Up
CREATE TABLE host_plugins (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plugin_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  description TEXT NOT NULL,
  default_version TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE host_plugin_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plugin_id INTEGER NOT NULL,
  version TEXT NOT NULL,
  arch TEXT NOT NULL,
  package_path TEXT NOT NULL,
  install_entry TEXT NOT NULL,
  upgrade_entry TEXT NOT NULL,
  uninstall_entry TEXT NOT NULL,
  checksum TEXT NOT NULL,
  capabilities_json TEXT NOT NULL,
  config_schema_json TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_host_plugin_versions_plugin FOREIGN KEY (plugin_id) REFERENCES host_plugins (id) ON DELETE CASCADE,
  CONSTRAINT uk_host_plugin_version_arch UNIQUE (plugin_id, version, arch)
);

CREATE TABLE host_plugin_instances (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  host_id INTEGER NOT NULL,
  plugin_id INTEGER NOT NULL,
  desired_version TEXT NOT NULL,
  installed_version TEXT NOT NULL DEFAULT '',
  install_status TEXT NOT NULL DEFAULT 'pending',
  runtime_status TEXT NOT NULL DEFAULT 'pending_online',
  health_status TEXT NOT NULL DEFAULT 'unknown',
  agent_id TEXT NOT NULL DEFAULT '',
  last_seen_at DATETIME NULL,
  capabilities_json TEXT NOT NULL,
  last_error TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_host_plugin_instances_plugin FOREIGN KEY (plugin_id) REFERENCES host_plugins (id) ON DELETE CASCADE,
  CONSTRAINT uk_host_plugin_instance UNIQUE (host_id, plugin_id)
);

CREATE TABLE host_plugin_config_revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  instance_id INTEGER NOT NULL,
  version TEXT NOT NULL,
  config_yaml TEXT NOT NULL,
  checksum TEXT NOT NULL,
  delivery_status TEXT NOT NULL DEFAULT 'pending',
  created_by INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_host_plugin_config_revisions_instance FOREIGN KEY (instance_id) REFERENCES host_plugin_instances (id) ON DELETE CASCADE
);

CREATE TABLE host_plugin_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  instance_id INTEGER NOT NULL,
  operation TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_by INTEGER NOT NULL DEFAULT 0,
  started_at DATETIME NULL,
  finished_at DATETIME NULL,
  error_message TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_host_plugin_tasks_instance FOREIGN KEY (instance_id) REFERENCES host_plugin_instances (id) ON DELETE CASCADE
);

CREATE TABLE host_plugin_task_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL,
  stream TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_host_plugin_task_logs_task FOREIGN KEY (task_id) REFERENCES host_plugin_tasks (id) ON DELETE CASCADE
);

CREATE INDEX idx_host_plugin_config_revisions_instance_id ON host_plugin_config_revisions (instance_id);
CREATE INDEX idx_host_plugin_tasks_instance_id ON host_plugin_tasks (instance_id);
CREATE INDEX idx_host_plugin_tasks_status ON host_plugin_tasks (status);
CREATE INDEX idx_host_plugin_task_logs_task_id ON host_plugin_task_logs (task_id);

-- +migrate Down
DROP TABLE IF EXISTS host_plugin_task_logs;
DROP TABLE IF EXISTS host_plugin_tasks;
DROP TABLE IF EXISTS host_plugin_config_revisions;
DROP TABLE IF EXISTS host_plugin_instances;
DROP TABLE IF EXISTS host_plugin_versions;
DROP TABLE IF EXISTS host_plugins;
