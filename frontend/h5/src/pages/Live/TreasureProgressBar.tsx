import React, { useEffect, useState, useCallback } from 'react';
import { treasureApi } from '@/services/api';
import { useToast } from '@/components/Toast';
import styles from './TreasureProgressBar.module.css';

interface TreasureTier {
  tier: number;
  threshold_seconds: number;
  coins: number;
  state: 'locked' | 'unlockable' | 'claimed';
}

interface TreasureStatus {
  watched_seconds: number;
  coin_balance: number;
  tiers: TreasureTier[];
}

interface TreasureProgressBarProps {
  liveStreamId: number;
  isAuthenticated: boolean;
}

export const TreasureProgressBar: React.FC<TreasureProgressBarProps> = ({ liveStreamId, isAuthenticated }) => {
  const { showToast } = useToast();
  const [status, setStatus] = useState<TreasureStatus | null>(null);
  const [animatingTier, setAnimatingTier] = useState<number | null>(null);

  // 用于乐观更新时长，使进度条更平滑
  const [localSeconds, setLocalSeconds] = useState(0);

  const fetchStatus = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const res = await treasureApi.getStatus();
      setStatus(res);
      setLocalSeconds(res.watched_seconds || 0);
    } catch (err) {
      // 忽略错误，防止频繁弹窗
    }
  }, [isAuthenticated]);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  // 心跳和本地计时逻辑
  useEffect(() => {
    if (!isAuthenticated || !liveStreamId) return;

    let heartbeatTimer: number;
    let localTimer: number;

    const startTimers = () => {
      // 每30秒发一次心跳
      heartbeatTimer = window.setInterval(async () => {
        if (document.visibilityState === 'visible') {
          try {
            await treasureApi.heartbeat(liveStreamId);
            // 心跳后重新拉取状态以确保同步
            fetchStatus();
          } catch (e) {
            // ignore
          }
        }
      }, 30000);

      // 每秒更新一次本地显示时长（仅前端表现）
      localTimer = window.setInterval(() => {
        if (document.visibilityState === 'visible') {
          setLocalSeconds(prev => {
            const next = prev + 1;
            // 简单判断是否达到某个节点，如果是，主动刷新状态
            if (status?.tiers) {
              const reachedTier = status.tiers.find(t => t.state === 'locked' && next >= t.threshold_seconds);
              if (reachedTier) {
                fetchStatus(); // 达到门槛时主动向后端确认
              }
            }
            return next;
          });
        }
      }, 1000);
    };

    const clearTimers = () => {
      window.clearInterval(heartbeatTimer);
      window.clearInterval(localTimer);
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        fetchStatus(); // 回到页面时对齐权威态
        startTimers();
      } else {
        clearTimers();
      }
    };

    // 初始化启动
    if (document.visibilityState === 'visible') {
      startTimers();
    }

    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      clearTimers();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [isAuthenticated, liveStreamId, fetchStatus, status?.tiers]);

  const handleClaim = async (tier: TreasureTier) => {
    if (!isAuthenticated) {
      showToast('请先登录');
      return;
    }
    if (tier.state !== 'unlockable') return;

    try {
      setAnimatingTier(tier.tier);
      const res = await treasureApi.claim(tier.tier);

      // 延迟更新状态，让动画播完
      setTimeout(() => {
        setStatus(prev => prev ? {
          ...prev,
          coin_balance: res.coin_balance ?? (prev.coin_balance + tier.coins),
          tiers: prev.tiers.map(t => t.tier === tier.tier ? { ...t, state: 'claimed' } : t)
        } : prev);
        setAnimatingTier(null);
      }, 1000);
    } catch (err: any) {
      showToast(err?.message || '领取失败');
      setAnimatingTier(null);
      fetchStatus(); // 失败后重拉状态
    }
  };

  // 默认 mock 节点位置和数据，即使没登录也能展示进度条
  const displayTiers = status?.tiers || [
    { tier: 0, threshold_seconds: 180, coins: 100, state: 'locked' },
    { tier: 1, threshold_seconds: 600, coins: 300, state: 'locked' },
    { tier: 2, threshold_seconds: 1800, coins: 800, state: 'locked' }
  ] as TreasureTier[];

  const displayCoins = status?.coin_balance || 0;
  const maxSeconds = 1800; // 30分钟为满
  const progressPercent = Math.min(100, Math.max(0, (localSeconds / maxSeconds) * 100));

  return (
    <div className={styles.container}>
      <div className={styles.glassPanel}>
        <div className={styles.header}>
          <span>观看时长领奖励</span>
          <div className={styles.coinDisplay}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <circle cx="12" cy="12" r="10"></circle>
              <text x="12" y="16" fontSize="12" fill="#000" textAnchor="middle" fontWeight="bold">¥</text>
            </svg>
            <span>{displayCoins.toLocaleString()}</span>
          </div>
        </div>
        <div className={styles.trackWrap}>
          <div className={styles.track}>
            <div className={styles.fill} style={{ width: `${progressPercent}%` }}></div>
          </div>

          {displayTiers.map((tier) => {
            const leftPos = (tier.threshold_seconds / maxSeconds) * 100;
            const isUnlockable = tier.state === 'unlockable' && animatingTier !== tier.tier;
            const isAnimating = animatingTier === tier.tier;
            const isClaimed = tier.state === 'claimed' || isAnimating;

            return (
              <div
                key={tier.tier}
                className={`${styles.node} ${isUnlockable ? styles.unlockable : ''}`}
                style={{ left: `${leftPos}%` }}
                onClick={() => handleClaim(tier)}
                title={tier.threshold_seconds / 60 + '分钟'}
              >
                {isAnimating && <div className={styles.floatCoin}>+ {tier.coins}</div>}
                <svg className={`${styles.chestSvg} ${tier.state === 'locked' ? styles.chestLocked : ''} ${isClaimed ? styles.chestClaimed : ''}`} viewBox="0 0 24 24">
                  <path d="M4 10h16v10a2 2 0 01-2 2H6a2 2 0 01-2-2V10z" fill="#f59f00"/>
                  <path d="M2 6a2 2 0 012-2h16a2 2 0 012 2v4H2V6z" fill="#fcc419"/>
                  <rect x="10" y="8" width="4" height="4" fill="#fff" rx="1"/>
                </svg>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};
