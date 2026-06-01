import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { notificationApi, NotificationItem } from '../../services/notification';
import { getCountBucket, trackEvent } from '../../utils/trackEvent';
import PageHeader from '@/components/shared/PageHeader';
import styles from './Notifications.module.css';

type NotificationType =
  | 'bid_outbid'
  | 'auction_won'
  | 'auction_lost'
  | 'auction_win'
  | 'auction_lose'
  | 'auction_start'
  | 'auction_starting'
  | 'live_start'
  | 'live_stream_starting_soon'
  | 'live_stream_now_live'
  | 'order'
  | 'order_paid'
  | 'order_shipped'
  | 'order_completed'
  | string;

interface NotificationRecord extends NotificationItem {
  type: NotificationType;
  is_read?: boolean;
  live_stream_id?: number | string;
  auction_id?: number | string;
  order_id?: number | string;
}

function extractList(response: any): NotificationRecord[] {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  return [];
}

function readField(notification: NotificationRecord, key: 'live_stream_id' | 'auction_id' | 'order_id') {
  return notification[key] ?? notification.data?.[key];
}

function isUnread(notification: NotificationRecord) {
  if (typeof notification.is_read === 'boolean') return !notification.is_read;
  return !notification.read_at;
}

function formatTime(value?: string) {
  if (!value) return '时间待确认';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function getMeta(type: NotificationType) {
  switch (type) {
    case 'live_start':
    case 'live_stream_starting_soon':
    case 'live_stream_now_live':
      return { label: '开播提醒', icon: 'LIVE', tone: 'gold' as const };
    case 'auction_won':
    case 'auction_win':
      return { label: '竞拍成功', icon: 'WIN', tone: 'success' as const };
    case 'auction_lost':
    case 'auction_lose':
      return { label: '竞拍结果', icon: 'LOT', tone: 'muted' as const };
    case 'auction_start':
    case 'auction_starting':
    case 'bid_outbid':
      return { label: '竞拍提醒', icon: 'BID', tone: 'warning' as const };
    case 'order':
    case 'order_paid':
    case 'order_shipped':
    case 'order_completed':
      return { label: '订单通知', icon: 'ORD', tone: 'blue' as const };
    default:
      return { label: '系统通知', icon: 'MSG', tone: 'muted' as const };
  }
}

function getTouchpointType(type: NotificationType) {
  if (type === 'live_start' || type === 'live_stream_starting_soon' || type === 'live_stream_now_live') {
    return 'live_start';
  }
  return 'notification';
}

function getTarget(notification: NotificationRecord) {
  const type = notification.type;
  const liveStreamId = readField(notification, 'live_stream_id');
  const auctionId = readField(notification, 'auction_id');
  const orderId = readField(notification, 'order_id');

  if (['live_start', 'live_stream_starting_soon', 'live_stream_now_live'].includes(type) && liveStreamId) {
    return `/live?id=${liveStreamId}`;
  }

  if (['auction_won', 'auction_lost', 'auction_win', 'auction_lose'].includes(type) && auctionId) {
    return `/result?id=${auctionId}`;
  }

  if (['auction_start', 'auction_starting', 'bid_outbid'].includes(type) && auctionId) {
    return `/detail?id=${auctionId}`;
  }

  // T3.6 / F-D2 §6：订单类通知携带 order_id 时跳转订单详情页
  if ((type === 'order' || type.startsWith('order_')) && orderId) {
    return `/order/${orderId}`;
  }

  return null;
}

const pageSize = 20;

const NotificationsPage: React.FC = () => {
  const navigate = useNavigate();
  const [notifications, setNotifications] = useState<NotificationRecord[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [markingAll, setMarkingAll] = useState(false);

  useEffect(() => {
    let alive = true;

    async function loadNotifications() {
      setLoading(true);
      setError(null);

      try {
        const [listResponse, unreadResponse] = await Promise.all([
          notificationApi.list(1, pageSize),
          notificationApi.getUnreadCount(),
        ]);
        if (!alive) return;
        const items = extractList(listResponse);
        setNotifications(items);
        setUnreadCount(unreadResponse.count || 0);
        trackEvent('notification_list_exposed', {
          source: 'notification_center',
          entry: 'notification_center',
          type: 'notification',
          result: 'success',
          countBucket: getCountBucket(items.length),
        });
      } catch (err) {
        if (!alive) return;
        console.error('获取通知列表失败:', err);
        setNotifications([]);
        setUnreadCount(0);
        setError('消息通知暂时无法加载');
      } finally {
        if (alive) setLoading(false);
      }
    }

    loadNotifications();

    return () => {
      alive = false;
    };
  }, []);

  const stats = useMemo(() => {
    const unread = notifications.filter(isUnread).length;
    return {
      total: notifications.length,
      unread: Math.max(unreadCount, unread),
      actionable: notifications.filter(getTarget).length,
    };
  }, [notifications, unreadCount]);

  const markOneAsRead = async (notification: NotificationRecord) => {
    if (!isUnread(notification)) return;

    await notificationApi.markAsRead(notification.id);
    setNotifications((items) =>
      items.map((item) =>
        item.id === notification.id ? { ...item, read_at: item.read_at || new Date().toISOString(), is_read: true } : item
      )
    );
    setUnreadCount((count) => Math.max(0, count - 1));
  };

  const handleOpenNotification = async (notification: NotificationRecord) => {
    const target = getTarget(notification);

    try {
      await markOneAsRead(notification);
    } catch (err) {
      console.error('标记通知已读失败:', err);
    }

    trackEvent('notification_item_clicked', {
      source: 'notification_center',
      entry: 'notification_item',
      type: getTouchpointType(notification.type),
      result: 'clicked',
    });

    if (target) {
      navigate(target);
    }
  };

  const handleMarkAllAsRead = async () => {
    setMarkingAll(true);
    try {
      await notificationApi.markAllAsRead();
      const now = new Date().toISOString();
      setNotifications((items) => items.map((item) => ({ ...item, read_at: item.read_at || now, is_read: true })));
      setUnreadCount(0);
      trackEvent('mark_read', {
        source: 'notification_center',
        entry: 'mark_all_read',
        type: 'all',
        result: 'success',
      });
    } catch (err) {
      console.error('全部标记已读失败:', err);
      trackEvent('mark_read', {
        source: 'notification_center',
        entry: 'mark_all_read',
        type: 'all',
        result: 'failed',
      });
      setError('全部标记已读失败，请稍后重试');
    } finally {
      setMarkingAll(false);
    }
  };

  return (
    <section className={styles.page}>
      <PageHeader
        classes={{
          header: styles.header,
          backButton: styles.backButton,
          eyebrow: styles.eyebrow,
        }}
        back={{ onClick: () => navigate(-1) }}
        eyebrow="Notification Center"
        title="消息通知"
        actions={<Link className={styles.homeLink} to="/">首页</Link>}
      />

      <section className={styles.summaryGrid} aria-label="通知概览">
        <div className={styles.summaryCard}>
          <span>{stats.total} 条消息</span>
          <p>全部通知</p>
        </div>
        <div className={styles.summaryCard}>
          <span>{stats.unread} 条未读</span>
          <p>待处理</p>
        </div>
        <div className={styles.summaryCard}>
          <span>{stats.actionable}</span>
          <p>可跳转</p>
        </div>
      </section>

      <div className={styles.toolbar}>
        <p>开播提醒、竞拍结果和系统通知将在此汇总。</p>
        <button type="button" disabled={stats.unread === 0 || markingAll} onClick={handleMarkAllAsRead}>
          {markingAll ? '处理中...' : '全部已读'}
        </button>
      </div>

      {error && <div className={styles.errorBanner}>{error}</div>}

      <main className={styles.content} aria-live="polite">
        {loading ? (
          <div className={styles.statePage}>
            <div className={styles.spinner} />
            <p>加载消息通知...</p>
          </div>
        ) : notifications.length === 0 ? (
          <div className={styles.statePage}>
            <div className={styles.emptyIcon}>MSG</div>
            <p>暂无消息通知</p>
            <span>开播提醒、竞拍结果和订单状态会显示在这里</span>
            <Link to="/">去看看竞拍</Link>
          </div>
        ) : (
          <div className={styles.notificationList}>
            {notifications.map((notification) => {
              const meta = getMeta(notification.type);
              const target = getTarget(notification);
              const unread = isUnread(notification);
              const card = (
                <>
                  <div className={styles.iconFrame} data-tone={meta.tone}>
                    {meta.icon}
                  </div>
                  <div className={styles.messageBody}>
                    <div className={styles.messageTitleRow}>
                      <span>{meta.label}</span>
                      <time dateTime={notification.created_at}>{formatTime(notification.created_at)}</time>
                    </div>
                    <h2>{notification.title || meta.label}</h2>
                    <p>{notification.content}</p>
                    {!target && (
                      <small>
                        {notification.type.startsWith('order') || notification.type === 'order'
                          ? '订单详情页尚未开放，当前仅展示通知内容'
                          : '该通知暂无可跳转目标'}
                      </small>
                    )}
                  </div>
                  {unread && <span className={styles.unreadDot} aria-label="未读" />}
                </>
              );

              return target ? (
                <button
                  key={notification.id}
                  className={unread ? styles.unreadCard : styles.card}
                  type="button"
                  onClick={() => handleOpenNotification(notification)}
                  aria-label={`打开通知：${notification.title || ''} ${notification.content}`}
                >
                  {card}
                </button>
              ) : (
                <article key={notification.id} className={unread ? styles.unreadCard : styles.card}>
                  {card}
                </article>
              );
            })}
          </div>
        )}
      </main>
    </section>
  );
};

export default NotificationsPage;
