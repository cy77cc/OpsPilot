# AI Form Assistant Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a first usable AI-assisted form flow for complex text fields, with authenticated SSE streaming, deterministic output cleanup, a 3-second stall hint, and explicit per-field opt-in instead of changing every existing `GuidedFormItem`.

**Architecture:** Reuse the existing authenticated AI streaming shape under `/api/v1/ai/*`. The backend should follow the current `handler/*` plus `interfaces/http` split, register a new SSE endpoint at `POST /api/v1/ai/assist/form/stream`, sanitize `form_context`, build a dedicated lightweight prompt, and normalize emitted text deterministically before forwarding standard `delta`, `done`, and `error` events. The frontend should reuse `consumeAIStream`, add an explicit `aiAssist` config to `GuidedFormItem`, and gate the icon, popover, and stall hint behind both a global feature flag and field-level opt-in.

**Tech Stack:** Go, Gin, CloudWeGo Eino ADK, React, TypeScript, Ant Design, TailwindCSS, Vitest.

---

## Scope Corrections

- The public endpoint must be `POST /api/v1/ai/assist/form/stream`, not `/api/ai/v1/ai/assist/form/stream`.
- Route registration must follow the existing streaming pattern in `internal/bootstrap/modules.go`, not the non-streaming registration in `internal/modules/ai/api/routes.go`.
- Do not rely on a standalone `OutputNormalizationTool` that the agent may never call. Normalize output deterministically in backend code before emitting SSE chunks.
- `GuidedFormItem` must stay backward-compatible. Only fields with an explicit `aiAssist` prop render the AI affordance.
- Phase 1 rollout is limited to the `配置 JSON` field in `web/src/pages/Monitor/ChannelsConfigPage.tsx`. The backend contract stays generic so PromQL/Cron/regex fields can opt in later.

## File Map

- Modify: `api/ai/v1/ai.go`
  - Add form-assist request types shared by the HTTP layer.
- Modify: `internal/bootstrap/modules.go`
  - Register the new authenticated SSE route alongside `/ai/chat`.
- Create: `internal/modules/ai/handler/assist/service.go`
  - Core use case: prompt building, context sanitization, model invocation, chunk emission.
- Create: `internal/modules/ai/handler/assist/prompt.go`
  - Handwritten prompt builder for field-aware assist requests.
- Create: `internal/modules/ai/handler/assist/normalization.go`
  - Deterministic output cleanup helpers.
- Create: `internal/modules/ai/handler/assist/handler.go`
  - Thin adapter exposing the service to the HTTP interface package.
- Create: `internal/modules/ai/handler/assist/service_test.go`
  - Unit tests for normalization, prompt shaping, and context sanitization.
- Create: `internal/modules/ai/interfaces/http/form_assist_handler.go`
  - HTTP binding plus SSE writer integration for the new route.
- Create: `internal/modules/ai/interfaces/http/form_assist_handler_test.go`
  - SSE contract tests for headers, request mapping, and public error behavior.
- Create: `web/src/features/ai/types/formAssist.ts`
  - Shared frontend types for field metadata and opt-in config.
- Create: `web/src/features/ai/api/assistApi.ts`
  - Streaming client for the form-assist endpoint.
- Create: `web/src/api/modules/ai.formAssist.test.ts`
  - Frontend API tests for fetch path, event parsing, and error propagation.
- Modify: `web/src/api/modules/ai.ts`
  - Export the new assist API through `aiApi`.
- Create: `web/src/features/ai/hooks/useFormAssist.ts`
  - Hook for popover state, preview accumulation, feature flag, and the 3-second hint timer.
- Create: `web/src/features/ai/hooks/useFormAssist.test.tsx`
  - Hook tests for timers, reset behavior, apply flow, and disabled mode.
- Modify: `web/src/components/FormGuidance/GuidedFormItem.tsx`
  - Add optional AI affordance without breaking existing guide behavior.
- Modify: `web/src/components/FormGuidance/types.ts`
  - Keep `FieldGuide` and re-export only the shared guidance types used by the component.
- Modify: `web/src/components/FormGuidance/index.ts`
  - Export the new popover component if needed by tests or page integration.
- Create: `web/src/components/FormGuidance/AIFormAssistantPopover.tsx`
  - Presentational popover with prompt input, streamed preview, and apply/cancel actions.
- Create: `web/src/components/FormGuidance/AIFormAssistantPopover.test.tsx`
  - Component tests for preview, buttons, disabled/applying states, and error rendering.
