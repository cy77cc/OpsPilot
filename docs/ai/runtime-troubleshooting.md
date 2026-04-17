# AI Runtime Troubleshooting

## Finding Trace IDs

- Chat runs store `trace_id` on `ai_runs`.
- The backend model field lives in `internal/modules/ai/model/run.go`.
- The trace value is generated or normalized in `internal/modules/ai/infra/observability/trace.go`.

## Context Budget Selection

- Budget policy is owned in runtime/app code through `internal/modules/ai/runtime/context/selector.go` and `internal/modules/ai/runtime/context/compressor.go`.
- The app command boundary applies the explicit default budget in `internal/modules/ai/app/command/chat_command_handler.go`.
- The live session-history assembly path consumes that budget in `internal/modules/ai/logic/chat/chat.go`.

## Projection Updates

- Incremental projection state is maintained in `internal/modules/ai/runtime/projection/updater.go`.
- Active projection reads/writes are handled in `internal/modules/ai/logic/chat/projection.go`.
- Terminal projection status updates are finalized in `internal/modules/ai/logic/chat/chat.go`.

## Stream Reconnect And Pending State

- Frontend stream parsing helpers live in `web/src/features/ai/stream/eventDispatcher.ts` and `web/src/features/ai/stream/streamClient.ts`.
- Reconnect behavior is owned by `web/src/features/ai/stream/reconnectController.ts`.
- In-memory pending run state is owned by `web/src/features/ai/state/pendingRunStore.ts`.
