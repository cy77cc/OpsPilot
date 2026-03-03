package cluster

import (
	"time"
)

// ClusterNode represents a node in a cluster
type ClusterNode struct {
	ID               uint      `json:"id"`
	ClusterID        uint      `json:"cluster_id"`
	HostID           *uint     `json:"host_id"`
	Name             string    `json:"name"`
	IP               string    `json:"ip"`
	Role             string    `json:"role"` // control-plane, worker
	Status           string    `json:"status"`
	KubeletVersion   string    `json:"kubelet_version"`
	ContainerRuntime string    `json:"container_runtime"`
	OSImage          string    `json:"os_image"`
	KernelVersion    string    `json:"kernel_version"`
	AllocatableCPU   string    `json:"allocatable_cpu"`
	AllocatableMem   string    `json:"allocatable_mem"`
	Labels           string    `json:"labels"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ClusterDetail represents detailed cluster information
type ClusterDetail struct {
	ID             uint       `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Version        string     `json:"version"`
	K8sVersion     string     `json:"k8s_version"`
	Status         string     `json:"status"`
	Source         string     `json:"source"`
	Type           string     `json:"type"`
	NodeCount      int        `json:"node_count"`
	Endpoint       string     `json:"endpoint"`
	PodCIDR        string     `json:"pod_cidr"`
	ServiceCIDR    string     `json:"service_cidr"`
	ManagementMode string     `json:"management_mode"`
	CredentialID   *uint      `json:"credential_id"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ClusterListItem represents a cluster in list view
type ClusterListItem struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	K8sVersion  string     `json:"k8s_version"`
	Status      string     `json:"status"`
	Source      string     `json:"source"`
	NodeCount   int        `json:"node_count"`
	Endpoint    string     `json:"endpoint"`
	Description string     `json:"description"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ClusterCreateReq represents request to create a cluster (import external)
type ClusterCreateReq struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Kubeconfig    string `json:"kubeconfig"`
	Endpoint      string `json:"endpoint"`
	CACert        string `json:"ca_cert"`
	Cert          string `json:"cert"`
	Key           string `json:"key"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip_tls_verify"`
	AuthMethod    string `json:"auth_method"` // kubeconfig, certificate|cert, token
}

// ClusterUpdateReq represents request to update a cluster
type ClusterUpdateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ClusterTestResp represents cluster connectivity test result
type ClusterTestResp struct {
	ClusterID uint   `json:"cluster_id"`
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
	Version   string `json:"version,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

// BootstrapStepStatus represents the status of a bootstrap step
type BootstrapStepStatus struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"` // pending, running, succeeded, failed
	Message    string     `json:"message,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	HostID     uint       `json:"host_id,omitempty"`
	Output     string     `json:"output,omitempty"`
}

// BootstrapTaskDetail represents detailed bootstrap task information
type BootstrapTaskDetail struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	ClusterID            *uint                 `json:"cluster_id"`
	K8sVersion           string                `json:"k8s_version"`
	VersionChannel       string                `json:"version_channel"`
	RepoMode             string                `json:"repo_mode"`
	RepoURL              string                `json:"repo_url"`
	ImageRepository      string                `json:"image_repository"`
	EndpointMode         string                `json:"endpoint_mode"`
	ControlPlaneEndpoint string                `json:"control_plane_endpoint"`
	VIPProvider          string                `json:"vip_provider"`
	EtcdMode             string                `json:"etcd_mode"`
	CNI                  string                `json:"cni"`
	PodCIDR              string                `json:"pod_cidr"`
	ServiceCIDR          string                `json:"service_cidr"`
	Status               string                `json:"status"`
	Steps                []BootstrapStepStatus `json:"steps"`
	CurrentStep          int                   `json:"current_step"`
	ErrorMessage         string                `json:"error_message,omitempty"`
	ResolvedConfigJSON   string                `json:"resolved_config_json,omitempty"`
	DiagnosticsJSON      string                `json:"diagnostics_json,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

type BootstrapProfileExternalEtcd struct {
	Endpoints []string `json:"endpoints"`
	CACert    string   `json:"ca_cert"`
	Cert      string   `json:"cert"`
	Key       string   `json:"key"`
}

type BootstrapProfileCreateReq struct {
	Name                 string                        `json:"name" binding:"required"`
	Description          string                        `json:"description"`
	VersionChannel       string                        `json:"version_channel"`
	K8sVersion           string                        `json:"k8s_version"`
	RepoMode             string                        `json:"repo_mode"`
	RepoURL              string                        `json:"repo_url"`
	ImageRepository      string                        `json:"image_repository"`
	EndpointMode         string                        `json:"endpoint_mode"`
	ControlPlaneEndpoint string                        `json:"control_plane_endpoint"`
	VIPProvider          string                        `json:"vip_provider"`
	EtcdMode             string                        `json:"etcd_mode"`
	ExternalEtcd         *BootstrapProfileExternalEtcd `json:"external_etcd"`
}

type BootstrapProfileUpdateReq struct {
	Description          string                        `json:"description"`
	VersionChannel       string                        `json:"version_channel"`
	K8sVersion           string                        `json:"k8s_version"`
	RepoMode             string                        `json:"repo_mode"`
	RepoURL              string                        `json:"repo_url"`
	ImageRepository      string                        `json:"image_repository"`
	EndpointMode         string                        `json:"endpoint_mode"`
	ControlPlaneEndpoint string                        `json:"control_plane_endpoint"`
	VIPProvider          string                        `json:"vip_provider"`
	EtcdMode             string                        `json:"etcd_mode"`
	ExternalEtcd         *BootstrapProfileExternalEtcd `json:"external_etcd"`
}

type BootstrapProfileItem struct {
	ID                   uint        `json:"id"`
	Name                 string      `json:"name"`
	Description          string      `json:"description"`
	VersionChannel       string      `json:"version_channel"`
	K8sVersion           string      `json:"k8s_version"`
	RepoMode             string      `json:"repo_mode"`
	RepoURL              string      `json:"repo_url"`
	ImageRepository      string      `json:"image_repository"`
	EndpointMode         string      `json:"endpoint_mode"`
	ControlPlaneEndpoint string      `json:"control_plane_endpoint"`
	VIPProvider          string      `json:"vip_provider"`
	EtcdMode             string      `json:"etcd_mode"`
	ExternalEtcd         interface{} `json:"external_etcd,omitempty"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}
