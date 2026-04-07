# 云主机多产品类型支持实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 支持各云厂商的多种主机产品类型，通过账号级别隔离不同产品类型，实现 UCloud 轻量应用云主机查询支持。

**Architecture:** 扩展现有云厂商适配器架构，将注册键从 `provider` 改为 `provider:product_type` 格式，每个产品类型作为独立适配器注册。数据模型新增 `product_type` 字段存储产品类型标识。

**Tech Stack:** Go 1.21+, GORM, UCloud SDK (ulighthost), React, Ant Design

---

## Task 1: 数据模型扩展

**Files:**
- Modify: `internal/model/node.go`
- Modify: `internal/service/host/logic/cloud.go` (CloudAccountReq struct)

- [ ] **Step 1: 添加 ProductType 字段到 HostCloudAccount 模型**

在 `internal/model/node.go` 的 `HostCloudAccount` struct 中添加 `ProductType` 字段：

```go
// HostCloudAccount 是云厂商账户表模型，存储云 API 凭证。
type HostCloudAccount struct {
	ID                 uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Provider           string    `gorm:"column:provider;type:varchar(32);not null;index" json:"provider"`
	ProductType        string    `gorm:"column:product_type;type:varchar(32);not null;default:'uhost';index" json:"product_type"` // 新增：产品类型
	AccountName        string    `gorm:"column:account_name;type:varchar(128);not null;uniqueIndex:idx_provider_account" json:"account_name"`
	AccessKeyID        string    `gorm:"column:access_key_id;type:varchar(256);not null;uniqueIndex:idx_provider_ak" json:"access_key_id"`
	AccessKeySecretEnc string    `gorm:"column:access_key_secret_enc;type:text;not null" json:"-"`
	RegionDefault      string    `gorm:"column:region_default;type:varchar(64)" json:"region_default"`
	ExtraConfig        string    `gorm:"column:extra_config;type:text" json:"extra_config"`
	Status             string    `gorm:"column:status;type:varchar(32);default:active" json:"status"`
	CreatedBy          uint64    `gorm:"column:created_by;index" json:"created_by"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
```

- [ ] **Step 2: 添加 ProductType 到 CloudAccountReq**

在 `internal/service/host/logic/cloud.go` 的 `CloudAccountReq` struct 中添加 `ProductType` 字段：

```go
// CloudAccountReq 创建云账号请求参数。
type CloudAccountReq struct {
	Provider        string `json:"provider"`
	ProductType     string `json:"product_type"` // 新增：产品类型
	AccountName     string `json:"account_name"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	RegionDefault   string `json:"region_default"`
	// UCloud 额外配置
	ProjectId string `json:"project_id"`
	IsIntl    bool   `json:"is_intl"`
}
```

- [ ] **Step 3: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译成功，无错误

- [ ] **Step 4: Commit**

```bash
git add internal/model/node.go internal/service/host/logic/cloud.go
git commit -m "feat(host): add ProductType field to HostCloudAccount model"
```

---

## Task 2: 扩展 CloudProvider 接口

**Files:**
- Modify: `internal/service/host/logic/cloud/provider.go`
- Modify: `internal/service/host/logic/cloud/types.go`

- [ ] **Step 1: 扩展 CloudProvider 接口**

在 `internal/service/host/logic/cloud/provider.go` 中扩展接口，添加 `ProductType()` 和 `ProductTypeName()` 方法：

```go
// CloudProvider 定义云厂商适配器接口。
type CloudProvider interface {
	// Name 返回云厂商标识。
	Name() string

	// DisplayName 返回云厂商显示名称。
	DisplayName() string

	// ProductType 返回产品类型标识。
	//
	// 返回值:
	//   - 产品类型标识，如 "uhost"、"ulighthost"、"ecs"、"swas"
	ProductType() string

	// ProductTypeName 返回产品类型显示名称。
	//
	// 返回值:
	//   - 用户友好的显示名称，如 "云服务器"、"轻量应用云主机"
	ProductTypeName() string

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

- [ ] **Step 2: 扩展 ProviderInfo 类型**

在 `internal/service/host/logic/cloud/types.go` 中扩展 `ProviderInfo` struct：

```go
// ProviderInfo 云厂商信息。
type ProviderInfo struct {
	// Name 云厂商标识。
	Name string `json:"name"`

	// DisplayName 云厂商显示名称。
	DisplayName string `json:"display_name"`

	// ProductType 产品类型标识。
	ProductType string `json:"product_type"`

	// ProductTypeName 产品类型显示名称。
	ProductTypeName string `json:"product_type_name"`
}
```

- [ ] **Step 3: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译失败，因为现有适配器没有实现新方法（这是预期的）

- [ ] **Step 4: Commit**

```bash
git add internal/service/host/logic/cloud/provider.go internal/service/host/logic/cloud/types.go
git commit -m "feat(cloud): extend CloudProvider interface with ProductType methods"
```

---

## Task 3: 修改注册表支持 provider:product_type 格式

**Files:**
- Modify: `internal/service/host/logic/cloud/registry.go`

- [ ] **Step 1: 修改注册表注册和查询逻辑**

修改 `internal/service/host/logic/cloud/registry.go`，使用 `provider:product_type` 作为注册键：

```go
// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
package cloud

import (
	"errors"
	"sync"
)

// ErrProviderNotFound 云厂商适配器未找到错误。
var ErrProviderNotFound = errors.New("cloud provider not found")

// Registry 云厂商适配器注册表。
type Registry struct {
	providers map[string]CloudProvider
	mu        sync.RWMutex
}

// globalRegistry 全局注册表实例。
var globalRegistry = NewRegistry()

// NewRegistry 创建新的注册表实例。
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]CloudProvider),
	}
}

