# 前端实现方案

## 1. 当前架构分析

### 1.1 现有文件结构

```
web/src/components/AI/
├── Copilot.tsx                    # AI 助手主入口
├── AIAssistantDrawer.tsx          # 抽屉容器
├── types.ts                       # 类型定义
├── components/
│   ├── ToolCard.tsx               # 工具执行卡片
│   ├── ConfirmationPanel.tsx      # 审批确认面板
│   └── ThoughtChainView.tsx       # 思考链视图
├── hooks/
│   ├── useAIChat.ts               # 聊天状态管理
│   └── useAutoScene.ts            # 场景自动感知
└── constants/
    └── sceneMapping.ts            # 场景映射
```

### 1.2 现有类型定义

```typescript
// web/src/components/AI/types.ts

// 思考阶段
export type ThoughtStageKey =
  | 'rewrite'      // 意图识别
  | 'plan'         // 规划
  | 'execute'      // 执行
  | 'user_action'; // 用户操作

// 思考阶段状态
export type ThoughtStageStatus = 'loading' | 'success' | 'error';

// 思考阶段项
export interface ThoughtStageItem {
  key: ThoughtStageKey;
  title: string;
  description?: string;
  content?: string;
  footer?: string;
  details?: ThoughtStageDetailItem[];
  status: ThoughtStageStatus;
  collapsible?: boolean;
  blink?: boolean;
}
```

### 1.3 现有 SSE 事件处理

```typescript
// web/src/api/modules/ai.ts

export interface AIChatStreamHandlers {
  onMeta?: (payload: SSEMetaEvent) => void;
  onDelta?: (payload: SSEDeltaEvent) => void;
  onThinkingDelta?: (payload: SSEThinkingDeltaEvent) => void;
  onToolCall?: (payload: SSEToolCallEvent) => void;
  onToolResult?: (payload: SSEToolResultEvent) => void;
  onApprovalRequired?: (payload: SSEApprovalRequiredEvent) => void;
  onDone?: (payload: SSEDoneEvent) => void;
  onError?: (payload: SSEErrorEvent) => void;
}
```

### 1.4 差距分析

| 差距 | 说明 | 解决方案 |
|------|------|---------|
| 缺少 PlanStep 类型 | 无法展示步骤列表 | 新增类型定义 |
| 事件处理器不完整 | 新增事件无对应处理 | 扩展 handlers |
| 思考过程展示简单 | 只有折叠面板 | 新增 Timeline 组件 |
| 审批界面简陋 | 只有 Panel 形式 | 新增 Modal 形式 |

---

## 2. 类型定义扩展

### 2.1 新增类型

```typescript
// web/src/components/AI/types.ts

// === 新增: 规划步骤 ===
export interface PlanStep {
  id: string;
  content: string;
  tool_hint?: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  result?: {
    ok: boolean;
    summary?: string;
    error?: string;
  };
}

// === 新增: 工具执行记录 ===
export interface ToolExecution {
  id: string;
  stepId: string;
  toolName: string;
  arguments: Record<string, unknown>;
  result?: string;
  status: 'running' | 'success' | 'error';
  startedAt?: string;
  completedAt?: string;
}

// === 扩展: ThoughtStageItem ===
export interface ThoughtStageItem {
  key: ThoughtStageKey;
  title: string;
  description?: string;
  content?: string;
  footer?: string;
  details?: ThoughtStageDetailItem[];
  status: ThoughtStageStatus;
  collapsible?: boolean;
  blink?: boolean;

  // === 新增字段 ===
  steps?: PlanStep[];           // plan 阶段的步骤列表
  currentStepIndex?: number;    // 当前执行步骤索引
  executions?: ToolExecution[]; // execute 阶段的工具执行记录
  replanReason?: string;        // replan 原因
}

// === 新增: 审批请求扩展 ===
export interface ApprovalRequest {
  approval_id: string;
  tool_name: string;
  risk: 'low' | 'medium' | 'high';
  title: string;
  description: string;
  summary: string;
  details?: Record<string, unknown>;
  params?: Record<string, unknown>;
}

// === 新增: SSE 事件类型 ===
export interface SSEPhaseStartedEvent {
  type: 'phase_started';
  data: { phase: string; title: string; status: string };
}

export interface SSEPlanGeneratedEvent {
  type: 'plan_generated';
  data: {
    plan_id: string;
    steps: Array<{ id: string; content: string; tool_hint?: string }>;
    total: number;
  };
}

export interface SSEStepStartedEvent {
  type: 'step_started';
  data: {
    step_id: string;
    title: string;
    tool_name?: string;
    params?: Record<string, unknown>;
    status: string;
  };
}

export interface SSEStepCompleteEvent {
  type: 'step_complete';
  data: { step_id: string; status: string; summary?: string };
}

export interface SSEReplanTriggeredEvent {
  type: 'replan_triggered';
  data: { reason: string; completed_steps: number };
}
```

