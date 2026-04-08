# AI Evaluation Harness Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Inherits:
  - `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md`
- References:
  - `script/ai/check_contract_parity.sh`
  - `script/ai/validate_multi_domain_rollout.sh`
- Scope: evaluation case schema, judge contracts, replay-driven validation, dataset organization, regression strategy, and reuse guidance for the AI harness
- Goal: define a reproducible evaluation harness that judges canonical runtime behavior from persisted AI artifacts instead of inferring truth from UI transcript state

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary, the event-first truth model, the reuse-and-deletion posture, and the evaluation role defined by the parent blueprint and sibling detailed specs.

If this document conflicts with the blueprint, the core domain model, the runtime kernel, or the event and projection design on object ownership, canonical ordering, approval semantics, replay semantics, or naming, the upstream documents win.

Hard boundary:

- evaluation consumes canonical event and projection artifacts
- evaluation may consume persisted content blobs, diagnosis reports, and trace-derived read models
- evaluation must not invent truth from UI transcript text, screenshot text, or stream fragments that are not anchored to canonical artifacts
- transcript judges may explain and score a path only after the path is reconstructed from canonical artifacts
- canonical events and durable referenced artifacts are the only sources that can establish truth
- derived views are convenience overlays for inspection and scoring, not alternate truth sources

Explicit terminology:

- `Outcome Judge` means a judge that checks whether the run or replay reached the expected result
- `Transcript Judge` means a judge that checks whether the path was reasonable and consistent with the canonical facts
- `Replay-Driven Validation` means validation that starts from canonical events and derived projections, not from human-readable prose
- `Pass@k` means a development metric where at least one of `k` attempts must succeed
- `Pass^k` means a regression metric where all `k` attempts must succeed

## 2. Harness Goals

The harness exists to answer a small set of product questions:

1. Did the runtime choose the correct route, domain, and execution path for this case?
2. Did the runtime select the right tool family, approval behavior, and resume behavior?
3. Did the runtime reach the expected durable outcome?
4. Can the same result be reproduced from canonical replay inputs?
5. Did a change preserve the regression cases that matter for rollout confidence?

The harness is not meant to be a general-purpose model benchmark.

Non-goals:

1. It does not infer ground truth from UI transcript prose when canonical artifacts disagree or are missing.
2. It does not create a second source of truth for route, tool, approval, or outcome semantics.
3. It does not replace runtime, projection, or gateway tests.
4. It does not require a new framework or library.
5. It does not treat ad hoc transcript readability as equivalent to canonical correctness.

The harness should favor reproducibility over cleverness:

- prefer stable fixtures over live mutable state
- prefer canonical event cursors over derived UI cursors
- prefer explicit tolerance rules over implicit human judgment
- prefer narrow, explainable failure modes over vague scoring

## 3. EvaluationCase Schema

`EvaluationCase` is the stable regression scenario consumed by the harness.

The core domain model owns the case object; the harness owns how the case is executed, sampled, and judged.

### 3.1 Core Field Groups

| Field group | Required contents |
| --- | --- |
| Identity | `case_id`, name, description, tags, version, suite name |
| Scenario | user input, task intent, expected domain, expected route, expected operation shape |
| Fixture references | seed session id, seed task id, seed run id, canonical event ids, projection ids, content ids, approval ids when applicable |
| Assertions | expected route, expected tool family, expected approval behavior, expected outcome, expected transcript properties, allowed variance |
| Judge config | judge set, scoring mode, pass threshold, sample count, seed policy, deterministic versus tolerant execution mode |
| Provenance | origin run id, origin report id, origin bug or review link, origin author or source, capture timestamp |
| Lifecycle metadata | created_at, updated_at, superseded_by, archived_at |

### 3.2 Case Rules

1. A case should be versioned rather than mutated when the intended behavior changes.
2. A case may point to real canonical run artifacts, but it must not own live runtime truth.
3. A case should be replayable without depending on the current UI state.
4. A case should state whether a tolerance is intentional or accidental.
5. A case should pin any fixture version that matters for reproducibility.

### 3.3 Concrete Case Template

