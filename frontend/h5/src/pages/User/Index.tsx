import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { orderApi, userApi } from '../../services/api';
import { useAuth } from '../../store/authContext';
import BadgeDot from '../../components/BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import { getLiveRoomFootprints } from '../../utils/liveRoomFootprints';
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

function roleLabel(role?: number) {
  if (role === 2) return '管理员';
  if (role === 1) return '商家/主播';
  return '普通用户';
}

const UserCenter: React.FC = () => {
  const { user: authUser, logout } = useAuth();
  const { unreadTotal, wonNotPaid } = useTouchpointNotifications();
  const navigate = useNavigate();
  const [profile, setProfile] = useState<ProfileUser | null>(authUser);
  const [balance, setBalance] = useState<BalanceData | null>(null);
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    let alive = true;

    async function loadProfile() {
      setLoading(true);
      setError(null);

      const [profileResult, balanceResult, statsResult] = await Promise.allSettled([
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
  const footprints = useMemo(() => getLiveRoomFootprints(), []);
  const pendingAuctionCount = wonNotPaid;

  const handleLogout = async () => {
    setLoggingOut(true);
    try {
      await Promise.resolve(logout());
      navigate('/login');
    } finally {
      setLoggingOut(false);
    }
  };

  const trackPendingPaymentClick = () => {
    trackEvent('entry_clicked', {
      source: 'profile',
      entry: 'orders',
      type: 'pending_payment',
      result: 'clicked',
    });
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

      <section className={styles.auctionCommandCard} aria-label="我的竞拍">
        <div className={styles.sectionHeader}>
          <div>
            <p className={styles.cardLabel}>Auction</p>
            <h2>我的竞拍</h2>
          </div>
          <span>记录含中标</span>
        </div>
        <Link to="/orders" className={styles.primaryAuctionCta} onClick={trackPendingPaymentClick}>
          <div>
            <strong>{pendingAuctionCount > 0 ? `${pendingAuctionCount} 件中标待支付` : '查看竞拍记录'}</strong>
            <span>去我的订单完成支付</span>
          </div>
          <b>›</b>
        </Link>
        <div className={styles.auctionMetrics}>
          <Link to="/history" className={styles.metricCard} onClick={trackAuctionHistoryClick}>
            <BadgeDot count={pendingAuctionCount} className={styles.metricBadge} />
            <strong>{statValue(stats?.auction_history_count)}</strong>
            <span>竞拍记录</span>
          </Link>
          <Link to="/notifications" className={styles.metricCard} onClick={trackNotificationCenterClick}>
            <BadgeDot count={unreadTotal} className={styles.metricBadge} />
            <strong>{statValue(unreadTotal)}</strong>
            <span>消息通知</span>
          </Link>
          <Link to="/following" className={styles.metricCard}>
            <strong>{statValue(stats?.following_count)}</strong>
            <span>收藏</span>
          </Link>
        </div>
      </section>

      <section className={styles.footprintCard} aria-label="最近浏览直播间">
        <div className={styles.sectionHeader}>
          <div>
            <p className={styles.cardLabel}>Footprints</p>
            <h2>足迹</h2>
          </div>
          <span>最近 10 个直播间</span>
        </div>
        {footprints.length > 0 ? (
          <div className={styles.footprintList}>
            {footprints.map((item) => (
              <Link
                key={item.live_stream_id}
                to={`/live?live_stream_id=${item.live_stream_id}`}
                className={styles.footprintItem}
              >
                <div
                  className={styles.footprintCover}
                  style={item.cover ? { backgroundImage: `url(${item.cover})` } : undefined}
                />
                <strong>{repairUtf8Mojibake(item.name)}</strong>
                <span>最近浏览</span>
              </Link>
            ))}
          </div>
        ) : (
          <p className={styles.emptyText}>暂无直播间浏览足迹</p>
        )}
      </section>

      <section className={styles.serviceGrid} aria-label="账户与服务">
        <Link to="/wallet" className={styles.serviceItem}>
          <span className={styles.serviceIcon}>¥</span>
          <span className={styles.serviceText}>
            <strong>钱包</strong>
            <small>{balance ? `可用 ${formatCurrency(pickAvailable(balance))}` : '可用 ¥0'}</small>
          </span>
        </Link>
        <Link to="/addresses" className={styles.serviceItem}>
          <span className={styles.serviceIcon}>D</span>
          <span className={styles.serviceText}>
            <strong>收货地址</strong>
            <small>管理配送</small>
          </span>
        </Link>
        <Link to="/" className={styles.serviceItem}>
          <span className={styles.newBadge}>新</span>
          <span className={styles.serviceIcon}>S</span>
          <span className={styles.serviceText}>
            <strong>个人卖家申请</strong>
            <small>暂未开放</small>
          </span>
        </Link>
        <Link to="/" className={styles.serviceItem}>
          <span className={styles.newBadge}>新</span>
          <span className={styles.serviceIcon}>B</span>
          <span className={styles.serviceText}>
            <strong>企业商家入驻</strong>
            <small>暂未开放</small>
          </span>
        </Link>
      </section>

      <nav className={styles.secondaryMenu} aria-label="个人中心功能">
        <Link to="/" className={styles.secondaryItem}>
          设置（暂未开放）
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
