// Package model 提供数据库模型定义。
//
// 本文件定义环境安装相关的数据模型，用于支持运行时环境的引导安装。
package model

import "time"

// EnvironmentInstallJob 是环境安装任务表模型，记录运行时引导安装的长时间运行任务。
//
// 表名: environment_install_jobs
// 用途: 追踪通过 SSH 执行的运行时引导工作 (Kubernetes/Docker Compose)
//
// 状态流转:
//   - queued: 排队中
//   - running: 执行中
//   - succeeded: 成功
//   - failed: 失败
type EnvironmentInstallJob struct {
	ID              string     `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                    // 任务 ID
	Name            string     `gorm:"column:name;type:varchar(128);not null" json:"name"`                 // 任务名称
	RuntimeType     string     `gorm:"column:runtime_type;type:varchar(16);not null;index" json:"runtime_type"` // 运行时类型: k8s/compose
	TargetEnv       string     `gorm:"column:target_env;type:varchar(32);not null;default:'staging'" json:"target_env"` // 目标环境
	TargetID        uint       `gorm:"column:target_id;default:0;index" json:"target_id"`                  // 部署目标 ID
	ClusterID       uint       `gorm:"column:cluster_id;default:0;index" json:"cluster_id"`                // 集群 ID
	Status          string     `gorm:"column:status;type:varchar(32);not null;default:'queued';index" json:"status"` // 状态
	PackageVersion  string     `gorm:"column:package_version;type:varchar(64);default:''" json:"package_version"` // 安装包版本
	PackagePath     string     `gorm:"column:package_path;type:varchar(512);default:''" json:"package_path"` // 安装包路径
	PackageChecksum string     `gorm:"column:package_checksum;type:varchar(128);default:''" json:"package_checksum"` // 安装包校验和
	StartedAt       *time.Time `gorm:"column:started_at" json:"started_at"`                                // 开始时间
	FinishedAt      *time.Time `gorm:"column:finished_at" json:"finished_at"`                              // 结束时间
	ErrorMessage    string     `gorm:"column:error_message;type:text" json:"error_message"`                // 错误信息
	ResultJSON      string     `gorm:"column:result_json;type:longtext" json:"result_json"`                // 结果详情 (JSON)
	CreatedBy       uint64     `gorm:"column:created_by;index" json:"created_by"`                          // 创建人 ID
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`           // 创建时间
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                 // 更新时间
}

// TableName 返回环境安装任务表名。
func (EnvironmentInstallJob) TableName() string { return "environment_install_jobs" }

// EnvironmentInstallJobStep 是环境安装任务步骤表模型，记录每个步骤的日志和计时信息。
//
// 表名: environment_install_job_steps
// 用途: 为引导安装提供诊断支持
//
// 阶段说明:
//   - preflight: 预检查
//   - install: 安装
//   - verify: 验证
//   - rollback: 回滚
type EnvironmentInstallJobStep struct {
	ID           uint       `gorm:"column:id;primaryKey" json:"id"`                              // 步骤 ID
	JobID        string     `gorm:"column:job_id;type:varchar(64);not null;index" json:"job_id"` // 任务 ID
	StepName     string     `gorm:"column:step_name;type:varchar(64);not null" json:"step_name"` // 步骤名称
	Phase        string     `gorm:"column:phase;type:varchar(32);default:''" json:"phase"`      // 阶段: preflight/install/verify/rollback
	Status       string     `gorm:"column:status;type:varchar(32);not null;default:'queued';index" json:"status"` // 状态
	HostID       uint       `gorm:"column:host_id;default:0" json:"host_id"`                    // 主机 ID
	Output       string     `gorm:"column:output;type:text" json:"output"`                      // 输出日志
	ErrorMessage string     `gorm:"column:error_message;type:text" json:"error_message"`        // 错误信息
	StartedAt    *time.Time `gorm:"column:started_at" json:"started_at"`                        // 开始时间
	FinishedAt   *time.Time `gorm:"column:finished_at" json:"finished_at"`                      // 结束时间
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`         // 创建时间
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`         // 更新时间
}

// TableName 返回环境安装任务步骤表名。
func (EnvironmentInstallJobStep) TableName() string { return "environment_install_job_steps" }

// ClusterCredential 是集群凭证表模型，存储平台集群和外部集群的加密凭证信息。
//
// 表名: cluster_credentials
// 用途: 管理访问 Kubernetes 集群的认证材料
//
// 来源说明:
//   - platform_managed: 平台托管集群
//   - external_managed: 外部导入集群
//
// 认证方式:
//   - kubeconfig: 使用 kubeconfig 文件
//   - cert: 使用 CA 证书 + 客户端证书
//   - token: 使用 Bearer Token
type ClusterCredential struct {
	ID              uint       `gorm:"column:id;primaryKey" json:"id"`                                     // 凭证 ID
	Name            string     `gorm:"column:name;type:varchar(128);not null" json:"name"`                 // 凭证名称
	RuntimeType     string     `gorm:"column:runtime_type;type:varchar(16);not null;default:'k8s';index" json:"runtime_type"` // 运行时类型
	Source          string     `gorm:"column:source;type:varchar(32);not null;index" json:"source"`        // 来源: platform_managed/external_managed
	ClusterID       uint       `gorm:"column:cluster_id;default:0;index" json:"cluster_id"`                // 集群 ID
	Endpoint        string     `gorm:"column:endpoint;type:varchar(256);default:''" json:"endpoint"`       // 集群端点
	AuthMethod      string     `gorm:"column:auth_method;type:varchar(32);default:'kubeconfig'" json:"auth_method"` // 认证方式
	KubeconfigEnc   string     `gorm:"column:kubeconfig_enc;type:longtext" json:"-"`                       // 加密的 kubeconfig
	CACertEnc       string     `gorm:"column:ca_cert_enc;type:longtext" json:"-"`                          // 加密的 CA 证书
	CertEnc         string     `gorm:"column:cert_enc;type:longtext" json:"-"`                             // 加密的客户端证书
	KeyEnc          string     `gorm:"column:key_enc;type:longtext" json:"-"`                              // 加密的客户端私钥
	TokenEnc        string     `gorm:"column:token_enc;type:longtext" json:"-"`                            // 加密的 Bearer Token
	MetadataJSON    string     `gorm:"column:metadata_json;type:longtext" json:"metadata_json"`            // 元数据 (JSON)
	Status          string     `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`      // 状态: active/inactive
	LastTestAt      *time.Time `gorm:"column:last_test_at" json:"last_test_at"`                            // 最后测试时间
	LastTestStatus  string     `gorm:"column:last_test_status;type:varchar(32);default:''" json:"last_test_status"` // 最后测试状态
	LastTestMessage string     `gorm:"column:last_test_message;type:varchar(512);default:''" json:"last_test_message"` // 最后测试消息
	CreatedBy       uint64     `gorm:"column:created_by;index" json:"created_by"`                          // 创建人 ID
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`                 // 创建时间
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                 // 更新时间
}

// TableName 返回集群凭证表名。
func (ClusterCredential) TableName() string { return "cluster_credentials" }