---

## 3. 组件设计

### 3.1 组件结构图

```
AssistantMessage
├── ThinkingProcessPanel (思考过程折叠面板)
│   └── Collapse
│       └── Collapse.Panel
│           └── StageTimeline (阶段时间线)
│               ├── Timeline.Item (RewriteStage)
│               ├── Timeline.Item (PlanStage)
│               │   └── PlanStepsList (步骤列表)
│               ├── Timeline.Item (ExecuteStage)
│               │   └── ToolExecutionTimeline
│               │       └── ToolCard × N
│               └── Timeline.Item (UserActionStage)
│                   └── ApprovalModal (审批弹窗)
└── MarkdownContent (最终回复)
```

### 3.2 组件职责

| 组件 | 职责 | Props |
|------|------|-------|
| `ThinkingProcessPanel` | 折叠面板容器，控制展开/收起 | `stages`, `isStreaming`, `defaultExpanded` |
| `StageTimeline` | 阶段时间线，展示各阶段状态 | `stages`, `activeStageKey` |
| `PlanStepsList` | 步骤列表，展示规划内容 | `steps`, `currentStepIndex` |
| `ToolExecutionTimeline` | 工具执行时间线 | `executions` |
| `ApprovalModal` | 审批确认弹窗 | `visible`, `request`, `onConfirm`, `onCancel` |

---

## 4. 组件实现

### 4.1 ThinkingProcessPanel

```tsx
// web/src/components/AI/components/ThinkingProcessPanel.tsx

import React from 'react';
import { Collapse, Badge } from 'antd';
import { BulbOutlined, CheckCircleOutlined, LoadingOutlined } from '@ant-design/icons';
import { StageTimeline } from './StageTimeline';
import type { ThoughtStageItem } from '../types';

interface ThinkingProcessPanelProps {
  stages: ThoughtStageItem[];
  isStreaming?: boolean;
  defaultExpanded?: boolean;
}

export function ThinkingProcessPanel({
  stages,
  isStreaming,
  defaultExpanded = true,
}: ThinkingProcessPanelProps) {
  const hasStages = stages && stages.length > 0;
  if (!hasStages) return null;

  const allCompleted = stages.every(s => s.status === 'success');
  const anyLoading = stages.some(s => s.status === 'loading');

  return (
    <div className="thinking-process-panel" style={{ marginBottom: 12 }}>
      <Collapse
        defaultActiveKey={defaultExpanded ? ['thinking'] : []}
        ghost
        items={[
          {
            key: 'thinking',
            label: (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <BulbOutlined />
                <span>思考过程</span>
                {anyLoading && <LoadingOutlined spin style={{ marginLeft: 8 }} />}
                {allCompleted && (
                  <Badge status="success" style={{ marginLeft: 8 }} />
                )}
              </div>
            ),
            children: <StageTimeline stages={stages} />,
          },
        ]}
      />
    </div>
  );
}
```

### 4.2 StageTimeline

