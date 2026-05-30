import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { orderApi, userApi } from '../../services/api';
import { useAuth } from '../../store/authContext';
import BadgeDot from '../../components/BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import styles from './Profile.module.css';

interface ProfileUser {
  id?: number | string;
  name?: string;
  email?: string;
  avatar?: string;
  role?: number;
}

interface BalanceData {
  balance?: number | string;
  available_balance?: number | string;
  frozen_amount?: number | string;
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
  const [orders, setOrders] = useState<OrderSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    let alive = true;

    async function loadProfile() {
      setLoading(true);
      setError(null);

      const [profileResult, balanceResult, ordersResult] = await Promise.allSettled([
        userApi.getProfile(),
        userApi.getBalance(),
        orderApi.list(),
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
    return {
      ...authUser,
      ...profile,
      name: profile?.name || authUser?.name || '用户',
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
          <strong>---</strong>
          <span>关注</span>
        </Link>
        <div className={styles.statCard}>
          <strong>---</strong>
          <span>粉丝</span>
        </div>
        <Link to="/history" className={styles.statCard}>
          <strong>{orders.length || '---'}</strong>
          <span>竞拍记录</span>
        </Link>
      </div>

      <section className={styles.walletCard} aria-label="钱包余额">
        <div>
          <p className={styles.cardLabel}>钱包余额</p>
          <strong>{balance ? formatCurrency(balance.balance) : '暂不可用'}</strong>
          {balance?.available_balance !== undefined && (
            <span>可用 {formatCurrency(balance.available_balance)}</span>
          )}
        </div>
        <button className={styles.disabledAction} type="button" disabled>
          充值待开放
        </button>
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
        <Link to="/history" className={styles.menuItem}>
          <span className={styles.menuIcon}>A</span>
          <span className={styles.menuLabel}>
            我的竞拍
            <BadgeDot count={pendingPayment} className={styles.menuBadge} />
          </span>
          <b>›</b>
        </Link>
        <Link to="/following" className={styles.menuItem}>
          <span className={styles.menuIcon}>F</span>
          <span>关注直播</span>
          <b>›</b>
        </Link>
        <Link to="/notifications" className={styles.menuItem}>
          <span className={styles.menuIcon}>N</span>
          <span>消息通知</span>
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
