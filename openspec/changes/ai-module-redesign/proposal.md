# Proposal: AI Module Redesign

## Summary

重构 AI 模块，解决当前编译错误和接口缺失问题，实现 Plan-Execute 架构、带审批的对话模式、页面级场景感知能力。

## Problem Statement

### 当前问题

1. **编译错误**: `internal/aiv2` 目录不存在，但 `handler.go:91` 引用了 `aiv2.NewRuntime()`
2. **接口缺失**: 前端 `ai.ts` 定义了 12 个后端接口，但后端未实现
3. **架构不完整**: 缺少审批流程、场景感知等核心能力

### 影响范围

- 前端 AI 对话功能无法正常工作
- 工具能力查询、预览、执行接口不可用
- 审批流程缺失，变更操作无法安全执行

## Proposed Solution

### 架构选择

- **Agent 模式**: Plan-Execute (Planner → Executor → Replanner)
- **持久化**: Redis + MySQL
- **交互模式**: 带审批的对话
- **场景感知**: 页面级

### 核心组件

1. **Runtime Layer**
   - PlanExecuteAgent 实现
   - Redis Checkpoint Store
   - SSE 流式输出

2. **Tool Layer**
   - Tool Registry（统一工具注册）
   - Approval Gate（审批中间件）
   - Scene Context（场景上下文注入）

3. **API Layer**
   - 补齐 12 个缺失接口
   - 维护现有 9 个接口

## Scope

### In Scope

- Phase 1: 核心运行时修复（编译错误）
- Phase 2: 接口补齐（前端依赖）
- Phase 3: 审批流程实现

### Out of Scope

- Phase 4: 场景感知优化（后续迭代）
- Phase 5: 可观测性（后续迭代）
- 多模型切换、A/B 测试等高级功能

## Success Criteria

1. 编译通过，无错误
2. 前端 AI 对话功能正常工作
3. 工具能力查询、预览、执行接口可用
4. 变更工具触发审批流程，用户可批准/拒绝

## Timeline

- Phase 1: 1-2 天（阻塞问题修复）
- Phase 2: 2-3 天（接口补齐）
- Phase 3: 2-3 天（审批流程）

## Risks

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| Eino ADK API 变化 | 高 | 参考 cookbook 示例代码 |
| Redis 不可用 | 中 | 降级到内存存储 |
| 工具执行失败 | 中 | 错误处理和重试机制 |
