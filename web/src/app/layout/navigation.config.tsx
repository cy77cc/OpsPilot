import React from 'react';
import {
  LayoutDashboard,
  HardDrive,
  Layers,
  Folder,
  Cpu,
  Box,
  Rocket,
  Zap,
  Settings,
  Activity,
  LineChart,
  Network,
  Database,
  Clock,
  Users,
  ShieldCheck,
  FileSearch,
  History,
  Sparkles,
  Wrench,
  BrainCircuit,
  BarChart3,
} from 'lucide-react';
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
        { key: '/', icon: <LayoutDashboard size={18} />, label: '概览' }
      ],
    },
    {
      key: 'resource',
      title: '资源管理',
      items: [
        { key: '/deployment/infrastructure/hosts', icon: <HardDrive size={18} />, label: '主机' },
        { key: '/deployment/infrastructure/clusters', icon: <Layers size={18} />, label: '集群' },
        { key: '/projects', icon: <Folder size={18} />, label: '项目' },
        { key: '/nodes', icon: <Cpu size={18} />, label: '节点' },
      ],
    },
    {
      key: 'delivery',
      title: '应用交付',
      items: [
        { key: '/services', icon: <Box size={18} />, label: '服务' },
        { key: '/deployment', icon: <Rocket size={18} />, label: '部署' },
        { key: '/cicd', icon: <Zap size={18} />, label: 'CICD' },
        { key: '/automation', icon: <Settings size={18} />, label: '自动化' },
      ],
    },
    {
      key: 'observability',
      title: '运维观测',
      items: [
        { key: '/monitor', icon: <Activity size={18} />, label: '监控告警' },
        { key: '/deployment/observability/metrics', icon: <LineChart size={18} />, label: '仪表盘' },
        { key: '/deployment/observability/topology', icon: <Network size={18} />, label: '拓扑' },
        { key: '/cmdb', icon: <Database size={18} />, label: 'CMDB' },
        { key: '/tasks', icon: <Clock size={18} />, label: '任务中心' },
      ],
    },
    {
      key: 'governance',
      title: '治理安全',
      items: [
        { key: '/org', icon: <Users size={18} />, label: '部门管理' },
        { key: '/approvals', icon: <FileSearch size={18} />, label: '审批管理' },
        { key: '/deployment/observability/audit-logs', icon: <History size={18} />, label: '审计日志' },
      ],
    },
    {
      key: 'ai-control',
      title: 'AI 控制面',
      items: [
        { key: '/ai/chat', icon: <Sparkles size={18} />, label: 'AI 对话' },
        { key: '/settings/ai-models', icon: <Wrench size={18} />, label: '工具管理' },
        { key: '/ai/models', icon: <BrainCircuit size={18} />, label: '大模型管理' },
        { key: '/ai/usage', icon: <BarChart3 size={18} />, label: '使用统计' },
      ],
    },
  ];
}
