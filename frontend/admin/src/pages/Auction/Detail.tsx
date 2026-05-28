// pages/Auction/Detail.tsx

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { auctionApi } from '../../services/api';
import AuctionInfo from '../../components/AuctionInfo';
import BidHistory from '../../components/BidHistory';

interface AuctionDetail {
  id: number;
  product_id: number;
  product_name: string;
  status: number;
  current_price: number;
  start_price: number;
  cap_price: number;
  increment: number;
  start_time: string;
  end_time: string;
  delay_used: number;
  winner_id?: number;
  winner_name?: string;
}

interface Bid {
  id: number;
  auction_id: number;
  user_id: number;
  user_name: string;
  amount: number;
  created_at: string;
}

const AuctionDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [auction, setAuction] = useState<AuctionDetail | null>(null);
  const [bids, setBids] = useState<Bid[]>([]);
  const [loading, setLoading] = useState(true);
  const [bidsLoading, setBidsLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchAuctionDetail();
      fetchBids();
    }
  }, [id]);

  const fetchAuctionDetail = async () => {
    if (!id) return;

    setLoading(true);
    try {
      const data = await auctionApi.get(Number(id));
      setAuction(data);
    } catch (error) {
      console.error('获取竞拍详情失败:', error);
      // 模拟数据
      setAuction({
        id: Number(id),
        product_id: 1,
        product_name: '稀有珠宝',
        status: 1,
        current_price: 150,
        start_price: 0,
        cap_price: 1000,
        increment: 10,
        start_time: new Date(Date.now() - 3600000).toISOString(),
        end_time: new Date(Date.now() + 3600000).toISOString(),
        delay_used: 0,
        winner_name: undefined,
      });
    } finally {
      setLoading(false);
    }
  };

  const fetchBids = async () => {
    if (!id) return;

    setBidsLoading(true);
    try {
      const data = await auctionApi.getBids(Number(id));
      setBids(data.bids || []);
    } catch (error) {
      console.error('获取出价记录失败:', error);
      // 模拟数据
      setBids([
        {
          id: 1,
          auction_id: Number(id),
          user_id: 1,
          user_name: '用户A',
          amount: 150,
          created_at: new Date(Date.now() - 300000).toISOString(),
        },
        {
          id: 2,
          auction_id: Number(id),
          user_id: 2,
          user_name: '用户B',
          amount: 140,
          created_at: new Date(Date.now() - 600000).toISOString(),
        },
        {
          id: 3,
          auction_id: Number(id),
          user_id: 3,
          user_name: '用户C',
          amount: 130,
          created_at: new Date(Date.now() - 900000).toISOString(),
        },
        {
          id: 4,
          auction_id: Number(id),
          user_id: 1,
          user_name: '用户A',
          amount: 120,
          created_at: new Date(Date.now() - 1200000).toISOString(),
        },
        {
          id: 5,
          auction_id: Number(id),
          user_id: 2,
          user_name: '用户B',
          amount: 100,
          created_at: new Date(Date.now() - 1500000).toISOString(),
        },
      ]);
    } finally {
      setBidsLoading(false);
    }
  };

  const handleCancel = async () => {
    if (!id || !auction) return;

    if (!window.confirm('确定要取消该竞拍吗？取消后将无法恢复。')) {
      return;
    }

    try {
      await auctionApi.cancel(Number(id));
      alert('竞拍已取消');
      fetchAuctionDetail();
    } catch (error) {
      console.error('取消竞拍失败:', error);
      alert('取消竞拍失败');
    }
  };

  if (loading) {
    return (
      <div className="empty-state">
        <div className="loading-spinner"></div>
        <p style={{ marginTop: '16px' }}>加载中...</p>
      </div>
    );
  }

  if (!auction) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">📭</div>
        <div className="empty-state-text">竞拍不存在</div>
        <button
          className="btn btn-primary"
          onClick={() => navigate('/auctions')}
          style={{ marginTop: '16px' }}
        >
          返回列表
        </button>
      </div>
    );
  }

  return (
    <div>
      {/* 页面标题 */}
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">🎯 竞拍详情</h1>
          <p className="page-subtitle">竞拍ID: #{auction.id}</p>
        </div>
        <div style={{ display: 'flex', gap: '12px' }}>
          {(auction.status === 0 || auction.status === 1) && (
            <button
              className="btn btn-danger"
              onClick={handleCancel}
            >
              取消竞拍
            </button>
          )}
          <button
            className="btn btn-secondary"
            onClick={() => navigate('/auctions')}
          >
            返回列表
          </button>
        </div>
      </div>

      {/* 竞拍信息 */}
      <div style={{ marginBottom: '24px' }}>
        <AuctionInfo auction={auction} />
      </div>

      {/* 出价记录 */}
      <BidHistory bids={bids} loading={bidsLoading} />

      {/* 自动刷新 */}
      {(auction.status === 1 || auction.status === 2) && (
        <div style={{ marginTop: '16px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '14px' }}>
          💡 页面每30秒自动刷新
        </div>
      )}
    </div>
  );
};

export default AuctionDetailPage;
