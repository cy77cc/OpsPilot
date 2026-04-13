# AI Runtime Architecture Redesign Design

- Date: 2026-04-10
- Scope: `internal/modules/ai`, AI router/runtime/context/tool architecture
- Goal: Replace the current AI orchestration shape with a learning-first, functionality-first architecture centered on a strong runtime kernel, LLM-first routing, tool discovery, and layered context management
- Status: Proposed parent design for the next implementation cycle

## 1. Background

OpsPilot already has meaningful AI assets:

- durable run, event, projection, and checkpoint data
- approval interruption and resume
- domain-oriented tools and specialist agents
- frontend SSE and projection consumption

But the current architecture still has the wrong center of gravity:

- `internal/modules/ai/logic/logic.go` mixes copilot flow, runtime orchestration, context assembly, event consumption, and persistence coordination
- `internal/modules/ai/agent/tools/orchestrator/tools.go` statically assembles a large multi-agent surface instead of supporting demand-driven delegation
- tool exposure is too broad, causing context pressure and tool-selection burden
- context is still too close to “prompt history assembly” instead of a true layered context system

This project is explicitly learning-first and functionality-first. Backward compatibility is not a design goal. If a cleaner architecture is materially better, the old orchestration shape should be replaced rather than preserved.

## 2. Design Position

The target architecture should not be “more agents by default”.

The target architecture should be:

- one strong runtime kernel
- one default primary agent
- optional specialist agents invoked only when justified
- tool discovery before tool loading
- layered context before prompt growth

This follows three practical conclusions from the referenced articles:

1. Multi-agent is useful when it improves context isolation, parallel search, or specialization.
2. Multi-agent is harmful when it adds coordination overhead without a true context boundary.
3. Context engineering is a first-class system concern, not a prompt-writing afterthought.

## 3. Primary Recommendation

OpsPilot AI should adopt a mixed architecture:

- `Session-only` path for lightweight collaboration and question answering
- `Session -> Task -> Run` path for execution-shaped or auditable work
- `single agent by default`
- `multi-agent only when the router judges that the task benefits from delegation`

This is not a compromise. It is the intended steady-state system shape.

## 4. Architecture Principles

1. Runtime quality comes before multi-agent complexity.
2. LLMs should make routing and formatting decisions whenever the decision is semantic rather than deterministic.
3. Deterministic code should remain only at hard boundaries such as auth, schema validation, approval gating, and dangerous side-effect enforcement.
4. Context should be written, selected, compressed, and isolated instead of endlessly appended.
5. Tool definitions should be discovered on demand, not preloaded wholesale.
6. Agents should be decomposed by context boundary, not by workflow stage.
7. Events and durable state remain the truth source for replay, auditing, recovery, and evaluation.
8. Large tool outputs should be stored as artifacts and selectively reintroduced, not dumped back into the model transcript.

## 5. Target Topology

The redesigned AI module should be organized into six primary layers.

### 5.1 Copilot

Responsibilities:

- manage `Session`
- support lightweight collaborative conversation
- maintain session summaries and user preference memory
- decide when a request can stay in collaboration mode

Non-responsibilities:

- no heavy run state machine
- no approval ownership
- no direct tool orchestration loop

### 5.2 Router

Responsibilities:

- classify `conversation` vs `operation`
- decide whether to create or continue `Task` and `Run`
- choose execution shape: `single_agent`, `delegated_specialist`, or `parallel_subagents`
- choose domain route and context plan

Non-responsibilities:

- no direct business tool execution
- no durable approval ownership
- no replay projection ownership

Router should be LLM-first. It should produce structured decisions, and code should validate those decisions rather than replace them with large rule trees.

### 5.3 Runtime

Responsibilities:

- own `Run` lifecycle
- execute the turn loop
- assemble execution context
- dispatch the selected agent path
- handle approval pause and resume
- persist runtime facts and canonical events

Runtime is the operational core of the system.

### 5.4 Agent Surface

Responsibilities:

- provide one default `primary agent`
- expose specialist agents for bounded domain work
- expose a verifier agent for black-box validation

The agent surface exists to support context isolation and specialization. It does not own lifecycle, approval, or persistence.

### 5.5 Tooling

Responsibilities:

- maintain tool catalog metadata
- support `tool_search`-first discovery
- normalize tool invocation and result contracts
- classify output shape and risk

