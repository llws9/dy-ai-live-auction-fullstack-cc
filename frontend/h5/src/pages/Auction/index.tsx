// pages/Auction/index.tsx

import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import BidButton from '../../components/BidButton';
import PriceDisplay from '../../components/PriceDisplay';

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

const AuctionPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [auction, setAuction] = useState<Auction | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchAuction();
    }
  }, [id]);

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

  const handleBidSuccess = (newPrice: number) => {
    if (auction) {
      setAuction({
        ...auction,
        current_price: newPrice,
      });
    }
  };

  if (loading) {
    return (
      <div style={{ padding: '20px', textAlign: 'center' }}>
        加载中...
      </div>
    );
  }

  if (!auction) {
    return (
      <div style={{ padding: '20px', textAlign: 'center' }}>
        竞拍不存在
      </div>
    );
  }

  return (
    <div style={{ padding: '20px', maxWidth: '600px', margin: '0 auto' }}>
      {/* 直播画面占位 */}
      <div style={{
        width: '100%',
        height: '300px',
        backgroundColor: '#000',
        borderRadius: '8px',
        marginBottom: '20px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#fff',
      }}>
        直播画面
      </div>

      {/* 价格显示 */}
      <PriceDisplay
        currentPrice={auction.current_price}
        endTime={auction.end_time}
      />

      {/* 出价按钮 */}
      <BidButton
        auctionId={auction.id}
        currentPrice={auction.current_price}
        increment={10}
        onBidSuccess={handleBidSuccess}
      />

      {/* 竞拍信息 */}
      <div style={{
        marginTop: '20px',
        padding: '15px',
        backgroundColor: '#f5f5f5',
        borderRadius: '8px',
      }}>
        <h3>竞拍信息</h3>
        <p>竞拍ID: {auction.id}</p>
        <p>状态: {getStatusText(auction.status)}</p>
        <p>开始时间: {new Date(auction.start_time).toLocaleString()}</p>
        <p>结束时间: {new Date(auction.end_time).toLocaleString()}</p>
        {auction.delay_used > 0 && <p>已延时: {auction.delay_used} 秒</p>}
      </div>
    </div>
  );
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
