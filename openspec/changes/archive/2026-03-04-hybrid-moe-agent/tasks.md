# Tasks: Hybrid MOE Agent 实现

## 阶段1: 基础设施 (专家注册表)

- [x] 创建专家注册表包结构 `internal/ai/experts/`
- [x] 定义专家配置结构体 `ExpertConfig`, `Expert`
- [x] 实现 `ExpertRegistry` 接口
  - [x] `NewExpertRegistry()` 构造函数
  - [x] `Load()` 从YAML加载配置
  - [x] `GetExpert()` 获取专家实例
  - [x] `ListExperts()` 列出所有专家
  - [x] `Reload()` 热重载配置
- [x] 实现专家实例化逻辑
  - [x] 根据tool_patterns过滤工具
  - [x] 构建ReAct Agent
  - [x] 设置Persona
- [x] 实现专家匹配方法
  - [x] `MatchByKeywords()` 关键词匹配
  - [x] `MatchByDomain()` 领域匹配
- [x] 创建配置文件 `configs/experts.yaml`
  - [x] 定义12个专家的完整配置
  - [x] 添加配置验证逻辑
- [x] 编写单元测试
  - [x] 配置加载测试
  - [x] 专家实例化测试
  - [x] 匹配逻辑测试
- [x] 添加配置默认值和fallback机制

## 阶段2: 混合路由器

- [x] 定义路由相关结构体
  - [x] `RouteRequest` 路由请求
  - [x] `RouteDecision` 路由决策
  - [x] `ExecutionStrategy` 执行策略
- [x] 创建场景映射配置 `configs/scene_mappings.yaml`
  - [x] 迁移现有sceneRegistry
  - [x] 定义所有场景的专家映射
- [x] 实现 `HybridRouter` 接口
  - [x] `NewHybridRouter()` 构造函数
  - [x] `Route()` 主路由方法
- [x] 实现三层路由策略
  - [x] `routeByScene()` Scene精确匹配
  - [x] `routeByKeywords()` 关键词规则匹配
  - [x] `routeByDomain()` 领域语义匹配 (可选)
  - [x] `routeDefault()` 默认兜底
- [x] 实现 `SceneMappings` 配置加载
- [x] 编写单元测试
  - [x] Scene路由测试
  - [x] 关键词路由测试
  - [x] Fallback测试
- [x] 添加路由决策日志和监控

## 阶段3: 调度器与执行器

- [x] 定义执行相关结构体
  - [x] `ExecuteRequest` 执行请求
  - [x] `ExecuteResult` 执行结果
  - [x] `ExpertTrace` 执行追踪
  - [x] `ExecutionPlan` 执行计划
  - [x] `ExecutionStep` 执行步骤
  - [x] `ExpertResult` 专家结果
- [x] 实现 `Orchestrator` 调度器
  - [x] `NewOrchestrator()` 构造函数
  - [x] `Execute()` 主执行方法
  - [x] `buildPlan()` 构建执行计划
  - [x] `executePlan()` 执行计划
  - [x] `aggregateResults()` 聚合结果
- [x] 实现 `ExpertExecutor` 执行器
  - [x] `NewExpertExecutor()` 构造函数
  - [x] `ExecuteStep()` 执行单步
  - [x] `buildExpertMessage()` 构建专家输入
  - [x] 支持上下文传递 (priorResults)
- [x] 实现执行策略
  - [x] `StrategySingle` 单专家执行
  - [x] `StrategySequential` 串行执行
  - [x] `StrategyParallel` 并行执行 (可选)
- [x] 编写单元测试
  - [x] 计划构建测试
  - [x] 串行执行测试
  - [ ] 上下文传递测试
  - [ ] 错误处理测试
- [x] 添加执行超时和取消支持

## 阶段4: 聚合器

- [x] 定义聚合相关结构体
  - [x] `AggregationMode` 聚合模式
- [x] 实现 `ResultAggregator` 聚合器
  - [x] `NewResultAggregator()` 构造函数
  - [x] `Aggregate()` 聚合方法
- [x] 实现模板聚合 `aggregateByTemplate()`
  - [x] 定义聚合模板
  - [x] 模板渲染逻辑
