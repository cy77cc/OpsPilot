# 火山云主机导入实现计划

**日期**: 2026-03-19
**设计文档**: [2026-03-19-volcengine-host-import-design.md](../specs/2026-03-19-volcengine-host-import-design.md)

---

## 概述

本计划将设计文档中的火山云主机导入功能转化为具体的实现任务，分阶段执行。

## 实现阶段

### Phase 1: 核心基础设施 (cloud 包)

创建云厂商适配器的核心接口和类型定义。

| 任务 | 文件 | 说明 |
|------|------|------|
| 1.1 | `internal/service/host/logic/cloud/types.go` | 定义 `CloudInstance`、`ListInstancesRequest`、`ProviderInfo` 类型 |
| 1.2 | `internal/service/host/logic/cloud/provider.go` | 定义 `CloudProvider` 接口 |
| 1.3 | `internal/service/host/logic/cloud/registry.go` | 实现云厂商注册表 |

### Phase 2: Mock 适配器

为阿里云和腾讯云创建 mock 实现，保持向后兼容。

| 任务 | 文件 | 说明 |
|------|------|------|
| 2.1 | `internal/service/host/logic/cloud/mock_provider.go` | 实现 `MockProvider`，复用现有 mock 逻辑 |

### Phase 3: 火山云适配器

实现火山云 SDK 集成。

| 任务 | 文件 | 说明 |
|------|------|------|
| 3.1 | `internal/service/host/logic/cloud/volcengine/client.go` | 封装火山云 ECS SDK 客户端 |
| 3.2 | `internal/service/host/logic/cloud/volcengine/converter.go` | 实现实例数据转换逻辑 |
| 3.3 | `internal/service/host/logic/cloud/volcengine/provider.go` | 实现 `CloudProvider` 接口 |

### Phase 4: 服务层集成

重构现有服务层代码，使用新的 provider 模式。

| 任务 | 文件 | 说明 |
|------|------|------|
| 4.1 | `internal/service/host/logic/cloud.go` | 重构 `QueryCloudInstances` 使用注册表 |
| 4.2 | `internal/service/host/routes.go` | 新增 `GET /cloud/providers` 路由 |
| 4.3 | `internal/service/host/handler/cloud.go` | 新增 `ListCloudProviders` handler |

### Phase 5: 前端更新

更新前端组件支持火山云。

| 任务 | 文件 | 说明 |
|------|------|------|
| 5.1 | `web/src/pages/Hosts/HostCloudImportPage.tsx` | 下拉选项增加火山云，默认选中火山云 |
| 5.2 | `web/src/types/host.ts` | `CloudProvider` 枚举增加 `VOLCENGINE` |
| 5.3 | `web/src/api/modules/hosts.ts` | 新增 `listCloudProviders` API 调用 |

### Phase 6: 测试

编写单元测试和集成测试。

| 任务 | 文件 | 说明 |
|------|------|------|
| 6.1 | `internal/service/host/logic/cloud/volcengine/converter_test.go` | 测试数据转换逻辑 |
| 6.2 | `internal/service/host/logic/cloud/registry_test.go` | 测试注册表功能 |

---

## 详细任务说明

### Task 1.1: 创建 types.go

```go
// internal/service/host/logic/cloud/types.go
package cloud

// ListInstancesRequest 查询实例请求。
type ListInstancesRequest struct {
    AccessKeyID     string
    AccessKeySecret string
    Region          string
    Keyword         string
    PageNumber      int
    PageSize        int
}

// CloudInstance 统一的云实例模型。
type CloudInstance struct {
    InstanceID string `json:"instance_id"`
    Name       string `json:"name"`
    IP         string `json:"ip"`
    PrivateIP  string `json:"private_ip"`
    Region     string `json:"region"`
    Zone       string `json:"zone"`
    Status     string `json:"status"`
    OS         string `json:"os"`
    CPU        int    `json:"cpu"`
    MemoryMB   int    `json:"memory_mb"`
    DiskGB     int    `json:"disk_gb"`
}

// ProviderInfo 云厂商信息。
type ProviderInfo struct {
    Name        string `json:"name"`
    DisplayName string `json:"display_name"`
}
```

### Task 1.2: 创建 provider.go

```go
// internal/service/host/logic/cloud/provider.go
package cloud

import "context"

// CloudProvider 定义云厂商适配器接口。
type CloudProvider interface {
    Name() string
    DisplayName() string
    ValidateCredential(ctx context.Context, ak, sk, region string) error
    ListInstances(ctx context.Context, req ListInstancesRequest) ([]CloudInstance, error)
}
```