- Modify: `web/src/components/FormGuidance/GuidedFormItem.test.tsx`
  - Preserve all current behavior and add AI opt-in coverage.
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
  - Pilot integration on `配置 JSON`, including a context builder from provider/target fields.
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
  - Page-level integration coverage for the pilot field.

---

### Task 1: Lock the Contract and Route Shape

**Files:**
- Modify: `api/ai/v1/ai.go`
- Modify: `internal/bootstrap/modules.go`
- Create: `internal/modules/ai/interfaces/http/form_assist_handler.go`
- Create: `internal/modules/ai/interfaces/http/form_assist_handler_test.go`

- [ ] **Step 1: Write the failing HTTP contract test**

Create `internal/modules/ai/interfaces/http/form_assist_handler_test.go` with a stub streamer that records the mapped request and emits one `delta` plus one `done` event. Assert all of the following:

- `POST /ai/assist/form/stream` responds with `text/event-stream`.
- The bound request includes `scene`, `user_prompt`, `field_meta`, and `form_context`.
- `uid` from Gin context is forwarded into the internal request.
- The response body contains `event: delta` and `event: done`.
- Internal errors are returned as public `error` SSE events instead of raw JSON envelopes.

- [ ] **Step 2: Run the HTTP contract test and verify it fails**

Run: `go test ./internal/modules/ai/interfaces/http -run FormAssist -v`
Expected: FAIL because `NewFormAssistHandler` does not exist yet.

- [ ] **Step 3: Add the API request types in `api/ai/v1/ai.go`**

Add the request contract used by the handler:

```go
type FieldMeta struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Purpose      string `json:"purpose"`
	Rules        string `json:"rules,omitempty"`
	Placeholder  string `json:"placeholder,omitempty"`
	CurrentValue string `json:"current_value,omitempty"`
}

type FormAssistRequest struct {
	Scene       string         `json:"scene"`
	UserPrompt  string         `json:"user_prompt"`
	FieldMeta   FieldMeta      `json:"field_meta"`
	FormContext map[string]any `json:"form_context"`
}
```

- [ ] **Step 4: Implement the HTTP SSE adapter**

Create `internal/modules/ai/interfaces/http/form_assist_handler.go` with the same structure as the existing chat SSE handler:

- Bind `aiv1.FormAssistRequest`.
- Set `Content-Type`, `Cache-Control`, and `Connection` headers for SSE.
- Use `ssehandler.NewSSEWriter(c.Writer)` for event output.
- Forward the request to a service interface shaped like:

```go
type FormAssistStreamer interface {
	StreamAssist(ctx context.Context, input FormAssistInput, emit logic.EventEmitter) error
}
```

- Emit only `delta`, `done`, and `error` events so the frontend can reuse `consumeAIStream` with no new parser branch.

- [ ] **Step 5: Register the route in the correct place**

Modify `internal/bootstrap/modules.go` so the authenticated AI streaming route helper registers both:

- `POST /api/v1/ai/chat`
- `POST /api/v1/ai/assist/form/stream`

Do not add this route to `internal/modules/ai/api/routes.go`.

- [ ] **Step 6: Run the HTTP contract test and verify it passes**

Run: `go test ./internal/modules/ai/interfaces/http -run FormAssist -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add api/ai/v1/ai.go internal/bootstrap/modules.go internal/modules/ai/interfaces/http/form_assist_handler.go internal/modules/ai/interfaces/http/form_assist_handler_test.go
git commit -m "ai: add form assist SSE contract"
```

---

### Task 2: Build the Backend Assist Service with Deterministic Normalization

**Files:**
- Create: `internal/modules/ai/handler/assist/service.go`
- Create: `internal/modules/ai/handler/assist/prompt.go`
- Create: `internal/modules/ai/handler/assist/normalization.go`
- Create: `internal/modules/ai/handler/assist/handler.go`
- Create: `internal/modules/ai/handler/assist/service_test.go`

- [ ] **Step 1: Write the failing service tests**

Create `internal/modules/ai/handler/assist/service_test.go` with focused cases for:

- `NormalizeFormAssistOutput` removes fenced blocks such as:

````text
Here is the query:
```promql
sum(rate(http_requests_total[5m])) > 0.2
```
````

Expected normalized output:

```text
sum(rate(http_requests_total[5m])) > 0.2
```

