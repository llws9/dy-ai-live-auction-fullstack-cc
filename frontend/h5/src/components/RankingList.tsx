import React from 'react';
import { useAuth } from '../store/authContext';

interface BidRank {
  rank: number;
  user_id: number;
  amount: number;
  bid_time: string;
}

interface RankingListProps {
  rankings: BidRank[];
  currentUserId?: number;
  loading?: boolean;
}

const RankingList: React.FC<RankingListProps> = ({
  rankings,
  currentUserId,
  loading = false,
}) => {
  const { user } = useAuth();
  const userId = currentUserId || user?.id;

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (seconds < 60) {
      return '刚刚';
    } else if (minutes < 60) {
      return `${minutes}分钟前`;
    } else if (hours < 24) {
      return `${hours}小时前`;
    } else {
      return date.toLocaleDateString();
    }
  };

  if (loading) {
    return (
      <div style={{ padding: '16px', textAlign: 'center', color: '#999' }}>
        加载中...
      </div>
    );
  }

  if (!rankings || rankings.length === 0) {
    return (
      <div style={{ padding: '16px', textAlign: 'center', color: '#999' }}>
        暂无出价记录
      </div>
    );
  }

  return (
    <div style={{ backgroundColor: '#fff', borderRadius: '8px', overflow: 'hidden' }}>
      <div
        style={{
          padding: '12px 16px',
          backgroundColor: '#fafafa',
          borderBottom: '1px solid #f0f0f0',
          fontSize: '14px',
          fontWeight: 'bold',
        }}
      >
        出价排名
      </div>

      <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
        {rankings.map((bid) => {
          const isCurrentUser = userId && bid.user_id === userId;

          return (
            <div
              key={`${bid.user_id}-${bid.bid_time}`}
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '12px 16px',
                borderBottom: '1px solid #f0f0f0',
                backgroundColor: isCurrentUser ? '#fff7e6' : '#fff',
              }}
            >
              {/* 排名 */}
              <div
                style={{
                  width: '32px',
                  height: '32px',
                  borderRadius: '50%',
                  backgroundColor:
                    bid.rank === 1
                      ? '#ffd700'
                      : bid.rank === 2
                      ? '#c0c0c0'
                      : bid.rank === 3
                      ? '#cd7f32'
                      : '#f0f0f0',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '14px',
                  fontWeight: 'bold',
                  color: bid.rank <= 3 ? '#fff' : '#666',
                  marginRight: '12px',
                }}
              >
                {bid.rank}
              </div>

              {/* 用户信息 */}
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '14px', fontWeight: '500', marginBottom: '2px' }}>
                  {isCurrentUser ? '我的出价' : `用户${bid.user_id}`}
                </div>
                <div style={{ fontSize: '12px', color: '#999' }}>
                  {formatTime(bid.bid_time)}
                </div>
              </div>

              {/* 出价金额 */}
              <div style={{ textAlign: 'right' }}>
                <div
                  style={{
                    fontSize: '16px',
                    fontWeight: 'bold',
                    color: isCurrentUser ? '#ff4d4f' : '#333',
                  }}
                >
                  ¥{bid.amount.toFixed(2)}
                </div>
                {bid.rank === 1 && (
                  <div
                    style={{
                      fontSize: '10px',
                      color: '#ff4d4f',
                      marginTop: '2px',
                    }}
                  >
                    领先
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {userId && rankings.findIndex((bid) => bid.user_id === userId) > 0 && (
        <div
          style={{
            padding: '12px 16px',
            backgroundColor: '#fafafa',
            borderTop: '1px solid #f0f0f0',
            textAlign: 'center',
            fontSize: '12px',
            color: '#666',
          }}
        >
          您的排名：第{rankings.findIndex((bid) => bid.user_id === userId) + 1}名
        </div>
      )}
    </div>
  );
};

export default RankingList;
