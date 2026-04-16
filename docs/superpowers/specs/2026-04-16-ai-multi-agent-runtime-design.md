# AI Multi-Agent Runtime Design

- Date: 2026-04-16
- References:
  - `docs/agent-integration-plan.md`
  - `docs/agent-integration-summary.md`
  - `docs/superpowers/specs/2026-04-10-ai-runtime-architecture-redesign-design.md`
  - `docs/superpowers/specs/2026-04-11-ai-module-execution-design.md`
- External references:
  - Tw93, `https://tw93.fun/2026-03-21/agent.html`
  - Anthropic, `https://claude.com/blog/building-multi-agent-systems-when-and-how-to-use-them`
  - LangChain, `https://blog.langchain.com/context-engineering-for-agents/`
  - CloudWeGo Eino ADK, `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/`
  - CloudWeGo Eino Supervisor, `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/supervisor/`
  - CloudWeGo Eino DeepAgents, `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/deepagents/`
  - CloudWeGo Eino HITL, `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_hitl/`
  - CloudWeGo Eino ChatModelAgent Middleware, `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/`
  - CloudWeGo Eino ToolReduction Middleware, `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_toolreduction/`
- Scope: multi-agent runtime shape for `internal/modules/ai`, including agent roles, runtime state machine, middleware layering, delegation contracts, summary-only return rules, event/projection model, recovery rules, and AI module directory boundaries
- Goal: add a controlled multi-agent design that solves context pollution from heavy tool outputs, especially Prometheus and other high-volume operational tools, without giving up approval control, replayability, or a single user-facing run model

## 1. Background

OpsPilot already has useful AI assets:

1. durable run, event, projection, and checkpoint storage
2. approval interruption and resume
3. domain-oriented tools and scenes
4. SSE-driven frontend projections
5. an initial agent integration design centered on one agent plus scene middleware

That shape is a good starting point but it is not sufficient for heavy operational analysis.

Prometheus, Kubernetes inventory, logs, and host runtime inspections can return very large outputs. If those outputs are pushed back into the primary agent transcript, the main agent loses context budget on raw evidence instead of spending it on routing, judgment, and user-facing explanation.

The target architecture should therefore not be "multi-agent everywhere". It should be:

1. one main runtime owner
2. specialist agents only where domain isolation helps
3. temporary isolation workers for high-volume analysis
4. summary-only returns from child agents
5. approval and write control kept at the main agent boundary

This matches the practical advice in the external references:

1. start from one stable loop and split only when context isolation or specialization is justified
2. treat context as managed memory rather than a growing transcript
3. keep large tool outputs outside the main context and reintroduce only compressed evidence

## 2. Design Decision

OpsPilot should adopt a mixed multi-agent runtime:

1. a single user-facing `Run`
2. one `SupervisorAgent` as the orchestration owner
3. a small set of fixed read-only `SpecialistAgent`s
4. an on-demand `IsolationWorker` for high-volume analysis
5. all write actions routed back through the supervisor and the approval pipeline

This is the preferred design over the alternatives below.

### 2.1 Alternatives considered

| Option | Shape | Why not preferred |
| --- | --- | --- |
| A | single agent plus scene middleware only | insufficient context isolation for Prometheus and other large-output tools |
| B | fixed specialists only | helps domain routing but does not fully solve extreme output volume inside a domain |
| C | supervisor plus specialists plus temporary isolation worker | recommended because it handles both specialization and output isolation |
| D | fully dynamic free-form agent creation | too expensive to reason about; lifecycle, approval, and audit boundaries become unstable |

### 2.2 Recommendation

Use option C.

It gives OpsPilot:

1. one stable runtime owner
2. clear specialist boundaries
3. a dedicated place to absorb large tool outputs
4. one consistent approval and recovery path

## 3. Core Architecture

The runtime should be organized into three execution layers.

### 3.1 Orchestrator layer

The orchestrator layer owns the user-facing run.

Primary responsibilities:

1. receive the request and maintain the main run lifecycle
2. decide whether to answer directly, delegate to a specialist, or ask a worker for isolated analysis
3. own approval, interruption, and resume
4. merge child summaries into a final user-facing answer
5. emit canonical runtime events and public projections

Non-responsibilities:

