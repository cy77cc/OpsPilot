# 多云厂商适配器扩展实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 扩展云厂商适配器系统，支持阿里云和 UCLOUD，包含重试机制和安全脱敏。

**Architecture:** 基于现有 CloudProvider 接口，新增 retry.go 通用重试机制，实现阿里云和 UCLOUD 两个适配器，直接使用 API 返回的地域名称（无需硬编码）。

**Tech Stack:** Go 1.26, Gin, 阿里云 SDK v9, UCLOUD SDK, 泛型重试机制

**Spec:** `docs/superpowers/specs/2026-03-20-multi-cloud-provider-design.md`

---

## Chunk 1: 基础能力 - 接口扩展与重试机制

### Task 1: 添加 ProviderCapabilities 类型

**Files:**
- Modify: `internal/service/host/logic/cloud/types.go`

- [ ] **Step 1: 在 types.go 中添加 ProviderCapabilities 类型**

在 `types.go` 文件末尾添加：

```go
// ProviderCapabilities 云厂商能力标识。
//
// 用于标识各云厂商支持的功能差异，前端可根据能力调整交互逻辑。
type ProviderCapabilities struct {
	// DynamicRegions 是否支持动态查询地域。
	//
	// 所有厂商均支持动态查询，此字段保留用于未来扩展。
	DynamicRegions bool `json:"dynamic_regions"`
}
```

- [ ] **Step 2: 在 types.go 中为 ListInstancesRequest 添加 NextToken 字段**

在 `ListInstancesRequest` 结构体中添加：

```go
// NextToken 分页令牌（可选）。
//
// 用于大数据量遍历，支持游标分页。
// 阿里云 V9 SDK 推荐使用 NextToken 替代 PageNumber。
// 如果提供 NextToken，优先使用游标分页。
NextToken string
```

- [ ] **Step 3: 提交基础类型扩展**

```bash
git add internal/service/host/logic/cloud/types.go
git commit -m "feat(cloud): add ProviderCapabilities and NextToken to types"
```

---

### Task 2: 更新火山云适配器实现 Capabilities

**Files:**
- Modify: `internal/service/host/logic/cloud/volcengine/provider.go`

- [ ] **Step 1: 在火山云 Provider 中实现 Capabilities 方法**

在 `volcengine/provider.go` 的 `Provider` 结构体方法区域添加：

```go
// Capabilities 返回火山云能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// init 自注册到全局 Registry。
func init() {
	cloud.Register(New())
}
```

- [ ] **Step 2: 提交火山云适配器更新**

```bash
git add internal/service/host/logic/cloud/volcengine/provider.go
git commit -m "feat(cloud): implement Capabilities for volcengine provider"
```

---

### Task 3: 更新 MockProvider 实现 Capabilities

**Files:**
- Modify: `internal/service/host/logic/cloud/mock_provider.go`

- [ ] **Step 1: 在 MockProvider 中实现 Capabilities 方法**

在 `mock_provider.go` 的 `MockProvider` 结构体方法区域添加：

```go
// Capabilities 返回 Mock 能力标识。
func (m *MockProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		DynamicRegions: true,
	}
}
```

- [ ] **Step 2: 提交 MockProvider 更新**

```bash
git add internal/service/host/logic/cloud/mock_provider.go
git commit -m "feat(cloud): implement Capabilities for MockProvider"
```

---

### Task 4: 扩展 CloudProvider 接口

**Files:**
- Modify: `internal/service/host/logic/cloud/provider.go`

- [ ] **Step 1: 在 CloudProvider 接口中添加 Capabilities 方法**

在 `provider.go` 的 `CloudProvider` 接口中，在 `DisplayName()` 方法后添加：

```go
// Capabilities 返回云厂商能力标识。
//
// 用于查询厂商支持的功能特性，如是否支持动态查询地域。
Capabilities() ProviderCapabilities
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/service/host/logic/cloud/...
```

Expected: 编译成功（所有实现已就位）

- [ ] **Step 3: 提交接口扩展**

```bash
git add internal/service/host/logic/cloud/provider.go
git commit -m "feat(cloud): add Capabilities method to CloudProvider interface"
```

---

### Task 5: 实现通用重试机制

**Files:**
- Create: `internal/service/host/logic/cloud/retry.go`
- Create: `internal/service/host/logic/cloud/retry_test.go`

- [ ] **Step 1: 编写重试机制测试**

创建 `internal/service/host/logic/cloud/retry_test.go`：

