// Package volcengine 提供火山云 ECS 实例查询适配器实现。
package volcengine

import (
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/service/storageebs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"

	"github.com/cy77cc/OpsPilot/internal/modules/host/logic/cloud"
)

// ConvertInstance 将火山云实例转换为统一的 CloudInstance 模型。
//
// 参数:
//   - inst: 火山云 SDK 返回的实例信息
//   - region: 地域标识（从请求参数传入，实例数据中无 Region 字段）
//   - volumeSizes: 云盘大小映射（volumeId -> sizeGB），可选
//
// 返回:
//   - 统一格式的云实例模型
func ConvertInstance(inst *ecs.InstanceForDescribeInstancesOutput, region string, volumeSizes map[string]int) *cloud.CloudInstance {
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
		DiskGB:     calculateTotalDisk(inst.LocalVolumes, inst.Volumes, volumeSizes),
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
// 遍历所有 LocalVolume（本地盘）和 Volumes（云盘）。
// 本地盘直接计算 Size * Count，云盘需要从 volumeSizes 映射获取大小。
//
// 参数:
//   - localVolumes: 本地磁盘列表
//   - volumes: 云盘列表（只有 VolumeId）
//   - volumeSizes: 云盘大小映射（volumeId -> sizeGB）
//
// 返回:
//   - 磁盘总大小（GB）
func calculateTotalDisk(localVolumes []*ecs.LocalVolumeForDescribeInstancesOutput, volumes []*ecs.VolumeForDescribeInstancesOutput, volumeSizes map[string]int) int {
	var total int

	// 计算本地盘大小
	for _, v := range localVolumes {
		if v.Size != nil && v.Count != nil {
			total += int(*v.Size) * int(*v.Count)
		} else if v.Size != nil {
			total += int(*v.Size)
		}
	}

	// 计算云盘大小（从预先查询的映射获取）
	for _, v := range volumes {
		if v.VolumeId != nil {
			if size, ok := volumeSizes[*v.VolumeId]; ok {
				total += size
			}
		}
	}

	return total
}

// ExtractVolumeIds 从实例列表中提取所有云盘 ID。
//
// 参数:
//   - instances: 实例列表
//
// 返回:
//   - 云盘 ID 列表
func ExtractVolumeIds(instances []*ecs.InstanceForDescribeInstancesOutput) []string {
	volumeIds := make([]string, 0)
	for _, inst := range instances {
		for _, v := range inst.Volumes {
			if v.VolumeId != nil && *v.VolumeId != "" {
				volumeIds = append(volumeIds, *v.VolumeId)
			}
		}
	}
	return volumeIds
}

// BuildVolumeSizeMap 从 DescribeVolumes 输出构建大小映射。
//
// 参数:
//   - output: DescribeVolumes API 返回结果
//
// 返回:
//   - 云盘大小映射（volumeId -> sizeGB）
func BuildVolumeSizeMap(output *storageebs.DescribeVolumesOutput) map[string]int {
	sizeMap := make(map[string]int)
	if output == nil || output.Volumes == nil {
		return sizeMap
	}
	for _, v := range output.Volumes {
		if v.VolumeId != nil && v.Size != nil {
			// Size 是 json.Number 类型，需要转换为 int
			size, err := v.Size.Int64()
			if err == nil {
				sizeMap[*v.VolumeId] = int(size)
			}
		}
	}
	return sizeMap
}
