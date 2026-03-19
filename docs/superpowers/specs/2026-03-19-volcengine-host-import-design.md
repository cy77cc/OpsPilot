# 火山云主机导入设计

**日期**: 2026-03-19
**状态**: 待实现
**作者**: Claude

---

## 概述

为主机导入功能增加火山云（VolcEngine）支持，采用接口适配器模式，设计统一的云厂商接口，便于后续扩展其他云平台。

## 目标

1. 实现火山云 ECS 实例查询和导入功能
2. 设计统一的 `CloudProvider` 接口
3. 保持对现有阿里云、腾讯云 mock 实现的兼容

## 架构设计

### 目录结构

```
internal/service/host/logic/cloud/
├── provider.go              # CloudProvider 接口定义
├── registry.go              # 云厂商注册表
├── types.go                 # 通用类型定义
├── mock_provider.go         # Mock 适配器（阿里云/腾讯云占位）
└── volcengine/
    ├── provider.go          # 火山云适配器入口
    ├── client.go            # SDK 客户端封装
    └── converter.go         # 实例数据转换
```

### 调用流程

```
前端请求
    ↓
Handler.QueryCloudInstances
    ↓
HostService.QueryCloudInstances
    ↓
Registry.GetProvider(provider)
    ↓
provider.ListInstances(ctx, req)
    ↓
返回统一的 []CloudInstance
```

## 接口定义

### CloudProvider 接口

```go
// CloudProvider 定义云厂商适配器接口。
type CloudProvider interface {
    // Name 返回云厂商标识（如 volcengine、alicloud、tencent）。
    Name() string

    // DisplayName 返回显示名称（如 火山云、阿里云、腾讯云）。
    DisplayName() string

    // ValidateCredential 验证凭证是否有效。
    ValidateCredential(ctx context.Context, ak, sk, region string) error

    // ListInstances 查询实例列表。
    ListInstances(ctx context.Context, req ListInstancesRequest) ([]CloudInstance, error)
}
```

### 请求/响应类型

```go
// ListInstancesRequest 查询实例请求。
type ListInstancesRequest struct {
    AccessKeyID     string
    AccessKeySecret string
    Region          string
    Keyword         string // 按名称/IP 过滤
    PageNumber      int
    PageSize        int
}

// CloudInstance 统一的云实例模型。
type CloudInstance struct {
    InstanceID string `json:"instance_id"`
    Name       string `json:"name"`
    IP         string `json:"ip"`          // 公网 IP，用于 SSH 连接
    PrivateIP  string `json:"private_ip"`  // 内网 IP（可选）
    Region     string `json:"region"`
    Zone       string `json:"zone"`
    Status     string `json:"status"`
    OS         string `json:"os"`
    CPU        int    `json:"cpu"`
    MemoryMB   int    `json:"memory_mb"`
    DiskGB     int    `json:"disk_gb"`
}
```

### 注册表

```go
// Registry 管理所有云厂商适配器。
type Registry struct {
    providers map[string]CloudProvider
    mu        sync.RWMutex
}

// 全局注册表实例。
var globalRegistry = NewRegistry()

// Register 注册云厂商适配器。
func Register(p CloudProvider)

// GetProvider 获取指定云厂商适配器。
func GetProvider(name string) (CloudProvider, error)

// ListProviders 列出所有已注册的云厂商。
func ListProviders() []ProviderInfo

// ProviderInfo 云厂商信息（用于前端下拉框）。
type ProviderInfo struct {
    Name        string `json:"name"`         // volcengine
    DisplayName string `json:"display_name"` // 火山云
}
```

## 火山云适配器实现

### Provider 入口

```go
// volcengine/provider.go
package volcengine

type VolcengineProvider struct{}

func New() *VolcengineProvider {
    return &VolcengineProvider{}
}

func (p *VolcengineProvider) Name() string {
    return "volcengine"
}

func (p *VolcengineProvider) DisplayName() string {
    return "火山云"
}

func (p *VolcengineProvider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
    // 调用 DescribeInstances 限制返回 1 条，验证凭证有效性
    client, err := NewClient(ak, sk, region)
    if err != nil {
        return err
    }
    _, err = client.DescribeInstances(ctx, &ecs.DescribeInstancesInput{
        MaxResults: volcengine.Int32(1),
    })
    return err
}

func (p *VolcengineProvider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
    client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
    if err != nil {
        return nil, err
    }

    input := &ecs.DescribeInstancesInput{}
    if req.Keyword != "" {
        input.SetInstanceName(req.Keyword)
    }
    if req.PageSize > 0 {
        input.SetMaxResults(int32(req.PageSize))
    }

    output, err := client.DescribeInstances(ctx, input)
    if err != nil {
        return nil, err
    }

    instances := make([]cloud.CloudInstance, 0, len(output.Instances))
    for _, inst := range output.Instances {
        instances = append(instances, *ConvertInstance(inst))
    }
    return instances, nil
}
```

### SDK 客户端封装

