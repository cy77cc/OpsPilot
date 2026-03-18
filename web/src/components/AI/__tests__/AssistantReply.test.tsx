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

    // All step titles should be visible
    expect(screen.getByText(/获取服务器列表/)).toBeInTheDocument();
    expect(screen.getByText(/批量执行健康检查/)).toBeInTheDocument();
    expect(screen.getByText(/汇总检查结果/)).toBeInTheDocument();

    // Only active step (index 1) is expanded
    expect(screen.getByText('host_exec')).toBeInTheDocument();
    expect(screen.getAllByTestId('x-markdown').at(0)).toHaveTextContent('正在执行 uptime');

    // Step 0 (done) is collapsed, so its activities are hidden
    expect(screen.queryByText('host_list_inventory')).not.toBeInTheDocument();

    // Final report content
    expect(screen.getAllByTestId('x-markdown').at(-1)).toHaveTextContent('## 最终报告');

    // Status icons
    expect(screen.getByText(/✓ 获取服务器列表/)).toBeInTheDocument(); // done
    expect(screen.getByText(/◐ 批量执行健康检查/)).toBeInTheDocument(); // active
    expect(screen.getByText(/○ 汇总检查结果/)).toBeInTheDocument(); // pending
  });

  it('shows all steps but only expands the current active step', () => {
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

    // All step titles should be visible
    expect(screen.getByText(/获取服务器列表/)).toBeInTheDocument();
    expect(screen.getByText(/批量执行健康检查/)).toBeInTheDocument();
    expect(screen.getByText(/汇总检查结果/)).toBeInTheDocument();

    // Only the active step should be expanded (showing content and activities)
    expect(screen.getByText('host_list_inventory')).toBeInTheDocument();
    expect(screen.getByTestId('x-markdown')).toHaveTextContent('正在获取主机列表');

    // Status icons should be present
    expect(screen.getByText(/◐ 获取服务器列表/)).toBeInTheDocument(); // active
    expect(screen.getByText(/○ 批量执行健康检查/)).toBeInTheDocument(); // pending
    expect(screen.getByText(/○ 汇总检查结果/)).toBeInTheDocument(); // pending
  });

  it('shows done icon for completed steps and expands only active step', () => {
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
              { id: 'plan-step-2', title: '汇总检查结果', status: 'pending' },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool_call', label: 'health_check', detail: '执行中', stepIndex: 1 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    // All step titles visible with correct status icons
    expect(screen.getByText(/✓ 获取服务器列表/)).toBeInTheDocument(); // done
    expect(screen.getByText(/◐ 批量执行健康检查/)).toBeInTheDocument(); // active
    expect(screen.getByText(/○ 汇总检查结果/)).toBeInTheDocument(); // pending

    // Only active step content is expanded
    expect(screen.getByText('health_check')).toBeInTheDocument();
    expect(screen.getByText('正在执行检查')).toBeInTheDocument();

    // Done step content should NOT be expanded (collapsed)
    expect(screen.queryByText('已找到 5 台服务器')).not.toBeInTheDocument();
  });

  it('collapses all steps when activeStepIndex is undefined (is_final=true)', () => {
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

    // All step titles should be visible with done icons
    expect(screen.getByText(/✓ 获取服务器列表/)).toBeInTheDocument();
    expect(screen.getByText(/✓ 批量执行健康检查/)).toBeInTheDocument();
    expect(screen.getByText(/✓ 汇总检查结果/)).toBeInTheDocument();

    // No step content should be expanded (all collapsed)
    expect(screen.queryByText('已找到 5 台服务器')).not.toBeInTheDocument();
    expect(screen.queryByText('检查完成')).not.toBeInTheDocument();
    expect(screen.queryByText('汇总完成')).not.toBeInTheDocument();

    // Final report content is still shown
    expect(screen.getByText('最终报告内容')).toBeInTheDocument();
  });
});
