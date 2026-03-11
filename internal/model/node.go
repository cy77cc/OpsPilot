// Package model 提供数据库模型定义。
//
// 本文件定义节点、SSH 密钥、云账户等基础设施相关的数据模型。
package model

import (
	"time"
)

// NodeID 是节点 ID 类型别名，便于类型安全和代码可读性。
type NodeID uint

// Node 是节点表模型，存储主机/虚拟机信息。
//
// 表名: nodes
// 关联:
//   - SSHKey (多对一，通过 ssh_key_id)
//   - Cluster (多对一，通过 cluster_id，非外键)
//
// 状态说明:
//   - Status: active/inactive/error/maintenance
//   - HealthState: healthy/unhealthy/unknown
//   - Source: manual_ssh/imported/cloud
type Node struct {
	ID                   NodeID     `gorm:"primaryKey;column:id" json:"id"`                                           // 节点 ID
	Name                 string     `gorm:"column:name;type:varchar(64);not null" json:"name"`                        // 节点名称
	Hostname             string     `gorm:"column:hostname;type:varchar(64)" json:"hostname"`                        // 主机名
	Labels               string     `gorm:"column:labels;type:json" json:"labels"`                                    // 标签 (JSON)
	Description          string     `gorm:"column:description;type:varchar(256)" json:"description"`                  // 描述
	IP                   string     `gorm:"column:ip;type:varchar(45);not null" json:"ip"`                           // IP 地址
	Port                 int        `gorm:"column:port;default:22" json:"port"`                                      // SSH 端口
	SSHUser              string     `gorm:"column:ssh_user;type:varchar(64);not null;default:root" json:"ssh_user"` // SSH 用户名
	SSHPassword          string     `gorm:"column:ssh_password;type:varchar(256)" json:"ssh_password"`               // SSH 密码 (加密存储)
	SSHKeyID             *NodeID    `gorm:"column:ssh_key_id" json:"ssh_key_id"`                                     // SSH 密钥 ID
	OS                   string     `gorm:"column:os;type:varchar(64)" json:"os"`                                    // 操作系统
	Arch                 string     `gorm:"column:arch;type:varchar(32)" json:"arch"`                                // 架构: amd64/arm64
	Kernel               string     `gorm:"column:kernel;type:varchar(64)" json:"kernel"`                            // 内核版本
	CpuCores             int        `gorm:"column:cpu_cores" json:"cpu_cores"`                                       // CPU 核数
	MemoryMB             int        `gorm:"column:memory_mb" json:"memory_mb"`                                       // 内存 (MB)
	DiskGB               int        `gorm:"column:disk_gb" json:"disk_gb"`                                           // 磁盘 (GB)
	Status               string     `gorm:"column:status;type:varchar(32);not null" json:"status"`                   // 状态: active/inactive/error/maintenance
	Role                 string     `gorm:"column:role;type:varchar(32)" json:"role"`                                // 角色: master/worker
	ClusterID            uint       `gorm:"column:cluster_id" json:"cluster_id"`                                     // 所属集群 ID
	Source               string     `gorm:"column:source;type:varchar(32);default:manual_ssh" json:"source"`         // 来源: manual_ssh/imported/cloud
	Provider             string     `gorm:"column:provider;type:varchar(32)" json:"provider"`                        // 云厂商: aliyun/aws/tencent
	ProviderID           string     `gorm:"column:provider_instance_id;type:varchar(128)" json:"provider_instance_id"` // 云厂商实例 ID
	ParentHostID         *NodeID    `gorm:"column:parent_host_id" json:"parent_host_id"`                            // 父主机 ID (虚拟机场景)
	HealthState          string     `gorm:"column:health_state;type:varchar(32);default:unknown" json:"health_state"` // 健康状态: healthy/unhealthy/unknown
	MaintenanceReason    string     `gorm:"column:maintenance_reason;type:varchar(512)" json:"maintenance_reason"`   // 维护原因
	MaintenanceBy        uint64     `gorm:"column:maintenance_by;default:0" json:"maintenance_by"`                    // 维护操作人 ID
	MaintenanceStartedAt *time.Time `gorm:"column:maintenance_started_at" json:"maintenance_started_at"`             // 维护开始时间
	MaintenanceUntil     *time.Time `gorm:"column:maintenance_until" json:"maintenance_until"`                       // 维护截止时间
	LastCheckAt          time.Time  `gorm:"column:last_check_at" json:"last_check_at"`                              // 最后检查时间
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`                      // 创建时间
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                      // 更新时间
}

// TableName 返回节点表名。
func (n *Node) TableName() string {
	return "nodes"
}

// SSHKey 是 SSH 密钥表模型，存储 SSH 密钥对。
//
// 表名: ssh_keys
// 用途: 为节点提供免密登录能力
type SSHKey struct {
	ID          NodeID    `gorm:"primaryKey;column:id" json:"id"`                              // 密钥 ID
	Name        string    `gorm:"column:name;type:varchar(64)" json:"name"`                    // 密钥名称
	PublicKey   string    `gorm:"column:public_key;type:text;not null" json:"public_key"`      // 公钥
	PrivateKey  string    `gorm:"column:private_key;type:longtext;not null" json:"private_key"` // 私钥 (加密存储)
	Passphrase  string    `gorm:"column:passphrase;type:varchar(128)" json:"passphrase"`       // 密钥密码 (加密存储)
	Fingerprint string    `gorm:"column:fingerprint;type:varchar(128)" json:"fingerprint"`    // 指纹
	Algorithm   string    `gorm:"column:algorithm;type:varchar(32)" json:"algorithm"`          // 算法: rsa/ed25519
	Encrypted   bool      `gorm:"column:encrypted;default:false" json:"encrypted"`             // 是否已加密
	UsageCount  int       `gorm:"column:usage_count;default:0" json:"usage_count"`             // 使用次数
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`          // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`          // 更新时间
}

