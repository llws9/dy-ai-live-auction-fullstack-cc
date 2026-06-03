import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { orderApi, userApi } from '../../services/api';
import { useAuth } from '../../store/authContext';
import BadgeDot from '../../components/BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import { repairUtf8Mojibake } from '../../utils/textEncoding';
import { trackEvent } from '../../utils/trackEvent';
import styles from './Profile.module.css';

interface ProfileUser {
  id?: number | string;
  name?: string;
  email?: string;
  avatar?: string;
  role?: number;
}

// 后端 T3.1 返回 { available_amount, frozen_amount, currency }；
// 兼容旧字段（balance/available_balance）以避免历史 mock 测试跑红。
interface BalanceData {
  available_amount?: number | string;
  frozen_amount?: number | string;
  currency?: string;
  // 旧字段（已废弃，保留兼容）
  balance?: number | string;
  available_balance?: number | string;
}

interface UserStats {
  following_count?: number | null;
  auction_history_count?: number | null;
  won_count?: number | null;
}

interface OrderSummary {
  id: number | string;
  product_name?: string;
  product?: {
    name?: string;
  };
  final_price?: number | string;
  status?: number | string;
  created_at?: string;
}

function extractList<T>(response: any): T[] {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.orders)) return response.orders;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  return [];
}

