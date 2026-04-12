package contracts

import "time"

// ClusterNode 集群节点响应结构。
type ClusterNode struct {
	ID               uint      `json:"id"`
	ClusterID        uint      `json:"cluster_id"`
	HostID           *uint     `json:"host_id"`
	Name             string    `json:"name"`
	IP               string    `json:"ip"`
	Role             string    `json:"role"`
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

// ClusterDetail 集群详情响应结构。
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
