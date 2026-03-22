# AI Tool Call Robustness Design

Date: 2026-03-22

## Goal

增强 tool 调用链路的健壮性，满足以下目标：

- 单个 tool 调用失败可接受，不应中断会话
- 面向全工具集统一增强参数接受能力
- 参数纠错仅覆盖字段名风格/大小写和基础类型转换
- 不做语义推断修复（模型语义错误由模型自行纠正）
- 清理并统一现有 tool 调用失败校验逻辑，减少重复实现

## Scope

In scope:

- 全工具调用前参数规范化层
- 统一 tool invocation 错误分类与恢复入口
- `Chat` 与 `ApprovalWorker` 的一致恢复语义
- 对现有冗余失败校验逻辑的收敛
- 相关单元测试与集成测试补齐
- 基础可观测指标与日志字段增强

Out of scope:

- 参数语义猜测修复
- 模型提示词大改
- 各业务工具逻辑重构
- 前端交互重做

## Problem Statement

更换模型供应商后，tool 参数形态漂移明显增加，典型表现为：

- 字段名大小写或命名风格不一致（camelCase、snake_case、kebab-case 混用）
- 参数类型漂移（数字字符串、布尔字符串等）
- 调用失败后错误校验与恢复路径存在重复实现，不同执行路径行为不完全一致

当前代码已经具备“部分工具失败可降级继续”的能力，但还存在两类问题：

1. 参数容错策略缺少统一入口，导致兼容能力不稳定。
2. 失败校验逻辑在多个位置重复，增加维护成本并造成行为分叉风险。

## Product Decisions

### 1. 全工具统一参数规范化（调用前）

在 tool 调用链路前增加统一参数规范化层，作为所有工具共享入口。

策略：

- 字段名归一化：大小写不敏感，命名风格归一到 schema 标准字段
- 基础类型转换：仅做可确定转换（如 `"2" -> 2`、`"true" -> true`、`2 -> "2"`）
- 保持最小侵入：工具内部默认值与业务校验逻辑不变

边界：

- 不做语义重排（如把 `query:"1h"` 识别成 `time_range`）
- 不做跨字段推断

### 2. 统一可恢复 tool 调用失败分类

将 tool invocation error 的解析与判定抽离为单一分类模块，所有执行路径复用。

结果语义：

- 命中可恢复 tool 调用失败：转为结构化 `tool_result(status=error)`，流程继续
- 未命中：保持 fatal runtime 路径

### 3. 删除重复失败校验逻辑并收敛入口

将现有散落在 `logic.go` 与 `approval_worker.go` 的重复错误判定/解析分支收敛为统一调用点，保证一致行为。

## Architecture

### Component A: Tool Args Normalizer

职责：对 raw tool args 做统一标准化，再交给真实工具处理。

输入：

- `tool_name`
- raw args JSON
- 目标工具 input schema（由 input struct 反射得到）

输出：

- `normalized_args`
- 规范化元数据（命中字段、转换项、失败项）

规则：

- Key canonicalization
  - 大小写不敏感
  - `timeRange/time-range/time_range` 统一映射到 `time_range`
  - 字段匹配优先级固定为：`exact` > `case-insensitive exact` > `normalized-form match`
  - 当多个字段归一化后冲突（如同时存在 `userName` 与 `user_name`）时，按上述优先级选唯一目标；若仍冲突则标记失败并保留原值，不做猜测映射
- Type coercion
  - `string -> int/float/bool`（可确定时）
  - `number/bool -> string`
  - `""` 对可选数值/布尔字段视为“未提供”，按 `null/zero-value` 路径处理，不触发解析报错
  - enum 值做大小写不敏感匹配（如 `OPEN -> open`），仅在唯一匹配时转换
  - 失败时保留原值并记录失败项

### Component B: Tool Invocation Error Classifier

职责：统一识别“可恢复工具调用失败”。

输入：

- `error`
- 可选上下文（agent、event）

输出：

- `RecoverableToolInvocationError`（包含 `call_id`、`tool_name`、`message`）
- 或 `nil`（非可恢复）

解析来源：

- 现有错误格式正则（stream tool call / invoke tool）统一迁入分类器
- 分类边界：
  - `RecoverableToolInvocationError` 仅覆盖调用层错误（参数解码、transport、middleware 调用失败、infra 连接失败等）
  - tool 已成功执行并返回业务错误（如资源不存在、余额不足、权限语义拒绝）按普通 `tool_result` 处理，不归类为 invocation error