// Register 注册云厂商适配器到全局注册表。
//
// 注册键格式: provider:product_type (如 "ucloud:uhost", "ucloud:ulighthost")
func Register(p CloudProvider) {
	globalRegistry.Register(p)
}

// GetProvider 从全局注册表获取指定云厂商适配器。
//
// 参数:
//   - key: 适配器键，格式为 "provider:product_type" 或 "provider" (向后兼容)
//
// 返回适配器实例或 ErrProviderNotFound 错误
func GetProvider(key string) (CloudProvider, error) {
	return globalRegistry.GetProvider(key)
}

// ListProviders 列出全局注册表中所有云厂商信息。
func ListProviders() []ProviderInfo {
	return globalRegistry.ListProviders()
}

// Register 注册云厂商适配器。
func (r *Registry) Register(p CloudProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 使用 provider:product_type 作为键
	key := p.Name() + ":" + p.ProductType()
	r.providers[key] = p
}

// GetProvider 获取指定云厂商适配器。
func (r *Registry) GetProvider(key string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// 首先尝试精确匹配 provider:product_type
	if p, ok := r.providers[key]; ok {
		return p, nil
	}
	
	// 向后兼容：如果 key 不包含 ":"，尝试查找默认产品类型
	// 例如 "ucloud" -> 查找 "ucloud:uhost"
	if !containsColon(key) {
		defaultKey := key + ":uhost"
		if p, ok := r.providers[defaultKey]; ok {
			return p, nil
		}
		defaultKey = key + ":ecs"
		if p, ok := r.providers[defaultKey]; ok {
			return p, nil
		}
	}
	
	return nil, ErrProviderNotFound
}

// containsColon 检查字符串是否包含冒号。
func containsColon(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

// ListProviders 列出所有云厂商信息。
func (r *Registry) ListProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, ProviderInfo{
			Name:            p.Name(),
			DisplayName:     p.DisplayName(),
			ProductType:     p.ProductType(),
			ProductTypeName: p.ProductTypeName(),
		})
	}
	return list
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译失败，适配器未实现新方法（预期）

- [ ] **Step 3: Commit**

```bash
git add internal/service/host/logic/cloud/registry.go
git commit -m "refactor(cloud): update registry to use provider:product_type key format"
```

---

## Task 4: 更新现有 UHost 适配器

**Files:**
- Modify: `internal/service/host/logic/cloud/ucloud/provider.go`

- [ ] **Step 1: 添加 ProductType 和 ProductTypeName 方法**

在 `internal/service/host/logic/cloud/ucloud/provider.go` 的 `Provider` struct 中添加方法：

