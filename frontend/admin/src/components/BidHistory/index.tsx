// components/BidHistory/index.tsx

import React from 'react';

interface Bid {
  id: number;
  auction_id: number;
  user_id: number;
  user_name?: string;
  amount: number;
  created_at: string;
}

interface BidHistoryProps {
  bids: Bid[];
  loading?: boolean;
}

const BidHistory: React.FC<BidHistoryProps> = ({ bids, loading }) => {
  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  if (loading) {
    return (
      <div className="data-table-wrapper">
        <div className="data-table-header">
          <h3 className="data-table-title">出价记录</h3>
        </div>
        <div className="empty-state">
          <div className="loading-spinner"></div>
          <p style={{ marginTop: '16px' }}>加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="data-table-wrapper">
      <div className="data-table-header">
        <h3 className="data-table-title">出价记录 ({bids.length})</h3>
      </div>

      {bids.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">📭</div>
          <div className="empty-state-text">暂无出价记录</div>
        </div>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
              <th>序号</th>
              <th>用户</th>
              <th>出价金额</th>
              <th>出价时间</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {bids.map((bid, index) => (
              <tr key={bid.id}>
                <td style={{ color: 'var(--text-muted)', fontWeight: 600 }}>
                  #{index + 1}
                </td>
                <td style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                  {bid.user_name || `用户 #${bid.user_id}`}
                </td>
                <td>
                  <span className="price-display medium">
                    ¥{bid.amount.toLocaleString()}
                  </span>
                </td>
                <td style={{ color: 'var(--text-muted)' }}>
                  {formatTime(bid.created_at)}
                </td>
                <td>
                  {index === 0 ? (
                    <span className="status-badge success">
                      🏆 领先
                    </span>
                  ) : (
                    <span className="status-badge default">
                      已出局
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* 价格走势图 */}
      {bids.length > 0 && (
        <div style={{ padding: '24px', borderTop: '1px solid var(--border-color)' }}>
          <h4 style={{ marginBottom: '16px', fontSize: '14px', fontWeight: 600 }}>
            价格走势
          </h4>
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: '8px', height: '120px' }}>
            {bids.slice(0, 20).reverse().map((bid, index) => {
              const maxAmount = Math.max(...bids.map(b => b.amount));
              const height = (bid.amount / maxAmount) * 100;
              return (
                <div
                  key={bid.id}
                  style={{
                    flex: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    gap: '4px',
                  }}
                >
                  <div
                    style={{
                      width: '100%',
                      height: `${height}%`,
                      background: 'linear-gradient(to top, var(--accent-primary), var(--accent-secondary))',
                      borderRadius: '4px 4px 0 0',
                      minHeight: '8px',
                    }}
                  />
                  {index % 5 === 0 && (
                    <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
                      ¥{bid.amount}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};

export default BidHistory;