1. no direct ownership of domain-specific analysis logic
2. no direct storage of large raw tool payloads inside the main transcript
3. no write execution by child agents

### 3.2 Specialist layer

Fixed specialist agents should be created only for stable operational domains.

Initial specialists:

1. `MonitorAgent`
2. `KubernetesAgent`
3. `HostAgent`
4. `CICDAgent`

Primary responsibilities:

1. handle domain-specific read-only analysis
2. select bounded tools within the domain
3. decide whether a heavy task needs an isolation worker
4. return a compact domain summary to the supervisor

Non-responsibilities:

1. no direct write execution
2. no independent user-facing run lifecycle
3. no uncontrolled recursive delegation

### 3.3 Isolation worker layer

The isolation worker exists for one reason: context protection.

Typical triggers:

1. Prometheus time series with large time windows or high-cardinality labels
2. large Kubernetes inventory scans
3. bulky host inspection output
4. heavy log or event aggregation

Primary responsibilities:

1. consume heavy inputs
2. reduce them into stable findings
3. persist raw data as artifacts
4. return summary-only results

Non-responsibilities:

1. no user-facing reply
2. no write actions
3. no further delegation

### 3.4 Standard execution path

Recommended execution path:

`User -> SupervisorAgent -> SpecialistAgent -> IsolationWorker -> summary -> SpecialistAgent -> SupervisorAgent -> UI`

For write actions:

`User -> SupervisorAgent -> policy and approval -> write tool -> UI`

## 4. AI Module Directory Structure

The current AI module should evolve toward explicit runtime and agent boundaries instead of concentrating orchestration inside `logic.go` and shared middleware folders.

Recommended structure:

```text
internal/modules/ai/
  agent/
    orchestrator/
      supervisor.go
      dispatch.go
      registry.go
      policy_gate.go
    specialists/
      monitor/
        agent.go
        prompt.go
        tools.go
        summarize.go
      kubernetes/
        agent.go
        prompt.go
        tools.go
        summarize.go
      host/
        agent.go
        prompt.go
        tools.go
        summarize.go
      cicd/
        agent.go
        prompt.go
        tools.go
        summarize.go
    workers/
      isolation/
        agent.go
        task.go
        reduce.go
        artifact.go
    middleware/
      shared/
        patch_tool_calls.go
        audit_bridge.go
        artifact_offload.go
        summary_formatter.go
      supervisor/
        approval_gate.go
        history_summary.go
      specialists/
        domain_prompt.go
        tool_reduction.go
      workers/
        strict_summary.go
    runtime/
      kernel.go
      run_state.go
      checkpoint.go
      resume.go
      events.go
      envelope.go
      projection.go
      projector.go
      public_event.go
      delegation_node.go
    contracts/
      delegation.go
      summary.go
      artifact.go
      failure.go
      scope.go
      approval.go
    prompts/
      supervisor.md
      monitor.md
      kubernetes.md
      host.md
      cicd.md
      isolation_worker.md
  logic/
    chat/
    approval/
    stream/
    event/
  dao/
  handler/
  model/
```

### 4.1 Boundary rules

1. `agent/orchestrator` owns cross-agent control flow
2. `agent/specialists` own domain analysis behavior
3. `agent/workers` own high-volume reduction tasks
4. `agent/contracts` define cross-layer protocol types and should be reused by runtime, projection, and audit code
5. `agent/runtime` owns execution lifecycle, checkpoint, resume, and projection rules

## 5. Runtime State Machine

OpsPilot should expose one main run per user request, even when the work internally spans multiple agents.

### 5.1 Main run states

Recommended main run states:

1. `created`
2. `routing`
3. `executing`
4. `delegating`
5. `waiting_subagent`
6. `waiting_approval`
7. `resuming`
8. `completed`
9. `failed`
10. `cancelled`

### 5.2 State meanings

| State | Meaning |
| --- | --- |
| `created` | run exists but no execution has started |
| `routing` | supervisor is deciding whether to answer, delegate, or prepare a write action |
| `executing` | active agent is reasoning or invoking safe tools |
| `delegating` | runtime is creating a child task for a specialist or worker |
| `waiting_subagent` | main run is paused while waiting for a child summary |
| `waiting_approval` | write execution is blocked on approval |
| `resuming` | runtime is restoring a paused path after approval or child completion |
| `completed` | main answer has been finalized |
| `failed` | no safe or useful continuation exists |
| `cancelled` | user or system cancelled the run |