### 5.6 Memory and Artifacts

Responsibilities:

- maintain layered context objects
- store summaries, scratchpads, and long-form tool outputs
- support selective retrieval into the model context

## 6. Session, Task, and Run

OpsPilot should explicitly adopt the mixed interaction model.

### 6.1 Session

`Session` remains the long-lived collaboration container.

Use `Session` only when:

- the user is asking for explanation, planning, or guidance
- no auditable execution attempt is needed
- no approval or checkpoint-backed continuation is required

### 6.2 Task

`Task` represents the business objective.

Create or continue a `Task` when:

- the user asks for diagnosis, execution, governed change, or investigation with durable progress
- the request may span multiple runs
- the system needs a stable object representing the goal

### 6.3 Run

`Run` represents one concrete execution attempt for a task.

Create a `Run` when:

- the request is operational rather than conversational
- durable state, replay, approval, or recovery may be needed
- the system is about to enter a runtime decision loop

Do not create a `Run` for every ordinary chat turn.

## 7. LLM-First Router

The router should produce a structured result rather than a free-form explanation.

Recommended schema:

```json
{
  "mode": "conversation | operation",
  "task_action": "none | create_task | continue_task",
  "execution_shape": "single_agent | delegated_specialist | parallel_subagents",
  "domain": "general | host | kubernetes | service | deployment | monitoring | governance | mixed",
  "needs_approval_risk_review": true,
  "context_plan": {
    "session_memory": "none | light | standard",
    "task_memory": "none | attach",
    "run_scratchpad": "none | create",
    "knowledge_lookup": [],
    "tool_strategy": "direct | tool_search_first"
  },
  "reason": "short explanation",
  "confidence": 0.0
}
```

Code responsibilities after router output:

- validate schema
- reject invalid enum values
- apply hard safety gates
- fall back safely on low-confidence or malformed output

The router should not be replaced by large static rule trees. It should remain a semantic decision layer powered by the model.

## 8. Runtime Shape

The runtime should be split out of the current monolith and rebuilt around clear boundaries.

Recommended module responsibilities:

- `copilot/service`: session-only collaboration path
- `router/service`: structured route decisions
- `runtime/kernel`: run state machine and turn loop
- `runtime/context`: execution context assembly
- `runtime/dispatcher`: primary/specialist/verifier dispatch
- `projection`: SSE and replay projection
- `policy`: approval, risk review, and side-effect gates

Recommended run states:

- `created`
- `routing`
- `planning`
- `executing`
- `waiting_approval`
- `resuming`
- `completed`
- `failed`
- `cancelled`

Critical rule:

- `conversation` stays outside the heavy run state machine
- only `operation` enters `Task` and `Run`

## 9. Agent Strategy

OpsPilot should stop treating every domain agent as a default peer in the top-level prompt.

### 9.1 Default Agent

`primary_agent` should handle the majority of tasks.

It should know:

- how to ask for more context
- how to use `tool_search`
- how to delegate when justified
- how to answer or finish cleanly

### 9.2 Specialist Agents

Keep a small set of bounded specialists:

- `host`
- `kubernetes`
- `service`
- `deployment`
- `monitoring`
- `governance`
- `verifier`

They should be loaded on demand, not exposed by default in one large multi-agent surface.

### 9.3 What To Avoid

Do not decompose by workflow stage:

- planner agent
- implementation agent
- test-writing agent
- review agent

as default production flow for the same unit of work.

That decomposition causes repeated context handoff and fidelity loss.

### 9.4 When To Upgrade To Multi-Agent

The router should choose delegation when one or more of these conditions hold:

- the subtask has a clean context boundary
- the subtask requires specialized tools or domain instruction
- the task can be parallelized without heavy shared state
- verification can be done black-box by a verifier

Multi-agent should be an upgrade path, not the default shape.

## 10. Tool Architecture

Tool overload is one of the current architecture’s main weaknesses.

The redesign should center tools around `tool_search`.

### 10.1 Default Tool Visibility

The primary agent should only keep a minimal always-visible tool surface:

- `tool_search`
- `load_session_memory`
- `load_task_context`
- `load_run_state`
- `delegate_to_specialist`
- `finish_or_answer`

Domain business tools should not be permanently attached to the primary agent prompt.

### 10.2 Tool Search First

