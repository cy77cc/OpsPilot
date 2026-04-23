import React from 'react';
import { message } from 'antd';
import { Api } from '../../../api';

export interface NamespacePolicyLoadResult {
  namespaces: any[];
  bindings: any[];
}

export interface CreateNamespaceInput {
  clusterId: string;
  name: string;
  env?: string;
}

export interface SaveNamespaceBindingsInput {
  clusterId: string;
  teamId: string;
  namespaces: string[];
}

export interface RemoveNamespaceInput {
  clusterId: string;
  name: string;
}

export interface NamespacePolicyActions {
  load: () => Promise<NamespacePolicyLoadResult>;
  createNamespace: (input: CreateNamespaceInput) => Promise<void>;
  saveBindings: (input: SaveNamespaceBindingsInput) => Promise<void>;
  removeNamespace: (input: RemoveNamespaceInput) => Promise<void>;
}

interface UseNamespacePolicyActionsOptions {
  clusterId: string;
  teamId?: string;
  setTeamId: (teamId: string) => void;
}

export function useNamespacePolicyActions({
  clusterId,
  teamId,
  setTeamId,
}: UseNamespacePolicyActionsOptions): NamespacePolicyActions {
  const teamIdRef = React.useRef(teamId);
  const setTeamIdRef = React.useRef(setTeamId);

  React.useEffect(() => {
    teamIdRef.current = teamId;
  }, [teamId]);

  React.useEffect(() => {
    setTeamIdRef.current = setTeamId;
  }, [setTeamId]);

  const load = React.useCallback(async () => {
    try {
      const [nsRes, bindRes] = await Promise.all([
        Api.kubernetes.getClusterNamespaces(clusterId),
        Api.kubernetes.getNamespaceBindings(clusterId, teamIdRef.current || undefined),
      ]);

      return {
        namespaces: nsRes.data.list || [],
        bindings: bindRes.data.list || [],
      };
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载命名空间失败');
      throw err;
    }
  }, [clusterId]);

  const createNamespace = React.useCallback(async ({ clusterId: targetClusterId, name, env }: CreateNamespaceInput) => {
    await Api.kubernetes.createNamespace(targetClusterId, { name, env });
    message.success('命名空间已创建');
  }, []);

  const saveBindings = React.useCallback(async ({ clusterId: targetClusterId, teamId: nextTeamId, namespaces }: SaveNamespaceBindingsInput) => {
    await Api.kubernetes.putNamespaceBindings(
      targetClusterId,
      nextTeamId,
      namespaces
        .map((namespace) => ({ namespace: namespace.trim() }))
        .filter((binding) => binding.namespace),
    );
    setTeamIdRef.current(nextTeamId);
    message.success('绑定已更新');
  }, []);

  const removeNamespace = React.useCallback(async ({ clusterId: targetClusterId, name }: RemoveNamespaceInput) => {
    await Api.kubernetes.deleteNamespace(targetClusterId, name);
    message.success('命名空间删除请求已提交');
  }, []);

  return React.useMemo(() => ({
    load,
    createNamespace,
    saveBindings,
    removeNamespace,
  }), [createNamespace, load, removeNamespace, saveBindings]);
}
