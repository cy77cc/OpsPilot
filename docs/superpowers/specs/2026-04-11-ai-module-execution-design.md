# AI Module Execution Design

- Date: 2026-04-11
- References:
  - `docs/superpowers/plans/2026-04-08-ai-module-detailed-designs.md`
  - `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`
- External references:
  - Tw93, `https://tw93.fun/2026-03-21/agent.html`
  - Anthropic, `https://claude.com/blog/building-multi-agent-systems-when-and-how-to-use-them`
  - LangChain, `https://blog.langchain.com/context-engineering-for-agents/`
- Scope: optimize the April 8 AI detailed designs for a practice-first rebuild, then define short execution-plan shapes for each major design area
- Goal: turn the existing AI redesign documents into a smaller, execution-oriented design set that supports delete-first implementation without compatibility work

## 1. Background

The current April 8 AI documents define a strong target architecture, but they still lean more toward design completeness than implementation execution. For this project, that balance is wrong.

This repository is a learning-oriented project. Functional implementation is the first priority. Backward compatibility is not a goal. If the old AI structure conflicts with the new design, the old structure should be deleted instead of wrapped.

The design set should therefore optimize for:

1. Smaller boundaries.
2. Shorter execution plans.
3. Delete-first migration.
4. Single-runtime correctness before multi-agent sophistication.
5. Evidence-based verification.

## 2. Design Constraints

The following constraints apply to all optimized AI designs and all later execution plans.

### 2.1 Product and learning constraints

1. This is a practice-first rebuild.
2. Delivering a working path matters more than preserving legacy structure.
3. Compatibility layers are out of scope unless a later design explicitly proves they are required.
4. When old and new structures conflict, the conflicting old structure should be removed.

### 2.2 Architectural constraints

1. Keep the approved `Copilot + Runtime` product boundary from the April 8 blueprint.
2. Execute implementation in the order of a single stable runtime loop, not in the order of abstract subsystem completeness.
3. Treat multi-agent orchestration as an extension point, not the first implementation target.

### 2.3 Documentation constraints

1. Each major AI design should be short enough to feed a short execution plan.
2. Each execution plan should fit on roughly one page.
3. Each execution plan should identify concrete deletions, retained assets, minimum additions, and hard verification points.

## 3. Optimization Principles

The April 8 design set should be optimized around six principles.

### 3.1 Single runtime first

Start with one runtime loop that can classify, execute, pause for approval, resume, and stop correctly.

Do not let speculative multi-agent orchestration shape the first implementation. This follows the same practical guidance emphasized by Anthropic: start with one agent and split only when specialization or isolation materially improves the result.

### 3.2 Context layering over prompt accumulation

The system must separate:

1. stable instructions
2. current run state
3. current turn evidence
4. on-demand knowledge or skill content
5. tool outputs and approval state

This follows the context-engineering direction from LangChain and the skill-loading advice from Tw93. The runtime should stop building one large prompt blob that mixes instructions, state, memory, and dynamic evidence.

### 3.3 Load on demand

Only indexes and routing hints should stay always available. Detailed skill text, long references, and specialized knowledge should load only when the current turn actually needs them.

This reduces context growth, makes routing clearer, and keeps the runtime explainable.

### 3.4 Delete first

The rebuild should prefer deletion over preservation.

Data assets may survive if they already match the new boundaries. Old orchestration glue, scene-driven control flow, prompt glue, and implicit runtime behavior should not survive by default.

### 3.5 Canonical events are truth

Canonical persisted events should remain the only truth model for replay, SSE, UI read models, and evaluation.

Derived projections are convenience layers. They must never become the durable source of truth.

### 3.6 Evaluation serves refactoring

The first evaluation strategy should protect the rebuild rather than become its own product.

Initial cases should focus on:

1. route correctness
2. tool-family correctness
3. approval correctness
4. durable outcome correctness
5. replay consistency

Transcript quality, broad benchmarking, and framework-heavy evaluation work are explicitly secondary.

