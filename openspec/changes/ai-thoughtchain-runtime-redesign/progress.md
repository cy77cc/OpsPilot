# ai-thoughtchain-runtime-redesign Progress

**Last Updated:** 2026-03-14

## Summary

This change is **~40-45% complete**. The core ThoughtChain event system and orchestrator integration are done, but legacy removal, unified approval API, frontend UI upgrade, and testing remain.

## Completed Work

### Backend Core (~70%)

| Component | Status | Evidence |
|-----------|--------|----------|
| ThoughtChain event names | ✅ | `internal/ai/events/events.go` - `chain_started`, `chain_node_open`, `chain_node_patch`, `chain_node_close`, `chain_collapsed`, `final_answer_*` |
| SSE Converter | ✅ | `internal/ai/runtime/sse_converter.go` - full event conversion |
| Orchestrator chain lifecycle | ✅ | `internal/ai/orchestrator.go:355-424` - `emitChainStarted`, `openChainNode`, `closeActiveNode`, `startFinalAnswer` |
| Approval as first-class node | ✅ | `internal/ai/orchestrator.go:604-622` - approval node with `waiting` status |
| ChainNodeInfo type | ✅ | `internal/ai/runtime/runtime.go:58-69` - `ChainNodeKind`, `ChainNodeInfo` |
| PhaseDetector | ✅ | `internal/ai/runtime/phase_detector.go` - detects planning/executing/replanning |

### Observability (~90%)

| Component | Status | Evidence |
|-----------|--------|----------|
| ThoughtChain metrics | ✅ | `internal/ai/observability/metrics.go:30-47` - `ThoughtChainRecord`, `ThoughtChainNodeRecord`, `ThoughtChainApprovalRecord` |
| Prometheus counters | ✅ | `opspilot_ai_thoughtchain_runs_total`, `opspilot_ai_thoughtchain_nodes_total`, `opspilot_ai_thoughtchain_approvals_total` |
| Duration histograms | ✅ | `opspilot_ai_thoughtchain_duration_seconds`, `opspilot_ai_thoughtchain_node_duration_seconds`, `opspilot_ai_thoughtchain_approval_wait_seconds` |
| Observance calls | ✅ | `internal/ai/orchestrator.go:370-376`, `624-635`, `216-220` - integrated into lifecycle |

### Frontend Core (~50%)

| Component | Status | Evidence |
|-----------|--------|----------|
| Runtime state type | ✅ | `web/src/components/AI/types.ts` - `ThoughtChainRuntimeState`, `RuntimeThoughtChainNode` |
| Event reducer | ✅ | `web/src/components/AI/thoughtChainRuntime.ts:37-101` - `reduceThoughtChainRuntimeEvent` |
| Replay restoration | ✅ | `web/src/components/AI/thoughtChainRuntime.ts:221-295` - `runtimeStateFromReplayTurn` |
| Node kind normalization | ✅ | `web/src/components/AI/thoughtChainRuntime.ts:186-219` - `normalizeNodeKind`, `normalizeNodeStatus` |

---

## Incomplete Work

### 1. Legacy Runtime Removal (0%)

**Blocked by: Nothing - should be done first**

Legacy events still emitted in `internal/ai/events/events.go`:
- `Meta`, `Delta`, `ThinkingDelta`, `ToolCall`, `ToolResult`, `TurnState`
- These should NOT be the "primary path" per design

Legacy types still in `internal/ai/runtime/runtime.go`:
- `PhaseEvent`, `PlanEvent`, `StepEvent`, `ReplanEvent` (lines 93-155)
- These duplicate ThoughtChain concepts

Frontend legacy code not removed:
- `messageBlocks.ts`, `turnLifecycle.ts` - old message/block logic
- Fallback JSON dumping paths in components

### 2. ThoughtChain Runtime Contract (~60%)

Missing events per spec:
- `chain_paused` - for approval wait state
- `chain_resumed` - for approval continuation
- `chain_error` - defined but not emitted

Session replay contract:
- No formal spec for how sessions store chain nodes
- `runtimeStateFromReplayTurn` reconstructs from legacy `turn.blocks`

### 3. Unified Approval Flow (~40%)

**Current state:** Uses old `checkpoint_id` + `step_id` recovery

**Required new API:**
```
POST /api/v1/ai/chains/{chain_id}/approvals/{node_id}/decision
Body: { "approved": bool, "reason": string }
```

