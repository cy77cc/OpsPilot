import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';
import type {
  UserNotification,
  NotificationListParams,
  UnreadCountResponse,
} from '../../types/notification';

// Mock 数据用于开发
const MOCK_NOTIFICATIONS: UserNotification[] = [
  {
    id: '1',
    user_id: '1',
    notification_id: '101',
    read_at: undefined,
    notification: {
      id: '101',
      type: 'alert',
      title: 'CPU 使用率超过 90%',
      content: '主机 node-01 的 CPU 使用率达到 95%，请及时处理',
      severity: 'critical',
      source: '主机 node-01',
      source_id: 'alert-001',
      action_url: '/monitor?alert_id=alert-001',
      action_type: 'confirm',
      created_at: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    },
  },
  {
    id: '2',
    user_id: '1',
    notification_id: '102',
    read_at: undefined,
    notification: {
      id: '102',
      type: 'alert',
      title: '内存使用率超过 80%',
      content: '主机 node-02 的内存使用率达到 85%',
      severity: 'warning',
      source: '主机 node-02',
      source_id: 'alert-002',
      action_url: '/monitor?alert_id=alert-002',
      action_type: 'confirm',
      created_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    },
  },
  {
    id: '3',
    user_id: '1',
    notification_id: '103',
    read_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    notification: {
      id: '103',
      type: 'task',
      title: '部署任务完成',
      content: 'deployment-web 部署成功',
      severity: 'info',
      source: 'deployment-web',
      source_id: 'task-001',
      action_url: '/tasks/task-001',
      action_type: 'view',
      created_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    },
  },
  {
    id: '4',
    user_id: '1',
    notification_id: '104',
    read_at: undefined,
    notification: {
      id: '104',
      type: 'alert',
      title: '磁盘空间不足',
      content: '主机 node-03 的磁盘使用率达到 92%',
      severity: 'warning',
      source: '主机 node-03',
      source_id: 'alert-003',
      action_url: '/monitor?alert_id=alert-003',
      action_type: 'confirm',
      created_at: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
    },
  },
  {
    id: '5',
    user_id: '1',
    notification_id: '105',
    read_at: new Date(Date.now() - 120 * 60 * 1000).toISOString(),
    notification: {
      id: '105',
      type: 'system',
      title: '系统维护通知',
      content: '系统将于今晚 22:00 进行维护，预计持续 1 小时',
      severity: 'info',
      source: '系统',
      source_id: 'sys-001',
      action_url: '/help',
      created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    },
  },
];

const USE_MOCK = false; // Phase 2 使用真实 API

export const notificationApi = {
  // 获取通知列表
  async getNotifications(params?: NotificationListParams): Promise<ApiResponse<PaginatedResponse<UserNotification>>> {
    if (USE_MOCK) {
      let list = [...MOCK_NOTIFICATIONS];

      // 过滤未读
      if (params?.unreadOnly) {
        list = list.filter((n) => !n.read_at);
      }

      // 过滤类型
      if (params?.type) {
        list = list.filter((n) => n.notification.type === params.type);
      }

      // 过滤严重级别
      if (params?.severity) {
        list = list.filter((n) => n.notification.severity === params.severity);
      }

      // 按时间倒序
      list.sort((a, b) => new Date(b.notification.created_at).getTime() - new Date(a.notification.created_at).getTime());

      const page = params?.page || 1;
      const pageSize = params?.pageSize || 20;
      const start = (page - 1) * pageSize;
      const paginatedList = list.slice(start, start + pageSize);

      return {
        success: true,
        data: {
          list: paginatedList,
          total: list.length,
        },
      };
    }

    const response = await apiService.get<UserNotification[]>('/notifications', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
        unread_only: params?.unreadOnly,
        type: params?.type,
        severity: params?.severity,
      },
    });
    const raw = Array.isArray(response.data) ? response.data : (response.data as any)?.list || [];
    const total = Array.isArray(response.data) ? response.total || 0 : (response.data as any)?.total || response.total || 0;
    return {
      ...response,
      data: { list: raw, total },
    };
  },

  // 获取未读数量
  async getUnreadCount(): Promise<ApiResponse<UnreadCountResponse>> {
    if (USE_MOCK) {
      const unread = MOCK_NOTIFICATIONS.filter((n) => !n.read_at);
      return {
        success: true,
        data: {
          total: unread.length,
          by_type: {
            alert: unread.filter((n) => n.notification.type === 'alert').length,
            task: unread.filter((n) => n.notification.type === 'task').length,
            system: unread.filter((n) => n.notification.type === 'system').length,
            approval: unread.filter((n) => n.notification.type === 'approval').length,
          },
          by_severity: {
            critical: unread.filter((n) => n.notification.severity === 'critical').length,
            warning: unread.filter((n) => n.notification.severity === 'warning').length,
            info: unread.filter((n) => n.notification.severity === 'info').length,
          },
        },
      };
    }

    return apiService.get('/notifications/unread-count');
  },

  // 标记已读
  async markAsRead(id: string): Promise<ApiResponse<void>> {
    if (USE_MOCK) {
      const notification = MOCK_NOTIFICATIONS.find((n) => n.id === id);
      if (notification) {
        notification.read_at = new Date().toISOString();
      }
      return { success: true, data: undefined };
    }

    return apiService.post(`/notifications/${encodeURIComponent(id)}/read`);
  },

  // 忽略通知
  async dismiss(id: string): Promise<ApiResponse<void>> {
    if (USE_MOCK) {
      const notification = MOCK_NOTIFICATIONS.find((n) => n.id === id);
      if (notification) {
        notification.dismissed_at = new Date().toISOString();
      }
      return { success: true, data: undefined };
    }

    return apiService.post(`/notifications/${encodeURIComponent(id)}/dismiss`);
  },

  // 确认告警
  async confirm(id: string): Promise<ApiResponse<void>> {
    if (USE_MOCK) {
      const notification = MOCK_NOTIFICATIONS.find((n) => n.id === id);
      if (notification) {
        notification.confirmed_at = new Date().toISOString();
        notification.read_at = new Date().toISOString();
      }
      return { success: true, data: undefined };
    }

    return apiService.post(`/notifications/${encodeURIComponent(id)}/confirm`);
  },

  // 审批通过（通知维度）
  async approve(id: string): Promise<ApiResponse<void>> {
    return this.confirm(id);
  },

  // 审批驳回（通知维度）
  async reject(id: string): Promise<ApiResponse<void>> {
    return this.dismiss(id);
  },

  // 全部已读
  async markAllAsRead(): Promise<ApiResponse<void>> {
    if (USE_MOCK) {
      MOCK_NOTIFICATIONS.forEach((n) => {
        if (!n.read_at) {
          n.read_at = new Date().toISOString();
        }
      });
      return { success: true, data: undefined };
    }

    return apiService.post('/notifications/read-all');
  },
};
