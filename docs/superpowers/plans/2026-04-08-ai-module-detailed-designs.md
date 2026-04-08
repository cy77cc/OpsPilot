# AI Module Detailed Designs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce the detailed AI design documents that inherit from the approved blueprint and lock the implementation boundaries before code changes begin.

**Architecture:** Use the approved blueprint as the parent spec, then author the detailed designs in dependency order from domain model to runtime, tool/policy, event/projection, UI/API, and evaluation. Each document must explicitly inherit the parent boundary so no later spec redefines the AI module shape.

**Tech Stack:** Markdown specs under `docs/superpowers/specs`, existing OpsPilot AI docs, git

---

## File Structure

### Existing files to reference

- [ ] Review parent blueprint: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- [ ] Review earlier AI redesign reference: `docs/superpowers/specs/2026-04-08-ai-module-redesign-design.md`
- [ ] Review approval/runtime reference: `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`
- [ ] Review approval/UI reference: `docs/superpowers/specs/2026-03-28-chat-event-iterator-approval-ui-design.md`
- [ ] Review tool/approval reference: `docs/superpowers/specs/2026-03-28-tool-inline-approval-agentflow-design.md`

### New files to create

- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md`
- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md`
- [ ] Create: `docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`

### Existing files to update

- [ ] Modify: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
Purpose: add forward links to the detailed specs once they exist.

## Task 1: Lock Detailed Design Sequence

**Files:**
- Modify: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Create: `docs/superpowers/plans/2026-04-08-ai-module-detailed-designs.md`

- [ ] **Step 1: Add a checklist of detailed design deliverables to the blueprint**

```md
## 18. Detailed Design Split

This blueprint should be followed by eight detailed designs.

1. `AI Module Blueprint Design`
2. `Core Domain Model Design`
3. `Runtime Kernel Design`
4. `Tool Contract and Policy Design`
5. `Event and Projection Design`
6. `Copilot UI and Interaction Design`
7. `Gateway and API Contract Design`
8. `Evaluation Harness Design`
```

- [ ] **Step 2: Confirm the plan file exists and is readable**

Run: `test -f docs/superpowers/plans/2026-04-08-ai-module-detailed-designs.md && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit the planning checkpoint**

```bash
git add docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md docs/superpowers/plans/2026-04-08-ai-module-detailed-designs.md
git commit -m "Record the AI detailed design sequence"
```

## Task 2: Write Core Domain Model Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-redesign-design.md`

- [ ] **Step 1: Draft the document header and inheritance section**

```md
# AI Core Domain Model Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Scope: object model for Session, Task, Run, Turn, ToolCall, Approval, Event, EvaluationCase
- Goal: define durable object boundaries and ownership before runtime implementation

## 1. Inheritance From Blueprint

This document inherits the product boundary, deletion posture, and module ownership defined in the parent blueprint.
If this document conflicts with the blueprint on object ownership, the blueprint wins.
```

- [ ] **Step 2: Write the object sections with fixed field groups**

```md
## 2. Session
- identity fields
- collaboration summary fields
- linkage to tasks
- non-responsibilities

## 3. Task
- objective fields
- session linkage
- task status and ownership

## 4. Run
- run lifecycle fields
- checkpoint linkage
- route and domain fields

## 5. Turn
- turn sequencing
- phase tracking
- evidence summary

## 6. ToolCall
- contract reference
- input/result references
- policy and approval linkage

## 7. Approval
- pending action snapshot
- decision metadata
- resume target

## 8. Event
- canonical envelope
- payload rules
- causality references

## 9. EvaluationCase
- case identity
- expected route/tool/outcome assertions
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
Expected: no output

- [ ] **Step 4: Commit the domain model spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md
git commit -m "Define the AI core domain model"
```

## Task 3: Write Runtime Kernel Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`

- [ ] **Step 1: Draft the runtime loop and state machine sections**

```md
# AI Runtime Kernel Design

## 1. Inheritance From Blueprint
## 2. Runtime Responsibilities
## 3. Run State Machine
## 4. Turn Loop
## 5. Stop Conditions
## 6. Failure Categories
## 7. Checkpoint and Resume Semantics
## 8. Approval Transition Rules
```

