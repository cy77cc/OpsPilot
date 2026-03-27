/**
 * TokenManager - 管理 Token 过期检查和主动刷新
 *
 * 职责：
 * 1. 解析 JWT 获取过期时间
 * 2. 定时检查 token 是否即将过期
 * 3. 触发主动刷新事件
 */

// 自定义事件名称
export const TOKEN_EVENTS = {
  REFRESHED: 'tokenRefreshed',
  EXPIRED: 'tokenExpired',
  NEEDS_REFRESH: 'tokenNeedsRefresh',
} as const;

// 刷新阈值：过期前 5 分钟主动刷新
const REFRESH_THRESHOLD_MS = 5 * 60 * 1000;

// 检查间隔：每分钟检查一次
const CHECK_INTERVAL_MS = 60 * 1000;

/**
 * 解析 JWT 获取过期时间
 * @param token JWT token
 * @returns 过期时间戳（毫秒）或 null（解析失败）
 */
export function parseJwtExpiresAt(token: string): number | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }

    // Base64Url 解码
    const payload = parts[1];
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );

    const decoded = JSON.parse(jsonPayload);

    if (typeof decoded.exp !== 'number') {
      return null;
    }

    // exp 是秒级时间戳，转换为毫秒
    return decoded.exp * 1000;
  } catch {
    return null;
  }
}

/**
 * 检查 token 是否即将过期
 * @param token JWT token
 * @param thresholdMs 过期阈值（毫秒），默认 5 分钟
 * @returns true 表示即将过期或已过期
 */
export function isTokenExpiringSoon(
  token: string,
  thresholdMs: number = REFRESH_THRESHOLD_MS
): boolean {
  const expiresAt = parseJwtExpiresAt(token);
  if (!expiresAt) {
    // 无法解析时，保守地认为需要刷新
    return true;
  }

  const now = Date.now();
  return expiresAt - now <= thresholdMs;
}

/**
 * 获取 token 剩余有效时间（毫秒）
 * @param token JWT token
 * @returns 剩余毫秒数，解析失败返回 0
 */
export function getTokenRemainingTime(token: string): number {
  const expiresAt = parseJwtExpiresAt(token);
  if (!expiresAt) {
    return 0;
  }

  const remaining = expiresAt - Date.now();
  return remaining > 0 ? remaining : 0;
}

/**
 * Token 过期检查管理器
 */
class TokenExpiryChecker {
  private intervalId: ReturnType<typeof setInterval> | null = null;
  private isRefreshing = false;

  /**
   * 启动定时检查
   */
  start(): void {
    if (this.intervalId !== null) {
      return; // 已启动
    }

    // 立即检查一次
    this.checkTokenExpiry();

    // 设置定时器
    this.intervalId = setInterval(() => {
      this.checkTokenExpiry();
    }, CHECK_INTERVAL_MS);
  }

  /**
   * 停止定时检查
   */
  stop(): void {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
    this.isRefreshing = false;
  }

  /**
   * 检查 token 过期状态
   */
  private checkTokenExpiry(): void {
    const token = localStorage.getItem('token');
    const refreshToken = localStorage.getItem('refreshToken');

    // 没有 token，不需要检查
    if (!token) {
      return;
    }

    // 没有 refreshToken，无法刷新
    if (!refreshToken) {
      return;
    }

    // 正在刷新中，跳过
    if (this.isRefreshing) {
      return;
    }

    // 检查是否即将过期
    if (isTokenExpiringSoon(token)) {
      this.isRefreshing = true;

      // 触发需要刷新的事件
      window.dispatchEvent(
        new CustomEvent(TOKEN_EVENTS.NEEDS_REFRESH, {
          detail: { remainingTime: getTokenRemainingTime(token) },
        })
      );
    }
  }

  /**
   * 标记刷新完成
   */
  markRefreshComplete(): void {
    this.isRefreshing = false;
  }

  /**
   * 重置刷新状态
   */
  reset(): void {
    this.isRefreshing = false;
  }
}

// 单例实例
export const tokenExpiryChecker = new TokenExpiryChecker();

/**
 * 触发 token 刷新成功事件
 */
export function dispatchTokenRefreshed(data: {
  token: string;
  refreshToken?: string;
}): void {
  window.dispatchEvent(
    new CustomEvent(TOKEN_EVENTS.REFRESHED, {
      detail: data,
    })
  );
  tokenExpiryChecker.markRefreshComplete();
}

/**
 * 触发 token 过期事件
 */
export function dispatchTokenExpired(): void {
  window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));
  tokenExpiryChecker.reset();
}

/**
 * 启动 token 过期检查
 */
export function startTokenExpiryCheck(): void {
  tokenExpiryChecker.start();
}

/**
 * 停止 token 过期检查
 */
export function stopTokenExpiryCheck(): void {
  tokenExpiryChecker.stop();
}
