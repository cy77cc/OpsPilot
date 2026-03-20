import React from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { AssistantReply } from '../AssistantReply';
import CopilotSurface, { copyAssistantReplyToClipboard } from '../CopilotSurface';

const mockXMarkdown = vi.hoisted(() => vi.fn());
const mockUseXChat = vi.hoisted(() => vi.fn());
const mockUseXConversations = vi.hoisted(() => vi.fn());

vi.mock('@ant-design/x-sdk', async () => {
  const actual = await vi.importActual<typeof import('@ant-design/x-sdk')>('@ant-design/x-sdk');
  return {
    ...actual,
    useXChat: mockUseXChat,
    useXConversations: mockUseXConversations,
  };
});

vi.mock('@ant-design/x-markdown', () => ({
  default: (props: any) => {
    mockXMarkdown(props);
    return <div data-testid="x-markdown">{props.content}</div>;
  },
}));

vi.mock('../../../api/modules/ai', () => ({
  aiApi: {
    getSessions: vi.fn(async () => ({ data: [] })),
    getScenePrompts: vi.fn(async () => ({ data: { prompts: [] } })),
    getSession: vi.fn(async () => ({ data: { messages: [] } })),
    getRunProjection: vi.fn(async () => ({ data: null })),
    getRunContent: vi.fn(async () => ({ data: null })),
    createSession: vi.fn(),
    chatStream: vi.fn(),
  },
}));

vi.mock('../ToolResultCard', () => ({
  default: ({ activity }: { activity: { label: string; rawContent?: string } }) => (
    <div data-testid="tool-result-card">
      {activity.label}
      {activity.rawContent ? `:${activity.rawContent}` : null}
    </div>
  ),
}));

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

beforeEach(() => {
  vi.clearAllMocks();
  Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
    configurable: true,
    writable: true,
    value: vi.fn(),
  });
  Object.defineProperty(HTMLElement.prototype, 'offsetTop', {
    configurable: true,
    get() {
      return Number((this as HTMLElement).dataset.offsetTop || 0);
    },
  });
  Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
    configurable: true,
    get() {
      return Number((this as HTMLElement).dataset.offsetHeight || 0);
    },
  });
  vi.stubGlobal(
    'IntersectionObserver',
    class IntersectionObserver {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
});

