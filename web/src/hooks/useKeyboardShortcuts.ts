import { useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

interface KeyboardShortcutsOptions {
  onOpenHelp?: () => void;
  enableNavigation?: boolean;
  enableListNavigation?: boolean;
}

/**
 * 全局键盘快捷键 Hook
 *
 * 支持的快捷键:
 * - / : 聚焦搜索框
 * - Esc : 关闭弹窗/取消操作
 * - ? : 打开快捷键帮助
 * - g+h : 跳转到首页
 * - g+s : 跳转到服务管理
 * - g+d : 跳转到部署管理
 * - g+m : 跳转到监控中心
 * - g+c : 跳转到配置中心
 * - g+t : 跳转到任务中心
 * - j : 列表向下
 * - k : 列表向上
 * - Enter : 选择当前项
 * - Space : 切换选中状态
 */
export const useKeyboardShortcuts = (options: KeyboardShortcutsOptions = {}) => {
  const {
    onOpenHelp,
    enableNavigation = true,
    enableListNavigation = false,
  } = options;

  const navigate = useNavigate();
  const [gPressed, setGPressed] = React.useState(false);

  // 导航快捷键
  const handleNavigation = useCallback(
    (key: string) => {
      if (!enableNavigation) return false;

      const navigationMap: Record<string, string> = {
        h: '/',
        s: '/services',
        d: '/deployment',
        m: '/monitoring',
        c: '/config',
        t: '/tasks',
      };

      if (gPressed && navigationMap[key]) {
        navigate(navigationMap[key]);
        setGPressed(false);
        return true;
      }

      return false;
    },
    [gPressed, navigate, enableNavigation]
  );

  // 列表导航快捷键
  const handleListNavigation = useCallback(
    (e: KeyboardEvent) => {
      if (!enableListNavigation) return false;

      // 检查是否在输入框中
      const target = e.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable
      ) {
        return false;
      }

      switch (e.key) {
        case 'j': {
          // 向下导航
          const focusable = document.querySelectorAll('[data-list-item]');
          const current = document.activeElement;
          const currentIndex = Array.from(focusable).indexOf(current as Element);
          const next = focusable[currentIndex + 1] as HTMLElement;
          if (next) {
            next.focus();
            next.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
          }
          return true;
        }
        case 'k': {
          // 向上导航
          const focusable = document.querySelectorAll('[data-list-item]');
          const current = document.activeElement;
          const currentIndex = Array.from(focusable).indexOf(current as Element);
          const prev = focusable[currentIndex - 1] as HTMLElement;
          if (prev) {
            prev.focus();
            prev.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
          }
          return true;
        }
        case 'Enter': {
          // 选择当前项
          const current = document.activeElement as HTMLElement;
          if (current.hasAttribute('data-list-item')) {
            current.click();
            return true;
          }
          return false;
        }
        case ' ': {
          // 切换选中状态
          e.preventDefault();
          const current = document.activeElement as HTMLElement;
          if (current.hasAttribute('data-list-item')) {
            const checkbox = current.querySelector('input[type="checkbox"]') as HTMLInputElement;
            if (checkbox) {
              checkbox.click();
              return true;
            }
          }
          return false;
        }
        default:
          return false;
      }
    },
    [enableListNavigation]
  );

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // 检查是否在输入框中
      const target = e.target as HTMLElement;
      const isInInput =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable;

      // Cmd+K / Ctrl+K - 命令面板 (已在 AppLayout 中处理)
      // 这里不需要处理，避免重复

      // / - 聚焦搜索框
      if (e.key === '/' && !isInInput) {
        e.preventDefault();
        const searchInput = document.querySelector('input[placeholder*="搜索"]') as HTMLInputElement;
        if (searchInput) {
          searchInput.focus();
        }
        return;
      }

      // Esc - 关闭弹窗/取消操作
      if (e.key === 'Escape') {
        // 失焦当前元素
        if (document.activeElement instanceof HTMLElement) {
          document.activeElement.blur();
        }
        return;
      }

      // ? - 打开快捷键帮助
      if (e.key === '?' && !isInInput && onOpenHelp) {
        e.preventDefault();
        onOpenHelp();
        return;
      }

      // g - 导航前缀键
      if (e.key === 'g' && !isInInput) {
        e.preventDefault();
        setGPressed(true);
        // 2秒后自动重置
        setTimeout(() => setGPressed(false), 2000);
        return;
      }

      // g+[key] - 导航快捷键
      if (handleNavigation(e.key)) {
        e.preventDefault();
        return;
      }

      // 列表导航快捷键
      if (handleListNavigation(e)) {
        e.preventDefault();
        return;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleNavigation, handleListNavigation, onOpenHelp]);

  return {
    gPressed,
  };
};

// React import for useState
import React from 'react';
