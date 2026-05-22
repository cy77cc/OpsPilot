-- +migrate Up
CREATE TABLE opsagent_ca (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ca_cert_pem TEXT NOT NULL,
  ca_key_pem TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE opsagent_host_certificates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  host_id INTEGER NOT NULL,
  instance_id INTEGER NOT NULL,
  serial_number TEXT NOT NULL,
  cert_pem TEXT NOT NULL,
  key_pem TEXT NOT NULL,
  not_after TIMESTAMP NOT NULL,
  revoked INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_opsagent_host_cert_instance FOREIGN KEY (instance_id) REFERENCES host_plugin_instances (id) ON DELETE CASCADE
);

CREATE TABLE host_plugin_packages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plugin_key TEXT NOT NULL,
  version TEXT NOT NULL,
  arch TEXT NOT NULL,
  filename TEXT NOT NULL,
  storage_path TEXT NOT NULL,
  checksum TEXT NOT NULL,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  uploaded_by INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_host_plugin_package UNIQUE (plugin_key, version, arch)
);

CREATE INDEX idx_opsagent_host_cert_host_id ON opsagent_host_certificates (host_id);
CREATE INDEX idx_opsagent_host_cert_instance_id ON opsagent_host_certificates (instance_id);
CREATE INDEX idx_host_plugin_packages_plugin_key ON host_plugin_packages (plugin_key);

-- +migrate Down
DROP TABLE IF EXISTS opsagent_host_certificates;
DROP TABLE IF EXISTS opsagent_ca;
DROP TABLE IF EXISTS host_plugin_packages;
