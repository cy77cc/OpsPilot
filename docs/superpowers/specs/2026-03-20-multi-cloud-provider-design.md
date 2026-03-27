# 多云厂商适配器扩展设计

## 概述

在现有火山云适配器基础上，扩展支持阿里云和 UCLOUD，采用框架先行的方式抽象公共能力，降低后续扩展成本。

## 目标

1. 新增阿里云 ECS 实例查询适配器
2. 新增 UCLOUD UHost 实例查询适配器
3. 扩展 CloudProvider 接口，暴露厂商能力标识
4. 统一限流重试机制，提升用户体验
5. 确保敏感信息安全处理

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      CloudProvider 接口                      │
│  Name() | DisplayName() | Capabilities()                    │
│  ValidateCredential() | ListInstances()                     │
│  ListRegions() | ListZones()                                │
└─────────────────────────────────────────────────────────────┘
                              ▲
          ┌───────────────────┼───────────────────┐
          │                   │                   │
    ┌─────┴─────┐       ┌─────┴─────┐       ┌─────┴─────┐
    │ Volcengine│       │ Alicloud  │       │  UCloud   │
    │ (已实现)  │       │  (新增)   │       │  (新增)   │
    └───────────┘       └───────────┘       └───────────┘
```

### 目录结构

```
internal/service/host/logic/cloud/
├── provider.go          # CloudProvider 接口定义（已有，需更新）
├── types.go             # 公共类型定义（已有，需更新）
├── registry.go          # 全局注册表（已有）
├── retry.go             # 【新增】通用重试机制
├── volcengine/          # 火山云（已有，需更新）
│   ├── provider.go      # 添加 Capabilities() 方法
│   ├── client.go
│   └── converter.go
├── alicloud/            # 【新增】阿里云
│   ├── provider.go
│   ├── client.go
│   └── converter.go
└── ucloud/              # 【新增】UCLOUD
    ├── provider.go
    ├── client.go
    └── converter.go
