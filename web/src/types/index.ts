export interface Host {
  id: string;
  name: string;
  ip: string;
  status: 'online' | 'offline' | 'warning' | 'maintenance';
  cpu: number;
  memory: number;
  disk: number;
  network: number;
  tags: string[];
  region: string;
  createdAt: string;
  lastActive: string;
}

export interface Config {
  id: string;
  name: string;
  key: string;
  value: string;
  version: number;
  status: 'active' | 'draft' | 'deprecated';
  type: string;
  env: string;
  updatedAt: string;
  updatedBy: string;
}

export interface ConfigVersion {
  id: string;
  configId: string;
  version: number;
  value: string;
  createdAt: string;
  createdBy: string;
  comment: string;
}

export interface Task {
  id: string;
  name: string;
  type: 'scheduled' | 'dependency' | 'parallel';
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
  schedule?: string;
  lastRun?: string;
  nextRun?: string;
  duration?: number;
  createdAt: string;
}

export interface TaskLog {
  id: string;
  taskId: string;
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  message: string;
}

export interface K8sCluster {
  id: string;
  name: string;
  version: string;
  status: 'healthy' | 'unhealthy' | 'warning';
  nodes: number;
  pods: number;
  cpu: number;
  memory: number;
  createdAt: string;
}

export interface K8sNode {
  id: string;
  name: string;
  role: 'master' | 'worker';
  status: 'ready' | 'notReady';
  cpu: number;
  memory: number;
  pods: number;
  labels: Record<string, string>;
}

export interface K8sContainer {
  name: string;
  image: string;
  status: 'running' | 'terminated' | 'waiting';
  cpu: number;
  memory: number;
  restarts: number;
}

export interface K8sPod {
  id: string;
  name: string;
  namespace: string;
  status: 'pending' | 'running' | 'failed' | 'succeeded' | 'terminating';
  phase?: string;
  node: string;
  cpu: number;
  memory: number;
  restarts: number;
  age: string;
  labels?: Record<string, string>;
  containers?: K8sContainer[];
  qosClass?: string;
  createdBy?: string;
  startTime?: string;
}

export interface K8sService {
  id: string;
  name: string;
  namespace: string;
  type: 'ClusterIP' | 'NodePort' | 'LoadBalancer' | 'ExternalName';
  clusterIP: string;
  externalIP?: string;
  ports: { port: number; targetPort: number; protocol: string }[];
  selector?: Record<string, string>;
  age: string;
} 

export interface K8sIngress {
  id: string;
  name: string;
  namespace: string;
  host: string;
  path: string;
  service: string;
  port: number;
  tls: boolean;
}

export interface Alert {
  id: string;
  title: string;
  severity: 'critical' | 'warning' | 'info';
  source: string;
  status: 'firing' | 'resolved';
  createdAt: string;
  resolvedAt?: string;
}

export interface AlertRule {
  id: string;
  name: string;
  condition: string;
  severity: 'critical' | 'warning' | 'info';
  enabled: boolean;
  channels: string[];
  createdAt: string;
}

export interface MonitorMetric {
  timestamp: string;
  value: number;
}

export interface IntegrationTool {
  id: string;
  name: string;
  icon: string;
  url: string;
  status: 'connected' | 'disconnected' | 'error';
  description: string;
}

// Service Management Types
export interface Service {
  id: string;
  name: string;
  status: 'running' | 'syncing' | 'deploying' | 'error';
  owner: string;
  environment: 'production' | 'staging' | 'development';
  tags: string[];
  cpu: number;
  memory: number;
  replicas: number;
  lastDeployTime: string;
  createdAt: string;
  k8sResources: {
    pods: K8sPod[];
    services: K8sService[];
    ingresses: K8sIngress[];
  };
  config: string;
  metrics: {
    cpu: number[];
    memory: number[];
  };
}

export interface ServiceQuota {
  cpuLimit: number;
  memoryLimit: number;
  cpuUsed: number;
  memoryUsed: number;
}

export interface ServiceFormData {
  name: string;
  description: string;
  owner: string;
  environment: 'production' | 'staging' | 'development';
  tags: string[];
  cpu: number;
  memory: number;
  replicas: number;
  image: string;
  port: number;
  targetPort: number;
  dependencies: string[];
}

// ============================================================================
// Host Service Management Module (v2 - 2026-02-23)
// ============================================================================

/**
 * Host maintenance status enumeration
 */
export enum HostStatus {
  ONLINE = 'online',
  OFFLINE = 'offline',
  INITIALIZING = 'initializing',
  MAINTENANCE = 'maintenance'
}

/**
 * Disk type enumeration
 */
export enum DiskType {
  SSD = 'ssd',
  HDD = 'hdd',
  NVME = 'nvme'
}

/**
 * Cloud provider enumeration
 */
export enum CloudProvider {
  AWS = 'aws',
  GCP = 'gcp',
  ALIBABA = 'alibaba',
  TENCENT = 'tencent',
  HUAWEI = 'huawei',
  ON_PREMISE = 'on_premise'
}

/**
 * Agent type enumeration
 */
export enum AgentType {
  TELEGRAF = 'telegraf',
  NODE_EXPORTER = 'node_exporter',
  CUSTOM = 'custom'
}

/**
 * Script interpreter enumeration
 */
export enum ScriptInterpreter {
  BASH = 'bash',
  PYTHON = 'python',
  POWERSHELL = 'powershell',
  ANSIBLE = 'ansible'
}

/**
 * Authentication type enumeration
 */
export enum AuthType {
  KEY = 'key',
  PASSWORD = 'password'
}

