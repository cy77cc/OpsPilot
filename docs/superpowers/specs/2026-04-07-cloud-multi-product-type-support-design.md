# 云主机多产品类型支持设计

## 背景

当前云主机导入功能只支持各云厂商的标准云服务器产品（UCloud UHost、阿里云 ECS 等），不支持轻量应用服务器等特殊产品类型。用户在香港地域的 UCloud 轻量应用云主机无法被查询到。

## 目标

支持各云厂商的多种主机产品类型，通过账号级别隔离不同产品类型，支持子账号、子项目隔离。

## 产品类型清单

| 云厂商 | product_type | 显示名称 | SDK 服务 |
|--------|-------------|----------|----------|
| ucloud | `uhost` | 云服务器 | uhost |
| ucloud | `ulighthost` | 轻量应用云主机 | ulighthost |
| alicloud | `ecs` | 云服务器 ECS | ecs |
| alicloud | `swas` | 轻量应用服务器 | swas-open |
| volcengine | `ecs` | 云服务器 | ecs |
| volcengine | `swas` | 轻量应用服务器 | swas |

## 数据模型改动

### host_cloud_accounts 表

新增字段：

```go
ProductType string `gorm:"column:product_type;type:varchar(32);not null;default:'ecs'" json:"product_type"`
```

### 完整 HostCloudAccount 模型

```go
type HostCloudAccount struct {
    ID                 uint64    `gorm:"column:id;primaryKey;autoIncrement"`
    Provider           string    `gorm:"column:provider;type:varchar(32);not null;index"`
    ProductType        string    `gorm:"column:product_type;type:varchar(32);not null;default:'ecs'"`
    AccountName        string    `gorm:"column:account_name;type:varchar(128);not null"`
    AccessKeyID        string    `gorm:"column:access_key_id;type:varchar(256);not null"`
    AccessKeySecretEnc string    `gorm:"column:access_key_secret_enc;type:text;not null"`
    RegionDefault      string    `gorm:"column:region_default;type:varchar(64)"`
    ExtraConfig        string    `gorm:"column:extra_config;type:text"`
    Status             string    `gorm:"column:status;type:varchar(32);default:active"`
    CreatedBy          uint64    `gorm:"column:created_by;index"`
    CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
    UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
```

## 后端实现

### 1. 适配器注册机制

注册键格式：`{provider}:{product_type}`

```go
// 注册示例
cloud.Register("ucloud:uhost", ucloud.NewUHostProvider())
cloud.Register("ucloud:ulighthost", ucloud.NewULightHostProvider())
cloud.Register("alicloud:ecs", alicloud.NewECSProvider())
cloud.Register("alicloud:swas", alicloud.NewSWASProvider())
```

### 2. 适配器接口扩展

```go
// CloudProvider 接口新增方法
type CloudProvider interface {
    Name() string
    DisplayName() string
    ProductType() string  // 新增：返回产品类型标识
    ProductTypeName() string  // 新增：返回产品类型显示名称
    Capabilities() ProviderCapabilities
    ValidateCredential(ctx context.Context, ak, sk, region string) error
    ListInstances(ctx context.Context, req ListInstancesRequest) ([]CloudInstance, error)
    ListRegions(ctx context.Context, ak, sk string) ([]Region, error)
    ListZones(ctx context.Context, ak, sk, region string) ([]Zone, error)
}
```

### 3. 新增文件结构

```
internal/service/host/logic/cloud/
├── provider.go          # 接口定义
├── registry.go          # 注册表（修改）
├── types.go             # 类型定义
├── ucloud/
│   ├── provider.go      # UHost 适配器（修改）
│   ├── client.go
│   ├── converter.go
│   ├── regions.go
│   └── ulighthost/      # 新增目录
│       ├── provider.go  # ULightHost 适配器
│       ├── client.go
│       └── converter.go
├── alicloud/
│   ├── provider.go      # ECS 适配器（修改）
│   └── swas/            # 新增目录（预留）
│       └── provider.go
└── volcengine/
    ├── provider.go      # ECS 适配器（修改）
    └── swas/            # 新增目录（预留）
        └── provider.go
```

### 4. ULightHost 适配器实现要点

