# AI Module Redesign Design

- Date: 2026-04-08
- Scope: `internal/ai`, `internal/service/ai`, AI HTTP API, AI runtime, run/event/policy/memory boundaries
- Goal: Redesign OpsPilot AI into a learning-oriented Hybrid assistant runtime that supports conversation, diagnosis, execution, approval, and replay as first-class capabilities.

## 1. Background

OpsPilot already has a meaningful AI base:

- session/message/run persistence
- SSE streaming and run projection
- approval interruption and resume
- domain-oriented sub-agents and tool integrations

But the current structure still mixes several responsibilities too tightly:

- `internal/service/ai/logic/logic.go` carries application service logic, runtime orchestration, context assembly, event consumption, and persistence coordination together
- scene prompts and tool constraints are still largely injected by building augmented user messages
- orchestrator, runtime, policy, memory, and tool contracts are not yet separated into stable subsystem boundaries

The redesign target is not backward compatibility. This is a practice project aimed at learning and implementing a stronger Agent system shape directly.

## 2. Product Positioning

OpsPilot AI should be a `Hybrid` system:

- top layer: collaborative Copilot for conversation, explanation, planning, and guidance
- bottom layer: operator runtime for diagnosable, resumable, policy-aware execution

It should act as a general platform assistant while still being able to perform:

- task execution
- operations diagnosis
- change governance

This means the architecture must separate unified entry from layered capabilities.

## 3. Design Principles

1. Session and run are different objects.
- Session exists for collaboration and long-lived context.
- Run exists for a verifiable execution loop.

2. Router decides interaction mode before choosing an agent.
- First decide whether this is conversation or operation.
- Then choose domain routing and execution mode.

3. Kernel owns execution mechanics.
- Planning loops, event emission, checkpointing, policy gating, approval, replay, and evaluation belong to the kernel.

4. Domain agents own domain reasoning, not platform control flow.
- Kubernetes, host, service, deployment, monitoring, governance agents should reason within their domain only.

5. Context must be layered.
- Stable rules, runtime state, on-demand knowledge, and memory must not be mixed into one expanding prompt blob.

6. Deterministic constraints belong outside prompts.
- Tool allowlists, blocklists, approval requirements, and execution safety rules must be machine-enforced by policy/tool systems.

7. Events are the system truth source.
- UI, replay, projection, tracing, evaluation, and debugging all derive from event streams.

8. Harness is part of the product.
- Replay, evaluation, routing inspection, and context snapshots are built-in capabilities, not afterthoughts.

## 4. Target Architecture

The redesigned AI module is organized into seven layers.

### 4.1 AI Gateway

Responsibility:

- expose HTTP/SSE/WebSocket API for chat, task, run, approval, replay, trace, and evaluation views
- translate external protocol requests into internal application calls

Non-responsibility:

- no agent orchestration
- no context assembly
- no policy decisions

### 4.2 Collaboration Layer

Responsibility:

- manage `Session`
- manage user messages and assistant messages
- maintain collaboration summaries and session-scoped memory
- provide conversation-oriented response path for lightweight requests

Non-responsibility:

- no tool execution loop
- no approval state machine

### 4.3 Agent Router

Responsibility:

- classify request type: question, investigation, execution, governance, cross-domain task
- decide interaction mode: `conversation` or `operation`
- select a domain agent or orchestration mode
- decide whether to create a `Task` and `Run`

Non-responsibility:

- no direct tool execution
- no approval enforcement
- no event persistence

Router must also provide a degradation path when classification confidence is low.

Recommended fallback behaviors:

- ask the user a direct disambiguation question for materially different paths such as chat vs diagnosis vs execution
- default to the lower-risk conversation path when no strong execution signal is present
- emit explicit routing-confidence metadata into the event stream for replay and evaluation

### 4.4 Agent Kernel

Responsibility:

- own run lifecycle
- assemble execution context
- drive planning/execution/replan loop
- invoke tools through a standard pipeline
- call policy engine before side effects
- suspend for approval and resume from checkpoint
- emit structured events
- update task/run memory
- attach evaluation hooks
- support replay and recovery