```go
package cloud

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDoWithRetry_Success(t *testing.T) {
	callCount := 0
	result, err := DoWithRetry(context.Background(), "alicloud", DefaultRetryConfig, "test", func() (string, error) {
		callCount++
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount, "should not retry on success")
}

func TestDoWithRetry_RetryableError(t *testing.T) {
	callCount := 0

	result, err := DoWithRetry(context.Background(), "alicloud", RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}, "test", func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("Throttling: rate exceeded")
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount, "should retry on throttling")
}

func TestDoWithRetry_NonRetryableError(t *testing.T) {
	callCount := 0

	result, err := DoWithRetry(context.Background(), "alicloud", DefaultRetryConfig, "test", func() (string, error) {
		callCount++
		return "", errors.New("InvalidAccessKeyId.NotFound")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAccessKeyId.NotFound")
	assert.Equal(t, 1, callCount, "should not retry on non-retryable error")
}

func TestDoWithRetry_MaxRetriesExceeded(t *testing.T) {
	callCount := 0

	result, err := DoWithRetry(context.Background(), "alicloud", RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}, "test", func() (string, error) {
		callCount++
		return "", errors.New("Throttling: rate exceeded")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Throttling")
	assert.Equal(t, 3, callCount, "should try MaxRetries+1 times")
	assert.Empty(t, result)
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	callCount := 0
	result, err := DoWithRetry(ctx, "alicloud", DefaultRetryConfig, "test", func() (string, error) {
		callCount++
		return "", errors.New("Throttling")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 0, callCount, "should not call function when context is cancelled")
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		provider string
		err      error
		expected bool
	}{
		{"alicloud", errors.New("Throttling: rate exceeded"), true},
		{"alicloud", errors.New("ServiceUnavailable"), true},
		{"alicloud", errors.New("InternalError"), true},
		{"alicloud", errors.New("InvalidAccessKeyId"), false},
		{"volcengine", errors.New("RequestLimitExceeded"), true},
		{"ucloud", errors.New("RetCode: 172"), false},
		{"ucloud", errors.New("error code 172"), true},
		{"unknown", errors.New("Throttling"), false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.err.Error(), func(t *testing.T) {
			result := isRetryableError(tt.provider, tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/service/host/logic/cloud/... -run TestDoWithRetry -v
```

Expected: FAIL (retry.go not exists)

- [ ] **Step 3: 实现重试机制**

创建 `internal/service/host/logic/cloud/retry.go`：

```go
// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
package cloud

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryConfig 重试配置。
type RetryConfig struct {
	// MaxRetries 最大重试次数。
	MaxRetries int

	// InitialDelay 初始延迟。
	InitialDelay time.Duration

	// MaxDelay 最大延迟。
	MaxDelay time.Duration

	// Multiplier 退避乘数。
	Multiplier float64
}

// DefaultRetryConfig 默认重试配置。
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 500 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2.0,
}

// RetryableErrors 可重试的错误码映射。
//
// 键为云厂商标识，值为可重试的错误码列表。
var RetryableErrors = map[string][]string{
	"alicloud":   {"Throttling", "ServiceUnavailable", "InternalError"},
	"volcengine": {"RequestLimitExceeded", "ServiceUnavailable"},
	"ucloud":     {"172", "5000"}, // 172: 请求频率限制, 5000: 服务内部错误
}

// DoWithRetry 执行带重试的操作。
//
// 参数:
//   - ctx: 上下文
//   - provider: 云厂商标识
//   - config: 重试配置
//   - op: 操作名称（用于日志）
//   - fn: 实际操作函数
//
// 返回:
//   - 成功返回结果
//   - 重试耗尽后返回最后一次错误
func DoWithRetry[T any](ctx context.Context, provider string, config RetryConfig, op string, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error
	delay := config.InitialDelay

	for i := 0; i <= config.MaxRetries; i++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		// 检查是否可重试
		if !isRetryableError(provider, lastErr) {
			return result, lastErr
		}

		// 最后一次重试失败，不再等待
		if i == config.MaxRetries {
			break
		}

		// 等待后重试（使用 time.NewTimer 避免内存泄露）
		logrus.Debugf("云 API 请求失败，%s 后重试 (第 %d 次): %s", delay, i+1, op)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, ctx.Err()
		case <-timer.C:
		}

		// 指数退避
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return result, lastErr
}

// isRetryableError 检查错误是否可重试。
func isRetryableError(provider string, err error) bool {
	if err == nil {
		return false
	}

	codes, ok := RetryableErrors[provider]
	if !ok {
		return false
	}

	errStr := err.Error()
	for _, code := range codes {
		if strings.Contains(errStr, code) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/service/host/logic/cloud/... -run TestDoWithRetry -v
```

Expected: PASS

- [ ] **Step 5: 提交重试机制**

```bash
git add internal/service/host/logic/cloud/retry.go internal/service/host/logic/cloud/retry_test.go
git commit -m "feat(cloud): add generic retry mechanism with exponential backoff"
```

---

## Chunk 2: 阿里云适配器

### Task 6: 添加阿里云 SDK 依赖

- [ ] **Step 1: 安装阿里云 ECS SDK v9**

```bash
go get github.com/alibabacloud-go/ecs-20140526/v9
```

- [ ] **Step 2: 验证依赖安装成功**

```bash
go mod tidy
grep "alibabacloud-go/ecs" go.mod
```

Expected: 显示 `github.com/alibabacloud-go/ecs-20140526 v9.x.x`

---

### Task 7: 实现阿里云客户端封装

**Files:**
- Create: `internal/service/host/logic/cloud/alicloud/client.go`

- [ ] **Step 1: 创建阿里云客户端文件**

创建 `internal/service/host/logic/cloud/alicloud/client.go`：