```go
package ulighthost

import (
    "github.com/ucloud/ucloud-sdk-go/services/ulighthost"
)

type Provider struct{}

func New() *Provider {
    return &Provider{}
}

func (p *Provider) Name() string {
    return "ucloud"
}

func (p *Provider) ProductType() string {
    return "ulighthost"
}

func (p *Provider) ProductTypeName() string {
    return "轻量应用云主机"
}

func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
    client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
    if err != nil {
        return nil, err
    }
    
    listReq := &ulighthost.DescribeULHostInstanceRequest{}
    listReq.Region = &req.Region
    listReq.Limit = ptr.Int(100)
    
    resp, err := client.DescribeULHostInstance(listReq)
    if err != nil {
        return nil, err
    }
    
    // 转换实例数据
    instances := make([]cloud.CloudInstance, 0, len(resp.ULHostSet))
    for _, inst := range resp.ULHostSet {
        instances = append(instances, cloud.CloudInstance{
            InstanceID: *inst.ULHostId,
            Name:       *inst.Name,
            IP:         getPublicIP(inst),
            PrivateIP:  getPrivateIP(inst),
            Region:     req.Region,
            Zone:       *inst.Zone,
            Status:     convertStatus(*inst.State),
            OS:         *inst.OsName,
            CPU:        *inst.CPU,
            MemoryMB:   *inst.Memory,
            DiskGB:     *inst.DiskSize,
        })
    }
    
    return instances, nil
}
```

### 5. 业务逻辑修改

```go
// CreateCloudAccount 创建云账号
func (s *HostService) CreateCloudAccount(ctx context.Context, uid uint64, req CloudAccountReq) (*model.HostCloudAccount, error) {
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
    // ...
}

// QueryCloudInstances 查询云实例
func (s *HostService) QueryCloudInstances(ctx context.Context, req CloudQueryReq) ([]CloudInstanceInfo, error) {
    var account model.HostCloudAccount
    // ...
    
    // 获取适配器：使用 provider:product_type 格式
    providerKey := fmt.Sprintf("%s:%s", account.Provider, account.ProductType)
    provider, err := cloud.GetProvider(providerKey)
    // ...
}
```

## 前端实现

### 1. 添加账号表单

```tsx
// 产品类型选项（根据云厂商动态变化）
const productTypeOptions: Record<string, { value: string; label: string }[]> = {
  ucloud: [
    { value: 'uhost', label: '云服务器' },
    { value: 'ulighthost', label: '轻量应用云主机' },
  ],
  alicloud: [
    { value: 'ecs', label: '云服务器 ECS' },
    { value: 'swas', label: '轻量应用服务器' },
  ],
  volcengine: [
    { value: 'ecs', label: '云服务器' },
    { value: 'swas', label: '轻量应用服务器' },
  ],
};

// Form.Item 动态显示
<Form.Item shouldUpdate>
  {({ getFieldValue }) => {
    const provider = getFieldValue('provider');
    const options = productTypeOptions[provider] || [];
    if (options.length <= 1) return null;
    return (
      <Form.Item name="productType" label="产品类型">
        <Select style={{ width: 160 }} options={options} />
      </Form.Item>
    );
  }}
</Form.Item>
```

### 2. 账号列表展示

表格新增「产品类型」列：

```tsx
{
  title: '产品类型',
  dataIndex: 'productType',
  width: 120,
  render: (v, record) => {
    const label = productTypeOptions[record.provider]?.find(o => o.value === v)?.label || v;
    return <Tag>{label}</Tag>;
  },
}
```

## 实施计划

### 第一阶段：基础架构
1. 修改 `HostCloudAccount` 模型，添加 `ProductType` 字段
2. 修改注册表，支持 `provider:product_type` 格式
3. 更新 `CloudProvider` 接口

### 第二阶段：UCloud 适配器
1. 修改现有 UHost 适配器
2. 实现 ULightHost 适配器
3. 测试香港地域轻量应用云主机查询

### 第三阶段：前端适配
1. 添加账号表单增加产品类型选择
2. 账号列表显示产品类型
3. 兼容现有数据（默认 ecs/uhost）

### 第四阶段：其他云厂商（预留）
1. 阿里云 SWAS 适配器
2. 火山引擎 SWAS 适配器

## 兼容性考虑

- 现有账号默认 `product_type` 为 `ecs` 或 `uhost`
- 数据库迁移自动填充默认值
- 前端对只有一个产品类型的云厂商隐藏选择框