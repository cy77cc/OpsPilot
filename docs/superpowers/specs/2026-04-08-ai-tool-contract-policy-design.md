# AI Tool Contract and Policy Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Inherits: `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- Inherits: `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- Reference: `docs/superpowers/specs/2026-03-28-tool-inline-approval-agentflow-design.md`
- Scope: tool registry scope, tool contract schema, use-when and dont-use-when rules, evidence contract rules, invocation pipeline, policy decision model, approval decision inputs, failure and deny semantics
- Goal: define the contract-first tool boundary and the deterministic policy layer that the runtime kernel consumes

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary, the event-first truth model, the reusable-data / disposable-glue posture, and the object ownership rules defined by the parent blueprint, the core domain model, and the runtime kernel.

If this document conflicts with the parent blueprint or the upstream runtime/domain specs on object ownership, lifecycle state, approval resume, or naming, the upstream documents win.

Explicit terminology rule:

- approval resume is in-run continuation
- same-run adjustment should use `refinement` or `reassessment`
- `replan` is reserved for newly created run attempts, not same-run behavior

This document does not redefine `Run`, `Turn`, `ToolCall`, or `Approval`. It defines the tool contract and policy boundary that the runtime kernel uses when it creates and transitions those objects.

## 2. Tool Registry Scope

The tool registry is the authoritative inventory of structured capabilities available to the runtime.

It owns:

- canonical tool identities
- tool contract versions
- input and output schema references
- side-effect classification
- risk metadata
- evidence requirements
- dry-run support flags
- timeout and retry hints
- approval posture hints
- pack membership metadata

It does not own:

- run lifecycle state
- turn sequencing
- approval decisions
- runtime policy outcomes
- UI rendering state

### 2.1 Global Registry Versus Domain Tool Packs

The global registry is the source of truth for all tools. Domain tool packs are curated subsets or groupings of the global registry for a specific domain, such as host, service, Kubernetes, deployment, monitoring, or governance.

Domain tool packs may:

- group tools by domain
- provide domain-local descriptions and examples
- narrow the recommended `use-when` and `dont-use-when` guidance for their own tool set
- attach domain-specific evidence expectations that are stricter than the global minimum

Domain tool packs may not:

- define global approval policy
- override global deny rules
- bypass required evidence
- weaken side-effect restrictions
- claim ownership of the policy engine

The global policy engine remains the final authority. Domain agents may propose a tool from their pack, but they do not own the policy decision for that tool.

### 2.2 Versioning Rule

Tool contracts are versioned. A materially changed contract gets a new version instead of silently mutating the old one.

Because compatibility is not the priority for this project, consumers should resolve the latest approved contract version unless a replay, fixture, or evaluation case pins an older version for reproducibility.

## 3. Tool Contract Schema

Each tool contract should be structured enough for the runtime kernel, policy engine, replay system, and evaluation harness to inspect without reading prompt text.

The exact storage format can vary, but the logical schema must cover the following field groups.

| Field group | Required contents |
| --- | --- |
| Identity | `tool_id`, canonical tool name, contract version, owning pack or domain, status |
| Capability summary | human-readable purpose, short description, supported operation classes |
| Schema | input schema, output schema, required fields, optional fields, validation rules |
| Side-effect metadata | side-effect level, idempotency behavior, retry behavior, dry-run support |
| Policy metadata | risk level, default approval posture, deny conditions, freshness constraints |
| Evidence metadata | required evidence kinds, result summary shape, artifact references, preview rules |
| Operational metadata | timeout, concurrency hints, rate limits, execution mode constraints |
| Trace metadata | correlation keys, causality keys, audit labels |

### 3.1 Contract Requirements

Every tool contract must make the following explicit:

1. What the tool does.
2. What inputs it accepts.
3. What outputs it may return.
4. What side effects it can produce.
5. What evidence the runtime must persist.
6. Whether a dry-run or preview mode is supported.
7. Whether approval is ever required before execution.
8. Whether the tool may be retried safely.
9. Whether the tool is safe to call more than once with the same logical intent.

### 3.2 Contract Non-Goals

The tool contract is not a prompt template.
The tool contract is not a policy decision.
The tool contract is not a runtime turn record.
The tool contract is not a human approval snapshot.

## 4. Use-When and Dont-Use-When Rules

`use-when` and `dont-use-when` guidance is part of the contract. It helps the runtime and domain agents reason about coarse capability fit before policy evaluation.

The router may use coarse domain metadata from these rules to decide whether a request belongs to a domain or should remain conversational, but it does not select concrete tools.

### 4.1 Use-When

`use-when` should describe the positive conditions that make a tool a good fit.

It should cover:

- the task shape or intent
- the domain context
- the kind of evidence the tool can produce
- the user-visible outcome the tool can support
- any prerequisites that must already be true

### 4.2 Dont-Use-When

`dont-use-when` should describe the counter-scenarios where the tool should not be selected.

It should cover:

- contradictory task shapes
- unsupported environments
- unsafe operating states
- cases where a safer or cheaper capability exists
- cases where the tool would produce misleading evidence

