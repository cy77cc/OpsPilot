// Package volcengine 提供火山云 ECS 实例查询适配器实现。
package volcengine

import (
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将火山云实例转换为统一的 CloudInstance 模型。
//
// 参数:
//   - inst: 火山云 SDK 返回的实例信息
//   - region: 地域标识（从请求参数传入，实例数据中无 Region 字段）
//
// 返回:
//   - 统一格式的云实例模型
func ConvertInstance(inst *ecs.InstanceForDescribeInstancesOutput, region string) *cloud.CloudInstance {
	return &cloud.CloudInstance{
		InstanceID: volcengine.StringValue(inst.InstanceId),
		Name:       volcengine.StringValue(inst.InstanceName),
		IP:         getPublicIP(inst),
		PrivateIP:  getPrivateIP(inst),
		Region:     region,
		Zone:       volcengine.StringValue(inst.ZoneId),
		Status:     convertStatus(inst.Status),
		OS:         volcengine.StringValue(inst.OsName),
		CPU:        int(volcengine.Int32Value(inst.Cpus)),
		MemoryMB:   int(volcengine.Int32Value(inst.MemorySize)),
		DiskGB:     calculateTotalDisk(inst.LocalVolumes),
	}
}

// getPublicIP 获取公网 IP 地址。
//
// 优先从 EipAddress（弹性公网 IP）获取。
//
// 返回:
//   - 公网 IP 地址，无则返回空字符串
func getPublicIP(inst *ecs.InstanceForDescribeInstancesOutput) string {
	if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
		return *inst.EipAddress.IpAddress
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
//
// 从 NetworkInterfaces 的 PrimaryIpAddress 获取。
//
// 返回:
//   - 内网 IP 地址，无则返回空字符串
func getPrivateIP(inst *ecs.InstanceForDescribeInstancesOutput) string {
	for _, nic := range inst.NetworkInterfaces {
		if nic.PrimaryIpAddress != nil && *nic.PrimaryIpAddress != "" {
			return *nic.PrimaryIpAddress
		}
	}
	return ""
}

// convertStatus 转换实例状态为标准格式。
//
// 火山云状态值:
//   - Running -> running
//   - Stopped -> stopped
//   - Starting -> starting
//   - Stopping -> stopping
//
// 参数:
//   - status: 火山云原始状态值
//
// 返回:
//   - 标准化的状态值
func convertStatus(status *string) string {
	if status == nil {
		return "unknown"
	}

	switch *status {
	case "Running":
		return "running"
	case "Stopped":
		return "stopped"
	case "Starting":
		return "starting"
	case "Stopping":
		return "stopping"
	default:
		return strings.ToLower(*status)
	}
}

// calculateTotalDisk 计算磁盘总大小。
//
// 遍历所有 LocalVolume，累加 Size * Count。
//
// 参数:
//   - volumes: 本地磁盘列表
//
// 返回:
//   - 磁盘总大小（GB）
func calculateTotalDisk(volumes []*ecs.LocalVolumeForDescribeInstancesOutput) int {
	var total int
	for _, v := range volumes {
		if v.Size != nil && v.Count != nil {
			total += int(*v.Size) * int(*v.Count)
		} else if v.Size != nil {
			total += int(*v.Size)
		}
	}
	return total
}
