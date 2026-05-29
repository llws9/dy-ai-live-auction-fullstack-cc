// frontend/h5/src/pages/Home/index.tsx

import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Skeleton } from '@/components/shared';
import styles from './Home.module.css';

interface Auction {
  id: number;
  product_id: number;
  product_name?: string;
  product_image?: string;
  status: number;
  current_price: number;
  end_time: string;
  start_time: string;
  bidder_count?: number;
}

const HomePage: React.FC = () => {
  const navigate = useNavigate();
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'all' | 'ongoing' | 'ended'>('all');

  useEffect(() => {
    fetchAuctions();
  }, []);

  const fetchAuctions = async () => {
    try {
      const response = await fetch('/api/v1/auctions');
      const data = await response.json();
      setAuctions(data.auctions || []);
    } catch (error) {
      console.error('获取竞拍列表失败:', error);
      // 模拟数据
      setAuctions([
        {
          id: 1,
          product_id: 1,
          product_name: '等待加载真实数据...',
          product_image: '',
          status: 1,
          current_price: 0,
          end_time: new Date(Date.now() + 3600000).toISOString(),
          start_time: new Date().toISOString(),
          bidder_count: 0,
        }
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    const configs: Record<number, { text: string; className: string }> = {
      0: { text: '即将开始', className: styles.statusPending },
      1: { text: '进行中', className: styles.statusOngoing },
      2: { text: '延时中', className: styles.statusOngoing },
      3: { text: '已结束', className: styles.statusEnded },
      4: { text: '已取消', className: styles.statusEnded },
    };
    return configs[status] || { text: '未知', className: styles.statusEnded };
  };

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = date.getTime() - now.getTime();

    if (diff > 0) {
      const minutes = Math.floor(diff / 60000);
      if (minutes < 60) return `${minutes}分钟后结束`;
      const hours = Math.floor(minutes / 60);
      return `${hours}小时后结束`;
    }
    return '已结束';
  };

  const filteredAuctions = auctions.filter((auction) => {
    if (activeTab === 'ongoing') return auction.status === 1 || auction.status === 2;
    if (activeTab === 'ended') return auction.status === 3 || auction.status === 4;
    return true;
  });

  return (
    <div className={styles.container}>
      {/* 顶部区域 */}
      <header className={styles.header}>
        <div className={styles.headerTop}>
          <h1 className={styles.logo}>🎯 直播竞拍</h1>
          <div className={styles.headerActions}>
            <button className={styles.headerBtn} onClick={() => navigate('/follow')}>
              ❤️ 关注
            </button>
            <button className={styles.headerBtn} onClick={() => navigate('/history')}>
              📜 历史
            </button>
          </div>
        </div>
        <p className={styles.subtitle}>发现好物，竞拍心仪商品</p>

        {/* 直播间入口 */}
        <Link to="/live" className={styles.liveEntry}>
          <div className={styles.liveEntryContent}>
            <div className={styles.liveIcon}>🎥</div>
            <div className={styles.liveText}>
              <div className={styles.liveTitle}>进入直播间</div>
              <div className={styles.liveDesc}>实时竞拍 · 互动体验</div>
            </div>
          </div>
          <div className={styles.liveBadge}>直播中</div>
        </Link>
      </header>

      {/* 标签筛选 */}
      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${activeTab === 'all' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('all')}
        >
          全部
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'ongoing' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('ongoing')}
        >
          进行中
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'ended' ? styles.tabActive : ''}`}
          onClick={() => setActiveTab('ended')}
        >
          已结束
        </button>
      </div>

      {/* 竞拍列表 */}
      <div className={styles.content}>
        {loading ? (
          <div className={styles.grid}>
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className={styles.card}>
                <Skeleton variant="rectangular" height={180} />
                <div className={styles.cardContent}>
                  <Skeleton variant="text" width="80%" />
                  <Skeleton variant="text" width="50%" />
                </div>
              </div>
            ))}
          </div>
        ) : filteredAuctions.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>📭</span>
            <p className={styles.emptyText}>暂无竞拍商品</p>
          </div>
        ) : (
          <div className={styles.grid}>
            {filteredAuctions.map((auction) => {
              const statusConfig = getStatusConfig(auction.status);
              const isActive = auction.status === 1 || auction.status === 2;

              return (
                <Link
                  key={auction.id}
                  to={`/auction/${auction.id}`}
                  className={styles.card}
                >
                  {/* 商品图片 */}
                  <div className={styles.cardImageWrapper}>
                    <img
                      src={auction.product_image || 'https://via.placeholder.com/400x300?text=No+Image'}
                      alt={auction.product_name || '商品'}
                      className={styles.cardImage}
                      loading="lazy"
                    />
                    {/* 状态标签 */}
                    <div className={`${styles.statusBadge} ${statusConfig.className}`}>
                      {isActive && <span className={styles.liveDot} />}
                      {statusConfig.text}
                    </div>
                    {/* 竞拍人数 */}
                    {auction.bidder_count && (
                      <div className={styles.bidderCount}>
                        👥 {auction.bidder_count}人参与
                      </div>
                    )}
                  </div>

                  {/* 商品信息 */}
                  <div className={styles.cardContent}>
                    <h3 className={styles.cardTitle}>
                      {auction.product_name || `竞拍商品 #${auction.id}`}
                    </h3>

                    <div className={styles.priceRow}>
                      <div className={styles.priceSection}>
                        <span className={styles.priceLabel}>当前价</span>
                        <span className={styles.priceValue}>
                          ¥{auction.current_price.toLocaleString()}
                        </span>
                      </div>
                      {isActive && (
                        <span className={styles.timeBadge}>
                          ⏱️ {formatTime(auction.end_time)}
                        </span>
                      )}
                    </div>

                    {/* 出价按钮 */}
                    {isActive && (
                      <div className={styles.bidButton}>
                        立即出价 →
                      </div>
                    )}
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default HomePage;
