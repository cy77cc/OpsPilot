# AI Error Codes

## Stream Codes

`AI_STREAM_INTERNAL`
- Meaning: generic runtime failure while serving the chat stream.
- Retryable: yes.
- Notes: internal error details are hidden from the client.

`AI_STREAM_CURSOR_EXPIRED`
- Meaning: the supplied `last_event_id` is too old or no longer available for replay.
- Retryable: no with the same cursor; the client should refresh run/session state first.

## API Codes

Current rebuilt AI surface primarily emits stream-oriented public errors for `/api/v1/ai/chat`.
Additional `AI_API_*` codes should be documented here as they are added to the rebuilt backend surface.

## Trace IDs

Trace IDs are exposed on run payloads as `trace_id`.
They are generated in `internal/modules/ai/infra/observability/trace.go` and threaded through the chat command boundary into the run record.
