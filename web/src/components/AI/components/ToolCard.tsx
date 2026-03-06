import React from 'react';
import { LoadingOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import type { ToolExecution, ToolStatus } from '../types';

interface ToolCardProps {
  tool: ToolExecution;
}

/**
 * 简化版工具执行卡片
 * 显示工具名、状态、耗时
 */
export function ToolCard({ tool }: ToolCardProps) {
  const statusConfig = getStatusConfig(tool.status);

  return (
    <div className="ai-tool-card">
      <span className="tool-icon">🔧</span>
      <span className="tool-name">{formatToolName(tool.name)}</span>
      <span className={`tool-status ${tool.status}`}>
        {statusConfig.icon}
      </span>
      {tool.duration !== undefined && (
        <span className="tool-duration">{tool.duration.toFixed(1)}s</span>
      )}
      {tool.error && (
        <span className="tool-error" title={tool.error}>
          ⚠️
        </span>
      )}
    </div>
  );
}

/**
 * 获取状态配置
 */
function getStatusConfig(status: ToolStatus) {
  switch (status) {
    case 'running':
      return {
        icon: <LoadingOutlined spin />,
        text: '执行中',
      };
    case 'success':
      return {
        icon: <CheckCircleOutlined />,
        text: '成功',
      };
    case 'error':
      return {
        icon: <CloseCircleOutlined />,
        text: '失败',
      };
    default:
      return {
        icon: null,
        text: '',
      };
  }
}

/**
 * 格式化工具名称
 */
function formatToolName(name: string): string {
  // 移除前缀并格式化
  const cleanName = name
    .replace(/^(k8s_|host_|service_|monitor_)/, '')
    .replace(/_/g, ' ');

  // 首字母大写
  return cleanName
    .split(' ')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}
