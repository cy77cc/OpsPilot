/**
 * 通知类型定义
 */

// 通知类型
export type NotificationType = 'alert' | 'task' | 'system' | 'approval';

// 通知严重级别
export type NotificationSeverity = 'critical' | 'warning' | 'info';

// 可执行操作类型
export type NotificationActionType = 'confirm' | 'approve' | 'view';

// 通知主体
export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  content?: string;
  severity: NotificationSeverity;
  source: string;
  source_id: string;
  action_url?: string;
  action_type?: NotificationActionType;
  created_at: string;
}

// 用户通知关联
export interface UserNotification {
  id: string;
  user_id: string;
  notification_id: string;
  read_at?: string;
  dismissed_at?: string;
  confirmed_at?: string;
  notification: Notification;
}

// 通知列表请求参数
export interface NotificationListParams {
  page?: number;
  pageSize?: number;
  unreadOnly?: boolean;
  type?: NotificationType;
  severity?: NotificationSeverity;
}

// 未读数量响应
export interface UnreadCountResponse {
  total: number;
  by_type: Record<NotificationType, number>;
  by_severity: Record<NotificationSeverity, number>;
}

// WebSocket 消息类型
export type WSMessageType = 'new' | 'update' | 'delete';

export interface WSMessage {
  type: WSMessageType;
  notification?: UserNotification;
  id?: string;
  read_at?: string;
  dismissed_at?: string;
  confirmed_at?: string;
}

// WebSocket 连接状态
export type WSConnectionStatus = 'connecting' | 'connected' | 'disconnected';
