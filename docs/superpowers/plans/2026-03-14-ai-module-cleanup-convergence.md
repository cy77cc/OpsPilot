# AI Module Cleanup Convergence Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove conflicting legacy AI-module code paths so the upcoming `plan-execute-replan-visualization` implementation has a single clear frontend and backend direction.

**Architecture:** Keep one dominant AI interaction model per side. On the frontend, converge toward the visualization-oriented assistant flow and remove legacy thought-chain / approval rendering paths that duplicate or conflict with it. On the backend, converge the SSE/runtime contract toward the proposal-aligned lifecycle and remove obsolete naming and compatibility branches that encode the older event model.

**Tech Stack:** Go backend runtime/orchestrator, React + TypeScript frontend, OpenSpec proposal context

---

## Chunk 1: Cleanup Rules And Scope

### Task 1: Define convergence rules

**Files:**
- Modify: `openspec/changes/plan-execute-replan-visualization/tasks.md`
- Create: `docs/refactor/ai-module-cleanup-convergence.md`

- [ ] Record the cleanup objective and boundaries:
  - remove code that conflicts with `plan-execute-replan-visualization`
  - keep code that is still part of the active AI message/block path
  - skip unrelated cleanup and skip verification
- [ ] List legacy structures targeted for removal or deprecation:
  - old thought-chain stage event names
  - duplicate frontend thought-chain assembly logic
  - duplicate approval panel rendering paths
  - stale barrel exports that keep old entry points alive

## Chunk 2: Frontend Convergence

### Task 2: Remove conflicting legacy AI UI/state paths

**Files:**
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/components/AI/hooks/useAIChat.ts`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/index.ts`
- Modify or Delete: `web/src/components/AI/components/ConfirmationPanel.tsx`
- Modify or Delete: `web/src/components/AI/components/MessageBubble.tsx`
- Modify or Delete: `web/src/components/AI/components/MessageList.tsx`
- Modify or Delete: `web/src/components/AI/thoughtChainMetrics.ts`

- [ ] Identify which frontend path is authoritative for upcoming visualization work
- [ ] Remove duplicated thought-chain-specific helper logic that is no longer the chosen path
- [ ] Remove exports and component usage that preserve obsolete approval/thought-chain rendering
- [ ] Keep shared types only if they still serve the surviving path; otherwise delete or simplify them

## Chunk 3: Backend Convergence

### Task 3: Remove conflicting legacy SSE/runtime paths

**Files:**
- Modify: `internal/ai/events/events.go`
- Modify: `internal/ai/orchestrator.go`
- Modify: `internal/ai/runtime/*.go`
- Modify: `internal/ai/state/chat_store.go`

- [ ] Identify proposal-conflicting event names and compatibility shims
- [ ] Remove or rename obsolete legacy event constants that keep the old stage model alive
- [ ] Simplify orchestrator/runtime flow so emitted semantics do not preserve two competing models
- [ ] Keep persisted structures only if still needed by the surviving frontend/backend contract

## Chunk 4: Integration

### Task 4: Merge results without verification

**Files:**
- Modify: files changed by Tasks 1-3

- [ ] Integrate architecture guidance into frontend/backend changes
- [ ] Resolve overlaps conservatively without reintroducing removed legacy paths
- [ ] Do not run validation commands or claim the code is verified
