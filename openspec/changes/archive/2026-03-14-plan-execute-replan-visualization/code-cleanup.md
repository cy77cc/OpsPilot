# 代码清理建议

## 概述

在实现 Plan-Execute-Replan 思考过程可视化功能时，应同步清理冗余和低质量的代码。本文档列出建议删除或优化的代码。

---

## 一、建议删除的代码

### 1.1 前端组件

| 文件 | 原因 | 替代方案 |
|------|------|---------|
| `web/src/components/AI/components/ApprovalConfirmationPanel.tsx` | 空封装，只转发 props | 直接使用 `ConfirmationPanel` 或新的 `ApprovalModal` |
| `web/src/components/AI/components/ThoughtChainDetailItem.tsx` | 实现简单，功能被新组件覆盖 | 新的 `PlanStepsList` 组件 |
| `web/src/components/AI/components/ThoughtChainStageCard.tsx` | 实现简单，功能被新组件覆盖 | 新的 `StageTimeline` 组件 |

**删除方式**: 先实现新组件，验证功能后删除旧组件

### 1.2 后端未使用的事件常量

| 事件 | 文件 | 原因 |
|------|------|------|
| `PlannerState` | `events/events.go` | 从未被调用 |
| `ClarifyRequired` | `events/events.go` | 从未被调用 |

**处理方式**: 暂时保留（可能未来使用），但添加注释说明当前未实现

---

## 二、建议优化的代码

### 2.1 后端优化

#### 2.1.1 `orchestrator.go` - streamExecution 方法拆分

**问题**: 方法长度 ~100 行，职责过多

**优化方案**: 拆分为多个小方法

```go
// 当前
func (o *Orchestrator) streamExecution(...) {
    // 1. 初始化
    // 2. 事件循环
    // 3. 文本处理
    // 4. 审批处理
    // 5. 完成处理
}

// 优化后
func (o *Orchestrator) streamExecution(...) {
    detector := runtime.NewPhaseDetector()

    for {
        event, ok := iter.Next()
        if !ok { break }

        o.processEvent(ctx, event, state, emit, detector)
    }

    return o.finalizeExecution(ctx, state, emit)
}

func (o *Orchestrator) processEvent(...) {
    // 处理单个事件
    o.handlePhaseChange(event, emit, detector)
    o.handleTextContent(event, emit)
    o.handleToolCalls(event, emit)
    o.handleInterrupt(event, state, emit)
}
```

#### 2.1.2 `sse_converter.go` - OnPlanCreated 参数类型优化

**问题**: `steps []string` 是字符串数组，前端无法展示结构化信息

**优化方案**: 使用结构化类型

```go
// 当前
func (c *SSEConverter) OnPlanCreated(planID, content string, steps []string) StreamEvent

// 优化后
type PlanStep struct {
    ID       string `json:"id"`
    Content  string `json:"content"`
    ToolHint string `json:"tool_hint,omitempty"`
}

func (c *SSEConverter) OnPlanCreated(planID string, steps []PlanStep) StreamEvent
```

#### 2.1.3 `orchestrator.go` - 阶段检测逻辑内联

**问题**: 阶段检测逻辑硬编码在事件处理中

**优化方案**: 抽取为独立的 `PhaseDetector`

```go
// 当前 (隐式)
// 没有显式的阶段检测

// 优化后 (新增文件)
// internal/ai/runtime/phase_detector.go
type PhaseDetector struct {
    currentPhase string
}

func (d *PhaseDetector) Detect(event *adk.AgentEvent) string {
    // 从 RunPath 和 AgentName 推断阶段
}
```

### 2.2 前端优化

#### 2.2.1 `types.ts` - 类型定义整理

**问题**: `AskRequest` 和 `ConfirmationRequest` 功能重叠

**优化方案**: 合并类型

```typescript
// 当前
export interface AskRequest {
  id: string;
  kind?: 'approval' | 'confirmation' | 'review' | 'interrupt';
  title: string;
  // ...
}

export interface ConfirmationRequest {
  id: string;
  title: string;
  // ...
}

// 优化后: 使用一个统一的审批请求类型
export interface ApprovalRequest {
  id: string;
  kind: 'approval' | 'confirmation' | 'review';
  title: string;
  description: string;
  risk: RiskLevel;
  status: 'pending' | 'approved' | 'rejected' | 'submitting';
  details?: Record<string, unknown>;
  params?: Record<string, unknown>;
}
```

#### 2.2.2 `ConfirmationPanel.tsx` - 组件增强

**问题**: 当前组件功能简单，不支持 Modal 形式

**优化方案**: 扩展为支持两种展示模式

```typescript
interface ConfirmationPanelProps {
  confirmation: ConfirmationRequest;
  mode?: 'inline' | 'modal';  // 新增: 展示模式
  visible?: boolean;           // modal 模式需要
  onConfirm: () => void;
  onCancel: () => void;
}

// inline 模式: 当前实现
// modal 模式: 新增弹窗实现
```

#### 2.2.3 `thoughtChainMetrics.ts` - 简化或删除

**问题**: 计算逻辑复杂，实际使用场景有限

**建议**:
1. 如果监控不需要这些指标，删除整个文件
2. 如果需要，保留但添加单元测试

---

## 三、重构顺序建议

### 阶段 1: 后端事件增强 (不删除旧代码)

1. 新建 `phase_detector.go`
2. 新建 `plan_parser.go`
3. 增强 `sse_converter.go` (新增方法，保留旧方法)
4. 修改 `orchestrator.go` (使用新组件)

### 阶段 2: 前端组件实现 (并行开发)

1. 新建 `ThinkingProcessPanel.tsx`
2. 新建 `StageTimeline.tsx`
3. 新建 `PlanStepsList.tsx`
4. 新建 `ToolExecutionTimeline.tsx`
5. 新建 `ApprovalModal.tsx`

### 阶段 3: 集成测试

1. 验证新组件功能
2. 验证 SSE 事件流

### 阶段 4: 清理旧代码 (确认功能正常后)

1. 删除 `ApprovalConfirmationPanel.tsx`
2. 删除 `ThoughtChainDetailItem.tsx`
3. 删除 `ThoughtChainStageCard.tsx`
4. 清理 `types.ts` 中重复的类型定义
5. (可选) 删除 `thoughtChainMetrics.ts`

---

## 四、风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 删除组件后其他地方引用 | 运行时报错 | 使用 IDE 全局搜索确认无引用 |
| 类型合并导致兼容性问题 | TypeScript 编译错误 | 分步重构，保持向后兼容 |
| 后端事件格式变化 | 前端解析失败 | 保持事件向后兼容，新事件使用新字段 |

---

## 五、代码行数估算

| 操作 | 文件数 | 预估行数 |
|------|--------|---------|
| 删除 | 3 | ~50 行 |
| 优化 | 4 | ~200 行修改 |
| 新增 | 7 | ~600 行 |

**净效果**: 代码质量提升，总行数增加约 350 行（新增功能），删除约 50 行冗余代码。