This is the core runtime of the new AI module.

Kernel state must be durable rather than memory-only. Approval waits, resumable runs, and long investigations require a persisted state machine model whose checkpoints can survive process restart and long idle gaps.

### 4.5 Domain Agents

Recommended agents:

- `kubernetes`
- `host`
- `service`
- `deployment`
- `monitoring`
- `governance`

Responsibility:

- understand requests in their domain
- produce domain-local plans
- choose domain tools
- interpret evidence and produce domain conclusions

Non-responsibility:

- no global approval logic
- no shared event protocol ownership
- no session lifecycle management

### 4.6 Tool & Policy Layer

Subsystems:

- Tool Registry
- Tool Invocation Pipeline
- Risk Engine
- Policy Engine
- Approval Engine
- Safety Guardrails

Responsibility:

- define tool contracts and metadata
- evaluate whether a tool action is allowed
- enforce dry-run, approval, confirmation, deny decisions
- standardize rollback, timeout, idempotency, and freshness checks

### 4.7 Memory & Evaluation Layer

Subsystems:

- Session Memory Manager
- Task Memory Manager
- Run Scratchpad Store
- Knowledge Memory Store
- Evaluation Harness
- Replay/Inspector views

Responsibility:

- compress and update working memory
- distill reusable knowledge
- evaluate routing, tool selection, safety, and outcomes
- provide human-debuggable replay and inspection

## 5. Core Domain Model

The redesigned AI module should treat the following six objects as first-class.

### 5.1 Session

Purpose:

- long-lived collaboration container
- stores topic, user preference, high-level summaries, linked tasks

Session should answer: what is the user trying to do over time?

### 5.2 Task

Purpose:

- semantic goal extracted from a session
- represents the business objective, not a specific model invocation

Examples:

- diagnose a spike in 5xx after deployment
- prepare a rollout plan for a service
- execute a governed production change

Task should answer: what outcome is the system trying to achieve?

### 5.3 Run

Purpose:

- one concrete execution attempt for a task
- minimum unit for verification, replay, approval, recovery, and auditing

Recommended states:

- `created`
- `planning`
- `running`
- `policy_blocked`
- `waiting_approval`
- `resuming`
- `paused`
- `completed`
- `failed`
- `cancelled`

Run should answer: what exactly happened during this attempt?

### 5.4 Event

Purpose:

- immutable fact stream for a run
- single truth source for UI, trace, replay, evaluation, and debug views

Representative event types:

- `session_started`
- `task_created`
- `run_created`
- `route_selected`
- `context_loaded`
- `plan_created`
- `step_started`
- `tool_called`
- `tool_result_received`
- `policy_evaluated`
- `approval_requested`
- `approval_resolved`
- `memory_updated`
- `run_replanned`
- `run_completed`
- `run_failed`

### 5.5 Memory

Memory is not raw chat history. It is compressed working state.

Subtypes:

- `session memory`: user intent, preferences, enduring constraints
- `task memory`: goals, facts, decisions, unresolved questions
- `run scratchpad`: current execution notes, intermediate conclusions, pending next steps
- `knowledge memory`: reusable operational knowledge distilled across tasks

### 5.6 Policy Decision

Purpose:

- structured determination of execution boundary
- deterministic result returned by policy logic, not free-form agent judgment

Representative outcomes:

- `allow`
- `allow_with_log`
- `allow_with_dry_run`
- `require_confirmation`
- `require_approval`
- `deny`

## 6. Interaction Model

The system supports two primary paths.

### 6.1 Collaboration Path

Use for:

- Q&A
- explanation
- lightweight guidance
- requirement clarification
- low-cost investigation that does not need a full run

Flow:

1. request enters gateway
2. collaboration layer loads session summary and minimal context
3. router decides this is a conversation request
4. router selects general or domain-specific answering path
5. response is returned directly
6. session summary is updated

This path should remain lightweight and fast.

### 6.2 Execution Path

Use for:

- diagnosis
- multi-step investigation
- governed change
- any request requiring verification, replay, approval, or resumability

Flow:

1. gateway receives request
2. router classifies request as operation
3. system creates `Task`
4. system creates `Run`
5. kernel assembles context and enters execution loop
6. kernel emits events on each significant state transition
7. tool invocations go through policy engine
8. approval pauses the run when necessary
9. resume continues from checkpoint
10. run completion updates task and session summaries

### 6.3 Execution Modes

Kernel should explicitly support five modes inspired by the referenced article:

1. `Direct Answer`
- no run or only a lightweight observation run

2. `Routing`
- select the right domain agent path

3. `Plan-and-Execute`
- plan first, then execute step by step

4. `Orchestrator-Workers`
- cross-domain decomposition with domain-specific workers

5. `Evaluator-Optimizer`
- critique and refine diagnosis, plan quality, or action proposals until acceptable

For latency-sensitive paths, router may use a lighter model or deterministic classifier before escalating to the full execution stack. The goal is to avoid paying full multi-stage latency for trivial or high-frequency requests.

## 7. Context Engineering

This is the most important redesign axis.

The current message augmentation strategy should be replaced with a layered context system.

### 7.1 Core System Layer

Stable, short, cache-friendly system rules:

- assistant identity
- immutable execution constraints
- completion standard
- mandatory verification standard

This layer should change rarely.

### 7.2 Capability Index Layer

Stable routing index describing:

- available domain agents
- available tool groups
- when to use each domain or skill

This layer should remain short and index-like, not full documentation.

### 7.3 Runtime Context Layer

Per-request or per-run injected state:

- session/task/run identifiers
- scene
- resource references
- user identity and permission context
- current goal
- active plan status
- latest policy state

### 7.4 On-Demand Knowledge Layer

Loaded only when needed:

- domain instructions
- tool usage guidance
- scene SOPs
- runbook excerpts
- resource-specific knowledge cards

### 7.5 Memory Layer

Loaded through managers rather than raw message replay:

- session summary
- task brief
- run scratchpad
- distilled knowledge

### 7.6 Token Budgeting

`PromptCompiler` should enforce explicit token budgeting instead of relying on best-effort prompt growth.

Recommended budget slices:

- core system and capability index: fixed reserved budget
- runtime context: bounded reserved budget
- memory summaries: elastic but capped budget
- on-demand knowledge: priority-ranked budget
- run scratchpad and recent evidence: last-budget-wins segment with automatic compression

Required behaviors:

- estimate token usage before final prompt assembly
- trigger summarization when scratchpad or evidence exceeds budget
- truncate lowest-priority context first, never core constraints first
- preserve plan decisions, unresolved blockers, and active policy state during compression
- emit budget decisions into trace metadata for debugging context loss

### 7.7 Context Rules

1. raw message history is not default context
2. long tool output should be replaced by compact summaries in prompt space
3. raw tool output belongs in content storage and event payloads
4. scene prompts become structured context providers, not direct prompt concatenation
5. tool constraints are enforced by policy/tool layers, not by prose alone

### 7.8 New Internal Components

Recommended components:

- `ContextAssembler`
- `MemoryManager`
- `KnowledgeLoader`
- `PromptCompiler`

This replaces direct `buildAugmentedMessage`-style assembly with structured context composition.

## 8. Tool Contracts and Policy

### 8.1 Tool Contract

Each tool should be defined by structured metadata, not only name and description.

Recommended fields:

- `tool_id`
- `domain`
- `intent_tags`
- `risk_level`
- `side_effect_level`
- `requires_freshness`
- `requires_approval`
- `rollback_support`
- `input_schema`
- `result_schema`
- `evidence_schema`
- `timeout`
- `idempotent`
- `dry_run_supported`

### 8.2 Tool Grouping

Recommended groups:

1. `Observe Tools`
- status, logs, metrics, resources, release history

2. `Analyze Tools`
- aggregation, comparison, diffing, risk analysis, diagnosis assistance

3. `Change Tools`
- deploy, scale, restart, patch, rollback, policy-changing operations

4. `Control Tools`
- pause run, resume run, submit approval, create task, switch domain, record decision