- JSON output keeps the JSON body and drops prose like `Here is the JSON you asked for:`.
- `sanitizeFormContext` drops obvious sensitive keys such as `password`, `secret`, `token`, and `api_key`.
- `BuildPrompt` includes `field_meta.label`, `field_meta.purpose`, `field_meta.rules`, and `current_value`.

- [ ] **Step 2: Run the service tests and verify they fail**

Run: `go test ./internal/modules/ai/handler/assist -v`
Expected: FAIL because the package does not exist yet.

- [ ] **Step 3: Implement deterministic normalization and context sanitization**

Create `internal/modules/ai/handler/assist/normalization.go` with helpers like:

```go
func NormalizeFormAssistOutput(raw string) string
func SanitizeFormContext(input map[string]any) map[string]any
```

Implementation rules:

- Trim whitespace.
- If output is fenced, keep only the fenced body.
- Remove one-line lead-ins such as `Here is the query:` or `Here is the JSON:`.
- Never mutate the original `form_context` map in place.

- [ ] **Step 4: Implement prompt building and streaming service**

Create `internal/modules/ai/handler/assist/prompt.go` and `service.go` with:

- An internal input type that includes `UserID`.
- A handwritten system prompt instructing the model to return only the final field value, with no Markdown wrapper and no explanation.
- A service method:

```go
func (s *Service) StreamAssist(ctx context.Context, input FormAssistInput, emit logic.EventEmitter) error
```

Streaming rules:

- Build the prompt from `scene`, `field_meta`, `current_value`, and sanitized `form_context`.
- Invoke the lightweight model path once per request.
- Normalize every emitted chunk before forwarding it as `delta`.
- Emit a terminal `done` event with `status: "completed"` when the stream finishes.
- Map internal failures to public `error` events without leaking raw provider details.

- [ ] **Step 5: Implement the thin handler package adapter**

Create `internal/modules/ai/handler/assist/handler.go` so bootstrap code can construct the service the same way chat/approval handlers do:

```go
type HTTPHandler struct {
	svc *Service
}
```

This package should stay thin and delegate the actual SSE HTTP work to `internal/modules/ai/interfaces/http/form_assist_handler.go`.

- [ ] **Step 6: Run the service tests and verify they pass**

Run: `go test ./internal/modules/ai/handler/assist -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/modules/ai/handler/assist/service.go internal/modules/ai/handler/assist/prompt.go internal/modules/ai/handler/assist/normalization.go internal/modules/ai/handler/assist/handler.go internal/modules/ai/handler/assist/service_test.go
git commit -m "ai: implement form assist service"
```

---

### Task 3: Add the Frontend Streaming Client

**Files:**
- Create: `web/src/features/ai/types/formAssist.ts`
- Create: `web/src/features/ai/api/assistApi.ts`
- Modify: `web/src/api/modules/ai.ts`
- Create: `web/src/api/modules/ai.formAssist.test.ts`

- [ ] **Step 1: Write the failing frontend API test**

Create `web/src/api/modules/ai.formAssist.test.ts` that:

- Mocks `fetch` with a `ReadableStream`.
- Calls `aiApi.formAssistStream(...)`.
- Asserts the request path is `/ai/assist/form/stream`.
- Asserts the body contains `scene`, `user_prompt`, `field_meta`, and `form_context`.
- Verifies streamed `delta`, `done`, and `error` events are surfaced through handlers.

- [ ] **Step 2: Run the frontend API test and verify it fails**

Run: `cd web && npm run test:run -- src/api/modules/ai.formAssist.test.ts`
Expected: FAIL because `formAssistStream` does not exist yet.

- [ ] **Step 3: Add shared frontend form-assist types**

Create `web/src/features/ai/types/formAssist.ts` with explicit opt-in types:

```ts
export type FormAssistFieldMeta = {
  key: string;
  label: string;
  purpose: string;
  rules?: string;
  placeholder?: string;
  currentValue?: string;
};

export type FormAssistConfig = {
  scene: string;
  fieldMeta: FormAssistFieldMeta;
  getFormContext?: () => Record<string, unknown>;
  disabled?: boolean;
};
```

- [ ] **Step 4: Implement `assistApi.ts` using the existing stream parser**

Create `web/src/features/ai/api/assistApi.ts` and:

- Reuse `consumeAIStream`.
- Reuse the same auth headers pattern as `chatApi.ts`.
- Use `POST ${base}/ai/assist/form/stream`.
- Do not create a second SSE parser.

