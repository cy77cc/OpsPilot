# AI Module Blueprint Design

- Date: 2026-04-08
- Scope: OpsPilot AI module end-state blueprint across backend runtime, API contract, frontend Copilot surfaces, storage model, replay, approval, and evaluation harness
- Goal: Define the parent design for rebuilding the AI module into a dual-core `Copilot + Runtime` system, then use it as the boundary source for all later detailed designs
- Status: Parent blueprint for follow-up detailed design documents

## 1. Background

OpsPilot already has a non-trivial AI foundation:

- session, message, and run persistence
- SSE streaming and run projection
- approval interruption and resume
- domain-oriented agents and tool integrations

That base is useful, but the current implementation shape is still too entangled for reliable evolution:

- `internal/service/ai/logic/logic.go` mixes service orchestration, run lifecycle, context assembly, iterator consumption, SSE emission, and persistence coordination
- scene-driven prompt augmentation still carries part of the control flow
- tools, policy, approval, replay, and evaluation are not isolated into stable module boundaries
- frontend AI surfaces already consume projection-like data, but the runtime truth model is not yet cleanly event-first

This project is a practice-first implementation project. Functional correctness and a clean end-state architecture matter more than backward compatibility. Existing code may be deleted aggressively if it blocks the target shape.

## 2. Product Positioning

OpsPilot AI should be rebuilt as a dual-core system:

- `Copilot`: the collaboration-facing layer for conversation, explanation, planning, and guidance
- `Runtime`: the execution-facing layer for routing, tool use, approval, resume, replay, and evaluation

The user should experience one unified AI entry point, but internally the two concerns must stay separate:

- Copilot owns the collaboration surface
- Runtime owns verifiable execution

The module is not positioned as a pure chatbot and not as a pure workflow engine. It is a hybrid operator assistant with strict separation between conversational interaction and operational execution.

## 3. Blueprint Role

This document is the parent blueprint for the AI module. Its job is to lock the target shape before detailed design begins.

This document defines:

- product positioning
- architectural direction
- core objects
- runtime state model
- context layering
- tool and policy boundaries
- event and replay principles
- module decomposition
- deletion and reuse rules
- the detailed design split that follows

This document does not define:

- final table schemas
- exact API field-by-field contracts
- exact frontend component trees
- exact migration scripts
- exact package or file names for every implementation detail

Those belong to later detailed design documents.

### 3.1 Relationship To Existing AI Design Docs

OpsPilot already contains earlier AI redesign and runtime-focused design documents. This blueprint does not replace their detailed observations, but it becomes the parent boundary for all future AI design work.

Rule:

- if an older AI design document conflicts with this blueprint on module boundaries, object ownership, or migration posture, this blueprint wins
- older documents may still be reused as input material for later detailed designs when they stay inside this blueprint's boundaries

## 4. Design Principles

1. `Session` and `Run` are different objects.
`Session` represents long-lived collaboration. `Run` represents one verifiable execution loop.

2. Runtime control flow must not live inside prompt assembly.
Routing, approval, resume, replay, and policy decisions are system behaviors, not prompt conventions.

3. The execution loop must stay small.
The runtime loop should only observe current state, make one decision, execute one step, record the result, and continue or stop.

4. Canonical truth comes from persisted objects and events.
UI, replay, debugging, and evaluation read from stored state and event streams, not from transient goroutine state.

5. Context must be layered.
Stable instructions, runtime state, on-demand knowledge, and memory cannot be mixed into one growing prompt blob.

6. Deterministic constraints belong outside the model.
Tool allowlists, validation, side-effect controls, approval requirements, and hard safety rules should be enforced by code and policy.

7. Tooling must be contract-first.
Tools are not just prompt annotations. They are structured capabilities with schemas, risk metadata, and evidence contracts.

8. Harness is part of the product.
Replay, projection, route inspection, tool selection inspection, and evaluation are not optional support tooling.

9. Single-runtime quality comes before multi-agent autonomy.
V1 should stabilize one runtime loop with strong tool, approval, replay, and evaluation mechanics. Multi-agent orchestration is an extension point, not the first system shape.

10. Data assets may be reused; control-flow glue should not be preserved by default.
Existing DAO and durable models may survive. Existing orchestration glue should be deleted if it fights the target architecture.

## 5. End-State Architecture

The target end-state is organized into eight module groups.

### 5.1 `gateway`

Responsibilities:

- expose HTTP, SSE, and future WebSocket APIs
- accept chat, run, replay, approval, and evaluation-related requests
- translate external protocol requests into internal use cases

Non-responsibilities:

- no runtime loop ownership
- no tool execution logic
- no policy decisions

### 5.2 `copilot`

Responsibilities:

