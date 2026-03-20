# 多云厂商适配器扩展设计

## 概述

在现有火山云适配器基础上，扩展支持阿里云和 UCLOUD，采用框架先行的方式抽象公共能力，降低后续扩展成本。

## 目标

1. 新增阿里云 ECS 实例查询适配器
2. 新增 UCLOUD UHost 实例查询适配器
3. 扩展 CloudProvider 接口，暴露厂商能力标识

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
    ├── converter.go
    └── regions.go       # 内置地域映射表
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
    // 阿里云、火山云：true（调用云 API 获取地域列表）
    // UCLOUD：false（使用内置映射表）
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
    //
    // 新增方法，用于查询厂商支持的功能特性。
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

### 二、分页参数映射

`ListInstancesRequest` 已定义 `PageNumber` 和 `PageSize`，各厂商映射如下：

| 厂商 | 分页参数 | 映射方式 |
|------|----------|----------|
| 火山云 | `MaxResults` | `PageSize` 直接映射 |
| 阿里云 | `PageNumber`, `PageSize` | 直接使用 |
| UCLOUD | `Offset`, `Limit` | `Offset = (PageNumber-1) * PageSize`, `Limit = PageSize` |

**默认值约定**：
- `PageNumber` 默认 1
- `PageSize` 默认 100，最大 100

### 三、阿里云适配器（alicloud/）

#### 3.1 SDK 依赖

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

> **版本说明**：使用 v9 版本，这是阿里云 Go SDK 的最新稳定版。

#### 3.2 客户端创建（client.go）

```go
// Client 阿里云 ECS 客户端封装。
type Client struct {
    ecs *ecs.Client
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
        return nil, fmt.Errorf("创建阿里云客户端失败: %w", err)
    }
    return &Client{ecs: client}, nil
}

// DescribeInstances 查询 ECS 实例列表。
func (c *Client) DescribeInstances(ctx context.Context, req *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
    return c.ecs.DescribeInstancesWithOptions(req, &util.RuntimeOptions{})
}

// DescribeRegions 查询地域列表。
func (c *Client) DescribeRegions(ctx context.Context, req *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error) {
    return c.ecs.DescribeRegionsWithOptions(req, &util.RuntimeOptions{})
}

// DescribeZones 查询可用区列表。
func (c *Client) DescribeZones(ctx context.Context, req *ecs.DescribeZonesRequest) (*ecs.DescribeZonesResponse, error) {
    return c.ecs.DescribeZonesWithOptions(req, &util.RuntimeOptions{})
}
```

#### 3.3 实例字段映射（converter.go）

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

#### 3.4 地域名称映射

```go
// regionNames 地域中文名称映射。
var regionNames = map[string]string{
    // 国内地域
    "cn-hangzhou":    "华东1（杭州）",
    "cn-shanghai":    "华东2（上海）",
    "cn-qingdao":     "华北1（青岛）",
    "cn-beijing":     "华北2（北京）",
    "cn-zhangjiakou": "华北3（张家口）",
    "cn-huhehaote":   "华北5（呼和浩特）",
    "cn-wulanchabu":  "华北6（乌兰察布）",
    "cn-shenzhen":    "华南1（深圳）",
    "cn-guangzhou":   "华南2（广州）",
    "cn-chengdu":     "西南1（成都）",
    "cn-hongkong":    "中国香港",
    // 海外地域
    "ap-northeast-1": "日本东京",
    "ap-northeast-2": "韩国首尔",
    "ap-southeast-1": "新加坡",
    "ap-southeast-2": "澳大利亚悉尼",
    "ap-southeast-3": "马来西亚吉隆坡",
    "ap-southeast-5": "印度尼西亚雅加达",
    "ap-south-1":     "印度孟买",
    "eu-central-1":   "德国法兰克福",
    "eu-west-1":      "英国伦敦",
    "us-east-1":      "美国弗吉尼亚",
    "us-west-1":      "美国硅谷",
}
```

#### 3.5 错误处理

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
            return fmt.Errorf("请求过于频繁，请稍后重试")
        case "ServiceUnavailable":
            return fmt.Errorf("服务暂时不可用，请稍后重试")
        }
        return fmt.Errorf("[阿里云][%s] %s", code, serverErr.Message())
    }

    return fmt.Errorf("[阿里云] %w", err)
}
```

### 四、UCLOUD 适配器（ucloud/）

#### 4.1 SDK 依赖

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

#### 4.2 客户端创建（client.go）

```go
// Client UCLOUD UHost 客户端封装。
type Client struct {
    uhost *uhost.UHostClient
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
    return &Client{uhost: uhost.NewClient(&config, credential)}, nil
}

