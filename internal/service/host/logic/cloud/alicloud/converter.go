// Package alicloud 提供阿里云 ECS 实例查询适配器实现。
package alicloud

import (
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/client"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Instance 类型别名，简化类型名称。
type Instance = ecs.DescribeInstancesResponseInstancesInstance

// ConvertInstance 将阿里云 ECS 实例转换为统一的 CloudInstance 模型。
//
// 参数:
//   - inst: 阿里云 SDK 返回的实例信息
//
// 返回:
//   - 统一格式的云实例模型
func ConvertInstance(inst *Instance) *cloud.CloudInstance {
	return &cloud.CloudInstance{
		InstanceID: ptrToString(inst.InstanceId),
		Name:       ptrToString(inst.InstanceName),
		IP:         getPublicIP(inst),
		PrivateIP:  getPrivateIP(inst),
		Region:     ptrToString(inst.RegionId),
		Zone:       ptrToString(inst.ZoneId),
		Status:     convertStatus(ptrToString(inst.Status)),
		OS:         ptrToString(inst.OSName),
		CPU:        ptrToInt(inst.Cpu),
		MemoryMB:   ptrToInt(inst.Memory) * 1024, // GB -> MB
		DiskGB:     0,                            // 磁盘信息需单独查询 DescribeInstanceDisks
	}
}

// ptrToString 安全地解引用字符串指针。
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrToInt 安全地解引用 int 指针。
func ptrToInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// getPublicIP 获取公网 IP 地址。
//
// 优先级：EIP > 公网 IP
// EIP 是用户绑定的弹性公网 IP，通常是主要的外网访问入口。
//
// 返回:
//   - 公网 IP 地址，无则返回空字符串
func getPublicIP(inst *Instance) string {
	// 优先返回 EIP（弹性公网 IP）
	if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil && *inst.EipAddress.IpAddress != "" {
		return *inst.EipAddress.IpAddress
	}
	// 其次返回公网 IP
	if inst.PublicIpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 && inst.PublicIpAddress.IpAddress[0] != nil {
		return *inst.PublicIpAddress.IpAddress[0]
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
//
// 从 VpcAttributes.PrivateIpAddress 获取 VPC 内网 IP。
//
// 返回:
//   - 内网 IP 地址，无则返回空字符串
func getPrivateIP(inst *Instance) string {
	if inst.VpcAttributes != nil &&
		inst.VpcAttributes.PrivateIpAddress != nil &&
		len(inst.VpcAttributes.PrivateIpAddress.IpAddress) > 0 &&
		inst.VpcAttributes.PrivateIpAddress.IpAddress[0] != nil {
		return *inst.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}
	return ""
}

// convertStatus 转换实例状态为标准格式。
//
// 阿里云状态值:
//   - Running -> running
//   - Stopped -> stopped
//   - Starting -> starting
//   - Stopping -> stopping
//
// 参数:
//   - status: 阿里云原始状态值
//
// 返回:
//   - 标准化的状态值
func convertStatus(status string) string {
	switch status {
	case "Running":
		return "running"
	case "Stopped":
		return "stopped"
	case "Starting":
		return "starting"
	case "Stopping":
		return "stopping"
	default:
		return strings.ToLower(status)
	}
}