/**
 * Host hardware configuration
 */
export interface HostHardware {
  cpuCores: number;
  memoryGB: number;
  diskGB: number;
  diskType: DiskType;
}

/**
 * Host location information
 */
export interface HostLocation {
  region: string;
  zone: string;
  rack?: string;
  cabinet?: string;
}

/**
 * Host ownership information
 */
export interface HostOwnership {
  businessGroup: string;
  owner: string;
  ownerEmail?: string;
}

/**
 * Host access credentials
 */
export interface HostCredentials {
  sshKeyId?: string;
  credentialId?: string;
  sshPort: number;
  sshUser: string;
}

/**
 * Extended Host asset model (v2)
 */
export interface HostV2 {
  id: string;
  hostname: string;
  privateIp: string;
  publicIp?: string;
  osVersion: string;
  hardware: HostHardware;
  location: HostLocation;
  ownership: HostOwnership;
  status: HostStatus;
  credentials: HostCredentials;
  tags: Record<string, string>;
  createdAt: string;
  updatedAt: string;
  lastActiveAt?: string;
}

/**
 * Host real-time metrics
 */
export interface HostMetrics {
  hostId: string;
  timestamp: string;
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  diskIops?: number;
  networkIn?: number;
  networkOut?: number;
  loadAverage?: [number, number, number];
}

/**
 * Host online status
 */
export interface HostStatusInfo {
  hostId: string;
  online: boolean;
  agentStatus: 'connected' | 'disconnected' | 'error';
  lastHeartbeat: string;
  heartbeatLatency?: number;
  currentLoad?: {
    cpuUsage: number;
    memoryUsage: number;
    diskUsage: number;
  };
}

/**
 * Host summary for list display
 */
export interface HostSummary {
  id: string;
  hostname: string;
  privateIp: string;
  publicIp?: string;
  status: HostStatus;
  osVersion: string;
  cpuCores: number;
  memoryGB: number;
  diskGB: number;
  region: string;
  businessGroup: string;
  owner: string;
  tags: Record<string, string>;
  lastActiveAt?: string;
}

/**
 * Host audit log
 */
export interface HostAuditLog {
  id: string;
  hostId: string;
  action: string;
  operator: string;
  timestamp: string;
  changes: {
    field: string;
    oldValue: string;
    newValue: string;
  }[];
  requestId?: string;
  clientIp?: string;
}

// ============================================================================
// Configuration Center Types (2026-02-23)
// ============================================================================

export type ConfigFormat = 'text' | 'json' | 'yaml';
export type ConfigEnv = 'dev' | 'test' | 'staging' | 'prod';
export type ReleaseStatus = 'success' | 'failed';
export type AuditAction = 'create' | 'update' | 'delete' | 'release' | 'rollback';

/**
 * Configuration Application
 */
export interface ConfigApp {
  id: string;
  name: string;
  serviceId?: string;
  description: string;
  namespaces: string[];
  createdAt: string;
  updatedAt?: string;
}

/**
 * Configuration Item Version (simplified for embedded versions)
 */
export interface ConfigItemVersion {
  version: number;
  value: string;
  createdBy: string;
  createdAt: string;
  comment: string;
}

/**
 * Configuration Item
 */
export interface ConfigItem {
  id: string;
  appId: string;
  namespace: string;
  env: ConfigEnv;
  key: string;
  value: string;
  format: ConfigFormat;
  isSecret: boolean;
  versions: ConfigItemVersion[];
  createdAt: string;
  updatedAt: string;
  updatedBy: string;
}

/**
 * Release Record
 */
export interface Release {
  id: string;
  appId: string;
  namespace: string;
  key: string;
  env: ConfigEnv;
  fromVersion: number;
  toVersion: number;
  releasedBy: string;
  releasedAt: string;
  status: ReleaseStatus;
  comment?: string;
}

/**
 * Audit Log
 */
export interface AuditLog {
  id: string;
  appId: string;
  appName?: string;
  namespace: string;
  key: string;
  action: AuditAction;
  operator: string;
  timestamp: string;
  details: string;
  status: 'success' | 'failed';
  oldValue?: string;
  newValue?: string;
}

/**
 * Config Template
 */
export interface ConfigTemplate {
  id: string;
  name: string;
  description: string;
  format: ConfigFormat;
  content: string;
  category: string;
}

// ============================================================================
// Job Scheduler Types (2026-02-23)
// ============================================================================

export interface Job {
  id: string;
  name: string;
  type: 'shell' | 'http' | 'python' | 'ansible' | 'kubectl';
  command: string;
  schedule: string;
  timeout: number;
  hostGroupId?: string;
  strategy: 'random' | 'round-robin' | 'specify' | 'broadcast';
  retryCount: number;
  retryInterval: number;
  concurrencyPolicy: 'Allow' | 'Forbid' | 'Replace';
  description?: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Execution {
  id: string;
  jobId: string;
  hostId?: string;
  startTime: string;
  endTime?: string;
  status: 'pending' | 'running' | 'success' | 'failed' | 'killed';
  exitCode?: number;
  stdout: string;
  stderr: string;
  retryCount: number;
}

export interface JobSchedule {
  id: string;
  jobId: string;
  scheduledTime: string;
  actualStartTime?: string;
  actualEndTime?: string;
  status: 'scheduled' | 'executed' | 'missed';
}

export type JobStatus = 'pending' | 'running' | 'success' | 'failed' | 'killed' | 'disabled';

export type ScheduleStatus = 'scheduled' | 'executed' | 'missed';
