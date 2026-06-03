// pages/Auction/index.tsx

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { Button, Loading } from '@/components/shared';
import BidButton from '../../components/BidButton';
import PriceDisplay from '../../components/PriceDisplay';
import WebSocketService from '../../services/websocket';
import styles from './Auction.module.css';

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

const toMoneyNumber = (value: unknown, fallback = 0) => {
  const n = Number(value);
  return Number.isFinite(n) ? n : fallback;
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

const AuctionPage: React.FC = () => {
  const { id: pathId } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const id = pathId ?? searchParams.get('auction_id') ?? searchParams.get('id');
  const navigate = useNavigate();
  const [auction, setAuction] = useState<Auction | null>(null);
  const [loading, setLoading] = useState(true);
  const [bidRecords, setBidRecords] = useState<BidRecord[]>([]);
  const videoRef = useRef<HTMLVideoElement>(null);
  const wsRef = useRef<WebSocketService | null>(null);

  const connectWebSocket = useCallback(() => {
    if (!id) return;

    const auctionId = parseInt(id, 10);
    const ws = new WebSocketService(auctionId);

    ws.on('rank_update', (data: { ranking: RankItem[] }) => {
      if (data.ranking) {
        const records: BidRecord[] = data.ranking.map((item, index) => ({
          id: Date.now() + index,
          user_id: item.user_id,
          user_name: item.user_name || `用户${item.user_id}`,
          amount: toMoneyNumber(item.amount),
          created_at: new Date().toISOString(),
        }));
        setBidRecords(records);
      }
    });

    ws.on('bid_placed', (data: any) => {
      if (data.current_price) {
        setAuction((prev) => (prev ? { ...prev, current_price: toMoneyNumber(data.current_price, prev.current_price) } : prev));
      }
    });

    ws.on('sync_response', (data: any) => {
      if (data) {
        setAuction((prev) => (
          prev
            ? {
              ...prev,
            current_price: toMoneyNumber(data.current_price, prev.current_price),
            winner_id: data.winner_id,
            end_time: new Date(data.end_time).toISOString(),
            status: data.status,
            }
            : prev
        ));

        if (data.ranking) {
          const records: BidRecord[] = data.ranking.map((item: RankItem, index: number) => ({
            id: Date.now() + index,
            user_id: item.user_id,
            user_name: item.user_name || `用户${item.user_id}`,
            amount: toMoneyNumber(item.amount),
            created_at: new Date().toISOString(),
          }));
          setBidRecords(records);
        }
      }
    });

    ws.connect().then(() => {
      wsRef.current = ws;
      ws.requestSync();
    }).catch((error) => {
      console.error('WebSocket connection failed:', error);
    });
  }, [id]);

  const fetchAuction = useCallback(async () => {
    if (!id) return;

    try {
      const response = await fetch(`/api/v1/auctions/${id}`);
      const data = await response.json();
      setAuction(data);
    } catch (error) {
      console.error('获取竞拍信息失败:', error);
    } finally {
      setLoading(false);
    }
  }, [id]);

  const fetchBidRecords = useCallback(async () => {
    if (!id) return;

    try {
      const response = await fetch(`/api/v1/auctions/${id}/bids`);
      const data = await response.json();
      setBidRecords(data.bids || []);
    } catch (error) {
      console.error('获取出价记录失败:', error);
      setBidRecords([
        { id: 1, user_id: 2, user_name: '用户A', amount: 150, created_at: new Date().toISOString() },
        { id: 2, user_id: 3, user_name: '用户B', amount: 140, created_at: new Date().toISOString() },
        { id: 3, user_id: 4, user_name: '用户C', amount: 130, created_at: new Date().toISOString() },
      ]);
    }
  }, [id]);

  useEffect(() => {
    if (!id) {
      setLoading(false);
      return;
    }

    fetchAuction();
    fetchBidRecords();
    connectWebSocket();

    return () => {
      if (wsRef.current) {
        wsRef.current.disconnect();
      }
    };
  }, [id, fetchAuction, fetchBidRecords, connectWebSocket]);

  const handleBidSuccess = (newPrice: number) => {
    if (auction) {
      setAuction({ ...auction, current_price: newPrice });
      fetchBidRecords();
    }
  };

  if (loading) {
    return (
      <div className={styles.loadingContainer}>
        <Loading size="lg" />
      </div>
    );
  }

  if (!auction) {
    return (
      <div className={styles.errorContainer}>
        <div className={styles.errorIcon}>😢</div>
        <h3 className={styles.errorTitle}>竞拍不存在</h3>
        <Button onClick={() => navigate('/')}>返回首页</Button>
      </div>
    );
  }

  const isActive = auction.status === 1 || auction.status === 2;
  const getRankingItemClass = (index: number): string => {
    if (index === 0) return `${styles.rankingItem} ${styles.rankingItemTop1}`;
    if (index === 1) return `${styles.rankingItem} ${styles.rankingItemTop2}`;
    if (index === 2) return `${styles.rankingItem} ${styles.rankingItemTop3}`;
    return styles.rankingItem;
  };

  return (
    <div className={styles.container}>
      {/* 视频背景区域 */}
      <div className={styles.videoSection}>
        <video
          ref={videoRef}
          className={styles.video}
          autoPlay
          loop
          muted
          playsInline
          poster="https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?w=800"
        >
          <source src="https://www.w3schools.com/html/mov_bbb.mp4" type="video/mp4" />
        </video>
        <div className={styles.videoOverlay}>
          {/* 顶部状态栏 */}
          <div className={styles.topBar}>
            <button className={styles.backBtn} onClick={() => navigate('/')}>
              ← 返回
            </button>
            <div className={styles.liveBadge}>
              <span className={styles.liveDot} />
              {getStatusText(auction.status)}
            </div>
          </div>

          {/* 倒计时悬浮 */}
          <div className={styles.floatingCountdown}>
            <PriceDisplay
              currentPrice={auction.current_price}
              endTime={auction.end_time}
            />
          </div>
        </div>
      </div>

      {/* 出价区域 */}
      <div className={styles.bidSection}>
        <div className={styles.bidHeader}>
          <h3 className={styles.bidTitle}>💰 出价竞拍</h3>
          {auction.delay_used > 0 && (
            <span className={styles.delayBadge}>已延时 {auction.delay_used}秒</span>
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
          <div className={styles.endedMessage}>
            {auction.status === 3 ? (
              <>
                <span className={styles.endedIcon}>🏆</span>
                <p>竞拍已结束</p>
                <button
                  className={styles.viewResultBtn}
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
      <div className={styles.rankingSection}>
        <h3 className={styles.rankingTitle}>📊 出价排行</h3>
        {bidRecords.length > 0 ? (
          <div className={styles.rankingList}>
            {bidRecords.slice(0, 5).map((record, index) => (
              <div key={record.id} className={getRankingItemClass(index)}>
                <div className={styles.rankingLeft}>
                  <span className={`${styles.rankingNum} ${
                    index === 0 ? styles.rankingNum1 :
                    index === 1 ? styles.rankingNum2 :
                    index === 2 ? styles.rankingNum3 : ''
                  }`}>
                    {index + 1}
                  </span>
                  <span className={styles.rankingName}>{record.user_name}</span>
                </div>
                <span className={styles.rankingAmount}>¥{record.amount}</span>
              </div>
            ))}
          </div>
        ) : (
          <div className={styles.emptyRanking}>暂无出价记录</div>
        )}
      </div>

      {/* 竞拍信息 */}
      <div className={styles.infoSection}>
        <h3 className={styles.infoTitle}>📋 竞拍详情</h3>
        <div className={styles.infoGrid}>
          <div className={styles.infoItem}>
            <span className={styles.infoLabel}>竞拍ID</span>
            <span className={styles.infoValue}>{auction.id}</span>
          </div>
          <div className={styles.infoItem}>
            <span className={styles.infoLabel}>状态</span>
            <span className={`${styles.infoValue} ${isActive ? styles.statusActive : styles.statusEnded}`}>
              {getStatusText(auction.status)}
            </span>
          </div>
          <div className={styles.infoItem}>
            <span className={styles.infoLabel}>开始时间</span>
            <span className={styles.infoValue}>{new Date(auction.start_time).toLocaleString()}</span>
          </div>
          <div className={styles.infoItem}>
            <span className={styles.infoLabel}>结束时间</span>
            <span className={styles.infoValue}>{new Date(auction.end_time).toLocaleString()}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AuctionPage;