describe('AssistantReply', () => {
  it('keeps streamed body visible until projection-backed content is ready, then swaps once', () => {
    mockUseXConversations.mockReturnValue({
      conversations: [{ key: 'sess-1', label: 'Session 1' }],
      activeConversationKey: 'sess-1',
      setActiveConversationKey: vi.fn(),
      addConversation: vi.fn(),
      setConversation: vi.fn(),
      setConversations: vi.fn(),
      getConversation: vi.fn(),
    });

    const firstMessageState = [
      {
        id: 'assistant-1',
        status: 'updating',
        message: {
          id: 'assistant-1',
          role: 'assistant',
          content: 'streamed body',
          runtime: {
            activities: [],
            status: { kind: 'streaming', label: '持续生成中' },
          },
        },
      },
    ];
    const projectionPendingState = [
      {
        id: 'assistant-1',
        status: 'success',
        message: {
          id: 'assistant-1',
          role: 'assistant',
          content: '回答内容不可恢复',
          runtime: {
            activities: [],
            status: { kind: 'error', label: 'projection missing summary' },
          },
        },
      },
    ];
    const projectionReadyState = [
      {
        id: 'assistant-1',
        status: 'success',
        message: {
          id: 'assistant-1',
          role: 'assistant',
          content: 'projection summary',
          runtime: {
            activities: [],
            summary: { title: '结论' },
            status: { kind: 'completed', label: 'completed' },
          },
        },
      },
    ];

    mockUseXChat.mockReturnValue({
      messages: firstMessageState,
      onRequest: vi.fn(),
      isRequesting: true,
      queueRequest: vi.fn(),
    });

    const view = render(
      <MemoryRouter initialEntries={['/services/payment-api']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId('x-markdown')).toHaveTextContent('streamed body');

    mockUseXChat.mockReturnValue({
      messages: projectionPendingState,
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });
    view.rerender(
      <MemoryRouter initialEntries={['/services/payment-api']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId('x-markdown')).toHaveTextContent('streamed body');
    expect(screen.queryByText('回答内容不可恢复')).not.toBeInTheDocument();

    mockUseXChat.mockReturnValue({
      messages: projectionReadyState,
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });
    view.rerender(
      <MemoryRouter initialEntries={['/services/payment-api']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId('x-markdown')).toHaveTextContent('projection summary');
    expect(screen.queryByText('streamed body')).not.toBeInTheDocument();
  });

  it('renders phase, inline tool activity, markdown body, and footer as one assistant reply', () => {
    render(
      <AssistantReply
        content="发现 2 个异常节点"
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '诊断助手正在巡检节点',
          activities: [
            { id: 'call-1', kind: 'tool', label: '正在获取节点状态', status: 'done' },
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

  it('does not render a summary card when projection-backed content already provides the final body', () => {
    render(
      <AssistantReply
        content="建议先处理 node-2 的磁盘压力。"
        status="success"
        runtime={{
          activities: [],
          summary: {
            title: '结论',
          },
          status: { kind: 'completed', label: '已生成' },
        }}
      />,
    );

    expect(screen.queryByText('结论')).not.toBeInTheDocument();
    expect(screen.getAllByText('建议先处理 node-2 的磁盘压力。')).toHaveLength(1);
  });

  it('keeps summary metrics inline without duplicating the final markdown body', () => {
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
              {
                id: 'plan-step-1',
                title: '批量执行健康检查',
                status: 'active',
                segments: [
                  { type: 'text', text: '正在执行 uptime ' },
                  { type: 'tool_ref', callId: 'call-2' },
                ],
              },
              { id: 'plan-step-2', title: '汇总检查结果', status: 'pending' },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool', label: 'host_list_inventory', detail: 'ok', status: 'done', stepIndex: 0 },
            { id: 'call-2', kind: 'tool', label: 'host_exec', detail: '正在执行 uptime', status: 'active', stepIndex: 1 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText('host_exec')).toBeInTheDocument();
    expect(screen.getByText('批量执行健康检查')).toBeInTheDocument();
    expect(screen.getByText(/已完成 1 个步骤/)).toBeInTheDocument();
    expect(screen.queryByText('汇总检查结果')).not.toBeInTheDocument();
    expect(screen.getAllByTestId('x-markdown').at(-1)).toHaveTextContent('## 最终报告');
  });

  it('renders one inline tool reference inside active step content without separate tool call or result rows', () => {
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
            { id: 'call-1', kind: 'tool', label: 'os_get_net_stat', detail: 'ok', rawContent: 'ok', status: 'done', stepIndex: 0 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText(/Let me start by gathering network statistics/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'os_get_net_stat' })).toBeInTheDocument();
    expect(screen.getByText('from the host.')).toBeInTheDocument();
    expect(screen.queryAllByText('os_get_net_stat')).toHaveLength(1);
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
            { id: 'call-1', kind: 'tool', label: 'host_list_inventory', detail: '执行中', status: 'active', stepIndex: 0 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText('获取服务器列表')).toBeInTheDocument();
    expect(screen.getByTestId('x-markdown')).toHaveTextContent('正在获取主机列表');
    expect(screen.queryByText('批量执行健康检查')).not.toBeInTheDocument();
    expect(screen.queryByText('汇总检查结果')).not.toBeInTheDocument();
    expect(screen.queryByText(/已完成/)).not.toBeInTheDocument();
  });

  it('renders tool-only active steps without requiring markdown body content', () => {
    render(
      <AssistantReply
        content=""
        status="updating"
        runtime={{
          phase: 'executing',
          phaseLabel: '执行工具',
          plan: {
            activeStepIndex: 0,
            steps: [
              {
                id: 'plan-step-0',
                title: '执行工具',
                status: 'active',
                segments: [
                  { type: 'tool_ref', callId: 'call-1' },
                ],
              },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool', label: 'k8s_query', status: 'active', stepIndex: 0 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getAllByText('执行工具')).toHaveLength(2);
    expect(screen.getByText('k8s_query')).toBeInTheDocument();
    expect(screen.queryByTestId('x-markdown')).not.toBeInTheDocument();
  });

  it('only exposes button semantics for terminal tool states', () => {
    const { rerender } = render(
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
                title: '采集信息',
                status: 'active',
                segments: [
                  { type: 'text', text: 'Checking cluster state ' },
                  { type: 'tool_ref', callId: 'call-1' },
                ],
              },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool', label: 'cluster_query', status: 'active', stepIndex: 0 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.queryByRole('button', { name: 'cluster_query' })).not.toBeInTheDocument();

    rerender(
      <AssistantReply
        content=""
        status="success"
        runtime={{
          phase: 'completed',
          phaseLabel: '执行完成',
          plan: {
            activeStepIndex: 0,
            steps: [
              {
                id: 'plan-step-0',
                title: '采集信息',
                status: 'active',
                segments: [
                  { type: 'text', text: 'Checking cluster state ' },
                  { type: 'tool_ref', callId: 'call-1' },
                ],
              },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool', label: 'cluster_query', status: 'done', rawContent: 'ok', stepIndex: 0 },
          ],
          status: { kind: 'completed', label: '已完成' },
        }}
      />,
    );

    expect(screen.getByRole('button', { name: 'cluster_query' })).toBeInTheDocument();
    expect(screen.queryAllByText('cluster_query')).toHaveLength(1);
  });

  it('labels interrupted and incomplete terminal tool states explicitly', () => {
    render(
      <AssistantReply
        content=""
        status="success"
        runtime={{
          phase: 'completed',
          phaseLabel: '执行完成',
          plan: {
            activeStepIndex: 0,
            steps: [
              {
                id: 'plan-step-0',
                title: '采集信息',
                status: 'active',
                segments: [
                  { type: 'tool_ref', callId: 'call-1' },
                  { type: 'tool_ref', callId: 'call-2' },
                ],
              },
            ],
          },
          activities: [
            { id: 'call-1', kind: 'tool', label: 'cluster_query', status: 'error', detail: '执行未完成', rawContent: 'partial', stepIndex: 0 },
            { id: 'call-2', kind: 'tool', label: 'cluster_exec', status: 'error', detail: '异常中断: timeout', rawContent: 'timeout', stepIndex: 0 },
          ],
          status: { kind: 'completed', label: '已完成' },
        }}
      />,
    );

    expect(screen.getByRole('button', { name: 'cluster_query（未完成）' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'cluster_exec（异常中断）' })).toBeInTheDocument();
  });

  it('can expand completed steps to see their content', () => {
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
            { id: 'call-1', kind: 'tool', label: 'health_check', detail: '执行中', status: 'active', stepIndex: 1 },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.queryByText('已找到 5 台服务器')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText(/已完成 1 个步骤/));
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

    expect(screen.getByText(/已完成 3 个步骤/)).toBeInTheDocument();
    expect(screen.queryByText('◐')).not.toBeInTheDocument();
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
            { id: 'call-1', kind: 'tool', label: 'host_list_inventory', status: 'done', rawContent: 'ok', stepIndex: 0 },
          ],
          status: { kind: 'completed', label: '已完成' },
        }}
      />,
    );

    expect(screen.getByText('最终报告内容')).toBeInTheDocument();
    expect(screen.queryAllByText('host_list_inventory')).toHaveLength(0);
  });

  it('copies only the final markdown body and excludes runtime text', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText,
      },
    });

    await copyAssistantReplyToClipboard(
      '最终结论：处理 node-2 的磁盘压力。',
      {
        activities: [
          { id: 'call-1', kind: 'tool', label: 'host_exec', detail: '异常中断', rawContent: 'stderr', status: 'error', stepIndex: 0 },
        ],
        summary: {
          title: '结论',
          items: [
            { label: '高风险', value: '1', tone: 'danger' },
          ],
        },
        status: { kind: 'completed', label: '已完成' },
      },
    );

    expect(writeText).toHaveBeenCalledWith('最终结论：处理 node-2 的磁盘压力。');
  });
});