```go
// Package alicloud 提供阿里云 ECS 实例查询适配器实现。
//
// 本包封装阿里云 Go SDK，实现 cloud.CloudProvider 接口，
// 支持阿里云 ECS 实例的查询和导入功能。
package alicloud

import (
	"context"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v9/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Client 阿里云 ECS 客户端封装。
//
// 封装阿里云 SDK 的 ECS 服务客户端，提供简化的实例查询接口。
type Client struct {
	ecs *ecs.Client
	ak  string // 保存用于日志脱敏检查
}

// NewClient 创建阿里云 ECS 客户端。
//
// 参数:
//   - ak: AccessKey ID
//   - sk: AccessKey Secret
//   - region: 地域标识（如 "cn-hangzhou"、"cn-shanghai"）
//
// 注意：查询地域列表（DescribeRegions）时，需使用默认地域如 cn-hangzhou。
func NewClient(ak, sk, region string) (*Client, error) {
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("阿里云 AccessKey ID 和 Secret 不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("阿里云地域不能为空，如 cn-hangzhou、cn-shanghai")
	}

	endpoint := fmt.Sprintf("ecs.%s.aliyuncs.com", region)
	config := &openapi.Config{
		AccessKeyId:     tea.String(ak),
		AccessKeySecret: tea.String(sk),
		RegionId:        tea.String(region),
		Endpoint:        tea.String(endpoint),
	}
	client, err := ecs.NewClient(config)
	if err != nil {
		// 注意：错误信息中不包含 sk，避免泄露
		return nil, fmt.Errorf("创建阿里云客户端失败: %w", err)
	}
	return &Client{ecs: client, ak: ak}, nil
}

// DescribeInstances 查询 ECS 实例列表（带重试）。
func (c *Client) DescribeInstances(ctx context.Context, req *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeInstances", func() (*ecs.DescribeInstancesResponse, error) {
		return c.ecs.DescribeInstancesWithOptions(req, nil)
	})
}

// DescribeRegions 查询地域列表（带重试）。
//
// 返回结果包含 LocalName 字段（中文名称），无需硬编码映射。
func (c *Client) DescribeRegions(ctx context.Context, req *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error) {
	return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeRegions", func() (*ecs.DescribeRegionsResponse, error) {
		return c.ecs.DescribeRegionsWithOptions(req, nil)
	})
}

// DescribeZones 查询可用区列表（带重试）。
func (c *Client) DescribeZones(ctx context.Context, req *ecs.DescribeZonesRequest) (*ecs.DescribeZonesResponse, error) {
	return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeZones", func() (*ecs.DescribeZonesResponse, error) {
		return c.ecs.DescribeZonesWithOptions(req, nil)
	})
}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/service/host/logic/cloud/alicloud/...
```

Expected: 无错误

---

### Task 8: 实现阿里云数据转换器

**Files:**
- Create: `internal/service/host/logic/cloud/alicloud/converter.go`
- Create: `internal/service/host/logic/cloud/alicloud/converter_test.go`

- [ ] **Step 1: 编写转换器测试**

创建 `internal/service/host/logic/cloud/alicloud/converter_test.go`：

```go
package alicloud

import (
	"testing"

	ecs "github.com/alibabacloud-go/ecs-20140526/v9/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

func TestConvertInstance(t *testing.T) {
	instance := &ecs.Instance{
		InstanceId:   tea.String("i-abcdefgh12345678"),
		InstanceName: tea.String("test-instance"),
		RegionId:     tea.String("cn-hangzhou"),
		ZoneId:       tea.String("cn-hangzhou-h"),
		Status:       tea.String("Running"),
		OSName:       tea.String("Ubuntu 22.04"),
		Cpu:          tea.Int32(4),
		Memory:       tea.Int32(8), // GB
	}

	result := ConvertInstance(instance)

	assert.Equal(t, "i-abcdefgh12345678", result.InstanceID)
	assert.Equal(t, "test-instance", result.Name)
	assert.Equal(t, "cn-hangzhou", result.Region)
	assert.Equal(t, "cn-hangzhou-h", result.Zone)
	assert.Equal(t, "running", result.Status)
	assert.Equal(t, "Ubuntu 22.04", result.OS)
	assert.Equal(t, 4, result.CPU)
	assert.Equal(t, 8192, result.MemoryMB) // 8GB -> 8192MB
}

func TestGetPublicIP_EIP(t *testing.T) {
	instance := &ecs.Instance{
		EipAddress: &ecs.EipAddress{
			IpAddress: tea.String("1.2.3.4"),
		},
		PublicIpAddress: &ecs.PublicIpAddress{
			IpAddress: tea.StringSlice([]string{"5.6.7.8"}),
		},
	}

	ip := getPublicIP(instance)
	assert.Equal(t, "1.2.3.4", ip, "should prefer EIP over public IP")
}

func TestGetPublicIP_PublicIP(t *testing.T) {
	instance := &ecs.Instance{
		PublicIpAddress: &ecs.PublicIpAddress{
			IpAddress: tea.StringSlice([]string{"5.6.7.8"}),
		},
	}

	ip := getPublicIP(instance)
	assert.Equal(t, "5.6.7.8", ip, "should return public IP when no EIP")
}

func TestGetPublicIP_NoIP(t *testing.T) {
	instance := &ecs.Instance{}

	ip := getPublicIP(instance)
	assert.Empty(t, ip, "should return empty string when no public IP")
}

func TestGetPrivateIP(t *testing.T) {
	instance := &ecs.Instance{
		VpcAttributes: &ecs.VpcAttributes{
			PrivateIpAddress: &ecs.PrivateIpAddress{
				IpAddress: tea.StringSlice([]string{"10.0.0.1"}),
			},
		},
	}

	ip := getPrivateIP(instance)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestConvertStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Running", "running"},
		{"Stopped", "stopped"},
		{"Starting", "starting"},
		{"Stopping", "stopping"},
		{"Unknown", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertStatus(&tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateDiskSize(t *testing.T) {
	instance := &ecs.Instance{
		SystemDisk: &ecs.SystemDisk{
			Size: tea.Int32(40),
		},
		DataDisks: &ecs.DataDisks{
			Disk: []*ecs.DataDisk{
				{Size: tea.Int32(100)},
				{Size: tea.Int32(200)},
			},
		},
	}

	size := calculateDiskSize(instance)
	assert.Equal(t, 340, size) // 40 + 100 + 200
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/service/host/logic/cloud/alicloud/... -v
```