```

## 详细设计

### 一、接口扩展（provider.go + types.go）

#### 1.1 新增 ProviderCapabilities 类型（types.go）

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

#### 1.2 扩展 CloudProvider 接口（provider.go）

```go
// CloudProvider 定义云厂商适配器接口。
type CloudProvider interface {
    // Name 返回云厂商标识。
    Name() string

    // DisplayName 返回云厂商显示名称。
    DisplayName() string

    // Capabilities 返回云厂商能力标识。
    Capabilities() ProviderCapabilities

    // ValidateCredential 验证云账号凭证是否有效。
    ValidateCredential(ctx context.Context, ak, sk, region string) error

    // ListInstances 查询云厂商实例列表。
    ListInstances(ctx context.Context, req ListInstancesRequest) ([]CloudInstance, error)

    // ListRegions 查询云厂商支持的地域列表。
    ListRegions(ctx context.Context, ak, sk string) ([]Region, error)

    // ListZones 查询云厂商指定地域的可用区列表。
    ListZones(ctx context.Context, ak, sk, region string) ([]Zone, error)
}
```

#### 1.3 更新火山云适配器（volcengine/provider.go）

```go
// Capabilities 返回火山云能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
    return cloud.ProviderCapabilities{
        DynamicRegions: true,
    }
}
```

### 二、分页参数设计

#### 2.1 请求参数扩展（types.go）

```go
// ListInstancesRequest 查询云实例请求参数。
type ListInstancesRequest struct {
    // AccessKeyID 云 API 访问密钥 ID。
    AccessKeyID string

    // AccessKeySecret 云 API 访问密钥 Secret。
    AccessKeySecret string

    // Region 地域标识。
    Region string

    // Zone 可用区标识（可选）。
    Zone string

    // Keyword 过滤关键词。
    Keyword string

    // PageNumber 页码（从 1 开始）。
    PageNumber int

    // PageSize 每页数量。
    PageSize int

    // NextToken 分页令牌（可选）。
    //
    // 用于大数据量遍历，支持游标分页。
    // 阿里云 V9 SDK 推荐使用 NextToken 替代 PageNumber。
    // 如果提供 NextToken，优先使用游标分页。
    NextToken string
}
```

#### 2.2 分页参数映射

| 厂商 | 分页方式 | 映射逻辑 |
|------|----------|----------|
| 火山云 | `MaxResults` | `PageSize` 直接映射 |
| 阿里云 | `PageNumber`, `PageSize` 或 `NextToken`, `MaxResults` | 优先使用 `NextToken`（如提供），否则使用 `PageNumber` |
| UCLOUD | `Offset`, `Limit` | `Offset = (PageNumber-1) * PageSize`, `Limit = PageSize` |

**使用场景说明**：
- **前端 UI 翻页**：使用 `PageNumber` + `PageSize`，简单直观
- **批量同步/导出**：使用 `NextToken` 游标分页，避免深分页性能问题

**默认值约定**：
- `PageNumber` 默认 1
- `PageSize` 默认 100，最大 100

### 三、通用重试机制（retry.go）

#### 3.1 设计目标

云厂商 API 触发限流（Throttling）是非常普遍的现象，将重试责任推给用户体验不好。在客户端封装层引入自动重试机制，透明处理临时性错误。

#### 3.2 重试策略

```go
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
    "alicloud": {"Throttling", "ServiceUnavailable", "InternalError"},
    "volcengine": {"RequestLimitExceeded", "ServiceUnavailable"},
    "ucloud": {"172", "5000"}, // 172: 请求频率限制, 5000: 服务内部错误
}
```

#### 3.3 重试实现

```go
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

        // 等待后重试
        log.Debugf("云 API 请求失败，%s 后重试 (第 %d 次): %s", delay, i+1, op)
        select {
        case <-ctx.Done():
            return result, ctx.Err()
        case <-time.After(delay):
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

### 四、阿里云适配器（alicloud/）

#### 4.1 SDK 依赖

```go
import (
    openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
    ecs "github.com/alibabacloud-go/ecs-20140526/v9/client"
)
```

安装命令：
```bash
go get github.com/alibabacloud-go/ecs-20140526/v9
```

> **版本说明**：使用 v9 版本，这是阿里云 Go SDK 的最新稳定版，全面支持 NextToken 游标分页。

#### 4.2 客户端创建（client.go）

```go
// Client 阿里云 ECS 客户端封装。
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
        AccessKeyId:     &ak,
        AccessKeySecret: &sk,
        RegionId:        &region,
        Endpoint:        &endpoint,
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
        return c.ecs.DescribeInstancesWithOptions(req, &util.RuntimeOptions{})
    })
}

// DescribeRegions 查询地域列表（带重试）。
//
// 返回结果包含 LocalName 字段（中文名称），无需硬编码映射。
func (c *Client) DescribeRegions(ctx context.Context, req *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error) {
    return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeRegions", func() (*ecs.DescribeRegionsResponse, error) {
        return c.ecs.DescribeRegionsWithOptions(req, &util.RuntimeOptions{})
    })
}

// DescribeZones 查询可用区列表（带重试）。
func (c *Client) DescribeZones(ctx context.Context, req *ecs.DescribeZonesRequest) (*ecs.DescribeZonesResponse, error) {
    return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeZones", func() (*ecs.DescribeZonesResponse, error) {
        return c.ecs.DescribeZonesWithOptions(req, &util.RuntimeOptions{})
    })
}
```

#### 4.3 实例字段映射（converter.go）

| CloudInstance 字段 | 阿里云字段 | 转换逻辑 |
|-------------------|-----------|----------|
| InstanceID | InstanceId | 直接赋值 |
| Name | InstanceName | 直接赋值 |
| IP | EipAddress.IpAddress 或 PublicIpAddress.IpAddress[0] | **优先 EIP，其次公网 IP** |
| PrivateIP | VpcAttributes.PrivateIpAddress.IpAddress[0] | VPC 内网 IP |
| Region | RegionId | 直接赋值 |
| Zone | ZoneId | 直接赋值 |
| Status | Status | Running→running, Stopped→stopped |
| OS | OSName | 直接赋值 |
| CPU | Cpu | 直接赋值 |
| MemoryMB | Memory * 1024 | 阿里云返回 GB，需转 MB |
| DiskGB | 磁盘累加 | DataDisks.Size 总和 + SystemDisk.Size |

**IP 地址获取逻辑**：

```go
// getPublicIP 获取公网 IP 地址。
//
// 优先级：EIP > 公网 IP
// EIP 是用户绑定的弹性公网 IP，通常是主要的外网访问入口。
func getPublicIP(inst *ecs.Instance) string {
    // 优先返回 EIP（弹性公网 IP）
    if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil && *inst.EipAddress.IpAddress != "" {
        return *inst.EipAddress.IpAddress
    }
    // 其次返回公网 IP
    if inst.PublicIpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 {
        return inst.PublicIpAddress.IpAddress[0]
    }
    return ""
}
```

#### 4.4 地域查询实现

**设计决策**：阿里云 `DescribeRegions` API 返回结果包含 `LocalName` 字段（中文名称），无需硬编码 `regionNames` 映射。直接使用 API 返回数据即可，避免未来新增地域时的维护成本。

```go
// ListRegions 查询阿里云支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
    // 使用默认地域创建客户端（DescribeRegions 不需要指定地域）
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
            RegionId:  *r.RegionId,
            LocalName: *r.LocalName, // 直接使用 API 返回的中文名称
        })
    }
    return regions, nil
}
```

#### 4.5 错误处理

```go
// wrapError 包装阿里云错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
    if err == nil {
        return nil
    }

    var serverErr *errors.ServerError
    if errors.As(err, &serverErr) {
        code := serverErr.ErrorCode()
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
            return fmt.Errorf("缺少必要参数: %s", serverErr.Message())
        case "Throttling":
            // 重试机制已处理，这里只是最后的错误提示
            return fmt.Errorf("请求过于频繁，已重试多次仍失败，请稍后重试")
        case "ServiceUnavailable":
            return fmt.Errorf("服务暂时不可用，请稍后重试")
        }
        return fmt.Errorf("[阿里云][%s] %s", code, serverErr.Message())
    }

    return fmt.Errorf("[阿里云] %w", err)
}
```

### 五、UCLOUD 适配器（ucloud/）

#### 5.1 SDK 依赖

```go
import (
    "github.com/ucloud/ucloud-sdk-go/services/uhost"
    "github.com/ucloud/ucloud-sdk-go/ucloud"
    "github.com/ucloud/ucloud-sdk-go/ucloud/auth"
)
```

安装命令：
```bash
go get github.com/ucloud/ucloud-sdk-go
```

#### 5.2 客户端创建（client.go）

```go
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
//
// UCLOUD 提供基础 API GetRegion 用于查询地域和可用区信息。
func (c *Client) GetRegion(ctx context.Context, req *ucloud.GetRegionRequest) (*ucloud.GetRegionResponse, error) {
    return cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*ucloud.GetRegionResponse, error) {
        return c.uhost.GetRegion(req)
    })
}
```

#### 5.3 实例字段映射（converter.go）

| CloudInstance 字段 | UCLOUD 字段 | 转换逻辑 |
|-------------------|-------------|----------|
| InstanceID | UHostId | 直接赋值 |
| Name | Name | 直接赋值 |
| IP | IPSet 中 Type=VIP/BGP 的 IP | **需过滤 IPSet 数组** |
| PrivateIP | IPSet 中 PrivateIP | 第一个非空内网 IP |
| Region | 请求参数传入 | 实例数据无 Region 字段 |
| Zone | Zone | 直接赋值 |
| Status | State | Running→running, Stopped→stopped |
| OS | OsName | 直接赋值 |
| CPU | CPU | 直接赋值（核数） |
| MemoryMB | Memory | 直接赋值（MB） |
| DiskGB | DiskSpace | 直接赋值（GB） |

**IP 地址获取逻辑**：

```go
// getPublicIP 获取公网 IP 地址。
//
// 从 IPSet 数组中筛选 Type 为 VIP 或 BGP 的公网 IP。
func getPublicIP(inst *uhost.UHostInstanceSet) string {
    for _, ip := range inst.IPSet {
        if ip.Type == "VIP" || ip.Type == "BGP" {
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
```

#### 5.4 地域和可用区查询

**设计决策**：UCLOUD 提供 `GetRegion` API 用于查询地域和可用区列表，无需硬编码。动态查询可以：
1. 自动获取最新地域和可用区
2. 避免因机房裁撤或新增导致的列表过时
3. 适应不同用户的可用区访问权限差异

```go
// ListRegions 查询 UCLOUD 支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
    // UCLOUD GetRegion 不需要指定地域
    client, err := NewClient(ak, sk, "cn-bj2")
    if err != nil {
        return nil, err
    }

    req := &ucloud.GetRegionRequest{}
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

    req := &ucloud.GetRegionRequest{}
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
```

#### 5.5 错误处理

```go
// wrapError 包装 UCLOUD 错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
    if err == nil {
        return nil
    }

    // UCLOUD SDK 错误类型
    var ucloudErr *ucloud.Error
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
            // 重试机制已处理
            return fmt.Errorf("请求频率限制，已重试多次仍失败，请稍后重试")
        }
        return fmt.Errorf("[UCLOUD][%d] %s", ucloudErr.RetCode, ucloudErr.Message)
    }

    return fmt.Errorf("[UCLOUD] %w", err)
}
```

### 六、安全要求

#### 6.1 日志脱敏规范

**严格禁止**在日志中输出 AccessKey Secret，所有日志输出必须遵循以下规则：

```go
// 日志输出示例（正确做法）
log.Infof("创建云客户端: provider=%s, ak=%s***", provider, ak[:8])

// 错误做法（禁止）
log.Infof("创建云客户端: ak=%s, sk=%s", ak, sk)  // 禁止输出完整 sk
log.Debugf("凭证: %v", credential)              // 禁止输出包含 sk 的对象
```

#### 6.2 错误信息处理

```go
// NewClient 错误处理示例
func NewClient(ak, sk, region string) (*Client, error) {
    if ak == "" || sk == "" {
        // 错误信息不包含 sk
        return nil, fmt.Errorf("AccessKey ID 和 Secret 不能为空")
    }

    client, err := ecs.NewClient(config)
    if err != nil {
        // 错误包装时过滤敏感信息
        return nil, fmt.Errorf("创建客户端失败: %w", sanitizeError(err))
    }
    // ...
}

// sanitizeError 清理错误信息中的敏感数据
func sanitizeError(err error) error {
    if err == nil {
        return nil
    }
    // 移除可能包含的 AccessKey Secret
    msg := err.Error()
    msg = regexp.MustCompile(`AccessKeySecret[=:]\s*\S+`).ReplaceAllString(msg, "AccessKeySecret=***")
    msg = regexp.MustCompile(`Secret[=:]\s*\S+`).ReplaceAllString(msg, "Secret=***")
    return errors.New(msg)
}
```

#### 6.3 凭证存储

- AccessKey Secret 在数据库中必须加密存储（已有实现）
- 传输过程使用 HTTPS
- 内存中不长时间持有明文 Secret

### 七、前端调整

#### 7.1 云厂商选项

前端已有静态选项，需更新为：

```typescript
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'ucloud', label: 'UCLOUD' },
  { value: 'tencent', label: '腾讯云' },  // 后续扩展
];
```

#### 7.2 功能差异处理

所有厂商均支持动态查询地域和可用区，前端无需区分处理。

### 八、测试策略

#### 8.1 测试文件位置

测试文件与源文件同级，遵循 Go 惯例：
```
internal/service/host/logic/cloud/
├── retry_test.go               # 重试机制测试
├── alicloud/
│   ├── converter_test.go       # 阿里云转换测试
│   └── provider_test.go        # 阿里云适配器测试
└── ucloud/
    ├── converter_test.go       # UCLOUD 转换测试
    └── provider_test.go        # UCLOUD 适配器测试
```

#### 8.2 Mock 策略

由于云 SDK 客户端是结构体而非接口，采用接口包装方式 Mock：

```go
// 定义客户端接口（用于 Mock）
type ECSClient interface {
    DescribeInstances(ctx context.Context, req *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error)
    DescribeRegions(ctx context.Context, req *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error)
    DescribeZones(ctx context.Context, req *ecs.DescribeZonesRequest) (*ecs.DescribeZonesResponse, error)
}

// Provider 持有接口而非具体类型
type Provider struct {
    client ECSClient
}
```

#### 8.3 测试用例

**Converter 测试**（核心转换逻辑，必须覆盖）：

| 测试文件 | 测试场景 | 预期结果 |
|----------|----------|----------|
| alicloud/converter_test.go | 正常实例数据转换 | 字段正确映射 |
| alicloud/converter_test.go | IP 地址优先级（EIP > 公网 IP） | 返回 EIP |
| alicloud/converter_test.go | 无公网 IP 实例 | IP 字段为空 |
| alicloud/converter_test.go | 状态映射（Running/Stopped） | running/stopped |
| ucloud/converter_test.go | IPSet 过滤（VIP/BGP） | 返回公网 IP |
| ucloud/converter_test.go | 内网 IP 获取 | 返回 PrivateIP |

**重试机制测试**：

| 测试文件 | 测试场景 | 预期结果 |
|----------|----------|----------|
| retry_test.go | 正常请求直接返回 | 无重试 |
| retry_test.go | Throttling 错误自动重试 | 重试后成功 |
| retry_test.go | 不可重试错误直接返回 | 无重试 |
| retry_test.go | 重试耗尽返回错误 | 返回最后一次错误 |

**Provider 测试**（Mock 客户端）：

| 测试文件 | 测试场景 | 预期结果 |
|----------|----------|----------|
| alicloud/provider_test.go | ListInstances 成功 | 返回实例列表 |
| alicloud/provider_test.go | 认证错误 | 友好错误提示 |
| alicloud/provider_test.go | ListRegions 使用 API LocalName | 返回地域列表 |
| ucloud/provider_test.go | ListInstances 成功 | 返回实例列表 |
| ucloud/provider_test.go | ListRegions 动态查询 | 返回地域列表 |

#### 8.4 覆盖率要求

- Converter 文件覆盖率：≥ 80%
- Provider 文件覆盖率：≥ 60%
- Retry 文件覆盖率：≥ 80%
- 整体模块覆盖率：≥ 40%（项目最低要求）

### 九、Provider 注册

在应用启动时注册所有云厂商适配器：

```go
// internal/service/host/logic/cloud/init.go
func init() {
    Register(volcengine.New())
    Register(alicloud.New())
    Register(ucloud.New())
}
```

## 实现计划

### 阶段一：基础能力

1. 新增 `retry.go`，实现通用重试机制
2. 更新 `types.go`，添加 `ProviderCapabilities` 和 `NextToken` 字段
3. 更新 `provider.go`，添加 `Capabilities()` 方法
4. 更新火山云适配器，实现 `Capabilities()` 方法

### 阶段二：阿里云适配器

1. 添加 SDK 依赖（v9）
2. 实现 `alicloud/client.go` - ECS 客户端封装（带重试）
3. 实现 `alicloud/converter.go` - 实例数据转换
4. 实现 `alicloud/provider.go` - CloudProvider 接口实现
5. 编写单元测试

### 阶段三：UCLOUD 适配器

1. 添加 SDK 依赖
2. 实现 `ucloud/client.go` - UHost 客户端封装（带重试）
3. 实现 `ucloud/converter.go` - 实例数据转换
4. 实现 `ucloud/provider.go` - CloudProvider 接口实现
5. 编写单元测试

### 阶段四：注册与前端

1. 添加 init.go，注册所有适配器
2. 更新前端 providerOptions
3. 运行集成测试

### 阶段五：集成验证

1. 使用真实 AccessKey 验证阿里云实例查询
2. 使用真实 AccessKey 验证 UCLOUD 实例查询
3. 验证限流重试机制正常工作
4. 验证前端云账号创建、实例查询、导入流程
5. 检查日志确保无敏感信息泄露

## 验收标准

1. ✅ 可创建阿里云和 UCLOUD 云账号
2. ✅ 可查询阿里云 ECS 实例列表，字段正确显示
3. ✅ 可查询 UCLOUD UHost 实例列表，字段正确显示
4. ✅ 地域和可用区通过 API 动态获取，下拉选项正确展示
5. ✅ 实例导入功能正常工作
6. ✅ 限流时自动重试，用户体验良好
7. ✅ 日志中无 AccessKey Secret 明文
8. ✅ 单元测试覆盖率达到要求
9. ✅ 错误提示友好，便于用户排查问题

## 风险与应对

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 阿里云 SDK 版本兼容性 | 编译失败 | 使用最新稳定版 v9 |
| UCLOUD GetRegion API 响应格式变化 | 解析失败 | 充分测试，添加字段存在性检查 |
| API 限流频繁 | 请求失败 | 自动重试机制 + 友好提示 |
| 字段映射错误 | 数据显示异常 | 充分测试，对比实际 API 返回 |
| 日志泄露敏感信息 | 安全风险 | 代码审查 + 自动化检测 |
| 测试环境限制 | 无法验证 | 使用真实 AccessKey 测试 |

## 变更记录

| 日期 | 版本 | 变更说明 |
|------|------|----------|
| 2026-03-20 | v2 | 根据反馈优化：移除阿里云硬编码地域映射，UCLOUD 改用 GetRegion API，添加重试机制，添加日志脱敏要求 |
| 2026-03-20 | v1 | 初始版本 |
