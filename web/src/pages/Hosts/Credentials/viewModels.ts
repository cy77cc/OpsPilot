import type {
  CredentialDetail,
  CredentialItem,
  CredentialPermissionItem,
  CredentialStats,
  CredentialTemplate,
  CredentialUsageRecord,
} from '../../../api/modules/hosts';

export interface CredentialListRowViewModel extends CredentialItem {
  key: string;
  typeLabel: string;
  authMethodLabel: string;
  statusLabel: string;
  statusTone: 'success' | 'warning' | 'danger' | 'default';
  expireHint?: string;
}

export interface CredentialDetailViewModel extends CredentialDetail {
  typeLabel: string;
  authMethodLabel: string;
  statusLabel: string;
  relationClusterCount: number;
}

export interface CredentialStatsCardViewModel {
  key: string;
  title: string;
  value: string;
  accent: string;
  soft: string;
  helper: string;
  icon: 'key' | 'safe' | 'warning' | 'danger' | 'recent';
  sparkline: number[];
}

export const credentialUsageFixtures: CredentialUsageRecord[] = [
  {
    id: 'usage-1',
    time: '2024-05-12 14:30:22',
    credentialName: 'prod-ssh-key',
    operator: 'admin',
    target: 'web-prod-01',
    method: 'SSH',
    result: 'success',
    sourceIp: '10.10.0.12',
    remark: '手动登录',
  },
  {
    id: 'usage-2',
    time: '2024-05-12 10:11:08',
    credentialName: 'db-root-password',
    operator: 'ops',
    target: 'db-master-01',
    method: 'SSH',
    result: 'failure',
    sourceIp: '10.10.0.25',
    remark: '认证失败',
  },
];

export const credentialPermissionFixtures: CredentialPermissionItem[] = [
  {
    id: 'perm-1',
    credentialName: 'prod-ssh-key',
    targetUserOrRole: 'admin / 超级管理员',
    permissions: ['查看', '使用', '编辑', '轮换'],
    scope: '全局',
    effectiveTime: '2024-04-01 00:00:00',
    expireTime: '长期有效',
    status: 'active',
  },
  {
    id: 'perm-2',
    credentialName: 'k8s-admin-token',
    targetUserOrRole: 'SRE',
    permissions: ['查看', '使用'],
    scope: '生产项目组',
    effectiveTime: '2024-05-01 00:00:00',
    expireTime: '2024-12-31 23:59:59',
    status: 'active',
  },
];

const typeLabelMap: Record<CredentialItem['type'], string> = {
  ssh_key: 'SSH 密钥',
  password: '密码',
  token: 'Token',
  certificate: '证书',
};

const authMethodLabelMap: Record<string, string> = {
  'SSH Key': 'SSH Key',
  ssh_key: 'SSH Key',
  key: 'SSH Key',
  '用户名密码': '用户名密码',
  username_password: '用户名密码',
  password: '用户名密码',
  'Bearer Token': 'Bearer Token',
  bearer_token: 'Bearer Token',
  token: 'Bearer Token',
  tls_cert: 'SSL/TLS 证书',
  certificate: 'SSL/TLS 证书',
};

const statsSparklineMap: Record<string, number[]> = {
  total: [16, 16, 17, 18, 17, 20, 19],
  available: [18, 20, 19, 21, 24, 22, 25],
  expiringSoon: [12, 14, 15, 13, 11, 10, 8],
  expired: [7, 9, 12, 10, 8, 8, 6],
  recent: [14, 16, 15, 17, 19, 18, 20],
};

export const toCredentialRowViewModel = (item: CredentialItem): CredentialListRowViewModel => {
  const statusLabelMap = {
    available: '可用',
    expiring_soon: '即将过期',
    expired: '已过期',
    disabled: '禁用',
  } as const;

  const expireHint =
    item.status === 'expiring_soon'
      ? '剩余 5 天'
      : item.status === 'expired'
        ? '已过期'
        : undefined;

  const statusTone =
    item.status === 'available'
      ? 'success'
      : item.status === 'expiring_soon'
        ? 'warning'
        : item.status === 'expired'
          ? 'danger'
          : 'default';

  return {
    ...item,
    key: item.id,
    typeLabel: typeLabelMap[item.type] || item.type,
    authMethodLabel: authMethodLabelMap[item.authMethod] || item.authMethod,
    statusLabel: statusLabelMap[item.status] || item.status,
    statusTone,
    expireHint,
  };
};

export const toCredentialDetailViewModel = (detail: CredentialDetail): CredentialDetailViewModel => ({
  ...detail,
  typeLabel: typeLabelMap[detail.type] || detail.type,
  authMethodLabel: authMethodLabelMap[detail.authMethod] || detail.authMethod,
  statusLabel:
    detail.status === 'available'
      ? '可用'
      : detail.status === 'expiring_soon'
        ? '即将过期'
        : detail.status === 'expired'
          ? '已过期'
          : '禁用',
  relationClusterCount: 2,
  tags: detail.tags?.length ? detail.tags : ['生产', '核心'],
  description: detail.description || '生产环境服务器访问密钥',
  recentUsage: detail.recentUsage || '2024-05-12 14:30:22',
});

export const buildStatsCards = (stats?: CredentialStats): CredentialStatsCardViewModel[] => [
  {
    key: 'total',
    title: '总凭证数',
    value: String(stats?.total || 0),
    helper: '较上周 +3',
    accent: '#2f6bff',
    soft: '#edf3ff',
    icon: 'key',
    sparkline: statsSparklineMap.total,
  },
  {
    key: 'available',
    title: '可用凭证',
    value: String(stats?.available || 0),
    helper: `可用率 ${stats?.total ? ((stats.available / stats.total) * 100).toFixed(1) : '0'}%`,
    accent: '#27ae60',
    soft: '#edf9f1',
    icon: 'safe',
    sparkline: statsSparklineMap.available,
  },
  {
    key: 'expiringSoon',
    title: '即将过期',
    value: String(stats?.expiringSoon || 0),
    helper: '7 天内过期',
    accent: '#fa8c16',
    soft: '#fff4e8',
    icon: 'warning',
    sparkline: statsSparklineMap.expiringSoon,
  },
  {
    key: 'expired',
    title: '已过期',
    value: String(stats?.expired || 0),
    helper: '已过期凭证',
    accent: '#ff4d4f',
    soft: '#fff1f0',
    icon: 'danger',
    sparkline: statsSparklineMap.expired,
  },
  {
    key: 'recent',
    title: '最近更新',
    value: stats?.recentUpdate || '-',
    helper: `由 ${stats?.recentUpdateBy || '-'} 更新`,
    accent: '#7a5af8',
    soft: '#f3efff',
    icon: 'recent',
    sparkline: statsSparklineMap.recent,
  },
];

export const buildTemplateRows = (items: CredentialTemplate[]) =>
  items.map((item, index) => ({
    ...item,
    relatedCredential: item.sshKeyName || (item.authType === 'key' ? 'SSH 密钥凭证' : '用户名密码凭证'),
    scope: ['生产环境', '开发环境', '跳板机'][index % 3],
    statusLabel: '启用',
  }));
