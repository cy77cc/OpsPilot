package logic

type PreviewRunReq struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

type ExecuteRunReq struct {
	ApprovalToken string         `json:"approval_token"`
	Action        string         `json:"action"`
	Params        map[string]any `json:"params"`
}

type CreateInventoryReq struct {
	Name      string `json:"name"`
	HostsJSON string `json:"hosts_json"`
}

type CreatePlaybookReq struct {
	Name       string `json:"name"`
	ContentYML string `json:"content_yml"`
	RiskLevel  string `json:"risk_level"`
}
