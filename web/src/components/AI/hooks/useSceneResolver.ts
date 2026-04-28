import { useLocation } from 'react-router-dom';
import type { SceneContext } from '../types';

interface SceneRule {
  patterns: RegExp[];
  scene: string;
  resourceType: string;
  extractResourceId?: (segments: string[]) => string | undefined;
}

const SCENE_REGISTRY: SceneRule[] = [
  {
    patterns: [/^\/resources\/hosts/, /^\/hosts/],
    scene: 'host',
    resourceType: 'host',
    extractResourceId: (segments) => segments[segments.length - 1],
  },
  {
    patterns: [/^\/k8s-legacy/],
    scene: 'k8s',
    resourceType: 'k8s',
  },
  {
    patterns: [/^\/resources\/clusters/, /^\/k8s/],
    scene: 'cluster',
    resourceType: 'cluster',
    extractResourceId: (segments) => segments[segments.length - 1],
  },
  {
    patterns: [/^\/delivery\/services/],
    scene: 'service',
    resourceType: 'service',
    extractResourceId: (segments) => segments[1],
  },
];

const DEFAULT_SCENE = { scene: 'ai', resourceType: 'page' };

export function resolveScene(pathname: string): { scene: string; context: SceneContext } {
  const normalized = pathname || '/';
  const segments = normalized.split('/').filter(Boolean);

  for (const rule of SCENE_REGISTRY) {
    if (rule.patterns.some((p) => p.test(normalized))) {
      return {
        scene: rule.scene,
        context: {
          route: normalized,
          resourceType: rule.resourceType,
          resourceId: rule.extractResourceId?.(segments),
        },
      };
    }
  }

  return {
    scene: DEFAULT_SCENE.scene,
    context: {
      route: normalized,
      resourceType: DEFAULT_SCENE.resourceType,
    },
  };
}

export function useSceneResolver(): { scene: string; context: SceneContext } {
  const location = useLocation();
  return resolveScene(location.pathname);
}
