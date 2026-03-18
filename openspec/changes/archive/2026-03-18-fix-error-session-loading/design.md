## Context

The AI chat system stores message status in `ai_chat_messages.status` column. During streaming, the status is set to `'streaming'` and updated to `'done'` on success. However, when errors occur during streaming, the status is not updated, leaving it as `'streaming'` indefinitely.

The frontend `CopilotSurface.tsx` loads historical messages and maps status:
```typescript
status: (message.status === 'done' ? 'success' : 'loading')
```

This binary mapping treats all non-'done' statuses as loading, including error states.

## Goals / Non-Goals

**Goals:**
- Fix backend to set message status to `'error'` when streaming fails
- Fix frontend to properly map terminal error states

**Non-Goals:**
- Adding new message statuses (use existing `'error'`)
- Changing the error UI/UX design

## Decisions

### 1. Backend: Update message status on error

**Location:** `internal/service/ai/logic/logic.go`

When streaming fails (in the `for` loop that processes events), update the assistant message status to `'error'` instead of leaving it as `'streaming'`.

The error handling is already present (lines 189-206), but it only updates the run status to `'failed'`. We need to also update the message status.

**Rationale:** The message record should reflect its terminal state for accurate history display.

### 2. Frontend: Proper status mapping

**Location:** `web/src/components/AI/CopilotSurface.tsx:349-352`

Replace the binary mapping with proper status mapping:

| Backend Status | Bubble Status | Visual |
|----------------|---------------|--------|
| `'done'` | `'success'` | Normal completed message |
| `'error'` | `'error'` | Error styling |
| `'interrupted'` | `'abort'` | Cancelled styling |
| `'streaming'` | `'loading'` | Loading indicator |
| Other | `'loading'` | Default to loading |

**Rationale:** Aligns with `@ant-design/x` `MessageStatus` enum which supports these states.

## Risks / Trade-offs

- **Risk:** Existing error sessions with `'streaming'` status won't be automatically fixed
  - Mitigation: Accept as historical data; new errors will be handled correctly