```tsx
// web/src/components/AI/components/StageTimeline.tsx

import React from 'react';
import { Timeline, theme } from 'antd';
import {
  CheckCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { PlanStepsList } from './PlanStepsList';
import { ToolExecutionTimeline } from './ToolExecutionTimeline';
import type { ThoughtStageItem, ThoughtStageKey } from '../types';

interface StageTimelineProps {
  stages: ThoughtStageItem[];
}

const stageLabels: Record<ThoughtStageKey, string> = {
  rewrite: '意图识别',
  plan: '整理执行步骤',
  execute: '执行步骤',
  user_action: '等待确认',
};

export function StageTimeline({ stages }: StageTimelineProps) {
  const { token } = theme.useToken();

  const timelineItems = stages.map((stage) => ({
    key: stage.key,
    dot: <StageIcon status={stage.status} />,
    children: (
      <StageContent stage={stage} />
    ),
  }));

  return <Timeline items={timelineItems} />;
}

function StageIcon({ status }: { status: string }) {
  const { token } = theme.useToken();

  switch (status) {
    case 'success':
      return <CheckCircleOutlined style={{ color: token.colorSuccess }} />;
    case 'loading':
      return <LoadingOutlined spin style={{ color: token.colorPrimary }} />;
    case 'error':
      return <ExclamationCircleOutlined style={{ color: token.colorError }} />;
    default:
      return <ClockCircleOutlined style={{ color: token.colorTextDisabled }} />;
  }
}

function StageContent({ stage }: { stage: ThoughtStageItem }) {
  return (
    <div style={{ paddingBottom: 8 }}>
      <div style={{ fontWeight: 500, marginBottom: 4 }}>
        {stage.title || stageLabels[stage.key]}
      </div>

      {stage.description && (
        <div style={{ color: '#666', fontSize: 13, marginBottom: 8 }}>
          {stage.description}
        </div>
      )}

      {stage.key === 'plan' && stage.steps && (
        <PlanStepsList
          steps={stage.steps}
          currentStepIndex={stage.currentStepIndex}
        />
      )}

      {stage.key === 'execute' && stage.executions && (
        <ToolExecutionTimeline executions={stage.executions} />
      )}

      {stage.content && (
        <div style={{ fontSize: 13, color: '#666' }}>{stage.content}</div>
      )}
    </div>
  );
}
```

### 4.3 PlanStepsList

```tsx
// web/src/components/AI/components/PlanStepsList.tsx

import React from 'react';
import { theme, Tag } from 'antd';
import {
  CheckCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import type { PlanStep } from '../types';

interface PlanStepsListProps {
  steps: PlanStep[];
  currentStepIndex?: number;
}

export function PlanStepsList({ steps, currentStepIndex }: PlanStepsListProps) {
  const { token } = theme.useToken();

  if (!steps || steps.length === 0) return null;

  return (
    <div style={{ marginTop: 8 }}>
      {steps.map((step, index) => (
        <div
          key={step.id}
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            padding: '8px 12px',
            marginBottom: 4,
            borderRadius: token.borderRadiusSM,
            background: step.status === 'running' ? token.colorPrimaryBg : 'transparent',
            opacity: step.status === 'pending' ? 0.6 : 1,
            transition: 'all 0.2s ease',
          }}
        >
          <StepIcon status={step.status} index={index + 1} />
          <span style={{ flex: 1, marginLeft: 8, fontSize: 13 }}>
            {step.content}
          </span>
          {step.tool_hint && (
            <Tag
              style={{ marginLeft: 8, fontSize: 11 }}
              color={step.status === 'running' ? 'processing' : 'default'}
            >
              {step.tool_hint}
            </Tag>
          )}
        </div>
      ))}
    </div>
  );
}

function StepIcon({ status, index }: { status: PlanStep['status']; index: number }) {
  const { token } = theme.useToken();

  switch (status) {
    case 'completed':
      return <CheckCircleOutlined style={{ color: token.colorSuccess }} />;
    case 'running':
      return <LoadingOutlined spin style={{ color: token.colorPrimary }} />;
    case 'failed':
      return <CloseCircleOutlined style={{ color: token.colorError }} />;
    case 'skipped':
      return <MinusCircleOutlined style={{ color: token.colorTextDisabled }} />;
    default:
      return (
        <span
          style={{
            width: 20,
            height: 20,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderRadius: '50%',
            background: token.colorBgTextHover,
            fontSize: 12,
            color: token.colorTextSecondary,
          }}
        >
          {index}
        </span>
      );
  }
}
```

### 4.4 ToolExecutionTimeline

