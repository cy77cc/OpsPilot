/**
 * Host Service Management Module Type Definitions
 * 
 * This file contains TypeScript interfaces for the DevOps platform's
 * host service management module.
 * 
 * Version: 1.0.0
 * Date: 2026-02-23
 */

// ============================================================================
// Enumerations
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
  VOLCENGINE = 'volcengine',
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
 * Audit action type
 */
export type AuditAction =
  | 'host.created'
  | 'host.updated'
  | 'host.deleted'
  | 'host.status_changed'
  | 'host.initialized'
  | 'host.credential_changed'
  | 'host.tag_added'
  | 'host.tag_removed'
  | 'script.executed'
  | 'command.executed'
  | 'terminal.session_started'
  | 'terminal.session_ended';

// ============================================================================
// Base Interfaces
// ============================================================================

/**
 * Host tag (key-value format)
 */
export interface HostTag {
  key: string;
  value: string;
}

/**
 * Host hardware configuration
 */
export interface HostHardware {
  /** CPU core count */
  cpuCores: number;
  /** Memory size in GB */
  memoryGB: number;
  /** Disk size in GB */
  diskGB: number;
  /** Disk type */
  diskType: DiskType;
}

/**
 * Host location information
 */
export interface HostLocation {
  /** Data center / region */
  region: string;
  /** Availability zone */
  zone: string;
  /** Rack location (optional) */
  rack?: string;
  /** Server cabinet number (optional) */
  cabinet?: string;
}

/**
 * Host ownership information
 */
export interface HostOwnership {
  /** Business group */
  businessGroup: string;
  /** Owner */
  owner: string;
  /** Owner email (optional) */
  ownerEmail?: string;
}

/**
 * Host access credentials
 */
export interface HostCredentials {
  /** SSH key ID (linked to key management system) */
  sshKeyId?: string;
  /** Credential center index ID */
  credentialId?: string;
  /** SSH port (default 22) */
  sshPort: number;
  /** SSH username */
  sshUser: string;
}

// ============================================================================
// Main Host Model
// ============================================================================

/**
 * Main host asset model
 */
export interface Host {
  /** Unique host identifier (UUID) */
  id: string;
  /** Hostname */
  hostname: string;
  /** Private IP address */
  privateIp: string;
  /** Public IP address (optional) */
  publicIp?: string;
  /** Operating system version */
  osVersion: string;
  /** Hardware configuration */
  hardware: HostHardware;
  /** Location information */
  location: HostLocation;
  /** Ownership information */
  ownership: HostOwnership;
  /** Maintenance status */
  status: HostStatus;
  /** Access credentials */
  credentials: HostCredentials;
  /** Tags (key-value format) */
  tags: Record<string, string>;
  /** Creation timestamp */
  createdAt: string;
  /** Last update timestamp */
  updatedAt: string;
  /** Last active timestamp */
  lastActiveAt?: string;
}

// ============================================================================
// Host Metrics & Status
// ============================================================================

/**
 * Host real-time metrics
 */
export interface HostMetrics {
  /** Host ID */
  hostId: string;
  /** Collection timestamp */
  timestamp: string;
  /** CPU usage percentage */
  cpuUsage: number;
  /** Memory usage percentage */
  memoryUsage: number;
  /** Disk usage percentage */
  diskUsage: number;
  /** Disk IO (MB/s) */
  diskIops?: number;
  /** Network inbound (Mbps) */
  networkIn?: number;
  /** Network outbound (Mbps) */
  networkOut?: number;
  /** Load average (1min/5min/15min) */
  loadAverage?: [number, number, number];
}

/**
 * Host online status
 */