Expected: FAIL (converter.go not exists)

- [ ] **Step 3: 实现转换器**

创建 `internal/service/host/logic/cloud/alicloud/converter.go`：

```go
// Package alicloud 提供阿里云 ECS 实例查询适配器实现。
package alicloud

import (
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v9/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将阿里云实例转换为统一的 CloudInstance 模型。
//
// 参数:
//   - inst: 阿里云 SDK 返回的实例信息
//
// 返回:
//   - 统一格式的云实例模型
func ConvertInstance(inst *ecs.Instance) *cloud.CloudInstance {
	return &cloud.CloudInstance{
		InstanceID: tea.StringValue(inst.InstanceId),
		Name:       tea.StringValue(inst.InstanceName),
		IP:         getPublicIP(inst),
		PrivateIP:  getPrivateIP(inst),
		Region:     tea.StringValue(inst.RegionId),
		Zone:       tea.StringValue(inst.ZoneId),
		Status:     convertStatus(inst.Status),
		OS:         tea.StringValue(inst.OSName),
		CPU:        int(tea.Int32Value(inst.Cpu)),
		MemoryMB:   int(tea.Int32Value(inst.Memory)) * 1024, // GB -> MB
		DiskGB:     calculateDiskSize(inst),
	}
}

// getPublicIP 获取公网 IP 地址。
//
// 优先级：EIP > 公网 IP
// EIP 是用户绑定的弹性公网 IP，通常是主要的外网访问入口。
func getPublicIP(inst *ecs.Instance) string {
	// 优先返回 EIP（弹性公网 IP）
	if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
		if ip := tea.StringValue(inst.EipAddress.IpAddress); ip != "" {
			return ip
		}
	}
	// 其次返回公网 IP
	if inst.PublicIpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 {
		return tea.StringValue(inst.PublicIpAddress.IpAddress[0])
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
func getPrivateIP(inst *ecs.Instance) string {
	if inst.VpcAttributes != nil && inst.VpcAttributes.PrivateIpAddress != nil {
		if len(inst.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
			return tea.StringValue(inst.VpcAttributes.PrivateIpAddress.IpAddress[0])
		}
	}
	return ""
}

// convertStatus 转换实例状态为标准格式。
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

// calculateDiskSize 计算磁盘总大小。
func calculateDiskSize(inst *ecs.Instance) int {
	var total int

	// 系统盘
	if inst.SystemDisk != nil && inst.SystemDisk.Size != nil {
		total += int(tea.Int32Value(inst.SystemDisk.Size))
	}

	// 数据盘
	if inst.DataDisks != nil && inst.DataDisks.Disk != nil {
		for _, disk := range inst.DataDisks.Disk {
			if disk.Size != nil {
				total += int(tea.Int32Value(disk.Size))
			}
		}
	}

	return total
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/service/host/logic/cloud/alicloud/... -v
```

Expected: PASS

- [ ] **Step 5: 提交阿里云转换器**

```bash
git add internal/service/host/logic/cloud/alicloud/
git commit -m "feat(cloud): add alicloud converter with EIP priority"
```

---

### Task 9: 实现阿里云 Provider

**Files:**
- Create: `internal/service/host/logic/cloud/alicloud/provider.go`

- [ ] **Step 1: 创建 Provider 实现**

创建 `internal/service/host/logic/cloud/alicloud/provider.go`：