- [ ] **Step 5: Export the API through `aiApi`**

Modify `web/src/api/modules/ai.ts` so `aiApi.formAssistStream` is available to hooks and tests.

- [ ] **Step 6: Run the frontend API test and verify it passes**

Run: `cd web && npm run test:run -- src/api/modules/ai.formAssist.test.ts`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add web/src/features/ai/types/formAssist.ts web/src/features/ai/api/assistApi.ts web/src/api/modules/ai.ts web/src/api/modules/ai.formAssist.test.ts
git commit -m "web: add form assist streaming api"
```

---

### Task 4: Implement the `useFormAssist` Hook

**Files:**
- Create: `web/src/features/ai/hooks/useFormAssist.ts`
- Create: `web/src/features/ai/hooks/useFormAssist.test.tsx`

- [ ] **Step 1: Write the failing hook tests**

Create `web/src/features/ai/hooks/useFormAssist.test.tsx` with fake timers and assert:

- The stall hint appears only after 3 seconds of inactivity.
- The hint does not appear when the feature flag is off.
- `preview` accumulates streamed `delta` chunks in order.
- `applySuggestion()` calls the provided `onApply` callback with the final preview text.
- `cancel()` clears prompt text, preview text, error state, and open state.

- [ ] **Step 2: Run the hook tests and verify they fail**

Run: `cd web && npm run test:run -- src/features/ai/hooks/useFormAssist.test.tsx`
Expected: FAIL because `useFormAssist` does not exist yet.

- [ ] **Step 3: Implement the hook**

Create `web/src/features/ai/hooks/useFormAssist.ts` with these responsibilities:

- Read the global feature flag from `localStorage` key `ai-form-assist-enabled`.
- Track `isOpen`, `isStreaming`, `prompt`, `preview`, `error`, and `showHint`.
- Start a 3-second timer only when:
  - the feature is enabled,
  - the field is opted in,
  - the field currently has a non-empty value,
  - the popover is closed,
  - no stream request is in progress.
- Use `aiApi.formAssistStream(...)` to populate `preview`.
- Expose `open`, `close`, `submit`, `applySuggestion`, and `dismissHint`.

- [ ] **Step 4: Run the hook tests and verify they pass**

Run: `cd web && npm run test:run -- src/features/ai/hooks/useFormAssist.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/features/ai/hooks/useFormAssist.ts web/src/features/ai/hooks/useFormAssist.test.tsx
git commit -m "web: add useFormAssist hook"
```

---

### Task 5: Add Opt-In AI UI to `GuidedFormItem`

**Files:**
- Modify: `web/src/components/FormGuidance/types.ts`
- Modify: `web/src/components/FormGuidance/GuidedFormItem.tsx`
- Modify: `web/src/components/FormGuidance/GuidedFormItem.test.tsx`
- Modify: `web/src/components/FormGuidance/index.ts`
- Create: `web/src/components/FormGuidance/AIFormAssistantPopover.tsx`
- Create: `web/src/components/FormGuidance/AIFormAssistantPopover.test.tsx`

- [ ] **Step 1: Extend the failing component tests**

Update `web/src/components/FormGuidance/GuidedFormItem.test.tsx` to add coverage for:

- No AI icon when `aiAssist` is undefined.
- AI icon appears when `aiAssist` is provided and the feature flag is enabled.
- Clicking the AI icon opens the popover without breaking existing focus-driven guide-card behavior.
- Existing focus/blur handlers still fire.

Create `web/src/components/FormGuidance/AIFormAssistantPopover.test.tsx` to cover:

- Prompt textarea rendering.
- Stream preview rendering.
- Apply button disabled when preview is empty.
- Error message rendering.

- [ ] **Step 2: Run the component tests and verify they fail**

Run: `cd web && npm run test:run -- src/components/FormGuidance/GuidedFormItem.test.tsx src/components/FormGuidance/AIFormAssistantPopover.test.tsx`
Expected: FAIL because the AI prop and popover component do not exist yet.

- [ ] **Step 3: Add the opt-in prop and keep backward compatibility**

Modify `web/src/components/FormGuidance/GuidedFormItem.tsx` so it accepts:

```ts
aiAssist?: FormAssistConfig;
```

Rules:

- If `aiAssist` is undefined, render exactly as today.
- Keep the existing guide-card logic unchanged.
- Render the AI affordance only when the feature flag is enabled and the field is opted in.
- Do not inject the AI affordance into unrelated pages automatically.

- [ ] **Step 4: Implement the popover**

Create `web/src/components/FormGuidance/AIFormAssistantPopover.tsx` with:

- A natural-language textarea.
- A streamed preview region.
- `取消` and `采纳建议` actions.
- Inline error text and a loading state.

- [ ] **Step 5: Run the component tests and verify they pass**

Run: `cd web && npm run test:run -- src/components/FormGuidance/GuidedFormItem.test.tsx src/components/FormGuidance/AIFormAssistantPopover.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/components/FormGuidance/types.ts web/src/components/FormGuidance/GuidedFormItem.tsx web/src/components/FormGuidance/GuidedFormItem.test.tsx web/src/components/FormGuidance/index.ts web/src/components/FormGuidance/AIFormAssistantPopover.tsx web/src/components/FormGuidance/AIFormAssistantPopover.test.tsx
git commit -m "web: add opt-in ai form guidance UI"
```

---

### Task 6: Pilot the Feature on Monitoring Channel JSON Config

**Files:**
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`