### 5.3 Child lifecycle

Child agents use a lighter lifecycle:

1. `spawned`
2. `analyzing`
3. `summarizing`
4. `returned`
5. `failed`
6. `expired`

Child agents are runtime-internal units. They do not become first-class user-facing runs.

## 6. Delegation Topology and Limits

Delegation depth should be intentionally shallow.

Allowed paths:

1. `SupervisorAgent -> SpecialistAgent`
2. `SpecialistAgent -> IsolationWorker`

Forbidden path:

1. `IsolationWorker -> any further delegation`

This enforces a maximum practical depth of two child layers and prevents uncontrolled agent trees.

### 6.1 Prometheus standard flow

Prometheus-heavy work should follow this standard path:

1. supervisor routes to `MonitorAgent`
2. `MonitorAgent` detects large result volume or expensive aggregation
3. `MonitorAgent` delegates to `IsolationWorker`
4. `IsolationWorker` executes heavy analysis and stores raw outputs as artifacts
5. `IsolationWorker` returns a compressed summary
6. `MonitorAgent` adds domain interpretation
7. supervisor merges the domain summary into the main answer

## 7. Middleware Strategy

The first implementation should not force `tool_search` across all layers.

### 7.1 Recommendation

1. do not enable full-chain `tool_search` by default
2. keep specialist tool sets small and static first
3. rely on `ToolReduction`, artifact offload, and summary contracts to solve output pollution
4. allow optional `tool_search` later on the supervisor if the visible tool and specialist set grows materially

### 7.2 Why not force `tool_search`

`tool_search` solves tool exposure volume, not result volume. OpsPilot's urgent problem is result pollution from large tools such as Prometheus. For the first multi-agent iteration, static and bounded specialist tool sets are easier to reason about and easier to verify.

### 7.3 Middleware by layer

Recommended first-pass middleware composition:

#### Supervisor

1. approval gate
2. audit and event bridge
3. patch tool calls
4. history summarization

#### Specialist

1. domain prompt injection
2. tool reduction
3. artifact offload
4. summary formatter
5. audit and event bridge

#### Isolation worker

1. tool reduction
2. artifact offload
3. strict summary formatter
4. patch tool calls

### 7.4 Middleware role boundaries

1. approval belongs only to the supervisor
2. reduction belongs where large outputs are produced
3. artifact offload is mandatory wherever raw payloads may exceed safe context size
4. summary formatting must enforce short, structured returns before child results reach the parent

## 8. Delegation Contract

Delegation should use explicit typed contracts rather than free-form prompts alone.

### 8.1 Delegation task

Recommended `DelegationTask` fields:

1. `task_id`
2. `parent_run_id`
3. `delegation_id`
4. `target_agent`
5. `intent`
6. `question`
7. `scope`
8. `constraints`
9. `input_artifacts`
10. `expected_output`
11. `deadline_hint`

### 8.2 Scope model

`scope` should be reusable across agents and tools. It should support fields such as:

1. `cluster`
2. `namespace`
3. `service`
4. `host`
5. `time_range`
6. `environment`

### 8.3 Expected output

`expected_output` should narrow the child result shape. Example categories:

1. `metric_anomaly_summary`
2. `resource_inventory_summary`
3. `host_health_summary`
4. `pipeline_failure_summary`
5. `release_readiness_summary`

## 9. Summary Return Contract

Child agents return summaries, not raw transcripts and not raw payloads.

### 9.1 Delegation summary

Recommended `DelegationSummary` fields:

1. `task_id`
2. `agent_name`
3. `status`
4. `summary`
5. `key_findings`
6. `risk_level`
7. `confidence`
8. `recommended_next_action`
9. `artifact_refs`
10. `metrics`

### 9.2 Prometheus rule

Prometheus-related child agents must not return:

1. full time series
2. full label sets
3. large tables
4. raw response JSON

They may return:

1. anomaly summary
2. aggregate metrics such as peak, baseline, delta, or error-rate growth
3. affected scope
4. time window
5. artifact references

### 9.3 Child agent permissions

The first multi-agent version should enforce:

1. specialist and worker agents are read-only
2. they may recommend actions but not execute writes
3. write actions must always be re-issued by the supervisor through the approval path

