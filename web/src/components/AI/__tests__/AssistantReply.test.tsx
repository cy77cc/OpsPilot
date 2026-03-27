import React from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AssistantReply } from '../AssistantReply';
import CopilotSurface, { copyAssistantReplyToClipboard } from '../CopilotSurface';
import { aiApi } from '../../../api/modules/ai';

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

vi.mock('../../../api/modules/ai', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../api/modules/ai')>();
  actual.aiApi.getSessions = vi.fn(async () => ({ success: true, data: [] }));
  actual.aiApi.getScenePrompts = vi.fn(async () => ({ success: true, data: { scene: 'default', prompts: [] } }));
  actual.aiApi.getSession = vi.fn(async () => ({
    success: true,
    data: {
      id: 'session-1',
      title: 'Session 1',
      messages: [],
      createdAt: new Date(0).toISOString(),
      updatedAt: new Date(0).toISOString(),
    },
  }));
  actual.aiApi.getRunProjection = vi.fn(async () => ({ success: true, data: null }));
  actual.aiApi.getRunContent = vi.fn(async () => ({ success: true, data: null }));
  actual.aiApi.createSession = vi.fn();
  actual.aiApi.chatStream = vi.fn();
  actual.aiApi.submitApproval = vi.fn();
  actual.aiApi.getApproval = vi.fn();
  actual.aiApi.listPendingApprovals = vi.fn();
  return {
    ...actual,
    aiApi: actual.aiApi,
  };
});