## 4. Optimized Design Set

The current detailed design set should be reorganized into four core designs plus one short evaluation strategy.

### 4.1 Core design set

1. `AI Core Runtime Design`
2. `AI Tool Policy Design`
3. `AI Event Projection Design`
4. `AI Boundary Design`

### 4.2 Evaluation companion

5. `AI Evaluation Strategy`

### 4.3 Why this split

The current design set is still too fragmented for a delete-first rebuild. Several documents are logically dependent and should not remain separate:

1. The domain model and runtime kernel are too tightly coupled to stay as independent primary documents.
2. Copilot UI interaction and gateway API contract both define the shell around runtime and should be merged into one boundary design.
3. Evaluation should become a lightweight strategy document rather than another heavy architecture document.

This split reduces conceptual duplication while preserving the decisions that affect implementation.

## 5. Core Design Rewrites

### 5.1 AI Core Runtime Design

This document should merge the current domain-model and runtime-kernel emphasis into one execution-centered design.

The primary question should be: what is the smallest runtime loop that can correctly run, pause, resume, and stop?

The document should:

1. define request entry to run creation
2. define run and turn ownership
3. define the observe-decide-act-record loop
4. define checkpoint and approval continuation
5. define terminal stop conditions

The document should stop spending space on object descriptions that do not affect the runtime loop.

The document should explicitly delete or reject:

1. scene-driven control flow
2. prompt-driven orchestration
3. hidden runtime ownership inside `logic.go`
4. implicit context accumulation

### 5.2 AI Tool Policy Design

This document should stay independent and focus on hard execution boundaries.

The primary question should be: what must be decided in code before a tool is allowed to execute?

The document should:

1. define the tool registry contract
2. define input and output contracts
3. define risk and side-effect metadata
4. define approval triggers
5. define allow, deny, dry-run, and require-approval outcomes
6. define failure categories

The document should explicitly delete or reject:

1. prompt-only tool safety rules
2. hidden tool policy inside model behavior
3. side-effect decisions made only from prose instructions

### 5.3 AI Event Projection Design

This document should become smaller and harder.

The primary question should be: what is the minimum canonical event truth required to reconstruct the important system views?

The document should:

1. define the minimum canonical event families
2. define envelope and causality rules
3. define append-only truth constraints
4. define projection responsibilities for replay, SSE, UI, and evaluation
5. define what cannot be treated as truth

The document should explicitly delete or reject:

1. UI state as durable truth
2. SSE transport shape as canonical truth
3. transcript fragments as the only durable record of tool or approval intent

### 5.4 AI Boundary Design

This document should merge Copilot interaction and gateway API concerns into one shell-boundary design.

The primary question should be: how does the outside world enter and observe runtime without becoming a second control plane?

The document should:

1. define external request entry points
2. define conversation versus operation handoff
3. define approval and replay entry surfaces
4. define SSE or stream exposure as transport only
5. define UI ownership as presentation and interaction only

The document should explicitly delete or reject:

1. API-layer orchestration ownership
2. UI-layer runtime ownership
3. duplicate execution decisions outside runtime and policy

### 5.5 AI Evaluation Strategy

This should become a short strategy document rather than a full heavy design.

The primary question should be: what is the minimum evaluation set that protects delete-first refactoring?

The document should:

1. define the first regression suites
2. define the canonical artifacts consumed by cases
3. define blocking verdict categories
4. define what counts as pass or failure
5. define which evaluation work is intentionally deferred

The strategy should reject:

1. benchmark-style expansion before the runtime is stable
2. transcript-only truth inference
3. framework-first harness work

## 6. Execution Plan Shape

Each core design should produce a short execution plan with the same structure.

### 6.1 Required sections

1. `Goal`
2. `Delete First`
3. `Keep`
4. `Add`
5. `Verify`

### 6.2 Execution-plan rules