```tsx
// web/src/components/AI/components/ToolExecutionTimeline.tsx

import React from 'react';
import { theme, Timeline } from 'antd';
import {
  CheckCircleOutlined,
  LoadingOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import type { ToolExecution } from '../types';

interface ToolExecutionTimelineProps {
  executions: ToolExecution[];
}

export function ToolExecutionTimeline({ executions }: ToolExecutionTimelineProps) {
  const { token } = theme.useToken();

  if (!executions || executions.length === 0) return null;

  const items = executions.map((exec) => ({
    key: exec.id,
    dot: <ToolStatusIcon status={exec.status} />,
    children: (
      <div style={{ fontSize: 13 }}>
        <div style={{ fontWeight: 500, marginBottom: 4 }}>
          🔧 {exec.toolName}
        </div>
        {exec.arguments && Object.keys(exec.arguments).length > 0 && (
          <div
            style={{
              padding: '4px 8px',
              background: token.colorBgTextHover,
              borderRadius: token.borderRadiusSM,
              fontSize: 12,
              marginBottom: 4,
            }}
          >
            参数: {JSON.stringify(exec.arguments)}
          </div>
        )}
        {exec.result && (
          <div style={{ color: token.colorTextSecondary, marginTop: 4 }}>
            结果: {truncateResult(exec.result)}
          </div>
        )}
      </div>
    ),
  }));

  return <Timeline items={items} />;
}

function ToolStatusIcon({ status }: { status: string }) {
  const { token } = theme.useToken();

  switch (status) {
    case 'success':
      return <CheckCircleOutlined style={{ color: token.colorSuccess }} />;
    case 'running':
      return <LoadingOutlined spin style={{ color: token.colorPrimary }} />;
    case 'error':
      return <CloseCircleOutlined style={{ color: token.colorError }} />;
    default:
      return null;
  }
}

function truncateResult(result: string, maxLength = 100): string {
  if (result.length <= maxLength) return result;
  return result.slice(0, maxLength) + '...';
}
```

### 4.5 ApprovalModal

```tsx
// web/src/components/AI/components/ApprovalModal.tsx

import React, { useState } from 'react';
import { Modal, Button, Space, Alert, Typography, Collapse, theme, Tag } from 'antd';
import {
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import type { ApprovalRequest } from '../types';

interface ApprovalModalProps {
  visible: boolean;
  request: ApprovalRequest | null;
  onConfirm: () => void;
  onCancel: () => void;
  loading?: boolean;
}

const riskColors: Record<string, string> = {
  low: 'success',
  medium: 'warning',
  high: 'error',
};

const riskLabels: Record<string, string> = {
  low: '低风险',
  medium: '中风险',
  high: '高风险',
};

export function ApprovalModal({
  visible,
  request,
  onConfirm,
  onCancel,
  loading,
}: ApprovalModalProps) {
  const { token } = theme.useToken();

  if (!request) return null;

  return (
    <Modal
      open={visible}
      title={null}
      footer={null}
      closable={false}
      maskClosable={false}
      width={520}
      styles={{ body: { padding: 0 } }}
    >
      {/* Header */}
      <div style={{ padding: '24px 24px 16px' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
          <ExclamationCircleOutlined
            style={{
              fontSize: 24,
              color: request.risk === 'high' ? token.colorError : token.colorWarning,
            }}
          />
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Typography.Title level={5} style={{ margin: 0 }}>
                {request.title || '操作确认'}
              </Typography.Title>
              <Tag color={riskColors[request.risk]}>{riskLabels[request.risk]}</Tag>
            </div>
            <Typography.Text type="secondary">
              {request.description || request.summary}
            </Typography.Text>
          </div>
        </div>
      </div>

      {/* Details */}
      {request.details && (
        <div style={{ padding: '0 24px' }}>
          <Collapse
            ghost
            items={[
              {
                key: 'details',
                label: '查看详情',
                children: (
                  <pre
                    style={{
                      margin: 0,
                      padding: 12,
                      background: token.colorBgTextHover,
                      borderRadius: token.borderRadiusSM,
                      fontSize: 12,
                      maxHeight: 200,
                      overflow: 'auto',
                    }}
                  >
                    {JSON.stringify(request.details, null, 2)}
                  </pre>
                ),
              },
            ]}
          />
        </div>
      )}

      {/* Actions */}
      <div
        style={{
          padding: '16px 24px 24px',
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 8,
        }}
      >
        <Button onClick={onCancel} disabled={loading}>
          取消执行
        </Button>
        <Button type="primary" onClick={onConfirm} loading={loading}>
          确认执行
        </Button>
      </div>
    </Modal>
  );
}
```

---

## 5. 状态管理扩展

### 5.1 useAIChat 扩展

