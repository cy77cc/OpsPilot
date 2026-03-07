import React, { useState } from 'react';
import { LoadingOutlined, CheckCircleOutlined, CloseCircleOutlined, DownOutlined, RightOutlined } from '@ant-design/icons';
import { theme } from 'antd';
import type { ToolExecution, ToolStatus } from '../types';

interface ToolCardProps {
  tool: ToolExecution;
}

/**
 * 工具执行卡片 (增强版)
 * 显示工具名、状态、耗时、参数和结果
 */
export function ToolCard({ tool }: ToolCardProps) {
  const { token } = theme.useToken();
  const [expanded, setExpanded] = useState(false);
  const hasDetails = tool.params || tool.result;

  const statusConfig = getStatusConfig(tool.status, token);

  return (
    <div
      style={{
        background: token.colorBgTextHover,
        border: `1px solid ${token.colorBorderSecondary}`,
        borderRadius: token.borderRadius,
        marginBottom: 8,
        overflow: 'hidden',
      }}
    >
      {/* 头部 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          padding: '8px 12px',
          gap: 8,
          cursor: hasDetails ? 'pointer' : 'default',
        }}
        onClick={() => hasDetails && setExpanded(!expanded)}
      >
        {hasDetails && (
          <span style={{ fontSize: 10, color: token.colorTextSecondary }}>
            {expanded ? <DownOutlined /> : <RightOutlined />}
          </span>
        )}
        {!hasDetails && <span style={{ width: 10 }} />}
        <span style={{ fontSize: 14 }}>🔧</span>
        <span style={{ fontWeight: 500, flex: 1 }}>{formatToolName(tool.name)}</span>
        <span style={{ color: statusConfig.color }}>
          {statusConfig.icon}
        </span>
        {tool.duration !== undefined && (
          <span style={{
            fontSize: 12,
            color: token.colorTextSecondary,
            marginLeft: 4,
          }}>
            {tool.duration.toFixed(1)}s
          </span>
        )}
        {tool.result?.latency_ms !== undefined && tool.duration === undefined && (
          <span style={{
            fontSize: 12,
            color: token.colorTextSecondary,
            marginLeft: 4,
          }}>
            {(tool.result.latency_ms / 1000).toFixed(1)}s
          </span>
        )}
      </div>

      {/* 展开的详情 */}
      {expanded && hasDetails && (
        <div style={{
          padding: '8px 12px',
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          background: token.colorBgContainer,
        }}>
          {/* 参数 */}
          {tool.params && Object.keys(tool.params).length > 0 && (
            <div style={{ marginBottom: 8 }}>
              <div style={{
                fontSize: 12,
                color: token.colorTextSecondary,
                marginBottom: 4,
              }}>
                参数:
              </div>
              <pre style={{
                margin: 0,
                padding: '8px 10px',
                background: token.colorBgTextHover,
                borderRadius: token.borderRadiusSM,
                fontSize: 12,
                overflow: 'auto',
                maxHeight: 120,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}>
                {JSON.stringify(tool.params, null, 2)}
              </pre>
            </div>
          )}

          {/* 结果 */}
          {tool.result && (
            <div>
              <div style={{
                fontSize: 12,
                color: token.colorTextSecondary,
                marginBottom: 4,
              }}>
                结果: {tool.result.ok ? (
                  <span style={{ color: token.colorSuccess }}>✅ 成功</span>
                ) : (
                  <span style={{ color: token.colorError }}>❌ 失败</span>
                )}
              </div>
              {tool.result.error && (
                <div style={{
                  padding: '8px 10px',
                  background: token.colorErrorBg,
                  borderRadius: token.borderRadiusSM,
                  fontSize: 12,
                  color: token.colorError,
                  marginBottom: 8,
                }}>
                  {tool.result.error}
                </div>
              )}
              {tool.result.data && (
                <pre style={{
                  margin: 0,
                  padding: '8px 10px',
                  background: token.colorBgTextHover,
                  borderRadius: token.borderRadiusSM,
                  fontSize: 12,
                  overflow: 'auto',
                  maxHeight: 200,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                }}>
                  {typeof tool.result.data === 'string'
                    ? tool.result.data
                    : JSON.stringify(tool.result.data, null, 2)}
                </pre>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * 获取状态配置
 */
function getStatusConfig(status: ToolStatus, token: any) {
  switch (status) {
    case 'running':
      return {
        icon: <LoadingOutlined spin />,
        text: '执行中',
        color: token.colorPrimary,
      };
    case 'success':
      return {
        icon: <CheckCircleOutlined />,
        text: '成功',
        color: token.colorSuccess,
      };
    case 'error':
      return {
        icon: <CloseCircleOutlined />,
        text: '失败',
        color: token.colorError,
      };
    default:
      return {
        icon: null,
        text: '',
        color: token.colorText,
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
