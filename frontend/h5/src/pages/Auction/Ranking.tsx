// frontend/h5/src/pages/Auction/Ranking.tsx

import React from 'react';

interface RankItem {
  rank: number;
  user_id: number;
  user_name?: string;
  amount: number;
}

interface RankingProps {
  ranking: RankItem[];
}

const Ranking: React.FC<RankingProps> = ({ ranking }) => {
  return (
    <div style={{
      padding: '15px',
      backgroundColor: '#fff',
      borderRadius: '8px',
      boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
    }}>
      <h3 style={{ margin: '0 0 15px 0', fontSize: '16px' }}>实时排名</h3>

      {ranking.length === 0 ? (
        <div style={{ textAlign: 'center', color: '#999', padding: '20px' }}>
          暂无出价记录
        </div>
      ) : (
        <div>
          {ranking.map((item, index) => (
            <div
              key={index}
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '10px 0',
                borderBottom: index < ranking.length - 1 ? '1px solid #f0f0f0' : 'none',
              }}
            >
              {/* 排名 */}
              <div style={{
                width: '30px',
                height: '30px',
                borderRadius: '50%',
                backgroundColor: item.rank <= 3 ? '#ff4d4f' : '#f0f0f0',
                color: item.rank <= 3 ? '#fff' : '#666',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 'bold',
                marginRight: '10px',
              }}>
                {item.rank}
              </div>

              {/* 用户信息 */}
              <div style={{ flex: 1 }}>
                <div style={{ fontWeight: 'bold' }}>
                  {item.user_name || `用户${item.user_id}`}
                </div>
              </div>

              {/* 出价金额 */}
              <div style={{
                fontWeight: 'bold',
                color: '#ff4d4f',
              }}>
                ¥{item.amount.toFixed(2)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default Ranking;
