/**
 * 浏览器通知工具
 * 用于发送系统级通知（需要用户授权）
 */

type NotificationPermission = 'default' | 'granted' | 'denied';

interface NotificationOptions {
  title: string;
  body?: string;
  icon?: string;
  tag?: string;
  requireInteraction?: boolean;
  onClick?: () => void;
}

/**
 * 检查浏览器是否支持通知
 */
export const isNotificationSupported = (): boolean => {
  return 'Notification' in window;
};

/**
 * 获取当前通知权限状态
 */
export const getNotificationPermission = (): NotificationPermission => {
  if (!isNotificationSupported()) {
    return 'denied';
  }
  return Notification.permission as NotificationPermission;
};

/**
 * 请求通知权限
 */
export const requestNotificationPermission = async (): Promise<NotificationPermission> => {
  if (!isNotificationSupported()) {
    console.warn('浏览器不支持通知功能');
    return 'denied';
  }

  try {
    const permission = await Notification.requestPermission();
    return permission as NotificationPermission;
  } catch (error) {
    console.error('请求通知权限失败:', error);
    return 'denied';
  }
};

/**
 * 发送浏览器通知
 */
export const sendBrowserNotification = async (options: NotificationOptions): Promise<Notification | null> => {
  if (!isNotificationSupported()) {
    return null;
  }

  // 检查权限
  let permission = getNotificationPermission();

  if (permission === 'default') {
    permission = await requestNotificationPermission();
  }

  if (permission !== 'granted') {
    console.warn('用户未授权通知权限');
    return null;
  }

  try {
    const notification = new Notification(options.title, {
      body: options.body,
      icon: options.icon || '/favicon.ico',
      tag: options.tag,
      requireInteraction: options.requireInteraction,
    });

    if (options.onClick) {
      notification.onclick = () => {
        options.onClick?.();
        notification.close();
        // 将窗口聚焦
        window.focus();
      };
    }

    // 自动关闭（5秒后）
    setTimeout(() => {
      notification.close();
    }, 5000);

    return notification;
  } catch (error) {
    console.error('发送浏览器通知失败:', error);
    return null;
  }
};

/**
 * 发送通知（带权限检查和自动请求）
 */
export const notify = async (
  title: string,
  body?: string,
  options?: Partial<NotificationOptions>
): Promise<Notification | null> => {
  // 检查用户是否启用浏览器通知
  const browserNotificationEnabled = localStorage.getItem('notification:browser:enabled') !== 'false';
  if (!browserNotificationEnabled) {
    return null;
  }

  return sendBrowserNotification({
    title,
    body,
    ...options,
  });
};

/**
 * 设置是否启用浏览器通知
 */
export const setBrowserNotificationEnabled = (enabled: boolean): void => {
  localStorage.setItem('notification:browser:enabled', String(enabled));
};

/**
 * 获取浏览器通知是否启用
 */
export const isBrowserNotificationEnabled = (): boolean => {
  return localStorage.getItem('notification:browser:enabled') !== 'false';
};

export default {
  isNotificationSupported,
  getNotificationPermission,
  requestNotificationPermission,
  sendBrowserNotification,
  notify,
  setBrowserNotificationEnabled,
  isBrowserNotificationEnabled,
};
