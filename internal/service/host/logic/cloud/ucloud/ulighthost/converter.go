package ulighthost

import (
	"log/slog"
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将 ULightHost 实例转换为统一的 CloudInstance 模型。
func ConvertInstance(inst ulighthost.ULHostInstanceSet, region string) *cloud.CloudInstance {
	publicIP := getPublicIP(inst)
	privateIP := getPrivateIP(inst)

	// 调试日志
	slog.Debug("ULightHost instance IP info",
		"instanceId", inst.ULHostId,
		"name", inst.Name,
		"ipSetCount", len(inst.IPSet),
		"publicIP", publicIP,
		"privateIP", privateIP,
	)

	return &cloud.CloudInstance{
		InstanceID: inst.ULHostId,
		Name:       inst.Name,
		IP:         publicIP,
		PrivateIP:  privateIP,
		Region:     region,
		Zone:       inst.Zone,
		Status:     convertStatus(inst.State),
		OS:         inst.ImageName,
		CPU:        inst.CPU,
		MemoryMB:   inst.Memory,
		DiskGB:     calculateTotalDisk(inst.DiskSet),
	}
}

// getPublicIP 获取公网 IP 地址。
func getPublicIP(inst ulighthost.ULHostInstanceSet) string {
	for i, ip := range inst.IPSet {
		slog.Debug("ULightHost IPSet", "index", i, "type", ip.Type, "ip", ip.IP)
		// 匹配公网 IP 类型：Bgp、Internation、International
		ipType := strings.ToLower(ip.Type)
		if ipType == "bgp" || ipType == "internation" || ipType == "international" || strings.HasPrefix(ipType, "bgp") {
			if ip.IP != "" {
				return ip.IP
			}
		}
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
func getPrivateIP(inst ulighthost.ULHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		if strings.ToLower(ip.Type) == "private" && ip.IP != "" {
			return ip.IP
		}
	}
	return ""
}

// convertStatus 转换实例状态为标准格式。
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
	case "Rebooting":
		return "rebooting"
	default:
		return strings.ToLower(status)
	}
}

// calculateTotalDisk 计算磁盘总大小。
func calculateTotalDisk(disks []ulighthost.ULHostDiskSet) int {
	var total int
	for _, disk := range disks {
		total += disk.Size
	}
	return total
}
