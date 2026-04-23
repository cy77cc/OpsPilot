import { useCallback, useEffect, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { message } from 'antd';
import { Api } from '../../../../api';
import type {
  ClusterServiceInfo,
  ConfigMapInfo,
  DaemonSetInfo,
  DeploymentInfo,
  HPAInfo,
  IngressInfo,
  LimitRangeInfo,
  NamespaceInfo,
  PVCInfo,
  PVInfo,
  PodInfo,
  ResourceQuotaInfo,
  SecretInfo,
  ServiceInfo,
  StatefulSetInfo,
} from '../../../../api/modules/cluster';

type UseClusterResourcesResult = {
  namespaces: NamespaceInfo[];
  selectedNamespace: string;
  setSelectedNamespace: Dispatch<SetStateAction<string>>;
  deployments: DeploymentInfo[];
  statefulsets: StatefulSetInfo[];
  daemonsets: DaemonSetInfo[];
  pods: PodInfo[];
  services: ServiceInfo[];
  ingresses: IngressInfo[];
  configmaps: ConfigMapInfo[];
  secrets: SecretInfo[];
  pvcs: PVCInfo[];
  pvs: PVInfo[];
  clusterServices: ClusterServiceInfo[];
  resourceLoading: boolean;
  hpas: HPAInfo[];
  resourceQuotas: ResourceQuotaInfo[];
  limitRanges: LimitRangeInfo[];
  advancedLoading: boolean;
  refreshSelectedNamespaceResources: () => Promise<void>;
};

export function useClusterResources(clusterId: number): UseClusterResourcesResult {
  const [namespaces, setNamespaces] = useState<NamespaceInfo[]>([]);
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default');
  const [deployments, setDeployments] = useState<DeploymentInfo[]>([]);
  const [statefulsets, setStatefulsets] = useState<StatefulSetInfo[]>([]);
  const [daemonsets, setDaemonsets] = useState<DaemonSetInfo[]>([]);
  const [pods, setPods] = useState<PodInfo[]>([]);
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [ingresses, setIngresses] = useState<IngressInfo[]>([]);
  const [configmaps, setConfigmaps] = useState<ConfigMapInfo[]>([]);
  const [secrets, setSecrets] = useState<SecretInfo[]>([]);
  const [pvcs, setPvcs] = useState<PVCInfo[]>([]);
  const [pvs, setPvs] = useState<PVInfo[]>([]);
  const [clusterServices, setClusterServices] = useState<ClusterServiceInfo[]>([]);
  const [resourceLoading, setResourceLoading] = useState(false);
  const [hpas, setHPAs] = useState<HPAInfo[]>([]);
  const [resourceQuotas, setResourceQuotas] = useState<ResourceQuotaInfo[]>([]);
  const [limitRanges, setLimitRanges] = useState<LimitRangeInfo[]>([]);
  const [advancedLoading, setAdvancedLoading] = useState(false);

  const loadNamespaces = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getNamespaces(clusterId);
      setNamespaces(res.data.list || []);
    } catch (err) {
      console.error('Failed to load namespaces:', err);
    }
  }, [clusterId]);

  const loadResources = useCallback(async (namespace: string) => {
    if (!clusterId) return;
    setResourceLoading(true);
    try {
      const [depRes, stsRes, dsRes, podRes, svcRes, ingRes, cmRes, secRes, pvcRes] = await Promise.all([
        Api.cluster.getDeployments(clusterId, namespace),
        Api.cluster.getStatefulSets(clusterId, namespace),
        Api.cluster.getDaemonSets(clusterId, namespace),
        Api.cluster.getPods(clusterId, namespace),
        Api.cluster.getServices(clusterId, namespace),
        Api.cluster.getIngresses(clusterId, namespace),
        Api.cluster.getConfigMaps(clusterId, namespace),
        Api.cluster.getSecrets(clusterId, namespace),
        Api.cluster.getPVCs(clusterId, namespace),
      ]);
      setDeployments(depRes.data.list || []);
      setStatefulsets(stsRes.data.list || []);
      setDaemonsets(dsRes.data.list || []);
      setPods(podRes.data.list || []);
      setServices(svcRes.data.list || []);
      setIngresses(ingRes.data.list || []);
      setConfigmaps(cmRes.data.list || []);
      setSecrets(secRes.data.list || []);
      setPvcs(pvcRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载资源失败');
    } finally {
      setResourceLoading(false);
    }
  }, [clusterId]);

  const loadPVs = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getPVs(clusterId);
      setPvs(res.data.list || []);
    } catch (err) {
      console.error('Failed to load PVs:', err);
    }
  }, [clusterId]);

  const loadClusterServices = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getClusterServices(clusterId);
      setClusterServices(res.data.list || []);
    } catch (err) {
      console.error('Failed to load cluster services:', err);
    }
  }, [clusterId]);

  const loadAdvancedResources = useCallback(async (namespace: string) => {
    if (!clusterId) return;
    setAdvancedLoading(true);
    try {
      const [hpaRes, quotaRes, limitRes] = await Promise.all([
        Api.cluster.getHPAs(clusterId, namespace),
        Api.cluster.getResourceQuotas(clusterId, namespace),
        Api.cluster.getLimitRanges(clusterId, namespace),
      ]);
      setHPAs(hpaRes.data.list || []);
      setResourceQuotas(quotaRes.data.list || []);
      setLimitRanges(limitRes.data.list || []);
    } catch (err) {
      console.error('Failed to load advanced resources:', err);
    } finally {
      setAdvancedLoading(false);
    }
  }, [clusterId]);

  const refreshSelectedNamespaceResources = useCallback(async () => {
    if (!selectedNamespace) return;
    await loadResources(selectedNamespace);
  }, [loadResources, selectedNamespace]);

  useEffect(() => {
    void loadNamespaces();
    void loadPVs();
    void loadClusterServices();
  }, [loadClusterServices, loadNamespaces, loadPVs]);

  useEffect(() => {
    if (!selectedNamespace) return;
    void loadResources(selectedNamespace);
    void loadAdvancedResources(selectedNamespace);
  }, [loadAdvancedResources, loadResources, selectedNamespace]);

  return {
    namespaces,
    selectedNamespace,
    setSelectedNamespace,
    deployments,
    statefulsets,
    daemonsets,
    pods,
    services,
    ingresses,
    configmaps,
    secrets,
    pvcs,
    pvs,
    clusterServices,
    resourceLoading,
    hpas,
    resourceQuotas,
    limitRanges,
    advancedLoading,
    refreshSelectedNamespaceResources,
  };
}