- manage collaboration-centric `Session` lifecycle
- manage user and assistant message presentation
- maintain session summaries and lightweight conversation memory
- support lightweight question-and-answer paths
- hand execution-shaped requests to router/runtime

Non-responsibilities:

- no execution state machine
- no approval lifecycle ownership
- no direct tool pipeline management

### 5.3 `router`

Responsibilities:

- classify incoming intent
- decide `conversation` vs `operation`
- determine domain
- decide whether a request should create a `Task` and `Run`
- choose between single-runtime execution and future expansion modes

Non-responsibilities:

- no direct tool invocation
- no replay ownership
- no event storage ownership

### 5.4 `runtime`

Responsibilities:

- own `Run` state machine
- own `Turn` loop
- assemble execution context
- invoke model decisions
- drive tool and approval transitions
- persist checkpoints and resume state
- emit canonical runtime events
- finalize answers and outcomes

This is the core of the new AI module.

### 5.5 `tools`

Responsibilities:

- hold tool registry and structured contracts
- normalize invocation and result handling
- expose domain tool packs
- define tool-side metadata needed by policy, replay, and evaluation

### 5.6 `policy`

Responsibilities:

- risk evaluation
- allow/deny/dry-run/require-approval decisions
- hard safety guardrails
- argument validation and side-effect classification

### 5.7 `projection`

Responsibilities:

- convert canonical events into UI read models
- convert canonical events into SSE public events
- build replay blocks and inspection views
- provide trace-friendly and evaluation-friendly projections

### 5.8 `eval`

Responsibilities:

- define evaluation cases
- run replay-driven checks
- validate routing, tool choice, approval, and outcome correctness
- support regression harnesses for runtime changes

## 6. Core Domain Objects

The blueprint locks eight first-class objects.

### 6.1 `Session`

Purpose:

- long-lived collaboration container
- stores topic, title, user preference, summary, and linked work items

Question it answers:

- what is the user trying to do over time?

### 6.2 `Task`

Purpose:

- semantic business goal derived from a session
- anchor for a unit of work that may produce one or more runs

Question it answers:

- what objective is the system trying to achieve?

### 6.3 `Run`

Purpose:

- one concrete execution instance
- top-level runtime work unit with durable lifecycle and checkpoints

Question it answers:

- what happened during this specific execution attempt?

### 6.4 `Turn`

Purpose:

- one internal runtime decision/execution round inside a run
- boundary for planning, tool selection, tool execution, and answer production

Question it answers:

- what did the runtime decide and do in this round?

### 6.5 `ToolCall`

Purpose:

- one structured tool invocation record
- stores input, result, risk classification, timing, and completion status

Question it answers:

- what external capability was invoked and what happened?

### 6.6 `Approval`

Purpose:

- one explicit human decision point
- stores pending operation details, risk explanation, resume target, and final decision

Question it answers:

- what action needed human approval and what was the decision?

### 6.7 `Event`

Purpose:

- canonical fact record for runtime behavior
- backing source for replay, UI projection, debugging, and evaluation

Question it answers:

- what did the system observe, decide, execute, and conclude?

### 6.8 `EvaluationCase`

Purpose:

- minimal verifiable test case for route, tool, approval, and outcome quality

Question it answers:

- did the runtime behave correctly for this scenario?

## 7. Object Relationships

The primary relationships are:

- `Session -> Task`
- `Task -> Run`
- `Run -> Turn`
- `Turn -> ToolCall`
- `Run -> Approval`
- `Run -> Event`
- `Task/Run -> EvaluationCase`

Design intent:

- `Session` owns collaboration continuity
- `Task` owns business objective continuity
- `Run` owns execution lifecycle
- `Event` owns factual replayability
- `Approval` owns interrupt-and-resume workflow

This separation prevents the system from collapsing back into a message-centric implementation where execution state is inferred from assistant text.

## 8. Runtime State Model

### 8.1 Run States

Top-level `Run` status should stay intentionally small:

- `created`
- `routing`
- `planning`
- `executing`
- `waiting_approval`
- `resuming`
- `completed`
- `failed`
- `cancelled`

Rules:

- top-level states should only represent durable, externally meaningful lifecycle changes
- UI-specific sub-states and debugging detail should not inflate this enum
- richer detail should live in `Turn.phase` and `Event.type`

### 8.2 Turn Phases

`Turn` carries lighter-weight internal execution progress:

- `context_ready`
- `model_decided`
- `tool_selected`
- `tool_running`
- `tool_finished`
- `awaiting_human`
- `answering`
- `turn_done`

This lets the runtime express internal progress without polluting the stable `Run` lifecycle.

### 8.3 Approval Semantics

`waiting_approval` is not an in-memory pause. It is a durable workflow boundary.

