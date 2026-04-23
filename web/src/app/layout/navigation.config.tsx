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
  SafetyCertificateOutlined,
  DeploymentUnitOutlined,
  MessageOutlined,
  AuditOutlined,
  PartitionOutlined,
  AppstoreOutlined,
  ThunderboltOutlined,
  ProjectOutlined,
  RocketOutlined,
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
      items: [
        { key: '/', icon: <DashboardOutlined />, label: '概览' }
      ],
    },
    {
      key: 'resource',
      title: '资源管理',
      items: [
        { key: '/deployment/infrastructure/hosts', icon: <DesktopOutlined />, label: '主机' },
        { key: '/deployment/infrastructure/clusters', icon: <PartitionOutlined />, label: '集群' },
        { key: '/projects', icon: <ProjectOutlined />, label: '项目' },
        { key: '/nodes', icon: <DeploymentUnitOutlined />, label: '节点' },
      ],
    },
    {
      key: 'delivery',
      title: '应用交付',
      items: [
        { key: '/services', icon: <CloudServerOutlined />, label: '服务' },
        { key: '/deployment', icon: <RocketOutlined />, label: '部署' },
        { key: '/cicd', icon: <ThunderboltOutlined />, label: 'CICD' },
        { key: '/automation', icon: <ToolOutlined />, label: '自动化' },
      ],
    },
    {
      key: 'observability',
      title: '运维观测',
      items: [
        { key: '/monitor', icon: <AlertOutlined />, label: '监控告警' },
        { key: '/deployment/observability/metrics', icon: <DashboardOutlined />, label: '仪表盘' },
        { key: '/deployment/observability/topology', icon: <DeploymentUnitOutlined />, label: '拓扑' },
        { key: '/cmdb', icon: <AppstoreOutlined />, label: 'CMDB' },
        { key: '/tasks', icon: <ClockCircleOutlined />, label: '任务中心' },
      ],
    },
    {
      key: 'governance',
      title: '治理安全',
      items: [
        { key: '/governance/users', icon: <UserOutlined />, label: '用户管理' },
        { key: '/governance/permissions', icon: <SafetyCertificateOutlined />, label: '权限管理' },
        { key: '/approvals', icon: <AuditOutlined />, label: '审批管理' },
        { key: '/deployment/observability/audit-logs', icon: <FileTextOutlined />, label: '审计日志' },
      ],
    },
    {
      key: 'ai-control',
      title: 'AI 控制面',
      items: [
        { key: '/ai/chat', icon: <MessageOutlined />, label: 'AI 对话' },
        { key: '/settings/ai-models', icon: <SettingOutlined />, label: '工具管理' },
        { key: '/ai/models', icon: <AppstoreOutlined />, label: '大模型管理' },
        { key: '/ai/usage', icon: <DashboardOutlined />, label: '使用统计' },
      ],
    },
  ];
}
