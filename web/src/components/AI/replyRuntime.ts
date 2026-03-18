import type { AssistantReplyRuntime } from './types';

export function createEmptyAssistantRuntime(): AssistantReplyRuntime {
  return {
    activities: [],
    phase: undefined,
    phaseLabel: undefined,
    summary: undefined,
    status: undefined,
  };
}