```yaml
case_id: host-diagnosis-basic
name: Diagnose sustained CPU pressure on a host
version: 1
suite: core-smoke
tags:
  - host
  - routing
  - replay
  - approval
scenario:
  user_input: "检查主机 CPU 持续过高的原因"
  expected_domain: host
  expected_route: operation
  expected_operation_shape: investigate
fixtures:
  seed_session_id: session-host-diagnosis-basic
  seed_run_id: run-host-diagnosis-basic
  canonical_event_ids:
    - evt-001
    - evt-002
    - evt-003
  projection_ids:
    - projection-host-diagnosis-basic
  content_ids:
    - content-host-diagnosis-basic-tool-args
assertions:
  expected_tool_family: host_observe
  approval_expected: false
  expected_outcome:
    summary_present: true
    evidence_list_non_empty: true
    terminal_state: completed
judge_config:
  scoring_mode: pass_fail
  sample_count: 1
  pass_threshold: 1.0
  deterministic: true
provenance:
  origin_run_id: run-host-diagnosis-basic
  origin_report_id: report-host-diagnosis-basic
```

The template is illustrative. Implementations may add fields, but they should not remove the field groups that drive replay and regression reproducibility.

## 4. Outcome Judges

Outcome judges decide whether the system reached the expected durable result.

They answer questions such as:

- Did the run complete, fail, or cancel in the expected way?
- Did the final answer or action match the case expectation?
- Did the approval flow resolve the way the case required?
- Did the replay reconstruct the same canonical outcome?

### 4.0 Judge Taxonomy And Aggregation

The harness uses three judge families that align with the upstream evaluation semantics:

- `Route Judge` -> `route_evaluated`
- `Tool Judge` -> `tool_choice_evaluated`
- `Outcome Judge` -> `outcome_evaluated`

Rule:

1. Route and tool judges are first-class checks, not implicit notes hidden inside outcome scoring.
2. Outcome judging is blocking by default.
3. Transcript judging is advisory by default.
4. A case may opt into transcript gating when transcript consistency is part of the acceptance bar.
5. The final verdict is composed from route, tool, and outcome judges first, then any transcript result.
6. Route, tool, and outcome judges cannot be skipped in a case that declares them.

Composition rule:

| Judge family | Default behavior | Can block final verdict? | Typical purpose |
| --- | --- | --- | --- |
| Route Judge | blocking | yes | verify route/domain selection and route rationale |
| Tool Judge | blocking | yes | verify tool family selection and tool-choice rationale |
| Outcome Judge | blocking | yes | verify durable result and terminal state |
| Transcript Judge | advisory unless opted in | only when case enables transcript gating | verify path readability and canonical consistency |

If any blocking judge fails, the case fails.

If all blocking judges pass and transcript gating is disabled, the case may still pass even when the transcript judge only advises a warning.

If transcript gating is enabled for a case, transcript failure becomes blocking for that case.

### 4.1 Outcome Judge Input Contract

| Input field group | Required contents |
| --- | --- |
| Case identity | `case_id`, version, suite name |
| Canonical facts | canonical event ids or cursor, durable content ids, diagnosis report ids when relevant, approval ids when relevant |
| Derived views | replay blocks, summarized projections, UI-friendly renderings derived from canonical facts |
| Expected assertions | terminal state, outcome shape, approval expectation, allowed variance |
| Execution metadata | sample id, seed, replay cursor, judge version |
| Evidence limits | explicitly listed artifacts that may be consulted |

Contract rules:

1. `canonical_facts` are the only inputs that can establish truth.
2. `derived_views` are convenience overlays for inspection, explanation, and scoring only.
3. If a derived view conflicts with a canonical fact, the canonical fact wins.
4. A judge may read a derived view to save effort, but it must be able to trace every verdict back to canonical facts.

### 4.2 Outcome Judge Output Contract

| Output field | Meaning |
| --- | --- |
| `verdict` | `pass`, `fail`, or `inconclusive` |
| `score` | normalized confidence or match score when the harness needs it |
| `matched_assertions` | assertions that were satisfied |
| `failed_assertions` | assertions that failed |
| `failure_category` | stable category for the failure |
| `evidence_refs` | canonical ids or cursor references that supported the verdict |
| `notes` | optional human-readable explanation |

Outcome judges should remain narrow:

1. They should check the canonical result first.
2. They should not infer missing facts from prose.
3. They should not convert a weak transcript into a successful outcome.
4. They should fail closed when the canonical artifacts are incomplete.

