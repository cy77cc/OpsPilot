# Spec: AI Approval Gate

## Overview

定义 AI 工具调用的审批流程，实现带审批的对话模式。

## Requirements

### REQ-AP-001: 工具风险分类

系统 SHALL 为每个工具定义风险等级。

**验收标准:**
- GIVEN 工具注册到 Tool Registry
- WHEN 定义工具元数据
- THEN 必须指定 mode (readonly | mutating)
- AND 必须指定 risk (low | medium | high)

### REQ-AP-002: 只读工具自动执行

系统 SHALL 自动执行只读工具，无需审批。

**验收标准:**
- GIVEN 用户调用只读工具 (mode=readonly)
- WHEN Agent 决定执行该工具
- THEN 系统直接执行
- AND 不产生审批中断

### REQ-AP-003: 变更工具审批中断

系统 SHALL 对变更工具产生审批中断。

**验收标准:**
- GIVEN 用户调用变更工具 (mode=mutating)
- WHEN Agent 决定执行该工具
- THEN 系统产生 approval_required 事件
- AND 中断当前执行
- AND 等待用户审批

### REQ-AP-004: 审批恢复

系统 SHALL 支持审批后恢复执行。

**验收标准:**
- GIVEN 审批中断发生
- WHEN 用户批准 (approved=true)
- THEN 系统从中断点恢复执行
- AND 继续后续流程

- WHEN 用户拒绝 (approved=false)
- THEN 系统不执行该工具
- AND 返回拒绝消息给用户

### REQ-AP-005: 审批持久化

系统 SHALL 持久化审批状态。

**验收标准:**
- GIVEN 审批请求创建
- WHEN 审批状态变更
- THEN 状态写入 MySQL ai_approvals 表
- AND 临时状态写入 Redis

## Data Structures

### ApprovalInfo

```go
type ApprovalInfo struct {
    ToolName        string     // 工具名称
    ArgumentsInJSON string     // 工具参数 JSON
    RiskLevel       RiskLevel  // 风险等级
    Summary         string     // 操作摘要
}
```

### ApprovalResult

```go
type ApprovalResult struct {
    Approved   bool    // 是否批准
    Reason     *string // 原因说明
}
```

### RiskLevel

```go
type RiskLevel string

const (
    RiskLow    RiskLevel = "low"
    RiskMedium RiskLevel = "medium"
    RiskHigh   RiskLevel = "high"
)
```

## SSE Events

### approval_required Event

```json
{
    "id": "approval-uuid",
    "tool": "scale_deployment",
    "tool_name": "扩缩容部署",
    "params": {
        "deployment": "nginx",
        "replicas": 3
    },
    "params_json": "{\"deployment\":\"nginx\",\"replicas\":3}",
    "risk": "medium",
    "risk_level": "medium",
    "mode": "mutating",
    "status": "pending",
    "created_at": "2026-03-13T16:00:00Z"
}
```

## Approval Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ tool_call   │────▶│ ApprovalGate│────▶│ Interrupt   │
│ (mutating)  │     │   检查      │     │   等待      │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 继续执行    │◀────│   Resume    │◀────│ 用户审批    │
│ tool_result │     │   恢复      │     │ 批准/拒绝   │
└─────────────┘     └─────────────┘     └─────────────┘
```

## Tool Classification Examples

| 工具 | Mode | Risk | 说明 |
|-----|------|------|------|
| get_cluster_info | readonly | low | 获取集群信息 |
| list_pods | readonly | low | 列出 Pod |
| scale_deployment | mutating | medium | 扩缩容部署 |
| restart_pod | mutating | medium | 重启 Pod |
| delete_deployment | mutating | high | 删除部署 |
| execute_host_command | mutating | high | 执行主机命令 |
