package v1

// PreviewRunRequest is the request body for previewing an automation run.
type PreviewRunRequest struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

// ExecuteRunRequest is the request body for executing an approved automation run.
type ExecuteRunRequest struct {
	ApprovalToken string         `json:"approval_token"`
	Action        string         `json:"action"`
	Params        map[string]any `json:"params"`
}

// CreateInventoryRequest is the request body for creating an Ansible inventory.
type CreateInventoryRequest struct {
	Name      string `json:"name"`
	HostsJSON string `json:"hosts_json"`
}

// CreatePlaybookRequest is the request body for creating an Ansible playbook.
type CreatePlaybookRequest struct {
	Name       string `json:"name"`
	ContentYML string `json:"content_yml"`
	RiskLevel  string `json:"risk_level"`
}
