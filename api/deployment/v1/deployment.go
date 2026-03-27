package v1

type EnvironmentBootstrapReq struct {
	Name           string `json:"name"`
	RuntimeType    string `json:"runtime_type"`
	PackageVersion string `json:"package_version"`
	Env            string `json:"env"`
	TargetID       uint   `json:"target_id"`
	ClusterID      uint   `json:"cluster_id"`
	ControlPlaneID uint   `json:"control_plane_host_id"`
	WorkerIDs      []uint `json:"worker_host_ids"`
	NodeIDs        []uint `json:"node_ids"`
}

type EnvironmentBootstrapResp struct {
	JobID          string `json:"job_id"`
	Status         string `json:"status"`
	RuntimeType    string `json:"runtime_type"`
	PackageVersion string `json:"package_version"`
	TargetID       uint   `json:"target_id,omitempty"`
}

type ClusterCredentialImportReq struct {
	Name        string `json:"name"`
	RuntimeType string `json:"runtime_type"`
	AuthMethod  string `json:"auth_method"`
	Endpoint    string `json:"endpoint"`
	Kubeconfig  string `json:"kubeconfig"`
	CACert      string `json:"ca_cert"`
	Cert        string `json:"cert"`
	Key         string `json:"key"`
	Token       string `json:"token"`
}

type ClusterCredentialResp struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	RuntimeType     string `json:"runtime_type"`
	Source          string `json:"source"`
	ClusterID       uint   `json:"cluster_id"`
	Endpoint        string `json:"endpoint"`
	AuthMethod      string `json:"auth_method"`
	Status          string `json:"status"`
	LastTestStatus  string `json:"last_test_status,omitempty"`
	LastTestMessage string `json:"last_test_message,omitempty"`
}

type ClusterCredentialTestResp struct {
	CredentialID uint   `json:"credential_id"`
	Connected    bool   `json:"connected"`
	Message      string `json:"message"`
	LatencyMS    int64  `json:"latency_ms,omitempty"`
}
