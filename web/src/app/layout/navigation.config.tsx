import React from 'react';
import {
  DashboardOutlined,
  DesktopOutlined,
  SettingOutlined,
  AlertOutlined,
  CloudOutlined,
  ClockCircleOutlined,
  ToolOutlined,
  UserOutlined,
  CloudServerOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import type { MenuSection } from './navigation.types';

interface BuildMenuSectionsOptions {
  t: (key: string) => string;
  governanceMenuEnabled: boolean;
  canReadGovernance: boolean;
}

export const LEGACY_GOVERNANCE_MENU_ITEMS = [
  { key: '/settings/users', label: '用户管理' },
  { key: '/settings/roles', label: '角色管理' },
  { key: '/settings/permissions', label: '权限列表' },
] as const;

export function buildMenuSections({
  t,
  governanceMenuEnabled,
  canReadGovernance,
}: BuildMenuSectionsOptions): MenuSection[] {
  return [
    {
      key: 'overview',
      title: '总览',
      items: [{ key: '/', icon: <DashboardOutlined />, label: t('menu.dashboard') }],
    },
    {
      key: 'delivery',
      title: '研发交付',
      items: [
        { key: '/services', icon: <CloudServerOutlined />, label: t('menu.services') },
        { key: '/deployment', icon: <CloudOutlined />, label: '发布中心' },
        { key: '/deployment/targets', icon: <CloudOutlined />, label: '发布目标' },
        { key: '/cicd', icon: <ToolOutlined />, label: 'CI/CD' },
      ],
    },
    {
      key: 'infrastructure',
      title: '基础设施',
      items: [
        { key: '/deployment/infrastructure/clusters', icon: <CloudOutlined />, label: '集群管理' },
        { key: '/deployment/infrastructure/hosts', icon: <DesktopOutlined />, label: '主机管理' },
      ],
    },
    {
      key: 'observability',
      title: '观测治理',
      items: [
        { key: '/monitor', icon: <AlertOutlined />, label: t('menu.monitor') },
        { key: '/deployment/observability/topology', icon: <CloudOutlined />, label: '部署拓扑' },
        { key: '/deployment/observability/audit-logs', icon: <AlertOutlined />, label: '审计日志' },
        { key: '/deployment/observability/policies', icon: <AlertOutlined />, label: '策略管理' },
        { key: '/deployment/observability/aiops', icon: <AlertOutlined />, label: 'AIOps 洞察' },
      ],
    },
    {
      key: 'ops',
      title: '运维运营',
      items: [
        { key: '/automation', icon: <ToolOutlined />, label: t('menu.automation') },
        { key: '/tasks', icon: <ClockCircleOutlined />, label: t('menu.tasks') },
        { key: '/cmdb', icon: <CloudServerOutlined />, label: t('menu.cmdb') },
      ],
    },
    {
      key: 'support',
      title: '平台与支持',
      items: [
        { key: '/settings', icon: <SettingOutlined />, label: '基础设置' },
        { key: '/settings/ai-models', icon: <SettingOutlined />, label: 'AI 模型配置' },
        ...(!governanceMenuEnabled
          ? LEGACY_GOVERNANCE_MENU_ITEMS.map(({ key, label }) => ({
              key,
              icon: <UserOutlined />,
              label,
            }))
          : []),
        ...(governanceMenuEnabled && canReadGovernance
          ? [{ key: '/governance/users', icon: <UserOutlined />, label: '访问治理' }]
          : []),
        { key: '/tools', icon: <ToolOutlined />, label: t('menu.tools') },
        { key: '/help', icon: <FileTextOutlined />, label: '帮助中心' },
      ],
    },
  ];
}