// DescribeUHostInstance 查询 UHost 实例列表。
func (c *Client) DescribeUHostInstance(ctx context.Context, req *uhost.DescribeUHostInstanceRequest) (*uhost.DescribeUHostInstanceResponse, error) {
    return c.uhost.DescribeUHostInstance(req)
}
```

#### 4.3 实例字段映射（converter.go）

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

#### 4.4 内置地域映射表（regions.go）

```go
// BuiltinRegions 内置地域列表。
//
// UCLOUD 不提供查询地域列表的 API，使用内置映射表。
// 维护说明：如需新增地域，直接追加到此列表。
var BuiltinRegions = []cloud.Region{
    {RegionId: "cn-bj2", LocalName: "北京"},
    {RegionId: "cn-sh2", LocalName: "上海"},
    {RegionId: "cn-gd2", LocalName: "广州"},
    {RegionId: "hk-hk1", LocalName: "香港"},
    {RegionId: "us-ws-1", LocalName: "美西（洛杉矶）"},
    {RegionId: "us-la-1", LocalName: "美南（洛杉矶）"},
    {RegionId: "sg-sg1", LocalName: "新加坡"},
    {RegionId: "tw-tp-1", LocalName: "台北"},
    {RegionId: "idn-jakarta-1", LocalName: "雅加达"},
    {RegionId: "ind-mumbai-1", LocalName: "孟买"},
    {RegionId: "bra-saopaulo-1", LocalName: "圣保罗"},
    {RegionId: "uae-dubai-1", LocalName: "迪拜"},
    {RegionId: "afr-nigeria-1", LocalName: "尼日利亚"},
    {RegionId: "vn-sng-1", LocalName: "越南"},
}

// BuiltinZones 内置可用区列表（按地域分组）。
//
// 维护说明：如需新增可用区，追加到对应地域的数组中。
var BuiltinZones = map[string][]cloud.Zone{
    "cn-bj2": {
        {ZoneId: "cn-bj2-01", LocalName: "北京一可用区"},
        {ZoneId: "cn-bj2-02", LocalName: "北京二可用区"},
        {ZoneId: "cn-bj2-03", LocalName: "北京三可用区"},
        {ZoneId: "cn-bj2-04", LocalName: "北京四可用区"},
        {ZoneId: "cn-bj2-05", LocalName: "北京五可用区"},
    },
    "cn-sh2": {
        {ZoneId: "cn-sh2-01", LocalName: "上海一可用区"},
        {ZoneId: "cn-sh2-02", LocalName: "上海二可用区"},
        {ZoneId: "cn-sh2-03", LocalName: "上海三可用区"},
    },
    "cn-gd2": {
        {ZoneId: "cn-gd2-01", LocalName: "广州一可用区"},
        {ZoneId: "cn-gd2-02", LocalName: "广州二可用区"},
        {ZoneId: "cn-gd2-03", LocalName: "广州三可用区"},
    },
    "hk-hk1": {
        {ZoneId: "hk-hk1-01", LocalName: "香港一可用区"},
        {ZoneId: "hk-hk1-02", LocalName: "香港二可用区"},
    },
    "us-ws-1": {
        {ZoneId: "us-ws-1-01", LocalName: "美西一可用区"},
        {ZoneId: "us-ws-1-02", LocalName: "美西二可用区"},
    },
    "sg-sg1": {
        {ZoneId: "sg-sg1-01", LocalName: "新加坡一可用区"},
        {ZoneId: "sg-sg1-02", LocalName: "新加坡二可用区"},
    },
}
```

#### 4.5 错误处理

```go
// wrapError 包装 UCLOUD 错误，提供更友好的错误信息。
//
// UCLOUD SDK 返回的错误包含 RetCode 字段，根据错误码映射友好提示。
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
        }
        return fmt.Errorf("[UCLOUD][%d] %s", ucloudErr.RetCode, ucloudErr.Message)
    }

    return fmt.Errorf("[UCLOUD] %w", err)
}
```

### 五、前端调整

#### 5.1 云厂商选项

前端已有静态选项，需更新为：

```typescript
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'ucloud', label: 'UCLOUD' },
  { value: 'tencent', label: '腾讯云' },  // 后续扩展
];
```

#### 5.2 功能差异处理

UCLOUD 不支持动态查询地域，前端调用 `listCloudRegions` 时：
- 阿里云、火山云：调用云 API 获取
- UCLOUD：返回后端内置映射表

前端无需区分，由后端适配器统一处理。

### 六、测试策略

#### 6.1 测试文件位置

测试文件与源文件同级，遵循 Go 惯例：
```
internal/service/host/logic/cloud/
├── converter_test.go           # 公共转换工具测试
├── alicloud/
│   ├── converter_test.go       # 阿里云转换测试
│   └── provider_test.go        # 阿里云适配器测试
└── ucloud/
    ├── converter_test.go       # UCLOUD 转换测试
    └── provider_test.go        # UCLOUD 适配器测试
