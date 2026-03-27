package v1

import "time"

// ProbeReq is the request body for the Probe endpoint (POST /hosts/probe).
// It carries SSH connectivity parameters used to test access before registering a host.
type ProbeReq struct {
	Name     string  `json:"name"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
	AuthType string  `json:"auth_type"` // "password" or "key"
	Username string  `json:"username"`
	Password string  `json:"password"`
	SSHKeyID *uint64 `json:"ssh_key_id"`
}

// ProbeFacts holds the system facts collected during an SSH probe.
type ProbeFacts struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Kernel   string `json:"kernel"`
	CPUCores int    `json:"cpu_cores"`
	MemoryMB int    `json:"memory_mb"`
	DiskGB   int    `json:"disk_gb"`
}

// ProbeResp is the response body for the Probe endpoint.
type ProbeResp struct {
	ProbeToken string     `json:"probe_token"`
	Reachable  bool       `json:"reachable"`
	LatencyMS  int64      `json:"latency_ms"`
	Facts      ProbeFacts `json:"facts"`
	Warnings   []string   `json:"warnings"`
	ErrorCode  string     `json:"error_code,omitempty"`
	Message    string     `json:"message,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

// CreateReq is the request body for creating a new host (POST /hosts).
// A valid ProbeToken obtained from the Probe endpoint is required.
type CreateReq struct {
	ProbeToken   string   `json:"probe_token"`
	Name         string   `json:"name"`
	IP           string   `json:"ip"`
	Port         int      `json:"port"`
	AuthType     string   `json:"auth_type"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	SSHKeyID     *uint64  `json:"ssh_key_id"`
	Description  string   `json:"description"`
	Labels       []string `json:"labels"`
	Role         string   `json:"role"`
	ClusterID    uint     `json:"cluster_id"`
	Source       string   `json:"source"`
	Provider     string   `json:"provider"`
	ProviderID   string   `json:"provider_instance_id"`
	ParentHostID *uint64  `json:"parent_host_id"`
	Force        bool     `json:"force"`
	Status       string   `json:"status"`
}

// UpdateCredentialsReq is the request body for updating SSH credentials of an existing host
// (PUT /hosts/:id/credentials).
type UpdateCredentialsReq struct {
	AuthType string  `json:"auth_type"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	SSHKeyID *uint64 `json:"ssh_key_id"`
	Port     int     `json:"port"`
}

// ActionReq is the request body for single-host action operations (POST /hosts/:id/action).
type ActionReq struct {
	Action string     `json:"action"`
	Reason string     `json:"reason,omitempty"`
	Until  *time.Time `json:"until,omitempty"`
}

// HealthCheckReq is the request body for on-demand host health checks
// (POST /hosts/:id/health/check).
type HealthCheckReq struct {
	Deep bool `json:"deep"`
}

// HealthSnapshot represents host health diagnostics result.
type HealthSnapshot struct {
	ID                 uint64    `json:"id"`
	HostID             uint64    `json:"host_id"`
	State              string    `json:"state"`
	ConnectivityStatus string    `json:"connectivity_status"`
	ResourceStatus     string    `json:"resource_status"`
	SystemStatus       string    `json:"system_status"`
	LatencyMS          int64     `json:"latency_ms"`
	CpuLoad            float64   `json:"cpu_load"`
	MemoryUsedMB       int       `json:"memory_used_mb"`
	MemoryTotalMB      int       `json:"memory_total_mb"`
	DiskUsedPct        float64   `json:"disk_used_pct"`
	InodeUsedPct       float64   `json:"inode_used_pct"`
	SummaryJSON        string    `json:"summary_json"`
	ErrorMessage       string    `json:"error_message"`
	CheckedAt          time.Time `json:"checked_at"`
}

// BatchReq is the request body for batch host operations (POST /hosts/batch).
type BatchReq struct {
	HostIDs []uint64 `json:"host_ids"`
	Action  string   `json:"action"`
	Tags    []string `json:"tags"`
}

// AddTagReq is the request body for adding a tag to a host (POST /hosts/:id/tags).
type AddTagReq struct {
	Tag string `json:"tag" binding:"required"`
}

// SSHExecReq is the request body for running a command on a single host via SSH
// (POST /hosts/:id/exec).
type SSHExecReq struct {
	Command string `json:"command" binding:"required"`
}

// BatchExecReq is the request body for running a command across multiple hosts via SSH
// (POST /hosts/batch/exec).
type BatchExecReq struct {
	HostIDs []uint64 `json:"host_ids"`
	Command string   `json:"command" binding:"required"`
}

// WriteFileReq is the request body for writing file content to a remote host via SFTP
// (PUT /hosts/:id/files/content).
type WriteFileReq struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

// MakeDirReq is the request body for creating a directory on a remote host via SFTP
// (POST /hosts/:id/files/mkdir).
type MakeDirReq struct {
	Path string `json:"path" binding:"required"`
}

// RenamePathReq is the request body for renaming a file or directory on a remote host via SFTP
// (POST /hosts/:id/files/rename).
type RenamePathReq struct {
	OldPath string `json:"old_path" binding:"required"`
	NewPath string `json:"new_path" binding:"required"`
}

// SSHKeyCreateReq is the request body for creating an SSH key pair (POST /ssh-keys).
type SSHKeyCreateReq struct {
	Name       string `json:"name"`
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase"`
}

// SSHKeyVerifyReq is the request body for verifying an SSH key against a remote host
// (POST /ssh-keys/:id/verify).
type SSHKeyVerifyReq struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

// CloudAccountReq is the request body for creating or testing a cloud provider account
// (POST /cloud/accounts, POST /cloud/:provider/test).
type CloudAccountReq struct {
	Provider        string `json:"provider"`
	AccountName     string `json:"account_name"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	RegionDefault   string `json:"region_default"`
}

// CloudQueryReq is the request body for querying instances from a cloud provider
// (POST /cloud/:provider/instances).
type CloudQueryReq struct {
	Provider  string `json:"provider"`
	AccountID uint64 `json:"account_id"`
	Region    string `json:"region"`
	Keyword   string `json:"keyword"`
}

// CloudInstance represents a single cloud compute instance returned by a cloud provider query.
type CloudInstance struct {
	InstanceID string `json:"instance_id"`
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Region     string `json:"region"`
	Status     string `json:"status"`
	OS         string `json:"os"`
	CPU        int    `json:"cpu"`
	MemoryMB   int    `json:"memory_mb"`
	DiskGB     int    `json:"disk_gb"`
}

// CloudImportReq is the request body for importing cloud instances as managed hosts
// (POST /cloud/:provider/import).
type CloudImportReq struct {
	Provider  string          `json:"provider"`
	AccountID uint64          `json:"account_id"`
	Instances []CloudInstance `json:"instances"`
	Role      string          `json:"role"`
	Labels    []string        `json:"labels"`
}

// KVMPreviewReq is the request body for previewing a KVM virtual machine configuration
// (POST /hosts/:id/kvm/preview).
type KVMPreviewReq struct {
	Name          string `json:"name"`
	CPU           int    `json:"cpu"`
	MemoryMB      int    `json:"memory_mb"`
	DiskGB        int    `json:"disk_gb"`
	NetworkBridge string `json:"network_bridge"`
	Template      string `json:"template"`
}

// KVMProvisionReq is the request body for provisioning a KVM virtual machine on a host
// (POST /hosts/:id/kvm/provision).
type KVMProvisionReq struct {
	Name          string  `json:"name"`
	CPU           int     `json:"cpu"`
	MemoryMB      int     `json:"memory_mb"`
	DiskGB        int     `json:"disk_gb"`
	NetworkBridge string  `json:"network_bridge"`
	Template      string  `json:"template"`
	IP            string  `json:"ip"`
	SSHUser       string  `json:"ssh_user"`
	Password      string  `json:"password"`
	SSHKeyID      *uint64 `json:"ssh_key_id"`
}
