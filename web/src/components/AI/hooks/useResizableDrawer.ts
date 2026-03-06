import { useCallback, useEffect, useState } from 'react';
import type { DrawerWidthConfig } from '../types';

const DEFAULT_CONFIG: DrawerWidthConfig = {
  default: 520,
  min: 480,
  max: 800,
};

const STORAGE_KEY = 'ai-assistant-drawer-width';

/**
 * 可变宽度 Drawer Hook
 * 支持拖拽调整抽屉宽度，并持久化到 localStorage
 */
export function useResizableDrawer(config: Partial<DrawerWidthConfig> = {}) {
  const { default: defaultWidth, min, max } = { ...DEFAULT_CONFIG, ...config };

  // 从 localStorage 恢复宽度
  const getInitialWidth = useCallback(() => {
    if (typeof window === 'undefined') return defaultWidth;
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= min && parsed <= max) {
        return parsed;
      }
    }
    return defaultWidth;
  }, [defaultWidth, min, max]);

  const [width, setWidth] = useState(getInitialWidth);
  const [isResizing, setIsResizing] = useState(false);

  // 保存宽度到 localStorage
  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, String(width));
  }, [width]);

  // 开始拖拽
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  // 拖拽中
  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      // 使用 requestAnimationFrame 优化性能
      requestAnimationFrame(() => {
        const newWidth = window.innerWidth - e.clientX;
        const clampedWidth = Math.min(Math.max(newWidth, min), max);
        setWidth(clampedWidth);
      });
    },
    [isResizing, min, max]
  );

  // 结束拖拽
  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  // 绑定全局事件
  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      // 拖拽时禁用文本选择
      document.body.style.userSelect = 'none';
      document.body.style.cursor = 'ew-resize';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  // 重置宽度
  const resetWidth = useCallback(() => {
    setWidth(defaultWidth);
  }, [defaultWidth]);

  return {
    width,
    isResizing,
    handleMouseDown,
    resetWidth,
  };
}