### 4.3 Selection Rules

The runtime and domain agents should apply the guidance as follows:

1. If a `dont-use-when` condition matches, the tool should not be selected.
2. If no `use-when` condition matches, the tool should not be selected unless the user explicitly requests it and policy still permits it.
3. A `use-when` match is necessary but not sufficient. Policy still decides whether execution is allowed.
4. `use-when` and `dont-use-when` never override hard policy rules.

### 4.4 Domain Pack Usage

Domain packs may tighten `use-when` and `dont-use-when` for their own tools.

They may not:

- loosen a global `dont-use-when`
- convert a global deny into a local allow
- bypass approval by relabeling the use case
- redefine evidence expectations for the global registry

That keeps domain reasoning local while policy remains global.

## 5. Evidence Contract Rules

The evidence contract defines what must be persisted so a tool action can be replayed, audited, and evaluated.

Evidence must be sufficient to answer:

- what was asked of the tool
- what contract version was used
- what policy decision was made
- what the tool returned
- what side effects were expected
- what side effects were observed
- whether the runtime executed the tool in preview mode or real mode

### 5.1 Required Evidence Elements

Each tool call must be able to produce a durable evidence bundle containing:

- tool identity and contract version
- normalized input snapshot
- argument preview
- result preview or failure summary
- policy decision and matched rule ids
- approval linkage if approval was required
- timestamps and correlation ids
- execution mode, including dry-run if applicable

### 5.2 Evidence Storage Rules

1. Large input and output payloads should live in content blobs or similar durable artifacts.
2. The persisted `ToolCall` should reference those blobs instead of copying them.
3. The approval snapshot should reference the frozen evidence bundle, not transient prompt text.
4. The runtime must never depend on assistant prose as the only record of tool evidence.

### 5.3 Evidence Quality Rules

Evidence is invalid if it cannot explain the decision path.

If the contract cannot produce the required evidence, the tool should not execute.

If a tool runs in dry-run mode, the evidence must clearly mark the call as preview-only and must not blur preview output with side-effect-bearing output.

## 6. Invocation Pipeline

The invocation pipeline is the deterministic path from a selected tool candidate to a persisted `ToolCall` result.

### 6.1 Pipeline Stages

1. Select a candidate tool from the registry or a domain pack.
2. Normalize the request into the tool contract schema.
3. Create or update the `ToolCall` record with the planned invocation snapshot.
4. Evaluate policy using the runtime context and contract metadata.
5. Apply the policy decision.
6. If allowed, execute the tool and persist evidence.
7. If approval is required, freeze the approval snapshot and checkpoint, then pause the run.
8. If denied, persist the denial and continue with a safe alternate path or stop.
9. Emit canonical events for the decision and outcome.

### 6.2 Relationship Between `ToolCall`, Policy, Approval, and Runtime

The objects relate to each other in one direction:

- the runtime creates a `ToolCall`
- policy evaluates that planned `ToolCall`
- policy may require an `Approval`
- the `Approval` snapshot freezes the tool request and its evidence context
- the runtime kernel transitions the `Run` and `Turn` based on the policy outcome

The runtime must not invent approval state without a persisted `ToolCall`.
The approval snapshot must not be rewritten after it has been persisted.
Approval resume is in-run continuation of the same interrupted attempt, not a new run.

### 6.3 Invocation Boundaries

The pipeline should execute only one bounded action per decision round.

If a tool result changes the plan, the runtime should treat that as same-run refinement or reassessment and open a new turn if needed.

If the objective or attempt identity changes materially, the system should create a new run instead of mutating the old `ToolCall` or approval record into a different attempt.

## 7. Policy Decision Model

Policy is the deterministic safety boundary of the system.

It should be inspectable, replayable, and testable without relying on the model.

Policy consumes:

- tool contract metadata
- tool arguments
- side-effect classification
- dry-run support
- freshness and idempotency constraints
- run and turn state
- task and domain context
- user or operator identity when relevant
- approval posture hints
- runtime mode and environment constraints

The runtime may carry an alternate-path hint for its own continuation logic after policy returns a result, but policy must not use alternate-path availability to choose among outcomes.

Policy returns one decision plus structured reasons, matched rules, and traceable evidence.

### 7.1 Policy Decision Table

| Decision | Meaning | Runtime behavior | Persisted facts |
| --- | --- | --- | --- |
| `allow` | The action is safe enough to execute as planned | invoke the tool in normal mode and continue the same run | policy allow record, `ToolCall` execution record, evidence bundle |
| `deny` | The action is forbidden in the current context | do not invoke the tool; emit a denial and continue with a safe alternate path or fail the run if none exists | policy denial record, blocked tool snapshot, denial reason |
| `dry_run_only` | The tool may be inspected or previewed, but not used for the restricted side effect | invoke the tool only in preview mode and mark the result as non-side-effecting | policy dry-run record, preview evidence, no-side-effect marker |
| `require_approval` | A human gate is required before the action can proceed | freeze the approval snapshot, persist the checkpoint, and pause the run | approval snapshot, checkpoint id, resume target binding |

