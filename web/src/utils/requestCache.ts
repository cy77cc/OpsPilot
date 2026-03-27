/**
 * 请求缓存管理器
 *
 * 实现简单的请求缓存和去重功能
 */

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  expiresIn: number;
}

interface PendingRequest<T> {
  promise: Promise<T>;
  timestamp: number;
}

class RequestCache {
  private cache: Map<string, CacheEntry<any>> = new Map();
  private pendingRequests: Map<string, PendingRequest<any>> = new Map();
  private defaultTTL: number = 5 * 60 * 1000; // 5 分钟

  /**
   * 获取缓存数据
   */
  get<T>(key: string): T | null {
    const entry = this.cache.get(key);

    if (!entry) {
      return null;
    }

    const now = Date.now();
    if (now - entry.timestamp > entry.expiresIn) {
      this.cache.delete(key);
      return null;
    }

    return entry.data;
  }

  /**
   * 设置缓存数据
   */
  set<T>(key: string, data: T, expiresIn: number = this.defaultTTL): void {
    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      expiresIn,
    });
  }

  /**
   * 删除缓存
   */
  delete(key: string): void {
    this.cache.delete(key);
  }

  /**
   * 清空所有缓存
   */
  clear(): void {
    this.cache.clear();
    this.pendingRequests.clear();
  }

  /**
   * 合并重复请求
   * 如果相同的请求正在进行中，返回同一个 Promise
   */
  async dedupe<T>(key: string, fetcher: () => Promise<T>): Promise<T> {
    // 检查是否有正在进行的请求
    const pending = this.pendingRequests.get(key);
    if (pending) {
      return pending.promise;
    }

    // 创建新请求
    const promise = fetcher()
      .then((data) => {
        this.pendingRequests.delete(key);
        return data;
      })
      .catch((error) => {
        this.pendingRequests.delete(key);
        throw error;
      });

    this.pendingRequests.set(key, {
      promise,
      timestamp: Date.now(),
    });

    return promise;
  }

  /**
   * 带缓存的请求
   */
  async fetch<T>(
    key: string,
    fetcher: () => Promise<T>,
    options: {
      ttl?: number;
      forceRefresh?: boolean;
    } = {}
  ): Promise<T> {
    const { ttl = this.defaultTTL, forceRefresh = false } = options;

    // 如果不强制刷新，先检查缓存
    if (!forceRefresh) {
      const cached = this.get<T>(key);
      if (cached !== null) {
        return cached;
      }
    }

    // 使用去重功能获取数据
    const data = await this.dedupe(key, fetcher);

    // 缓存数据
    this.set(key, data, ttl);

    return data;
  }

  /**
   * 清理过期缓存
   */
  cleanup(): void {
    const now = Date.now();

    // 清理过期缓存
    for (const [key, entry] of this.cache.entries()) {
      if (now - entry.timestamp > entry.expiresIn) {
        this.cache.delete(key);
      }
    }

    // 清理超时的待处理请求（超过 30 秒）
    for (const [key, pending] of this.pendingRequests.entries()) {
      if (now - pending.timestamp > 30000) {
        this.pendingRequests.delete(key);
      }
    }
  }

  /**
   * 获取缓存统计信息
   */
  getStats() {
    return {
      cacheSize: this.cache.size,
      pendingRequests: this.pendingRequests.size,
    };
  }
}

// 全局请求缓存实例
export const requestCache = new RequestCache();

// 定期清理过期缓存（每 5 分钟）
if (typeof window !== 'undefined') {
  setInterval(() => {
    requestCache.cleanup();
  }, 5 * 60 * 1000);
}

/**
 * 生成缓存键
 */
export const generateCacheKey = (url: string, params?: Record<string, any>): string => {
  if (!params) {
    return url;
  }

  const sortedParams = Object.keys(params)
    .sort()
    .map((key) => `${key}=${JSON.stringify(params[key])}`)
    .join('&');

  return `${url}?${sortedParams}`;
};
