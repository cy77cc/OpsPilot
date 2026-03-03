-- +migrate Up
CREATE TABLE IF NOT EXISTS cluster_bootstrap_profiles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(512) DEFAULT '',
  version_channel VARCHAR(32) NOT NULL DEFAULT 'stable-1',
  k8s_version VARCHAR(32) DEFAULT '',
  repo_mode VARCHAR(16) NOT NULL DEFAULT 'online',
  repo_url VARCHAR(512) DEFAULT '',
  image_repository VARCHAR(256) DEFAULT '',
  endpoint_mode VARCHAR(16) NOT NULL DEFAULT 'nodeIP',
  control_plane_endpoint VARCHAR(256) DEFAULT '',
  vip_provider VARCHAR(32) DEFAULT '',
  etcd_mode VARCHAR(16) NOT NULL DEFAULT 'stacked',
  external_etcd_json LONGTEXT,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_cluster_bootstrap_profiles_name (name),
  KEY idx_cluster_bootstrap_profiles_created_by (created_by),
  KEY idx_cluster_bootstrap_profiles_created_at (created_at)
);

-- +migrate Down
DROP TABLE IF EXISTS cluster_bootstrap_profiles;