1. `Goal` names one minimum implementation result.
2. `Delete First` lists the concrete legacy control flow, entry points, or abstractions that conflict with the design.
3. `Keep` lists only the assets worth retaining, such as durable tables, DAOs, or already-correct event assets.
4. `Add` lists only the minimum new structures needed for the target path.
5. `Verify` lists concrete checks or tests, not vague quality statements.

## 7. Recommended Execution Plan Split

The optimized design set should produce four short execution plans and one short evaluation strategy.

### 7.1 AI Runtime Execution Plan

Goal:

- replace mixed orchestration with one explicit runtime loop

Delete first:

- old orchestration glue
- scene-driven control flow
- mixed ownership in `internal/modules/ai/logic/logic.go` and related legacy control paths

Keep:

- durable persistence models that still fit the new run boundary
- reusable DAO or storage helpers
- approval or replay assets that already align with checkpoint-backed execution

Add:

- single run and turn loop
- explicit checkpoint and resume path
- explicit stop and failure handling

Verify:

- request can enter runtime
- run can complete
- run can pause for approval
- run can resume from approval
- run can fail with stable failure metadata

### 7.2 AI Tool and Policy Execution Plan

Goal:

- move tool execution decisions out of prompt behavior and into explicit contracts and policy

Delete first:

- prompt-only tool safety rules
- hidden side-effect assumptions
- implicit approval gating

Keep:

- stable tool adapters that already expose clear contracts
- existing approval artifacts that can be mapped into policy outcomes

Add:

- minimum tool registry
- policy decision path
- approval trigger mapping
- allow and deny semantics

Verify:

- safe call runs directly
- blocked call is denied
- approval-required call pauses correctly
- failed tool call maps to a stable failure category

### 7.3 AI Event and Projection Execution Plan

Goal:

- rebuild system truth on canonical append-only events and derive the rest from them

Delete first:

- projections treated as truth
- UI or stream payloads stored as canonical state
- transcript-only durable intent

Keep:

- event storage assets that already match append-only semantics
- reusable replay or SSE builders that can be strictly converted into projection consumers

Add:

- minimum canonical event families
- projection builders for replay, SSE, and UI
- ordering and causality enforcement

Verify:

- one run can be reconstructed from canonical events
- replay matches the event stream
- SSE payloads are derived from the same canonical path
- evaluation can read canonical artifacts without transcript inference

### 7.4 AI Boundary Execution Plan

Goal:

- reduce Copilot UI and gateway behavior to a clean shell around runtime

Delete first:

- duplicate orchestration in API handlers
- duplicate orchestration in UI state flows
- boundary code that makes execution decisions outside runtime or policy

Keep:

- protocol translation code
- stable request and response shells
- UI components that only render projections or collect user input

Add:

- single request handoff shape into runtime
- clear conversation versus operation entry rules
- approval and replay boundary paths

Verify:

- chat-style request enters through the boundary and reaches the right owner
- operation-style request reaches runtime without duplicate orchestration
- approval response reaches the resume path only through boundary entry points
- replay request only reads projections or canonical artifacts

## 8. Deferred Work

The following work is explicitly deferred until the first delete-first rebuild path is stable:

1. first-class multi-agent orchestration
2. broad long-term memory strategy
3. benchmark-heavy evaluation expansion
4. broad compatibility shims
5. optimization for generalized framework reuse

## 9. Acceptance Criteria

This design is successful when:

1. the AI redesign documentation can be reduced to a smaller set of boundary-focused documents
2. each resulting document can generate a short execution plan
3. the execution plans are delete-first and implementation-oriented
4. multi-agent work is kept out of the first runtime milestone
5. evaluation is framed as rebuild protection rather than framework growth

## 10. Recommended Next Step

The next workflow artifact should be a set of short implementation plans derived from this document:

1. `AI Runtime Execution Plan`
2. `AI Tool and Policy Execution Plan`
3. `AI Event and Projection Execution Plan`
4. `AI Boundary Execution Plan`
5. `AI Evaluation Strategy`

Those plans should stay concise and should optimize for implementation order, deletion order, and verification evidence.