export interface HostStatusInfo {
  hostId: string;
  /** Online status */
  online: boolean;
  /** Agent status */
  agentStatus: 'connected' | 'disconnected' | 'error';
  /** Last heartbeat timestamp */
  lastHeartbeat: string;
  /** Heartbeat latency in milliseconds */
  heartbeatLatency?: number;
  /** Current load */
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

// ============================================================================
// API Request/Response Types
// ============================================================================

/**
 * Import host request
 */
export interface ImportHostRequest {
  /** Target host IP or hostname */
  target: string;
  /** SSH port (default 22) */
  port?: number;
  /** SSH username */
  username: string;
  /** Authentication */
  auth: {
    type: AuthType.KEY;
    keyId: string;
  } | {
    type: AuthType.PASSWORD;
    credentialId: string;
  };
  /** Location information */
  location?: {
    region: string;
    zone: string;
  };
  /** Ownership information */
  ownership?: {
    businessGroup: string;
    owner: string;
  };
}

/**
 * Import host response
 */
export interface ImportHostResponse {
  validation: {
    sshConnected: boolean;
    osDetected: boolean;
    hardwareCollected: boolean;
  };
  hostInfo: {
    hostname: string;
    privateIp: string;
    osVersion: string;
    cpuCores: number;
    memoryGB: number;
    diskGB: number;
    diskType: DiskType;
  };
  suggestedTags: Record<string, string>;
}

/**
 * Batch import request
 */
export interface BatchImportRequest {
  /** Cloud provider type */
  provider: CloudProvider;
  /** Cloud provider region */
  region: string;
  /** Cloud provider account ID */
  accountId: string;
  /** Sync options */
  options: {
    incremental?: boolean;
    autoCreate?: boolean;
    updateExisting?: boolean;
    tagFilters?: Record<string, string>;
  };
  /** Mapping configuration */
  mapping?: {
    businessGroupField?: string;
    ownerField?: string;
    tagPrefix?: string;
  };
}

/**
 * Batch import response
 */
export interface BatchImportResponse {
  taskId: string;
  estimatedCount: number;
  startedAt: string;
}

/**
 * Initialize host request
 */
export interface InitializeHostRequest {
  /** Agent type */
  agentType: AgentType;
  /** Initialization script content (Base64 encoded) */
  script?: string;
  /** Agent configuration */
  config?: {
    prometheusPort?: number;
    scrapeInterval?: number;
    customMetrics?: string[];
  };
  /** Execution options */
  options: {
    force?: boolean;
    timeout?: number;
  };
}

/**
 * Initialize host response
 */
export interface InitializeHostResponse {
  taskId: string;
  stages: ['upload' | 'execute' | 'configure' | 'start'];
}

/**
 * Execute script request
 */
export interface ExecuteScriptRequest {
  /** Script content (Base64 encoded) */
  script: string;
  /** Script interpreter */
  interpreter: ScriptInterpreter;
  /** Execution parameters */
  params?: {
    timeout?: number;
    workingDir?: string;
    env?: Record<string, string>;
    async?: boolean;
  };
  /** Execution credentials */
  credentials?: {
    sshKeyId?: string;
    credentialId?: string;
  };
}

/**
 * Execute script response (async)
 */
export interface ExecuteScriptResponse {
  taskId: string;
  status: 'queued' | 'running';
  wsUrl?: string;
}

/**
 * Execute script response (sync)
 */
export interface ExecuteScriptSyncResponse {
  taskId: string;
  status: 'success' | 'failed' | 'timeout';
  stdout: string;
  stderr: string;
  exitCode: number;
  duration: number;
  executedAt: string;
}

/**
 * Update host request
 */
export interface UpdateHostRequest {
  hostname?: string;
  location?: Partial<HostLocation>;
  ownership?: Partial<HostOwnership>;
  tags?: Record<string, string>;
  status?: HostStatus;
}

// ============================================================================
// Audit Log Types
// ============================================================================

/**
 * Host audit log
 */
export interface HostAuditLog {
  /** Log ID */
  id: string;
  /** Host ID */
  hostId: string;
  /** Action type */
  action: AuditAction;
  /** Operator */
  operator: string;
  /** Timestamp */
  timestamp: string;
  /** Changes */
  changes: {
    field: string;
    oldValue: string;
    newValue: string;
  }[];
  /** Request ID for tracing */
  requestId?: string;
  /** Client IP */
  clientIp?: string;
}

// ============================================================================
// WebSocket Protocol Types
// ============================================================================

/**
 * Terminal message type
 */
export type TerminalMessageType = 'input' | 'output' | 'resize' | 'heartbeat' | 'error' | 'session_end';

/**
 * Terminal message
 */
export interface TerminalMessage {
  /** Message type */
  type: TerminalMessageType;
  /** Payload */
  payload: string | object;
  /** Timestamp */
  timestamp: number;
}

/**
 * Terminal input payload
 */
export interface TerminalInputPayload {
  data: string;
  isResizeInput?: boolean;
}

/**
 * Terminal resize payload
 */
export interface TerminalResizePayload {
  cols: number;
  rows: number;
  width: number;
  height: number;
}

/**
 * Terminal output payload
 */
export interface TerminalOutputPayload {
  data: string;
  stream: 'stdout' | 'stderr';
}

/**
 * Terminal error payload
 */
export interface TerminalErrorPayload {
  code: 'SESSION_CLOSED' | 'AUTH_FAILED' | 'HOST_OFFLINE';
  message: string;
}

/**
 * Terminal session end payload
 */
export interface TerminalSessionEndPayload {
  reason: 'user_disconnect' | 'timeout' | 'host_reboot';
  statistics: {
    duration: number;
    inputBytes: number;
    outputBytes: number;
  };
}

// ============================================================================
// Prometheus Service Discovery Types
// ============================================================================

/**
 * Prometheus file_sd target
 */
export interface PrometheusSDTarget {
  targets: string[];
  labels: Record<string, string>;
}

// ============================================================================
// Common API Response Types
// ============================================================================

/**
 * Generic API response wrapper
 */
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T | null;
}

/**
 * Paginated list response
 */
export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

/**
 * Error response
 */
export interface ErrorResponse {
  code: number;
  message: string;
  data: {
    stage?: string;
    detail?: string;
    suggestion?: string;
  } | null;
}