```

#### 6.2 Mock 策略

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

#### 6.3 测试用例

**Converter 测试**（核心转换逻辑，必须覆盖）：

| 测试文件 | 测试场景 | 预期结果 |
|----------|----------|----------|
| alicloud/converter_test.go | 正常实例数据转换 | 字段正确映射 |
| alicloud/converter_test.go | IP 地址优先级（EIP > 公网 IP） | 返回 EIP |
| alicloud/converter_test.go | 无公网 IP 实例 | IP 字段为空 |
| alicloud/converter_test.go | 状态映射（Running/Stopped） | running/stopped |
| ucloud/converter_test.go | IPSet 过滤（VIP/BGP） | 返回公网 IP |
| ucloud/converter_test.go | 内网 IP 获取 | 返回 PrivateIP |

**Provider 测试**（Mock 客户端）：

| 测试文件 | 测试场景 | 预期结果 |
|----------|----------|----------|
| alicloud/provider_test.go | ListInstances 成功 | 返回实例列表 |
| alicloud/provider_test.go | 认证错误 | 友好错误提示 |
| alicloud/provider_test.go | ListRegions 成功 | 返回地域列表 |
| ucloud/provider_test.go | ListInstances 成功 | 返回实例列表 |
| ucloud/provider_test.go | ListRegions（内置） | 返回内置列表 |

#### 6.4 覆盖率要求

- Converter 文件覆盖率：≥ 80%
- Provider 文件覆盖率：≥ 60%
- 整体模块覆盖率：≥ 40%（项目最低要求）

### 七、Provider 注册

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

### 阶段一：接口扩展

1. 更新 `types.go`，添加 `ProviderCapabilities` 类型
2. 更新 `provider.go`，添加 `Capabilities()` 方法
3. 更新火山云适配器，实现 `Capabilities()` 方法

### 阶段二：阿里云适配器

1. 添加 SDK 依赖（v9）
2. 实现 `alicloud/client.go` - ECS 客户端封装
3. 实现 `alicloud/converter.go` - 实例数据转换
4. 实现 `alicloud/provider.go` - CloudProvider 接口实现
5. 编写单元测试

### 阶段三：UCLOUD 适配器

1. 添加 SDK 依赖
2. 实现 `ucloud/regions.go` - 内置地域映射表
3. 实现 `ucloud/client.go` - UHost 客户端封装
4. 实现 `ucloud/converter.go` - 实例数据转换
5. 实现 `ucloud/provider.go` - CloudProvider 接口实现
6. 编写单元测试

### 阶段四：注册与前端

1. 添加 init.go，注册所有适配器
2. 更新前端 providerOptions
3. 运行集成测试

### 阶段五：集成验证

1. 使用真实 AccessKey 验证阿里云实例查询
2. 使用真实 AccessKey 验证 UCLOUD 实例查询
3. 验证前端云账号创建、实例查询、导入流程

## 验收标准

1. ✅ 可创建阿里云和 UCLOUD 云账号
2. ✅ 可查询阿里云 ECS 实例列表，字段正确显示
3. ✅ 可查询 UCLOUD UHost 实例列表，字段正确显示
4. ✅ 地域和可用区下拉选项正确展示
5. ✅ 实例导入功能正常工作
6. ✅ 单元测试覆盖率达到要求
7. ✅ 错误提示友好，便于用户排查问题

## 风险与应对

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 阿里云 SDK 版本兼容性 | 编译失败 | 使用最新稳定版 v9 |
| UCLOUD 地域变动 | 可用区列表过时 | 硬编码常用地域，文档说明维护方式 |
| API 限流 | 查询失败 | 错误提示用户稍后重试，后续可加重试逻辑 |
| 字段映射错误 | 数据显示异常 | 充分测试，对比实际 API 返回 |
| 测试环境限制 | 无法验证 | 使用真实 AccessKey 测试 |