A single policy evaluation returns exactly one of these four outcomes. `dry_run_only` and `require_approval` are mutually exclusive within one evaluation.

### 7.2 Decision Semantics

`allow` means the tool may execute immediately under the current contract.

`deny` means the tool must not execute. The runtime may continue only if a safe alternate path exists.

`dry_run_only` means the tool may execute only in preview mode, and the runtime must not treat the preview as a real side effect.

`require_approval` means the runtime must persist the approval snapshot and enter `waiting_approval`.

If preview data is needed before a human gate, the runtime uses a staged flow:

1. First evaluation returns `dry_run_only`.
2. The runtime executes the preview and persists the preview evidence.
3. A later evaluation for the real side-effecting action may return `require_approval`.

That keeps preview generation separate from the approval decision for the real action.

### 7.3 Decision Precedence

Policy should resolve in this order:

1. Hard deny conditions
2. Validation and schema failures, which map to `deny` with a `contract_violation` reason
3. Preview-only restrictions
4. Human approval requirements
5. Normal allow

That order prevents approval from being used as a workaround for a contract or safety violation.

## 8. Approval Decision Inputs

When policy requires approval, the approval reviewer must see the frozen operation, not a mutable draft.

The approval decision input set should include:

- `tool_call_id`
- tool name and contract version
- frozen argument preview
- side-effect and risk summary
- matched policy rule ids
- reason approval is required
- expected outcome or impact
- dry-run preview if one exists
- resume target binding
- checkpoint id
- task and run summary
- safe alternate path summary if one exists, provided by the runtime as post-policy context only

### 8.1 Required Inputs

The minimum approval snapshot must include:

1. The exact tool identity.
2. The exact arguments that triggered the gate.
3. The reason the action needs approval.
4. The checkpoint that will be resumed if approved.
5. The target binding for the interrupted call.

### 8.2 Optional Inputs

Optional approval context may include:

- brief task objective summary
- preview output from a dry-run
- operator notes
- retry history
- alternate path suggestions

Optional data is helpful, but it must not replace the required frozen snapshot.

### 8.3 Approval Integrity Rules

1. The approval snapshot is write-once for the current run and tool call.
2. Later evidence may be appended, but the original approval request must remain intact.
3. Approval resume is in-run continuation only.
4. A materially fresh attempt requires a new run and a new approval record.

## 9. Failure and Deny Semantics

`deny` is a policy outcome. It is not the same thing as a tool failure.

### 9.1 Deny Semantics

When policy returns `deny`:

- the tool must not execute
- the runtime must emit a denial event
- the `ToolCall` should be marked blocked or skipped, not succeeded
- the runtime may continue with a safe alternate path if one exists
- if no safe alternate path exists, the run should fail with the appropriate policy-related failure category

### 9.2 Tool Failure Semantics

When policy allows execution but the tool fails:

- the `ToolCall` failed after execution began
- the failure is a tool failure, not a policy deny
- the runtime may retry inside the same run only if the contract and current state allow it
- a retry must create a new `ToolCall`

### 9.3 Approval Failure Semantics

If approval is rejected, expired, or cannot be resumed safely:

- the current run should stop
- the failure should be recorded with the appropriate approval-related category
- approval resume must not silently fall back to a synthetic checkpoint

### 9.4 Dry-Run Failure Semantics

If a tool only supports preview mode and the preview fails:

- the failure should be recorded as a preview or tool failure
- it must not be treated as an unauthorized side effect
- the runtime should not upgrade the call into a real execution unless policy explicitly re-evaluates the action

### 9.5 Stop Versus Continue

The runtime should continue in the same run when:

- a denial still leaves a safe alternate path
- a tool failure is recoverable and the contract permits retry
- a same-run refinement or reassessment is needed after new evidence

The runtime should stop the current run when:

- there is no safe alternate path after denial
- a tool failure is unrecoverable
- approval is rejected or expires
- approval checkpoint validation fails

## 10. Current System Reuse Guidance

The tool and policy boundary should reuse durable assets that already behave like durable truth.

Likely reusable:

- existing tool metadata and tool-call correlation ids
- current tool argument and result content blobs
- existing approval preview payload shapes
- dry-run or preview support that already exists in specific integrations
- existing timeout, retry, and idempotency patterns
- current result summaries that can be normalized into evidence bundles

Likely disposable:

- tool allowlists encoded only in prompt text
- scene-specific tool routing glue
- UI-only approval state that does not exist in durable records
- duplicated tool result previews that act as the only source of truth
- policy logic hidden inside service orchestration code
- ad hoc contract fragments that cannot be inspected by the runtime or replay system

The practical rule is to keep durable tool metadata and evidence artifacts if they still fit the target shape, while deleting control-flow glue that only exists to support the old runtime structure.

## 11. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- `docs/superpowers/specs/2026-03-28-tool-inline-approval-agentflow-design.md`
- `docs/superpowers/specs/2026-04-08-ai-module-redesign-design.md`
