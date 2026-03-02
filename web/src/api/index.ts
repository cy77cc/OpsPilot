// API模块索引文件

// 基础API服务
export * from './api';
import apiService from './api';

// 模块API
export * from './modules/hosts';
import { hostApi } from './modules/hosts';

export * from './modules/tasks';
import { taskApi } from './modules/tasks';

export * from './modules/kubernetes';
import { kubernetesApi } from './modules/kubernetes';

export * from './modules/monitoring';
import { monitoringApi } from './modules/monitoring';

export * from './modules/configs';
import { configApi } from './modules/configs';

export * from './modules/rbac';
import { rbacApi } from './modules/rbac';

export * from './modules/auth';
import { authApi } from './modules/auth';

export * from './modules/projects';
import { projectApi } from './modules/projects';

export * from './modules/services';
import { serviceApi } from './modules/services';

export * from './modules/ai';
import { aiApi } from './modules/ai';

export * from './modules/tools';
import { toolApi } from './modules/tools';

export * from './modules/cmdb';
import { cmdbApi } from './modules/cmdb';

export * from './modules/automation';
import { automationApi } from './modules/automation';

export * from './modules/cicd';
import { cicdApi } from './modules/cicd';

export * from './modules/topology';
import { topologyApi } from './modules/topology';
export * from './modules/deployment';
import { deploymentApi } from './modules/deployment';
export * from './modules/notification';
import { notificationApi } from './modules/notification';

// 统一导出所有API
export const Api = {
  cluster: {
    getList: () => apiService.get<{ list: any[]; total: number }>("/clusters"),
    getDetail: (id: number) => apiService.get<any>(`/clusters/${id}`),
    getClusters: () => apiService.get<{ list: any[]; total: number }>("/clusters"),
    getClusterDetail: (id: number) => apiService.get<any>(`/clusters/${id}`),
    getClusterNodes: (id: number) => apiService.get<{ list: any[]; total: number }>(`/clusters/${id}/nodes`),
  },
  // 基础服务
  service: apiService,

  // 模块API
  hosts: hostApi,
  tasks: taskApi,
  kubernetes: kubernetesApi,
  monitoring: monitoringApi,
  configs: configApi,
  rbac: rbacApi,
  auth: authApi,
  projects: projectApi,
  services: serviceApi,
  ai: aiApi,
  tools: toolApi,
  cmdb: cmdbApi,
  automation: automationApi,
  cicd: cicdApi,
  topology: topologyApi,
  deployment: deploymentApi,
  notification: notificationApi,
};

export default Api;