// TableName 返回 SSH 密钥表名。
func (s *SSHKey) TableName() string {
	return "ssh_keys"
}

// NodeEvent 是节点事件表模型，记录节点状态变更历史。
//
// 表名: node_events
// 用途: 审计和问题排查
type NodeEvent struct {
	ID        NodeID    `gorm:"primaryKey;column:id" json:"id"`                     // 事件 ID
	NodeID    uint      `gorm:"column:node_id" json:"node_id"`                      // 节点 ID
	Type      string    `gorm:"column:type;type:varchar(32)" json:"type"`           // 事件类型: status_change/health_check
	Message   string    `gorm:"column:message;type:text" json:"message"`            // 事件消息
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"` // 创建时间
}

// TableName 返回节点事件表名。
func (n *NodeEvent) TableName() string {
	return "node_events"
}

// HostCloudAccount 是云厂商账户表模型，存储云 API 凭证。
//
// 表名: host_cloud_accounts
// 用途: 支持从云厂商导入主机
type HostCloudAccount struct {
	ID                 uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                           // 账户 ID
	Provider           string    `gorm:"column:provider;type:varchar(32);not null;index" json:"provider"`        // 云厂商: aliyun/aws/tencent
	AccountName        string    `gorm:"column:account_name;type:varchar(128);not null" json:"account_name"`     // 账户名称
	AccessKeyID        string    `gorm:"column:access_key_id;type:varchar(256);not null" json:"access_key_id"`   // Access Key ID
	AccessKeySecretEnc string    `gorm:"column:access_key_secret_enc;type:longtext;not null" json:"-"`          // Access Key Secret (加密存储)
	RegionDefault      string    `gorm:"column:region_default;type:varchar(64)" json:"region_default"`           // 默认区域
	Status             string    `gorm:"column:status;type:varchar(32);default:active" json:"status"`            // 状态: active/inactive
	CreatedBy          uint64    `gorm:"column:created_by;index" json:"created_by"`                              // 创建人 ID
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                     // 创建时间
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                     // 更新时间
}

// TableName 返回云账户表名。
func (HostCloudAccount) TableName() string { return "host_cloud_accounts" }

// HostImportTask 是主机导入任务表模型，记录从云厂商导入主机的任务。
//
// 表名: host_import_tasks
// 状态: pending/running/success/failed
type HostImportTask struct {
	ID           string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                       // 任务 ID
	Provider     string    `gorm:"column:provider;type:varchar(32);not null;index" json:"provider"`       // 云厂商
	AccountID    uint64    `gorm:"column:account_id;index" json:"account_id"`                            // 云账户 ID
	RequestJSON  string    `gorm:"column:request_json;type:longtext" json:"request_json"`                // 请求参数 (JSON)
	ResultJSON   string    `gorm:"column:result_json;type:longtext" json:"result_json"`                  // 导入结果 (JSON)
	Status       string    `gorm:"column:status;type:varchar(32);index" json:"status"`                   // 状态: pending/running/success/failed
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message"`                  // 错误消息
	CreatedBy    uint64    `gorm:"column:created_by;index" json:"created_by"`                            // 创建人 ID
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                   // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                   // 更新时间
}

// TableName 返回主机导入任务表名。
func (HostImportTask) TableName() string { return "host_import_tasks" }

// HostVirtualizationTask 是主机虚拟化任务表模型，记录在主机上创建虚拟机的任务。
//
// 表名: host_virtualization_tasks
// 用途: 支持在物理机上创建虚拟机
type HostVirtualizationTask struct {
	ID           string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                       // 任务 ID
	HostID       uint64    `gorm:"column:host_id;index" json:"host_id"`                                  // 宿主机 ID
	Hypervisor   string    `gorm:"column:hypervisor;type:varchar(32);not null" json:"hypervisor"`        // 虚拟化类型: kvm/vmware
	RequestJSON  string    `gorm:"column:request_json;type:longtext" json:"request_json"`                // 请求参数 (JSON)
	VMName       string    `gorm:"column:vm_name;type:varchar(128)" json:"vm_name"`                      // 虚拟机名称
	VMIP         string    `gorm:"column:vm_ip;type:varchar(64)" json:"vm_ip"`                           // 虚拟机 IP
	Status       string    `gorm:"column:status;type:varchar(32);index" json:"status"`                   // 状态: pending/running/success/failed
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message"`                  // 错误消息
	CreatedBy    uint64    `gorm:"column:created_by;index" json:"created_by"`                            // 创建人 ID
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                   // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                   // 更新时间
}

// TableName 返回主机虚拟化任务表名。
func (HostVirtualizationTask) TableName() string { return "host_virtualization_tasks" }
