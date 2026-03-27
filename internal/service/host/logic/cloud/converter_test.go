// Package cloud_test 测试火山云适配器的数据转换逻辑。
package cloud_test

import (
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	volcengineSDK "github.com/volcengine/volcengine-go-sdk/volcengine"

	cloudpkg "github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/volcengine"
)

func TestConvertInstance(t *testing.T) {
	tests := []struct {
		name     string
		inst     *ecs.InstanceForDescribeInstancesOutput
		region   string
		expected cloudpkg.CloudInstance
	}{
		{
			name: "完整实例信息",
			inst: &ecs.InstanceForDescribeInstancesOutput{
				InstanceId:   volcengineSDK.String("i-abc123"),
				InstanceName: volcengineSDK.String("test-instance"),
				ZoneId:       volcengineSDK.String("cn-beijing-a"),
				Status:       volcengineSDK.String("Running"),
				OsName:       volcengineSDK.String("Ubuntu 22.04"),
				Cpus:         volcengineSDK.Int32(4),
				MemorySize:   volcengineSDK.Int32(8192),
				EipAddress: &ecs.EipAddressForDescribeInstancesOutput{
					IpAddress: volcengineSDK.String("1.2.3.4"),
				},
				NetworkInterfaces: []*ecs.NetworkInterfaceForDescribeInstancesOutput{
					{
						PrimaryIpAddress: volcengineSDK.String("10.0.1.100"),
					},
				},
				LocalVolumes: []*ecs.LocalVolumeForDescribeInstancesOutput{
					{
						Size:  volcengineSDK.Int32(100),
						Count: volcengineSDK.Int32(1),
					},
				},
			},
			region: "cn-beijing",
			expected: cloudpkg.CloudInstance{
				InstanceID: "i-abc123",
				Name:       "test-instance",
				IP:         "1.2.3.4",
				PrivateIP:  "10.0.1.100",
				Region:     "cn-beijing",
				Zone:       "cn-beijing-a",
				Status:     "running",
				OS:         "Ubuntu 22.04",
				CPU:        4,
				MemoryMB:   8192,
				DiskGB:     100,
			},
		},
		{
			name: "无公网IP",
			inst: &ecs.InstanceForDescribeInstancesOutput{
				InstanceId:   volcengineSDK.String("i-def456"),
				InstanceName: volcengineSDK.String("internal-instance"),
				ZoneId:       volcengineSDK.String("cn-shanghai-b"),
				Status:       volcengineSDK.String("Stopped"),
				OsName:       volcengineSDK.String("CentOS 7"),
				Cpus:         volcengineSDK.Int32(2),
				MemorySize:   volcengineSDK.Int32(4096),
				NetworkInterfaces: []*ecs.NetworkInterfaceForDescribeInstancesOutput{
					{
						PrimaryIpAddress: volcengineSDK.String("192.168.1.50"),
					},
				},
				LocalVolumes: []*ecs.LocalVolumeForDescribeInstancesOutput{
					{
						Size:  volcengineSDK.Int32(50),
						Count: volcengineSDK.Int32(2),
					},
				},
			},
			region: "cn-shanghai",
			expected: cloudpkg.CloudInstance{
				InstanceID: "i-def456",
				Name:       "internal-instance",
				IP:         "",
				PrivateIP:  "192.168.1.50",
				Region:     "cn-shanghai",
				Zone:       "cn-shanghai-b",
				Status:     "stopped",
				OS:         "CentOS 7",
				CPU:        2,
				MemoryMB:   4096,
				DiskGB:     100,
			},
		},
		{
			name: "空实例信息",
			inst: &ecs.InstanceForDescribeInstancesOutput{
				InstanceId: volcengineSDK.String("i-empty"),
				Status:     volcengineSDK.String("Unknown"),
			},
			region: "cn-guangzhou",
			expected: cloudpkg.CloudInstance{
				InstanceID: "i-empty",
				IP:         "",
				PrivateIP:  "",
				Region:     "cn-guangzhou",
				Zone:       "",
				Status:     "unknown",
				CPU:        0,
				MemoryMB:   0,
				DiskGB:     0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := volcengine.ConvertInstance(tt.inst, tt.region)

			if result.InstanceID != tt.expected.InstanceID {
				t.Errorf("InstanceID = %s, want %s", result.InstanceID, tt.expected.InstanceID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name = %s, want %s", result.Name, tt.expected.Name)
			}
			if result.IP != tt.expected.IP {
				t.Errorf("IP = %s, want %s", result.IP, tt.expected.IP)
			}
			if result.PrivateIP != tt.expected.PrivateIP {
				t.Errorf("PrivateIP = %s, want %s", result.PrivateIP, tt.expected.PrivateIP)
			}
			if result.Region != tt.expected.Region {
				t.Errorf("Region = %s, want %s", result.Region, tt.expected.Region)
			}
			if result.Zone != tt.expected.Zone {
				t.Errorf("Zone = %s, want %s", result.Zone, tt.expected.Zone)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Status = %s, want %s", result.Status, tt.expected.Status)
			}
			if result.OS != tt.expected.OS {
				t.Errorf("OS = %s, want %s", result.OS, tt.expected.OS)
			}
			if result.CPU != tt.expected.CPU {
				t.Errorf("CPU = %d, want %d", result.CPU, tt.expected.CPU)
			}
			if result.MemoryMB != tt.expected.MemoryMB {
				t.Errorf("MemoryMB = %d, want %d", result.MemoryMB, tt.expected.MemoryMB)
			}
			if result.DiskGB != tt.expected.DiskGB {
				t.Errorf("DiskGB = %d, want %d", result.DiskGB, tt.expected.DiskGB)
			}
		})
	}
}
