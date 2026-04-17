export * from '../../features/ai/api/shared';
export * from '../../features/ai/api/chatApi';
export * from '../../features/ai/api/sessionApi';
export * from '../../features/ai/api/runApi';
export * from '../../features/ai/api/approvalApi';

import { chatApi } from '../../features/ai/api/chatApi';
import { sessionApi } from '../../features/ai/api/sessionApi';
import { runApi } from '../../features/ai/api/runApi';
import { approvalApi } from '../../features/ai/api/approvalApi';

export const aiApi = {
  ...chatApi,
  ...sessionApi,
  ...runApi,
  ...approvalApi,
};
