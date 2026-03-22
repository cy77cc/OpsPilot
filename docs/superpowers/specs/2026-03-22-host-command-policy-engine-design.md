# Host Command Policy Engine 设计说明

- 日期: 2026-03-22
- 状态: Draft (待评审)
- 作者: AI 协作

## 1. 背景与问题

当前主机命令执行能力存在权限边界不清问题：

1. 诊断场景工具集中仍包含可执行非只读命令的入口。
2. “只读/变更”更多依赖命名与约定，而非统一的强制策略内核。
3. 现有命令分类主要基于字符串匹配，缺少 AST 语义级校验。
4. 失败场景缺乏统一 fail-closed 策略，存在策略绕过风险。

本设计目标是在一次重构中建立统一 Host Command Policy Engine，所有主机命令执行入口统一走策略判定与审批桥接。

## 2. 目标与非目标

### 2.1 目标

1. 主机命令执行工具收敛为两个显式入口：
   - `host_exec_readonly`
   - `host_exec_change`
2. 在执行前统一进行 Shell AST 解析与节点级校验。
3. 对只读入口执行严格 Allowlist 校验。
4. 一旦 AST 解析失败、命中禁用操作符、命中白名单外命令，统一触发审批中断。
5. 不依赖 diagnosis/change agent 之间协作即可完成中断与恢复。
6. 审计链路记录策略决策依据与违规原因，便于追溯。

### 2.2 非目标

1. 本次不设计多租户差异化白名单策略 UI。
2. 本次不引入自动“跨 agent 转交执行”能力。
3. 本次不扩展到非主机执行类工具（仅定义可复用接口）。

## 3. 总体方案

### 3.1 组件划分

新增/重构以下组件：

1. `HostCommandPolicyEngine`（统一策略内核）
2. `HostCommandASTParser`（基于 `mvdan.cc/sh/v3/syntax`）
3. `HostCommandValidator`（AST 节点级校验）
4. `HostExecFacade`（两个工具入口的统一门面）
5. `HostApprovalBridge`（审批中断桥接）

所有主机命令工具入口均先经过 `HostCommandPolicyEngine`，任何直接执行路径均视为违规实现。

### 3.2 工具收敛

保留并对外暴露：

1. `host_exec_readonly`
2. `host_exec_change`

处理策略：

1. 旧执行工具（如 `host_exec`、`host_exec_by_target`、`host_ssh_exec_readonly`）进入兼容期。
2. 兼容期内旧工具内部转调新门面与新策略引擎。
3. 完成调用方迁移后移除旧工具对外暴露。

## 4. 策略引擎设计

### 4.1 输入模型

`PolicyInput`：

1. `ToolName`
2. `AgentRole`（diagnosis/change/inspection）
3. `Target`
4. `CommandRaw`
5. `SessionID` / `RunID` / `CallID` / `CheckpointID`
6. 其他运行时上下文（用户、场景）

### 4.2 输出模型

`PolicyDecision`：

1. `DecisionType`
   - `allow_readonly_execute`
   - `require_approval_interrupt`
   - `deny`（保留）
2. `ReasonCodes`（可多值）
3. `Violations`（结构化详情）
4. `PolicyVersion`
5. `ASTSummary`（可审计摘要）

### 4.3 决策规则（高优先级到低优先级）

1. 解析失败 -> `require_approval_interrupt`
2. AST 命中禁用操作符 -> `require_approval_interrupt`
3. AST 命中白名单外命令 -> `require_approval_interrupt`
4. `host_exec_readonly` 且全部校验通过 -> `allow_readonly_execute`
5. `host_exec_change` -> 默认 `require_approval_interrupt`

默认策略为 fail-closed，任何策略异常均不直接放行执行。

## 5. AST 解析与节点级校验

### 5.1 解析器

使用 `mvdan.cc/sh/v3/syntax` 解析命令为 AST，要求：

1. 输入为单条命令文本。
2. 保留解析错误与位置信息。
3. 输出统一中间结构供 Validator 遍历。

### 5.2 Allowlist 校验

对 AST 中每个可执行命令节点提取基础命令名，必须全部命中 allowlist。

首批 allowlist（可配置）：

1. `cat`
2. `ls`
3. `grep`
4. `top`
5. `free`
6. `df`
7. `tail`

说明：

1. 以上列表只是初始集合，最终以策略配置为准。
2. 初始版本将 `awk` 从全局 allowlist 移除，避免通过 `system(...)` 等能力绕过只读边界。
3. 如未来必须支持 `awk`，必须在 Validator 中增加 `awk` 专属语义校验（显式禁用 `system` 等外部执行能力）后再灰度放开。

### 5.3 禁用操作符校验

命中以下能力即违规：

1. 输出重定向（`>`、`>>`）
2. 后台执行（`&`）
3. 命令替换（`$()`、反引号）

补充规则：