function toNumber(value: number | string | undefined, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function formatCurrency(value: number | string | undefined) {
  return `¥${toNumber(value).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}`;
}

function statValue(value?: number | null) {
  if (value === undefined || value === null) return '-';
  return String(value);
}

// 钱包余额：优先取 T3.1 新字段 available_amount，回退旧字段
function pickAvailable(b: BalanceData | null) {
  if (!b) return undefined;
  if (b.available_amount !== undefined) return b.available_amount;
  if (b.available_balance !== undefined) return b.available_balance;
  return b.balance;
}

function pickFrozen(b: BalanceData | null) {
  if (!b) return undefined;
  return b.frozen_amount;
}

function roleLabel(role?: number) {
  if (role === 2) return '管理员';
  if (role === 1) return '商家/主播';
  return '普通用户';
}

function orderStatusLabel(status?: number | string) {
  const normalized = String(status ?? '');
  if (normalized === '1' || normalized === 'paid') return '已支付';
  if (normalized === '2' || normalized === 'shipped') return '已发货';
  if (normalized === '3' || normalized === 'completed') return '已完成';
  return '待支付';
}

const UserCenter: React.FC = () => {
  const { user: authUser, logout } = useAuth();
  const { pendingPayment } = useTouchpointNotifications();
  const navigate = useNavigate();
  const [profile, setProfile] = useState<ProfileUser | null>(authUser);
  const [balance, setBalance] = useState<BalanceData | null>(null);
  const [stats, setStats] = useState<UserStats | null>(null);
  const [orders, setOrders] = useState<OrderSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    let alive = true;

    async function loadProfile() {
      setLoading(true);
      setError(null);

      const [profileResult, balanceResult, statsResult, ordersResult] = await Promise.allSettled([
        userApi.getProfile(),
        userApi.getBalance(),
        userApi.getStats(),
        orderApi.list({ page: 1, page_size: 2 }),
      ]);

      if (!alive) return;

      if (profileResult.status === 'fulfilled') {
        setProfile(profileResult.value);
      } else if (!authUser) {
        setError('获取用户信息失败');
      }

      if (balanceResult.status === 'fulfilled') {
        setBalance(balanceResult.value);
      }

      // stats 失败时降级为 null（UI 显示 "-"），不阻塞页面
      if (statsResult.status === 'fulfilled') {
        setStats(statsResult.value);
      }

      if (ordersResult.status === 'fulfilled') {
        setOrders(extractList<OrderSummary>(ordersResult.value).slice(0, 2));
      }

      setLoading(false);
    }

    loadProfile();

    return () => {
      alive = false;
    };
  }, [authUser]);

  const userInfo = useMemo<ProfileUser>(() => {
    const displayName = profile?.name || authUser?.name || '用户';
    return {
      ...authUser,
      ...profile,
      name: repairUtf8Mojibake(displayName),
    };
  }, [authUser, profile]);

  const handleLogout = async () => {
    setLoggingOut(true);
    try {
      await Promise.resolve(logout());
      navigate('/login');
    } finally {
      setLoggingOut(false);
    }
  };

  const trackAuctionHistoryClick = () => {
    trackEvent('entry_clicked', {
      source: 'profile',
      entry: 'auction_history',
      type: 'pending_payment',
      result: 'clicked',
    });
  };

  const trackNotificationCenterClick = () => {
    trackEvent('entry_clicked', {
      source: 'profile',
      entry: 'notification_center',
      type: 'notification',
      result: 'clicked',
    });
  };

  if (loading) {
    return (
      <section className={styles.statePage} aria-live="polite">
        <div className={styles.spinner} />
        <p>加载个人中心...</p>
      </section>
    );
  }

  if (error && !profile) {
    return (
      <section className={styles.statePage}>
        <p className={styles.errorText}>{error}</p>
        <button className={styles.retryButton} onClick={() => window.location.reload()}>
          重试
        </button>
      </section>
    );
  }

  return (
    <section className={styles.page}>
      <header className={styles.hero}>
        <div className={styles.avatarFrame}>
          {userInfo.avatar ? (
            <img src={userInfo.avatar} alt="用户头像" className={styles.avatar} />
          ) : (
            <span className={styles.avatarFallback}>{String(userInfo.name || '用').slice(0, 1)}</span>
          )}
        </div>
        <div className={styles.identity}>
          <p className={styles.eyebrow}>My Account</p>
          <h1>{userInfo.name}</h1>
          <div className={styles.badges}>
            <span>{roleLabel(userInfo.role)}</span>
            <span>ID: {userInfo.id ?? '---'}</span>
          </div>
        </div>
      </header>

      <div className={styles.statsGrid} aria-label="个人统计入口">
        <Link to="/following" className={styles.statCard}>
          <strong>{statValue(stats?.following_count)}</strong>
          <span>收藏</span>
        </Link>
        <Link to="/history" className={styles.statCard}>
          <strong>{statValue(stats?.auction_history_count)}</strong>
          <span>竞拍记录</span>
        </Link>
        <div className={styles.statCard}>
          <strong>{statValue(stats?.won_count)}</strong>
          <span>中标</span>
        </div>
      </div>

      <section className={styles.walletCard} aria-label="钱包余额">
        <div>
          <p className={styles.cardLabel}>钱包余额</p>
          <strong>{balance ? formatCurrency(pickAvailable(balance)) : '¥0'}</strong>
          {pickFrozen(balance) !== undefined && (
            <span>冻结 {formatCurrency(pickFrozen(balance))}</span>
          )}
        </div>
        <Link to="/addresses" className={styles.disabledAction} aria-label="管理收货地址">
          收货地址
        </Link>
      </section>

      <section className={styles.orderCard}>
        <div className={styles.sectionHeader}>
          <div>
            <p className={styles.cardLabel}>Orders</p>
            <h2>最近订单</h2>
          </div>
          <Link to="/history">全部</Link>
        </div>

        {orders.length > 0 ? (
          <div className={styles.orderList}>
            {orders.map((order) => (
              <Link key={order.id} to="/history" className={styles.orderItem}>
                <div>
                  <strong>{order.product_name || order.product?.name || `订单 #${order.id}`}</strong>
                  <span>{order.created_at ? new Date(order.created_at).toLocaleDateString() : '待更新'}</span>
                </div>
                <div className={styles.orderMeta}>
                  <strong>{formatCurrency(order.final_price)}</strong>
                  <span>{orderStatusLabel(order.status)}</span>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <p className={styles.emptyText}>暂无订单记录</p>
        )}
      </section>

      <nav className={styles.menu} aria-label="个人中心功能">
        <Link to="/history" className={styles.menuItem} onClick={trackAuctionHistoryClick}>
          <span className={styles.menuIcon}>A</span>
          <span className={styles.menuLabel}>
            我的竞拍
            <BadgeDot count={pendingPayment} className={styles.menuBadge} />
          </span>
          <b>›</b>
        </Link>
        <Link to="/following" className={styles.menuItem}>
          <span className={styles.menuIcon}>F</span>
          <span>我的收藏</span>
          <b>›</b>
        </Link>
        <Link to="/notifications" className={styles.menuItem} onClick={trackNotificationCenterClick}>
          <span className={styles.menuIcon}>N</span>
          <span>消息通知</span>
          <b>›</b>
        </Link>
        <Link to="/addresses" className={styles.menuItem}>
          <span className={styles.menuIcon}>D</span>
          <span>收货地址</span>
          <b>›</b>
        </Link>
        <Link to="/" className={`${styles.menuItem} ${styles.mutedItem}`}>
          <span className={styles.menuIcon}>S</span>
          <span>设置 (暂未开放)</span>
          <b>›</b>
        </Link>
      </nav>

      <button
        className={styles.logoutButton}
        type="button"
        onClick={handleLogout}
        disabled={loggingOut}
      >
        {loggingOut ? '退出中...' : '退出登录'}
      </button>
    </section>
  );
};

export default UserCenter;
