import { useCallback, useMemo } from 'react';
import { message } from 'antd';
import { Api } from '../../../api';
import { handleApiError } from '../../../utils/apiErrorHandler';

export type RolloutAction = 'promote' | 'abort' | 'rollback';

export interface RolloutActions {
  load: () => Promise<any[]>;
  preview: (input: Record<string, unknown>) => Promise<string>;
  apply: (input: Record<string, unknown>) => Promise<void>;
  act: (name: string, namespace: string, action: RolloutAction) => Promise<void>;
}

export function useRolloutActions(clusterId: string): RolloutActions {
  const load = useCallback(async () => {
    try {
      const res = await Api.kubernetes.listRollouts(clusterId);
      return res.data.list || [];
    } catch (err) {
      handleApiError(err, '加载 Rollouts 失败');
      throw err;
    }
  }, [clusterId]);

  const preview = useCallback(async (input: Record<string, unknown>) => {
    const res = await Api.kubernetes.previewRollout(clusterId, input);
    return res.data.manifest || '';
  }, [clusterId]);

  const apply = useCallback(async (input: Record<string, unknown>) => {
    await Api.kubernetes.applyRollout(clusterId, input);
    message.success('Rollout 已应用');
  }, [clusterId]);

  const act = useCallback(async (name: string, namespace: string, action: RolloutAction) => {
    if (action === 'promote') {
      await Api.kubernetes.promoteRollout(clusterId, name, { namespace });
    } else if (action === 'abort') {
      await Api.kubernetes.abortRollout(clusterId, name, { namespace });
    } else {
      await Api.kubernetes.rollbackRollout(clusterId, name, { namespace });
    }
    message.success(`操作已提交: ${action}`);
  }, [clusterId]);

  return useMemo(() => ({
    load,
    preview,
    apply,
    act,
  }), [act, apply, load, preview]);
}