```go
// Package alicloud 提供阿里云 ECS 实例查询适配器实现。
package alicloud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v9/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Provider 阿里云适配器。
//
// 实现 cloud.CloudProvider 接口，提供阿里云 ECS 实例查询能力。
type Provider struct{}

// New 创建阿里云适配器实例。
func New() *Provider {
	return &Provider{}
}

// Name 返回云厂商标识。
func (p *Provider) Name() string {
	return "alicloud"
}

// DisplayName 返回云厂商显示名称。
func (p *Provider) DisplayName() string {
	return "阿里云"
}

// Capabilities 返回阿里云能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证阿里云凭证是否有效。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	// 通过查询实例验证凭证，限制返回 1 条
	_, err = client.DescribeInstances(ctx, &ecs.DescribeInstancesRequest{
		MaxResults: tea.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("阿里云凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询阿里云 ECS 实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	input := &ecs.DescribeInstancesRequest{
		RegionId: tea.String(req.Region),
	}

	// 可用区过滤
	if req.Zone != "" {
		input.ZoneId = tea.StringSlice([]string{req.Zone})
	}

	// 分页参数
	pageSize := int32(100)
	if req.PageSize > 0 && req.PageSize <= 100 {
		pageSize = int32(req.PageSize)
	}
	input.MaxResults = tea.Int32(pageSize)

	// 游标分页优先
	if req.NextToken != "" {
		input.NextToken = tea.String(req.NextToken)
	} else if req.PageNumber > 1 {
		// 页码分页
		input.PageNumber = tea.Int32(int32(req.PageNumber))
	}

	// 调用 API
	output, err := client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询阿里云实例失败: %w", p.wrapError(err))
	}

	// 转换实例数据
	instances := make([]cloud.CloudInstance, 0, len(output.Body.Instances.Instance))
	for _, inst := range output.Body.Instances.Instance {
		converted := ConvertInstance(inst)

		// 关键词过滤
		if req.Keyword != "" {
			if !matchKeyword(converted, req.Keyword) {
				continue
			}
		}

		instances = append(instances, *converted)
	}

	return instances, nil
}

// ListRegions 查询阿里云支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	// 使用默认地域创建客户端
	client, err := NewClient(ak, sk, "cn-hangzhou")
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeRegions(ctx, &ecs.DescribeRegionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("查询阿里云地域失败: %w", p.wrapError(err))
	}

	regions := make([]cloud.Region, 0, len(output.Body.Regions.Region))
	for _, r := range output.Body.Regions.Region {
		regions = append(regions, cloud.Region{
			RegionId:  tea.StringValue(r.RegionId),
			LocalName: tea.StringValue(r.LocalName),
		})
	}
	return regions, nil
}

// ListZones 查询阿里云指定地域的可用区列表。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	client, err := NewClient(ak, sk, region)
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeZones(ctx, &ecs.DescribeZonesRequest{
		RegionId: tea.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("查询阿里云可用区失败: %w", p.wrapError(err))
	}

	zones := make([]cloud.Zone, 0, len(output.Body.Zones.Zone))
	for _, z := range output.Body.Zones.Zone {
		zones = append(zones, cloud.Zone{
			ZoneId:    tea.StringValue(z.ZoneId),
			LocalName: tea.StringValue(z.LocalName),
		})
	}
	return zones, nil
}

// matchKeyword 检查实例是否匹配关键词。
func matchKeyword(inst *cloud.CloudInstance, keyword string) bool {
	kw := strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(inst.Name), kw) ||
		strings.Contains(strings.ToLower(inst.InstanceID), kw) ||
		strings.Contains(inst.IP, kw) ||
		strings.Contains(inst.PrivateIP, kw)
}

// wrapError 包装阿里云错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	var teaErr = &tea.SDKError{}
	if errors.As(err, &teaErr) {
		code := teaErr.Code
		switch code {
		case "InvalidAccessKeyId.NotFound":
			return fmt.Errorf("AccessKey ID 不存在")
		case "SignatureDoesNotMatch":
			return fmt.Errorf("签名验证失败，请检查 AccessKey Secret")
		case "InvalidRegionId":
			return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-hangzhou、cn-shanghai")
		case "UnauthorizedOperation":
			return fmt.Errorf("无权限执行此操作，请检查 AccessKey 是否有 ECS 权限")
		case "MissingParameter":
			return fmt.Errorf("缺少必要参数: %s", teaErr.Message)
		case "Throttling":
			return fmt.Errorf("请求过于频繁，已重试多次仍失败，请稍后重试")
		case "ServiceUnavailable":
			return fmt.Errorf("服务暂时不可用，请稍后重试")
		}
		return fmt.Errorf("[阿里云][%s] %s", code, teaErr.Message)
	}

	return fmt.Errorf("[阿里云] %w", err)
}

// init 自注册到全局 Registry。
//
// 注意：这是自注册模式，避免循环依赖。
// cloud 包不能导入 alicloud 包，否则会形成循环：
// cloud -> alicloud -> cloud
func init() {
	cloud.Register(New())
}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/service/host/logic/cloud/alicloud/...
```

Expected: 无错误

- [ ] **Step 3: 提交阿里云 Provider**

```bash
git add internal/service/host/logic/cloud/alicloud/provider.go
git commit -m "feat(cloud): implement alicloud CloudProvider with all interface methods"
```

---

## Chunk 3: UCLOUD 适配器

### Task 10: 添加 UCLOUD SDK 依赖

- [ ] **Step 1: 安装 UCLOUD SDK**

```bash
go get github.com/ucloud/ucloud-sdk-go
```

- [ ] **Step 2: 验证依赖安装成功**

```bash
go mod tidy
grep "ucloud-sdk-go" go.mod
```

Expected: 显示 `github.com/ucloud/ucloud-sdk-go v0.x.x`

---

### Task 11: 实现 UCLOUD 客户端封装

**Files:**
- Create: `internal/service/host/logic/cloud/ucloud/client.go`

- [ ] **Step 1: 创建 UCLOUD 客户端文件**

创建 `internal/service/host/logic/cloud/ucloud/client.go`：