This grouping gives the kernel a better decision basis than a flat tool list.

### 8.3 Policy Engine

Before executing a side-effectful step, kernel must query policy engine.

Inputs:

- task and run context
- user identity and role
- target environment and resource
- tool contract metadata
- active platform state
- approval availability
- rollback readiness

Outputs:

- `allow`
- `allow_with_log`
- `allow_with_dry_run`
- `require_confirmation`
- `require_approval`
- `deny`

This decision is deterministic and outside model discretion.

### 8.4 Approval Engine

Approval is a run state transition, not a UI-only concept.

Recommended state transition:

- `running -> policy_blocked -> waiting_approval -> resuming -> completed|failed`

Approval payload should include:

- requested action
- reason approval is required
- risk summary
- expected impact
- rollback plan
- supporting evidence
- linked run step

Implementation direction:

- treat approval and resume as durable workflow transitions rather than in-memory goroutine suspension
- persist checkpoint, current step, pending approval payload, and resume target in database-backed runtime state
- design the kernel state model with the same durability goals as workflow engines such as Temporal or AWS Step Functions, while keeping the implementation local to OpsPilot

### 8.5 Safety Model

Safety should cover both execution and cognition.

Execution safety:

- dangerous action interception
- environment boundary enforcement
- dry-run requirement
- rollback precondition checks
- approval enforcement

Cognitive safety:

- no unsupported factual claims without evidence
- diagnosis conclusions must include evidence references
- risky recommendations must include uncertainty
- changes must include expected effect and rollback path

## 9. Router, Kernel, and Domain Agent Responsibilities

### 9.1 Router

Responsible for:

- request classification
- choosing conversation vs operation
- selecting domain or orchestration mode
- deciding task/run creation

Not responsible for:

- tool execution
- approval
- persistence mechanics
- replay

### 9.2 Kernel

Responsible for:

- execution mechanics
- context loading
- event emission
- policy/approval integration
- checkpoint/resume
- retry/replan/termination logic
- memory updates
- evaluation hooks

Not responsible for:

- deep domain-specific reasoning details

### 9.3 Domain Agents

Responsible for:

- domain understanding
- domain-local planning
- domain tool selection
- evidence interpretation
- domain summary

Not responsible for:

- global state machine control
- approval orchestration
- shared runtime contracts

### 9.4 Cross-Domain Tasks

Cross-domain work should enter an orchestrated run mode.

Examples:

- service release causes 5xx spike
- root cause may span deployment, service, monitoring, and Kubernetes

In this mode:

- kernel starts an orchestrated run
- main orchestrator creates a task graph
- domain agents execute independent or sequential slices
- evaluator merges evidence, contradictions, and unresolved questions
- final run result determines whether to stay diagnostic or enter change path

## 10. Event-Driven Observability and Harness

### 10.1 Event Stream as Truth Source

All AI behaviors should be represented through a unified run event stream.

Consumers:

- UI timeline
- run projection
- replay system
- trace explorer
- evaluation pipeline
- debugging tools

### 10.2 Three Visibility Layers

1. `Business Timeline`
- what a normal user/operator should see

2. `Agent Trace`
- routing, plan changes, tool chains, domain handoffs

3. `System Telemetry`
- latency, token usage, failure rate, approval wait time, resume success, tool success rate

All three layers should connect through the same `run_id` and `trace_id`.

Internal event structures should also map cleanly onto standard observability concepts such as spans and attributes so that Agent traces can be exported to OpenTelemetry-compatible backends when needed.

### 10.3 Evaluation Harness

Minimum built-in evaluation surfaces:

1. `Routing Eval`
- was the request routed to the right mode and domain?

2. `Tool Selection Eval`
- were the right tools chosen?

3. `Outcome Eval`
- did the conclusion or action achieve the goal with evidence?

4. `Safety Eval`
- were policy, approval, and evidence rules respected?

### 10.4 Replay as First-Class Capability

Replay must show more than SSE output. It should allow inspection of:

- context snapshot used at each major stage
- route decision
- plan evolution
- tool selections and results
- policy decisions
- approval pauses and resumes
- failure or replan causes

