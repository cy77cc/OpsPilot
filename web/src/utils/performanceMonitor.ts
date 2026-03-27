/**
 * 性能监控工具
 *
 * 使用 Performance API 监控应用性能
 */

interface PerformanceMetrics {
  // 页面加载性能
  pageLoad: {
    dns: number;
    tcp: number;
    request: number;
    response: number;
    domParse: number;
    domContentLoaded: number;
    loadComplete: number;
    firstPaint?: number;
    firstContentfulPaint?: number;
    largestContentfulPaint?: number;
  };
  // 资源加载性能
  resources: Array<{
    name: string;
    type: string;
    duration: number;
    size: number;
  }>;
  // 自定义性能标记
  marks: Map<string, number>;
  measures: Map<string, number>;
}

class PerformanceMonitor {
  private marks: Map<string, number> = new Map();
  private measures: Map<string, number> = new Map();

  /**
   * 获取页面加载性能指标
   */
  getPageLoadMetrics() {
    if (!window.performance || !window.performance.timing) {
      return null;
    }

    const timing = window.performance.timing;
    const navigation = timing.navigationStart;

    const metrics = {
      dns: timing.domainLookupEnd - timing.domainLookupStart,
      tcp: timing.connectEnd - timing.connectStart,
      request: timing.responseStart - timing.requestStart,
      response: timing.responseEnd - timing.responseStart,
      domParse: timing.domInteractive - timing.domLoading,
      domContentLoaded: timing.domContentLoadedEventEnd - navigation,
      loadComplete: timing.loadEventEnd - navigation,
    };

    // 获取 Paint Timing
    if (window.performance.getEntriesByType) {
      const paintEntries = window.performance.getEntriesByType('paint');
      paintEntries.forEach((entry: any) => {
        if (entry.name === 'first-paint') {
          (metrics as any).firstPaint = entry.startTime;
        } else if (entry.name === 'first-contentful-paint') {
          (metrics as any).firstContentfulPaint = entry.startTime;
        }
      });

      // 获取 LCP (Largest Contentful Paint)
      const lcpEntries = window.performance.getEntriesByType('largest-contentful-paint');
      if (lcpEntries.length > 0) {
        const lastEntry = lcpEntries[lcpEntries.length - 1] as any;
        (metrics as any).largestContentfulPaint = lastEntry.startTime;
      }
    }

    return metrics;
  }

  /**
   * 获取资源加载性能
   */
  getResourceMetrics() {
    if (!window.performance || !window.performance.getEntriesByType) {
      return [];
    }

    const resources = window.performance.getEntriesByType('resource') as PerformanceResourceTiming[];

    return resources.map((resource) => ({
      name: resource.name,
      type: resource.initiatorType,
      duration: resource.duration,
      size: resource.transferSize || 0,
    }));
  }

  /**
   * 标记性能时间点
   */
  mark(name: string): void {
    const timestamp = performance.now();
    this.marks.set(name, timestamp);

    if (window.performance && window.performance.mark) {
      window.performance.mark(name);
    }
  }

  /**
   * 测量两个标记之间的时间
   */
  measure(name: string, startMark: string, endMark?: string): number {
    const startTime = this.marks.get(startMark);
    if (!startTime) {
      console.warn(`Start mark "${startMark}" not found`);
      return 0;
    }

    const endTime = endMark ? this.marks.get(endMark) : performance.now();
    if (endMark && !endTime) {
      console.warn(`End mark "${endMark}" not found`);
      return 0;
    }

    const duration = (endTime || performance.now()) - startTime;
    this.measures.set(name, duration);

    if (window.performance && window.performance.measure) {
      try {
        window.performance.measure(name, startMark, endMark);
      } catch (e) {
        // Ignore errors
      }
    }

    return duration;
  }

  /**
   * 获取所有测量结果
   */
  getMeasures(): Map<string, number> {
    return new Map(this.measures);
  }

  /**
   * 清除所有标记和测量
   */
  clear(): void {
    this.marks.clear();
    this.measures.clear();

    if (window.performance) {
      if (window.performance.clearMarks) {
        window.performance.clearMarks();
      }
      if (window.performance.clearMeasures) {
        window.performance.clearMeasures();
      }
    }
  }

  /**
   * 监控长任务（Long Tasks）
   */
  observeLongTasks(callback: (entries: PerformanceEntry[]) => void): PerformanceObserver | null {
    if (!window.PerformanceObserver) {
      return null;
    }

    try {
      const observer = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        callback(entries);
      });

      observer.observe({ entryTypes: ['longtask'] });
      return observer;
    } catch (e) {
      console.warn('Long task observation not supported');
      return null;
    }
  }

  /**
   * 监控布局偏移（Layout Shift）
   */
  observeLayoutShift(callback: (entries: PerformanceEntry[]) => void): PerformanceObserver | null {
    if (!window.PerformanceObserver) {
      return null;
    }

    try {
      const observer = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        callback(entries);
      });

      observer.observe({ entryTypes: ['layout-shift'] });
      return observer;
    } catch (e) {
      console.warn('Layout shift observation not supported');
      return null;
    }
  }

  /**
   * 获取内存使用情况（仅 Chrome）
   */
  getMemoryUsage() {
    const memory = (performance as any).memory;
    if (!memory) {
      return null;
    }

    return {
      usedJSHeapSize: memory.usedJSHeapSize,
      totalJSHeapSize: memory.totalJSHeapSize,
      jsHeapSizeLimit: memory.jsHeapSizeLimit,
      usagePercent: (memory.usedJSHeapSize / memory.jsHeapSizeLimit) * 100,
    };
  }

  /**
   * 生成性能报告
   */
  generateReport(): PerformanceMetrics {
    return {
      pageLoad: this.getPageLoadMetrics() || ({} as any),
      resources: this.getResourceMetrics(),
      marks: new Map(this.marks),
      measures: new Map(this.measures),
    };
  }

  /**
   * 打印性能报告到控制台
   */
  logReport(): void {
    const report = this.generateReport();

    console.group('📊 Performance Report');

    console.group('⏱️ Page Load Metrics');
    console.table(report.pageLoad);
    console.groupEnd();

    console.group('📦 Resource Metrics');
    console.table(report.resources.slice(0, 20)); // 只显示前 20 个
    console.groupEnd();

    console.group('🏷️ Custom Measures');
    const measuresObj: Record<string, number> = {};
    report.measures.forEach((value, key) => {
      measuresObj[key] = Math.round(value * 100) / 100;
    });
    console.table(measuresObj);
    console.groupEnd();

    const memory = this.getMemoryUsage();
    if (memory) {
      console.group('💾 Memory Usage');
      console.table({
        'Used Heap': `${(memory.usedJSHeapSize / 1024 / 1024).toFixed(2)} MB`,
        'Total Heap': `${(memory.totalJSHeapSize / 1024 / 1024).toFixed(2)} MB`,
        'Heap Limit': `${(memory.jsHeapSizeLimit / 1024 / 1024).toFixed(2)} MB`,
        'Usage': `${memory.usagePercent.toFixed(2)}%`,
      });
      console.groupEnd();
    }

    console.groupEnd();
  }
}

// 全局性能监控实例
export const performanceMonitor = new PerformanceMonitor();

// 在开发环境下自动记录页面加载完成时的性能
if (import.meta.env.DEV) {
  window.addEventListener('load', () => {
    setTimeout(() => {
      performanceMonitor.logReport();
    }, 1000);
  });
}