```go
// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
//
// 本包封装 UCLOUD Go SDK，实现 cloud.CloudProvider 接口，
// 支持 UCLOUD UHost 实例的查询和导入功能。
package ucloud

import (
	"context"
	"fmt"

	"github.com/ucloud/ucloud-sdk-go/services/uhost"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Client UCLOUD UHost 客户端封装。
type Client struct {
	uhost *uhost.UHostClient
	ak    string
}

// NewClient 创建 UCLOUD UHost 客户端。
func NewClient(ak, sk, region string) (*Client, error) {
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("UCLOUD AccessKey ID 和 Secret 不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("UCLOUD 地域不能为空，如 cn-bj2、cn-sh2")
	}

	config := ucloud.NewConfig()
	config.Region = region
	credential := auth.NewKeyPairCredential(ak, sk)
	return &Client{
		uhost: uhost.NewClient(&config, credential),
		ak:    ak,
	}, nil
}

// DescribeUHostInstance 查询 UHost 实例列表（带重试）。
func (c *Client) DescribeUHostInstance(ctx context.Context, req *uhost.DescribeUHostInstanceRequest) (*uhost.DescribeUHostInstanceResponse, error) {
	return cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "DescribeUHostInstance", func() (*uhost.DescribeUHostInstanceResponse, error) {
		return c.uhost.DescribeUHostInstance(req)
	})
}

// GetRegion 查询地域和可用区列表（带重试）。
func (c *Client) GetRegion(ctx context.Context, req *uhost.GetRegionRequest) (*uhost.GetRegionResponse, error) {
	return cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*uhost.GetRegionResponse, error) {
		return c.uhost.GetRegion(req)
	})
}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/service/host/logic/cloud/ucloud/...
```

Expected: 无错误

---

### Task 12: 实现 UCLOUD 数据转换器

**Files:**
- Create: `internal/service/host/logic/cloud/ucloud/converter.go`
- Create: `internal/service/host/logic/cloud/ucloud/converter_test.go`

- [ ] **Step 1: 编写转换器测试**

创建 `internal/service/host/logic/cloud/ucloud/converter_test.go`：

```go
package ucloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucloud/ucloud-sdk-go/services/uhost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

func TestConvertInstance(t *testing.T) {
	instance := &uhost.UHostInstanceSet{
		UHostId:   "uhost-abcdefgh",
		Name:      "test-instance",
		Zone:      "cn-bj2-01",
		State:     "Running",
		OsName:    "Ubuntu 22.04",
		CPU:       4,
		Memory:    8192, // MB
		DiskSpace: 100,
		IPSet: []uhost.UHostIPSet{
			{
				Type:      "BGP",
				IP:        "1.2.3.4",
				PrivateIP: "10.0.0.1",
			},
		},
	}

	result := ConvertInstance(instance, "cn-bj2")

	assert.Equal(t, "uhost-abcdefgh", result.InstanceID)
	assert.Equal(t, "test-instance", result.Name)
	assert.Equal(t, "cn-bj2", result.Region)
	assert.Equal(t, "cn-bj2-01", result.Zone)
	assert.Equal(t, "running", result.Status)
	assert.Equal(t, "Ubuntu 22.04", result.OS)
	assert.Equal(t, 4, result.CPU)
	assert.Equal(t, 8192, result.MemoryMB)
	assert.Equal(t, 100, result.DiskGB)
}

func TestGetPublicIP_BGP(t *testing.T) {
	instance := &uhost.UHostInstanceSet{
		IPSet: []uhost.UHostIPSet{
			{Type: "BGP", IP: "1.2.3.4"},
			{Type: "Private", IP: "10.0.0.1"},
		},
	}

	ip := getPublicIP(instance)
	assert.Equal(t, "1.2.3.4", ip)
}

func TestGetPublicIP_VIP(t *testing.T) {
	instance := &uhost.UHostInstanceSet{
		IPSet: []uhost.UHostIPSet{
			{Type: "VIP", IP: "5.6.7.8"},
		},
	}

	ip := getPublicIP(instance)
	assert.Equal(t, "5.6.7.8", ip)
}

func TestGetPublicIP_EIP(t *testing.T) {
	instance := &uhost.UHostInstanceSet{
		IPSet: []uhost.UHostIPSet{
			{Type: "EIP", IP: "9.10.11.12"},
		},
	}

	ip := getPublicIP(instance)
	assert.Equal(t, "9.10.11.12", ip)
}

func TestGetPublicIP_NoPublic(t *testing.T) {
	instance := &uhost.UHostInstanceSet{
		IPSet: []uhost.UHostIPSet{
			{Type: "Private", IP: "10.0.0.1"},
		},
	}

	ip := getPublicIP(instance)
	assert.Empty(t, ip)
}

func TestGetPrivateIP(t *testing.T) {
	instance := &uhost.UHostInstanceSet{
		IPSet: []uhost.UHostIPSet{
			{Type: "BGP", IP: "1.2.3.4", PrivateIP: "10.0.0.1"},
		},
	}

	ip := getPrivateIP(instance)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestConvertStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Running", "running"},
		{"Stopped", "stopped"},
		{"Fail", "stopped"},
		{"Installing", "installing"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

- [ ] **Step 2: 实现转换器**

创建 `internal/service/host/logic/cloud/ucloud/converter.go`：

```go
// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
package ucloud

import (
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/uhost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将 UCLOUD 实例转换为统一的 CloudInstance 模型。
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
		DiskGB:     inst.DiskSpace,
	}
}

// getPublicIP 获取公网 IP 地址。
//
// 从 IPSet 数组中筛选 Type 为 VIP、BGP 或 EIP 的公网 IP。
func getPublicIP(inst *uhost.UHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		if ip.Type == "VIP" || ip.Type == "BGP" || ip.Type == "EIP" {
			if ip.IP != "" {
				return ip.IP
			}
		}
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
func getPrivateIP(inst *uhost.UHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		if ip.PrivateIP != "" {
			return ip.PrivateIP
		}
	}
	return ""
}

