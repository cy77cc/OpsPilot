-- +migrate Up
-- Dashboard observability cache tables

-- 集群资源快照表
CREATE TABLE IF NOT EXISTS cluster_resource_snapshots (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL COMMENT '关联集群ID',
  cpu_allocatable_cores DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '可分配 CPU 核数',
  cpu_requested_cores DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '已请求 CPU 核数',
  cpu_limit_cores DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT 'CPU 限制核数',
  cpu_usage_cores DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '实际 CPU 使用核数',
  memory_allocatable_mb BIGINT NOT NULL DEFAULT 0 COMMENT '可分配内存 MB',
  memory_requested_mb BIGINT NOT NULL DEFAULT 0 COMMENT '已请求内存 MB',
  memory_limit_mb BIGINT NOT NULL DEFAULT 0 COMMENT '内存限制 MB',
  memory_usage_mb BIGINT NOT NULL DEFAULT 0 COMMENT '实际内存使用 MB',
  pod_total INT NOT NULL DEFAULT 0 COMMENT 'Pod 总数',
  pod_running INT NOT NULL DEFAULT 0 COMMENT '运行中 Pod 数',
  pod_pending INT NOT NULL DEFAULT 0 COMMENT '等待中 Pod 数',
  pod_failed INT NOT NULL DEFAULT 0 COMMENT '失败 Pod 数',
  pv_count INT NOT NULL DEFAULT 0 COMMENT 'PV 数量',
  pvc_count INT NOT NULL DEFAULT 0 COMMENT 'PVC 数量',
  storage_used_gb DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '存储使用量 GB',
  collected_at TIMESTAMP NOT NULL COMMENT '采集时间',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_cluster_collected (cluster_id, collected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='集群资源快照表';

-- K8s 工作负载统计表
CREATE TABLE IF NOT EXISTS k8s_workload_stats (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL COMMENT '关联集群ID',
  namespace VARCHAR(128) NOT NULL DEFAULT '' COMMENT '命名空间（空表示集群级别）',
  deployment_total INT NOT NULL DEFAULT 0 COMMENT 'Deployment 总数',
  deployment_healthy INT NOT NULL DEFAULT 0 COMMENT '健康的 Deployment 数',
  statefulset_total INT NOT NULL DEFAULT 0 COMMENT 'StatefulSet 总数',
  statefulset_healthy INT NOT NULL DEFAULT 0 COMMENT '健康的 StatefulSet 数',
  daemonset_total INT NOT NULL DEFAULT 0 COMMENT 'DaemonSet 总数',
  daemonset_healthy INT NOT NULL DEFAULT 0 COMMENT '健康的 DaemonSet 数',
  service_count INT NOT NULL DEFAULT 0 COMMENT 'Service 数量',
  ingress_count INT NOT NULL DEFAULT 0 COMMENT 'Ingress 数量',
  collected_at TIMESTAMP NOT NULL COMMENT '采集时间',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_cluster_ns_collected (cluster_id, namespace, collected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='K8s 工作负载统计表';

-- 异常 Pod 缓存表
CREATE TABLE IF NOT EXISTS k8s_issue_pods (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL COMMENT '关联集群ID',
  namespace VARCHAR(128) NOT NULL COMMENT '命名空间',
  pod_name VARCHAR(256) NOT NULL COMMENT 'Pod 名称',
  issue_type VARCHAR(64) NOT NULL COMMENT '问题类型',
  issue_reason VARCHAR(256) NOT NULL COMMENT '问题原因',
  message TEXT COMMENT '详细信息',
  first_seen_at TIMESTAMP NOT NULL COMMENT '首次发现时间',
  last_seen_at TIMESTAMP NOT NULL COMMENT '最后发现时间',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_cluster_ns_pod (cluster_id, namespace, pod_name),
  INDEX idx_issue_type (issue_type),
  INDEX idx_last_seen (last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='异常 Pod 缓存表';

-- +migrate Down
DROP TABLE IF EXISTS k8s_issue_pods;
DROP TABLE IF EXISTS k8s_workload_stats;
DROP TABLE IF EXISTS cluster_resource_snapshots;
