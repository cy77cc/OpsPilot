# Host Exec Unified Approval Rollout

## Phase 1 Gate

- [ ] `host_exec` is the only model-visible host execution entry.
- [ ] Approval middleware emits `tool_approval` for all approval-required host calls.
- [ ] Replay tests pass for legacy `host_exec_readonly` suspended events.

## Phase 2 Gate

- [ ] Legacy host tool invocation count is near zero for the observation window.
- [ ] DB policy records have been normalized to `host_exec`.
- [ ] No run history depends on legacy host tool names for new emissions.

## Phase 3 Gate

- [ ] Legacy registrations and policy aliases are removed from code.
- [ ] Rollback path for the normalization migration is documented as no-op.
- [ ] Spec and tests continue to pass with `host_exec`-only emissions.