- [x] 实现LLM聚合 `aggregateByLLM()` (可选)
  - [x] 构建总结prompt
  - [x] 调用LLM生成最终响应
- [x] 编写单元测试
  - [x] 模板聚合测试
  - [ ] LLM聚合测试

## 阶段5: PlatformAgent重构

- [x] 重构 `PlatformAgent` 结构体
  - [x] 移除 `experts` map
  - [x] 添加 `registry`, `router`, `orchestrator` 字段
- [x] 更新 `NewPlatformAgent()` 构造函数
  - [x] 初始化ExpertRegistry
  - [x] 初始化HybridRouter
  - [x] 初始化Orchestrator
  - [x] 保留工具注册逻辑
- [x] 更新 `Stream()` 方法
  - [x] 调用Router获取决策
  - [x] 调用Orchestrator执行
  - [x] 支持流式输出
- [x] 更新 `Generate()` 方法
  - [x] 调用Router获取决策
  - [x] 调用Orchestrator执行
- [x] 移除废弃代码
  - [x] 删除 `selectAgent()` 方法
  - [x] 删除 `selectAgentByScene()` 方法
  - [x] 删除 `filterToolsByPrefix()` 方法
  - [ ] 删除 `sceneFromMessages()` 方法
- [x] 更新 `ToolMetas()` 方法
- [x] 保持 `RunTool()` 方法不变
- [x] 编写集成测试
  - [x] 端到端执行测试
  - [x] 向后兼容测试

## 阶段6: 配置迁移与清理

- [x] 迁移 `scene_context.go` 中的配置
  - [x] 将 `sceneRegistry` 迁移到 `scene_mappings.yaml`
  - [x] 更新 `scene_handler.go` 接入新Registry
- [x] 创建配置加载入口
  - [x] 在应用启动时加载配置
  - [ ] 添加配置热重载支持 (可选)
- [x] 清理废弃代码
  - [x] 移除 `scene_context.go` 中的硬编码map
  - [x] 更新相关import
- [x] 更新文档
  - [x] 添加配置说明文档
  - [x] 添加专家开发指南

## 阶段7: 测试与验证

- [x] 编写端到端测试
  - [x] 测试各场景的路由决策
  - [x] 测试多专家协作
  - [x] 测试错误场景
- [x] 性能测试
  - [x] 路由延迟测试
  - [x] 并发执行测试
- [x] 回归测试
  - [x] 现有功能无回归
  - [x] API兼容性验证
- [x] 集成测试
  - [x] 与前端联调
  - [x] 与其他模块集成

## 配置文件清单

| 文件 | 说明 | 状态 |
|------|------|------|
| `configs/experts.yaml` | 专家注册表配置 | 待创建 |
| `configs/scene_mappings.yaml` | 场景映射配置 | 待创建 |

## 代码文件清单

| 文件 | 说明 | 状态 |
|------|------|------|
| `internal/ai/experts/registry.go` | 专家注册表 | 待创建 |
| `internal/ai/experts/router.go` | 混合路由器 | 待创建 |
| `internal/ai/experts/orchestrator.go` | 调度器 | 待创建 |
| `internal/ai/experts/executor.go` | 执行器 | 待创建 |
| `internal/ai/experts/aggregator.go` | 聚合器 | 待创建 |
| `internal/ai/experts/types.go` | 类型定义 | 待创建 |
| `internal/ai/experts/config.go` | 配置加载 | 待创建 |
| `internal/ai/platform_agent.go` | 重构 | 待修改 |
| `internal/service/ai/scene_context.go` | 清理 | 待修改 |
| `internal/service/ai/scene_handler.go` | 适配 | 待修改 |

## 依赖关系

```
阶段1 (Registry) ─────────────────────────────────────────────┐
                                                              │
阶段2 (Router) ───────────────────────────────────────────────┤
                                                              │
阶段3 (Orchestrator + Executor) ──────────────────────────────┤
                                                              │
阶段4 (Aggregator) ───────────────────────────────────────────┤
                                                              │
阶段5 (PlatformAgent重构) ◀───────────────────────────────────┘
         │
         ▼
阶段6 (配置迁移)
         │
         ▼
阶段7 (测试验证)
```
