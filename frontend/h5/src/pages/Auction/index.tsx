// pages/Auction/index.tsx

import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import BidButton from '../../components/BidButton';
import PriceDisplay from '../../components/PriceDisplay';
import WebSocketService from '../../services/websocket';

interface Auction {
  id: number;
  product_id: number;
  status: number;
  current_price: number;
  winner_id?: number;
  start_time: string;
  end_time: string;
  delay_used: number;
}

interface BidRecord {
  id: number;
  user_id: number;
  user_name: string;
  amount: number;
  created_at: string;
}

interface RankItem {
  rank: number;
  user_id: number;
  user_name?: string;
  amount: number;
}

const AuctionPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [auction, setAuction] = useState<Auction | null>(null);
  const [loading, setLoading] = useState(true);
  const [bidRecords, setBidRecords] = useState<BidRecord[]>([]);
  const videoRef = useRef<HTMLVideoElement>(null);
  const wsRef = useRef<WebSocketService | null>(null);

  useEffect(() => {
    if (id) {
      fetchAuction();
      fetchBidRecords();
      connectWebSocket();
    }

    // 清理WebSocket连接
    return () => {
      if (wsRef.current) {
        wsRef.current.disconnect();
      }
    };
  }, [id]);

  const connectWebSocket = () => {
    if (!id) return;

    const auctionId = parseInt(id, 10);
    const ws = new WebSocketService(auctionId);

    // 注册排名更新处理器
    ws.on('rank_update', (data: { ranking: RankItem[] }) => {
      console.log('Received rank_update:', data);
      if (data.ranking) {
        // 转换排名数据为出价记录格式
        const records: BidRecord[] = data.ranking.map((item, index) => ({
          id: Date.now() + index,
          user_id: item.user_id,
          user_name: item.user_name || `用户${item.user_id}`,
          amount: item.amount,
          created_at: new Date().toISOString(),
        }));
        setBidRecords(records);
      }
    });

    // 注册出价通知处理器
    ws.on('bid_placed', (data: any) => {
      console.log('Received bid_placed:', data);
      if (auction && data.current_price) {
        setAuction({
          ...auction,
          current_price: data.current_price,
        });
      }
    });

    // 注册状态同步响应处理器（重连后）
    ws.on('sync_response', (data: any) => {
      console.log('Received sync_response:', data);
      if (data) {
        // 更新竞拍状态
        if (auction) {
          setAuction({
            ...auction,
            current_price: data.current_price,
            winner_id: data.winner_id,
            end_time: new Date(data.end_time).toISOString(),
            status: data.status,
          });
        }

        // 如果有排名数据，更新排名
        if (data.ranking) {
          const records: BidRecord[] = data.ranking.map((item: RankItem, index: number) => ({
            id: Date.now() + index,
            user_id: item.user_id,
            user_name: item.user_name || `用户${item.user_id}`,
            amount: item.amount,
            created_at: new Date().toISOString(),
          }));
          setBidRecords(records);
        }
      }
    });

    // 连接WebSocket
    ws.connect().then(() => {
      console.log('WebSocket connected successfully');
      wsRef.current = ws;

      // 重连成功后请求状态同步
      ws.requestSync();
    }).catch((error) => {
      console.error('WebSocket connection failed:', error);
    });
  };

  const fetchAuction = async () => {
    try {
      const response = await fetch(`/api/v1/auctions/${id}`);
      const data = await response.json();
      setAuction(data);
    } catch (error) {
      console.error('获取竞拍信息失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchBidRecords = async () => {
    try {
      const response = await fetch(`/api/v1/auctions/${id}/bids`);
      const data = await response.json();
      setBidRecords(data.bids || []);
    } catch (error) {
      console.error('获取出价记录失败:', error);
      // 模拟数据
      setBidRecords([
        { id: 1, user_id: 2, user_name: '用户A', amount: 150, created_at: new Date().toISOString() },
        { id: 2, user_id: 3, user_name: '用户B', amount: 140, created_at: new Date().toISOString() },
        { id: 3, user_id: 4, user_name: '用户C', amount: 130, created_at: new Date().toISOString() },
      ]);
    }
  };

  const handleBidSuccess = (newPrice: number) => {
    if (auction) {
      setAuction({
        ...auction,
        current_price: newPrice,
      });
      fetchBidRecords();
    }
  };

  if (loading) {
    return (
      <div style={styles.loadingContainer}>
        <div style={styles.loadingSpinner}></div>
        <p style={styles.loadingText}>加载中...</p>
      </div>
    );
  }

  if (!auction) {
    return (
      <div style={styles.errorContainer}>
        <div style={styles.errorIcon}>😢</div>
        <h3 style={styles.errorTitle}>竞拍不存在</h3>
        <button style={styles.backButton} onClick={() => navigate('/')}>
          返回首页
        </button>
      </div>
    );
  }

  const isActive = auction.status === 1 || auction.status === 2;

  return (
    <div style={styles.container}>
      {/* 视频背景区域 */}
      <div style={styles.videoSection}>
        <video
          ref={videoRef}
          style={styles.video}
          autoPlay
          loop
          muted
          playsInline
          poster="https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?w=800"
        >
          <source
            src="https://www.w3schools.com/html/mov_bbb.mp4"
            type="video/mp4"
          />
        </video>
        <div style={styles.videoOverlay}>
          {/* 顶部状态栏 */}
          <div style={styles.topBar}>
            <button style={styles.backBtn} onClick={() => navigate('/')}>
              ← 返回
            </button>
            <div style={styles.liveBadge}>
              <span style={styles.liveDot}></span>
              {getStatusText(auction.status)}
            </div>
          </div>

          {/* 倒计时悬浮 */}
          <div style={styles.floatingCountdown}>
            <PriceDisplay
              currentPrice={auction.current_price}
              endTime={auction.end_time}
            />
          </div>
        </div>
      </div>

      {/* 出价区域 */}
      <div style={styles.bidSection}>
        <div style={styles.bidHeader}>
          <h3 style={styles.bidTitle}>💰 出价竞拍</h3>
          {auction.delay_used > 0 && (
            <span style={styles.delayBadge}>已延时 {auction.delay_used}秒</span>
          )}
        </div>

        {isActive ? (
          <BidButton
            auctionId={auction.id}
            currentPrice={auction.current_price}
            increment={10}
            onBidSuccess={handleBidSuccess}
          />
        ) : (
          <div style={styles.endedMessage}>
            {auction.status === 3 ? (
              <>
                <span style={styles.endedIcon}>🏆</span>
                <p>竞拍已结束</p>
                <button
                  style={styles.viewResultBtn}
                  onClick={() => navigate(`/result/${auction.id}`)}
                >
                  查看结果
                </button>
              </>
            ) : auction.status === 4 ? (
              <p>竞拍已取消</p>
            ) : (
              <p>竞拍即将开始</p>
            )}
          </div>
        )}
      </div>

      {/* 出价记录排行 */}
      <div style={styles.rankingSection}>
        <h3 style={styles.rankingTitle}>📊 出价排行</h3>
        {bidRecords.length > 0 ? (
          <div style={styles.rankingList}>
            {bidRecords.slice(0, 5).map((record, index) => (
              <div
                key={record.id}
                style={{
                  ...styles.rankingItem,
                  ...(index === 0 ? styles.rankingItemTop1 : {}),
                  ...(index === 1 ? styles.rankingItemTop2 : {}),
                  ...(index === 2 ? styles.rankingItemTop3 : {}),
                }}
              >
                <div style={styles.rankingLeft}>
                  <span style={{
                    ...styles.rankingNum,
                    ...(index === 0 ? styles.rankingNum1 : {}),
                    ...(index === 1 ? styles.rankingNum2 : {}),
                    ...(index === 2 ? styles.rankingNum3 : {}),
                  }}>
                    {index + 1}
                  </span>
                  <span style={styles.rankingName}>{record.user_name}</span>
                </div>
                <span style={styles.rankingAmount}>¥{record.amount}</span>
              </div>
            ))}
          </div>
        ) : (
          <div style={styles.emptyRanking}>暂无出价记录</div>
        )}
      </div>

      {/* 竞拍信息 */}
      <div style={styles.infoSection}>
        <h3 style={styles.infoTitle}>📋 竞拍详情</h3>
        <div style={styles.infoGrid}>
          <div style={styles.infoItem}>
            <span style={styles.infoLabel}>竞拍ID</span>
            <span style={styles.infoValue}>{auction.id}</span>
          </div>
          <div style={styles.infoItem}>
            <span style={styles.infoLabel}>状态</span>
            <span style={{
              ...styles.infoValue,
              ...(isActive ? styles.statusActive : styles.statusEnded),
            }}>
              {getStatusText(auction.status)}
            </span>
          </div>
          <div style={styles.infoItem}>
            <span style={styles.infoLabel}>开始时间</span>
            <span style={styles.infoValue}>{new Date(auction.start_time).toLocaleString()}</span>
          </div>
          <div style={styles.infoItem}>
            <span style={styles.infoLabel}>结束时间</span>
            <span style={styles.infoValue}>{new Date(auction.end_time).toLocaleString()}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
    paddingBottom: '20px',
  },
  loadingContainer: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100vh',
    backgroundColor: '#f5f5f5',
  },
  loadingSpinner: {
    width: '40px',
    height: '40px',
    border: '3px solid #e0e0e0',
    borderTopColor: '#1890ff',
    borderRadius: '50%',
    animation: 'spin 1s linear infinite',
  },
  loadingText: {
    marginTop: '16px',
    color: '#666',
  },
  errorContainer: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100vh',
    padding: '20px',
  },
  errorIcon: {
    fontSize: '64px',
    marginBottom: '16px',
  },
  errorTitle: {
    fontSize: '20px',
    color: '#333',
    marginBottom: '20px',
  },
  backButton: {
    padding: '12px 24px',
    backgroundColor: '#1890ff',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    fontSize: '16px',
    cursor: 'pointer',
  },
  videoSection: {
    position: 'relative',
    width: '100%',
    height: '400px',
    overflow: 'hidden',
  },
  video: {
    width: '100%',
    height: '100%',
    objectFit: 'cover',
  },
  videoOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'linear-gradient(to bottom, rgba(0,0,0,0.3) 0%, rgba(0,0,0,0.1) 50%, rgba(0,0,0,0.7) 100%)',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
  },
  topBar: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '16px',
  },
  backBtn: {
    backgroundColor: 'rgba(0,0,0,0.5)',
    color: 'white',
    border: 'none',
    borderRadius: '20px',
    padding: '8px 16px',
    fontSize: '14px',
    cursor: 'pointer',
    backdropFilter: 'blur(10px)',
  },
  liveBadge: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    backgroundColor: 'rgba(255,77,79,0.9)',
    color: 'white',
    padding: '6px 12px',
    borderRadius: '20px',
    fontSize: '12px',
    fontWeight: 'bold',
  },
  liveDot: {
    width: '6px',
    height: '6px',
    backgroundColor: 'white',
    borderRadius: '50%',
    animation: 'pulse 1.5s infinite',
  },
  floatingCountdown: {
    padding: '16px',
  },
  bidSection: {
    margin: '-30px 16px 16px',
    backgroundColor: 'white',
    borderRadius: '16px',
    padding: '20px',
    boxShadow: '0 4px 20px rgba(0,0,0,0.1)',
    position: 'relative',
    zIndex: 10,
  },
  bidHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  bidTitle: {
    fontSize: '18px',
    margin: 0,
  },
  delayBadge: {
    backgroundColor: '#fff7e6',
    color: '#fa8c16',
    padding: '4px 10px',
    borderRadius: '12px',
    fontSize: '12px',
  },
  endedMessage: {
    textAlign: 'center',
    padding: '30px 0',
    color: '#666',
  },
  endedIcon: {
    fontSize: '48px',
    display: 'block',
    marginBottom: '10px',
  },
  viewResultBtn: {
    marginTop: '16px',
    padding: '12px 24px',
    backgroundColor: '#1890ff',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    fontSize: '16px',
    cursor: 'pointer',
  },
  rankingSection: {
    margin: '0 16px 16px',
    backgroundColor: 'white',
    borderRadius: '16px',
    padding: '20px',
    boxShadow: '0 2px 8px rgba(0,0,0,0.05)',
  },
  rankingTitle: {
    fontSize: '16px',
    margin: '0 0 16px 0',
    color: '#333',
  },
  rankingList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  rankingItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '12px',
    backgroundColor: '#f9f9f9',
    borderRadius: '8px',
  },
  rankingItemTop1: {
    background: 'linear-gradient(135deg, #ffd700 0%, #ffb700 100%)',
  },
  rankingItemTop2: {
    background: 'linear-gradient(135deg, #e0e0e0 0%, #c0c0c0 100%)',
  },
  rankingItemTop3: {
    background: 'linear-gradient(135deg, #cd7f32 0%, #b5651d 100%)',
    color: 'white',
  },
  rankingLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  rankingNum: {
    width: '24px',
    height: '24px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#e0e0e0',
    borderRadius: '50%',
    fontSize: '12px',
    fontWeight: 'bold',
  },
  rankingNum1: {
    backgroundColor: '#ffd700',
  },
  rankingNum2: {
    backgroundColor: '#c0c0c0',
  },
  rankingNum3: {
    backgroundColor: '#cd7f32',
    color: 'white',
  },
  rankingName: {
    fontSize: '14px',
  },
  rankingAmount: {
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#ff4d4f',
  },
  emptyRanking: {
    textAlign: 'center',
    padding: '20px',
    color: '#999',
  },
  infoSection: {
    margin: '0 16px',
    backgroundColor: 'white',
    borderRadius: '16px',
    padding: '20px',
    boxShadow: '0 2px 8px rgba(0,0,0,0.05)',
  },
  infoTitle: {
    fontSize: '16px',
    margin: '0 0 16px 0',
    color: '#333',
  },
  infoGrid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '16px',
  },
  infoItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
  },
  infoLabel: {
    fontSize: '12px',
    color: '#999',
  },
  infoValue: {
    fontSize: '14px',
    color: '#333',
  },
  statusActive: {
    color: '#52c41a',
  },
  statusEnded: {
    color: '#999',
  },
};

function getStatusText(status: number): string {
  const statusMap: Record<number, string> = {
    0: '待开始',
    1: '进行中',
    2: '延时中',
    3: '已结束',
    4: '已取消',
  };
  return statusMap[status] || '未知状态';
}

export default AuctionPage;
