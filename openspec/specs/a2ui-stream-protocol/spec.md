# A2UI Stream Protocol — Agent to UI 流式事件协议规范

## 1. 概述

A2UI（Agent-to-UI）是 OpsPilot AI 后端向前端推送流式执行进度的 SSE 协议。  
后端通过 `POST /api/v1/ai/chat` 以 `text/event-stream` 格式持续推送事件，  
前端消费事件并驱动对话气泡、任务清单、工具调用指示器、审批弹框等 UI 状态。

### 1.1 设计目标

- **语义完整**：每个事件承载一种独立语义，前端无需解析 `data` 内容来判断事件类型。
- **无泄漏**：内部 Agent 结构（planner steps JSON、replanner response JSON）不再通过 `delta` 泄露给前端，由专属事件承载。
- **可丢弃**：前端对未知事件类型不报错，直接忽略，保证向前兼容。
- **双向可追溯**：`call_id` 将 `tool_call` 与 `tool_result` 对应，`run_id` 贯穿整个执行生命周期。

### 1.2 适用接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/ai/chat` | 新建/续接对话，返回 SSE 流 |
| POST | `/api/v1/ai/resume/step/stream` | 审批后恢复执行，返回 SSE 流 |

---

## 2. SSE 传输格式

每条事件遵循标准 SSE 格式，字段之间用 `\n` 分隔，事件之间用 `\n\n` 分隔：

```
event: <event_name>
data: <json_payload>

```

- `event` 行：事件名，对应下文各事件类型的字符串值。
- `data` 行：单行 JSON，不换行。
- 心跳保活：服务端每 15 秒发送 `: ping` 注释行防止连接超时。

---

## 3. 事件生命周期

一次完整的 Plan-Execute-Replan 执行流的事件序列如下：

```
POST /api/v1/ai/chat
        │
        ▼
   ┌─────────┐
   │  meta   │  会话初始化：session_id / run_id
   └────┬────┘
        │
        ▼
   ┌──────────────┐
   │ agent_handoff│  路由决策：OpsPilotAgent → DiagnosisAgent
   └──────┬───────┘
          │
          ▼
      ┌──────┐
      │ plan │  初始步骤列表（Planner 输出）
      └──┬───┘
         │
         ▼  ────────────────────── 执行轮次（可重复 N 次）──────────────────────
         │
         ├── delta*         执行器推理文字（流式 token）
         ├── tool_call      工具调用请求
         │       │
         │       ├── tool_result      工具返回（普通工具）
         │       └── tool_approval    等待人工审批（高风险工具）
         │                │
         │           [用户操作 /resume/step/stream]
         │                │
         │           tool_result      审批通过后的工具返回
         │
         ├── delta*         执行器分析/小结文字
         │
         ▼
      ┌────────┐
      │ replan │  更新剩余步骤（Replanner 输出，可多轮）
      └──┬─────┘
         │
         │  （重复执行轮次 + replan，直至 is_final=true）
         │
         ▼
      ┌──────┐
      │ done │  执行完成
      └──────┘
```

**异常路径**：任意阶段出错时发送 `error` 事件，流随即关闭。

---

## 4. 事件类型规范

### 4.1 `meta` — 会话元信息

**触发时机**：连接建立后立即发送，在所有其他事件之前。  
**对应旧事件**：废弃 `init` + `status(running)`，合并为本事件。