1. 管道符 `|` 允许使用，但管道中每一个子命令都必须独立通过 allowlist 校验。
2. 多命令链路（`;`、`&&`、`||`）不再“一刀切禁用”，改为“逐段校验”：
   - 链路中的每一段命令都必须通过 AST 与 allowlist 校验；
   - 任一分段不通过则触发审批中断；
   - 若全部分段均为只读候选，则允许执行。
3. 对 `&&/||` 的执行语义按 shell 原语保留，但不改变“逐段先验校验”原则。

违规即触发审批中断，不直接执行。

## 6. 工具行为定义

### 6.1 `host_exec_readonly`

1. 输入：`target`、`command`
2. 处理：
   - 调用 `PolicyEngine.Evaluate`
   - `allow_readonly_execute` -> 执行命令并返回结果
   - `require_approval_interrupt` -> 触发审批中断
3. 不允许绕过策略直接执行

### 6.2 `host_exec_change`

1. 输入：`target`、`command`、`reason`（可选）
2. 处理：
   - 调用 `PolicyEngine.Evaluate`
   - 默认进入审批中断
   - 审批通过后执行
3. 参数快照冻结，恢复执行不可改参（至少冻结 `command`、`target`、`agent_role`、`session_id`）

## 7. 审批中断与恢复

### 7.1 中断触发

触发审批中断时写入：

1. 原始命令
2. AST 摘要
3. 命中违规列表
4. 风险等级
5. 策略版本
6. 审批超时时间
7. 中断响应状态（`status=suspended`）与 `approval_id`

行为约定：

1. `HostApprovalBridge` 在触发审批时立即返回挂起状态（如 `{"status":"suspended","approval_id":"..."}`），而不是同步等待人工结果。
2. 当前工具调用在挂起后立刻结束，由会话恢复机制在审批通过后继续，避免 LLM/tool call 因人工审批超时而失败。

### 7.2 恢复执行

1. 审批通过：执行中断时快照参数。
2. 审批拒绝：返回拒绝结果并记录审计事件。
3. 上下文缺失或不匹配：重新中断，不执行。
4. 恢复时必须校验 `approval_id` 与 `session_id`、`agent_role` 绑定关系，防止跨会话/跨角色重放。

## 8. 审计与可观测性

### 8.1 审计字段

每次调用记录：

1. `policy_version`
2. `tool_name`
3. `agent_role`
4. `command_hash`
5. `ast_parse_ok`
6. `violations[]`
7. `decision`
8. `approval_id`（如有）
9. `executed`
10. `exit_code`
11. `approver_id`（如有审批动作）
12. `approval_timestamp`（如有审批动作）
13. `reject_reason`（审批拒绝时）

### 8.2 指标

1. `host_policy_total{decision,reason}`
2. `host_policy_parse_fail_total`
3. `host_policy_violation_total{type}`
4. `host_policy_approval_latency_seconds`

## 9. 错误处理与安全基线

1. Fail-closed：解析器、校验器、策略配置异常均进入审批中断。
2. 参数冻结：审批恢复必须复用中断快照。
3. 本地执行收口：`localhost` 不再走旁路，统一走策略与审批链。

## 10. 测试策略

### 10.1 单元测试

1. AST 解析成功/失败
2. Allowlist 命中/未命中
3. 禁用操作符命中
4. 策略异常 fail-closed
5. 审批恢复参数冻结

### 10.2 工具边界测试

1. `NewDiagnosisTools` 仅暴露 `host_exec_readonly`（不暴露变更执行入口）
2. `NewChangeTools` 暴露 `host_exec_change` 与必要只读工具
3. 旧工具暴露行为符合兼容策略（或已下线）

### 10.3 回归测试

1. 非白名单命令在诊断场景必须触发审批中断。
2. 命中禁用操作符必须触发审批中断。
3. AST 解析失败必须触发审批中断。

## 11. 迁移计划

1. 引入 `HostCommandPolicyEngine` 与 AST Validator。
2. 新增 `host_exec_readonly`、`host_exec_change`。
3. 旧执行工具转调新引擎，保留兼容期。
4. 更新 diagnosis/change 工具装配。
5. 对接审批预览扩展字段（违规原因、AST 摘要）。
6. 完成调用迁移后下线旧工具。

## 12. 风险与缓解

1. 误拦截导致诊断可用性下降
   - 缓解：审批链兜底，不直接失败。
2. 白名单过宽导致越权
   - 缓解：初期从最小白名单启动，逐步扩展。
3. 兼容期行为分叉
   - 缓解：旧工具仅做新门面转发，不保留旧判定逻辑。

## 13. 验收标准

1. 主机执行工具对外仅两类语义入口：readonly/change。
2. 所有主机命令执行都经过统一策略引擎。
3. AST 解析失败/白名单外命令/禁用操作符均进入审批中断。
4. 诊断与变更 agent 协作未实现时，仍可在当前会话完成审批中断与恢复。
5. 测试覆盖关键策略路径且通过。
