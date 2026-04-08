# AI Copilot UI Interaction Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Inherits:
  - `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- References:
  - `web/src/components/AI/CopilotSurface.tsx`
  - `web/src/components/AI/AssistantReply.tsx`
  - `web/src/components/AI/ToolReference.tsx`
- Scope: frontend interaction design for Copilot conversation mode, operation mode, run detail, replay, approval, reconnect, and tail behavior
- Goal: define the UI contract for rendering derived AI projection blocks without making the frontend the runtime source of truth

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary, the event-first truth model, the durable object model, the runtime checkpoint and approval semantics, and the projection-first UI contract defined by the parent blueprint and upstream detailed designs.

If this document conflicts with the blueprint, the core domain model, the runtime kernel, the tool and policy design, or the event and projection design on object ownership, lifecycle state, or truth ownership, the upstream documents win.

Hard boundary:

- the UI consumes derived projection blocks
- the UI may cache and rehydrate read models
- the UI may tail a run through SSE or replay paging
- the UI is not the runtime truth owner
- the UI must not infer canonical state from assistant prose or transient component state

Explicit terminology rule:

- `conversation mode` is collaboration-first
- `operation mode` is execution-first
- `resume` continues the same run attempt
- `replay` is a derived inspection view, not a live runtime

## 2. UI Purpose

The AI Copilot surface is the user-facing shell for one AI module. It must support two different user intents without splitting into separate product areas:

1. Conversation mode
2. Operation mode

The same shell can present both, but the interaction model must make the difference visible.

### 2.1 Conversation Mode

Conversation mode is the lightweight collaboration path.

It should optimize for:

- quick questions
- explanation and clarification
- short planning
- low-friction follow-up from a previous answer

It should present:

- message-first layout
- minimal operational chrome
- compact assistant summaries
- no forced task-board framing unless the projection already contains operational blocks

It should not force:

- approval cards
- run diagnostics
- replay affordances
- tool detail expansion unless a tool block exists

### 2.2 Operation Mode

Operation mode is the execution path.

It should optimize for:

- task-oriented work
- tool-driven execution
- approval gates
- resume and reconnect continuity
- replayable inspection

It should present:

- the current run identity and status
- plan, tool_call, result, and status blocks
- approval cards when a tool requires human decision
- progress and failure state when execution pauses or fails
- reconnect and tail cues when a live run is still in flight

Operation mode should make the run visible without exposing raw runtime internals.

## 3. Surface Inventory

The UI should treat the AI experience as three related surfaces.

| Surface | Primary user question | Primary content | Secondary content |
| --- | --- | --- | --- |
| Copilot surface | What is happening right now? | Current conversation or operation thread | Session switcher, composer, scroll/tail controls |
| Run detail view | What exactly happened in this run? | Derived projection blocks in canonical order | Summary, approval state, tool outcomes, failure details |
| Replay view | How did this run unfold? | Block-by-block timeline and hydrated content | Cursor, block jump list, evidence links, resume checkpoints |

Surface rule:

- all three surfaces render the same derived facts from different presentation density
- the run detail and replay views should not add new truth that is absent from the projection

## 4. Derived Projection Contract

The frontend must read derived blocks from projection data that already reflects canonical event ordering and replay rules.

The UI may:

- load projections by run id
- page through projection blocks
- hydrate content blobs referenced by block items
- keep local render caches for continuity and performance
- reconnect to a pending run when the canonical persisted `event_id` is available

The UI must not:

- reconstruct canonical runtime state from message text
- mutate projection blocks as if they were authoritative records
- replace a missing projection with inferred runtime truth

Reconnect cursor rule:

- the reconnect/tail cursor is the canonical persisted `event_id`
- if the cursor is unknown, expired, or cannot be validated, reconnect must fail explicitly
- the UI must not guess a replacement cursor or synthesize continuity from surrounding content

### 4.1 Base Block Envelope

Every rendered block should be treated as a derived read model with at least these common fields.

| Field | Required | Purpose |
| --- | --- | --- |
| `id` | yes | stable block identity inside a projection |
| `run_id` | yes | parent run identity |
| `type` | yes | render family |
| `seq` or block order | yes | canonical display order |
| `status` | yes | visible block state |
| `title` | yes | compact label for navigation and headers |
| `source_event_ids` | yes | provenance back to canonical events |
| `summary` | no | short human-readable synopsis |
| `content_refs` | no | blob ids for hydrated content |
| `updated_at` | no | freshness marker for tail/reconnect |

Rule:

- the block envelope is derived state only
- the UI may cache it, sort it, or collapse it
- the UI must not treat it as the source of truth for runtime behavior

## 5. Block Inventory

The Copilot UI should render the following derived block families.

| Block type | Required fields | Optional fields | Render intent |
| --- | --- | --- | --- |
| `message` | `id`, `run_id`, `seq`, `role`, `content`, `status`, `source_event_ids` | `summary`, `content_refs`, `created_at`, `updated_at`, `author_label` | show user and assistant conversation turns |
| `plan` | `id`, `run_id`, `seq`, `title`, `steps`, `active_step_index`, `status`, `source_event_ids` | `summary`, `route_label`, `created_at`, `updated_at` | show the current execution plan and progress |
| `tool_call` | `id`, `run_id`, `seq`, `tool_name`, `arguments_preview`, `result_status`, `status`, `source_event_ids` | `call_id`, `policy_label`, `risk_level`, `result_preview`, `result_content_ref`, `created_at`, `updated_at` | show one structured tool invocation attempt |
| `approval` | `id`, `run_id`, `seq`, `approval_id`, `tool_call_id`, `operation_summary`, `risk_level`, `decision_state`, `approval_reason`, `expected_impact`, `checkpoint_id`, `resume_binding`, `status`, `source_event_ids` | `preview_fields`, `preview_json_ref`, `expires_at`, `timeout_seconds`, `resume_state`, `created_at`, `updated_at` | show the human decision gate for a frozen operation snapshot |
| `result` | `id`, `run_id`, `seq`, `outcome_status`, `summary`, `evidence_refs`, `status`, `source_event_ids` | `tool_call_id`, `approval_id`, `content_ref`, `created_at`, `updated_at` | show the final or intermediate outcome of an execution step |
| `error` | `id`, `run_id`, `seq`, `error_code`, `message`, `recoverable`, `status`, `source_event_ids` | `details`, `retry_hint`, `checkpoint_id`, `created_at`, `updated_at` | show a user-visible failure or interruption |
| `status` | `id`, `run_id`, `seq`, `run_status`, `label`, `status`, `source_event_ids` | `summary`, `approval_id`, `checkpoint_id`, `created_at`, `updated_at` | show the first-class visible run state, including `waiting_approval` |

### 5.1 Message Block

Message blocks are the primary conversation unit.

They should show:

- role
- content
- light metadata such as completion or streaming state

They should not expose raw runtime internals.

Source boundary:

- assistant message blocks come from the run projection
- user turns come from the session or Copilot transcript layer unless and until canonical user-input events are introduced
- the UI should not imply that user and assistant turns are emitted from the same run-projection message source

### 5.2 Plan Block

Plan blocks are the visible shape of a run's intended path.

They should show:

- ordered steps
- current active step
- completed and pending status

They should support collapse and expansion, but the step list must stay stable once derived from the projection.

### 5.3 Tool Call Block

Tool call blocks are the visible record of structured work.

They should show:

- tool name
- short argument summary
- current result state
- a result preview when available

The tool call block is an evidence view, not a control surface.

### 5.4 Approval Block

Approval blocks are the visible human gate for a tool action.

They should show:

- the operation being requested
- risk or approval context
- the approval state
- the checkpoint or resume linkage when present

Approval blocks may expose action controls only when the derived state is still pending and the backend contract allows submission.

### 5.5 Result Block

Result blocks are the user-visible summary of a step completion.

They should show:

- success or failure outcome
- concise explanation
- evidence or result references

### 5.6 Error Block

Error blocks are the user-visible representation of interruption or failure.

They should show:

- a short error label
- an error code when available
- whether recovery or retry is possible

## 6. Interaction Model

### 6.1 Conversation Flow

Conversation mode should feel like a compact chat surface.

Interaction rules:

1. The user starts with a question or short request.
2. The Copilot surface renders message blocks and a composer.
3. If the projection later introduces plan or tool blocks, the surface can upgrade in place into operation presentation.
4. The UI should not force a separate task creation step unless the router and runtime already classified the request as operational.

### 6.2 Operation Flow

Operation mode should feel like a working run.

Interaction rules:

1. The user submits an operational request.
2. The UI shows the run as pending, then active.
3. Plan and tool blocks appear in order as the projection advances.
4. If a tool requires approval, the approval block becomes the primary action point.
5. If the run resumes, the same run view continues rather than opening a new visual thread.

### 6.3 Approval Action Flow

The approval card should use a narrow, explicit interaction contract.

Visible structure:

- header with tool label and approval state
- short operation summary
- compact key/value preview
- optional raw JSON disclosure
- approve and reject actions

Action behavior:

1. Pending state shows both actions.
2. Submitting state disables repeat submission and shows progress.
3. Success updates the card state and keeps the block in the same run timeline.
4. Conflict or already-processed responses should refresh the canonical approval state from the backend.
5. Refresh failure should move the card into a readonly recovery state.

UX rule:

- the card should guide the user to decide on the operation snapshot, not on an abstract approval token

Approval completeness rule:

- every approval block must include the reason approval is required
- every approval block must include an expected impact or equivalent impact summary
- every approval block must include the resume binding and checkpoint context used to continue the same run
- every approval block reflects a frozen approval snapshot and must not be reconstructed from live mutable runtime state

### 6.4 Resume Updates

Resume updates should make the continuation obvious.

When approval is granted:

- the approval card should switch to a resuming or approved-resuming presentation
- the run header should show that the same run is continuing
- the tail should remain attached to the latest persisted `event_id` cursor
- new blocks should append in the same visual sequence

If a resume later fails retryably:

- the surface should keep the failed resume state visible
- any retry must start a new run attempt
- the failed run remains immutable evidence
- the user should see a clear retry affordance only for creating that new attempt

### 6.5 Reconnect And Tail Behavior

Reconnect and tail behavior should preserve continuity when the user navigates away, reloads, or returns to an in-flight run.

UX requirements:

1. If the last known run is still reconnectable, the surface should reopen on that run rather than starting from a blank composer.
2. If the projection is incomplete, the surface should show a tailing or catching-up state until the latest blocks arrive.
3. If the backend reports a resumable run, the UI should reflect the pending state even if the live socket was interrupted.
4. If the UI cannot recover the projection summary, it should show a bounded recovery skeleton instead of inventing content.
5. If the persisted cursor is missing, expired, or invalid, the reconnect attempt must fail explicitly and surface that failure instead of guessing a nearby cursor.

The UI may retain the previous visible assistant body while hydration catches up, but it must replace that body only when the derived projection supplies the authoritative content.

## 7. State Design

The surface should support five visible runtime states.

### 7.1 Empty State

Empty state applies when the user has no active conversation or when the current session has no meaningful messages yet.

It should show:

- a clear prompt to start asking
- lightweight prompt chips or examples
- no error framing

It should not show:

- run detail chrome
- approval controls
- replay controls

### 7.2 Running State

Running state applies when the projection shows an active run.

It should show:

- active assistant response
- live tail or streaming cue
- current plan step or in-progress tool call
- scroll-to-bottom affordance when the user has scrolled away

### 7.3 Failed State

Failed state applies when the run ends in error, interruption, or unrecoverable projection failure.

It should show:

- concise failure summary
- error block or terminal banner
- relevant next-step hint when recovery is possible

It should not hide the prior run content.

### 7.4 Waiting Approval State

Waiting approval state applies when execution is paused on a frozen approval snapshot and the user has not yet decided.

It should show:

- a visible `waiting_approval` label or badge
- the approval card as the primary action surface
- the frozen operation summary and checkpoint context

It should make clear that the run is paused, not failed.

### 7.5 Resuming State

Resuming state applies when approval has been granted and the same run is restoring from checkpoint.

It should show:

- resume banner or badge
- frozen approval snapshot if still useful for context
- live tail attached to the continuing run

Resuming state should feel like continuation, not retry from scratch.

## 8. Run Detail And Replay View

Run detail and replay views are projection-heavy inspection surfaces.

### 8.1 Run Detail View

Run detail view should answer:

- what was this run trying to do?
- what blocks were produced?
- where did approval appear?
- how did the run finish?

It should show:

- run header
- block timeline
- approval and result cards
- failure summary when applicable

### 8.2 Replay View

Replay view should answer:

- what happened in order?
- what event or block caused each change?
- where did the run branch or wait?

It should show:

- canonical ordering
- cursor or step navigation
- hydration loading where content is fetched lazily
- per-block provenance cues when available

Replay view should prioritize traceability over compactness.

## 9. Approval Card Design

Approval cards should be visually distinct from ordinary tool blocks.

Design requirements:

1. A stronger border or background treatment than standard tool cards.
2. A clear risk label or approval-state label.
3. A short operation preview that reads like a human decision, not raw JSON.
4. An expandable raw payload section for auditability.
5. Approve and reject controls placed together and disabled while submission is pending.
6. A readonly conflict or refresh-needed state when the canonical approval state changed elsewhere.

Content guidance:

- show the operation the user is approving
- show the reason approval is required
- show timeout or expiry context when available
- do not bury the primary decision controls below unrelated metadata

Approval cards should stay in the timeline as evidence even after the decision is made.

## 10. Empty, Running, Failed, Waiting_Approval, And Resuming States

The same surface can move through these states during one user session.

### 10.1 Empty To Running

The surface should transition from onboarding or empty-state guidance to the active run without changing route or forcing a hard reset.

### 10.2 Running To Failed

If a run fails, the last successful blocks should remain visible and the failure should be appended as the terminal state.

### 10.3 Running To Resuming

If an approval gate is accepted, the current run should remain visible while the resume state becomes active.

### 10.4 Failed To New Attempt

If the user starts a new attempt after a terminal failure, the UI should present it as a new run under the same session, not as a mutated old run.

## 11. Reuse And Disposal Guidance

The current frontend AI components should be reused selectively.

### 11.1 Reuse

- `web/src/components/AI/CopilotSurface.tsx`
  - reuse as the shell for drawer, session switching, tailing, and layout orchestration
  - keep it focused on presentation flow, not canonical state derivation

- `web/src/components/AI/AssistantReply.tsx`
  - reuse as the renderer for derived message, plan, summary, and hydrated markdown content
  - keep formatting local and avoid moving runtime ownership into the component

- `web/src/components/AI/ToolReference.tsx`
  - reuse as the compact inline approval and tool reference affordance
  - keep its job limited to rendering and local action submission, not truth ownership

### 11.2 Dispose Or Replace

The following patterns should not be preserved as design dependencies:

- scene-driven UI assumptions that only exist to bias prompt assembly
- any UI state that treats assistant prose as the source of runtime truth
- any duplicated run-status derivation that is not backed by projection data
- any approval display that hides the structured operation snapshot behind a generic label
- any reconnect behavior that cannot restore from a run id and last known event id

If a current component only survives by encoding one of those patterns, it should be simplified or replaced during implementation.

## 12. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
