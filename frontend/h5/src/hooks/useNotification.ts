import { useState, useEffect, useCallback } from 'react';
import { useWebSocket } from './useWebSocket';

interface NotificationItem {
  id: number;
  type: string;
  title: string;
  content: string;
  data?: Record<string, unknown>;
  read_at?: string;
  created_at: string;
}

interface NotificationState {
  notifications: NotificationItem[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

export const useNotification = () => {
  const [state, setState] = useState<NotificationState>({
    notifications: [],
    unreadCount: 0,
    loading: false,
    error: null,
  });

  const { lastMessage } = useWebSocket();

  // 获取通知列表
  const fetchNotifications = useCallback(async (page = 1, pageSize = 20) => {
    setState((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const token = localStorage.getItem('token');
      const response = await fetch(
        `${API_BASE_URL}/api/v1/notifications?page=${page}&page_size=${pageSize}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error('获取通知失败');
      }

      const data = await response.json();
      setState((prev) => ({
        ...prev,
        notifications: data.items || [],
        loading: false,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : '获取通知失败',
      }));
    }
  }, []);

  // 获取未读数量
  const fetchUnreadCount = useCallback(async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`${API_BASE_URL}/api/v1/notifications/unread-count`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error('获取未读数量失败');
      }

      const data = await response.json();
      setState((prev) => ({
        ...prev,
        unreadCount: data.data?.count || 0,
      }));
    } catch (error) {
      console.error('获取未读数量失败:', error);
    }
  }, []);

  // 标记已读
  const markAsRead = useCallback(async (id: number) => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`${API_BASE_URL}/api/v1/notifications/${id}/read`, {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error('标记已读失败');
      }

      // 更新本地状态
      setState((prev) => ({
        ...prev,
        notifications: prev.notifications.map((n) =>
          n.id === id ? { ...n, read_at: new Date().toISOString() } : n
        ),
        unreadCount: Math.max(0, prev.unreadCount - 1),
      }));
    } catch (error) {
      console.error('标记已读失败:', error);
    }
  }, []);

  // 标记全部已读
  const markAllAsRead = useCallback(async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`${API_BASE_URL}/api/v1/notifications/read-all`, {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error('标记全部已读失败');
      }

      // 更新本地状态
      setState((prev) => ({
        ...prev,
        notifications: prev.notifications.map((n) => ({
          ...n,
          read_at: n.read_at || new Date().toISOString(),
        })),
        unreadCount: 0,
      }));
    } catch (error) {
      console.error('标记全部已读失败:', error);
    }
  }, []);

  // 处理WebSocket通知消息
  useEffect(() => {
    if (lastMessage && lastMessage.type === 'notification') {
      const notification = lastMessage.data as NotificationItem;

      setState((prev) => ({
        ...prev,
        notifications: [notification, ...prev.notifications].slice(0, 50),
        unreadCount: prev.unreadCount + 1,
      }));

      // 显示浏览器通知（如果支持）
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(notification.title, {
          body: notification.content,
        });
      }
    }
  }, [lastMessage]);

  // 初始化加载
  useEffect(() => {
    fetchNotifications();
    fetchUnreadCount();

    // 定时刷新未读数量
    const interval = setInterval(fetchUnreadCount, 60000);
    return () => clearInterval(interval);
  }, [fetchNotifications, fetchUnreadCount]);

  return {
    notifications: state.notifications,
    unreadCount: state.unreadCount,
    loading: state.loading,
    error: state.error,
    fetchNotifications,
    fetchUnreadCount,
    markAsRead,
    markAllAsRead,
  };
};
