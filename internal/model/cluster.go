package model

import (
	"time"
)

// Cluster 集群表
type Cluster struct {
	ID             uint       `gorm:"primaryKey;column:id" json:"id"`
	Name           string     `gorm:"column:name;type:varchar(64);not null;uniqueIndex" json:"name"`
	Description    string     `gorm:"column:description;type:varchar(256)" json:"description"`
	Version        string     `gorm:"column:version;type:varchar(64)" json:"version"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	Type           string     `gorm:"column:type;type:varchar(32);not null" json:"type"`                                     // kubernetes / openshift
	Source         string     `gorm:"column:source;type:varchar(32);default:'platform_managed';index" json:"source"`         // platform_managed / external_managed
	EnvType        string     `gorm:"column:env_type;type:varchar(32);not null;default:'development';index" json:"env_type"` // development / staging / production
	Endpoint       string     `gorm:"column:endpoint;type:varchar(256)" json:"endpoint"`
	KubeConfig     string     `gorm:"column:kubeconfig;type:mediumtext" json:"-"` // 逐步废弃，迁移到 credential
	CACert         string     `gorm:"column:ca_cert;type:text" json:"-"`
	Token          string     `gorm:"column:token;type:text" json:"-"`
	Nodes          string     `gorm:"column:nodes;type:json" json:"nodes"`
	AuthMethod     string     `gorm:"column:auth_method;type:varchar(32)" json:"auth_method"` // kubeconfig / cert / token
	CredentialID   *uint      `gorm:"column:credential_id;index" json:"credential_id"`
	K8sVersion     string     `gorm:"column:k8s_version;type:varchar(32)" json:"k8s_version"`
	PodCIDR        string     `gorm:"column:pod_cidr;type:varchar(32)" json:"pod_cidr"`
	ServiceCIDR    string     `gorm:"column:service_cidr;type:varchar(32)" json:"service_cidr"`
	ManagementMode string     `gorm:"column:management_mode;type:varchar(32);default:'k8s-only'" json:"management_mode"`
	LastSyncAt     *time.Time `gorm:"column:last_sync_at" json:"last_sync_at"`
	CreatedBy      string     `gorm:"column:created_by;type:varchar(64)" json:"created_by"`
	UpdatedBy      string     `gorm:"column:updated_by;type:varchar(64)" json:"updated_by"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Cluster) TableName() string {
	return "clusters"
}

// ClusterBootstrapProfile defines reusable bootstrap defaults.
type ClusterBootstrapProfile struct {
	ID                   uint      `gorm:"primaryKey;column:id" json:"id"`
	Name                 string    `gorm:"column:name;type:varchar(128);not null;uniqueIndex" json:"name"`
	Description          string    `gorm:"column:description;type:varchar(512)" json:"description"`
	VersionChannel       string    `gorm:"column:version_channel;type:varchar(32);default:'stable-1'" json:"version_channel"`
	K8sVersion           string    `gorm:"column:k8s_version;type:varchar(32)" json:"k8s_version"`
	RepoMode             string    `gorm:"column:repo_mode;type:varchar(16);default:'online'" json:"repo_mode"`
	RepoURL              string    `gorm:"column:repo_url;type:varchar(512)" json:"repo_url"`
	ImageRepository      string    `gorm:"column:image_repository;type:varchar(256)" json:"image_repository"`
	EndpointMode         string    `gorm:"column:endpoint_mode;type:varchar(16);default:'nodeIP'" json:"endpoint_mode"`
	ControlPlaneEndpoint string    `gorm:"column:control_plane_endpoint;type:varchar(256)" json:"control_plane_endpoint"`
	VIPProvider          string    `gorm:"column:vip_provider;type:varchar(32)" json:"vip_provider"`
	EtcdMode             string    `gorm:"column:etcd_mode;type:varchar(16);default:'stacked'" json:"etcd_mode"`
	ExternalEtcdJSON     string    `gorm:"column:external_etcd_json;type:longtext" json:"external_etcd_json"`
	CreatedBy            uint64    `gorm:"column:created_by;index" json:"created_by"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ClusterBootstrapProfile) TableName() string {
	return "cluster_bootstrap_profiles"
}
