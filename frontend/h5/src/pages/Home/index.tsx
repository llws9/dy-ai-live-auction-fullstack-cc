// frontend/h5/src/pages/Home/index.tsx

import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';

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
          product_name: '限定款奢侈品包包',
          product_image: 'https://images.unsplash.com/photo-1548036328-c9fa89d128fa?w=400',
          status: 1,
          current_price: 150,
          end_time: new Date(Date.now() + 3600000).toISOString(),
          start_time: new Date().toISOString(),
          bidder_count: 12,
        },
        {
          id: 2,
          product_id: 2,
          product_name: '签名版限量球鞋',
          product_image: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400',
          status: 1,
          current_price: 280,
          end_time: new Date(Date.now() + 1800000).toISOString(),
          start_time: new Date().toISOString(),
          bidder_count: 8,
        },
        {
          id: 3,
          product_id: 3,
          product_name: '古董怀表收藏品',
          product_image: 'https://images.unsplash.com/photo-1509048191080-d2984bad6ae5?w=400',
          status: 3,
          current_price: 520,
          end_time: new Date(Date.now() - 3600000).toISOString(),
          start_time: new Date(Date.now() - 7200000).toISOString(),
          bidder_count: 25,
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    const configs: Record<number, { text: string; color: string; bgColor: string }> = {
      0: { text: '即将开始', color: '#fa8c16', bgColor: '#fff7e6' },
      1: { text: '进行中', color: '#52c41a', bgColor: '#f6ffed' },
      2: { text: '延时中', color: '#ff4d4f', bgColor: '#fff1f0' },
      3: { text: '已结束', color: '#999', bgColor: '#f5f5f5' },
      4: { text: '已取消', color: '#999', bgColor: '#f5f5f5' },
    };
    return configs[status] || { text: '未知', color: '#999', bgColor: '#f5f5f5' };
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
    <div style={styles.container}>
      {/* 顶部区域 */}
      <div style={styles.header}>
        <div style={styles.headerTop}>
          <h1 style={styles.logo}>🎯 直播竞拍</h1>
          <button style={styles.historyBtn} onClick={() => navigate('/history')}>
            📜 历史
          </button>
        </div>
        <p style={styles.subtitle}>发现好物，竞拍心仪商品</p>

        {/* 直播间入口 */}
        <Link to="/live" style={styles.liveEntry}>
          <div style={styles.liveEntryContent}>
            <div style={styles.liveIcon}>🎥</div>
            <div style={styles.liveText}>
              <div style={styles.liveTitle}>进入直播间</div>
              <div style={styles.liveDesc}>实时竞拍 · 互动体验</div>
            </div>
          </div>
          <div style={styles.liveBadge}>直播中</div>
        </Link>
      </div>

      {/* 标签筛选 */}
      <div style={styles.tabs}>
        <button
          style={{ ...styles.tab, ...(activeTab === 'all' ? styles.tabActive : {}) }}
          onClick={() => setActiveTab('all')}
        >
          全部
        </button>
        <button
          style={{ ...styles.tab, ...(activeTab === 'ongoing' ? styles.tabActive : {}) }}
          onClick={() => setActiveTab('ongoing')}
        >
          进行中
        </button>
        <button
          style={{ ...styles.tab, ...(activeTab === 'ended' ? styles.tabActive : {}) }}
          onClick={() => setActiveTab('ended')}
        >
          已结束
        </button>
      </div>

      {/* 竞拍列表 */}
      <div style={styles.content}>
        {loading ? (
          <div style={styles.loading}>
            <div style={styles.loadingSpinner}></div>
            <p>加载中...</p>
          </div>
        ) : filteredAuctions.length === 0 ? (
          <div style={styles.empty}>
            <span style={styles.emptyIcon}>📭</span>
            <p style={styles.emptyText}>暂无竞拍商品</p>
          </div>
        ) : (
          <div style={styles.grid}>
            {filteredAuctions.map((auction) => {
              const statusConfig = getStatusConfig(auction.status);
              const isActive = auction.status === 1 || auction.status === 2;

              return (
                <Link
                  key={auction.id}
                  to={`/auction/${auction.id}`}
                  style={styles.card}
                >
                  {/* 商品图片 */}
                  <div style={styles.cardImageWrapper}>
                    <img
                      src={auction.product_image || 'https://via.placeholder.com/400x300?text=No+Image'}
                      alt={auction.product_name || '商品'}
                      style={styles.cardImage}
                    />
                    {/* 状态标签 */}
                    <div style={{
                      ...styles.statusBadge,
                      backgroundColor: statusConfig.bgColor,
                      color: statusConfig.color,
                    }}>
                      {isActive && <span style={styles.liveDot}></span>}
                      {statusConfig.text}
                    </div>
                    {/* 竞拍人数 */}
                    {auction.bidder_count && (
                      <div style={styles.bidderCount}>
                        👥 {auction.bidder_count}人参与
                      </div>
                    )}
                  </div>

                  {/* 商品信息 */}
                  <div style={styles.cardContent}>
                    <h3 style={styles.cardTitle}>
                      {auction.product_name || `竞拍商品 #${auction.id}`}
                    </h3>

                    <div style={styles.priceRow}>
                      <div style={styles.priceSection}>
                        <span style={styles.priceLabel}>当前价</span>
                        <span style={styles.priceValue}>
                          ¥{auction.current_price.toLocaleString()}
                        </span>
                      </div>
                      {isActive && (
                        <span style={styles.timeBadge}>
                          ⏱️ {formatTime(auction.end_time)}
                        </span>
                      )}
                    </div>

                    {/* 出价按钮 */}
                    {isActive && (
                      <div style={styles.bidButton}>
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

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
  },
  header: {
    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    padding: '20px 16px',
    color: 'white',
  },
  headerTop: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '8px',
  },
  logo: {
    fontSize: '24px',
    margin: 0,
    fontWeight: 'bold',
  },
  historyBtn: {
    backgroundColor: 'rgba(255,255,255,0.2)',
    color: 'white',
    border: 'none',
    borderRadius: '20px',
    padding: '8px 16px',
    fontSize: '14px',
    cursor: 'pointer',
  },
  subtitle: {
    margin: 0,
    opacity: 0.9,
    fontSize: '14px',
  },
  tabs: {
    display: 'flex',
    gap: '8px',
    padding: '16px',
    backgroundColor: 'white',
    borderBottom: '1px solid #f0f0f0',
  },
  tab: {
    flex: 1,
    padding: '10px',
    backgroundColor: '#f5f5f5',
    border: 'none',
    borderRadius: '8px',
    fontSize: '14px',
    color: '#666',
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  tabActive: {
    backgroundColor: '#1890ff',
    color: 'white',
    fontWeight: 'bold',
  },
  content: {
    padding: '16px',
  },
  loading: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '60px 0',
    color: '#999',
  },
  loadingSpinner: {
    width: '40px',
    height: '40px',
    border: '3px solid #e0e0e0',
    borderTopColor: '#1890ff',
    borderRadius: '50%',
    animation: 'spin 1s linear infinite',
    marginBottom: '16px',
  },
  empty: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '60px 0',
  },
  emptyIcon: {
    fontSize: '48px',
    marginBottom: '16px',
  },
  emptyText: {
    color: '#999',
    margin: 0,
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))',
    gap: '12px',
  },
  card: {
    backgroundColor: 'white',
    borderRadius: '12px',
    overflow: 'hidden',
    textDecoration: 'none',
    color: 'inherit',
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    transition: 'transform 0.2s, box-shadow 0.2s',
  },
  cardImageWrapper: {
    position: 'relative',
    paddingTop: '75%',
    backgroundColor: '#f0f0f0',
  },
  cardImage: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: '100%',
    height: '100%',
    objectFit: 'cover',
  },
  statusBadge: {
    position: 'absolute',
    top: '8px',
    left: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
    padding: '4px 10px',
    borderRadius: '12px',
    fontSize: '12px',
    fontWeight: 'bold',
  },
  liveDot: {
    width: '6px',
    height: '6px',
    backgroundColor: '#52c41a',
    borderRadius: '50%',
    animation: 'pulse 1.5s infinite',
  },
  bidderCount: {
    position: 'absolute',
    bottom: '8px',
    right: '8px',
    backgroundColor: 'rgba(0,0,0,0.6)',
    color: 'white',
    padding: '4px 8px',
    borderRadius: '4px',
    fontSize: '11px',
  },
  cardContent: {
    padding: '12px',
  },
  cardTitle: {
    fontSize: '14px',
    margin: '0 0 8px 0',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    color: '#333',
  },
  priceRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-end',
    marginBottom: '8px',
  },
  priceSection: {
    display: 'flex',
    flexDirection: 'column',
  },
  priceLabel: {
    fontSize: '10px',
    color: '#999',
  },
  priceValue: {
    fontSize: '18px',
    fontWeight: 'bold',
    color: '#ff4d4f',
  },
  timeBadge: {
    fontSize: '10px',
    color: '#fa8c16',
    backgroundColor: '#fff7e6',
    padding: '2px 6px',
    borderRadius: '4px',
  },
  bidButton: {
    textAlign: 'center',
    padding: '8px',
    backgroundColor: '#1890ff',
    color: 'white',
    borderRadius: '6px',
    fontSize: '12px',
    fontWeight: 'bold',
  },
  // 直播间入口
  liveEntry: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: '16px',
    padding: '12px 16px',
    background: 'rgba(255,255,255,0.15)',
    borderRadius: '12px',
    textDecoration: 'none',
    backdropFilter: 'blur(10px)',
    border: '1px solid rgba(255,255,255,0.2)',
  },
  liveEntryContent: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  liveIcon: {
    width: '40px',
    height: '40px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'rgba(255,255,255,0.2)',
    borderRadius: '50%',
    fontSize: '20px',
  },
  liveText: {
    display: 'flex',
    flexDirection: 'column',
  },
  liveTitle: {
    fontSize: '16px',
    fontWeight: 600,
    color: 'white',
  },
  liveDesc: {
    fontSize: '12px',
    color: 'rgba(255,255,255,0.7)',
  },
};

export default HomePage;