### 4.3 Outcome Failure Categories

| Category | Meaning |
| --- | --- |
| `missing_canonical_artifact` | the expected event, projection, content, or report artifact is absent |
| `projection_incomplete` | the replay or derived projection cannot be reconstructed enough to judge the case |
| `route_mismatch` | the effective route differs from the expected route |
| `domain_mismatch` | the runtime chose the wrong domain or capability family |
| `tool_family_mismatch` | the selected tool family differs from the case expectation |
| `approval_mismatch` | approval was requested, skipped, rejected, or resumed differently than expected |
| `outcome_mismatch` | the final durable result does not match the expected outcome |
| `nondeterministic_variance_out_of_bounds` | observed variation exceeds the case tolerance |
| `capture_drift` | the captured canonical artifacts no longer describe the intended regression |
| `judge_internal_error` | the judge could not complete its own evaluation reliably |

### 4.4 Route Judge Contract

The route judge covers route selection and route rationale.

It maps directly to `route_evaluated`.

#### Route Judge Input Contract

| Input field group | Required contents |
| --- | --- |
| Canonical facts | canonical route events, canonical event ids or cursor, durable route-linked artifacts |
| Derived views | route summary blocks, replay blocks, inspection-friendly route renderings |
| Expected assertions | expected route, expected domain, expected route rationale, allowed route variance |
| Execution metadata | sample id, seed, judge version |

#### Route Judge Output Contract

| Output field | Meaning |
| --- | --- |
| `verdict` | `pass`, `fail`, or `inconclusive` |
| `route_match` | whether the effective route matched the case expectation |
| `domain_match` | whether the selected domain matched the case expectation |
| `rationale_match` | whether the route rationale was consistent with canonical facts |
| `failure_category` | stable route failure category |
| `evidence_refs` | canonical ids or cursor references used in the judgment |
| `notes` | optional explanation |

Route failure categories use the same namespace as the outcome judge unless a more specific route category is needed.

Route Judge negative cases should primarily assert `expected_runtime_failure_category` when the route failure is a runtime behavior defect.

### 4.5 Tool Judge Contract

The tool judge covers tool-family selection and tool-choice rationale.

It maps directly to `tool_choice_evaluated`.

#### Tool Judge Input Contract

| Input field group | Required contents |
| --- | --- |
| Canonical facts | canonical tool-selection events, canonical event ids or cursor, durable tool-linked artifacts |
| Derived views | tool summary blocks, replay blocks, inspection-friendly tool renderings |
| Expected assertions | expected tool family, expected tool selection reason, allowed tool variance |
| Execution metadata | sample id, seed, judge version |

#### Tool Judge Output Contract

| Output field | Meaning |
| --- | --- |
| `verdict` | `pass`, `fail`, or `inconclusive` |
| `tool_family_match` | whether the selected tool family matched the case expectation |
| `selection_match` | whether the selected tool choice matched the case expectation |
| `rationale_match` | whether the tool-choice rationale was consistent with canonical facts |
| `failure_category` | stable tool failure category |
| `evidence_refs` | canonical ids or cursor references used in the judgment |
| `notes` | optional explanation |

Tool failure categories use the same namespace as the outcome judge unless a more specific tool category is needed.

Tool Judge negative cases should primarily assert `expected_runtime_failure_category` when the tool failure is a runtime behavior defect.

## 5. Transcript Judges

Transcript judges decide whether the execution path was reasonable and internally consistent.

They are secondary to outcome judges.

Transcript judges may score:

- route explanation quality
- tool selection rationale
- approval necessity and timing
- whether the assistant path followed the canonical facts
- whether the transcript changed in a way that is consistent with the replay

Transcript judges must not do this:

- treat UI prose as truth when canonical artifacts disagree
- invent a route, tool, or approval fact from a polished message
- accept a transcript that cannot be anchored to replayable canonical artifacts

### 5.1 Transcript Judge Input Contract

| Input field group | Required contents |
| --- | --- |
| Canonical facts | canonical events, content ids, approval facts, replay cursors, durable report artifacts |
| Derived views | projection blocks, replay-derived transcript, narrative blocks derived from canonical facts |
| Case expectations | expected path, expected rationale, tolerated wording differences |
| Comparison metadata | judge version, seed, sample id, tolerated variance rules |

