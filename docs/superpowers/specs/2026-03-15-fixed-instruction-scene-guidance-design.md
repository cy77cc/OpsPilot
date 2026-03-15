# Fixed Instruction And Scene Guidance Design

## Summary

This change replaces dynamic system-prompt rendering with a fixed agent instruction and moves runtime scene context into the user-facing input envelope. The goal is to make the agent's behavior stable while still biasing tool selection toward the current scene.

The scene is treated as a prioritization signal, not a capability boundary. The agent should prefer tools related to the active scene, but it must remain free to cross domains when the user's request or newly discovered evidence requires it.

## Problem

The previous design made `BuildInstruction` responsible for turning `RuntimeContext` into a dynamic system prompt. That created three problems:

1. Prompt quality depended on sparse frontend display fields such as `SceneName` and `ProjectName`.
2. Prompt behavior became tightly coupled to unstable runtime payload shape.
3. The agent could become harder to reason about because stable behavior rules and per-request context were mixed together.

This was already visible in cases where the runtime context had a meaningful `Scene` but empty display fields, causing the system prompt to degrade to generic values.

## Goals

- Keep the agent's system behavior stable across requests.
- Let the agent understand the platform's tool domains and execution rules globally.
- Bias the agent toward scene-relevant tools without hard-restricting tool access.
- Preserve cross-domain reasoning when the task requires it.
- Keep runtime context injection simple, short, and deterministic.

## Non-Goals

- Building a hard scene-to-tool allowlist.
- Encoding the full runtime context object into the prompt.
- Using dynamic system prompt generation for each request.
- Making scene act as a permission or policy boundary.

## Design

### Fixed Instruction

The agent uses a single fixed instruction string. This instruction describes:

- The agent's role as the OpsPilot infrastructure operations assistant.
- The main tool domains available in the system.
- The rule that scene influences initial tool preference, not final tool scope.
- The requirement to prefer read-first investigation before mutating actions.
- The existing approval and governance expectations for mutating tools.

The fixed instruction should not depend on `RuntimeContext` values. It should remain stable so the agent consistently learns the same operating rules.

### Runtime Context In User Input

Per-request runtime context is prepended to the user's message as a short structured envelope.

Recommended format:

```text
[Runtime Context]
scene: deployment:hosts
project: 1
page: /deployment/infrastructure/hosts
selected_resources: none

[User Request]
帮我检查这批主机状态
```

This envelope is part of the effective user input, not part of the system prompt. The raw user request remains intact under a separate section.

Only high-value fields should be injected:

- `scene`
- `project`
- `page`
- `selected_resources`

Low-value or noisy fields such as the full `UserContext` and arbitrary metadata maps should not be dumped into the prompt.

### Scene Guidance Model

The scene is used as an initial routing hint:

- The agent first determines the likely primary tool domain from user intent.
- If a `scene` is present, the agent prefers tool domains related to that scene for the first investigation steps.
- If the request clearly exceeds the scene domain, or if scene-relevant tools do not provide enough evidence, the agent may expand to other domains.

Example domain preference rules:

- `deployment:*` prefers `deployment`, then `host`, then `service` and `kubernetes`
- `service:*` prefers `service`, then `deployment`, then `kubernetes`
- `host:*` prefers `host`, then `deployment`, then `monitor`
- `k8s:*` prefers `kubernetes`, then `service`, then `deployment`

These are guidance rules for the prompt, not enforcement rules in the tool registry.

### Tool Information In Prompt

The fixed instruction should describe tool domains and decision rules, not enumerate every concrete tool name in a long static list.

The prompt should explain domains such as:

- `host`
- `deployment`
- `service`
- `kubernetes`
- `monitor`
- `governance`

It should also define these rules:

1. Start with the domain most relevant to the user task.
2. If a scene exists, use it as the default starting bias.
3. Use readonly tools first when assessing current state or blast radius.
4. Expand to neighboring domains if evidence is incomplete.
5. Do not treat scene as a hard boundary.

Concrete tool availability still comes from the runtime tool list provided by ADK.

## Architecture Impact

### Agent

`ChatModelAgent` should use a fixed `Instruction` string. It no longer depends on dynamic instruction rendering.

### Orchestrator

The orchestrator should stop generating per-request system instructions from runtime context. Instead, it should construct the input envelope and pass the combined text as the message payload for the run.

### Runtime

`BuildInstruction` becomes unnecessary for runtime-scoped personalization. It can either be removed entirely or replaced with a thin fixed-prompt accessor, depending on how the code is organized.

### Tool Selection

No hard scene filtering is introduced in the registry by this design. Tool access remains broad; prompt guidance handles preference.

## Data Flow

1. Frontend sends user message plus `RuntimeContext`.
2. Backend builds a compact runtime-context prefix.
3. Backend combines prefix and raw user message into the effective user input.
4. Agent receives:
   - fixed instruction
   - current tool list
   - effective user input with runtime context envelope
5. Agent chooses tools by user intent first, scene bias second, evidence third.

## Error Handling

- Missing `scene` should not be treated as an error.
- Missing `project` or `page` should simply omit those lines from the context envelope.
- Invalid or noisy metadata should not leak into the prompt by default.
- If runtime context cannot be normalized, the agent should still run with fixed instruction and raw user input.

## Testing

Tests should cover:

- fixed instruction remains constant across requests
- runtime context is encoded into a compact user-input envelope
- scene influences prompt-visible context without hard-limiting tools
- empty optional fields are omitted cleanly
- mutating-tool approval behavior remains unchanged
- cross-domain tool usage is still possible from a scene-biased request

## Trade-Offs

### Benefits

- More stable agent behavior
- Clearer separation between global operating rules and per-request context
- Less fragile dependence on frontend display fields
- Easier reasoning about prompt changes

### Costs

- The user-input envelope becomes a prompt contract that must stay stable
- Scene guidance is heuristic, so prompt wording quality matters
- Some previous dynamic instruction machinery becomes redundant and should be cleaned up

## Recommendation

Adopt the fixed-instruction plus runtime-context-envelope model.

This gives the agent a stable global operating policy while preserving enough per-request context to bias tool choice intelligently. It matches the desired behavior: the scene should guide where the agent starts, but it must never become an artificial wall that prevents the correct cross-domain tool from being used.