- [ ] **Step 2: Include a concrete state transition table**

```md
| From | Trigger | To | Persist Before Transition |
| --- | --- | --- | --- |
| created | route begins | routing | run record |
| routing | route resolved | planning/executing | route metadata |
| executing | approval required | waiting_approval | approval snapshot + checkpoint |
| waiting_approval | approved | resuming | decision event |
| resuming | checkpoint restored | executing | resume event |
| executing | final answer accepted | completed | final outcome |
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
Expected: no output

- [ ] **Step 4: Commit the runtime kernel spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md
git commit -m "Define the AI runtime kernel"
```

## Task 4: Write Tool Contract and Policy Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `docs/superpowers/specs/2026-03-28-tool-inline-approval-agentflow-design.md`

- [ ] **Step 1: Draft tool contract sections**

```md
# AI Tool Contract and Policy Design

## 1. Inheritance From Blueprint
## 2. Tool Registry Scope
## 3. Tool Contract Schema
## 4. Use-When and Dont-Use-When Rules
## 5. Evidence Contract Rules
## 6. Invocation Pipeline
## 7. Policy Decision Model
## 8. Approval Decision Inputs
## 9. Failure and Deny Semantics
```

- [ ] **Step 2: Include a concrete policy decision table**

```md
| Decision | Meaning | Runtime Behavior |
| --- | --- | --- |
| allow | action is safe enough | invoke tool |
| deny | action is forbidden | emit denial event and continue/replan |
| dry_run_only | side effect not yet allowed | switch invocation mode to preview |
| require_approval | human gate required | persist approval and pause run |
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
Expected: no output

- [ ] **Step 4: Commit the tool and policy spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md
git commit -m "Define AI tool contracts and policy rules"
```

## Task 5: Write Event and Projection Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `docs/superpowers/specs/2026-03-28-chat-event-iterator-approval-ui-design.md`
- Reference: `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`

- [ ] **Step 1: Draft event family and canonical envelope sections**

```md
# AI Event and Projection Design

## 1. Inheritance From Blueprint
## 2. Canonical Event Families
## 3. Canonical Event Envelope
## 4. Event Payload Rules
## 5. Replay Projection Rules
## 6. SSE Mapping Rules
## 7. Trace and Evaluation Projection Rules
```

- [ ] **Step 2: Include a UI block projection table**

```md
| Event Family | UI Block | Notes |
| --- | --- | --- |
| assistant output | message | streaming and final text |
| turn planning | plan | step list or replan summary |
| tool lifecycle | tool_call | arguments, result, status |
| approval lifecycle | approval | preview, risk, decision |
| run failure | error | machine code and user-safe message |
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
Expected: no output

- [ ] **Step 4: Commit the event and projection spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-event-projection-design.md
git commit -m "Define AI events and projections"
```

## Task 6: Write Copilot UI and Interaction Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `web/src/components/AI/CopilotSurface.tsx`
- Reference: `web/src/components/AI/AssistantReply.tsx`
- Reference: `web/src/components/AI/ToolReference.tsx`

- [ ] **Step 1: Draft conversation mode vs operation mode sections**

```md
# AI Copilot UI and Interaction Design

## 1. Inheritance From Blueprint
## 2. Surface Goals
## 3. Conversation Mode
## 4. Operation Mode
## 5. Run Detail and Replay View
## 6. Approval Card Design
## 7. Empty, Running, Failed, and Resuming States
```

- [ ] **Step 2: Include a block inventory for the frontend**

```md
| Block | Purpose | Required Fields |
| --- | --- | --- |
| message | render assistant text | text, status |
| plan | show intended steps | title, steps |
| tool_call | show external action | tool name, args, result, status |
| approval | request human decision | risk, preview, decision actions |
| result | show conclusion | summary, evidence |
| error | show failure | message, code, retryability |
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md`
Expected: no output

