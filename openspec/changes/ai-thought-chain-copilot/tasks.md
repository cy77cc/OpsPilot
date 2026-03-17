## 1. Streaming contract and runtime mapping

- [ ] 1.1 Audit the current AI chat SSE/log stream and define the normalized frontend event contract for `run_path`, `tool_call`, planner/replanner output, `replanner.response`, and completion.
- [ ] 1.2 Update backend AI chat streaming or API adaptation so complex-agent runs expose the fields needed to drive ThoughtChain and summary/final-response transitions.
- [ ] 1.3 Add backend or API-layer tests that verify complex agent runs emit stable planning, executor tool-call, replanner response, and done signals.

## 2. Frontend thinking-state model

- [ ] 2.1 Extend AI frontend types to represent `transientSummary`, `finalResponse`, card status, and nested ThoughtChain items derived from executor tool calls.
- [ ] 2.2 Implement a reducer/normalizer that maps streaming events into the copilot thinking state using `run_path` as the primary hierarchy source.
- [ ] 2.3 Add reducer tests covering the full flow `input -> intent -> transfer -> planner -> executor -> replanner -> executor -> tool_call* -> replanner -> done`.

## 3. Copilot rendering and UX

- [ ] 3.1 Update the assistant message rendering in the copilot surface to use Ant Design X `Think` as the default-collapsed `Deep thinking` container and Ant Design X `ThoughtChain` as the executor/tool-call chain renderer.
- [ ] 3.2 Render planner/replanner textual output outside the thought chain as user-visible streaming summary content.
- [ ] 3.3 Switch the main visible正文 to `finalResponse` once `replanner.response` arrives and collapse earlier planning/replanning summaries by default.
- [ ] 3.4 Preserve the lightweight rendering path for simple QA-style responses that do not produce a complex thought chain.
- [ ] 3.5 Avoid introducing a parallel custom timeline/card implementation when official `Think` and `ThoughtChain` cover the required interaction.

## 4. Verification

- [ ] 4.1 Add UI tests for complex-agent streaming, including collapsed thinking state, tool-call chain rendering, response handoff, and completed state.
- [ ] 4.2 Add regression coverage ensuring plain QA responses still render correctly without ThoughtChain noise.
- [ ] 4.3 Validate the finished change with OpenSpec and document the implementation handoff for `/opsx:apply`.