**Evidence needed:**
- Route registration in `internal/service/ai/routes.go`
- Handler in `internal/service/ai/handler/`
- Frontend call site using `chain_id` + `node_id`

### 4. Frontend ThoughtChain Experience (~30%)

**Missing node-specific renderers:**

| Node Kind | Required UI | Status |
|-----------|-------------|--------|
| `plan` | Step list with status | ❌ Missing |
| `step` | Progress indicator | ❌ Missing |
| `tool` | Tool card with result | ⚠️ Partial |
| `approval` | Confirmation panel | ⚠️ Partial |
| `replan` | Reason summary | ❌ Missing |
| `answer` | Final text stream | ✅ Done |

**Race condition bug:**
- New conversation with recommended prompt can show "unavailable" state
- Need to create assistant chain container immediately
- See: `web/src/components/AI/components/RuntimeThoughtChain.tsx`

### 5. Observability Gaps (~10% missing)

**Missing trace/span propagation:**
- No `trace_id` threading through execution
- No `chain_id` → `node_id` correlation in traces
- Prometheus metrics exist but lack tracing context

### 6. Validation And Testing (0%)

**Required tests:**

| Category | Needed | Status |
|----------|--------|--------|
| Backend lifecycle | Chain event ordering, approval pause/resume, replan | ❌ |
| Legacy non-emission | Verify old events NOT on primary path | ❌ |
| Frontend rendering | Node types, approval interaction, replay/live consistency | ❌ |
| Race handling | Recommended prompt on new conversation | ❌ |

---

## File Reference

### Key Files (Implemented)

```
internal/ai/events/events.go           # Event name constants
internal/ai/runtime/runtime.go         # Core types, ExecutionStore, CheckpointStore
internal/ai/runtime/sse_converter.go   # SSE event conversion
internal/ai/runtime/phase_detector.go  # Planning/executing/replanning detection
internal/ai/runtime/plan_parser.go     # Plan extraction from LLM output
internal/ai/orchestrator.go            # Main execution loop
internal/ai/observability/metrics.go   # Prometheus metrics
web/src/components/AI/types.ts         # Frontend types
web/src/components/AI/thoughtChainRuntime.ts  # State management
```

### Files To Create/Modify

```
# New unified approval API
internal/service/ai/handler/chain_approval.go
internal/service/ai/routes.go (add chain routes)

# Remove legacy
internal/ai/events/events.go (delete old events)
internal/ai/runtime/runtime.go (delete PhaseEvent, PlanEvent, etc.)
web/src/components/AI/messageBlocks.ts (remove or simplify)
web/src/components/AI/turnLifecycle.ts (remove)

# Frontend UI upgrade
web/src/components/AI/components/ChainNodePlan.tsx
web/src/components/AI/components/ChainNodeTool.tsx
web/src/components/AI/components/ChainNodeApproval.tsx
web/src/components/AI/components/ChainNodeReplan.tsx

# Tests
internal/ai/runtime/runtime_test.go
internal/ai/orchestrator_test.go
web/src/components/AI/thoughtChainRuntime.test.ts
```

---

## Recommended Execution Order

1. **Phase 1: Legacy Removal** (highest priority)
   - Delete old event names from `events.go`
   - Delete `PhaseEvent`, `PlanEvent`, `StepEvent`, `ReplanEvent` from `runtime.go`
   - Remove frontend legacy code paths
   - This prevents "dual truth" problems

2. **Phase 2: Unified Approval API**
   - Implement `POST /api/v1/ai/chains/{chain_id}/approvals/{node_id}/decision`
   - Update frontend to use new endpoint
   - Remove old checkpoint-based recovery

3. **Phase 3: Frontend UI Upgrade**
   - Implement node-specific renderers
   - Fix recommended prompt race condition

4. **Phase 4: Testing**
   - Add backend tests
   - Add frontend tests
   - Run `openspec validate`

5. **Phase 5: Tracing** (optional)
   - Add trace/span propagation if needed

---

## Design Decisions Reference

From `design.md`:

1. **ThoughtChain is the only primary runtime protocol** - partially implemented
2. **Delete old primary-path wiring before attaching new** - NOT done
3. **Approval is a first-class chain node** - done
4. **One chain store for live and replay** - done
5. **Observability is callback-first** - done
