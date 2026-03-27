import type { K8sPod, K8sService } from './index';

export interface Toleration {
  key: string;
  operator: 'Exists' | 'Equal';
  value?: string;
  effect?: 'NoSchedule' | 'PreferNoSchedule' | 'NoExecute';
  tolerationSeconds?: number;
}

export interface Affinity {
  nodeAffinity?: NodeAffinity;
  podAffinity?: PodAffinity;
  podAntiAffinity?: PodAntiAffinity;
}

export interface NodeAffinity {
  preferredDuringSchedulingIgnoredDuringExecution?: PreferredSchedulingTerm[];
  requiredDuringSchedulingIgnoredDuringExecution?: NodeSelector;
}

export interface PreferredSchedulingTerm {
  weight: number;
  preference: NodeSelectorTerm;
}

export interface NodeAffinity {
  preferredDuringSchedulingIgnoredDuringExecution?: PreferredSchedulingTerm[];
  requiredDuringSchedulingIgnoredDuringExecution?: NodeSelector;
}

export interface PreferredSchedulingTerm {
  weight: number;
  preference: NodeSelectorTerm;
}

export interface NodeSelector {
  nodeSelectorTerms: NodeSelectorTerm[];
}

export interface NodeSelectorTerm {
  matchExpressions?: NodeSelectorRequirement[];
  matchFields?: NodeSelectorRequirement[];
}

export interface NodeSelectorRequirement {
  key: string;
  operator: string;
  values?: string[];
}

export interface PodAffinity {
  requiredDuringSchedulingIgnoredDuringExecution?: PodAffinityTerm[];
  preferredDuringSchedulingIgnoredDuringExecution?: WeightedPodAffinityTerm[];
}

export interface PodAntiAffinity {
  requiredDuringSchedulingIgnoredDuringExecution?: PodAffinityTerm[];
  preferredDuringSchedulingIgnoredDuringExecution?: WeightedPodAffinityTerm[];
}

export interface PodAffinityTerm {
  labelSelector?: LabelSelector;
  namespaces?: string[];
  topologyKey: string;
}

export interface WeightedPodAffinityTerm {
  weight: number;
  podAffinityTerm: PodAffinityTerm;
}

export interface LabelSelector {
  matchLabels?: Record<string, string>;
  matchExpressions?: LabelSelectorRequirement[];
}

export interface LabelSelectorRequirement {
  key: string;
  operator: string;
  values?: string[];
}

export interface Cluster {
  id: string;
  name: string;
  apiUrl: string;
  caCert: string;
  token: string;
  version: string;
  region: string;
  status: 'healthy' | 'unhealthy' | 'disconnected';
  connectionStatus: 'connected' | 'disconnected';
  nodeCount: number;
  stats: {
    nodeCount: number;
    cpuTotal: number;
    cpuAllocated: number;
    memoryTotalGB: number;
    memoryAllocatedGB: number;
    podTotal: number;
    podAllocated: number;
  };
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

export interface Node {
  id: string;
  clusterId: string;
  hostId: string;
  name: string;
  arch: string;
  os: string;
  kernelVersion: string;
  runtimeVersion: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  taints: Array<Taint>;
  status: 'ready' | 'notReady' | 'schedulingDisabled';
  unschedulable: boolean;
  capacity: NodeResources;
  allocatable: NodeResources;
  conditions: NodeCondition[];
  internalIP?: string;
  externalIP?: string;
  creationTimestamp: string;
}

interface Taint {
  key: string;
  value: string;
  effect: 'NoSchedule' | 'PreferNoSchedule' | 'NoExecute';
}

interface NodeResources {
  cpu: number;
  memory: number;
  pods: number;
  ephemeralStorage?: number;
  storage?: number;
}

interface NodeCondition {
  type: 'Ready' | 'MemoryPressure' | 'DiskPressure' | 'PIDPressure' | 'NetworkUnavailable';
  status: 'True' | 'False' | 'Unknown';
  reason?: string;
  message?: string;
  lastHeartbeatTime: string;
  lastTransitionTime: string;
}

export interface Pod extends K8sPod{
  clusterId: string;
  namespace: string;
  status: 'running' | 'pending' | 'failed' | 'succeeded' | 'terminating';
  phase: string;
  nodeName: string;
  nodeID: string;
  readyStatus: string;
  restarts: number;
  age: string;
  cpuRequests?: number;
  cpuLimits?: number;
  memoryRequests?: number;
  memoryLimits?: number;
  resources: {
    requests: {
      cpu: string;
      memory: string;
    };
    limits: {
      cpu: string;
      memory: string;
    };
  };
  nodeSelector?: Record<string, string>;
  tolerations?: Toleration[];
  affinity?: Affinity;
}

export interface Deployment {
  id: string;
  clusterId: string;
  name: string;
  namespace: string;
  replicas: number;
  readyReplicas: number;
  updatedReplicas: number;
  availableReplicas: number;
  unavailableReplicas: number;
  status: 'Active' | 'Progressing' | 'Failed' | 'Terminating';
  age: string;
  images: string[];
  selector: Record<string, string>;
  labels: Record<string, string>;
}

export interface ServiceResource extends K8sService {
  clusterId: string;
  namespace: string;
  type: 'ClusterIP' | 'NodePort' | 'LoadBalancer' | 'ExternalName';
  ingress: string[];
}

export interface ResourceUsage {
  percentage: number;
  allocated: number;
  total: number;
}

export interface ClusterMetrics {
  timestamp: string;
  cpuUsage: number;
  memoryUsage: number;
  podUsage: number;
  nodeCount: number;
}
