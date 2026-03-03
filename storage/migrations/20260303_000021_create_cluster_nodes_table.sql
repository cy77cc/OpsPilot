-- +migrate Up
-- Create cluster_nodes table for storing node information synced from Kubernetes API

CREATE TABLE IF NOT EXISTS cluster_nodes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  cluster_id BIGINT UNSIGNED NOT NULL COMMENT '关联集群ID',
  host_id BIGINT UNSIGNED NULL COMMENT '关联主机ID（可选）',
  name VARCHAR(64) NOT NULL COMMENT '节点名称',
  ip VARCHAR(45) NOT NULL COMMENT '节点IP地址',
  role VARCHAR(32) NOT NULL DEFAULT 'worker' COMMENT '节点角色: control-plane/worker/etcd',
  status VARCHAR(32) NOT NULL DEFAULT 'unknown' COMMENT '节点状态: ready/notready/unknown',
  kubelet_version VARCHAR(32) NULL COMMENT 'Kubelet版本',
  kube_proxy_version VARCHAR(32) NULL COMMENT 'Kube-proxy版本',
  container_runtime VARCHAR(32) NULL COMMENT '容器运行时',
  os_image VARCHAR(128) NULL COMMENT '操作系统镜像',
  kernel_version VARCHAR(64) NULL COMMENT '内核版本',
  allocatable_cpu VARCHAR(16) NULL COMMENT '可分配CPU',
  allocatable_mem VARCHAR(16) NULL COMMENT '可分配内存',
  allocatable_pods INT NOT NULL DEFAULT 0 COMMENT '可分配Pod数量',
  labels JSON NULL COMMENT '节点标签',
  taints JSON NULL COMMENT '节点污点',
  conditions JSON NULL COMMENT '节点状态条件',
  joined_at TIMESTAMP NULL COMMENT '加入集群时间',
  last_seen_at TIMESTAMP NULL COMMENT '最后心跳时间',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_cluster_node (cluster_id, name),
  INDEX idx_cluster_id (cluster_id),
  INDEX idx_host_id (host_id),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='集群节点表';

-- +migrate Down
DROP TABLE IF EXISTS cluster_nodes;
