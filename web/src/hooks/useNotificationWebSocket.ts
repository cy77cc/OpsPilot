import { useRef, useCallback, useEffect } from 'react';
import type { WSMessage, WSConnectionStatus } from '../types/notification';

interface UseNotificationWebSocketOptions {
  userId?: number | string;
  onMessage?: (message: WSMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  reconnectInterval?: number;
  maxReconnectInterval?: number;
}

export const useNotificationWebSocket = (options: UseNotificationWebSocketOptions = {}) => {
  const {
    userId,
    onMessage,
    onConnect,
    onDisconnect,
    reconnectInterval = 1000,
    maxReconnectInterval = 30000,
  } = options;

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const currentReconnectIntervalRef = useRef(reconnectInterval);
  const statusRef = useRef<WSConnectionStatus>('disconnected');
  const broadcastChannelRef = useRef<BroadcastChannel | null>(null);

  const userIdRef = useRef(userId);

  useEffect(() => {
    userIdRef.current = userId;
  }, [userId]);

  // 使用 ref 存储回调，避免依赖变化
  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);

  useEffect(() => {
    onMessageRef.current = onMessage;
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
  }, [onMessage, onConnect, onDisconnect]);

  // 多标签页同步
  const initBroadcastChannel = useCallback(() => {
    if (typeof BroadcastChannel !== 'undefined') {
      broadcastChannelRef.current = new BroadcastChannel('notification-sync');
      broadcastChannelRef.current.onmessage = (event) => {
        if (event.data.type === 'notification-update') {
          onMessageRef.current?.(event.data.message);
        }
      };
    }
  }, []);

  // 广播消息到其他标签页
  const broadcast = useCallback((message: WSMessage) => {
    if (broadcastChannelRef.current) {
      broadcastChannelRef.current.postMessage({
        type: 'notification-update',
        message,
      });
    }
  }, []);

  // 连接 WebSocket
  const connect = useCallback(() => {
    const currentUserId = userIdRef.current;
    if (!currentUserId || statusRef.current === 'connecting') {
      return;
    }

    statusRef.current = 'connecting';

    // 构建 WebSocket URL
    let wsUrl: string;
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

    if (import.meta.env.VITE_WS_URL) {
      // 使用配置的 WebSocket URL
      wsUrl = `${import.meta.env.VITE_WS_URL}/ws/notifications?user_id=${currentUserId}`;
    } else if (import.meta.env.DEV) {
      // 开发环境：通过 vite proxy 代理，使用当前 host
      // vite.config.ts 中配置了 /ws 代理到后端
      wsUrl = `${wsProtocol}//${window.location.host}/ws/notifications?user_id=${currentUserId}`;
    } else {
      // 生产环境：使用当前 host
      wsUrl = `${wsProtocol}//${window.location.host}/ws/notifications?user_id=${currentUserId}`;
    }

    console.log('WebSocket: 正在连接', wsUrl);

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        statusRef.current = 'connected';
        currentReconnectIntervalRef.current = reconnectInterval;
        onConnectRef.current?.();
        console.log('WebSocket: 已连接');
      };

      ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          onMessageRef.current?.(message);
          broadcast(message);
        } catch (error) {
          console.error('WebSocket: 解析消息失败', error);
        }
      };

      ws.onclose = (event) => {
        statusRef.current = 'disconnected';
        onDisconnectRef.current?.();
        console.log('WebSocket: 连接关闭', event.code, event.reason);

        // 只有在非正常关闭时才自动重连
        // 1000 = 正常关闭, 1001 = 端点离开, 1005 = 无状态码
        if (event.code !== 1000 && event.code !== 1001 && event.code !== 1005) {
          scheduleReconnect();
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket: 连接错误', error);
        // 注意：不要在这里调用 ws.close()，onclose 会自动触发
      };
    } catch (error) {
      console.error('WebSocket: 创建连接失败', error);
      statusRef.current = 'disconnected';
      scheduleReconnect();
    }
  }, [reconnectInterval, broadcast]);

  // 安排重连
  const scheduleReconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }

    const delay = currentReconnectIntervalRef.current;
    currentReconnectIntervalRef.current = Math.min(
      currentReconnectIntervalRef.current * 2,
      maxReconnectInterval
    );

    reconnectTimeoutRef.current = setTimeout(() => {
      console.log(`WebSocket: 尝试重连 (延迟 ${delay}ms)`);
      connect();
    }, delay);
  }, [connect, maxReconnectInterval]);

  // 断开连接
  const disconnect = useCallback(() => {
    // 清除重连定时器
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    // 正常关闭 WebSocket
    if (wsRef.current) {
      // 使用 1000 状态码表示正常关闭，避免触发重连
      wsRef.current.close(1000, 'Component unmounting');
      wsRef.current = null;
    }
    statusRef.current = 'disconnected';
  }, []);

  // 发送消息
  const send = useCallback((data: object) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  // 初始化 - 只在 userId 变化时执行
  useEffect(() => {
    initBroadcastChannel();

    if (userId) {
      connect();
    }

    return () => {
      disconnect();
      if (broadcastChannelRef.current) {
        broadcastChannelRef.current.close();
      }
    };
  }, [userId]);

  return {
    connect,
    disconnect,
    send,
    get status() {
      return statusRef.current;
    },
  };
};

export default useNotificationWebSocket;
