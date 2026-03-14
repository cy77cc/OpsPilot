import React from 'react';
import { Button, Space, Spin, theme } from 'antd';
import type { ConfirmationRequest, RiskLevel } from '../types';

interface ConfirmationPanelProps {
  confirmation: ConfirmationRequest;
}

/**
 * 审批确认面板
 */
export function ConfirmationPanel({ confirmation }: ConfirmationPanelProps) {
  const { token } = theme.useToken();
  const riskConfig = getRiskConfig(confirmation.risk);
  const status = confirmation.status || 'waiting_user';
  const waiting = status === 'waiting_user';
  const submitting = status === 'submitting';
  const failed = status === 'failed';

  if (submitting) {
    return (
      <div className="ai-confirmation-compact submitting">
        <Space size={10}>
          <Spin size="small" />
          <div>
            <div className="confirmation-compact-title">正在提交确认</div>
            <div className="confirmation-compact-text">审批结果已发送，正在继续处理当前步骤。</div>
          </div>
        </Space>
      </div>
    );
  }

  if (failed) {
    return (
      <div className="ai-confirmation-compact failed">
        <div className="confirmation-compact-copy">
          <div className="confirmation-compact-title">审批提交失败</div>
          <div className="confirmation-compact-text">
            {confirmation.errorMessage || '审批结果提交失败，请重试。'}
          </div>
        </div>
        <div className="confirmation-compact-actions">
          <Button type="link" onClick={() => confirmation.onConfirm()}>
            重试
          </Button>
          <Button type="text" onClick={() => confirmation.onCancel()}>
            取消
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`ai-confirmation-panel ${status}`}
      style={{
        borderColor: token.colorWarningBorder,
        background: `linear-gradient(180deg, ${token.colorWarningBg} 0%, ${token.colorBgElevated} 100%)`,
        boxShadow: token.boxShadowTertiary,
      }}
    >
      <div className="confirmation-header">
        <div className="confirmation-heading">
          <span className="confirmation-kicker">执行前确认</span>
          <span className="confirmation-title">{confirmation.title}</span>
          <span className="confirmation-subtitle">
            该步骤会在确认后继续执行
          </span>
        </div>
        <Space size={8}>
          <span
            className={`confirmation-risk ${confirmation.risk}`}
            style={{
              color: riskConfig.color,
              background: token.colorBgContainer,
              borderColor: riskConfig.borderColor,
            }}
          >
            {riskConfig.label}
          </span>
        </Space>
      </div>

      <div className="confirmation-description">
        {confirmation.description}
      </div>

      {confirmation.details && (
        <details
          className="confirmation-details"
          style={{
            background: token.colorBgContainer,
            borderColor: token.colorBorderSecondary,
          }}
        >
          <summary>查看详情</summary>
          <pre
            className="confirmation-details-content"
            style={{
              background: token.colorFillQuaternary,
              color: token.colorTextSecondary,
            }}
          >
            {JSON.stringify(confirmation.details, null, 2)}
          </pre>
        </details>
      )}

      <div className="confirmation-actions">
        <Button
          type="primary"
          aria-label={`${confirmation.title}，确认执行`}
          style={{ minHeight: 44 }}
          onClick={() => confirmation.onConfirm()}
        >
          确认执行
        </Button>
        <Button
          aria-label={`${confirmation.title}，取消`}
          style={{ minHeight: 44 }}
          onClick={() => confirmation.onCancel()}
        >
          取消
        </Button>
      </div>
    </div>
  );
}

/**
 * 风险等级配置
 */
function getRiskConfig(risk: RiskLevel) {
  switch (risk) {
    case 'high':
      return { label: '高风险', color: '#cf1322', borderColor: 'rgba(207, 19, 34, 0.24)' };
    case 'medium':
      return { label: '中风险', color: '#d48806', borderColor: 'rgba(212, 136, 6, 0.24)' };
    case 'low':
      return { label: '低风险', color: '#389e0d', borderColor: 'rgba(56, 158, 13, 0.24)' };
    default:
      return { label: '未知风险', color: '#8c8c8c', borderColor: 'rgba(140, 140, 140, 0.24)' };
  }
}
