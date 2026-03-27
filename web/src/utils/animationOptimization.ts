/**
 * 动画性能优化工具
 *
 * 提供一系列优化动画性能的工具函数和配置
 */

/**
 * 使用 will-change 优化动画性能
 * 在动画开始前添加，动画结束后移除
 */
export const optimizeAnimation = (element: HTMLElement, properties: string[]) => {
  element.style.willChange = properties.join(', ');

  return () => {
    element.style.willChange = 'auto';
  };
};

/**
 * 检测是否支持硬件加速
 */
export const supportsHardwareAcceleration = (): boolean => {
  const testElement = document.createElement('div');
  testElement.style.transform = 'translateZ(0)';
  return testElement.style.transform !== '';
};

/**
 * 强制使用 GPU 加速
 */
export const forceGPUAcceleration = (element: HTMLElement) => {
  element.style.transform = 'translateZ(0)';
  element.style.backfaceVisibility = 'hidden';
  element.style.perspective = '1000px';
};

/**
 * 防抖动画触发
 */
export const debounceAnimation = (callback: () => void, delay: number = 16) => {
  let timeoutId: number;

  return () => {
    clearTimeout(timeoutId);
    timeoutId = window.setTimeout(callback, delay);
  };
};

/**
 * 使用 requestAnimationFrame 优化动画
 */
export const rafThrottle = (callback: () => void) => {
  let rafId: number | null = null;

  return () => {
    if (rafId !== null) return;

    rafId = requestAnimationFrame(() => {
      callback();
      rafId = null;
    });
  };
};

/**
 * 检测用户是否偏好减少动画
 */
export const prefersReducedMotion = (): boolean => {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
};

/**
 * 根据用户偏好返回动画配置
 */
export const getAnimationConfig = (defaultConfig: any) => {
  if (prefersReducedMotion()) {
    return {
      ...defaultConfig,
      duration: 0.01,
      transition: { duration: 0.01 },
    };
  }
  return defaultConfig;
};

/**
 * Framer Motion 性能优化配置
 */
export const optimizedMotionConfig = {
  // 使用 transform 和 opacity 进行动画（GPU 加速）
  layoutId: undefined,
  // 减少重绘
  style: {
    willChange: 'transform, opacity',
  },
  // 使用硬件加速
  transformTemplate: ({ x, y, rotate }: any) => {
    return `translate3d(${x}, ${y}, 0) rotate(${rotate})`;
  },
};

/**
 * 批量动画优化
 * 将多个动画合并到一个 requestAnimationFrame 中
 */
export class AnimationBatcher {
  private queue: Array<() => void> = [];
  private rafId: number | null = null;

  add(callback: () => void) {
    this.queue.push(callback);
    this.scheduleFlush();
  }

  private scheduleFlush() {
    if (this.rafId !== null) return;

    this.rafId = requestAnimationFrame(() => {
      this.flush();
    });
  }

  private flush() {
    const callbacks = this.queue.slice();
    this.queue = [];
    this.rafId = null;

    callbacks.forEach((callback) => callback());
  }
}

/**
 * 全局动画批处理器实例
 */
export const animationBatcher = new AnimationBatcher();

/**
 * 性能监控
 */
export const measureAnimationPerformance = (name: string, callback: () => void) => {
  const start = performance.now();
  callback();
  const end = performance.now();
  const duration = end - start;

  if (duration > 16.67) {
    // 超过一帧的时间（60fps）
    console.warn(`Animation "${name}" took ${duration.toFixed(2)}ms (> 16.67ms)`);
  }

  return duration;
};

/**
 * 延迟加载动画
 * 只在元素进入视口时才启用动画
 */
export const useIntersectionAnimation = (
  element: HTMLElement | null,
  callback: () => void,
  options?: IntersectionObserverInit
) => {
  if (!element) return;

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          callback();
          observer.disconnect();
        }
      });
    },
    { threshold: 0.1, ...options }
  );

  observer.observe(element);

  return () => observer.disconnect();
};
