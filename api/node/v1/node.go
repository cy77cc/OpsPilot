package v1

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// -------------------- Node APIs --------------------

// CreateNodeReq 创建节点请求
type CreateNodeReq struct {
	Name        string            `json:"name" binding:"required"`       // 节点名称
	Hostname    string            `json:"hostname"`                      // 主机名
	Description string            `json:"description"`                   // 描述
	IP          string            `json:"ip" binding:"required,ip"`      // 节点IP
	Port        int               `json:"port" binding:"required,min=1"` // SSH端口
	SSHUser     string            `json:"ssh_user" binding:"required"`   // SSH用户名
	SSHPassword string            `json:"ssh_password"`
	SSHKeyID    model.NodeID      `json:"ssh_key_id" binding:"required"`           // SSH密钥ID
	ClusterID   int64             `json:"cluster_id"`                              // 所属集群ID (可选)
	Labels      map[string]string `json:"labels"`                                  // 标签
	Role        string            `json:"role" binding:"oneof=master worker none"` // 节点角色
}

// UpdateNodeReq 更新节点请求
type UpdateNodeReq struct {
	ID          model.NodeID      `json:"id" binding:"required"`
	Name        string            `json:"name"`
	Hostname    string            `json:"hostname"`
	Description string            `json:"description"`
	SSHUser     string            `json:"ssh_user"`
	SSHKeyID    int64             `json:"ssh_key_id"`
	ClusterID   int64             `json:"cluster_id"`
	Labels      map[string]string `json:"labels"`
	Role        string            `json:"role"`
}

// NodeResp 节点响应
type NodeResp struct {
	ID          model.NodeID `json:"id"`
	Name        string       `json:"name"`
	Hostname    string       `json:"hostname"`
	Description string       `json:"description"`
	IP          string       `json:"ip"`
	Port        int          `json:"port"`
	SSHUser     string       `json:"ssh_user"`
	SSHKeyID    model.NodeID `json:"ssh_key_id"`
	OS          string       `json:"os"`
	Arch        string       `json:"arch"`
	Kernel      string       `json:"kernel"`
	CPUCores    int          `json:"cpu_cores"`
	MemoryMB    int          `json:"memory_mb"`
	DiskGB      int          `json:"disk_gb"`
	Status      string       `json:"status"`
	Role        string       `json:"role"`
	ClusterID   int64        `json:"cluster_id"`
	Labels      string       `json:"labels"`
	LastCheckAt time.Time    `json:"last_check_at"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// ListNodeReq 节点列表请求
type ListNodeReq struct {
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=10"`
	ClusterID int64  `form:"cluster_id"`
	Status    string `form:"status"`
	Keyword   string `form:"keyword"` // 搜索 Name/IP
}

// ListNodeResp 节点列表响应
type ListNodeResp struct {
	Total int64      `json:"total"`
	List  []NodeResp `json:"list"`
}

// -------------------- SSH Key APIs --------------------

// CreateSSHKeyReq 创建SSH密钥请求
type CreateSSHKeyReq struct {
	Name       string `json:"name" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
	Passphrase string `json:"passphrase"`
}

// SSHKeyResp SSH密钥响应
type SSHKeyResp struct {
	ID        model.NodeID `json:"id"`
	Name      string       `json:"name"`
	CreatedAt time.Time    `json:"created_at"`
}

// ListSSHKeyReq SSH密钥列表请求
type ListSSHKeyReq struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"page_size,default=10"`
}

// ListSSHKeyResp SSH密钥列表响应
type ListSSHKeyResp struct {
	Total int64        `json:"total"`
	List  []SSHKeyResp `json:"list"`
}

// -------------------- Node Operation APIs --------------------

// NodeShellReq 节点Shell请求 (WebSocket)
type NodeShellReq struct {
	NodeID model.NodeID `form:"node_id" binding:"required"`
}

// NodeBatchOpReq 批量操作请求 (如批量安装Docker、加入集群)
type NodeBatchOpReq struct {
	NodeIDs []model.NodeID `json:"node_ids" binding:"required"`
	OpType  string         `json:"op_type" binding:"required"` // install_docker, join_cluster, etc.
	Params  string         `json:"params"`                     // JSON参数
}
