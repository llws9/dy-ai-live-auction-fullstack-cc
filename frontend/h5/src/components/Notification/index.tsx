import React, { useState } from 'react';
import { useNotification } from '../../hooks/useNotification';
import './Notification.css';

interface NotificationItem {
  id: number;
  type: string;
  title: string;
  content: string;
  data?: Record<string, unknown>;
  read_at?: string;
  created_at: string;
}

export const Notification: React.FC = () => {
  const { notifications, unreadCount, markAsRead, markAllAsRead, loading } = useNotification();
  const [isOpen, setIsOpen] = useState(false);
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);

  const filteredNotifications = showUnreadOnly
    ? notifications.filter((n: NotificationItem) => !n.read_at)
    : notifications;

  const handleNotificationClick = async (notification: NotificationItem) => {
    if (!notification.read_at) {
      await markAsRead(notification.id);
    }
  };

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'bid_outbid':
        return '💰';
      case 'auction_won':
        return '🏆';
      case 'auction_lost':
        return '😢';
      case 'order_paid':
        return '💳';
      case 'order_shipped':
        return '📦';
      case 'order_completed':
        return '✅';
      default:
        return '📢';
    }
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    return `${days}天前`;
  };

  return (
    <div className="notification-container">
      <button
        className="notification-bell"
        onClick={() => setIsOpen(!isOpen)}
      >
        🔔
        {unreadCount > 0 && (
          <span className="notification-badge">{unreadCount > 99 ? '99+' : unreadCount}</span>
        )}
      </button>

      {isOpen && (
        <div className="notification-dropdown">
          <div className="notification-header">
            <h3>通知</h3>
            <div className="notification-actions">
              <label className="filter-label">
                <input
                  type="checkbox"
                  checked={showUnreadOnly}
                  onChange={(e) => setShowUnreadOnly(e.target.checked)}
                />
                仅未读
              </label>
              {unreadCount > 0 && (
                <button className="mark-all-btn" onClick={markAllAsRead}>
                  全部已读
                </button>
              )}
            </div>
          </div>

          <div className="notification-list">
            {loading ? (
              <div className="notification-loading">加载中...</div>
            ) : filteredNotifications.length === 0 ? (
              <div className="notification-empty">暂无通知</div>
            ) : (
              filteredNotifications.map((notification: NotificationItem) => (
                <div
                  key={notification.id}
                  className={`notification-item ${notification.read_at ? 'read' : 'unread'}`}
                  onClick={() => handleNotificationClick(notification)}
                >
                  <span className="notification-icon">
                    {getNotificationIcon(notification.type)}
                  </span>
                  <div className="notification-content">
                    <div className="notification-title">{notification.title}</div>
                    <div className="notification-text">{notification.content}</div>
                    <div className="notification-time">
                      {formatTime(notification.created_at)}
                    </div>
                  </div>
                  {!notification.read_at && <span className="unread-dot" />}
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default Notification;
