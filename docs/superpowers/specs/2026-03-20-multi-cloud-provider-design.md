# 多云厂商适配器扩展设计

## 概述

在现有火山云适配器基础上，扩展支持阿里云和 UCLOUD，采用框架先行的方式抽象公共能力，降低后续扩展成本。

## 目标

1. 新增阿里云 ECS 实例查询适配器
2. 新增 UCLOUD UHost 实例查询适配器
3. 抽象公共基础能力，统一错误处理和厂商能力标识

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      CloudProvider 接口                      │
│  Name() | DisplayName() | ValidateCredential()              │
│  ListInstances() | ListRegions() | ListZones()              │
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
├── provider.go          # CloudProvider 接口定义（已有）
├── types.go             # 公共类型定义（已有）
├── registry.go          # 全局注册表（已有）
├── base.go              # 【新增】公共基础能力
├── volcengine/          # 火山云（已有）
│   ├── provider.go
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

### 一、公共基础能力（base.go）

#### ProviderCapabilities 云厂商能力标识

```go
type ProviderCapabilities struct {
    // DynamicRegions 是否支持动态查询地域
    // 阿里云、火山云：true
    // UCLOUD：false（使用内置映射表）
    DynamicRegions bool
}
```

#### BaseProvider 公共基础能力

```go
type BaseProvider struct {
    capabilities ProviderCapabilities
}

// WrapError 统一错误包装（带厂商前缀）
func (b *BaseProvider) WrapError(provider string, err error) error
```

### 二、阿里云适配器（alicloud/）

#### 2.1 SDK 依赖

```go
import (
    openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
    ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
)
```

安装命令：
```bash
go get github.com/alibabacloud-go/ecs-20140526/v7
```

#### 2.2 客户端创建（client.go）

```go
func NewClient(ak, sk, region string) (*ecs.Client, error) {
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
    return ecs.NewClient(config)
}
```

#### 2.3 主要 API 方法

| 功能 | 阿里云 API | 请求参数 | 说明 |
|------|-----------|----------|------|
| 查询实例 | `DescribeInstances` | RegionId, ZoneId, InstanceName, PageNumber, PageSize | 支持多条件过滤 |
| 查询地域 | `DescribeRegions` | 无 | 动态查询所有地域 |
| 查询可用区 | `DescribeZones` | RegionId | 指定地域下的可用区 |

#### 2.4 实例字段映射（converter.go）

| CloudInstance 字段 | 阿里云字段 | 转换逻辑 |
|-------------------|-----------|----------|
| InstanceID | InstanceId | 直接赋值 |
| Name | InstanceName | 直接赋值 |
| IP | PublicIpAddress.IpAddress[0] | 优先公网 IP，其次 EipAddress.IpAddress |
| PrivateIP | VpcAttributes.PrivateIpAddress.IpAddress[0] | VPC 内网 IP |
| Region | RegionId | 直接赋值 |
| Zone | ZoneId | 直接赋值 |
| Status | Status | Running→running, Stopped→stopped |
| OS | OSName | 直接赋值 |
| CPU | Cpu | 直接赋值 |
| MemoryMB | Memory * 1024 | 阿里云返回 GB，需转 MB |
| DiskGB | 磁盘累加 | DataDisks + SystemDisk |

#### 2.5 地域名称映射

