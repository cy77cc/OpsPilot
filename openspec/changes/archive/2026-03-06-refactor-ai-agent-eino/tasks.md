# Tasks: AI 助手模块重构

## Phase 1: 核心 Agent 重构

### Task 1.1: 重构 Agent 创建逻辑
- **File**: `internal/ai/agent.go`
- **Priority**: P0
- **Estimate**: 2h

**Changes**:
- [x] 删除 `newADKPlanExecuteAgent` 函数
- [x] 新增 `newPlatformAgent` 函数，使用 `adk.NewChatModelAgent`
- [x] 定义统一的 `platformAgentInstruction` 常量
- [x] 更新相关导入

**Acceptance Criteria**:
- Agent 创建成功，无编译错误
- Instruction 包含核心能力说明和执行规则

---

### Task 1.2: 新增 Runner 封装
- **File**: `internal/ai/runner.go` (新建)
- **Priority**: P0
- **Estimate**: 3h

**Changes**:
- [x] 定义 `PlatformRunner` 结构体
- [x] 实现 `NewPlatformRunner` 构造函数
- [x] 实现 `Query` 方法
- [x] 实现 `Resume` 方法
- [x] 实现 `Close` 方法

**Acceptance Criteria**:
- Runner 可以正常创建和销毁
- Query 返回有效的 Iterator

---

### Task 1.3: 实现 CheckPointStore
- **File**: `internal/ai/checkpoint_store.go` (新建)
- **Priority**: P0
- **Estimate**: 2h

**Changes**:
- [x] 实现 `InMemoryCheckPointStore`
- [x] 实现 `RedisCheckPointStore`（如果 Redis 可用）
- [x] 编写单元测试

**Acceptance Criteria**:
- 存储读写正常
- 测试覆盖率 > 80%

---

## Phase 2: Handler 集成

### Task 2.1: 更新 Chat Handler
- **File**: `internal/service/ai/chat_handler.go`
- **Priority**: P0
- **Estimate**: 3h

**Changes**:
- [x] 替换 `ADKAgent.Stream` 调用为 `runner.Query`
- [x] 更新事件处理逻辑适配新 API
- [x] 保持现有 SSE 事件格式兼容

**Acceptance Criteria**:
- 对话流程正常工作
- SSE 事件正确发送

---

### Task 2.2: 新增审批响应路由
- **File**: `internal/service/ai/routes.go`
- **Priority**: P1
- **Estimate**: 1h

**Changes**:
- [x] 新增 `POST /api/ai/approval/respond` 路由
- [x] 实现 `handleApprovalResponse` 处理函数
- [x] 调用 `runner.Resume` 恢复执行

**Acceptance Criteria**:
- 审批响应可以正确恢复执行

---

### Task 2.3: 更新 ServiceContext
- **File**: `internal/svc/svc.go`
- **Priority**: P0
- **Estimate**: 1h

**Changes**:
- [x] 初始化 `PlatformRunner` 替代 `PlatformAgent`
- [x] 更新 `AI` 字段类型

**Acceptance Criteria**:
- 服务启动正常
- AI 功能可用

---

## Phase 3: 清理与测试

### Task 3.1: 删除废弃代码
- **Priority**: P2
- **Estimate**: 1h

**Changes**:
- [x] 删除 `internal/ai/runtime_agent.go`（功能已合并）
- [x] 清理未使用的导入和函数

---

### Task 3.2: 编写单元测试
- **Priority**: P1
- **Estimate**: 3h

**Changes**:
- [x] `runner_test.go`: Runner 功能测试
- [x] `checkpoint_store_test.go`: 存储测试
- [x] `agent_test.go`: Agent 创建测试

**Acceptance Criteria**:
- 核心逻辑测试覆盖
- 所有测试通过

---

### Task 3.3: 集成测试验证
- **Priority**: P0
- **Estimate**: 2h

**Changes**:
- [ ] 手动测试工具调用（host_list_inventory, host_ssh_exec_readonly）
- [ ] 验证中断恢复流程
- [ ] 验证 SSE 事件流

**Acceptance Criteria**:
- 工具调用正常，不再输出"计划 JSON"
- 审批流程正常工作

---

## Task Dependencies

```
Task 1.1 ──┬──> Task 1.2 ──> Task 2.1
           │
           └──> Task 1.3 ──> Task 2.1

Task 2.1 ──> Task 2.3 ──> Task 3.3
Task 2.2 ──> Task 3.3

Task 3.1, Task 3.2 (并行)
```

## Progress Tracking

| Phase | Task | Status | Owner | Notes |
|-------|------|--------|-------|-------|
| 1 | 1.1 Agent 重构 | done | codex | ChatModelAgent + 统一 instruction |
| 1 | 1.2 Runner 封装 | done | codex | PlatformRunner 已接管 Query/Resume |
| 1 | 1.3 CheckPointStore | done | codex | 内存/Redis 存储与测试已完成 |
| 2 | 2.1 Chat Handler | done | codex | SSE 保持兼容并附带 resume 信息 |
| 2 | 2.2 审批路由 | done | codex | 新增 approval/respond |
| 2 | 2.3 ServiceContext | done | codex | AI 字段切换为 PlatformRunner |
| 3 | 3.1 清理代码 | done | codex | runtime_agent.go 已移除 |
| 3 | 3.2 单元测试 | done | codex | 相关测试已补齐并通过 |
| 3 | 3.3 集成验证 | pending | - | |
