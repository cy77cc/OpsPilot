import { useCallback, useEffect, useRef, useState } from 'react';
import { message } from 'antd';
import { Api } from '../../../../api';
import type {
  CertificateInfo,
  Cluster,
  ClusterNode,
  ClusterUpgradePlan,
  ClusterVersionInfo,
  EventInfo,
} from '../../../../api/modules/cluster';

type UseClusterDetailResult = {
  isInitialLoading: boolean;
  cluster: Cluster | null;
  nodes: ClusterNode[];
  nodesLoading: boolean;
  events: EventInfo[];
  clusterVersion: ClusterVersionInfo | null;
  certificates: CertificateInfo[];
  upgradePlan: ClusterUpgradePlan | null;
  loadCluster: () => Promise<void>;
  loadNodes: () => Promise<void>;
  loadEvents: () => Promise<void>;
  loadClusterInfo: () => Promise<void>;
  syncNodes: () => Promise<void>;
};

export function useClusterDetail(clusterId: number): UseClusterDetailResult {
  const initialLoadRef = useRef(true);
  const [isInitialLoading, setIsInitialLoading] = useState(true);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [nodes, setNodes] = useState<ClusterNode[]>([]);
  const [nodesLoading, setNodesLoading] = useState(false);
  const [events, setEvents] = useState<EventInfo[]>([]);
  const [clusterVersion, setClusterVersion] = useState<ClusterVersionInfo | null>(null);
  const [certificates, setCertificates] = useState<CertificateInfo[]>([]);
  const [upgradePlan, setUpgradePlan] = useState<ClusterUpgradePlan | null>(null);

  const loadCluster = useCallback(async () => {
    if (!clusterId) return;
    const firstLoad = initialLoadRef.current;
    if (firstLoad) {
      setIsInitialLoading(true);
    }
    try {
      const res = await Api.cluster.getClusterDetail(clusterId);
      setCluster(res.data);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载集群信息失败');
    } finally {
      if (firstLoad) {
        initialLoadRef.current = false;
        setIsInitialLoading(false);
      }
    }
  }, [clusterId]);

  const loadNodes = useCallback(async () => {
    if (!clusterId) return;
    setNodesLoading(true);
    try {
      const res = await Api.cluster.getClusterNodes(clusterId);
      setNodes(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载节点列表失败');
    } finally {
      setNodesLoading(false);
    }
  }, [clusterId]);

  const loadEvents = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getEvents(clusterId);
      setEvents(res.data.list || []);
    } catch (err) {
      console.error('Failed to load events:', err);
    }
  }, [clusterId]);

  const loadClusterInfo = useCallback(async () => {
    if (!clusterId) return;
    try {
      const [versionRes, certRes, planRes] = await Promise.all([
        Api.cluster.getClusterVersion(clusterId),
        Api.cluster.getCertificates(clusterId),
        Api.cluster.getUpgradePlan(clusterId),
      ]);
      setClusterVersion(versionRes.data);
      setCertificates(certRes.data.list || []);
      setUpgradePlan(planRes.data);
    } catch (err) {
      console.error('Failed to load cluster info:', err);
    }
  }, [clusterId]);

  const syncNodes = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.syncClusterNodes(clusterId);
      setNodes(res.data.list || []);
      message.success('节点信息已同步');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '同步失败');
    }
  }, [clusterId]);

  useEffect(() => {
    void loadCluster();
    void loadNodes();
    void loadEvents();
    void loadClusterInfo();
  }, [loadCluster, loadNodes, loadEvents, loadClusterInfo]);

  return {
    isInitialLoading,
    cluster,
    nodes,
    nodesLoading,
    events,
    clusterVersion,
    certificates,
    upgradePlan,
    loadCluster,
    loadNodes,
    loadEvents,
    loadClusterInfo,
    syncNodes,
  };
}