- [ ] **Step 4: Commit the Copilot UI spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md
git commit -m "Define the AI Copilot interaction model"
```

## Task 7: Write Gateway and API Contract Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `api/ai/v1/ai.go`
- Reference: `web/src/api/modules/ai.ts`

- [ ] **Step 1: Draft API surface sections**

```md
# AI Gateway and API Contract Design

## 1. Inheritance From Blueprint
## 2. Entry APIs
## 3. Run Inspection APIs
## 4. Approval APIs
## 5. Replay and Content APIs
## 6. Evaluation and Debug APIs
## 7. SSE Event Contract
## 8. Backward-Incompatible Changes
```

- [ ] **Step 2: Include a concrete endpoint inventory**

```md
| Endpoint | Purpose |
| --- | --- |
| POST /ai/chat | create or continue collaboration entry |
| GET /ai/runs/:id | inspect run state |
| GET /ai/runs/:id/projection | fetch replay projection |
| GET /ai/approvals/:id | inspect approval |
| POST /ai/approvals/:id/decision | submit approval decision |
| GET /ai/runs/:id/stream | subscribe to public SSE events |
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md`
Expected: no output

- [ ] **Step 4: Commit the API contract spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md
git commit -m "Define the AI gateway contract"
```

## Task 8: Write Evaluation Harness Design

**Files:**
- Create: `docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`
- Reference: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Reference: `script/ai/check_contract_parity.sh`
- Reference: `script/ai/validate_multi_domain_rollout.sh`

- [ ] **Step 1: Draft the evaluation sections**

```md
# AI Evaluation Harness Design

## 1. Inheritance From Blueprint
## 2. Harness Goals
## 3. EvaluationCase Schema
## 4. Outcome Judges
## 5. Transcript Judges
## 6. Replay-Driven Validation
## 7. Pass@k vs Pass^k Usage
## 8. Regression Suite Strategy
```

- [ ] **Step 2: Include a concrete case template**

```md
## Example Case Template

- case_id: host-diagnosis-basic
- user_input: "检查主机 CPU 持续过高的原因"
- expected_route: operation
- expected_domain: host
- expected_tool_family: host_observe
- approval_expected: false
- expected_outcome_signals:
  - summary present
  - evidence list non-empty
```

- [ ] **Step 3: Run a placeholder scan on the new file**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`
Expected: no output

- [ ] **Step 4: Commit the evaluation harness spec**

```bash
git add docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md
git commit -m "Define the AI evaluation harness"
```

## Task 9: Cross-Spec Linkage and Final Review

**Files:**
- Modify: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md`
- Modify: `docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`

- [ ] **Step 1: Add a cross-link footer to each detailed design**

```md
## Related Specs

- Parent blueprint: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Upstream or sibling specs: [list only the actually relevant docs]
```

- [ ] **Step 2: Run a repo-wide placeholder scan across the AI detailed design set**

Run: `rg -n "TODO|TBD|XXX|placeholder|FIXME" docs/superpowers/specs/2026-04-08-ai-*.md`
Expected: no output

- [ ] **Step 3: Review spec coverage against the blueprint**

Run: `rg -n "^## " docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md docs/superpowers/specs/2026-04-08-ai-*.md`
Expected: section listings show every major blueprint area mapped into at least one detailed design

- [ ] **Step 4: Commit the final linked spec set**

```bash
git add docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md docs/superpowers/specs/2026-04-08-ai-*.md
git commit -m "Link the AI design set into one coherent spec tree"
```

## Self-Review

- Spec coverage:
  - blueprint ownership and object boundaries are covered by Tasks 2 and 3
  - tool, policy, approval, and event truth model are covered by Tasks 4 and 5
  - frontend and API boundaries are covered by Tasks 6 and 7
  - evaluation and regression strategy are covered by Task 8
  - cross-spec inheritance and linkage are covered by Task 9
- Placeholder scan:
  - this plan contains no `TODO`, `TBD`, `XXX`, `placeholder`, or `FIXME` markers outside grep commands
- Type consistency:
  - document names, object names, and module names are kept consistent with the approved blueprint
