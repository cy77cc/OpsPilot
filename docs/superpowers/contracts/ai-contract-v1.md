# AI Contract v1

## Purpose

This document defines the AI HTTP and streaming contract used by the handler layer. It is the release reference for route shape, stream replay behavior, approval submission semantics, and response field stability.

## Scope

- HTTP route registration under `/api/v1/ai`
- SSE event delivery for chat and replay flows
- Approval submission and replay safety
- Compatibility checks used by CI and release gating

## Verification

- Primary contract check: `go test ./internal/service/ai/handler -run RouteContract -v`
- Route coverage is enforced by the handler route contract tests.
- Any route, payload, or replay behavior change must be reflected in this document before release.

## Compatibility Matrix

| Surface | Contract Version | Backward Compatible | Forward Compatible | Notes |
| --- | --- | --- | --- | --- |
| AI HTTP routes | v1 | Yes, while paths, verbs, and auth requirements stay unchanged | Yes, if new routes are additive | Route registration is the canonical API surface. Removal or renaming is breaking and requires a contract bump. |
| Chat SSE events | v1 | Yes, if existing event names and payload keys remain available | Yes, if new fields are additive | `event_id` must remain available for replay and reconnect handling. |
| Approval submit response | v1 | Yes, if existing business codes and payload shape remain stable | Yes, if new optional fields are additive | Contract consumers rely on explicit not found, forbidden, and conflict-style outcomes. |
| Reconnect replay semantics | v1 | Yes, if `Last-Event-ID` resumes from the same event boundary | Yes, if replay stays monotonic and deduplicated | Reconnect must not drop already-committed events or replay duplicates. |
| Historical replay semantics | v1 | Yes, if older event history still reconstructs the same projection | Yes, if new projection fields are additive | Historical replays must remain deterministic for audits and incident review. |

## Stability Rules

1. Additive changes are preferred.
2. Removing or renaming a route, event, or response field is breaking unless the contract version is bumped.
3. Replay behavior must preserve event ordering and cursor semantics.
4. CI must fail when the route contract test fails.
5. Release gates must be updated when a contract change affects approval, reconnect, or historical replay behavior.

