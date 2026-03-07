import { useState, useEffect, useCallback } from 'react';
import { aiApi } from '../../../api/modules/ai';
import type { AIScenePromptItem } from '../../../api/modules/ai';

export interface PromptItem {
  key: string;
  description: string;
}

interface UseScenePromptsOptions {
  scene: string;
  enabled?: boolean;
}

interface UseScenePromptsResult {
  prompts: PromptItem[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

// 场景默认提示词 (fallback)
const SCENE_DEFAULT_PROMPTS: Record<string, PromptItem[]> = {
  'deployment:clusters': [
    { key: 'cluster-status', description: '查看所有集群状态' },
    { key: 'cluster-health', description: '帮我检查集群健康度' },
    { key: 'cluster-deploy', description: '部署应用到指定集群' },
  ],
  'deployment:hosts': [
    { key: 'host-list', description: '查看主机列表' },
    { key: 'host-health', description: '执行主机健康检查' },
    { key: 'host-exec', description: '在主机上执行命令' },
  ],
  'services:list': [
    { key: 'service-catalog', description: '查看服务目录' },
    { key: 'service-search', description: '搜索服务' },
    { key: 'service-create', description: '创建新服务' },
  ],
  'services:catalog': [
    { key: 'service-catalog', description: '查看服务目录' },
    { key: 'service-search', description: '搜索服务' },
    { key: 'service-create', description: '创建新服务' },
  ],
  'deployment:metrics': [
    { key: 'metrics-view', description: '查看监控指标' },
    { key: 'alerts-list', description: '查看告警列表' },
    { key: 'performance-analyze', description: '分析服务性能' },
  ],
  global: [
    { key: 'help', description: '有什么可以帮助你的？' },
  ],
};

/**
 * 场景提示词 Hook
 * 根据当前场景获取快捷指令提示
 */
export function useScenePrompts(options: UseScenePromptsOptions): UseScenePromptsResult {
  const { scene, enabled = true } = options;

  const [prompts, setPrompts] = useState<PromptItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPrompts = useCallback(async () => {
    if (!enabled || !scene) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await aiApi.getScenePrompts(scene);
      if (res.data?.prompts && res.data.prompts.length > 0) {
        setPrompts(
          res.data.prompts.map((p: AIScenePromptItem, index: number) => ({
            key: `prompt-${p.id || index}`,
            description: p.prompt_text,
          }))
        );
      } else {
        // 使用默认提示词
        setPrompts(getDefaultPrompts(scene));
      }
    } catch (err) {
      console.warn('Failed to load scene prompts, using defaults:', err);
      setPrompts(getDefaultPrompts(scene));
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [scene, enabled]);

  // 场景变化时重新加载
  useEffect(() => {
    loadPrompts();
  }, [loadPrompts]);

  return {
    prompts,
    loading,
    error,
    refresh: loadPrompts,
  };
}

/**
 * 获取场景默认提示词
 */
function getDefaultPrompts(scene: string): PromptItem[] {
  // 标准化场景名称
  const normalizedScene = scene.replace(/^scene:/, '').toLowerCase();

  // 查找精确匹配
  if (SCENE_DEFAULT_PROMPTS[normalizedScene]) {
    return SCENE_DEFAULT_PROMPTS[normalizedScene];
  }

  // 查找前缀匹配
  for (const [key, prompts] of Object.entries(SCENE_DEFAULT_PROMPTS)) {
    if (normalizedScene.startsWith(key) || key.startsWith(normalizedScene)) {
      return prompts;
    }
  }

  // 返回通用提示词
  return SCENE_DEFAULT_PROMPTS.global;
}
