import { useCallback, useRef } from 'react';

/**
 * 防抖 Hook
 *
 * 用于搜索输入等场景，减少不必要的 API 请求
 */
export const useDebounce = <T extends (...args: any[]) => any>(
  callback: T,
  delay: number = 300
) => {
  const timeoutRef = useRef<any>(null);

  return useCallback(
    (...args: Parameters<T>) => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      timeoutRef.current = setTimeout(() => {
        callback(...args);
      }, delay);
    },
    [callback, delay]
  );
};

/**
 * 节流 Hook
 *
 * 用于滚动、resize 等高频事件
 */
export const useThrottle = <T extends (...args: any[]) => any>(
  callback: T,
  delay: number = 300
) => {
  const lastRunRef = useRef<number>(0);
  const timeoutRef = useRef<any>(null);

  return useCallback(
    (...args: Parameters<T>) => {
      const now = Date.now();

      if (now - lastRunRef.current >= delay) {
        callback(...args);
        lastRunRef.current = now;
      } else {
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }

        timeoutRef.current = setTimeout(() => {
          callback(...args);
          lastRunRef.current = Date.now();
        }, delay - (now - lastRunRef.current));
      }
    },
    [callback, delay]
  );
};
