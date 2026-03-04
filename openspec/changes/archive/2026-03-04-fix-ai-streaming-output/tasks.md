# Tasks: 修复流式输出和工具动画

## 阶段1: 核心修复

- [x] 修复 `PlatformAgent.Stream()` 方法
  - [x] 单专家场景使用 `exp.Agent.Stream()` 替代 orchestrator.Execute
  - [x] 保留context传递（包含tool event emitter）
  - [x] 添加fallback到默认agent

## 阶段2: 多专家流式支持

- [x] 在 `Orchestrator` 中添加 `StreamExecute()` 方法
  - [x] 创建输出channel
  - [x] 顺序执行专家步骤
  - [x] 流式转发每个专家的输出
  - [x] 合并最终结果
- [x] 在 `ExpertExecutor` 中添加 `StreamStep()` 方法
  - [x] 使用 `exp.Agent.Stream()` 替代 `Generate()`
  - [x] 保持context传递

## 阶段3: 类型补充

- [x] 确保 `ExecuteRequest` 包含 `History` 字段
- [x] 确保相关类型定义完整

## 阶段4: 测试验证

- [x] 编写单元测试
  - [x] 测试单专家流式输出
  - [ ] 测试tool event触发
  - [ ] 测试context传递
- [ ] 手动测试
  - [ ] 前端验证流式输出效果
  - [ ] 前端验证工具动画效果
  - [ ] 验证多专家场景

## 文件变更清单

| 文件 | 变更类型 |
|------|----------|
| `internal/ai/platform_agent.go` | 修改 |
| `internal/ai/experts/orchestrator.go` | 修改 |
| `internal/ai/experts/executor.go` | 修改 |

## 优先级

1. **阶段1** - 最重要，解决单专家场景（占大多数情况）
2. **阶段2** - 增强多专家场景体验
3. **阶段3** - 类型补充
4. **阶段4** - 测试验证
