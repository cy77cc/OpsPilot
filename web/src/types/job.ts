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
