package contracts

// Scope represents a reusable targeting scope across specialist tasks.
type Scope struct {
	Cluster     string `json:"cluster,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Service     string `json:"service,omitempty"`
	Host        string `json:"host,omitempty"`
	Environment string `json:"environment,omitempty"`
	TimeRange   string `json:"time_range,omitempty"`
}

// ArtifactRef links to persisted artifacts used as input or output of delegation.
type ArtifactRef struct {
	Kind      string `json:"kind"`
	ContentID string `json:"content_id,omitempty"`
	Label     string `json:"label,omitempty"`
}