## 10. Event Model and Projection

Multi-agent runtime needs separate internal and public event layers.

### 10.1 Internal event families

Recommended families:

1. `run.*`
2. `agent.*`
3. `delegation.*`
4. `tool.*`
5. `approval.*`
6. `artifact.*`
7. `summary.*`
8. `projection.*`

### 10.2 Event envelope

Recommended envelope fields:

1. `event_id`
2. `run_id`
3. `turn_id`
4. `agent_kind`
5. `agent_name`
6. `event_type`
7. `status`
8. `parent_event_id`
9. `delegation_id`
10. `artifact_refs`
11. `occurred_at`
12. `payload`

### 10.3 Public projection model

The frontend should not consume all internal events. It should consume public projection nodes only.

Recommended public node types:

1. `message.user`
2. `run.phase`
3. `delegation.node`
4. `approval.node`
5. `message.assistant`
6. `run.error`
7. `run.completed`

### 10.4 Delegation node shape

Recommended `delegation.node` fields:

1. `node_id`
2. `run_id`
3. `delegation_id`
4. `agent_name`
5. `intent`
6. `status`
7. `title`
8. `summary`
9. `risk_level`
10. `started_at`
11. `finished_at`

### 10.5 Projection limits

The public projection must not include:

1. child internal reasoning
2. raw Prometheus series
3. raw tool payloads
4. large JSON bodies
5. worker intermediate detail

Raw evidence belongs in artifacts, audit logs, and internal traces.

## 11. Approval and Recovery

Approval is a supervisor-only concern.

### 11.1 Approval rules

1. only the supervisor may issue write intents
2. child agents may suggest actions but may not trigger writes directly
3. all write execution must pass through policy evaluation and approval

### 11.2 Resume target

Recovery should use explicit resume addresses rather than vague phase strings.

Recommended examples:

1. `supervisor.after_specialist_summary`
2. `supervisor.before_write_tool_call`
3. `supervisor.after_approval_granted`
4. `specialist.after_worker_summary`

### 11.3 Checkpoint content

Checkpoint content should stay summary-oriented.

Minimum fields:

1. main run state
2. active agent identity
3. delegation tree summary
4. completed child summaries
5. pending approval snapshot
6. pending tool invocation
7. artifact references
8. projection cursor

The checkpoint restores control flow. It is not a full transcript backup.

### 11.4 Cancellation and expiry

1. cancelling the main run cancels all unfinished child tasks
2. temporary workers may expire after return; artifacts and summaries remain
3. approval timeout should not continue automatically

## 12. Failure Model

Failure codes should be structured and actionable.

### 12.1 Recommended failure families

1. `unsupported_scope`
2. `insufficient_data`
3. `tool_failed`
4. `timeout`
5. `artifact_write_failed`
6. `summary_generation_failed`
7. `policy_denied`

### 12.2 Required runtime responses

| Failure code | Required runtime response |
| --- | --- |
| `tool_failed` | retry once in the same bounded context or narrow the scope |
| `insufficient_data` | return to supervisor for a smaller scope or a user clarification |
| `timeout` | return partial summary with lower confidence |
| `artifact_write_failed` | fail the child task; do not bypass artifact storage by dumping raw payload into parent context |
| `summary_generation_failed` | retry once with a shorter summary request |
| `unsupported_scope` | reroute in the supervisor |
| `policy_denied` | stop the action and surface a recommendation only |

## 13. Testing and Verification Strategy

Multi-agent implementation should be accepted only if the orchestration layer is tested directly.

### 13.1 Test layers

1. contract tests for `DelegationTask`, `DelegationSummary`, scope objects, and failure codes
2. middleware tests for reduction, artifact offload, summary formatting, and approval ownership
3. runtime orchestration tests for supervisor-specialist-worker transitions
4. projection tests for public node generation

### 13.2 Required acceptance scenarios

1. Prometheus-heavy results do not bloat the supervisor context
2. child agents return summaries only
3. specialist agents may call the isolation worker, but the worker may not recurse
4. write actions always return to the supervisor and approval path
5. approval resume restores the correct resume target
6. child failure does not destroy main run continuity
7. the frontend shows summary delegation nodes but not internal child detail

### 13.3 Metrics to collect

