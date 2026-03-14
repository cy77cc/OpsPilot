# AI API

## Overview

AI streaming now uses a `plan -> execute -> replan` transport built on top of the current turn/block runtime.

```text
/api/v1/ai/chat
  -> internal/service/ai HTTPHandler
  -> internal/ai Orchestrator
  -> internal/ai/runtime SSEConverter
  -> SSE events
  -> web/src/components/AI/Copilot.tsx
  -> turnLifecycle.ts
  -> messageBlocks.ts / AssistantMessageBlocks.tsx
```

The backend still emits some legacy compatibility events, but the current primary visualization path is:

- lifecycle: `meta`, `turn_started`, `turn_state`, `turn_done`, `done`, `error`
- plan/execute/replan: `phase_started`, `phase_complete`, `plan_generated`, `step_started`, `step_complete`, `replan_triggered`
- content/detail: `delta`, `tool_call`, `tool_result`, `approval_required`

Response headers:

- `X-AI-Runtime-Mode`: `model_first | compatibility | disabled`
- `X-AI-Compatibility-Enabled`: `true | false`
- `X-AI-Model-First-Enabled`: `true | false`
- `X-AI-Turn-Block-Streaming-Enabled`: `true | false`

## Chat

### `POST /api/v1/ai/chat`

Starts one assistant turn and streams SSE events.

Request:

```json
{
  "sessionId": "optional-session-id",
  "message": "查看 kube-system 中 cilium pod 状态",
  "context": {
    "scene": "deployment:k8s"
  }
}
```

Response:

- `Content-Type: text/event-stream`
- event stream may contain both native runtime events and compatibility events

Recommended native event order:

```text
meta
turn_started
phase_started(planning)
delta*
plan_generated?
phase_complete(planning)
phase_started(executing)
step_started*
tool_call*
tool_result*
step_complete*
replan_triggered?
phase_started(replanning)?
approval_required?
delta*
phase_complete(executing)?
turn_state
done
```

## Native SSE Events

### `meta`

```json
{
  "session_id": "session-1",
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "runtime_mode": "model_first",
  "model_first_enabled": true,
  "compatibility_enabled": false,
  "turn_block_streaming_enabled": true
}
```

### `turn_started`

```json
{
  "turn_id": "turn-1",
  "session_id": "session-1"
}
```

### `turn_state`

```json
{
  "turn_id": "turn-1",
  "plan_id": "plan-1",
  "status": "running"
}
```

### `phase_started`

```json
{
  "phase": "planning",
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "status": "loading",
  "title": "整理执行步骤",
  "summary": "正在分析并整理执行计划"
}
```

Fields:

- `phase`: `planning | executing | replanning`
- `status`: usually `loading | running | success`
- `title`: user-visible phase title
- `summary`: optional user-visible detail

### `phase_complete`

```json
{
  "phase": "planning",
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "status": "success",
  "summary": "已提取结构化计划"
}
```

### `plan_generated`

```json
{
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "steps": [
    {
      "id": "step-1",
      "content": "检查集群状态",
      "tool_hint": "get_cluster_info"
    },
    {
      "id": "step-2",
      "content": "获取 deployment 列表",
      "tool_hint": "list_deployments"
    }
  ],
  "total": 2
}
```

### `step_started`

```json
{
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "step_id": "step-1",
  "title": "检查集群状态",
  "status": "running",
  "tool_name": "get_cluster_info",
  "params": {
    "namespace": "kube-system"
  }
}
```

### `tool_call`

```json
{
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "step_id": "step-1",
  "tool_name": "get_cluster_info",
  "params": {
    "namespace": "kube-system"
  }
}
```

### `tool_result`

```json
{
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "step_id": "step-1",
  "tool_name": "get_cluster_info",
  "summary": "集群状态正常",
  "result": {
    "ok": true,
    "data": "3 pods found"
  }
}
```

### `step_complete`

```json
{
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "step_id": "step-1",
  "status": "success",
  "summary": "集群状态正常"
}
```

### `replan_triggered`

```json
{
  "plan_id": "plan-1",
  "turn_id": "turn-1",
  "reason": "发现后续需要重新规划",
  "summary": "当前执行流已切换到重新规划阶段"
}
```

### `approval_required`

```json
{
  "id": "approval-1",
  "plan_id": "plan-1",
  "step_id": "step-2",
  "checkpoint_id": "cp-1",
  "title": "重启异常 Pod",
  "tool_name": "k8s_restart_pod",
  "risk_level": "high",
  "mode": "mutating",
  "summary": "该步骤需要审批后继续执行",
  "params": {
    "namespace": "default",
    "pod_name": "app-1"
  }
}
```

### `delta`

Final assistant answer stream:

```json
{
  "content_chunk": "Cilium pod 当前运行正常。"
}
```

### `done`

Closes the turn and may include the persisted session snapshot.

## Compatibility Events

The server may still emit these for compatibility clients:

- `rewrite_result`
- `planner_state`
- `plan_created`
- `stage_delta`
- `step_update`
- `replan_started`
- `summary`
- `thinking_delta`

New frontend work should treat them as fallback input only. The primary rendering path is native turn/block plus the native plan/execute/replan lifecycle events above.

## Resume

### `POST /api/v1/ai/resume/step`

Canonical resume endpoint.

Request:

```json
{
  "session_id": "session-1",
  "plan_id": "plan-1",
  "step_id": "step-2",
  "approved": true,
  "reason": "approved by operator"
}
```

Response:

```json
{
  "resumed": true,
  "interrupted": false,
  "session_id": "session-1",
  "plan_id": "plan-1",
  "step_id": "step-2",
  "status": "completed",
  "message": "审批已通过，待审批步骤会继续执行。"
}
```

### `POST /api/v1/ai/resume/step/stream`

Streams the continuation after approval. It can emit the same native events as `/api/v1/ai/chat`, especially:

- `meta`
- `phase_started`
- `step_complete`
- `tool_result`
- `done`

### Compatibility aliases

- `POST /api/v1/ai/approval/respond`: alias for `/api/v1/ai/resume/step`
- `POST /api/v1/ai/adk/resume`: legacy ADK compatibility endpoint

## Sessions

### `GET /api/v1/ai/sessions?scene=<scene>`

Lists sessions for the current user and scene.

### `GET /api/v1/ai/sessions/current?scene=<scene>`

Returns the latest session for the current user and scene.

### `GET /api/v1/ai/sessions/:id?scene=<scene>`

Returns one session with full persisted messages.

Assistant session replay may include:

- `turns`
- `blocks`
- `content`
- legacy `thoughtChain`
- `recommendations`

## Notes

- The primary frontend implementation path is `Copilot.tsx -> turnLifecycle.ts -> messageBlocks.ts -> AssistantMessageBlocks.tsx`
- `summary` is not a replacement for `delta`
- `approval_required` should be treated as an execution gate, not as the final answer body
