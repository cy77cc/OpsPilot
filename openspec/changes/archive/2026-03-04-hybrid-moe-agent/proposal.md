# Proposal: Hybrid MOE Agent 架构重构

## 问题背景

当前AI助手的Agent选择机制存在严重的扩展性问题：

### 现状分析

1. **硬编码的Agent选择逻辑**
   - `selectAgent()` 中使用 switch/if-else 硬编码场景映射
   - 关键词匹配规则写死在代码中
   - 添加新Agent需要修改多处代码并重新部署

2. **粗粒度专家划分**
   - 仅5-6个专家，每个专家承担过多职责
   - `ops_expert` 混合了 host、os、container
   - `service_expert` 混合了 service、config、job、topology

3. **单一专家模式**
   - 只选择单个专家处理请求
   - 无法利用多专家协作处理复杂问题
   - 专家间无法传递上下文

4. **配置分散**
   - Agent定义在 `platform_agent.go`
   - Scene映射在 `scene_context.go`
   - 两者需要手动同步维护

## 目标

### 主要目标

1. **配置化专家注册表** - 通过YAML定义专家能力，支持热加载
2. **混合路由策略** - Scene精确匹配 → 关键词规则 → 语义相似度
3. **独立调度器** - 实现主从协作模式，支持多专家协调
4. **细粒度专家** - 拆分为12个专注领域的专家

### 非目标

- 不改变现有的工具实现（tools_registry.go等）
- 不改变前端交互逻辑
- 不引入额外的数据库依赖

## 范围

### 核心组件

| 组件 | 文件 | 说明 |
|------|------|------|
| 专家注册表 | `internal/ai/experts/registry.go` | YAML配置加载，专家实例管理 |
| 专家配置 | `configs/experts.yaml` | 12个专家的完整定义 |
| 混合路由器 | `internal/ai/experts/router.go` | 三层fallback路由策略 |
| 调度器 | `internal/ai/experts/orchestrator.go` | 主从协作执行逻辑 |
| 执行器 | `internal/ai/experts/executor.go` | 专家调用，上下文传递 |
| 聚合器 | `internal/ai/experts/aggregator.go` | 结果合并，最终输出 |
| 场景映射 | `configs/scene_mappings.yaml` | Scene到专家的映射配置 |

### 接口变更

```go
// 新接口
type ExpertRegistry interface {
    GetExpert(name string) (*Expert, bool)
    ListExperts() []*Expert
    Reload() error
}

type HybridRouter interface {
    Route(ctx context.Context, req *RouteRequest) *RouteDecision
}

type Orchestrator interface {
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)
}

// PlatformAgent 重构
type PlatformAgent struct {
    registry   *ExpertRegistry
    router     *HybridRouter
    orchestrator *Orchestrator
}
```

### 废弃接口

- `selectAgent()` - 移至HybridRouter
- `selectAgentByScene()` - 移至HybridRouter
- `sceneRegistry` (map) - 迁移到YAML配置
- `filterToolsByPrefix()` - 移至Expert配置

## 影响分析

### 影响的文件

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/ai/platform_agent.go` | 重构 | 移除硬编码，接入新组件 |
| `internal/service/ai/scene_context.go` | 迁移 | 配置迁移到YAML |
| `internal/service/ai/scene_handler.go` | 适配 | 接入新的Registry |

### 向后兼容

- 现有的Stream/Generate接口保持不变
- scene参数解析逻辑保持兼容
- 现有API路由无需修改

### 风险评估

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| 配置加载失败 | 中 | 内置默认配置，graceful degradation |
| 路由决策错误 | 低 | 多层fallback，最终有default兜底 |
| 性能影响 | 低 | 配置缓存，路由计算轻量 |

## 验收标准

1. 专家可通过YAML配置添加，无需改代码
2. 路由策略可通过配置调整
3. 支持主从协作模式处理复杂请求
4. 现有功能无回归
5. 单元测试覆盖率 > 80%

## 时间线

- 阶段1：专家注册表 + 配置文件 (2天)
- 阶段2：混合路由器 (2天)
- 阶段3：调度器 + 执行器 (3天)
- 阶段4：聚合器 + 集成测试 (2天)
