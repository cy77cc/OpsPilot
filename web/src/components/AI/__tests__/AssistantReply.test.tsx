import React from 'react';
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AssistantReply } from '../AssistantReply';

const mockXMarkdown = vi.hoisted(() => vi.fn());

vi.mock('@ant-design/x-markdown', () => ({
  default: (props: any) => {
    mockXMarkdown(props);
    return <div data-testid="x-markdown">{props.content}</div>;
  },
}));

afterEach(() => {
  cleanup();
});

describe('AssistantReply', () => {
  it('renders phase, activities, markdown body, and footer as one assistant reply', () => {
    render(
      <AssistantReply
        content="发现 2 个异常节点"
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '诊断助手正在巡检节点',
          activities: [
            { id: 'call-1', kind: 'tool_call', label: '正在获取节点状态', status: 'done' },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText('诊断助手正在巡检节点')).toBeInTheDocument();
    expect(screen.getByText('正在获取节点状态')).toBeInTheDocument();
    expect(screen.getByText('持续生成中')).toBeInTheDocument();
    expect(screen.getByTestId('x-markdown')).toHaveTextContent('发现 2 个异常节点');
  });

  it('renders runtime summary inline without duplicating the markdown body', () => {
    render(
      <AssistantReply
        content="建议先处理 node-2 的磁盘压力。"
        status="success"
        runtime={{
          activities: [],
          summary: {
            title: '巡检摘要',
            items: [
              { label: '节点总数', value: '3' },
              { label: '高风险', value: '1', tone: 'danger' },
            ],
          },
          status: { kind: 'completed', label: '已生成' },
        }}
      />,
    );

    expect(screen.getByText('巡检摘要')).toBeInTheDocument();
    expect(screen.getByText('节点总数')).toBeInTheDocument();
    expect(screen.getByText('高风险')).toBeInTheDocument();
    expect(screen.getAllByText('建议先处理 node-2 的磁盘压力。')).toHaveLength(1);
  });

  it('renders plan steps as auto-expanded execution sections', () => {
    render(
      <AssistantReply
        content="## 最终报告"
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '执行第二步',
          plan: {
            activeStepIndex: 1,
            steps: [
              { id: 'plan-step-0', title: '获取服务器列表', status: 'done', content: '已找到 5 台服务器' },
              { id: 'plan-step-1', title: '批量执行健康检查', status: 'active', content: '正在执行 uptime\n' },
              { id: 'plan-step-2', title: '汇总检查结果', status: 'pending' },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool_result', label: 'host_list_inventory', detail: 'ok', stepIndex: 0 },
            { id: 'call-2', kind: 'tool_call', label: 'host_exec', detail: '正在执行 uptime', stepIndex: 1 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText('获取服务器列表')).toBeInTheDocument();
    expect(screen.getByText('批量执行健康检查')).toBeInTheDocument();
    expect(screen.getByText('host_exec')).toBeInTheDocument();
    expect(screen.getAllByTestId('x-markdown').at(0)).toHaveTextContent('正在执行 uptime');
    expect(screen.queryByText('host_list_inventory')).not.toBeInTheDocument();
    expect(screen.queryByText('汇总检查结果')).not.toBeInTheDocument();
    expect(screen.getAllByTestId('x-markdown').at(-1)).toHaveTextContent('## 最终报告');
  });

  it('only reveals the current step while hiding future steps', () => {
    render(
      <AssistantReply
        content=""
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '执行第一步',
          plan: {
            activeStepIndex: 0,
            steps: [
              { id: 'plan-step-0', title: '获取服务器列表', status: 'active', content: '正在获取主机列表' },
              { id: 'plan-step-1', title: '批量执行健康检查', status: 'pending' },
              { id: 'plan-step-2', title: '汇总检查结果', status: 'pending' },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool_call', label: 'host_list_inventory', detail: '执行中', stepIndex: 0 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getAllByText('获取服务器列表').at(-1)).toBeInTheDocument();
    expect(screen.getByText('host_list_inventory')).toBeInTheDocument();
    expect(screen.getByTestId('x-markdown')).toHaveTextContent('正在获取主机列表');
    expect(screen.queryByText('批量执行健康检查')).not.toBeInTheDocument();
    expect(screen.queryByText('汇总检查结果')).not.toBeInTheDocument();
  });
});
