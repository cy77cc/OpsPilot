package cluster

import (
	"time"
)

// ClusterNode represents a node in a cluster
type ClusterNode struct {
	ID        uint      `json:"id"`
	ClusterID uint      `json:"cluster_id"`
	HostID    uint      `json:"host_id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Role      string    `json:"role"` // control-plane, worker
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClusterDetail represents detailed cluster information
type ClusterDetail struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	NodeCount   int       `json:"node_count"`
	Endpoint    string    `json:"endpoint"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ClusterListItem represents a cluster in list view
type ClusterListItem struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	NodeCount int       `json:"node_count"`
	CreatedAt time.Time `json:"created_at"`
}