```typescript
// web/src/components/AI/hooks/useAIChat.ts

// === 新增: 事件处理器 ===

const handlePhaseStarted = (
  payload: SSEPhaseStartedEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) => {
  const { phase, title } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    const stageKey = mapPhaseToStage(phase);
    return {
      ...item,
      thoughtChain: upsertThoughtStage(item.thoughtChain, {
        key: stageKey,
        title,
        status: 'loading',
      }),
    };
  }));
};

const handlePlanGenerated = (
  payload: SSEPlanGeneratedEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) => {
  const { steps } = payload;

  // 转换步骤格式
  const planSteps: PlanStep[] = steps.map((s, i) => ({
    id: s.id || `step-${i}`,
    content: s.content,
    tool_hint: s.tool_hint,
    status: 'pending',
  }));

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    return {
      ...item,
      thoughtChain: upsertThoughtStage(item.thoughtChain, {
        key: 'plan',
        status: 'success',
        steps: planSteps,
      }),
    };
  }));
};

const handleStepStarted = (
  payload: SSEStepStartedEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) => {
  const { step_id, title, tool_name, params } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    const thoughtChain = item.thoughtChain || [];
    const planStage = thoughtChain.find(s => s.key === 'plan');
    const executeStage = thoughtChain.find(s => s.key === 'execute');

    // 更新 plan 阶段的步骤状态
    let updatedSteps = planStage?.steps || [];
    const stepIndex = updatedSteps.findIndex(s => s.id === step_id);
    if (stepIndex >= 0) {
      updatedSteps = [...updatedSteps];
      updatedSteps[stepIndex] = { ...updatedSteps[stepIndex], status: 'running' };
    }

    // 添加工具执行记录
    const executions = executeStage?.executions || [];
    const newExecution: ToolExecution = {
      id: `exec-${step_id}`,
      stepId: step_id,
      toolName: tool_name || '',
      arguments: params || {},
      status: 'running',
      startedAt: new Date().toISOString(),
    };

    return {
      ...item,
      thoughtChain: upsertMultipleStages(thoughtChain, [
        {
          key: 'plan',
          steps: updatedSteps,
          currentStepIndex: stepIndex >= 0 ? stepIndex : undefined,
        },
        {
          key: 'execute',
          status: 'loading',
          executions: [...executions, newExecution],
        },
      ]),
    };
  }));
};

const handleToolResult = (
  payload: SSEToolResultEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) => {
  const { step_id, tool_name, result, status } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    const thoughtChain = item.thoughtChain || [];
    const executeStage = thoughtChain.find(s => s.key === 'execute');

    // 更新工具执行状态
    const executions = executeStage?.executions || [];
    const execIndex = executions.findIndex(e => e.stepId === step_id);

    if (execIndex >= 0) {
      const updatedExecutions = [...executions];
      updatedExecutions[execIndex] = {
        ...updatedExecutions[execIndex],
        result,
        status: status === 'success' ? 'success' : 'error',
        completedAt: new Date().toISOString(),
      };

      return {
        ...item,
        thoughtChain: upsertThoughtStage(thoughtChain, {
          key: 'execute',
          executions: updatedExecutions,
        }),
      };
    }

    return item;
  }));
};

const handleStepComplete = (
  payload: SSEStepCompleteEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) => {
  const { step_id, status, summary } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    const thoughtChain = item.thoughtChain || [];
    const planStage = thoughtChain.find(s => s.key === 'plan');

    // 更新步骤状态
    let updatedSteps = planStage?.steps || [];
    const stepIndex = updatedSteps.findIndex(s => s.id === step_id);

    if (stepIndex >= 0) {
      updatedSteps = [...updatedSteps];
      updatedSteps[stepIndex] = {
        ...updatedSteps[stepIndex],
        status: status === 'success' ? 'completed' : 'failed',
        result: { ok: status === 'success', summary },
      };
    }

    return {
      ...item,
      thoughtChain: upsertThoughtStage(thoughtChain, {
        key: 'plan',
        steps: updatedSteps,
      }),
    };
  }));
};

const handleReplanTriggered = (
  payload: SSEReplanTriggeredEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) => {
  const { reason, completed_steps } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    return {
      ...item,
      thoughtChain: upsertThoughtStage(item.thoughtChain, {
        key: 'plan',
        status: 'loading',
        title: '动态调整计划',
        description: reason || '根据执行结果重新规划',
        replanReason: reason,
      }),
    };
  }));
};

// === 辅助函数 ===

function mapPhaseToStage(phase: string): ThoughtStageKey {
  const mapping: Record<string, ThoughtStageKey> = {
    planning: 'plan',
    executing: 'execute',
    replanning: 'plan',
  };
  return mapping[phase] || 'execute';
}

function upsertMultipleStages(
  stages: ThoughtStageItem[],
  updates: Partial<ThoughtStageItem>[]
): ThoughtStageItem[] {
  let result = [...stages];
  for (const update of updates) {
    if (!update.key) continue;
    const index = result.findIndex(s => s.key === update.key);
    if (index >= 0) {
      result[index] = { ...result[index], ...update };
    } else {
      result.push({
        key: update.key,
        title: '',
        status: 'loading',
        ...update,
      } as ThoughtStageItem);
    }
  }
  return result;
}
```