vi.mock('../ToolResultCard', () => ({
  default: ({ activity }: { activity: { label: string; rawContent?: string } }) => (
    <div data-testid="tool-result-card">
      {activity.label}
      {activity.rawContent ? `:${activity.rawContent}` : null}
    </div>
  ),
}));

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

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

  it('delays approval-triggered conversation refresh to avoid stale snapshots', () => {
    const setTimeoutSpy = vi.spyOn(window, 'setTimeout');

    mockUseXConversations.mockReturnValue({
      conversations: [{ key: 'sess-1', label: 'Session 1' }],
      activeConversationKey: 'sess-1',
      setActiveConversationKey: vi.fn(),
      addConversation: vi.fn(),
      setConversation: vi.fn(),
      setConversations: vi.fn(),
      getConversation: vi.fn(),
    });

    mockUseXChat.mockReturnValue({
      messages: [],
      setMessages: vi.fn(),
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/services/payment-api']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    window.dispatchEvent(new CustomEvent('ai-approval-updated', {
      detail: { token: 'approval-1', status: 'approved' },
    }));

    const hasApprovalDelay = setTimeoutSpy.mock.calls.some((call) => call[1] === 1200);
    expect(hasApprovalDelay).toBe(true);
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

  it('normalizes escaped line breaks before rendering markdown body', () => {
    render(
      <AssistantReply
        content={'## Local 集群概览\\n\\n共有 21 个 Pod'}
        status="success"
      />,
    );

    expect(mockXMarkdown).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '## Local 集群概览\n\n共有 21 个 Pod',
      }),
    );
  });

  it('passes code renderer to XMarkdown for fenced code highlighting', () => {
    render(
      <AssistantReply
        content={'```python\nprint("hello")\n```'}
        status="success"
      />,
    );

    expect(mockXMarkdown).toHaveBeenCalledWith(
      expect.objectContaining({
        components: expect.objectContaining({
          code: expect.any(Function),
        }),
      }),
    );
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

  it('renders task board items from runtime.todos', () => {
    render(
      <AssistantReply
        content="正在生成任务板"
        status="updating"
        runtime={{
          todos: [
            {
              id: 'todo-1',
              content: '检查节点',
              activeForm: '正在检查节点',
              status: 'in_progress',
            },
            {
              id: 'todo-2',
              content: '汇总结果',
              status: 'pending',
            },
          ],
          activities: [],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    expect(screen.getByText('Task Board')).toBeInTheDocument();
    expect(screen.getByText('检查节点')).toBeInTheDocument();
    expect(screen.getByText('正在检查节点')).toBeInTheDocument();
    expect(screen.getByText('汇总结果')).toBeInTheDocument();
    expect(screen.getByText('进行中')).toBeInTheDocument();
    expect(screen.getByText('待办')).toBeInTheDocument();
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
    expect(screen.getByText('获取服务器列表')).toBeInTheDocument();
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
    fireEvent.click(screen.getByRole('button', { name: /获取服务器列表/ }));
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

    expect(screen.getByText('获取服务器列表')).toBeInTheDocument();
    expect(screen.getByText('批量执行健康检查')).toBeInTheDocument();
    expect(screen.getByText('汇总检查结果')).toBeInTheDocument();
    expect(screen.queryByText('◐')).not.toBeInTheDocument();
    expect(screen.getByText('最终报告内容')).toBeInTheDocument();
  });

  it('loads historical step content using stable mapping metadata', async () => {
    const onLoadStepContent = vi.fn().mockResolvedValue({
      content: '执行完成',
      segments: [{ type: 'text', text: '执行完成' }],
      activities: [],
    });

    render(
      <AssistantReply
        content="历史总结"
        status="success"
        runtime={{
          activities: [],
          plan: {
            steps: [
              { id: 'historical-step-1', title: '执行检查', status: 'done', loaded: false, sourceBlockIndex: 4 } as any,
            ],
          },
          status: { kind: 'completed', label: '已完成' },
        }}
        onLoadStepContent={onLoadStepContent}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /执行检查/ }));

    expect(onLoadStepContent).toHaveBeenCalledWith('historical-step-1', 4);
  });

  it('keeps historical step load failure local and recovers on retry', async () => {
    const onLoadStepContent = vi.fn()
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce({
        content: '恢复后的内容',
        segments: [{ type: 'text', text: '恢复后的内容' }],
        activities: [],
      });

    render(
      <AssistantReply
        content="历史总结"
        status="success"
        runtime={{
          activities: [],
          summary: { title: '结论' },
          plan: {
            steps: [
              { id: 'historical-step-1', title: '执行检查', status: 'done', loaded: false } as any,
            ],
          },
          status: { kind: 'completed', label: '已完成' },
        }}
        onLoadStepContent={onLoadStepContent}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /执行检查/ }));

    expect(await screen.findByText('加载失败')).toBeInTheDocument();
    expect(screen.getByText('历史总结')).toBeInTheDocument();

    fireEvent.click(screen.getByText('重试'));

    expect(onLoadStepContent).toHaveBeenCalledTimes(2);
    expect(await screen.findByText('恢复后的内容')).toBeInTheDocument();
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

  it('submits approval once when user clicks confirm repeatedly', async () => {
    vi.mocked(aiApi.submitApproval).mockResolvedValueOnce({
      success: true,
      data: {
        approval_id: 'approval-1',
        status: 'approved',
      },
    } as any);

    render(
      <AssistantReply
        content=""
        status="updating"
        runtime={{
          activities: [
            {
              id: 'call-1',
              kind: 'tool_approval',
              label: 'kubectl_apply',
              detail: '等待审批 300s',
              status: 'pending',
              approvalId: 'approval-1',
            },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    const approveButton = screen.getByRole('button', { name: /批\s*准/ });

    await act(async () => {
      fireEvent.click(approveButton);
      fireEvent.click(approveButton);
    });

    await waitFor(() => {
      expect(aiApi.submitApproval).toHaveBeenCalledTimes(1);
    });
    expect(aiApi.submitApproval).toHaveBeenCalledWith(
      'approval-1',
      { approved: true },
      expect.objectContaining({
        idempotencyKey: expect.any(String),
      }),
    );
  });

  it('falls back to a timeout state when approval submission stalls', async () => {
    vi.useFakeTimers();
    try {
      const deferred = createDeferred<any>();
      vi.mocked(aiApi.submitApproval).mockReturnValueOnce(deferred.promise);

      render(
        <AssistantReply
          content=""
          status="updating"
          runtime={{
            activities: [
              {
                id: 'call-1',
                kind: 'tool_approval',
                label: 'kubectl_apply',
                detail: '等待审批 300s',
                status: 'pending',
                approvalId: 'approval-1',
              },
            ],
            status: { kind: 'streaming', label: '持续生成中' },
          }}
        />,
      );

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /批\s*准/ }));
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(15000);
      });

      expect(screen.getByText(/提交超时，请刷新后重试/)).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /批\s*准/ })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /拒\s*绝/ })).not.toBeInTheDocument();

      await act(async () => {
        deferred.resolve({
          success: true,
          data: {
            approval_id: 'approval-1',
            status: 'approved',
          },
        });
        await Promise.resolve();
      });

      expect(screen.getByText(/提交超时，请刷新后重试/)).toBeInTheDocument();
      expect(screen.queryByText('已批准')).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows a refresh-needed failure state when approval submission fails', async () => {
    vi.mocked(aiApi.submitApproval).mockRejectedValueOnce(new Error('network down'));

    render(
      <AssistantReply
        content=""
        status="updating"
        runtime={{
          activities: [
            {
              id: 'call-1',
              kind: 'tool_approval',
              label: 'kubectl_apply',
              detail: '等待审批 300s',
              status: 'pending',
              approvalId: 'approval-1',
            },
          ],
          status: { kind: 'streaming', label: '持续生成中' },
        }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /批\s*准/ }));

    expect(screen.getByText('提交中')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText(/network down/)).toBeInTheDocument();
      expect(screen.getByText(/需刷新/)).toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: /批\s*准/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /拒\s*绝/ })).not.toBeInTheDocument();
  });

  it('defers mounting large markdown bodies until they enter the viewport', async () => {
    const callbacks: Array<(entries: Array<{ isIntersecting: boolean }>) => void> = [];
    class IntersectionObserverMock {
      readonly callback: (entries: Array<{ isIntersecting: boolean }>) => void;
      constructor(callback: (entries: Array<{ isIntersecting: boolean }>) => void) {
        this.callback = callback;
        callbacks.push(callback);
      }
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    (globalThis as any).IntersectionObserver = IntersectionObserverMock;
    mockXMarkdown.mockClear();

    const largeContent = Array.from({ length: 210 }, (_, index) => `line ${index + 1}`).join('\n');

    render(
      <AssistantReply
        content={largeContent}
        status="success"
      />,
    );

    expect(mockXMarkdown).not.toHaveBeenCalled();

    await act(async () => {
      callbacks[0]?.([{ isIntersecting: true }]);
      await Promise.resolve();
    });

    expect(mockXMarkdown).toHaveBeenCalled();
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

  it('normalizes escaped line breaks before copying markdown body', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText,
      },
    });

    await copyAssistantReplyToClipboard('## Local 集群概览\\n\\n共有 21 个 Pod');

    expect(writeText).toHaveBeenCalledWith('## Local 集群概览\n\n共有 21 个 Pod');
  });
});
