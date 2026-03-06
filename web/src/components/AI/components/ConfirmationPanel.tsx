import React from 'react';
import { Button, Alert } from 'antd';
import type { ConfirmationRequest, RiskLevel } from '../types';

interface ConfirmationPanelProps {
  confirmation: ConfirmationRequest;
}

/**
 * 审批确认面板
 */
export function ConfirmationPanel({ confirmation }: ConfirmationPanelProps) {
  const riskConfig = getRiskConfig(confirmation.risk);

  return (
    <div className="ai-confirmation-panel">
      <div className="confirmation-header">
        <span className="confirmation-title">{confirmation.title}</span>
        <span className={`confirmation-risk ${confirmation.risk}`}>
          {riskConfig.label}
        </span>
      </div>

      <div className="confirmation-description">
        {confirmation.description}
      </div>

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
          type="primary"
          onClick={() => confirmation.onConfirm()}
        >
          确认执行
        </Button>
        <Button
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