### 10.5 Developer-Facing Inspection Surfaces

Recommended built-in views:

- run inspector
- task graph viewer
- context snapshot viewer
- policy decision viewer
- tool contract explorer
- evaluation report viewer

These views are part of the learning value of the project.

## 11. OpenTelemetry Alignment

The AI runtime keeps its own event taxonomy, but it should not become an observability island.

Recommended mapping:

- `Run` lifecycle maps to a root trace/span group
- routing, planning, tool invocation, policy evaluation, approval wait, resume, and completion map to child spans
- event payload metadata maps to span attributes
- error and retry information maps to status/error annotations

This preserves a domain-native event model while allowing export into Jaeger, Datadog, Tempo, or other OpenTelemetry-compatible systems.

## 12. Proposed Internal Package Direction

This section describes the intended package shape, not a mandatory file-level final structure.

```text
internal/ai/
  gateway/          # protocol adapters if AI-specific gateway logic is extracted
  application/      # session/task/run application services
  router/           # request classification and mode selection
  kernel/           # run engine, lifecycle, resume, orchestration
  agents/           # domain agents only
  context/          # assembler, prompt compiler, scene providers, knowledge loaders
  memory/           # session/task/run memory managers
  policy/           # risk engine, approval policy, guardrails
  tools/            # registry, contracts, invocation pipeline
  events/           # event definitions, appenders, projectors, replay adapters
  eval/             # routing/tool/outcome/safety evaluation
  inspect/          # trace/replay/context inspection surfaces
```

A practical code goal is to remove the current concentration of responsibilities in service logic and promote AI internals into stable subsystems.

## 13. Migration Direction

Compatibility is not a design goal. The redesign can be implemented through direct restructuring.

Recommended phases:

1. establish the new object model
- add explicit `Task`
- normalize `Run` state machine
- normalize event taxonomy

2. extract the execution kernel
- move run lifecycle, policy checks, replay, and resume out of service logic

3. replace prompt augmentation with layered context
- introduce assembler, memory manager, knowledge loader, prompt compiler

4. normalize tools and policy
- convert tools to structured contracts
- move approval and safety checks into policy engine

5. split router from domain agents
- router decides mode/domain
- domain agents focus on domain reasoning only

6. add harness surfaces
- replay inspector, context snapshot, evaluation reporting

## 14. Risks

1. Overbuilding before proving behavior
- mitigation: implement one thin vertical slice first, but keep target boundaries intact

2. Kernel becoming another monolith
- mitigation: keep policy, memory, events, and context as separate subsystems under kernel coordination

3. Domain agents leaking control-flow concerns back in
- mitigation: standardize domain agent interfaces and keep lifecycle ownership in kernel

4. Evaluation data becoming too expensive or noisy
- mitigation: separate user timeline, agent trace, and system telemetry instead of mixing all details into one stream

5. Latency amplification across stacked model calls
- mitigation: show event-driven progress in UI, prefer small/fast models for routing and simple classification, and avoid entering the full execution stack for trivial requests

## 15. Success Criteria

The redesign is successful when:

1. Session, task, and run are clearly separated in both code and behavior.
2. Router decides conversation vs operation before domain dispatch.
3. Kernel owns execution lifecycle, policy, approval, and resume.
4. Domain agents no longer own global control flow.
5. Context is assembled in layers instead of being concatenated into one message blob.
6. Tool use is governed by structured contracts and deterministic policy decisions.
7. Event stream becomes the common source for projection, replay, trace, and evaluation.
8. The system can explain not only what it answered, but also why it acted, why it paused, and how it can be replayed.

## 16. Final Recommendation

OpsPilot AI should evolve from a chat-driven tool-calling module into an `Agent Runtime` with:

- collaborative session entry
- task/run execution core
- structured context engineering
- deterministic policy and approval boundaries
- domain-specialized agents
- event-native observability
- built-in replay and evaluation harness

For a learning-oriented practice project, this is the most valuable architecture to implement because it exposes the real engineering surfaces of Agent systems rather than hiding them inside prompts or oversized service handlers.