Before entering `waiting_approval`, the system must have persisted:

- current run status
- current turn identity and phase
- pending tool call details
- approval reason and risk level
- operation preview
- checkpoint and resume target

Resuming means reconstructing runtime state from persisted objects and re-entering the loop. It does not mean continuing a blocked goroutine.

## 9. Runtime Loop

The runtime loop should stay minimal:

1. load current run state
2. assemble current execution context
3. ask the model for one decision
4. interpret model output into structured runtime actions
5. pass tool actions through tool and policy pipelines
6. record objects, events, and state changes
7. continue or stop

Conceptually:

- observe
- decide
- act
- record
- continue or stop

The loop must not directly own:

- HTTP request handling
- SSE formatting
- frontend block shaping
- low-level DAO choreography unrelated to runtime state
- domain-specific presentation logic

Those belong to other layers.

## 10. Context Layering

The blueprint fixes five context layers.

### 10.1 `Resident Context`

Stable, always-on instructions:

- module identity
- non-negotiable behavior rules
- minimal output expectations

Properties:

- short
- stable
- cache-friendly

### 10.2 `Runtime Context`

Current run and turn state required for the next decision:

- task summary
- run summary
- current domain
- recent evidence summary
- pending step or open question

Properties:

- dynamic
- strictly scoped to current execution needs

### 10.3 `On-Demand Knowledge`

Domain knowledge loaded only when needed:

- host operations guidance
- kubernetes domain knowledge
- governance constraints
- service-specific procedures

Properties:

- described by lightweight descriptors
- loaded lazily when routing or execution needs it

### 10.4 `Memory`

Reusable experience that should not be pushed into every turn:

- session summaries
- task summaries
- reusable knowledge notes

Properties:

- retrieved when relevant
- not blindly injected into every prompt

### 10.5 `System Constraints`

Deterministic rules enforced outside prompt text:

- tool allow and deny rules
- parameter validation
- approval requirements
- timeout and retry rules
- hard safety constraints

Properties:

- code-enforced
- machine-checkable
- not dependent on model obedience

## 11. Tool Contract Model

Every tool should be defined as a structured contract rather than an informal prompt affordance.

Each contract should minimally describe:

- `tool_id`
- `name`
- `domain`
- `description`
- `use_when`
- `dont_use_when`
- `input_schema`
- `output_schema`
- `risk_level`
- `requires_approval`
- `side_effect_level`
- `idempotency`
- `timeout`
- `evidence_contract`

Key design intent:

- route tool usage by clear purpose and anti-purpose
- expose risk and side effects to policy
- define what good evidence looks like after execution

The blueprint treats tool definitions as one of the main quality levers of the AI module. Tool misuse should be debugged through contracts and policy before blaming model quality.

## 12. Policy Model

The policy layer should produce one of four decisions for a proposed tool action:

- `allow`
- `deny`
- `dry_run_only`
- `require_approval`

Policy evaluation should consider more than the tool name. Inputs should include:

- current run type
- domain
- tool arguments
- tool risk level
- side-effect level
- user identity or role when relevant
- current runtime mode
- safer alternative path availability

Policy is the deterministic safety boundary of the system. It should be inspectable, replayable, and testable without involving the model.

## 13. Event-First Truth Model

### 13.1 Canonical Event Families

The runtime should emit canonical events across six families.

Run lifecycle:

- `run_created`
- `run_routed`
- `run_started`
- `run_completed`
- `run_failed`
- `run_cancelled`

Turn lifecycle:

- `turn_started`
- `turn_planned`
- `turn_refined`
- `turn_completed`

Tool lifecycle:

- `tool_selected`
- `tool_started`
- `tool_succeeded`
- `tool_failed`
- `tool_skipped`

Approval lifecycle:

- `approval_requested`
- `approval_granted`
- `approval_rejected`
- `approval_expired`
- `run_resuming`

Assistant output:

- `assistant_delta`
- `assistant_message_final`
- `assistant_summary_updated`

Evaluation and safety:

- `route_evaluated`
- `tool_choice_evaluated`
- `outcome_evaluated`
- `safety_violation_detected`

### 13.2 Canonical Event Shape

Each event should carry enough structure for replay and diagnosis:

- `event_id`
- `run_id`
- `session_id`
- `turn_id`
- `event_type`
- `event_time`
- `actor_type`
- `status`
- `payload`
- `trace_id`
- `caused_by_event_id` when meaningful

### 13.3 Event Mapping Rule

Canonical events are the source of truth.

Projection layers may map them into:

- SSE public events
- replay blocks
- trace spans
- evaluation inputs

The system must not reverse this dependency by treating UI-facing SSE event names as the internal truth model.

## 14. Replay and UI Projection

