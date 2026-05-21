// pages/Result/index.tsx

import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';

interface AuctionResult {
  auction_id: number;
  product_id: number;
  status: number;
  final_price: number;
  winner_id?: number;
  started_at: string;
  ended_at: string;
  delay_used: number;
}

const ResultPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [result, setResult] = useState<AuctionResult | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchResult();
    }
  }, [id]);

  const fetchResult = async () => {
    try {
      const response = await fetch(`/api/v1/auctions/${id}/result`);
      const data = await response.json();
      setResult(data);
    } catch (error) {
      console.error('获取竞拍结果失败:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div style={{ padding: '20px', textAlign: 'center' }}>加载中...</div>;
  }

  if (!result) {
    return <div style={{ padding: '20px', textAlign: 'center' }}>竞拍结果不存在</div>;
  }

  const isWinner = result.winner_id === 1; // 简化判断

  return (
    <div style={{ padding: '20px', maxWidth: '600px', margin: '0 auto' }}>
      <h1>竞拍结果</h1>

      <div style={{
        padding: '30px',
        backgroundColor: isWinner ? '#f6ffed' : '#fff2f0',
        borderRadius: '8px',
        textAlign: 'center',
        marginBottom: '20px',
      }}>
        <div style={{ fontSize: '48px', marginBottom: '10px' }}>
          {isWinner ? '🎉' : '😢'}
        </div>
        <div style={{
          fontSize: '24px',
          fontWeight: 'bold',
          color: isWinner ? '#52c41a' : '#ff4d4f',
        }}>
          {isWinner ? '恭喜中标！' : '未中标'}
        </div>
      </div>

      <div style={{
        padding: '20px',
        backgroundColor: '#f5f5f5',
        borderRadius: '8px',
      }}>
        <h3>成交信息</h3>
        <p><strong>竞拍ID:</strong> {result.auction_id}</p>
        <p><strong>成交价格:</strong> ¥{result.final_price.toFixed(2)}</p>
        <p><strong>中标者ID:</strong> {result.winner_id || '无'}</p>
        <p><strong>开始时间:</strong> {new Date(result.started_at).toLocaleString()}</p>
        <p><strong>结束时间:</strong> {new Date(result.ended_at).toLocaleString()}</p>
        {result.delay_used > 0 && <p><strong>延时时长:</strong> {result.delay_used} 秒</p>}
      </div>

      {isWinner && (
        <div style={{ marginTop: '20px', textAlign: 'center' }}>
          <button
            style={{
              padding: '15px 40px',
              backgroundColor: '#1890ff',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              fontSize: '16px',
              cursor: 'pointer',
            }}
            onClick={() => alert('支付功能开发中...')}
          >
            立即支付
          </button>
        </div>
      )}
    </div>
  );
};

export default ResultPage;