```go
// volcengine/client.go
package volcengine

import (
    "context"

    "github.com/volcengine/volcengine-go-sdk/service/ecs"
    "github.com/volcengine/volcengine-go-sdk/volcengine"
    "github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
    "github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

type Client struct {
    ecs *ecs.ECS
}

func NewClient(ak, sk, region string) (*Client, error) {
    config := volcengine.NewConfig().
        WithCredentials(credentials.NewStaticCredentials(ak, sk, "")).
        WithRegion(region)
    sess, err := session.NewSession(config)
    if err != nil {
        return nil, err
    }
    return &Client{ecs: ecs.New(sess)}, nil
}

func (c *Client) DescribeInstances(ctx context.Context, input *ecs.DescribeInstancesInput) (*ecs.DescribeInstancesOutput, error) {
    return c.ecs.DescribeInstancesWithContext(ctx, input)
}
```

### 数据转换

```go
// volcengine/converter.go
package volcengine

import (
    "github.com/volcengine/volcengine-go-sdk/service/ecs"
    "github.com/volcengine/volcengine-go-sdk/volcengine"

    "github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将火山云实例转换为通用模型。
func ConvertInstance(inst *ecs.InstanceForDescribeInstancesOutput) *cloud.CloudInstance {
    return &cloud.CloudInstance{
        InstanceID: volcengine.StringValue(inst.InstanceId),
        Name:       volcengine.StringValue(inst.InstanceName),
        IP:         getPublicIP(inst),
        PrivateIP:  getPrivateIP(inst),
        Region:     getRegion(inst),
        Zone:       volcengine.StringValue(inst.ZoneId),
        Status:     convertStatus(inst.Status),
        OS:         volcengine.StringValue(inst.OsName),
        CPU:        int(volcengine.Int32Value(inst.Cpus)),
        MemoryMB:   int(volcengine.Int32Value(inst.MemorySize)),
        DiskGB:     calculateTotalDisk(inst.Volumes),
    }
}

// getPublicIP 获取公网 IP（优先 EipAddress）。
func getPublicIP(inst *ecs.InstanceForDescribeInstancesOutput) string {
    if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
        return *inst.EipAddress.IpAddress
    }
    for _, nic := range inst.NetworkInterfaces {
        if nic.PublicIpAddress != nil && *nic.PublicIpAddress != "" {
            return *nic.PublicIpAddress
        }
    }
    return ""
}

// getPrivateIP 获取内网 IP。
func getPrivateIP(inst *ecs.InstanceForDescribeInstancesOutput) string {
    for _, nic := range inst.NetworkInterfaces {
        if nic.PrimaryIpAddress != nil && *nic.PrimaryIpAddress != "" {
            return *nic.PrimaryIpAddress
        }
    }
    return ""
}

// getRegion 从 Placement 获取地域。
func getRegion(inst *ecs.InstanceForDescribeInstancesOutput) string {
    if inst.Placement != nil && inst.Placement.Region != nil {
        return *inst.Placement.Region
    }
    return ""
}

// convertStatus 转换实例状态。
func convertStatus(status *string) string {
    if status == nil {
        return "unknown"
    }
    // 火山云状态: Running, Stopped, Starting, Stopping 等
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

// calculateTotalDisk 计算总磁盘大小。
func calculateTotalDisk(volumes []*ecs.VolumeForDescribeInstancesOutput) int {
    var total int
    for _, v := range volumes {
        if v.Size != nil {
            total += int(*v.Size)
        }
    }
    return total
}
```

## 前端改动

### 下拉选项

```tsx
// HostCloudImportPage.tsx

const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'tencent', label: '腾讯云' },
]

// 默认值改为火山云
initialValues={{ provider: 'volcengine' }}
```

### 新增 API（可选）

```go
// GET /api/v1/hosts/cloud/providers
// 返回动态云厂商列表

func (h *Handler) ListCloudProviders(c *gin.Context) {
    providers := cloud.ListProviders()
    httpx.OK(c, providers)
}
```

## 数据模型

**无需改动表结构**，现有 `HostCloudAccount` 已支持：

| 字段 | 说明 |
|------|------|
| Provider | 云厂商标识：`volcengine`、`alicloud`、`tencent` |
| AccessKeyID | API 访问密钥 ID |
| AccessKeySecretEnc | 加密存储的访问密钥 |
| RegionDefault | 默认地域 |

## 错误处理

```go
// 统一错误码
var (
    ErrCloudProviderNotSupported = xcode.New(4001, "不支持的云厂商")
    ErrCloudCredentialInvalid    = xcode.New(4002, "云账号凭证无效")
    ErrCloudAPICallFailed        = xcode.New(4003, "云 API 调用失败")
)
```

## 测试策略

| 测试类型 | 范围 |
|---------|------|
| 单元测试 | `ConvertInstance()` 数据转换逻辑 |
| 集成测试 | Mock ECS SDK，验证 ListInstances 流程 |
| 手动测试 | 使用真实火山云账号验证完整流程 |

## 依赖

| 包 | 版本 | 说明 |
|---|------|------|
| github.com/volcengine/volcengine-go-sdk | v1.2.21 | 火山云 Go SDK |

## 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| 火山云 API 限流 | 实现分页查询，控制请求频率 |
| 凭证泄露 | 使用加密存储，日志脱敏 |
| 网络超时 | 设置合理的 HTTP 超时时间 |

## 后续扩展

1. 阿里云 SDK 集成
2. 腾讯云 SDK 集成
3. 支持更多火山云地域
4. 支持实例标签同步
