// frontend/h5/src/pages/Home/index.tsx

import React from 'react';
import { Link } from 'react-router-dom';

const HomePage: React.FC = () => {
  // 模拟数据 - 实际应从API获取
  const auctions = [
    { id: 1, product_id: 1, status: 1, current_price: 150, end_time: new Date(Date.now() + 3600000).toISOString() },
  ];

  const getStatusText = (status: number): string => {
    const statusMap: Record<number, string> = {
      0: '即将开始',
      1: '进行中',
      2: '延时中',
      3: '已结束',
      4: '已取消',
    };
    return statusMap[status] || '未知';
  };

  return (
    <div style={{ padding: '20px' }}>
      <h1>热门竞拍</h1>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '15px' }}>
        {auctions.map((auction) => (
          <Link
            key={auction.id}
            to={`/auction/${auction.id}`}
            style={{
              display: 'block',
              padding: '15px',
              backgroundColor: '#fff',
              borderRadius: '8px',
              boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
              textDecoration: 'none',
              color: 'inherit',
            }}
          >
            <div style={{ marginBottom: '10px' }}>
              <span style={{
                padding: '2px 8px',
                borderRadius: '4px',
                backgroundColor: auction.status === 1 ? '#f6ffed' : '#f5f5f5',
                color: auction.status === 1 ? '#52c41a' : '#666',
                fontSize: '12px',
              }}>
                {getStatusText(auction.status)}
              </span>
            </div>

            <div style={{ fontSize: '24px', fontWeight: 'bold', color: '#ff4d4f' }}>
              ¥{auction.current_price.toFixed(2)}
            </div>

            <div style={{ fontSize: '12px', color: '#999', marginTop: '10px' }}>
              竞拍ID: {auction.id}
            </div>
          </Link>
        ))}
      </div>

      {auctions.length === 0 && (
        <div style={{ padding: '40px', textAlign: 'center', color: '#999' }}>
          暂无进行中的竞拍
        </div>
      )}
    </div>
  );
};

export default HomePage;