```go
// ProductType 返回产品类型标识。
func (p *Provider) ProductType() string {
	return "uhost"
}

// ProductTypeName 返回产品类型显示名称。
func (p *Provider) ProductTypeName() string {
	return "云服务器"
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译失败，其他适配器未实现新方法

- [ ] **Step 3: Commit**

```bash
git add internal/service/host/logic/cloud/ucloud/provider.go
git commit -m "feat(ucloud): add ProductType methods to UHost provider"
```

---

## Task 5: 更新其他现有适配器

**Files:**
- Modify: `internal/service/host/logic/cloud/alicloud/provider.go`
- Modify: `internal/service/host/logic/cloud/volcengine/provider.go`
- Modify: `internal/service/host/logic/cloud/mock_provider.go`

- [ ] **Step 1: 更新阿里云适配器**

在 `internal/service/host/logic/cloud/alicloud/provider.go` 添加方法：

```go
// ProductType 返回产品类型标识。
func (p *Provider) ProductType() string {
	return "ecs"
}

// ProductTypeName 返回产品类型显示名称。
func (p *Provider) ProductTypeName() string {
	return "云服务器 ECS"
}
```

- [ ] **Step 2: 更新火山引擎适配器**

在 `internal/service/host/logic/cloud/volcengine/provider.go` 添加方法：

```go
// ProductType 返回产品类型标识。
func (p *Provider) ProductType() string {
	return "ecs"
}

// ProductTypeName 返回产品类型显示名称。
func (p *Provider) ProductTypeName() string {
	return "云服务器"
}
```

- [ ] **Step 3: 更新 Mock 适配器**

在 `internal/service/host/logic/cloud/mock_provider.go` 添加方法：

```go
// ProductType 返回产品类型标识。
func (p *MockProvider) ProductType() string {
	return "default"
}

// ProductTypeName 返回产品类型显示名称。
func (p *MockProvider) ProductTypeName() string {
	return "默认产品"
}
```

- [ ] **Step 4: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译成功

- [ ] **Step 5: Commit**

```bash
git add internal/service/host/logic/cloud/alicloud/provider.go \
        internal/service/host/logic/cloud/volcengine/provider.go \
        internal/service/host/logic/cloud/mock_provider.go
git commit -m "feat(cloud): add ProductType methods to all existing providers"
```

---

## Task 6: 创建 ULightHost 适配器

**Files:**
- Create: `internal/service/host/logic/cloud/ucloud/ulighthost/provider.go`
- Create: `internal/service/host/logic/cloud/ucloud/ulighthost/client.go`
- Create: `internal/service/host/logic/cloud/ucloud/ulighthost/converter.go`

- [ ] **Step 1: 创建目录结构**

```bash
mkdir -p /root/project/OpsPilot/internal/service/host/logic/cloud/ucloud/ulighthost
```

- [ ] **Step 2: 创建 client.go**

创建 `/root/project/OpsPilot/internal/service/host/logic/cloud/ucloud/ulighthost/client.go`：

```go
// Package ulighthost 提供 UCloud 轻量应用云主机查询适配器实现。
package ulighthost

