import React from 'react';
import { Alert, Button, Space, Spin } from 'antd';
import type { ConfirmationRequest, RiskLevel } from '../types';

interface ConfirmationPanelProps {
  confirmation: ConfirmationRequest;
}

/**
 * 审批确认面板
 */
export function ConfirmationPanel({ confirmation }: ConfirmationPanelProps) {
  const riskConfig = getRiskConfig(confirmation.risk);
  const status = confirmation.status || 'waiting_user';
  const waiting = status === 'waiting_user';
  const submitting = status === 'submitting';
  const failed = status === 'failed';

  return (
    <div className={`ai-confirmation-panel ${status}`}>
      <div className="confirmation-header">
        <div className="confirmation-heading">
          <span className="confirmation-title">{confirmation.title}</span>
          <span className="confirmation-subtitle">
            {submitting ? '正在提交你的操作结果' : '该步骤会在确认后继续执行'}
          </span>
        </div>
        <Space size={8}>
          {submitting ? <Spin size="small" /> : null}
          <span className={`confirmation-risk ${confirmation.risk}`}>
            {riskConfig.label}
          </span>
        </Space>
      </div>

      <div className="confirmation-description">
        {confirmation.description}
      </div>

      {failed && confirmation.errorMessage ? (
        <Alert
          className="confirmation-alert"
          type="error"
          showIcon
          message={confirmation.errorMessage}
        />
      ) : null}

      {/* 详情预览 */}
      {confirmation.details && (
        <details className="confirmation-details">
          <summary>查看详情</summary>
          <pre className="confirmation-details-content">
            {JSON.stringify(confirmation.details, null, 2)}
          </pre>
        </details>
      )}

      <div className="confirmation-actions">
        <Button
          type={failed ? 'default' : 'primary'}
          aria-label={`${confirmation.title}，确认执行`}
          style={{ minHeight: 44 }}
          loading={submitting}
          disabled={submitting}
          onClick={() => confirmation.onConfirm()}
        >
          {failed ? '重试确认' : '确认执行'}
        </Button>
        <Button
          aria-label={`${confirmation.title}，取消`}
          style={{ minHeight: 44 }}
          disabled={submitting}
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
      return { label: '高风险', color: '#ff4d4f' };
    case 'medium':
      return { label: '中风险', color: '#faad14' };
    case 'low':
      return { label: '低风险', color: '#52c41a' };
    default:
      return { label: '未知风险', color: '#8c8c8c' };
  }
}
