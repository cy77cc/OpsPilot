import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { RuntimeThoughtChain } from './RuntimeThoughtChain';

vi.mock('@ant-design/x', () => ({
  ThoughtChain: ({
    items,
  }: {
    items: Array<{ key: string; title: string; description?: string; content?: React.ReactNode }>;
  }) => (
    <div>
      {items.map((item) => (
        <div key={item.key}>
          <div>{item.title}</div>
          {item.description ? <div>{item.description}</div> : null}
          <div>{item.content}</div>
        </div>
      ))}
    </div>
  ),
}));

describe('RuntimeThoughtChain', () => {
  it('renders tool nodes with structured rows', () => {
    render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'tool-1',
            kind: 'tool',
            title: 'host_list_inventory',
            status: 'done',
            headline: '已获取 2 台主机',
            structured: {
              resource: 'hosts',
              rows: [
                { id: 1, name: 'test', status: 'online', ip: 'localhost' },
                { id: 2, name: 'VM-1', status: 'offline', ip: '172.22.0.2' },
              ],
            },
            raw: {
              total: 2,
            },
          },
        ]}
      />,
    );

    expect(screen.getByText('已获取 2 台主机')).toBeInTheDocument();
    expect(screen.getByText('test')).toBeInTheDocument();
    expect(screen.getByText('online')).toBeInTheDocument();
    expect(screen.getByText('localhost')).toBeInTheDocument();
    expect(screen.getByText('VM-1')).toBeInTheDocument();
    expect(screen.getByText('offline')).toBeInTheDocument();
    expect(screen.queryByText('"total": 2')).not.toBeInTheDocument();
  });

  it('renders approval nodes with confirmation panel', () => {
    render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'approval-1',
            kind: 'approval',
            title: '扩容 nginx 需要确认',
            status: 'waiting',
            headline: '该步骤会修改工作负载副本数',
            approval: {
              id: 'approval-1',
              title: '扩容 nginx 需要确认',
              description: '该步骤会修改工作负载副本数',
              risk: 'high',
              status: 'waiting_user',
              toolName: 'scale_deployment',
              toolDisplayName: '扩容 Deployment',
              planId: 'plan-1',
              stepId: 'step-1',
              checkpointId: 'cp-1',
              argumentsJson: '{"replicas":3}',
              editable: true,
            },
          },
        ]}
      />,
    );

    // Check for title (may appear multiple times - use getAllByText)
    expect(screen.getAllByText('扩容 nginx 需要确认').length).toBeGreaterThan(0);
    // Description may also appear multiple times
    expect(screen.getAllByText('该步骤会修改工作负载副本数').length).toBeGreaterThan(0);
    // ConfirmationPanel should render confirm/cancel buttons
    expect(screen.getByRole('button', { name: /确认执行/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /取消/ })).toBeInTheDocument();
  });

  it('renders structured plan steps without dumping raw JSON', () => {
    render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'plan-1',
            kind: 'plan',
            title: '整理执行步骤',
            status: 'done',
            headline: '已生成执行计划',
            structured: {
              steps: [
                { id: 'step-1', title: '检查集群状态', description: '读取当前节点和 deployment 信息', status: 'pending' },
                { id: 'step-2', title: '确认副本数', description: '准备评估扩容影响', tool_hint: 'scale_deployment' },
              ],
            },
            raw: {
              steps: ['检查集群状态', '确认副本数'],
            },
          },
        ]}
      />,
    );

    expect(screen.getByText('已生成执行计划')).toBeInTheDocument();
    expect(screen.getByText('检查集群状态')).toBeInTheDocument();
    expect(screen.getByText('读取当前节点和 deployment 信息')).toBeInTheDocument();
    expect(screen.getByText('确认副本数')).toBeInTheDocument();
    expect(screen.queryByText(/^\{.*"title":/)).not.toBeInTheDocument();
  });

  it('renders multiple tool nodes in sequence', () => {
    render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'tool-1',
            kind: 'tool',
            title: 'get_pods',
            status: 'done',
            headline: '已获取 Pod 列表',
          },
          {
            nodeId: 'tool-2',
            kind: 'tool',
            title: 'get_deployments',
            status: 'done',
            headline: '已获取 Deployment 列表',
          },
        ]}
      />,
    );

    expect(screen.getByText('已获取 Pod 列表')).toBeInTheDocument();
    expect(screen.getByText('已获取 Deployment 列表')).toBeInTheDocument();
  });

  it('returns null for empty nodes', () => {
    const { container } = render(
      <RuntimeThoughtChain nodes={[]} />,
    );

    expect(container.firstChild).toBeNull();
  });

  it('applies collapsed class when isCollapsed is true', () => {
    const { container } = render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'tool-1',
            kind: 'tool',
            title: 'get_pods',
            status: 'done',
            headline: '已获取 Pod 列表',
          },
        ]}
        isCollapsed
      />,
    );

    expect(container.firstChild).toHaveClass('runtime-chain--collapsed');
  });
});
