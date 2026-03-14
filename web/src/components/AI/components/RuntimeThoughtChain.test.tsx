import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { RuntimeThoughtChain } from './RuntimeThoughtChain';

vi.mock('@ant-design/x', () => ({
  ThoughtChain: ({
    items,
  }: {
    items: Array<{ key: string; title: string; content?: React.ReactNode }>;
  }) => (
    <div>
      {items.map((item) => (
        <div key={item.key}>
          <div>{item.title}</div>
          <div>{item.content}</div>
        </div>
      ))}
    </div>
  ),
}));

describe('RuntimeThoughtChain', () => {
  it('renders structured node details without dumping raw JSON', () => {
    render(
      <RuntimeThoughtChain
        nodes={[
          {
            nodeId: 'plan-1',
            kind: 'plan',
            title: '整理执行步骤',
            status: 'done',
            summary: '已生成执行计划',
            details: [
              { title: '检查集群状态', content: '读取当前节点和 deployment 信息', status: 'pending' },
              { title: '确认副本数', summary: '准备评估扩容影响', tool_hint: 'scale_deployment' },
            ],
          },
        ]}
      />,
    );

    expect(screen.getByText('检查集群状态')).toBeInTheDocument();
    expect(screen.getByText('读取当前节点和 deployment 信息')).toBeInTheDocument();
    expect(screen.getByText('确认副本数')).toBeInTheDocument();
    expect(screen.queryByText(/^\{.*"title":/)).not.toBeInTheDocument();
  });
});