1. delegation count per run
2. specialist hit rate
3. worker trigger rate
4. tool reduction trigger rate
5. artifact write volume
6. child summary token length
7. approval pause and resume success rate
8. run recovery success rate
9. supervisor prompt token peak

The final metric is especially important because it validates whether multi-agent isolation is actually reducing the main context burden.

## 14. Implementation Sequence

Recommended sequence:

1. create the shared contracts and summary return schema
2. split runtime ownership out of the current orchestration entrypoint
3. add `SupervisorAgent`
4. add one fixed `MonitorAgent`
5. add `IsolationWorker`
6. wire artifact offload and tool reduction for the monitor path
7. add public `delegation.node` projection support
8. add approval resume integration back into the supervisor
9. expand the same pattern to Kubernetes, host, and CI/CD specialists

## 15. Non-Goals

The first multi-agent design does not include:

1. open-ended runtime creation of arbitrary agent classes by the model
2. child-agent direct write execution
3. full exposure of child internal traces in the main UI
4. broad `tool_search` adoption before tool-set growth justifies it
5. deep recursive agent trees

## 16. Final Design Rules

1. one user request maps to one main run
2. the supervisor owns lifecycle, approval, and final answer
3. specialists are fixed and read-only by default
4. the isolation worker exists to protect context, not to become another general agent
5. child agents return summaries, not raw payloads
6. artifacts hold evidence; the main context holds conclusions
7. public projection shows summary nodes, not child internals
8. approval and resume always return through the supervisor

## 17. Current Gaps and Implementation Issues

Based on a review of the existing codebase on 2026-04-16, the following discrepancies and issues were identified:

### 17.1 Architectural Gaps

1.  **Single-Agent Bottleneck**: The current implementation in `internal/modules/ai/logic/logic.go` uses a single scene-based agent (`adk.NewChatModelAgent`). This matches "Option A" in Section 2.1, which this design explicitly identifies as insufficient for handling high-volume operational analysis.
2.  **Missing Directory Structure**: The recommended directory structure under `internal/modules/ai/agent/` (with `orchestrator/`, `specialists/`, and `workers/`) does not exist. The current structure is flatter and lacks clear role boundaries.
3.  **No Isolation Workers**: There is no implementation of the `IsolationWorker` layer. High-volume tools (like Prometheus) currently return raw data directly to the main agent context.

### 17.2 Runtime and Protocol Gaps

1.  **Incomplete State Machine**: The current run status and projection surfaces already cover `running`, `waiting_approval`, and `resuming`, but the runtime does not yet model delegation-specific states, child-task ownership, or resume targets for multi-agent execution. Critical states like `delegating` and `waiting_subagent` are missing.
2.  **Missing Delegation Contracts**: `DelegationTask` and `DelegationSummary` structs are not implemented. There is no typed protocol for inter-agent communication.
3.  **Event Family Gaps**: The current runtime SSE payload layer in `internal/modules/ai/agent/runtime/event_types.go` does not yet model `delegation.*`, `artifact.*`, or `summary.*` families, and the canonical runtime event model for multi-agent execution has not been introduced.

### 17.3 Middleware Gaps

1.  **Context Pollution**: Missing `artifact_offload.go` and `tool_reduction.go` middlewares. Large tool outputs such as `monitor_metric` are not yet reduced or offloaded before downstream consumption, which creates a high risk of polluting agent context.
2.  **Summary Formatting**: No `summary_formatter.go` exists to enforce compact returns from child agents.

### 17.4 Tool Implementation Issues

1.  **Raw Prometheus Output**: The `monitor_metric` tool in `internal/modules/ai/agent/tools/monitor/tools.go` returns raw `MetricPoint` lists (up to 500+ points). In a multi-agent setup, this should be handled by a worker and summarized.
2.  **OutputMode Mismatch**: Some tools (e.g., `monitor_metric`) specify `OutputMode: "summary_plus_artifact"` in their catalog metadata, but the actual tool implementation does not support artifact creation or summarization.

### 17.5 Frontend Gaps

1.  **Missing Projections**: The frontend `web/src/components/AI/historyProjection.ts` and `replyRuntime.ts` do not support the `delegation.node` projection type.
2.  **UI Components**: There are no UI components to render delegation summary nodes or provide links to offloaded artifacts.
