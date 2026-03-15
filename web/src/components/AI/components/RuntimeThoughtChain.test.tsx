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

  it('renders tool results as readable host rows before raw fallback', () => {
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

  it('renders replan body and structured step groups together', () => {
    render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'replan-1',
            kind: 'replan',
            title: '发现新信息，正在调整计划',
            status: 'done',
            headline: '执行结果触发重新规划',
            body: '集群查询结果返回的是 cluster 维度，需要切换到 host 维度继续。',
            structured: {
              steps: [
                { id: 'step-1', title: '改用 host_list_inventory', description: '读取主机列表与在线状态', status: 'pending' },
                { id: 'step-2', title: '输出汇总表格', description: '整理所有主机状态和统计信息', status: 'pending' },
              ],
            },
          },
        ]}
      />,
    );

    expect(screen.getByText('执行结果触发重新规划')).toBeInTheDocument();
    expect(screen.getByText('集群查询结果返回的是 cluster 维度，需要切换到 host 维度继续。')).toBeInTheDocument();
    expect(screen.getByText('改用 host_list_inventory')).toBeInTheDocument();
    expect(screen.getByText('输出汇总表格')).toBeInTheDocument();
  });
});
