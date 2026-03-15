import React, { useMemo } from 'react';
import { ThoughtChain } from '@ant-design/x';
import { theme } from 'antd';
import { ConfirmationPanel } from './ConfirmationPanel';
import type { ConfirmationRequest, RuntimeThoughtChainNode } from '../types';
import './RuntimeChain.css';

interface RuntimeThoughtChainProps {
  nodes: RuntimeThoughtChainNode[];
  isCollapsed?: boolean;
  onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean, editedArgs?: string, reason?: string) => void;
}

function asStructuredSteps(structured: Record<string, unknown> | undefined): Array<Record<string, unknown>> {
  const steps = structured?.steps;
  return Array.isArray(steps)
    ? steps.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object')
    : [];
}

function asStructuredRows(structured: Record<string, unknown> | undefined): Array<Record<string, unknown>> {
  const rows = structured?.rows;
  return Array.isArray(rows)
    ? rows.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object')
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

function renderToolRow(row: Record<string, unknown>, key: string) {
  const title = String(row.name || row.hostname || row.id || '').trim();
  const status = String(row.status || '').trim();
  const secondary = [row.ip, row.hostname].filter(Boolean).map((value) => String(value).trim()).filter(Boolean).join(' / ');

  return (
    <div key={key} className="runtime-chain__detailCard runtime-chain__detailCard--toolRow">
      {title ? <div className="runtime-chain__detailTitle">{title}</div> : null}
      {secondary ? <div className="runtime-chain__detailText">{secondary}</div> : null}
      {status ? <div className="runtime-chain__detailMeta">{status}</div> : null}
    </div>
  );
}

function toConfirmation(node: RuntimeThoughtChainNode, onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean, editedArgs?: string, reason?: string) => void): ConfirmationRequest | null {
  if (!node.approval) {
    return null;
  }
  const approvalPayload: Record<string, unknown> = {
    ...node.approval,
    // Ensure canonical fields are at the top level for resume identity
    plan_id: node.approval.planId,
    step_id: node.approval.stepId,
    checkpoint_id: node.approval.checkpointId,
    target: node.approval.target,
    tool_name: node.approval.toolName,
    tool_display_name: node.approval.toolDisplayName,
    arguments_json: node.approval.argumentsJson,
  };
  return {
    ...node.approval,
    onConfirm: (editedArgs?: string) => onApprovalDecision?.(approvalPayload, true, editedArgs),
    onCancel: (reason?: string) => onApprovalDecision?.(approvalPayload, false, undefined, reason),
  };
}

/**
 * 判断body是否为JSON片段（如 {"steps":, }, ["xxx"] 等）
 * 这些片段不应该直接显示给用户
 */
function isJsonFragment(body: string): boolean {
  const trimmed = body.trim();
  if (!trimmed) return false;
  // JSON对象片段: {, }, {"key":, "value"}
  if (/^[{}\[\]",:]/.test(trimmed)) return true;
  if (/^\\"/.test(trimmed)) return true; // 转义引号开头
  // 纯JSON数组片段: ["xxx", "yyy"]
  if (/^\[.*\]$/.test(trimmed) && trimmed.includes('\\"')) return true;
  return false;
}

/**
 * 获取可显示的body文本
 * 过滤掉JSON片段，只返回有意义的文本
 */
function getDisplayBody(node: RuntimeThoughtChainNode): string | null {
  const body = node.body?.trim();
  if (!body) return null;
  // 如果是JSON片段，不显示
  if (isJsonFragment(body)) return null;
  return body;
}

export function RuntimeThoughtChain({ nodes, isCollapsed = false, onApprovalDecision }: RuntimeThoughtChainProps) {
  const { token } = theme.useToken();

  // 计算所有节点ID
  const allNodeIds = useMemo(() => nodes.map((n) => n.nodeId), [nodes]);

  // 非受控模式：使用 defaultExpandedKeys
  // isCollapsed=true 时默认折叠所有节点，false 时默认展开所有节点
  // 但用户可以手动展开/折叠
  const defaultExpandedKeys = isCollapsed ? [] : allNodeIds;

  if (nodes.length === 0) {
    return null;
  }

  // 是否有实际内容需要展示（除了 description 之外）
  const hasExtraContent = (node: RuntimeThoughtChainNode): boolean => {
    const displayBody = getDisplayBody(node);
    return Boolean(
      displayBody ||
      asStructuredSteps(node.structured).length > 0 ||
      (node.kind === 'tool' && node.structured?.resource === 'hosts' && asStructuredRows(node.structured).length > 0) ||
      (Array.isArray(node.details) && node.details.length > 0) ||
      node.kind === 'approval'
    );
  };

  return (
    <div className={`runtime-chain ${isCollapsed ? 'runtime-chain--collapsed' : 'runtime-chain--expanded'}`}>
      <ThoughtChain
        items={nodes.map((node) => {
          const displayBody = getDisplayBody(node);
          const description = node.headline || node.summary;
          const showContent = hasExtraContent(node);

          return {
            key: node.nodeId,
            title: node.title || '思考完成',
            description: description || undefined,
            status: node.status === 'done' ? 'success' : node.status === 'error' ? 'error' : 'loading',
            collapsible: showContent, // 只有有额外内容时才可折叠
            content: showContent ? (
              <div className={`runtime-chain__node is-${node.status}`} style={{ color: token.colorText }}>
                {displayBody ? <div className="runtime-chain__nodeBodyText">{displayBody}</div> : null}
              {asStructuredSteps(node.structured).length > 0 ? (
                <div className="runtime-chain__nodeBody">
                  {asStructuredSteps(node.structured).map((step, index) => renderStructuredStep(step, `${node.nodeId}:step:${index}`))}
                </div>
              ) : null}
              {node.kind === 'tool' && node.structured?.resource === 'hosts' && asStructuredRows(node.structured).length > 0 ? (
                <div className="runtime-chain__nodeBody">
                  {asStructuredRows(node.structured).map((row, index) => renderToolRow(row, `${node.nodeId}:row:${index}`))}
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
            ) : undefined,
          };
        })}
        defaultExpandedKeys={defaultExpandedKeys}
      />
    </div>
  );
}

export default RuntimeThoughtChain;