Use Eino ADK `tool_search` middleware as the primary tool exposure strategy.

Flow:

1. agent recognizes a capability gap
2. agent calls `tool_search`
3. system returns a small candidate set
4. agent chooses and calls one tool

This reduces tool-selection burden and prompt bloat.

### 10.3 Tool Catalog Metadata

Each tool should publish metadata including:

- `domain`
- `capability`
- `risk_level`
- `side_effect_level`
- `output_mode`
- `expected_token_volume`
- `requires_freshness`
- `requires_approval`
- `idempotent`

This metadata should be reusable by router, runtime, policy, replay, and evaluation.

## 11. Artifact-First Tool Outputs

Long tool outputs should stop flowing directly into the message transcript.

Recommended output modes:

- `inline`: small result safe to return directly
- `summary_plus_artifact`: model gets summary, full output stored as artifact
- `artifact_only`: model gets handle and metadata only

Default candidates for artifact-first behavior:

- command execution
- logs
- large Kubernetes listings
- detailed status dumps
- long policy simulation results

The model should receive:

- execution status
- concise summary
- artifact id
- optionally selected excerpts

not the full raw payload by default.

## 12. Context Layering

OpsPilot should formalize context into five layers.

### 12.1 Instruction Layer

Stable role, system rules, safety posture, and platform-level constraints.

### 12.2 Session Memory

User preference, collaboration summary, long-lived goals, and recurring assumptions.

### 12.3 Task Memory

Business objective, success criteria, scope, and task-level constraints.

### 12.4 Run Scratchpad

Short-lived execution memory:

- current plan
- evidence summary
- unresolved questions
- next candidate action

### 12.5 Artifacts

Tool outputs, long logs, raw listings, retrieved documents, and other token-heavy objects stored outside direct message history.

Key context rules:

- do not treat message history as the only memory mechanism
- do not rehydrate all old content into the prompt
- select and compress before injection
- isolate large state outside the visible model transcript
- specialists receive only the context slice they need

## 13. Recommended Codebase Reshaping

### 13.1 Preserve as Assets, Not As Final Architecture

The following ideas and assets are worth preserving:

- run, event, projection, and checkpoint durability
- approval pause and resume model
- domain tool business logic
- existing frontend replay/projection consumption

### 13.2 Rewrite or Strongly Refactor

The following implementation areas should be treated as redesign targets:

- `internal/modules/ai/logic/logic.go`
- `internal/modules/ai/agent/tools/orchestrator/tools.go`
- `internal/modules/ai/agent/tools/registry.go`

These files represent the old center of gravity and should not be incrementally patched forever.

### 13.3 Reconsider Agent Boundaries

The following agent packages should be reviewed for merge, deletion, or conversion into tool-catalog categories:

- `internal/modules/ai/agent/tools/host`
- `internal/modules/ai/agent/tools/infrastructure`

If they do not represent stable context-isolated domains, they should not survive as first-class specialists.

### 13.4 Recommended New Module Boundaries

Suggested module groups for the rebuild:

- `internal/modules/ai/handler/chat`
- `internal/modules/ai/handler/approval`
- `internal/modules/ai/agent/runtime`
- `internal/modules/ai/service`
- `internal/modules/ai/agent/tools`
- `internal/modules/ai/service`
- `internal/modules/ai/agent/shared/approval`

Exact package names may change, but the boundary intent should remain stable.

## 14. Execution Priorities

Recommended implementation order:

1. establish router contract and mixed Session/Task/Run entry rules
2. introduce `tool_search`-first tool exposure
3. introduce layered context and artifact-first tool results
4. rebuild runtime boundaries around kernel, context, dispatcher, and projection
5. shrink static multi-agent orchestration into primary agent plus on-demand specialists
6. add verifier path and evaluation hooks

This order keeps the runtime center stable before adding more agent autonomy.

## 15. Final Recommendation

OpsPilot should not evolve toward “always-on multi-agent orchestration”.

OpsPilot should be rebuilt as:

- a dual-path `Copilot + Runtime` AI system
- with LLM-first structured routing
- with `tool_search` as the default tool exposure strategy
- with layered context and artifact-first outputs
- with one default primary agent
- and with specialist or parallel subagents used only when the router judges that they provide real context or specialization benefits

This architecture best matches the project’s learning-first posture, reduces accidental complexity, and creates a cleaner base for future experimentation.
