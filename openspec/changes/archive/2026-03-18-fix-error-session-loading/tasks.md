## 1. Backend Fix

- [x] 1.1 Update `internal/service/ai/logic/logic.go` to set message status to `'error'` when streaming fails
- [x] 1.2 Handle both error cases: `event.Err` and stream `Recv()` errors

## 2. Frontend Fix

- [x] 2.1 Update `web/src/components/AI/CopilotSurface.tsx` `defaultMessages` function to properly map status
- [x] 2.2 Add helper function `mapHistoryMessageStatus` for status mapping

## 3. Testing

- [x] 3.1 Test error session loads with correct error status
- [x] 3.2 Verify successful sessions still display correctly

## 4. Archive

- [x] 4.1 Archive change after implementation
