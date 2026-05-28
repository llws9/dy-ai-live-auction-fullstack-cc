import { useState, useEffect, useCallback, useRef } from 'react';
import { notificationApi, NotificationItem } from '../services/notification';
import { NotificationData } from './useWebSocket';

interface NotificationState {
  notifications: NotificationItem[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
  hasUnread: boolean;
}

export const useNotification = () => {
  const [state, setState] = useState<NotificationState>({
    notifications: [],
    unreadCount: 0,
    loading: false,
    error: null,
    hasUnread: false,
  });

  // 热拉时间戳，用于防抖
  const lastHotPullTimeRef = useRef<number>(0);
  // 热拉最小间隔（30秒）
  const HOT_PULL_MIN_INTERVAL = 30000;
  // WebSocket连接状态
  const wsConnectedRef = useRef(false);
  // 用户ID（用于WebSocket房间加入）
  const userIDRef = useRef<string | null>(null);

  // 处理WebSocket通知消息
  const handleWebSocketNotification = useCallback((notification: NotificationData) => {
    // 转换为NotificationItem格式
    const notificationItem: NotificationItem = {
      id: notification.id,
      type: notification.type,
      title: notification.title,
      content: notification.content,
      data: notification.data,
      read_at: undefined,
      created_at: notification.created_at,
    };

    // 更新通知状态
    setState((prev) => {
      // 去重：检查是否已存在该通知
      const existingIds = new Set(prev.notifications.map((n) => n.id));
      if (existingIds.has(notification.id)) {
        return prev;
      }

      return {
        ...prev,
        notifications: [notificationItem, ...prev.notifications].slice(0, 50),
        unreadCount: prev.unreadCount + 1,
        hasUnread: true,
      };
    });
  }, []);

  // 设置WebSocket连接状态（供外部组件调用）
  const setWsConnected = useCallback((connected: boolean) => {
    wsConnectedRef.current = connected;
  }, []);

  // 设置用户ID（供外部组件调用，用于WebSocket房间加入）
  const setUserID = useCallback((userID: string | null) => {
    userIDRef.current = userID;
  }, []);

  // 清除未读标记
  const clearHasUnread = useCallback(() => {
    setState((prev) => ({
      ...prev,
      hasUnread: false,
    }));
  }, []);

  // 获取通知列表
  const fetchNotifications = useCallback(async (page = 1, pageSize = 20) => {
    setState((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const data = await notificationApi.list(page, pageSize);
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
      const data = await notificationApi.getUnreadCount();
      setState((prev) => ({
        ...prev,
        unreadCount: data.count || 0,
      }));
    } catch (error) {
      console.error('获取未读数量失败:', error);
    }
  }, []);

  // 标记已读
  const markAsRead = useCallback(async (id: number) => {
    try {
      await notificationApi.markAsRead(id);

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
      await notificationApi.markAllAsRead();

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

  // 热拉通知 - 用户切换前台或登录时主动拉取
  // 最小间隔30秒防抖，避免频繁请求
  const hotPullNotifications = useCallback(async () => {
    const now = Date.now();
    if (lastHotPullTimeRef.current && now - lastHotPullTimeRef.current < HOT_PULL_MIN_INTERVAL) {
      console.log('[HotPull] Skipped due to debounce');
      return;
    }
    lastHotPullTimeRef.current = now;

    try {
      console.log('[HotPull] Fetching notifications...');
      const data = await notificationApi.hotPull();

      // 更新通知列表（新增的通知放到前面）
      if (data.notifications && data.notifications.length > 0) {
        setState((prev) => {
          // 去重：过滤掉已存在的通知
          const existingIds = new Set(prev.notifications.map((n) => n.id));
          const newNotifications = data.notifications.filter((n) => !existingIds.has(n.id));

          return {
            ...prev,
            notifications: [...newNotifications, ...prev.notifications].slice(0, 50),
            unreadCount: prev.unreadCount + newNotifications.length,
          };
        });

        console.log(`[HotPull] Received ${data.notifications.length} new notifications`);
      }
    } catch (error) {
      console.error('[HotPull] Failed:', error);
    }
  }, []);

  // 处理WebSocket通知消息
  // TODO: 集成 WebSocket 通知推送
  // useEffect(() => {
  //   if (lastMessage && lastMessage.type === 'notification') {
  //     const notification = lastMessage.data as NotificationItem;
  //
  //     setState((prev) => ({
  //       ...prev,
  //       notifications: [notification, ...prev.notifications].slice(0, 50),
  //       unreadCount: prev.unreadCount + 1,
  //     }));
  //
  //     // 显示浏览器通知（如果支持）
  //     if ('Notification' in window && Notification.permission === 'granted') {
  //       new Notification(notification.title, {
  //         body: notification.content,
  //       });
  //     }
  //   }
  // }, [lastMessage]);

  // 初始化加载
  useEffect(() => {
    fetchNotifications();
    fetchUnreadCount();

    // 定时刷新未读数量
    const interval = setInterval(fetchUnreadCount, 60000);
    return () => clearInterval(interval);
  }, [fetchNotifications, fetchUnreadCount]);

  // 监听 visibilitychange 事件 - 用户切换前台时触发热拉
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        // 检查用户是否已登录
        const token = localStorage.getItem('token');
        if (token) {
          console.log('[HotPull] Triggered by visibility change');
          hotPullNotifications();
        }
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [hotPullNotifications]);

  // 监听登录成功事件 - 登录成功后触发热拉
  useEffect(() => {
    const handleLoginSuccess = () => {
      console.log('[HotPull] Triggered by login success');
      hotPullNotifications();
    };

    window.addEventListener('login-success', handleLoginSuccess);
    return () => {
      window.removeEventListener('login-success', handleLoginSuccess);
    };
  }, [hotPullNotifications]);

  return {
    notifications: state.notifications,
    unreadCount: state.unreadCount,
    hasUnread: state.hasUnread,
    loading: state.loading,
    error: state.error,
    fetchNotifications,
    fetchUnreadCount,
    markAsRead,
    markAllAsRead,
    hotPullNotifications, // 暴露热拉方法，供登录成功后调用
    handleWebSocketNotification, // 处理WebSocket通知消息
    setWsConnected, // 设置WebSocket连接状态
    setUserID, // 设置用户ID
    clearHasUnread, // 清除未读标记
  };
};
