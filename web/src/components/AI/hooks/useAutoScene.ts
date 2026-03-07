import { useMemo, useState, useCallback, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { getSceneByPath, getSceneLabel, SCENE_MAPPINGS } from '../constants/sceneMapping';

export interface SceneOption {
  key: string;
  label: string;
}

const STORAGE_KEY = 'ai-copilot-scene';

/**
 * 自动场景检测与切换 Hook
 * - 自动检测当前路由对应场景
 * - 支持手动切换场景
 * - 持久化场景选择
 */
export function useAutoScene() {
  const location = useLocation();
  const [manualScene, setManualScene] = useState<string | null>(null);

  // 从 localStorage 恢复场景选择
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      setManualScene(saved);
    }
  }, []);

  // 自动检测的场景
  const detectedScene = useMemo(() => {
    const config = getSceneByPath(location.pathname);
    return config?.key || 'global';
  }, [location.pathname]);

  // 当前场景：手动选择 > 自动检测
  const scene = manualScene || detectedScene;

  // 可用场景列表（包含"自动"选项）
  const availableScenes = useMemo<SceneOption[]>(() => {
    const scenes: SceneOption[] = [];

    // 添加"自动检测"选项
    scenes.push({
      key: '__auto__',
      label: detectedScene === 'global' ? '自动 (全局)' : `自动 (${getSceneLabel(detectedScene)})`,
    });

    // 添加全局助手
    scenes.push({ key: 'global', label: '全局助手' });

    // 添加所有场景（去重）
    const seenKeys = new Set(['__auto__', 'global']);
    for (const config of SCENE_MAPPINGS) {
      if (!seenKeys.has(config.key)) {
        scenes.push({
          key: config.key,
          label: config.label,
        });
        seenKeys.add(config.key);
      }
    }

    return scenes;
  }, [detectedScene]);

  // 切换场景
  const setScene = useCallback((newScene: string) => {
    if (newScene === '__auto__') {
      // 选择"自动"模式
      setManualScene(null);
      localStorage.removeItem(STORAGE_KEY);
    } else {
      setManualScene(newScene);
      localStorage.setItem(STORAGE_KEY, newScene);
    }
  }, []);

  // 重置为自动检测
  const resetScene = useCallback(() => {
    setManualScene(null);
    localStorage.removeItem(STORAGE_KEY);
  }, []);

  // 用于 Select 的值（如果是自动模式，显示 __auto__）
  const selectValue = manualScene || '__auto__';

  return useMemo(
    () => ({
      scene,
      selectValue,
      setScene,
      resetScene,
      detectedScene,
      manualScene,
      availableScenes,
      isAuto: !manualScene,
    }),
    [scene, selectValue, setScene, resetScene, detectedScene, manualScene, availableScenes]
  );
}
