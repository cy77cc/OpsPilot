// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
package cloud

// ListInstancesRequest 查询云实例请求参数。
//
// 封装各云厂商通用的查询条件，适配器实现时转换为各自的 SDK 参数。
type ListInstancesRequest struct {
	// AccessKeyID 云 API 访问密钥 ID。
	AccessKeyID string

	// AccessKeySecret 云 API 访问密钥 Secret。
	AccessKeySecret string

	// Region 地域标识。
	//
	// 火山云示例: "cn-beijing"、"cn-shanghai"
	// 阿里云示例: "cn-hangzhou"、"cn-shanghai"
	Region string

	// Keyword 过滤关键词。
	//
	// 用于按实例名称或 IP 地址过滤，适配器可自行实现模糊匹配逻辑。
	Keyword string

	// PageNumber 页码（从 1 开始）。
	PageNumber int

	// PageSize 每页数量。
	PageSize int
}

// CloudInstance 统一的云实例模型。
//
// 从各云厂商实例信息转换而来，用于前端展示和导入处理。
type CloudInstance struct {
	// InstanceID 云厂商实例 ID。
	//
	// 火山云格式: "i-xxxxxxxx"
	// 阿里云格式: "i-xxxxxxxxxxxxxxxxxx"
	InstanceID string `json:"instance_id"`

	// Name 实例名称。
	Name string `json:"name"`

	// IP 公网 IP 地址。
	//
	// 用于 SSH 连接，优先从 EIP 或公网 IP 获取。
	IP string `json:"ip"`

	// PrivateIP 内网 IP 地址。
	//
	// 可选字段，用于 VPC 内部通信。
	PrivateIP string `json:"private_ip"`

	// Region 地域标识。
	Region string `json:"region"`

	// Zone 可用区标识。
	Zone string `json:"zone"`

	// Status 实例状态。
	//
	// 标准化状态值: "running"、"stopped"、"starting"、"stopping"、"unknown"
	Status string `json:"status"`

	// OS 操作系统信息。
	//
	// 如 "Ubuntu 22.04"、"CentOS 7.9"
	OS string `json:"os"`

	// CPU CPU 核心数。
	CPU int `json:"cpu"`

	// MemoryMB 内存大小（MB）。
	MemoryMB int `json:"memory_mb"`

	// DiskGB 磁盘总大小（GB）。
	DiskGB int `json:"disk_gb"`
}

// ProviderInfo 云厂商信息。
//
// 用于前端下拉选项展示和动态云厂商发现。
type ProviderInfo struct {
	// Name 云厂商标识。
	//
	// 如 "volcengine"、"alicloud"、"tencent"
	Name string `json:"name"`

	// DisplayName 云厂商显示名称。
	//
	// 如 "火山云"、"阿里云"、"腾讯云"
	DisplayName string `json:"display_name"`
}
