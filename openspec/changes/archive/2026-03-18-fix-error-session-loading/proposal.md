## Why

AI chat sessions with error or interrupted status display perpetual loading state instead of showing the terminal error state. This occurs because:
1. Backend leaves message status as `'streaming'` when errors occur during streaming
2. Frontend only recognizes `'done'` as success, treating all other statuses as `'loading'`

This confuses users who see a "loading" indicator on messages that actually failed.

## What Changes

- **Backend**: Update assistant message status to `'error'` when streaming fails or is interrupted
- **Frontend**: Map message statuses properly:
  - `'done'` → `'success'`
  - `'error'` → `'error'`
  - `'interrupted'` → `'abort'`
  - `'streaming'` → `'loading'`

## Capabilities

### New Capabilities

None - this is a bug fix, not a new capability.

### Modified Capabilities

None - no spec-level behavior changes, only fixing incorrect status handling.

## Impact

- `internal/service/ai/logic/logic.go` - Update message status on error
- `web/src/components/AI/CopilotSurface.tsx` - Fix status mapping in `defaultMessages`
