export * from '../../features/ai/api/shared';
export * from '../../features/ai/api/chatApi';
export * from '../../features/ai/api/sessionApi';
export * from '../../features/ai/api/runApi';
export * from '../../features/ai/api/approvalApi';
export * from '../../features/ai/api/adminModelApi';
export * from '../../features/ai/api/stubApi';
export * from '../../features/ai/api/assistApi';

import { chatApi } from '../../features/ai/api/chatApi';
import { sessionApi } from '../../features/ai/api/sessionApi';
import { runApi } from '../../features/ai/api/runApi';
import { approvalApi } from '../../features/ai/api/approvalApi';
import { adminModelApi } from '../../features/ai/api/adminModelApi';
import { stubApi } from '../../features/ai/api/stubApi';
import { assistApi } from '../../features/ai/api/assistApi';

export const aiApi = {
  ...chatApi,
  ...sessionApi,
  ...runApi,
  ...approvalApi,
  ...adminModelApi,
  ...stubApi,
  ...assistApi,
};
