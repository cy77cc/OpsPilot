# Spec: AI Tool API

## Overview

定义 AI 工具相关的 HTTP API 接口，包括能力查询、预览、执行、审批等。

## Requirements

### REQ-API-001: 工具能力列表

系统 SHALL 提供获取所有工具能力的接口。

**验收标准:**
- WHEN GET /ai/capabilities
- THEN 返回所有注册工具的能力描述
- AND 包含 name, description, mode, risk, schema

### REQ-API-002: 工具参数提示

系统 SHALL 提供获取工具参数提示的接口。

**验收标准:**
- WHEN GET /ai/tools/:name/params/hints
- THEN 返回该工具的参数提示信息
- AND 包含参数类型、默认值、枚举值等

### REQ-API-003: 工具预览

系统 SHALL 提供工具预览（dry-run）接口。

**验收标准:**
- WHEN POST /ai/tools/preview { tool, params }
- THEN 返回预览结果（不实际执行）
- AND 对于变更工具，返回预期变更摘要

### REQ-API-004: 工具执行

系统 SHALL 提供工具执行接口。

**验收标准:**
- WHEN POST /ai/tools/execute { tool, params, checkpoint_id? }
- THEN 执行指定工具
- AND 对于变更工具，需要有效的 checkpoint_id (来自审批流程)
- AND 返回执行 ID 和状态

### REQ-API-005: 执行状态查询

系统 SHALL 提供执行状态查询接口。

**验收标准:**
- WHEN GET /ai/executions/:id
- THEN 返回执行状态和结果
- AND 包含 status, result, error 等信息

### REQ-API-006: 审批管理

系统 SHALL 提供完整的审批管理接口。

**验收标准:**
- POST /ai/approvals - 创建审批
- GET /ai/approvals - 获取审批列表
- GET /ai/approvals/:id - 获取审批详情
- POST /ai/approvals/:id/approve - 批准审批
- POST /ai/approvals/:id/reject - 拒绝审批

### REQ-API-007: 场景工具

系统 SHALL 提供场景可用工具接口。

**验收标准:**
- WHEN GET /ai/scene/:scene/tools
- THEN 返回该场景可用的工具列表
- AND 包含工具描述和使用提示

### REQ-API-008: 场景提示词

系统 SHALL 提供场景提示词接口。

**验收标准:**
- WHEN GET /ai/scene/:scene/prompts
- THEN 返回该场景的预定义提示词
- AND 包含提示词文本和类型

## API Specifications

### GET /ai/capabilities

**Response:**
```json
{
    "code": 1000,
    "data": [
        {
            "name": "get_cluster_info",
            "description": "获取 Kubernetes 集群信息",
            "mode": "readonly",
            "risk": "low",
            "provider": "local",
            "schema": { ... }
        }
    ]
}
```

### GET /ai/tools/:name/params/hints

**Response:**
```json
{
    "code": 1000,
    "data": {
        "tool": "scale_deployment",
        "params": {
            "deployment": {
                "type": "string",
                "required": true,
                "hint": "部署名称",
                "enum_source": "deployments"
            },
            "replicas": {
                "type": "integer",
                "required": true,
                "hint": "副本数",
                "default": 1
            }
        }
    }
}
```

### POST /ai/tools/preview

**Request:**
```json
{
    "tool": "scale_deployment",
    "params": {
        "deployment": "nginx",
        "replicas": 3
    }
}
```

**Response:**
```json
{
    "code": 1000,
    "data": {
        "preview": {
            "current_replicas": 1,
            "target_replicas": 3,
            "affected_pods": 2
        },
        "dry_run": true
    }
}
```

### POST /ai/tools/execute

**Request:**
```json
{
    "tool": "scale_deployment",
    "params": {
        "deployment": "nginx",
        "replicas": 3
    },
    "checkpoint_id": "cp-abc-def-ghi"
}
```

**Response:**
```json
{
    "code": 1000,
    "data": {
        "id": "exec-uuid",
        "tool": "scale_deployment",
        "status": "running",
        "created_at": "2026-03-13T16:00:00Z"
    }
}
```

### POST /ai/approvals/:id/approve

**Request:**
```json
{
    "reason": "确认扩容"
}
```

**Response:**
```json
{
    "code": 1000,
    "data": {
        "approval": {
            "id": "approval-uuid",
            "status": "approved"
        },
        "execution": {
            "id": "exec-uuid",
            "status": "running"
        }
    }
}
```

### POST /ai/resume/step/stream - SSE 流式恢复

用于聊天窗口内的审批确认，返回 SSE 流继续对话。

**Request:**
```json
{
    "checkpoint_id": "cp-abc-def-ghi",
    "approved": true,
    "reason": "确认扩容"
}
```

**Response:** SSE Stream
```
event: stage_delta
data: {"stage":"user_action","status":"success","description":"已确认执行"}

event: step_update
data: {"step_id":"step-2","label":"scale_deployment","status":"loading"}

event: step_update
data: {"step_id":"step-2","label":"scale_deployment","status":"success","content":"扩容成功"}

event: delta
data: {"content_chunk":"扩容成功，"}

event: delta
data: {"content_chunk":"nginx 部署现在有 3 个副本在运行。"}

event: done
data: {}
```

### POST /ai/resume/step - 非流式恢复

用于需要同步返回结果的场景。

**Request:**
```json
{
    "checkpoint_id": "cp-abc-def-ghi",
    "approved": true,
    "reason": "确认扩容"
}
```

**Response:**
```json
{
    "code": 1000,
    "data": {
        "status": "completed",
        "result": {
            "message": "扩容成功",
            "replicas": 3
        }
    }
}
```

## Error Codes

| Code | Message | Description |
|------|---------|-------------|
| 4001 | Tool not found | 工具不存在 |
| 4002 | Invalid parameters | 参数校验失败 |
| 4003 | Approval required | 变更工具需要审批 |
| 4004 | Invalid checkpoint_id | 检查点 ID 无效或已过期 |
| 4005 | Approval expired | 审批已过期 |
| 4006 | Execution not found | 执行记录不存在 |
