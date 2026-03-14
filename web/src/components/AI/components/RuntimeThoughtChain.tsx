import React from 'react';
import { ThoughtChain } from '@ant-design/x';
import { theme } from 'antd';
import { ConfirmationPanel } from './ConfirmationPanel';
import type { ConfirmationRequest, RuntimeThoughtChainNode } from '../types';
import './RuntimeChain.css';

interface RuntimeThoughtChainProps {
  nodes: RuntimeThoughtChainNode[];
  isCollapsed?: boolean;
  onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean) => void;
}

function asStructuredSteps(structured: Record<string, unknown> | undefined): Array<Record<string, unknown>> {
  const steps = structured?.steps;
  return Array.isArray(steps)
    ? steps.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object')
    : [];
}

function renderDetail(detail: unknown, key: string) {
  if (typeof detail === 'string') {
    return <div key={key} className="runtime-chain__detailLine">{detail}</div>;
  }
  if (!detail || typeof detail !== 'object') {
    return <div key={key} className="runtime-chain__detailLine">{String(detail)}</div>;
  }

  const record = detail as Record<string, unknown>;
  const title = String(record.title || record.label || '').trim();
  const secondary = String(record.content || record.summary || record.description || '').trim();
  const hint = String(record.tool_hint || record.status || '').trim();

  return (
    <div key={key} className="runtime-chain__detailCard">
      {title ? <div className="runtime-chain__detailTitle">{title}</div> : null}
      {secondary ? <div className="runtime-chain__detailText">{secondary}</div> : null}
      {hint ? <div className="runtime-chain__detailMeta">{hint}</div> : null}
    </div>
  );
}

function renderStructuredStep(step: Record<string, unknown>, key: string) {
  const title = String(step.title || step.label || step.id || '').trim();
  const secondary = String(step.description || step.content || step.summary || '').trim();
  const hint = String(step.tool_hint || step.status || '').trim();

  return (
    <div key={key} className="runtime-chain__detailCard runtime-chain__detailCard--step">
      {title ? <div className="runtime-chain__detailTitle">{title}</div> : null}
      {secondary ? <div className="runtime-chain__detailText">{secondary}</div> : null}
      {hint ? <div className="runtime-chain__detailMeta">{hint}</div> : null}
    </div>
  );
}

function toConfirmation(node: RuntimeThoughtChainNode, onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean) => void): ConfirmationRequest | null {
  if (!node.approval) {
    return null;
  }
  return {
    ...node.approval,
    onConfirm: () => onApprovalDecision?.(node.approval?.details || {}, true),
    onCancel: () => onApprovalDecision?.(node.approval?.details || {}, false),
  };
}

export function RuntimeThoughtChain({ nodes, isCollapsed = false, onApprovalDecision }: RuntimeThoughtChainProps) {
  const { token } = theme.useToken();
  if (nodes.length === 0) {
    return null;
  }
  return (
    <div className={`runtime-chain ${isCollapsed ? 'runtime-chain--collapsed' : 'runtime-chain--expanded'}`}>
      {isCollapsed ? (
        <div className="runtime-chain__collapsed-title" style={{ color: token.colorTextSecondary }}>
          思考完成
        </div>
      ) : null}
      <ThoughtChain
        items={nodes.map((node) => ({
          key: node.nodeId,
          title: isCollapsed ? (node.title || '思考完成') : node.title,
          status: node.status === 'done' ? 'success' : node.status === 'error' ? 'error' : 'loading',
          content: (
            <div className={`runtime-chain__node is-${node.status}`} style={{ color: token.colorText }}>
              {node.headline || node.summary ? <div className="runtime-chain__nodeSummary">{node.headline || node.summary}</div> : null}
              {node.body ? <div className="runtime-chain__nodeBodyText">{node.body}</div> : null}
              {asStructuredSteps(node.structured).length > 0 ? (
                <div className="runtime-chain__nodeBody">
                  {asStructuredSteps(node.structured).map((step, index) => renderStructuredStep(step, `${node.nodeId}:step:${index}`))}
                </div>
              ) : null}
              {Array.isArray(node.details) && node.details.length > 0 ? (
                <div className="runtime-chain__nodeBody">
                  {node.details.map((detail, index) => renderDetail(detail, `${node.nodeId}:${index}`))}
                </div>
              ) : null}
              {node.kind === 'approval' ? (
                <div className="runtime-chain__approvalRow">
                  {(() => {
                    const confirmation = toConfirmation(node, onApprovalDecision);
                    return confirmation ? <ConfirmationPanel confirmation={confirmation} /> : null;
                  })()}
                </div>
              ) : null}
            </div>
          ),
        }))}
        defaultExpandedKeys={isCollapsed ? [] : nodes.map((node) => node.nodeId)}
      />
    </div>
  );
}

export default RuntimeThoughtChain;