Contract rules:

1. `canonical_facts` remain authoritative even when a derived transcript reads better.
2. `derived_views` are allowed to improve readability and scoring ergonomics.
3. A transcript judge may only explain facts that can be traced back to canonical facts.
4. If transcript gating is disabled, transcript evidence can warn but must not block by itself.

### 5.2 Transcript Judge Output Contract

| Output field | Meaning |
| --- | --- |
| `verdict` | `pass`, `fail`, or `inconclusive` |
| `score` | path-quality or consistency score when needed |
| `matched_path_signals` | route, tool, approval, and rationale signals that matched |
| `failed_path_signals` | path signals that did not match |
| `failure_category` | stable transcript failure category |
| `evidence_refs` | canonical ids or replay block ids used in the judgment |
| `notes` | optional explanation |

Transcript judge negative cases should primarily assert `expected_judge_failure_category` when the failure is about transcript inconsistency or incomplete derivation.

### 5.3 Transcript Failure Categories

| Category | Meaning |
| --- | --- |
| `transcript_only_claim` | the transcript asserted a fact not supported by canonical artifacts |
| `path_inconsistent_with_canonical` | the transcript path conflicts with replayable canonical facts |
| `reasoning_gap` | the path is mechanically correct but lacks the expected rationale signal |
| `approval_explanation_missing` | approval was required but the transcript failed to explain the gate clearly enough |
| `unsupported_transcript_shape` | the transcript cannot be reconstructed in a reliable shape from the available canonical data |
| `judge_internal_error` | the judge could not complete its own evaluation reliably |

### 5.4 Verdict Aggregation And Failure Namespaces

Each case may declare two distinct expected failure namespaces for negative scenarios:

- `expected_runtime_failure_category`
- `expected_judge_failure_category`

Semantics:

1. `expected_runtime_failure_category` is used when the case expects the runtime itself to stop, fail, reject, or otherwise produce a runtime-level negative outcome.
2. `expected_judge_failure_category` is used when the case expects the harness or a judge family to fail a check even if the runtime reached a normal terminal result.
3. Route and tool judges assert runtime failure categories when the observed route or tool behavior is the thing under test.
4. Outcome judges assert runtime failure categories when the terminal runtime outcome is the thing under test.
5. Transcript judges assert judge failure categories when the discrepancy is about transcript consistency, derivation quality, or unsupported narration.
6. A negative case may specify both namespaces when it needs to prove both the runtime behavior and the judge interpretation.

The harness should report both namespaces separately so that a route or tool regression does not get collapsed into a generic transcript issue, and a transcript-quality problem does not get mistaken for a runtime failure.

## 6. Replay-Driven Validation

Replay-driven validation is the primary evaluation mode.

The harness should validate cases in this order:

1. Load the canonical case definition.
2. Load the canonical event stream and durable content artifacts referenced by the case.
3. Rebuild the replay or projection view from persisted canonical data.
4. Run the route judge against the canonical artifacts.
5. Run the tool judge against the canonical artifacts.
6. Run the outcome judge against the canonical artifacts.
7. Run the transcript judge only on transcript material reconstructed from the same canonical artifacts.
8. Record the verdict, evidence refs, and failure categories for each judge family.

Boundary rules:

1. Replay must never source truth from the UI transcript alone.
2. A transcript judge may explain a path, but it must not create missing route or tool facts.
3. If canonical replay cannot be reconstructed, the harness should fail the case instead of guessing.
4. If a UI transcript and canonical event stream disagree, the canonical event stream wins.
5. If a case needs a derived summary, the summary must be regenerated from canonical data for that run.

Replay-driven validation is especially important for:

- approval and resume flows
- route and domain selection
- tool family choice
- terminal state correctness
- regression captures from bug reports

## 7. Pass@k Versus Pass^k Usage

The blueprint distinguishes exploratory success from regression confidence. The harness should keep that distinction.

| Metric | Use it for | Interpretation |
| --- | --- | --- |
| `Pass@k` | development exploration, prompt tuning, route experiments, and broad capability search | success if at least one of `k` samples passes |
| `Pass^k` | regression gating, rollout confidence, approval-sensitive scenarios, and canonical replay stability | success only if all `k` samples pass |

Rules:

