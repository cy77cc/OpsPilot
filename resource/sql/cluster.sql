CREATE TABLE `clusters` (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '集群ID',
    name VARCHAR(64) NOT NULL COMMENT '集群名称',
    description VARCHAR(256) COMMENT '集群描述',
    version VARCHAR(64) COMMENT 'Kubernetes版本',
    status VARCHAR(32) NOT NULL COMMENT '集群状态',
    type VARCHAR(32) NOT NULL COMMENT '集群类型', -- kubernetes / openshift
    source VARCHAR(32) NOT NULL DEFAULT 'platform_managed' COMMENT '集群来源', -- platform_managed / external_managed
    endpoint VARCHAR(256) NOT NULL COMMENT '集群API Endpoint',
    kubeconfig MEDIUMTEXT COMMENT 'KubeConfig配置',
    ca_cert TEXT COMMENT '集群CA证书',
    token TEXT COMMENT '集群访问令牌',
    nodes JSON COMMENT '节点列表',
    auth_method VARCHAR(32) NOT NULL COMMENT '认证方法', -- token / basic
    credential_id BIGINT UNSIGNED NULL COMMENT '关联凭据ID',
    k8s_version VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'Kubernetes版本(结构化)',
    pod_cidr VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'Pod CIDR',
    service_cidr VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'Service CIDR',
    management_mode VARCHAR(32) NOT NULL DEFAULT 'k8s-only' COMMENT '管理模式',
    last_sync_at TIMESTAMP NULL COMMENT '最近同步时间',
    created_by VARCHAR(64) COMMENT '创建人',
    updated_by VARCHAR(64) COMMENT '更新人',
    created_at TIMESTAMP NOT NULL COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL COMMENT '更新时间'
)
