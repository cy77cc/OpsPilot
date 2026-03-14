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
              {node.summary ? <div className="runtime-chain__nodeSummary">{node.summary}</div> : null}
              {Array.isArray(node.details) && node.details.length > 0 ? (
                <div className="runtime-chain__nodeBody">
                  {node.details.map((detail, index) => (
                    <div key={`${node.nodeId}:${index}`} style={{ fontSize: 12, lineHeight: 1.6 }}>
                      {typeof detail === 'string' ? detail : JSON.stringify(detail)}
                    </div>
                  ))}
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
