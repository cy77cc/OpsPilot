// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
package ucloud

import (
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/uhost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将 UCLOUD UHost 实例转换为统一的 CloudInstance 模型。
//
// 参数:
//   - inst: UCLOUD SDK 返回的实例信息
//   - region: 地域标识（从请求参数传入，实例数据中无 Region 字段）
//
// 返回:
//   - 统一格式的云实例模型
func ConvertInstance(inst *uhost.UHostInstanceSet, region string) *cloud.CloudInstance {
	return &cloud.CloudInstance{
		InstanceID: inst.UHostId,
		Name:       inst.Name,
		IP:         getPublicIP(inst),
		PrivateIP:  getPrivateIP(inst),
		Region:     region,
		Zone:       inst.Zone,
		Status:     convertStatus(inst.State),
		OS:         inst.OsName,
		CPU:        inst.CPU,
		MemoryMB:   inst.Memory,
		DiskGB:     calculateTotalDisk(inst.DiskSet),
	}
}

// getPublicIP 获取公网 IP 地址。
//
// 从 IPSet 数组中筛选 Type 为 Bgp 或 Internation 的公网 IP。
// UCLOUD 公网类型包括: Bgp（BGP 线路）、Internation（国际线路）。
//
// 返回:
//   - 公网 IP 地址，无则返回空字符串
func getPublicIP(inst *uhost.UHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		// UCLOUD 公网类型: Bgp（BGP 线路）、Internation（国际线路）
		if ip.Type == "Bgp" || ip.Type == "Internation" {
			if ip.IP != "" {
				return ip.IP
			}
		}
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
//
// 从 IPSet 数组中获取第一个 Type 为 Private 的 IP 地址。
//
// 返回:
//   - 内网 IP 地址，无则返回空字符串
func getPrivateIP(inst *uhost.UHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		// UCLOUD 内网类型为 Private
		if ip.Type == "Private" && ip.IP != "" {
			return ip.IP
		}
	}
	return ""
}

// convertStatus 转换实例状态为标准格式。
//
// UCLOUD 状态值:
//   - Running -> running
//   - Stopped -> stopped
//   - Starting -> starting
//   - Stopping -> stopping
//
// 参数:
//   - status: UCLOUD 原始状态值
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

// calculateTotalDisk 计算磁盘总大小。
//
// 遍历所有 DiskSet，累加 Size 字段。
//
// 参数:
//   - disks: 磁盘列表
//
// 返回:
//   - 磁盘总大小（GB）
func calculateTotalDisk(disks []uhost.UHostDiskSet) int {
	var total int
	for _, disk := range disks {
		total += disk.Size
	}
	return total
}
