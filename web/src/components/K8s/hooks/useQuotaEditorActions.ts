import { useCallback, useMemo } from 'react';
import { message } from 'antd';
import { Api } from '../../../api';
import { handleApiError } from '../../../utils/apiErrorHandler';

export interface QuotaEditorLoadResult {
  quotas: any[];
  limits: any[];
}

export interface SaveQuotaInput {
  clusterId: string;
  quota: {
    namespace: string;
    name: string;
    hard: Record<string, string>;
  };
}

export interface SaveLimitInput {
  clusterId: string;
  limitRange: {
    namespace: string;
    name: string;
    default: Record<string, string>;
    default_request: Record<string, string>;
    min: Record<string, string>;
    max: Record<string, string>;
  };
}

export interface RemoveQuotaInput {
  clusterId: string;
  name: string;
  namespace: string;
}

export interface QuotaEditorActions {
  load: () => Promise<QuotaEditorLoadResult>;
  saveQuota: (input: SaveQuotaInput) => Promise<void>;
  saveLimit: (input: SaveLimitInput) => Promise<void>;
  removeQuota: (input: RemoveQuotaInput) => Promise<void>;
}

export function useQuotaEditorActions(clusterId: string): QuotaEditorActions {
  const load = useCallback(async () => {
    try {
      const [qRes, lRes] = await Promise.all([
        Api.kubernetes.listQuotas(clusterId),
        Api.kubernetes.listLimitRanges(clusterId),
      ]);

      return {
        quotas: qRes.data.list || [],
        limits: lRes.data.list || [],
      };
    } catch (err) {
      handleApiError(err, '加载配额失败');
      throw err;
    }
  }, [clusterId]);

  const saveQuota = useCallback(async ({ clusterId: targetClusterId, quota }: SaveQuotaInput) => {
    await Api.kubernetes.applyQuota(targetClusterId, quota);
    message.success('Quota 已应用');
  }, []);

  const saveLimit = useCallback(async ({ clusterId: targetClusterId, limitRange }: SaveLimitInput) => {
    await Api.kubernetes.createLimitRange(targetClusterId, limitRange);
    message.success('LimitRange 已应用');
  }, []);

  const removeQuota = useCallback(async ({ clusterId: targetClusterId, name, namespace }: RemoveQuotaInput) => {
    await Api.kubernetes.deleteQuota(targetClusterId, name, namespace);
    message.success('Quota 已删除');
  }, []);

  return useMemo(() => ({
    load,
    saveQuota,
    saveLimit,
    removeQuota,
  }), [load, removeQuota, saveLimit, saveQuota]);
}