### 5.2 集成到 sendMessage

```typescript
// 在 useAIChat.ts 的 sendMessage 函数中

const handlers: AIChatStreamHandlers = {
  // ... 现有 handlers

  // === 新增 handlers ===
  onPhaseStarted: (payload) => handlePhaseStarted(payload, setMessages, assistantId),
  onPlanGenerated: (payload) => handlePlanGenerated(payload, setMessages, assistantId),
  onStepStarted: (payload) => handleStepStarted(payload, setMessages, assistantId),
  onToolCall: (payload) => handleToolCall(payload, setMessages, assistantId),
  onToolResult: (payload) => handleToolResult(payload, setMessages, assistantId),
  onStepComplete: (payload) => handleStepComplete(payload, setMessages, assistantId),
  onReplanTriggered: (payload) => handleReplanTriggered(payload, setMessages, assistantId),
};
```

---

## 6. 集成到消息渲染

### 6.1 修改 Copilot.tsx

```tsx
// web/src/components/AI/Copilot.tsx

import { ThinkingProcessPanel } from './components/ThinkingProcessPanel';
import { ApprovalModal } from './components/ApprovalModal';

// 在 AssistantMessage 组件中

function AssistantMessage({ message }: { message: ChatMessage }) {
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [approvalRequest, setApprovalRequest] = useState<ApprovalRequest | null>(null);

  // 监听审批请求
  useEffect(() => {
    if (message.pendingConfirmation) {
      setApprovalRequest({
        approval_id: message.pendingConfirmation.approval_id,
        tool_name: message.pendingConfirmation.tool_name,
        risk: message.pendingConfirmation.risk_level || 'medium',
        title: '操作确认',
        description: message.pendingConfirmation.summary,
        summary: message.pendingConfirmation.summary,
        details: message.pendingConfirmation.params,
      });
      setApprovalModalVisible(true);
    }
  }, [message.pendingConfirmation]);

  return (
    <div className="assistant-message">
      {/* 思考过程面板 */}
      {message.thoughtChain && message.thoughtChain.length > 0 && (
        <ThinkingProcessPanel
          stages={message.thoughtChain}
          isStreaming={message.status === 'streaming'}
          defaultExpanded={true}
        />
      )}

      {/* 消息内容 */}
      <MarkdownContent content={message.content} />

      {/* 审批弹窗 */}
      <ApprovalModal
        visible={approvalModalVisible}
        request={approvalRequest}
        onConfirm={() => {
          // 调用确认 API
          setApprovalModalVisible(false);
        }}
        onCancel={() => {
          // 调用取消 API
          setApprovalModalVisible(false);
        }}
      />
    </div>
  );
}
```

---

## 7. 文件变更汇总

| 文件 | 操作 | 主要变更 |
|------|------|---------|
| `web/src/components/AI/types.ts` | 修改 | +6 新类型定义 |
| `web/src/components/AI/hooks/useAIChat.ts` | 修改 | +7 事件处理器 |
| `web/src/components/AI/components/ThinkingProcessPanel.tsx` | 新建 | 思考过程折叠面板 |
| `web/src/components/AI/components/StageTimeline.tsx` | 新建 | 阶段时间线 |
| `web/src/components/AI/components/PlanStepsList.tsx` | 新建 | 步骤列表 |
| `web/src/components/AI/components/ToolExecutionTimeline.tsx` | 新建 | 工具执行时间线 |
| `web/src/components/AI/components/ApprovalModal.tsx` | 新建 | 审批弹窗 |
| `web/src/components/AI/Copilot.tsx` | 修改 | 集成新组件 |
| `web/src/api/modules/ai.ts` | 修改 | 新增 SSE 事件类型 |

---

## 8. 样式建议

```css
/* web/src/components/AI/styles/thinking-process.css */

.thinking-process-panel {
  margin-bottom: 12px;
}

.thinking-process-panel .ant-collapse-header {
  padding: 8px 12px !important;
  background: #f5f5f5;
  border-radius: 6px;
}

.plan-steps-list .plan-step.running {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { background: transparent; }
  50% { background: rgba(24, 144, 255, 0.08); }
}

.tool-execution-timeline {
  position: relative;
  padding-left: 8px;
}

.approval-modal .ant-modal-body {
  padding: 0;
}
```
