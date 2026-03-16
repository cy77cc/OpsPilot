package logic

type GraphNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type GraphEdge struct {
	ID       string         `json:"id"`
	From     string         `json:"from"`
	To       string         `json:"to"`
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type QueryFilter struct {
	ProjectID    uint
	ClusterID    uint
	ResourceType string
	Keyword      string
}