```json
{
  "session_id": "474e5391-76b0-426f-bdc5-6d3402130e47",
  "run_id":     "8b2c1a3e-9f4d-4b7e-a1c2-3d4e5f6a7b8c",
  "turn":       1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `session_id` | string | ✅ | 会话 UUID，新会话由服务端生成 |
| `run_id` | string | ✅ | 本次执行 UUID，用于轮询 `/runs/:runId` |
| `turn` | number | ✅ | 本会话内的轮次序号，首轮为 1 |

**前端响应**：存储 `session_id` 用于下一轮发送；存储 `run_id` 用于异常恢复查询。

---

### 4.2 `agent_handoff` — Agent 路由交接

**触发时机**：RouterAgent（OpsPilotAgent）决定将请求转交给子 Agent 时。  
**对应旧事件**：废弃 `intent`，本事件语义更精确（含来源 Agent）。

```json
{
  "from":   "OpsPilotAgent",
  "to":     "DiagnosisAgent",
  "intent": "diagnosis"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `from` | string | ✅ | 发起转交的 Agent 名称 |
| `to` | string | ✅ | 接收执行的子 Agent 名称 |
| `intent` | string | ✅ | 意图类型，见下表 |

**`intent` 枚举值**：

| 值 | 对应 Agent | 语义 |
|----|-----------|------|
| `diagnosis` | DiagnosisAgent | 只读诊断/排查 |
| `change` | ChangeAgent | 变更执行（含 HITL 审批） |
| `qa` | QAAgent | 知识问答/RAG |
| `unknown` | 其他 | 无法识别的意图 |

**前端响应**：展示「正在转交 DiagnosisAgent…」状态提示；依据 `intent` 调整 UI 风格（诊断/变更/问答）。

---

### 4.3 `plan` — 初始执行计划

**触发时机**：Planner 子 Agent 完成任务分解，输出初始步骤列表时。

```json
{
  "steps": [
    "使用 host_list_inventory 获取所有服务器的完整列表",
    "对所有服务器批量执行健康检查命令（uptime / df -h / free -m）",
    "汇总分析检查结果，生成服务器状态报告"
  ],
  "iteration": 0
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `steps` | string[] | ✅ | 有序步骤描述列表，最多 8 条 |
| `iteration` | number | ✅ | 规划轮次，初始计划固定为 `0` |

**前端响应**：在消息区域渲染任务清单 UI，初始状态全部为「待执行」。

---

### 4.4 `replan` — 动态重规划

**触发时机**：Replanner 子 Agent 根据执行结果更新剩余步骤时（每轮 execute 后触发一次）。

**中间轮次（还有剩余步骤）**：

```json
{
  "steps":     ["使用 host_batch_exec_apply 执行健康检查命令"],
  "completed": 2,
  "iteration": 1,
  "is_final":  false
}
```

**最终轮次（执行完成）**：

```json
{
  "steps":     [],
  "completed": 3,
  "iteration": 2,
  "is_final":  true
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `steps` | string[] | ✅ | 剩余待执行步骤（已完成的不再出现） |
| `completed` | number | ✅ | 本轮前已完成的步骤数量 |
| `iteration` | number | ✅ | 重规划轮次，从 `1` 开始递增 |
| `is_final` | boolean | ✅ | `true` 表示所有步骤已完成，`done` 事件随后发送 |

**前端响应**：勾选已完成项，追加/更新剩余步骤；`is_final=true` 时将全部步骤标记为完成。

---

### 4.5 `delta` — 流式文本增量

**触发时机**：Executor 产生推理文字、分析小结或最终答复时，逐 token 流式推送。

```json
{
  "content": "已成功获取所有服务器的完整列表。共发现 **5 台服务器**",
  "agent":   "executor"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | string | ✅ | 本次增量文本片段（UTF-8，可为空字符串） |
| `agent` | string | ❌ | 产生本增量的 Agent 名称，调试用途 |

**重要**：后端不再通过 `delta` 发送 `{"steps":[...]}` 或 `{"response":"..."}` 格式的 JSON 字符串。这些内容分别由 `plan` / `replan` 事件承载，前端侧的 `normalizeVisibleStreamChunk` 降级逻辑可在迁移完成后移除。

**前端响应**：将连续的 `content` 片段拼接并渲染为 Markdown 消息气泡。

---

### 4.6 `tool_call` — 工具调用请求

**触发时机**：Executor 决定调用某工具时，在工具实际执行之前发送。

```json
{
  "call_id":   "call_d29c8f6a989b4749a1d82334",
  "tool_name": "host_exec",
  "arguments": {
    "host_id": 1,
    "command": "uptime"
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `call_id` | string | ✅ | 工具调用 ID，与对应 `tool_result` 的 `call_id` 相同 |
| `tool_name` | string | ✅ | 工具函数名，如 `host_exec`、`k8s_query` |
| `arguments` | object | ✅ | 工具调用参数，结构因工具而异 |

**前端响应**：在消息区域展示「🔧 正在执行 host_exec…」指示器；记录 `call_id → tool_name` 映射，等待 `tool_result` 关闭。

---

### 4.7 `tool_result` — 工具执行结果

**触发时机**：工具执行完成并返回结果时（无论成功或失败）。

```json
{
  "call_id":   "call_d29c8f6a989b4749a1d82334",
  "tool_name": "host_exec",
  "content":   "{\"host_id\":1,\"command\":\"uptime\",\"stdout\":\"10:43:47 up 3 days...\",\"exit_code\":0}"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `call_id` | string | ✅ | 对应 `tool_call` 的 `call_id` |
| `tool_name` | string | ✅ | 工具函数名 |
| `content` | string | ✅ | 工具原始输出，JSON 字符串格式（不再次解析） |

**并发工具**：同一轮次可能并发多个 `tool_call`，对应多个乱序的 `tool_result`，前端通过 `call_id` 配对。

**前端响应**：关闭对应工具调用指示器；将结果以折叠卡片形式展示，默认折叠避免遮挡主要文本。

---

### 4.8 `tool_approval` — 人工审批等待

**触发时机**：执行器调用高风险工具（`risk=high` 或 `command_class=dangerous`）时触发 HITL 中断，等待人工确认。

```json
{
  "approval_id":   "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "call_id":       "call_abc123def456",
  "tool_name":     "host_batch_exec_apply",
  "preview": {
    "command":       "systemctl restart nginx",
    "command_class": "service_control",
    "risk":          "medium",
    "target_count":  5,
    "targets": [
      { "host_id": 1, "hostname": "node-01", "ip": "172.22.208.65" }
    ]
  },
  "timeout_seconds": 300
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `approval_id` | string | ✅ | 审批单 UUID，提交审批时使用 |
| `call_id` | string | ✅ | 对应的工具调用 ID |
| `tool_name` | string | ✅ | 工具函数名 |
| `preview.command` | string | ✅ | 待执行命令 |
| `preview.command_class` | string | ✅ | 命令分类：`readonly` / `service_control` / `dangerous` |
| `preview.risk` | string | ✅ | 风险级别：`low` / `medium` / `high` |
| `preview.target_count` | number | ✅ | 目标主机数量 |
| `preview.targets` | object[] | ✅ | 目标主机列表（含 host_id / hostname / ip） |
| `timeout_seconds` | number | ✅ | 审批超时秒数，超时后自动拒绝 |

**审批操作**：用户确认或拒绝后，调用 `POST /api/v1/ai/resume/step/stream`，执行继续，后续通过同一 SSE 流推送 `tool_result`。

**前端响应**：弹出审批确认对话框（阻塞式），展示命令预览、风险等级、目标主机列表；倒计时超时提示；SSE 连接保持打开状态等待用户操作。

---

### 4.9 `done` — 执行完成

**触发时机**：所有步骤执行完毕，`break_loop` 信号触发后发送。

```json
{
  "run_id":     "8b2c1a3e-9f4d-4b7e-a1c2-3d4e5f6a7b8c",
  "status":     "completed",
  "iterations": 2
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `run_id` | string | ✅ | 本次执行 UUID |
| `status` | string | ✅ | 固定为 `"completed"` |
| `iterations` | number | ✅ | 实际执行的 replan 迭代次数 |

**前端响应**：停止流式动画；关闭所有未完成的工具调用指示器；将任务清单全部标记完成。

---

### 4.10 `error` — 执行出错

**触发时机**：执行过程中发生不可恢复错误时，本事件是流中的最后一条。

```json
{
  "run_id":      "8b2c1a3e-9f4d-4b7e-a1c2-3d4e5f6a7b8c",
  "code":        "TOOL_TIMEOUT",
  "message":     "工具 host_exec 调用超时（55s），请重试。",
  "recoverable": false
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `run_id` | string | ❌ | 本次执行 UUID（若已分配） |
| `code` | string | ✅ | 错误码，见下表 |
| `message` | string | ✅ | 用户可见的中文错误描述 |
| `recoverable` | boolean | ✅ | `true` 表示可重试，`false` 表示需要用户介入 |

**错误码枚举**：

| `code` | 含义 | `recoverable` |
|--------|------|--------------|
| `AI_UNAVAILABLE` | AI 服务未初始化 | false |
| `SESSION_NOT_FOUND` | 会话不存在 | false |
| `TOOL_TIMEOUT` | 工具调用超时（硬超时 55s） | true |
| `TOOL_TIMEOUT_SOFT` | 工具执行较慢软警告（25s） | true |
| `EXECUTION_FAILED` | 执行器内部错误 | false |
| `APPROVAL_TIMEOUT` | 审批超时自动拒绝 | true |
| `APPROVAL_REJECTED` | 用户手动拒绝审批 | true |
| `MAX_ITERATIONS` | 达到最大迭代次数（20） | false |
| `CONTEXT_CANCELLED` | 客户端断开连接 | true |

**前端响应**：停止流式动画；展示错误提示 Toast；若 `recoverable=true` 展示重试按钮。

---

### 4.11 `thinking_delta` — 推理模型思考流（内部保留）

**触发时机**：仅在启用扩展推理模型（如 DeepSeek-R1）时，推送 `<think>` 区块内容。

```json
{
  "content": "用户问的是所有服务器的状态，需要先获取主机列表..."
}
```

> **注意**：本事件为内部保留事件，不在公开 SSE 白名单内（`stream.go` 的 `publicEventNames`）。  
> 如需对前端开放，需先更新白名单并评估信息安全影响。

---

## 5. 后端事件映射规则

后端 `logic.Chat()` 消费 `adk.AgentEvent` 迭代器，按以下规则映射为 A2UI 事件：

| ADK 事件特征 | A2UI 事件 | 触发条件 |
|------------|-----------|---------|
| 首次建立连接 | `meta` | 创建 Run 后立即发送 |
| `event.Action.TransferTo != ""` | `agent_handoff` | `role=tool` + `action.transfer_to` 非空 |
| `event.AgentName == "planner"` + `role=assistant` + content 含 `steps` JSON | `plan` | 解析 steps 数组，iteration=0 |
| `event.AgentName == "replanner"` + `role=assistant` + content 含 `steps` JSON | `replan` | 解析 steps 数组，is_final=false |
| `event.AgentName == "replanner"` + `role=assistant` + content 含 `response` JSON | `replan` | steps=[], is_final=true；response 文本另行以 `delta` 推送 |
| `role=assistant` + `is_streaming=true` + `content` 非空 + 非 planner/replanner | `delta` | 逐 token 推送 content |
| `role=assistant` + `is_streaming=true` + `tool_calls` 非空 | `tool_call` | 每个 tool_call 发送一条 |
| `role=tool` + `tool_name` 非 `transfer_to_agent` | `tool_result` | 工具返回时发送 |
| HITL 中断（`action.interrupt != nil`） | `tool_approval` | 解析 preview 字段 |
| `action.break_loop.done == true` | `done` | 附带 iterations 统计 |
| `event.Err != nil` | `error` | 附带 code 和 message |

---

## 6. 前端处理器接口（TypeScript）

```typescript
// A2UI 协议 v2 事件载荷类型定义

export interface A2UIMeta {
  session_id: string;
  run_id:     string;
  turn:       number;
}

export interface A2UIAgentHandoff {
  from:   string;
  to:     string;
  intent: 'diagnosis' | 'change' | 'qa' | 'unknown';
}

export interface A2UIPlan {
  steps:     string[];
  iteration: 0;
}

export interface A2UIReplan {
  steps:     string[];
  completed: number;
  iteration: number;
  is_final:  boolean;
}

export interface A2UIDelta {
  content: string;
  agent?:  string;
}

export interface A2UIToolCall {
  call_id:   string;
  tool_name: string;
  arguments: Record<string, unknown>;
}

export interface A2UIToolResult {
  call_id:   string;
  tool_name: string;
  content:   string;
}

export interface A2UIToolApprovalPreview {
  command:       string;
  command_class: 'readonly' | 'service_control' | 'dangerous';
  risk:          'low' | 'medium' | 'high';
  target_count:  number;
  targets:       Array<{ host_id: number; hostname: string; ip: string }>;
}

export interface A2UIToolApproval {
  approval_id:     string;
  call_id:         string;
  tool_name:       string;
  preview:         A2UIToolApprovalPreview;
  timeout_seconds: number;
}

export interface A2UIDone {
  run_id:     string;
  status:     'completed';
  iterations: number;
}

export interface A2UIError {
  run_id?:     string;
  code:        string;
  message:     string;
  recoverable: boolean;
}

// 事件处理器集合
export interface A2UIStreamHandlers {
  onMeta?:         (payload: A2UIMeta)         => void;
  onAgentHandoff?: (payload: A2UIAgentHandoff) => void;
  onPlan?:         (payload: A2UIPlan)         => void;
  onReplan?:       (payload: A2UIReplan)       => void;
  onDelta?:        (payload: A2UIDelta)        => void;
  onToolCall?:     (payload: A2UIToolCall)     => void;
  onToolResult?:   (payload: A2UIToolResult)   => void;
  onToolApproval?: (payload: A2UIToolApproval) => void;
  onDone?:         (payload: A2UIDone)         => void;
  onError?:        (payload: A2UIError)        => void;
}
```

---

## 7. 旧版事件迁移说明

| 旧事件 | 新事件 | 迁移策略 |
|--------|--------|---------|
| `init` | `meta` | 重命名；payload 新增 `turn` 字段 |
| `status` | 已废弃 | `running` 状态由 `meta` 隐含；`completed` 状态由 `done` 承载 |
| `intent` | `agent_handoff` | 重命名；payload 新增 `from` 字段，`intent_type` 改为 `intent` |
| `delta.contentChunk` | `delta.content` | 字段改名；`toContentChunk()` 适配层仍读取两者 |
| `delta({"steps":[...]})` | `plan` / `replan` | planner/replanner 输出不再进入 delta；`normalizeVisibleStreamChunk` 可在迁移完成后移除 |
| `delta({"response":"..."})` | `replan(is_final=true)` + `delta` | response 文本以 delta 发送；replan 携带 is_final=true 信号 |
| `progress` | 已废弃 | 由 `replan.completed` 取代 |
| `report_ready` | 已废弃 | 诊断报告通过 `GET /ai/diagnosis/:reportId` 查询 |

**过渡期策略**：后端在同一流中同时发送旧事件和新事件，前端通过版本协商头 `X-AI-Stream-Version: 2` 决定消费哪套。旧事件在完成全量迁移后从后端移除。

---

## 8. 并发工具调用示例

执行器一次发出 5 个并行 `host_exec` 调用的事件序列：

```
event: tool_call
data: {"call_id":"call_001","tool_name":"host_exec","arguments":{"host_id":1,"command":"uptime"}}

event: tool_call
data: {"call_id":"call_002","tool_name":"host_exec","arguments":{"host_id":2,"command":"uptime"}}

event: tool_call
data: {"call_id":"call_003","tool_name":"host_exec","arguments":{"host_id":3,"command":"uptime"}}

event: tool_call
data: {"call_id":"call_004","tool_name":"host_exec","arguments":{"host_id":6,"command":"uptime"}}

event: tool_call
data: {"call_id":"call_005","tool_name":"host_exec","arguments":{"host_id":7,"command":"uptime"}}

event: tool_result
data: {"call_id":"call_003","tool_name":"host_exec","content":"{\"stdout\":\"up 125 days...\"}"}

event: tool_result
data: {"call_id":"call_001","tool_name":"host_exec","content":"{\"stdout\":\"up 3 days...\"}"}

event: tool_result
data: {"call_id":"call_004","tool_name":"host_exec","content":"{\"stdout\":\"up 4 days...\"}"}

event: tool_result
data: {"call_id":"call_002","tool_name":"host_exec","content":"{\"stdout\":\"up 89 days...\"}"}

event: tool_result
data: {"call_id":"call_005","tool_name":"host_exec","content":"{\"stdout\":\"up 4 days...\"}"}
```

前端按 `call_id` 配对，`tool_result` 到达顺序与 `tool_call` 顺序无关。

---

## 9. 完整会话示例（474e5391 诊断会话）

以 `log/ai_events/474e5391-76b0-426f-bdc5-6d3402130e47.jsonl` 的真实执行为基准，
对应的 A2UI 事件序列：

```
meta            → session_id=474e5391..., run_id=..., turn=1
agent_handoff   → from=OpsPilotAgent, to=DiagnosisAgent, intent=diagnosis
plan            → steps=["获取服务器列表","批量健康检查命令预览","批量执行健康检查","汇总报告"], iteration=0

── 第 1 轮执行 ──────────────────────────────────────────────
tool_call       → host_list_inventory, arguments={}
tool_result     → host_list_inventory, content={"total":5,"list":[...]}
delta*          → "已成功获取所有服务器的完整列表。共发现 **5 台服务器**..."
replan          → steps=["批量预览健康检查命令","批量执行健康检查命令"], completed=1, iteration=1, is_final=false

── 第 2 轮执行 ──────────────────────────────────────────────
delta*          → "我将使用 host_batch_exec_preview 工具对所有5台服务器批量执行健康检查命令的预览..."
tool_call       → host_batch_exec_preview, arguments={"command":"uptime",...}
tool_result     → host_batch_exec_preview, content={"command":"uptime","command_class":"readonly","risk":"low",...}
delta*          → "`uptime` 命令预览成功。命令分类为只读，风险级别低..."
tool_call       → host_batch_exec_preview, arguments={"command":"df -h",...}
tool_result     → host_batch_exec_preview, content={"command":"df -h","command_class":"readonly",...}
delta*          → "`df -h` 命令预览成功..."
tool_call       → host_batch_exec_preview, arguments={"command":"free -m",...}
tool_result     → host_batch_exec_preview, content={"command":"free -m","command_class":"readonly",...}
delta*          → "所有三个健康检查命令的预览都已完成..."
replan          → steps=["对5台服务器执行uptime/df-h/free-m命令并汇总"], completed=2, iteration=2, is_final=false

── 第 3 轮执行 ──────────────────────────────────────────────
delta*          → "我将使用 host_batch_exec_apply 工具对所有5台服务器批量执行健康检查命令..."
tool_call       → host_batch_exec_preview, arguments={"command":"uptime",...}   ← 执行前再次预览确认
tool_result     → host_batch_exec_preview, ...
delta*          → "预览通过，现在执行 uptime 命令..."
tool_call × 5  → host_exec, arguments={"host_id":6,"command":"uptime"}         ← 5个并发调用
tool_call × 5  → host_exec, arguments={"host_id":7,"command":"uptime"}
                  ...（host_id: 1, 2, 3, 6, 7）
tool_result × 5 → host_exec（乱序到达，按 call_id 配对）
delta*          → "`uptime` 命令执行成功。现在执行 df -h 命令..."
tool_call × 5  → host_exec, arguments={...,"command":"df -h"}
tool_result × 5 → host_exec
delta*          → "`df -h` 命令执行成功。现在执行 free -m 命令..."
tool_call × 5  → host_exec, arguments={...,"command":"free -m"}
tool_result × 5 → host_exec
delta*          → "所有健康检查命令已成功执行。现在分析数据并生成报告..."
delta*          → "## 📊 服务器健康检查综合报告\n\n### 执行概要..."   ← 最终报告（多个 delta 片段）
replan          → steps=[], completed=4, iteration=3, is_final=true

done            → run_id=..., status=completed, iterations=2
```

> `delta*` 表示该阶段可能有多个连续 delta 事件，前端拼接后渲染。
> 并发 `tool_call × 5` 表示 5 条 tool_call 事件连续发出，对应 5 条乱序 tool_result。

---

## 10. 版本与兼容性

| 版本 | 状态 | 说明 |
|------|------|------|
| v1（当前） | 过渡期 | 后端发送 `init` / `status` / `intent` 等旧事件；前端兼容两套 |
| v2（本文档） | 目标态 | 后端发送新事件集；`normalizeVisibleStreamChunk` 可移除 |

**版本协商**：请求头携带 `X-AI-Stream-Version: 2` 时，后端仅发送 v2 事件；  
不携带或携带 `1` 时，后端同时发送 v1 + v2 事件（过渡期双发）。