1. Use `Pass@k` when the goal is to discover a workable path.
2. Use `Pass^k` when the goal is to prove the behavior is stable enough to ship or roll forward.
3. Prefer `Pass^k` for approval, resume, and replay cases because those are correctness-sensitive.
4. A case with meaningful nondeterminism should document the tolerated variance explicitly rather than weakening the regression rule.
5. If a change is meant to reduce variance, move the case from exploratory to regression status only after the result is reproducible.

## 8. Regression Suite Strategy

The harness should organize datasets into four top-level groups.

| Suite | Purpose | Typical update rule | Typical metric |
| --- | --- | --- | --- |
| Core smoke | smallest set of canonical cases that prove the harness and runtime still work | run on every AI change | `Pass^k` with a small `k` |
| Domain suites | domain-specific cases such as host, cluster, governance, or other operational packs | run when a domain pack or route rule changes | mostly `Pass^k`, sometimes `Pass@k` during development |
| Approval and resume suites | cases that exercise approval request, reject, expire, retry, and resume paths | run when approval, checkpoint, or resume logic changes | `Pass^k` |
| Regression captures | cases extracted from bugs, incidents, or release regressions | run until the fix is proven durable and the capture is promoted | `Pass^k` |

Dataset organization rules:

1. Core smoke should stay small enough to run frequently.
2. Domain suites should be grouped by the same domain packs used by the runtime and tool registry.
3. Approval and resume cases should include both success and failure transitions.
4. Regression captures should preserve the original canonical artifacts that exposed the bug whenever possible.
5. A captured regression should include the origin run id, origin report id, and the event cursor needed to replay it.

Recommended capture fields:

- source run id
- source event cursor
- source projection cursor if relevant
- source diagnosis report id
- canonical content ids
- expected terminal state
- expected_runtime_failure_category when the case is negative and the runtime behavior is the thing under test
- expected_judge_failure_category when the case is negative and the judge interpretation is the thing under test
- owner or source link

Regression suite maintenance rules:

1. Add new bugs as captures before or with the fix.
2. Promote a fixed capture into core smoke only when it represents a stable invariant.
3. Retire captures only when their underlying behavior is intentionally replaced by a new contract.
4. Keep approval and replay captures on the smallest possible canonical fixture set.
5. Negative captures should populate the runtime namespace, the judge namespace, or both, depending on which failure the case is intended to prove.

## 9. Reuse And Disposal Guidance

The harness should reuse existing repo assets before inventing new ones.

### 9.1 Reuse

- `script/ai/check_contract_parity.sh`
  - reuse as a preflight check for cases that depend on public AI route parity
  - use it to catch contract drift before running API-backed evaluation captures
  - do not treat it as a judge; it checks contract shape, not AI correctness

- `script/ai/validate_multi_domain_rollout.sh`
  - reuse as a broader validation wrapper when collecting a release candidate or confirming multi-domain readiness
  - use it as a repo-level smoke gate, not as an evaluation data source
  - do not fold its build and validation steps into judge logic

- Canonical AI artifacts
  - event streams, run projections, replay blocks, approval snapshots, run contents, and diagnosis reports are the primary evaluation inputs
  - trace spans and persisted content blobs are suitable evidence attachments when a case needs them

- Existing AI test coverage
  - current tests such as `internal/service/ai/logic/logic_test.go`, `internal/service/ai/logic/run_tailer_test.go`, and `internal/service/ai/handler/chat_test.go` are good sources for regression capture material
  - use them as capture seeds, not as a substitute for serialized cases

### 9.2 Dispose Or Avoid

- ad hoc UI transcript comparisons that are not backed by canonical artifacts
- screenshot-only judgments for route, tool, approval, or outcome correctness
- prompt-fixture assertions that never get serialized into a case
- duplicate evaluation logic in frontend state or transport code
- any judge that silently repairs missing canonical facts by guessing from prose

The reuse rule is simple:

- if the artifact can be replayed from canonical state, reuse it
- if the artifact only exists because of the UI surface, do not treat it as truth

## 10. Placeholder Scan

Scan scope:

- `docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`

Result:

- no unfinished draft markers are present in this spec
- no stub text is introduced by this design

## 11. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- `docs/superpowers/specs/2026-04-08-ai-gateway-api-contract-design.md`
