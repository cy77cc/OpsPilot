# AI Release Gate

## Purpose

Use this runbook before shipping AI handler changes that affect routing, approval handling, or replay behavior.

## Gate Checklist

- [ ] Contract check passes: `go test ./internal/service/ai/handler -run RouteContract -v`
- [ ] Route list matches `docs/superpowers/contracts/ai-contract-v1.md`
- [ ] Approval reject path verified end to end
- [ ] Approval timeout path verified end to end
- [ ] Reconnect replay verified with `Last-Event-ID`
- [ ] Historical replay verified against stored event history
- [ ] Any contract drift is documented and approved before release

## Required Verification Scenarios

### Approval Reject

- [ ] Submit a reject decision for a pending approval.
- [ ] Confirm the approval transitions to rejected and cannot be reprocessed.
- [ ] Confirm follow-up resume or retry behavior respects the final decision.

### Approval Timeout

- [ ] Leave a pending approval open until the timeout threshold is reached.
- [ ] Confirm the approval is marked expired or timed out.
- [ ] Confirm no duplicate decision event is emitted after timeout.

### Reconnect Replay

- [ ] Start a chat stream and record the last delivered SSE event ID.
- [ ] Disconnect and reconnect with `Last-Event-ID`.
- [ ] Confirm replay resumes from the next event without duplicate delivery.
- [ ] Confirm cursor expiration returns the documented contract error.

### Historical Replay

- [ ] Reconstruct an older session or run from persisted history.
- [ ] Confirm the replayed projection matches the stored event sequence.
- [ ] Confirm historical replay remains stable after additive schema changes.

## Release Notes

- [ ] Capture the exact test command output for the release record.
- [ ] Record any contract changes that require downstream consumer updates.
- [ ] Escalate any breaking change for contract version review before shipping.