- [ ] **Step 1: Write the failing page integration test**

Extend `web/src/pages/Monitor/ChannelsConfigPage.test.tsx` with a new test that:

- Enables `localStorage.setItem('ai-form-assist-enabled', '1')`.
- Opens the `新增渠道` modal.
- Verifies the AI affordance appears on `配置 JSON`.
- Mocks `aiApi.formAssistStream` to emit streamed JSON text.
- Clicks `采纳建议` and asserts the generated JSON is written back into the `配置 JSON` field.

- [ ] **Step 2: Run the page test and verify it fails**

Run: `cd web && npm run test:run -- src/pages/Monitor/ChannelsConfigPage.test.tsx`
Expected: FAIL because the page does not pass an `aiAssist` config yet.

- [ ] **Step 3: Wire the pilot field with explicit metadata and context**

Modify `web/src/pages/Monitor/ChannelsConfigPage.tsx` so the `配置 JSON` field gets:

- `scene: "monitoring"`
- `fieldMeta.key: "config_json"`
- `fieldMeta.label: "配置 JSON"`
- `fieldMeta.purpose: "Generate valid channel configuration JSON"`
- `fieldMeta.rules: "Return valid JSON only. No markdown fences. No explanation."`
- `getFormContext`: include current `provider` and `target`

Do not wire `Provider` or `目标地址` into the AI flow in this phase.

- [ ] **Step 4: Run the page test and verify it passes**

Run: `cd web && npm run test:run -- src/pages/Monitor/ChannelsConfigPage.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Monitor/ChannelsConfigPage.tsx web/src/pages/Monitor/ChannelsConfigPage.test.tsx
git commit -m "web: pilot ai form assist on channel config json"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run focused backend tests**

Run: `go test ./internal/modules/ai/interfaces/http ./internal/modules/ai/handler/assist -v`
Expected: PASS

- [ ] **Step 2: Run focused frontend tests**

Run: `cd web && npm run test:run -- src/api/modules/ai.formAssist.test.ts src/features/ai/hooks/useFormAssist.test.tsx src/components/FormGuidance/GuidedFormItem.test.tsx src/components/FormGuidance/AIFormAssistantPopover.test.tsx src/pages/Monitor/ChannelsConfigPage.test.tsx`
Expected: PASS

- [ ] **Step 3: Run broader module verification**

Run: `go test ./internal/modules/ai/...`
Expected: PASS

- [ ] **Step 4: Manually verify the pilot UI flow**

Manual checklist:

1. Enable the feature flag in DevTools with `localStorage.setItem('ai-form-assist-enabled', '1')`.
2. Open `监控 -> 渠道配置 -> 新增渠道`.
3. Confirm the AI affordance appears on `配置 JSON` but not on unrelated fields.
4. Enter `provider=email`, set a target address, open the popover, and ask for a small email JSON template.
5. Confirm preview text streams into the popover and `采纳建议` fills the `配置 JSON` field.

- [ ] **Step 5: Manually verify normalization edge cases**

Use a prompt or mocked response that returns fenced output and confirm the preview/result contains only the cleaned body:

- Fenced JSON:

````text
```json
{"from":"ops@example.com"}
```
````

- Fenced PromQL:

````text
```promql
sum(rate(http_requests_total[5m])) > 0.2
```
````

Expected: the stored preview/result excludes the code fences and any leading prose.