import (
	"context"

	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Client ULightHost 客户端封装。
type Client struct {
	client *ulighthost.ULightHostClient
}

// NewClient 创建 ULightHost 客户端。
func NewClient(ak, sk, region string) (*Client, error) {
	config := ucloud.NewConfig()
	config.Region = region

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	return &Client{
		client: ulighthost.NewClient(&config, &credential),
	}, nil
}

// DescribeULHostInstance 查询轻量应用云主机实例列表（带重试）。
func (c *Client) DescribeULHostInstance(ctx context.Context, req *ulighthost.DescribeULHostInstanceRequest) (*ulighthost.DescribeULHostInstanceResponse, error) {
	return cloud.DoWithRetry(ctx, "ucloud-ulighthost", cloud.DefaultRetryConfig, "DescribeULHostInstance", func() (*ulighthost.DescribeULHostInstanceResponse, error) {
		return c.client.DescribeULHostInstance(req)
	})
}
```

- [ ] **Step 3: 创建 converter.go**

创建 `/root/project/OpsPilot/internal/service/host/logic/cloud/ucloud/ulighthost/converter.go`：

```go
package ulighthost

import (
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// ConvertInstance 将 ULightHost 实例转换为统一的 CloudInstance 模型。
func ConvertInstance(inst *ulighthost.ULHostInstanceSet, region string) *cloud.CloudInstance {
	return &cloud.CloudInstance{
		InstanceID: inst.ULHostId,
		Name:       inst.Name,
		IP:         getPublicIP(inst),
		PrivateIP:  getPrivateIP(inst),
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
func getPublicIP(inst *ulighthost.ULHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		if ip.Type == "Bgp" || ip.Type == "Internation" {
			if ip.IP != "" {
				return ip.IP
			}
		}
	}
	return ""
}

// getPrivateIP 获取内网 IP 地址。
func getPrivateIP(inst *ulighthost.ULHostInstanceSet) string {
	for _, ip := range inst.IPSet {
		if ip.Type == "Private" && ip.IP != "" {
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
```

- [ ] **Step 4: 创建 provider.go**

创建 `/root/project/OpsPilot/internal/service/host/logic/cloud/ucloud/ulighthost/provider.go`：

```go
// Package ulighthost 提供 UCloud 轻量应用云主机查询适配器实现。
package ulighthost

import (
	"context"
	"fmt"

	"github.com/ucloud/ucloud-sdk-go/services/uaccount"
	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Provider ULightHost 适配器。
type Provider struct{}

// New 创建 ULightHost 适配器实例。
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

// ProductType 返回产品类型标识。
func (p *Provider) ProductType() string {
	return "ulighthost"
}

// ProductTypeName 返回产品类型显示名称。
func (p *Provider) ProductTypeName() string {
	return "轻量应用云主机"
}

// Capabilities 返回能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证凭证是否有效。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	limit := 1
	req := &ulighthost.DescribeULHostInstanceRequest{}
	req.Region = &region
	req.Limit = &limit

	_, err = client.DescribeULHostInstance(ctx, req)
	if err != nil {
		return fmt.Errorf("ULightHost 凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	input := &ulighthost.DescribeULHostInstanceRequest{}
	input.Region = &req.Region

	// 可用区过滤
	if req.Zone != "" && req.Zone != "undefined" && req.Zone != "all" {
		input.Zone = &req.Zone
	}

	// 分页参数
	limit := 100
	if req.PageSize > 0 {
		limit = req.PageSize
	}
	input.Limit = &limit

	if req.PageNumber > 1 {
		offset := (req.PageNumber - 1) * req.PageSize
		input.Offset = &offset
	}

	output, err := client.DescribeULHostInstance(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询 ULightHost 实例失败: %w", p.wrapError(err))
	}

	instances := make([]cloud.CloudInstance, 0, len(output.ULHostSet))
	for i := range output.ULHostSet {
		converted := ConvertInstance(&output.ULHostSet[i], req.Region)

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

// ListRegions 查询地域列表（复用 UHost 的地域查询）。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	config := ucloud.NewConfig()
	config.Region = "cn-bj2"

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	uaccountClient := uaccount.NewClient(&config, &credential)

	req := &uaccount.GetRegionRequest{}
	output, err := cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*uaccount.GetRegionResponse, error) {
		return uaccountClient.GetRegion(req)
	})
	if err != nil {
		return nil, fmt.Errorf("查询地域失败: %w", p.wrapError(err))
	}

	regionSet := make(map[string]bool)
	for _, r := range output.Regions {
		regionSet[r.Region] = true
	}

	regions := make([]cloud.Region, 0, len(regionSet))
	for regionId := range regionSet {
		regions = append(regions, cloud.Region{
			RegionId:  regionId,
			LocalName: getRegionLocalName(regionId),
		})
	}
	return regions, nil
}

// ListZones 查询可用区列表（复用 UHost 的可用区查询）。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	config := ucloud.NewConfig()
	config.Region = region

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	uaccountClient := uaccount.NewClient(&config, &credential)

	req := &uaccount.GetRegionRequest{}
	output, err := cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*uaccount.GetRegionResponse, error) {
		return uaccountClient.GetRegion(req)
	})
	if err != nil {
		return nil, fmt.Errorf("查询可用区失败: %w", p.wrapError(err))
	}

	zoneSet := make(map[string]bool)
	for _, r := range output.Regions {
		if r.Region == region {
			zoneSet[r.Zone] = true
		}
	}

	zones := make([]cloud.Zone, 0, len(zoneSet))
	for zoneId := range zoneSet {
		zones = append(zones, cloud.Zone{
			ZoneId:    zoneId,
			LocalName: getZoneLocalName(zoneId),
		})
	}
	return zones, nil
}

// wrapError 包装错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	if strings.Contains(errStr, "160") {
		return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-bj2、hk")
	}
	if strings.Contains(errStr, "170") {
		return fmt.Errorf("认证失败，请检查 AccessKey ID 和 Secret")
	}
	if strings.Contains(errStr, "171") {
		return fmt.Errorf("AccessKey 无效")
	}

	return fmt.Errorf("[ULightHost] %w", err)
}
```

- [ ] **Step 5: 添加缺少的 imports 到 provider.go**

更新 provider.go 的 imports：

```go
import (
	"context"
	"fmt"
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/uaccount"
	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)
```

- [ ] **Step 6: 添加 regions.go（复用 UHost 的地域名称）**

创建 `/root/project/OpsPilot/internal/service/host/logic/cloud/ucloud/ulighthost/regions.go`：

```go
package ulighthost

// 地域和可用区名称映射（复用 UHost 的定义）
var regionNames = map[string]string{
	"cn-bj2":  "华北二（北京）",
	"cn-sh2":  "华东二（上海）",
	"cn-gd":   "华南一（广州）",
	"hk":      "香港",
	"tw-tp":   "台北",
	"sg":      "亚太一（新加坡）",
	"us-ca":   "美国西（洛杉矶）",
	"us-ws":   "美国东（华盛顿）",
	"ge-fra":  "欧洲（法兰克福）",
}

var zoneNames = map[string]string{
	"hk-01": "香港可用区A",
	"hk-02": "香港可用区B",
	"sg-01": "亚太一（新加坡）可用区A",
	"sg-02": "亚太一（新加坡）可用区B",
}

func getRegionLocalName(regionId string) string {
	if name, ok := regionNames[regionId]; ok {
		return name
	}
	return regionId
}

func getZoneLocalName(zoneId string) string {
	if name, ok := zoneNames[zoneId]; ok {
		return name
	}
	return zoneId
}
```

- [ ] **Step 7: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译成功

- [ ] **Step 8: Commit**

```bash
git add internal/service/host/logic/cloud/ucloud/ulighthost/
git commit -m "feat(ucloud): add ULightHost provider for lightweight cloud instances"
```

---

## Task 7: 注册 ULightHost 适配器

**Files:**
- Modify: `internal/service/host/logic/cloud.go`

- [ ] **Step 1: 添加 ULightHost 注册**

在 `internal/service/host/logic/cloud.go` 的 `init()` 函数中注册 ULightHost 适配器：

```go
import (
	// ... existing imports ...
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/ucloud/ulighthost"
)

// init 初始化云厂商适配器注册表。
func init() {
	// 注册火山云适配器
	cloud.Register(volcengine.New())

	// 注册阿里云适配器
	cloud.Register(alicloud.New())

	// 注册 UCLOUD UHost 适配器
	cloud.Register(ucloud.New())

	// 注册 UCLOUD ULightHost 适配器
	cloud.Register(ulighthost.New())

	// 注册 Mock 适配器（腾讯云，待实现）
	cloud.Register(cloud.NewMockProvider("tencent", "腾讯云"))
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/service/host/logic/cloud.go
git commit -m "feat(cloud): register ULightHost provider"
```

---

## Task 8: 更新业务逻辑支持 ProductType

**Files:**
- Modify: `internal/service/host/logic/cloud.go`

- [ ] **Step 1: 更新 CreateCloudAccount 方法**

修改 `CreateCloudAccount` 方法，保存 `ProductType` 字段：

找到 `CreateCloudAccount` 函数，在创建 `HostCloudAccount` 时添加 `ProductType` 字段处理：

```go
func (s *HostService) CreateCloudAccount(ctx context.Context, uid uint64, req CloudAccountReq) (*model.HostCloudAccount, error) {
	if strings.TrimSpace(config.CFG.Security.EncryptionKey) == "" {
		return nil, errors.New("security.encryption_key is required")
	}
	if req.Provider == "" || req.AccountName == "" || req.AccessKeyID == "" || req.AccessKeySecret == "" {
		return nil, errors.New("provider/account_name/access_key_id/access_key_secret are required")
	}

	secretEnc, err := utils.EncryptText(req.AccessKeySecret, config.CFG.Security.EncryptionKey)
	if err != nil {
		return nil, err
	}

	// 默认产品类型
	productType := req.ProductType
	if productType == "" {
		productType = getDefaultProductType(req.Provider)
	}

	acc := &model.HostCloudAccount{
		Provider:           req.Provider,
		ProductType:        productType,
		AccountName:        req.AccountName,
		AccessKeyID:        req.AccessKeyID,
		AccessKeySecretEnc: secretEnc,
		RegionDefault:      req.RegionDefault,
		Status:             "active",
		CreatedBy:          uid,
	}

	// UCloud 额外配置：ProjectId（子账户必填）和 IsIntl（国际版）
	if req.Provider == "ucloud" && (req.ProjectId != "" || req.IsIntl) {
		extraConfig := map[string]interface{}{}
		if req.ProjectId != "" {
			extraConfig["project_id"] = req.ProjectId
		}
		if req.IsIntl {
			extraConfig["is_intl"] = true
		}
		extraConfigJSON, _ := json.Marshal(extraConfig)
		acc.ExtraConfig = string(extraConfigJSON)
	}

	if err := s.svcCtx.DB.WithContext(ctx).Create(acc).Error; err != nil {
		return nil, err
	}

	acc.AccessKeyID = utils.MaskAccessKey(acc.AccessKeyID)

	return acc, nil
}

// getDefaultProductType 获取云厂商的默认产品类型。
func getDefaultProductType(provider string) string {
	switch provider {
	case "ucloud":
		return "uhost"
	case "alicloud", "volcengine":
		return "ecs"
	default:
		return "default"
	}
}
```

- [ ] **Step 2: 更新 ListCloudAccounts 方法**

修改 `ListCloudAccounts` 方法，返回 `product_type` 字段：

```go
func (s *HostService) ListCloudAccounts(ctx context.Context, provider string) ([]model.HostCloudAccount, error) {
	query := s.svcCtx.DB.WithContext(ctx).Model(&model.HostCloudAccount{}).
		Select("id", "provider", "product_type", "account_name", "access_key_id", "region_default", "extra_config", "status", "created_by", "created_at", "updated_at")
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	var list []model.HostCloudAccount
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}

	for i := range list {
		list[i].AccessKeyID = utils.MaskAccessKey(list[i].AccessKeyID)
	}

	return list, nil
}
```

- [ ] **Step 3: 更新 QueryCloudInstances 方法**

修改 `QueryCloudInstances` 方法，使用 `provider:product_type` 格式获取适配器：

找到 `QueryCloudInstances` 函数，修改获取适配器的逻辑：

```go
func (s *HostService) QueryCloudInstances(ctx context.Context, req CloudQueryReq) ([]CloudInstanceInfo, error) {
	if req.AccountID == 0 {
		return nil, errors.New("account_id is required")
	}

	var account model.HostCloudAccount
	if err := s.svcCtx.DB.WithContext(ctx).First(&account, req.AccountID).Error; err != nil {
		return nil, err
	}

	// 获取云厂商适配器：使用 provider:product_type 格式
	providerKey := fmt.Sprintf("%s:%s", account.Provider, account.ProductType)
	provider, err := cloud.GetProvider(providerKey)
	if err != nil {
		return nil, err
	}

	secret, err := utils.DecryptText(account.AccessKeySecretEnc, config.CFG.Security.EncryptionKey)
	if err != nil {
		return nil, err
	}

	region := firstNonEmpty(req.Region, account.RegionDefault)

	listReq := cloud.ListInstancesRequest{
		AccessKeyID:     account.AccessKeyID,
		AccessKeySecret: secret,
		Region:          region,
		Zone:            req.Zone,
		Keyword:         req.Keyword,
	}

	instances, err := provider.ListInstances(ctx, listReq)
	if err != nil {
		return nil, err
	}

	result := make([]CloudInstanceInfo, 0, len(instances))
	for _, inst := range instances {
		result = append(result, CloudInstanceInfo{
			InstanceID: inst.InstanceID,
			Name:       inst.Name,
			IP:         inst.IP,
			Region:     inst.Region,
			Status:     inst.Status,
			OS:         inst.OS,
			CPU:        inst.CPU,
			MemoryMB:   inst.MemoryMB,
			DiskGB:     inst.DiskGB,
		})
	}

	return result, nil
}
```

- [ ] **Step 4: 添加 fmt import**

确保 `cloud.go` 有 `fmt` import：

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	// ... rest of imports
)
```

- [ ] **Step 5: 编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
```

Expected: 编译成功

- [ ] **Step 6: Commit**

```bash
git add internal/service/host/logic/cloud.go
git commit -m "feat(cloud): update business logic to support ProductType"
```

---

## Task 9: 更新前端添加产品类型选择

**Files:**
- Modify: `web/src/pages/Hosts/HostCloudImportPage.tsx`
- Modify: `web/src/api/modules/hosts.ts`

- [ ] **Step 1: 更新 API 类型定义**

在 `web/src/api/modules/hosts.ts` 中更新 `CloudAccount` 接口和 `createCloudAccount` 方法：

```typescript
export interface CloudAccount {
  id: string;
  provider: string;
  productType: string;  // 新增
  accountName: string;
  accessKeyId: string;
  regionDefault: string;
  status: string;
}

// 更新 listCloudAccounts 方法
async listCloudAccounts(provider?: string): Promise<ApiResponse<CloudAccount[]>> {
  const res = await apiService.get<any>('/hosts/cloud/accounts', { params: { provider } });
  const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
  return {
    ...res,
    data: rawList.map((x: any) => ({
      id: String(x.id),
      provider: x.provider,
      productType: x.product_type || 'uhost',  // 新增
      accountName: x.account_name,
      accessKeyId: x.access_key_id,
      regionDefault: x.region_default,
      status: x.status,
    })),
  };
}

// 更新 createCloudAccount 方法
async createCloudAccount(payload: { 
  provider: string; 
  productType?: string;  // 新增
  accountName: string; 
  accessKeyId: string; 
  accessKeySecret: string; 
  regionDefault?: string;
  projectId?: string;
  isIntl?: boolean 
}): Promise<ApiResponse<CloudAccount>> {
  const res = await apiService.post<any>('/hosts/cloud/accounts', {
    provider: payload.provider,
    product_type: payload.productType || '',  // 新增
    account_name: payload.accountName,
    access_key_id: payload.accessKeyId,
    access_key_secret: payload.accessKeySecret,
    region_default: payload.regionDefault || '',
    project_id: payload.projectId || '',
    is_intl: payload.isIntl || false,
  });
  // ... rest of the method
}
```

- [ ] **Step 2: 更新前端页面添加产品类型选择**

在 `web/src/pages/Hosts/HostCloudImportPage.tsx` 中添加产品类型选项和表单项：

在文件顶部添加产品类型配置：

```typescript
// 产品类型选项（根据云厂商动态变化）
const productTypeOptions: Record<string, { value: string; label: string }[]> = {
  ucloud: [
    { value: 'uhost', label: '云服务器' },
    { value: 'ulighthost', label: '轻量应用云主机' },
  ],
  alicloud: [
    { value: 'ecs', label: '云服务器 ECS' },
    // { value: 'swas', label: '轻量应用服务器' },  // 预留
  ],
  volcengine: [
    { value: 'ecs', label: '云服务器' },
    // { value: 'swas', label: '轻量应用服务器' },  // 预留
  ],
  tencent: [
    { value: 'cvm', label: '云服务器' },
  ],
};
```

在表单中添加产品类型选择（在 provider Select 之后）：

```tsx
<Form.Item shouldUpdate>
  {({ getFieldValue }) => {
    const provider = getFieldValue('provider');
    const options = productTypeOptions[provider] || [];
    if (options.length <= 1) return null;
    return (
      <Form.Item name="productType" rules={[{ required: true, message: '请选择产品类型' }]}>
        <Select style={{ width: 140 }} placeholder="产品类型" options={options} />
      </Form.Item>
    );
  }}
</Form.Item>
```

更新 `createAccount` 方法：

```typescript
const createAccount = async () => {
  const values = await accountForm.validateFields();
  try {
    await Api.hosts.createCloudAccount({
      provider: values.provider,
      productType: values.productType,  // 新增
      accountName: values.accountName,
      accessKeyId: values.accessKeyId,
      accessKeySecret: values.accessKeySecret,
      regionDefault: values.regionDefault,
      projectId: values.projectId,
      isIntl: values.isIntl,
    });
    message.success('云账号创建成功');
    accountForm.resetFields();
    loadAccounts();
  } catch (err: any) {
    message.error(err?.message || '创建失败');
  }
};
```

更新账号列表表格，添加产品类型列：

```tsx
{
  title: '产品类型',
  dataIndex: 'productType',
  width: 120,
  render: (v, record) => {
    const label = productTypeOptions[record.provider]?.find(o => o.value === v)?.label || v || '云服务器';
    return <Tag>{label}</Tag>;
  },
},
```

- [ ] **Step 3: 编译验证**

```bash
cd /root/project/OpsPilot/web && npm run build 2>&1 | head -20
```

Expected: 编译成功（忽略第三方库的类型错误）

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/Hosts/HostCloudImportPage.tsx web/src/api/modules/hosts.ts
git commit -m "feat(web): add product type selection for cloud accounts"
```

---

## Task 10: 测试验证

**Files:**
- Manual testing

- [ ] **Step 1: 重启后端服务**

```bash
# 停止现有服务
pkill -f OpsPilot || true

# 启动服务
cd /root/project/OpsPilot && go run . --config configs/config.yaml &
```

- [ ] **Step 2: 检查数据库迁移**

确认 `product_type` 列已添加：

```bash
# 使用 Go 代码检查
cat <<'EOF' > /tmp/check_db.go
package main

import (
    "fmt"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "os"
)

func main() {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=devops sslmode=disable",
        os.Getenv("POSTGRES_HOST"),
        os.Getenv("POSTGRES_PORT"),
        os.Getenv("POSTGRES_USER"),
        os.Getenv("POSTGRES_PASSWORD"),
    )
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    
    var result []map[string]interface{}
    db.Raw("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'host_cloud_accounts' AND column_name = 'product_type'").Scan(&result)
    if len(result) > 0 {
        fmt.Println("product_type column exists:", result)
    } else {
        fmt.Println("product_type column NOT found")
    }
}
EOF
source /root/project/OpsPilot/.env && go run /tmp/check_db.go
```

Expected: `product_type column exists`

- [ ] **Step 3: 测试添加 UCloud 轻量应用云主机账号**

在前端页面：
1. 选择「UCloud」云厂商
2. 选择「轻量应用云主机」产品类型
3. 填写 AccessKey 信息
4. 地域填写 `hk`
5. 点击添加

- [ ] **Step 4: 测试查询香港地域实例**

1. 选择刚创建的 UCloud 轻量应用云主机账号
2. 选择地域 `hk`
3. 点击查询实例

Expected: 能够查询到香港地域的轻量应用云主机实例

- [ ] **Step 5: 检查后端日志**

```bash
tail -50 /root/project/OpsPilot/log/app.log | grep -i ulighthost
```

Expected: 有 ULightHost 相关的查询日志

---

## Task 11: 清理和最终提交

- [ ] **Step 1: 删除调试代码**

移除之前添加的调试日志（如果有）：

```bash
# 检查是否有 slog.Debug 语句需要移除
grep -r "slog.Debug" internal/service/host/logic/cloud/ucloud/
```

- [ ] **Step 2: 最终编译验证**

```bash
cd /root/project/OpsPilot && go build ./...
cd /root/project/OpsPilot/web && npm run build
```

- [ ] **Step 3: 最终 Commit**

```bash
git add -A
git commit -m "feat(cloud): complete multi-product type support with ULightHost adapter"
```