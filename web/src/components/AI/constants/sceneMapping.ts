/**
 * 路由到场景的映射配置
 */

export interface SceneConfig {
  key: string;
  label: string;
  routePrefix: string;
}

/**
 * 路由前缀 -> 场景配置映射
 */
export const SCENE_MAPPINGS: SceneConfig[] = [
  // 部署管理
  { key: 'deployment:clusters', label: '集群管理', routePrefix: '/deployment/infrastructure/clusters' },
  { key: 'deployment:credentials', label: '凭证管理', routePrefix: '/deployment/infrastructure/credentials' },
  { key: 'deployment:hosts', label: '主机管理', routePrefix: '/deployment/infrastructure/hosts' },
  { key: 'deployment:targets', label: '部署目标', routePrefix: '/deployment/targets' },
  { key: 'deployment:releases', label: '发布管理', routePrefix: '/deployment/overview' },
  { key: 'deployment:approvals', label: '审批中心', routePrefix: '/deployment/approvals' },
  { key: 'deployment:topology', label: '部署拓扑', routePrefix: '/deployment/observability/topology' },
  { key: 'deployment:metrics', label: '指标监控', routePrefix: '/deployment/observability/metrics' },
  { key: 'deployment:audit', label: '审计日志', routePrefix: '/deployment/observability/audit-logs' },
  { key: 'deployment:aiops', label: 'AIOps', routePrefix: '/deployment/observability/aiops' },

  // 服务管理
  { key: 'services:list', label: '服务列表', routePrefix: '/services' },
  { key: 'services:deploy', label: '服务部署', routePrefix: '/services/deploy' },
  { key: 'services:provision', label: '服务创建', routePrefix: '/services/provision' },
  { key: 'services:catalog', label: '服务目录', routePrefix: '/services/catalog' },

  // 治理
  { key: 'governance:users', label: '用户管理', routePrefix: '/governance/users' },
  { key: 'governance:roles', label: '角色管理', routePrefix: '/governance/roles' },
  { key: 'governance:permissions', label: '权限管理', routePrefix: '/governance/permissions' },

  // 监控
  { key: 'deployment:metrics', label: '监控中心', routePrefix: '/monitor' },

  // K8s
  { key: 'deployment:clusters', label: 'K8s 管理', routePrefix: '/k8s' },

  // 主机
  { key: 'deployment:hosts', label: '主机管理', routePrefix: '/hosts' },
];

/**
 * 根据路由路径获取场景配置
 */
export function getSceneByPath(pathname: string): SceneConfig | null {
  // 精确匹配和前缀匹配
  for (const config of SCENE_MAPPINGS) {
    if (pathname.startsWith(config.routePrefix)) {
      return config;
    }
  }
  return null;
}

/**
 * 场景标签映射
 */
export const SCENE_LABELS: Record<string, string> = {
  'global': '全局助手',
  ...Object.fromEntries(SCENE_MAPPINGS.map(s => [s.key, s.label])),
};

/**
 * 获取场景标签
 */
export function getSceneLabel(sceneKey: string): string {
  return SCENE_LABELS[sceneKey] || sceneKey;
}