// convertStatus 转换实例状态为标准格式。
//
// UCLOUD 状态值:
//   - Running -> running
//   - Stopped -> stopped
//   - Fail -> stopped
//   - Installing -> installing
func convertStatus(state string) string {
	if state == "" {
		return "unknown"
	}

	switch state {
	case "Running":
		return "running"
	case "Stopped", "Fail":
		return "stopped"
	case "Installing":
		return "installing"
	default:
		return strings.ToLower(state)
	}
}
```

- [ ] **Step 3: 运行测试验证通过**

```bash
go test ./internal/service/host/logic/cloud/ucloud/... -v
```

Expected: PASS

- [ ] **Step 4: 提交 UCLOUD 转换器**

```bash
git add internal/service/host/logic/cloud/ucloud/
git commit -m "feat(cloud): add ucloud converter with IPSet filtering"
```

---

### Task 13: 实现 UCLOUD Provider

**Files:**
- Create: `internal/service/host/logic/cloud/ucloud/provider.go`

- [ ] **Step 1: 创建 Provider 实现**

创建 `internal/service/host/logic/cloud/ucloud/provider.go`：

```go
// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
package ucloud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/uhost"
	uclouderr "github.com/ucloud/ucloud-sdk-go/ucloud/error"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Provider UCLOUD 适配器。
type Provider struct{}

// New 创建 UCLOUD 适配器实例。
func New() *Provider {
	return &Provider{}
}

// Name 返回云厂商标识。
func (p *Provider) Name() string {
	return "ucloud"
}

// DisplayName 返回云厂商显示名称。
func (p *Provider) DisplayName() string {
	return "UCLOUD"
}

// Capabilities 返回 UCLOUD 能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证 UCLOUD 凭证是否有效。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	// 通过查询实例验证凭证
	req := &uhost.DescribeUHostInstanceRequest{}
	req.SetLimit(1)
	_, err = client.DescribeUHostInstance(ctx, req)
	if err != nil {
		return fmt.Errorf("UCLOUD 凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询 UCLOUD UHost 实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	input := &uhost.DescribeUHostInstanceRequest{}

	// 可用区过滤
	if req.Zone != "" {
		input.SetZone(req.Zone)
	}

	// 分页参数
	limit := 100
	if req.PageSize > 0 && req.PageSize <= 100 {
		limit = req.PageSize
	}
	input.SetLimit(limit)

	// Offset 分页
	if req.PageNumber > 1 {
		input.SetOffset((req.PageNumber - 1) * limit)
	}

	// 调用 API
	output, err := client.DescribeUHostInstance(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询 UCLOUD 实例失败: %w", p.wrapError(err))
	}

	// 转换实例数据
	instances := make([]cloud.CloudInstance, 0, len(output.UHostSet))
	for _, inst := range output.UHostSet {
		converted := ConvertInstance(&inst, req.Region)

		// 关键词过滤
		if req.Keyword != "" {
			kw := strings.ToLower(req.Keyword)
			if !strings.Contains(strings.ToLower(converted.Name), kw) &&
				!strings.Contains(strings.ToLower(converted.InstanceID), kw) &&
				!strings.Contains(converted.IP, kw) &&
				!strings.Contains(converted.PrivateIP, kw) {
				continue
			}
		}

		instances = append(instances, *converted)
	}

	return instances, nil
}

// ListRegions 查询 UCLOUD 支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	// UCLOUD GetRegion 不需要指定地域
	client, err := NewClient(ak, sk, "cn-bj2")
	if err != nil {
		return nil, err
	}

	req := &uhost.GetRegionRequest{}
	output, err := client.GetRegion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("查询 UCLOUD 地域失败: %w", p.wrapError(err))
	}

	regions := make([]cloud.Region, 0, len(output.Regions))
	for _, r := range output.Regions {
		regions = append(regions, cloud.Region{
			RegionId:  r.Region,
			LocalName: r.RegionName,
		})
	}
	return regions, nil
}

// ListZones 查询 UCLOUD 指定地域的可用区列表。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	client, err := NewClient(ak, sk, region)
	if err != nil {
		return nil, err
	}

	req := &uhost.GetRegionRequest{}
	output, err := client.GetRegion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("查询 UCLOUD 可用区失败: %w", p.wrapError(err))
	}

	zones := make([]cloud.Zone, 0)
	for _, r := range output.Regions {
		if r.Region == region {
			for _, z := range r.Zones {
				zones = append(zones, cloud.Zone{
					ZoneId:    z.Zone,
					LocalName: z.ZoneName,
				})
			}
			break
		}
	}
	return zones, nil
}

// wrapError 包装 UCLOUD 错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	var ucloudErr *uclouderr.Error
	if errors.As(err, &ucloudErr) {
		switch ucloudErr.RetCode {
		case 160:
			return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-bj2、cn-sh2")
		case 161:
			return fmt.Errorf("可用区无效")
		case 170:
			return fmt.Errorf("认证失败，请检查 AccessKey ID 和 Secret")
		case 171:
			return fmt.Errorf("AccessKey 无效")
		case 172:
			return fmt.Errorf("请求频率限制，已重试多次仍失败，请稍后重试")
		}
		return fmt.Errorf("[UCLOUD][%d] %s", ucloudErr.RetCode, ucloudErr.Message)
	}

	return fmt.Errorf("[UCLOUD] %w", err)
}

