import React from 'react';
import { cleanup, render, screen, fireEvent } from '@testing-library/react';
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

  it('shows completed steps in collapsible section and active step expanded', () => {
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

    // Active step (index 1) is expanded
    expect(screen.getByText('host_exec')).toBeInTheDocument();
    expect(screen.getByText('批量执行健康检查')).toBeInTheDocument();

    // Completed steps are in collapse header
    expect(screen.getByText(/已完成 1 个步骤/)).toBeInTheDocument();

    // Pending step is not shown
    expect(screen.queryByText('汇总检查结果')).not.toBeInTheDocument();

    // Final report content
    expect(screen.getAllByTestId('x-markdown').at(-1)).toHaveTextContent('## 最终报告');
  });

  it('renders tool calls inline inside active step content while keeping tool results separate', () => {
    render(
      <AssistantReply
        content=""
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '执行中',
          plan: {
            activeStepIndex: 0,
            steps: [
              {
                id: 'plan-step-0',
                title: '采集主机网络指标',
                status: 'active',
                segments: [
                  { type: 'text', text: 'Let me start by gathering network statistics ' },
                  { type: 'tool_ref', callId: 'call-1' },
                  { type: 'text', text: 'from the host.' },
                ],
              },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool_call', label: 'os_get_net_stat', status: 'active', stepIndex: 0 },
            { id: 'call-1:result', kind: 'tool_result', label: 'os_get_net_stat', detail: 'ok', rawContent: 'ok', status: 'done', stepIndex: 0 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText(/Let me start by gathering network statistics/)).toBeInTheDocument();
    expect(screen.getByText('os_get_net_stat')).toBeInTheDocument();
    expect(screen.getByText('from the host.')).toBeInTheDocument();
    expect(screen.queryByText('ok')).not.toBeInTheDocument();
  });

  it('shows only active step when no completed steps', () => {
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

    // Only active step is shown
    expect(screen.getByText('获取服务器列表')).toBeInTheDocument();
    expect(screen.getByText('host_list_inventory')).toBeInTheDocument();
    expect(screen.getByTestId('x-markdown')).toHaveTextContent('正在获取主机列表');

    // Pending steps are not shown
    expect(screen.queryByText('批量执行健康检查')).not.toBeInTheDocument();
    expect(screen.queryByText('汇总检查结果')).not.toBeInTheDocument();

    // No completed steps collapse
    expect(screen.queryByText(/已完成/)).not.toBeInTheDocument();
  });

  it('can expand completed steps to see their content', async () => {
    render(
      <AssistantReply
        content=""
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '执行第二步',
          plan: {
            activeStepIndex: 1,
            steps: [
              { id: 'plan-step-0', title: '获取服务器列表', status: 'done', content: '已找到 5 台服务器' },
              { id: 'plan-step-1', title: '批量执行健康检查', status: 'active', content: '正在执行检查' },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool_call', label: 'health_check', detail: '执行中', stepIndex: 1 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    // Completed step content is initially hidden
    expect(screen.queryByText('已找到 5 台服务器')).not.toBeInTheDocument();

    // Click to expand completed steps
    const collapseHeader = screen.getByText(/已完成 1 个步骤/);
    fireEvent.click(collapseHeader);

    // Now completed step content is visible
    expect(screen.getByText('已找到 5 台服务器')).toBeInTheDocument();
    expect(screen.getByText('获取服务器列表')).toBeInTheDocument();
  });

  it('shows completed steps collapse when plan is finished', () => {
    render(
      <AssistantReply
        content="最终报告内容"
        status="success"
        runtime={{
          phase: 'completed',
          phaseLabel: '执行完成',
          plan: {
            activeStepIndex: undefined,
            steps: [
              { id: 'plan-step-0', title: '获取服务器列表', status: 'done', content: '已找到 5 台服务器' },
              { id: 'plan-step-1', title: '批量执行健康检查', status: 'done', content: '检查完成' },
              { id: 'plan-step-2', title: '汇总检查结果', status: 'done', content: '汇总完成' },
            ],
          },
          activities: [],
          status: { kind: 'completed', label: '已完成' },
        }}
      />,
    );

    // Completed steps are in collapse
    expect(screen.getByText(/已完成 3 个步骤/)).toBeInTheDocument();

    // No active step
    expect(screen.queryByText('◐')).not.toBeInTheDocument();

    // Final report content is shown
    expect(screen.getByText('最终报告内容')).toBeInTheDocument();
  });

  it('does not re-list historical tool activities above the final markdown when plan is finished', () => {
    render(
      <AssistantReply
        content="最终报告内容"
        status="success"
        runtime={{
          phase: 'completed',
          phaseLabel: '执行完成',
          plan: {
            activeStepIndex: undefined,
            steps: [
              {
                id: 'plan-step-0',
                title: '获取服务器列表',
                status: 'done',
                segments: [
                  { type: 'text', text: '正在获取主机列表 ' },
                  { type: 'tool_ref', callId: 'call-1' },
                ],
              },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool_call', label: 'host_list_inventory', status: 'done', stepIndex: 0 },
            { id: 'call-1:result', kind: 'tool_result', label: 'host_list_inventory', status: 'done', rawContent: 'ok', stepIndex: 0 },
          ],
          status: { kind: 'completed', label: '已完成' },
        }}
      />,
    );

    expect(screen.getByText('最终报告内容')).toBeInTheDocument();
    expect(screen.queryAllByText('host_list_inventory')).toHaveLength(0);
  });
});