```go
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

#### 2.6 错误处理

```go
func (p *Provider) wrapError(err error) error {
    if err == nil {
        return nil
    }
    // 解析阿里云错误码
    switch errorCode {
    case "InvalidAccessKeyId.NotFound":
        return fmt.Errorf("AccessKey ID 不存在")
    case "SignatureDoesNotMatch":
        return fmt.Errorf("签名验证失败，请检查 AccessKey Secret")
    case "InvalidRegionId":
        return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-hangzhou、cn-shanghai")
    case "UnauthorizedOperation":
        return fmt.Errorf("无权限执行此操作，请检查 AccessKey 是否有 ECS 权限")
    }
    return fmt.Errorf("[阿里云] %w", err)
}
```

### 三、UCLOUD 适配器（ucloud/）

#### 3.1 SDK 依赖

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

#### 3.2 客户端创建（client.go）

```go
func NewClient(ak, sk, region string) (*uhost.UHostClient, error) {
    if ak == "" || sk == "" {
        return nil, fmt.Errorf("UCLOUD AccessKey ID 和 Secret 不能为空")
    }
    if region == "" {
        return nil, fmt.Errorf("UCLOUD 地域不能为空，如 cn-bj2、cn-sh2")
    }

    config := ucloud.NewConfig()
    config.Region = region
    credential := auth.NewKeyPairCredential(ak, sk)
    return uhost.NewClient(&config, credential), nil
}
```

#### 3.3 主要 API 方法

| 功能 | UCLOUD API | 请求参数 | 说明 |
|------|-----------|----------|------|
| 查询实例 | `DescribeUHostInstance` | Region, Zone, UHostIds, Offset, Limit | 支持多条件过滤 |
| 查询地域 | 无 | - | 使用内置映射表 |
| 查询可用区 | 无 | - | 使用内置映射表 |

#### 3.4 实例字段映射（converter.go）

| CloudInstance 字段 | UCLOUD 字段 | 转换逻辑 |
|-------------------|-------------|----------|
| InstanceID | UHostId | 直接赋值 |
| Name | Name | 直接赋值 |
| IP | IPSet[0].IP | 公网 IP（Type=VIP/BGP） |
| PrivateIP | IPSet[0].PrivateIP | 内网 IP |
| Region | 请求参数传入 | 实例数据无 Region 字段 |
| Zone | Zone | 直接赋值 |
| Status | State | Running→running, Stopped→stopped |
| OS | OsName | 直接赋值 |
| CPU | CPU | 直接赋值（核数） |
| MemoryMB | Memory | 直接赋值（MB） |
| DiskGB | DiskSpace | 直接赋值（GB） |

#### 3.5 内置地域映射表（regions.go）

```go
// BuiltinRegions 内置地域列表
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

// BuiltinZones 内置可用区列表（按地域分组）
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

#### 3.6 错误处理

```go
func (p *Provider) wrapError(err error) error {
    if err == nil {
        return nil
    }
    // UCLOUD SDK 返回的错误通常包含 RetCode 字段
    // 根据错误码映射友好提示
    return fmt.Errorf("[UCLOUD] %w", err)
}
```

### 四、前端调整

#### 4.1 云厂商选项

前端已有静态选项，无需调整：

```typescript
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'tencent', label: '腾讯云' },
];
```

#### 4.2 功能差异处理

UCLOUD 不支持动态查询地域，前端调用 `listCloudRegions` 时：
- 阿里云、火山云：调用云 API 获取
- UCLOUD：返回后端内置映射表

前端无需区分，由后端适配器统一处理。

## 实现计划

### 阶段一：公共基础能力

1. 新增 `base.go`，定义 `ProviderCapabilities` 和 `BaseProvider`
2. 更新火山云适配器，嵌入 `BaseProvider`

### 阶段二：阿里云适配器

1. 添加 SDK 依赖
2. 实现 `alicloud/client.go` - ECS 客户端封装
3. 实现 `alicloud/converter.go` - 实例数据转换
4. 实现 `alicloud/provider.go` - CloudProvider 接口实现
5. 注册到全局 Registry
6. 编写单元测试

### 阶段三：UCLOUD 适配器

1. 添加 SDK 依赖
2. 实现 `ucloud/regions.go` - 内置地域映射表
3. 实现 `ucloud/client.go` - UHost 客户端封装
4. 实现 `ucloud/converter.go` - 实例数据转换
5. 实现 `ucloud/provider.go` - CloudProvider 接口实现
6. 注册到全局 Registry
7. 编写单元测试

### 阶段四：集成验证

1. 使用真实 AccessKey 验证阿里云实例查询
2. 使用真实 AccessKey 验证 UCLOUD 实例查询
3. 验证前端云账号创建、实例查询、导入流程

## 验收标准

1. ✅ 可创建阿里云和 UCLOUD 云账号
2. ✅ 可查询阿里云 ECS 实例列表，字段正确显示
3. ✅ 可查询 UCLOUD UHost 实例列表，字段正确显示
4. ✅ 地域和可用区下拉选项正确展示
5. ✅ 实例导入功能正常工作
6. ✅ 单元测试覆盖核心转换逻辑

## 风险与应对

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 阿里云 SDK 版本兼容性 | 编译失败 | 使用最新稳定版 v7 |
| UCLOUD 地域变动 | 可用区列表过时 | 硬编码常用地域，变动频率低 |
| API 限流 | 查询失败 | 添加重试逻辑，提示用户稍后重试 |
| 字段映射错误 | 数据显示异常 | 充分测试，对比实际 API 返回 |