### Task 1.3: 创建 registry.go

```go
// internal/service/host/logic/cloud/registry.go
package cloud

import (
    "errors"
    "sync"
)

var (
    ErrProviderNotFound = errors.New("cloud provider not found")
)

type Registry struct {
    providers map[string]CloudProvider
    mu        sync.RWMutex
}

var globalRegistry = NewRegistry()

func NewRegistry() *Registry {
    return &Registry{
        providers: make(map[string]CloudProvider),
    }
}

func Register(p CloudProvider) {
    globalRegistry.Register(p)
}

func GetProvider(name string) (CloudProvider, error) {
    return globalRegistry.GetProvider(name)
}

func ListProviders() []ProviderInfo {
    return globalRegistry.ListProviders()
}

func (r *Registry) Register(p CloudProvider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[p.Name()] = p
}

func (r *Registry) GetProvider(name string) (CloudProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if p, ok := r.providers[name]; ok {
        return p, nil
    }
    return nil, ErrProviderNotFound
}

func (r *Registry) ListProviders() []ProviderInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    list := make([]ProviderInfo, 0, len(r.providers))
    for _, p := range r.providers {
        list = append(list, ProviderInfo{
            Name:        p.Name(),
            DisplayName: p.DisplayName(),
        })
    }
    return list
}
```

### Task 3.1: 火山云客户端

```go
// internal/service/host/logic/cloud/volcengine/client.go
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

### Task 4.1: 重构 cloud.go

将现有的 `QueryCloudInstances` 和 `TestCloudAccount` 方法改为使用注册表模式：

```go
func (s *HostService) QueryCloudInstances(ctx context.Context, req CloudQueryReq) ([]CloudInstance, error) {
    provider, err := cloud.GetProvider(req.Provider)
    if err != nil {
        return nil, err
    }

    // 获取账号凭证
    var account model.HostCloudAccount
    if err := s.svcCtx.DB.WithContext(ctx).First(&account, req.AccountID).Error; err != nil {
        return nil, err
    }

    // 解密 Secret
    secret, err := utils.DecryptText(account.AccessKeySecretEnc, config.CFG.Security.EncryptionKey)
    if err != nil {
        return nil, err
    }

    return provider.ListInstances(ctx, cloud.ListInstancesRequest{
        AccessKeyID:     account.AccessKeyID,
        AccessKeySecret: secret,
        Region:          firstNonEmpty(req.Region, account.RegionDefault),
        Keyword:         req.Keyword,
    })
}
```

---

## 执行顺序

```
Phase 1 (核心基础设施)
    ├── 1.1 types.go
    ├── 1.2 provider.go
    └── 1.3 registry.go
         ↓
Phase 2 (Mock 适配器)
    └── 2.1 mock_provider.go
         ↓
Phase 3 (火山云适配器)
    ├── 3.1 client.go
    ├── 3.2 converter.go
    └── 3.3 provider.go
         ↓
Phase 4 (服务层集成)
    ├── 4.1 重构 cloud.go
    ├── 4.2 routes.go
    └── 4.3 handler
         ↓
Phase 5 (前端更新)
    ├── 5.1 HostCloudImportPage.tsx
    ├── 5.2 types/host.ts
    └── 5.3 api/modules/hosts.ts
         ↓
Phase 6 (测试)
    ├── 6.1 converter_test.go
    └── 6.2 registry_test.go
```

---

## 验收标准

1. **功能验收**
   - [ ] 可以创建火山云账号并保存凭证
   - [ ] 可以查询火山云 ECS 实例列表
   - [ ] 可以导入选中的火山云实例
   - [ ] 阿里云/腾讯云 mock 功能正常

2. **代码质量**
   - [ ] 所有新文件有中文注释
   - [ ] 单元测试覆盖核心转换逻辑
   - [ ] 无 lint 错误

3. **前端验收**
   - [ ] 下拉框显示火山云选项
   - [ ] 默认选中火山云
   - [ ] 导入流程正常

---

## 风险与依赖

| 风险 | 缓解措施 |
|------|---------|
| 火山云 SDK API 变更 | 使用稳定版本 v1.2.21 |
| 现有功能回归 | 保持 mock 适配器兼容 |
| 凭证安全 | 复用现有加密机制 |
