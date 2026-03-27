# 项目注释添加设计文档

## 概述

为 OpsPilot 项目全部 Go 文件添加规范化中文注释，包括 Swagger API 文档注释和代码注释。

## 目标

- **Handler**: 标准 Swagger 注释（@Summary, @Tags, @Param, @Success, @Failure, @Router）
- **Model**: 四层注释（包级、类型级、字段级、方法级）
- **Logic/DAO/工具类**: 包级 + 方法级注释

## 注释规范

### Swagger Handler 注释格式

```go
// ListClusters 获取集群列表。
//
// @Summary 获取集群列表
// @Description 获取当前用户有权限访问的所有集群信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=[]model.Cluster}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters [get]
func (h *Handler) ListClusters(c *gin.Context) {
```

### Model 注释格式

```go
// Cluster 是集群表模型，存储 Kubernetes/OpenShift 集群信息。
//
// 表名: clusters
// 关联:
//   - Node (一对多)
//   - ClusterBootstrapProfile (多对一)
type Cluster struct {
    ID          uint   `gorm:"primaryKey" json:"id"`     // 集群 ID
    Name        string `gorm:"column:name" json:"name"`  // 集群名称 (唯一)
}
```

### Logic/DAO 方法注释格式

```go
// GetByID 根据 ID 获取集群信息。
//
// 参数:
//   - ctx: 上下文
//   - id: 集群 ID
//
// 返回: 集群对象，不存在返回 ErrRecordNotFound
func (r *ClusterDAO) GetByID(ctx context.Context, id uint) (*model.Cluster, error) {
```

## 执行策略

### 并行任务划分（16 Agent）

| Agent | 模块 | 文件范围 |
|-------|------|----------|
| 1 | cluster | internal/service/cluster/**, internal/model/cluster*.go |
| 2 | host | internal/service/host/**, internal/model/host*.go |
| 3 | deployment | internal/service/deployment/**, internal/model/deployment*.go |
| 4 | ai | internal/service/ai/**, internal/ai/**, internal/dao/ai/**, internal/model/ai*.go |
| 5 | user | internal/service/user/**, internal/dao/user/**, internal/model/user*.go |
| 6 | cicd | internal/service/cicd/**, internal/model/cicd*.go |
| 7 | monitoring | internal/service/monitoring/**, internal/model/monitoring*.go |
| 8 | jobs | internal/service/jobs/**, internal/model/job*.go |
| 9 | notification | internal/service/notification/**, internal/model/notification*.go |
| 10 | project | internal/service/project/**, internal/model/project*.go |
| 11 | service | internal/service/service/**, internal/model/service*.go |
| 12 | cmdb | internal/service/cmdb/**, internal/model/cmdb*.go |
| 13 | dashboard | internal/service/dashboard/**, internal/model/dashboard*.go |
| 14 | automation | internal/service/automation/**, internal/model/automation*.go |
| 15 | node + rbac | internal/service/node/**, internal/service/rbac/**, internal/model/node*.go |
| 16 | 基础设施 | internal/middleware/**, internal/httpx/**, internal/xcode/**, internal/svc/**, internal/cache/**, internal/client/**, internal/component/**, internal/runtimectx/** |

### 执行流程

1. **阶段 1**: 创建 Team 和 16 个任务
2. **阶段 2**: 16 个 Agent 并行执行
3. **阶段 3**: 汇总验证（swag init, go build）

## 已有注释处理

- 保留已有注释
- 补充缺失部分
- 统一格式规范

## 注释语言

全中文注释。

## 输出物

1. 修改后的 Go 文件（全部添加规范化注释）
2. 生成的 Swagger 文档（docs/swagger.json, docs/swagger.yaml）
3. 本设计文档

## 验收标准

- `swag init` 执行成功
- `go build` 编译通过
- 所有 Handler 函数有 Swagger 注释
- 所有 Model 类型有类型注释和字段注释
- 所有公开方法有方法注释