// init 自注册到全局 Registry。
//
// 注意：这是自注册模式，避免循环依赖。
func init() {
	cloud.Register(New())
}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./internal/service/host/logic/cloud/ucloud/...
```

Expected: 无错误

- [ ] **Step 3: 提交 UCLOUD Provider**

```bash
git add internal/service/host/logic/cloud/ucloud/provider.go
git commit -m "feat(cloud): implement ucloud CloudProvider with GetRegion API"
```

---

## Chunk 4: 注册与前端

### Task 14: 更新云厂商注册入口

**背景说明：**

现有注册逻辑在 `internal/service/host/logic/cloud.go` 的 `init()` 函数中。需要采用"自注册"模式避免循环依赖：

```
❌ 错误模式（循环依赖）：
cloud 包 -> 导入 alicloud 包 -> 导入 cloud 包 -> 循环！

✅ 正确模式（自注册）：
各 Provider 在自己的 init() 中调用 cloud.Register()
cloud.go 使用匿名导入触发注册
```

**Files:**
- Modify: `internal/service/host/logic/cloud.go`

- [ ] **Step 1: 更新 cloud.go 的 import 和 init 函数**

修改 `internal/service/host/logic/cloud.go`：

```go
import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
	// 匿名导入触发各云厂商适配器的自注册 init()
	_ "github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/alicloud"
	_ "github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/ucloud"
	_ "github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/volcengine"
	"github.com/cy77cc/OpsPilot/internal/utils"
)

// ... 其他代码保持不变 ...

// init 触发云厂商适配器注册。
//
// 注意：各 Provider 在自己的 init() 中调用 cloud.Register() 自注册。
// 这里不需要手动注册，匿名导入已触发注册。
func init() {
	// 自注册模式：各 Provider 在自己的 init() 中注册
	// 匿名导入确保 Provider 的 init() 被执行
}
```

- [ ] **Step 2: 删除旧的注册代码**

删除 `cloud.go` 中旧的 `init()` 函数内容：
```go
// 删除以下代码：
// cloud.Register(volcengine.New())
// cloud.Register(cloud.NewMockProvider("alicloud", "阿里云"))
// cloud.Register(cloud.NewMockProvider("tencent", "腾讯云"))
```

- [ ] **Step 3: 提交注册更新**

```bash
git add internal/service/host/logic/cloud.go
git commit -m "refactor(cloud): use self-registration pattern for providers

- Remove direct registration calls in cloud.go
- Use anonymous imports to trigger provider init()
- Each provider registers itself in its own init()
- Avoids circular dependency: cloud -> provider -> cloud"
```

---

### Task 15: 更新前端云厂商选项

**Files:**
- Modify: `web/src/pages/Hosts/HostCloudImportPage.tsx`

- [ ] **Step 1: 更新 providerOptions**

修改 `web/src/pages/Hosts/HostCloudImportPage.tsx` 中的 `providerOptions`：

```typescript
// 云厂商选项
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'ucloud', label: 'UCLOUD' },
];
```

- [ ] **Step 2: 提交前端更新**

```bash
git add web/src/pages/Hosts/HostCloudImportPage.tsx
git commit -m "feat(web): add alicloud and ucloud to provider options"
```

---

### Task 16: 集成测试

- [ ] **Step 1: 运行完整测试套件**

```bash
go test ./internal/service/host/logic/cloud/... -v -cover
```

Expected: 所有测试通过，覆盖率 ≥ 40%

- [ ] **Step 2: 编译验证**

```bash
go build ./...
```

Expected: 编译成功

- [ ] **Step 3: 启动后端服务验证**

```bash
make dev-backend
```

验证日志输出：
- 确认三个云厂商都已注册
- 确认 AccessKey Secret 未在日志中明文显示

---

### Task 17: 最终提交

- [ ] **Step 1: 运行 go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 2: 提交所有更改**

```bash
git add .
git commit -m "feat(cloud): complete multi-cloud provider implementation

- Add ProviderCapabilities and NextToken to types
- Implement generic retry mechanism with exponential backoff
- Add Alicloud ECS adapter with v9 SDK
- Add UCLOUD UHost adapter with GetRegion API
- Register all providers on init
- Update frontend provider options

Closes: multi-cloud provider extension design"
```

---

## 验收检查清单

- [ ] MockProvider 实现 `Capabilities()` 方法
- [ ] 火山云适配器实现 `Capabilities()` 和 `init()` 自注册
- [ ] 阿里云适配器实现 `init()` 自注册
- [ ] UCLOUD 适配器实现 `init()` 自注册
- [ ] `cloud.go` 使用匿名导入触发注册（无循环依赖）
- [ ] 重试机制使用 `time.NewTimer`（无内存泄露风险）
- [ ] UCLOUD 公网 IP 检测包含 EIP 类型
- [ ] 重试机制测试覆盖率 ≥ 80%
- [ ] 阿里云转换器测试覆盖率 ≥ 80%
- [ ] UCLOUD 转换器测试覆盖率 ≥ 80%
- [ ] 所有 Provider 实现完整接口
- [ ] 前端显示三个云厂商选项
- [ ] 编译无错误
- [ ] 日志无 AccessKey Secret 明文