Replay should be event-projected, not message-reconstructed.

The projection layer should transform canonical events into stable UI blocks, such as:

- `message`
- `plan`
- `tool_call`
- `approval`
- `result`
- `error`

Approval projection must include:

- tool name
- argument preview
- risk level
- reason approval is required
- expected impact
- decision result once resolved

This allows the frontend to become a read-model consumer instead of a second runtime implementation hidden in UI state code.

## 15. Evaluation Harness

The blueprint treats evaluation as a first-class subsystem.

### 15.1 Two Harness Lenses

`Outcome Harness` verifies whether the system actually achieved the end result.

Examples:

- expected task created or not
- expected operation completed or not
- expected approval flow resumed or not
- expected final answer produced or not

`Transcript Harness` verifies whether the path was reasonable.

Examples:

- route chosen correctly
- domain chosen correctly
- right tool family selected
- approval triggered when required
- unsafe tool path avoided

### 15.2 Evaluation Strategy

The blueprint adopts two modes of success measurement:

- `Pass@k` for capability exploration during development
- `Pass^k` for regression confidence before or during rollout

The system should start with a minimal harness:

- a library of evaluation cases
- replay or run drivers
- route/tool/outcome judges
- regression suites for core runtime changes

## 16. Multi-Agent Position

Multi-agent execution is an extension point, not the v1 architectural center.

The target runtime should reserve expansion points for:

- orchestrator-worker delegation
- domain sub-runs
- specialized evaluator agents

But v1 design emphasis remains:

- single runtime loop correctness
- tool contract quality
- policy clarity
- approval and resume reliability
- replayability
- evaluation coverage

This ordering prevents the system from masking runtime weaknesses behind orchestration complexity.

## 17. Reuse and Deletion Strategy

The blueprint explicitly prefers selective reuse.

### 17.1 Reuse First

Prefer reusing durable assets:

- DAO implementations where they still map cleanly
- storage models that remain useful
- event and projection fields worth keeping
- approval persistence records that already represent durable workflow state

### 17.2 Delete First

Prefer deleting glue code that locks the module into the wrong shape:

- monolithic service orchestrators
- prompt-assembled control flow
- compatibility adapters that only exist to preserve old execution behavior
- duplicated projection logic across backend and frontend

### 17.3 Compatibility Posture

Compatibility is not the priority for this project.

Allowed migration posture:

- replace old AI APIs outright if the new shape is cleaner
- rebuild frontend AI surfaces around new projections
- drop transitional behavior if it blocks end-state clarity

If a layer exists only to preserve old runtime behavior and not to protect durable data, it should be treated as suspect by default.

## 18. Detailed Design Split

This blueprint should be followed by eight detailed designs.

1. `AI Module Blueprint Design`
The parent document. Defines end-state direction and boundaries.

2. `Core Domain Model Design`
Defines fields, relationships, and persistence intent for `Session`, `Task`, `Run`, `Turn`, `ToolCall`, `Approval`, `Event`, and `EvaluationCase`.

3. `Runtime Kernel Design`
Defines loop mechanics, run and turn transitions, checkpoint rules, resume semantics, and stop conditions.

4. `Tool Contract and Policy Design`
Defines registry shape, contract fields, invocation pipeline, risk engine, and approval policy.

5. `Event and Projection Design`
Defines canonical event schema, SSE mapping, replay block model, and trace projection rules.

6. `Copilot UI and Interaction Design`
Defines chat surface, approval UI, replay interface, and the split between conversation mode and operation mode.

7. `Gateway and API Contract Design`
Defines request and response contracts for chat, run inspection, approval, replay, and evaluation-related APIs.

8. `Evaluation Harness Design`
Defines case schema, drivers, judges, success criteria, and regression strategy.

## 19. Acceptance Criteria For This Blueprint

This blueprint is successful if it gives later design documents a fixed parent boundary:

- the product is clearly positioned as `Copilot + Runtime`
- the runtime is defined around objects and state, not around prompt glue
- tools and policy are split from model reasoning
- events are fixed as the truth model
- replay and evaluation are built into the architecture
- selective reuse and aggressive deletion are explicitly allowed
- the follow-up detailed designs can be written without redefining the core shape

## 20. Final Recommendation

OpsPilot should rebuild the AI module as a dual-core operator assistant:

- one unified user-facing AI entry
- one collaboration-centric Copilot layer
- one durable execution-centric Runtime layer
- one event-first truth model
- one contract-first tool and policy system
- one replayable and evaluable execution substrate

The shortest path to that goal is not incremental compatibility work. It is a controlled rebuild that reuses durable data assets, deletes control-flow entanglement, and lets all later detailed designs inherit from a stable parent blueprint.
