import { useCallback, useMemo } from 'react';
import { message } from 'antd';
import { Api } from '../../../api';
import { handleApiError } from '../../../utils/apiErrorHandler';

export interface HPAEditorLoadResult {
  list: any[];
}

export interface SaveHPAInput {
  clusterId: string;
  existing: any[];
  hpa: any;
}

export interface RemoveHPAInput {
  clusterId: string;
  name: string;
  namespace: string;
}

export interface HPAEditorActions {
  load: () => Promise<HPAEditorLoadResult>;
  save: (input: SaveHPAInput) => Promise<void>;
  remove: (input: RemoveHPAInput) => Promise<void>;
}

export function useHPAEditorActions(clusterId: string): HPAEditorActions {
  const load = useCallback(async () => {
    try {
      const res = await Api.kubernetes.listHPA(clusterId);
      return { list: res.data.list || [] };
    } catch (err) {
      handleApiError(err, '加载 HPA 失败');
      throw err;
    }
  }, [clusterId]);

  const save = useCallback(async ({ clusterId: targetClusterId, existing, hpa }: SaveHPAInput) => {
    const found = existing.find((item) => item.name === hpa.name && item.namespace === hpa.namespace);
    if (found) {
      await Api.kubernetes.updateHPA(targetClusterId, hpa.name, hpa);
      message.success('HPA 已更新');
      return;
    }

    await Api.kubernetes.createHPA(targetClusterId, hpa);
    message.success('HPA 已创建');
  }, []);

  const remove = useCallback(async ({ clusterId: targetClusterId, name, namespace }: RemoveHPAInput) => {
    await Api.kubernetes.deleteHPA(targetClusterId, name, namespace);
    message.success('HPA 已删除');
  }, []);

  return useMemo(() => ({
    load,
    save,
    remove,
  }), [load, remove, save]);
}
