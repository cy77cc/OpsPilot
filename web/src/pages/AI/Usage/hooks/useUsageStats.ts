import { useRequest } from 'ahooks';
import { useState, useCallback } from 'react';
import { aiApi } from '../../../../api/modules/ai';
import type { UsageStats, UsageLogsResult, UsageStatsParams, UsageLogsParams } from '../../../../api/modules/ai';

export function useUsageStats(defaultParams?: UsageStatsParams) {
  const [params, setParams] = useState<UsageStatsParams>(defaultParams || {});

  const fetcher = useCallback(() => aiApi.getUsageStats(params), [params]);

  const { data, loading, error, refresh } = useRequest(fetcher, {
    refreshDeps: [params],
  });

  return {
    stats: data?.data,
    loading,
    error,
    params,
    setParams,
    refresh,
  };
}

export function useUsageLogs(defaultParams?: UsageLogsParams) {
  const [params, setParams] = useState<UsageLogsParams>(defaultParams || { page: 1, page_size: 20 });

  const fetcher = useCallback(() => aiApi.getUsageLogs(params), [params]);

  const { data, loading, error, refresh } = useRequest(fetcher, {
    refreshDeps: [params],
  });

  return {
    result: data?.data,
    loading,
    error,
    params,
    setParams,
    refresh,
  };
}
