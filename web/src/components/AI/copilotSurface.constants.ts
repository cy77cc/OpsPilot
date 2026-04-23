import type { PromptsItemType } from '@ant-design/x';
import { createEmptyAssistantRuntime } from './replyRuntime';
import { extractPendingRunFromMessage } from './providers/runReconnectController';
import { upsertPendingRun } from './pendingRunStore';
import type { AssistantReplyRuntime, AssistantReplyStatusKind } from './types';

export const NEW_SESSION_KEY = '__new__';

/**
 * mapHistoryMessageStatus 将后端消息状态映射为 Bubble 组件状态。
 *
 * 后端状态值:
 *   - 'done': 正常完成
 *   - 'error': 流式处理出错
 *   - 'streaming': 正在流式传输
 *   - 其他: 兜底为 loading
 *
 * Bubble 状态值:
 *   - 'success': 完成成功
 *   - 'error': 出错
 *   - 'abort': 中断/取消
 *   - 'loading': 加载中
 */
export function mapHistoryMessageStatus(status?: string): 'success' | 'error' | 'abort' | 'loading' {
  switch (status) {
    case 'done':
      return 'success';
    case 'error':
      return 'error';
    case 'interrupted':
      return 'abort';
    case 'streaming':
      return 'loading';
    default:
      return 'loading';
  }
}

export function buildHistoricalPendingRuntime(
  runtime: AssistantReplyRuntime | undefined,
  message: Record<string, any>,
): AssistantReplyRuntime | undefined {
  const pendingRun = extractPendingRunFromMessage(message);
  if (!pendingRun) {
    return runtime;
  }

  upsertPendingRun(pendingRun);

  const statusLabel = pendingRun.status === 'resuming'
    ? '恢复中'
    : '执行中';
  const statusKind: AssistantReplyStatusKind =
    pendingRun.status === 'running'
      ? 'streaming'
      : pendingRun.status || 'streaming';

  return {
    ...(runtime || createEmptyAssistantRuntime()),
    pendingRun,
    phase: 'executing',
    phaseLabel: statusLabel,
    status: {
      kind: statusKind,
      label: statusLabel,
    },
  };
}

export const SCENE_FALLBACK_PROMPTS: Record<string, PromptsItemType[]> = {
  host: [
    { key: 'host-health', label: '诊断主机健康', description: '帮我检查当前主机的健康状态和风险点' },
    { key: 'host-services', label: '检查服务状态', description: '分析这台主机上的关键服务状态' },
  ],
  cluster: [
    { key: 'cluster-health', label: '诊断集群健康', description: '帮我分析当前集群的健康状态和关键异常' },
    { key: 'cluster-capacity', label: '评估集群容量', description: '评估当前集群的资源容量与潜在瓶颈' },
  ],
  service: [
    { key: 'service-release', label: '发布影响分析', description: '分析当前服务最近发布的潜在影响' },
    { key: 'service-deps', label: '服务依赖梳理', description: '梳理这个服务的依赖与潜在故障点' },
  ],
  k8s: [
    { key: 'k8s-workload', label: '工作负载巡检', description: '检查当前 Kubernetes 工作负载的异常情况' },
    { key: 'k8s-events', label: '事件总结', description: '总结当前 Kubernetes 事件流里的异常信号' },
  ],
  ai: [
    { key: 'ai-general', label: '开始提问', description: '描述你当前遇到的问题或你想完成的操作' },
  ],
};
