import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { getSceneByPath, getSceneLabel } from '../constants/sceneMapping';
import type { SceneInfo } from '../types';

/**
 * 场景检测 Hook
 * 根据当前路由自动检测对应的场景
 */
export function useSceneDetector(): SceneInfo {
  const location = useLocation();

  const sceneInfo = useMemo(() => {
    const config = getSceneByPath(location.pathname);

    if (config) {
      return {
        key: config.key,
        label: getSceneLabel(config.key),
        description: `当前场景：${config.label}`,
      };
    }

    // 无场景映射，返回全局场景
    return {
      key: 'global',
      label: '全局助手',
      description: '通用 AI 助手',
    };
  }, [location.pathname]);

  return sceneInfo;
}

/**
 * 检查当前路由是否有场景支持
 */
export function useHasSceneSupport(): boolean {
  const { key } = useSceneDetector();
  return key !== 'global';
}

/**
 * 获取场景配置
 */
export function useSceneConfig() {
  const sceneInfo = useSceneDetector();
  const hasSceneSupport = sceneInfo.key !== 'global';

  return {
    ...sceneInfo,
    hasSceneSupport,
  };
}
