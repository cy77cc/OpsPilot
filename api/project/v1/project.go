package v1

import "time"

type CreateProjectReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type ProjectResp struct {
	ID          uint          `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	OwnerID     int64         `json:"owner_id"`
	CreatedAt   time.Time     `json:"created_at"`
	Services    []ServiceResp `json:"services,omitempty"`
}

type ServiceResp struct {
	ID            uint      `json:"id"`
	ProjectID     uint      `json:"project_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Image         string    `json:"image"`
	Replicas      int32     `json:"replicas"`
	ServicePort   int32     `json:"service_port"`
	ContainerPort int32     `json:"container_port"`
	NodePort      int32     `json:"node_port"`
	EnvVars       any       `json:"env_vars"`
	Resources     any       `json:"resources"`
	YamlContent   string    `json:"yaml_content"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateServiceReq struct {
	ProjectID     uint         `json:"project_id" binding:"required"`
	Name          string       `json:"name" binding:"required"`
	Type          string       `json:"type" binding:"required,oneof=stateless stateful"`
	Image         string       `json:"image" binding:"required"`
	Replicas      int32        `json:"replicas" binding:"min=1"`
	ServicePort   int32        `json:"service_port" binding:"required"`
	ContainerPort int32        `json:"container_port" binding:"required"`
	NodePort      int32        `json:"node_port"`
	EnvVars       []EnvVar     `json:"env_vars"`
	Resources     *ResourceReq `json:"resources"`
	YamlContent   string       `json:"yaml_content"`
}

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceReq struct {
	Limits   map[string]string `json:"limits"`
	Requests map[string]string `json:"requests"`
}

type DeployServiceReq struct {
	ServiceID uint `json:"service_id" binding:"required"`
	ClusterID uint `json:"cluster_id" binding:"required"` // Target cluster
}

type DeployProjectReq struct {
	ProjectID uint `json:"project_id" binding:"required"`
	ClusterID uint `json:"cluster_id" binding:"required"`
}