### Component C: Shared Recovery Adapter

职责：将分类结果转换为 synthetic `tool_result(error)` 事件并注入投影管线。

被调用方：

- `Chat` 主执行循环
- `ApprovalWorker` resume 执行循环

保证：

- 两条路径行为一致
- 最终 run status 一致映射为 `completed_with_tool_errors`（当存在 tool error）
- 单轮内加入重试熔断：同一 `tool_name + normalized_args` 连续失败达到阈值后，不再自动重试同形态调用，要求模型改参或降级回答

## Data Contract

### Normalization Metadata (for log/metrics only)

```json
{
  "tool_name": "monitor_metric",
  "normalized_keys": ["hostId->host_id", "timeRange->time_range"],
  "coercions": ["host_id:string->int"],
  "coercion_failures": [
    {"field": "count", "provided": "five", "expected": "int"}
  ]
}
```

### Recoverable Tool Error Payload

```json
{
  "ok": false,
  "status": "error",
  "error_type": "tool_invocation",
  "tool_name": "monitor_metric",
  "call_id": "call_xxx",
  "message": "..."
}
```

## Runtime Behavior

### Chat Path

1. 接收模型 tool args
2. 参数规范化
3. 调用工具
4. 若出现 invocation 错误，统一分类
5. 可恢复则注入 `tool_result(error)` 并继续
6. 最终 run 收敛为 `completed` 或 `completed_with_tool_errors`

### Approval Resume Path

复用完全相同流程，不允许单独分叉错误语义。

审批任务持久化参数采用“已规范化参数”作为恢复输入源，不再使用原始未规范化 JSON，避免 resume 时重复规范化导致漂移。

## Redundancy Cleanup Plan

目标是“一个分类器 + 一个恢复适配入口”。

- 抽离并集中：错误正则与解析逻辑
- 合并并替换：`logic.go` 中重复 helper
- 补齐并对齐：`approval_worker.go` 中 stream recv 错误恢复分支
- 删除不再需要的局部重复函数或重复判断分支

## Testing Strategy

### Unit Tests

- 参数规范化
  - key 风格/大小写映射
  - 字段冲突优先级与冲突失败路径
  - 类型转换成功与失败
  - enum 大小写匹配
  - 可选字段空串处理
  - 未知字段处理
- 错误分类器
  - 两类已知错误格式可解析
  - 噪声错误不误判
  - invocation error 与 business error 边界测试
- 恢复适配
  - 分类成功时生成正确 synthetic tool result
  - 单轮重试熔断触发后不再重复同形态调用

### Integration Tests

- Chat：tool invocation error 后继续并 done
- ApprovalWorker：resume 期间相同错误同样继续并 done
- 最终状态：含 tool 错误时为 `completed_with_tool_errors`

## Observability

新增指标：

- `tool_args_normalized_total`
- `tool_args_normalize_failed_total`
- `recoverable_tool_invocation_error_total`

新增日志字段：

- `tool_name`
- `call_id`
- `normalized_keys`
- `coercions`
- `coercion_failures`

## Risks and Mitigations

- 风险：过度类型转换导致参数含义偏移
  - 缓解：仅允许确定性转换；失败不强转
- 风险：统一入口引入兼容回归
  - 缓解：全工具回归测试 + 关键工具场景测试
- 风险：不同执行路径仍出现分叉
  - 缓解：`Chat` 与 `ApprovalWorker` 共享同一恢复适配函数

## Rollout Plan

1. 引入参数规范化组件（先埋点、后启用）
2. 引入统一错误分类器并切换 `Chat`
3. 切换 `ApprovalWorker` 到同一分类器/恢复入口
4. 删除冗余逻辑并补齐测试
5. 观察指标与日志，确认稳定后默认开启

## Acceptance Criteria

- 字段名风格/大小写偏差不再直接导致 tool 调用失败
- 基础类型漂移在可确定情况下被自动纠正
- enum 大小写差异不再导致无效参数
- 可选数值/布尔字段的空串输入不再触发解析失败
- tool invocation error 不中断会话
- `Chat` 与 `ApprovalWorker` 对同类错误输出一致
- 冗余错误校验逻辑被收敛至统一入口
