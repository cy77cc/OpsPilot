# Design Spec - AI Assistant Dashboard Metrics and Session Sorting

Implement missing AI assistant metrics on the dashboard and fix the sorting of recent conversations using Eino callbacks.

## 1. Problem Statement
- The AI dashboard's "Recent Conversations" are sorted by creation time instead of the last message time.
- AI metrics (Token count, Session count, Avg duration, Success rate) are currently empty because the underlying `ai_trace_spans` table is not populated.
- There is no centralized mechanism to collect and persist AI interaction metrics across different scenes (host, cluster, etc.).

## 2. Proposed Changes

### 2.1 Dashboard Logic Adjustments
- **File:** `internal/modules/dashboard/logic/logic.go`
- **Change:** In the `getAIActivity` function, update the query for recent AI sessions.
- **Old Sorting:** `.Order("created_at DESC")`
- **New Sorting:** `.Order("updated_at DESC")`
- **Reason:** `updated_at` is updated by `AIChatDAO` whenever a new message (user or assistant) is added to a session, accurately reflecting the latest activity.

### 2.2 Eino Metrics Callback Handler
- **New File:** `internal/modules/ai/logic/metrics/handler.go`
- **Implementation:** Implement `callbacks.Handler` with `OnChatModelStart` and `OnChatModelEnd` hooks.
- **Workflow:**
    1. **OnChatModelStart**:
        - Extract `AIMetadata` (SessionID, RunID, Scene, UserID) from the context using `runtimectx.AIMetadataFrom(ctx)`.
        - Store the start time in the callback state.
    2. **OnChatModelEnd**:
        - Calculate `DurationMS` (End time - Start time).
        - Extract `Usage` (Prompt tokens, Completion tokens) from the `einomodel.ChatModelOutput`.
        - Create a new record in `ai_trace_spans` with status (success/error), tokens, and duration.
        - Create a new record in `ai_usage_logs` using `UsageLogDAO.Create` for detailed auditing.

### 2.3 Global Callback Registration
- **File:** `internal/svc/ai_runtime.go`
- **Change:** Add an `initAIMetricsCallback` function and call it within `initAIRuntime`.
- **Logic:** Use `callbacks.AppendGlobalHandlers(metricsHandler)` to register the new metrics collector globally. This ensures that every AI request through Eino is tracked, regardless of the module or scene.

### 2.4 Data Model Review
- **Tables involved:**
    - `ai_chat_sessions`: Used for "Recent Conversations" (sorting by `updated_at`).
    - `ai_trace_spans`: Used for dashboard statistics (Session count, Tokens, Avg duration, Success rate).
    - `ai_usage_logs`: Used for detailed usage history and cost tracking.

## 3. Architecture & Data Flow
1. **User Request**: User sends a message via AI assistant.
2. **Eino Execution**: Eino starts the ChatModel interaction.
3. **Metrics Trigger (Start)**: `OnChatModelStart` captures metadata and start time.
4. **LLM Response**: LLM returns completion and token usage.
5. **Metrics Trigger (End)**: `OnChatModelEnd` calculates duration, extracts tokens, and persists data to `ai_trace_spans` and `ai_usage_logs`.
6. **Dashboard Query**: `getAIActivity` in `dashboard/logic` queries `ai_trace_spans` to aggregate and display metrics.

## 4. Testing Plan
- **Unit Tests**:
    - Test the `MetricsHandler` to ensure it correctly extracts `AIMetadata` from a mocked context.
    - Test that `OnChatModelEnd` correctly calculates duration and saves the expected data to the database.
- **Manual Verification**:
    - Perform AI chat sessions in different scenes (e.g., host diagnosis, cluster analysis).
    - Check the database to confirm `ai_trace_spans` and `ai_usage_logs` are populated.
    - Refresh the dashboard to verify metrics are displayed and "Recent Conversations" are correctly sorted.
