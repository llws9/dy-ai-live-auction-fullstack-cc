// components/AuctionInfo/index.tsx

import React from 'react';

interface AuctionInfoProps {
  auction: {
    id: number;
    product_id: number;
    product_name?: string;
    status: number;
    current_price: number;
    start_price: number;
    cap_price?: number;
    increment: number;
    start_time: string;
    end_time: string;
    delay_used: number;
    winner_id?: number;
    winner_name?: string;
  };
}

const AuctionInfo: React.FC<AuctionInfoProps> = ({ auction }) => {
  const getStatusConfig = (status: number) => {
    const configs: Record<number, { text: string; class: string; icon: string }> = {
      0: { text: '待开始', class: 'info', icon: '⏰' },
      1: { text: '进行中', class: 'success', icon: '⚡' },
      2: { text: '延时中', class: 'warning', icon: '🔥' },
      3: { text: '已结束', class: 'default', icon: '✓' },
      4: { text: '已取消', class: 'error', icon: '✕' },
    };
    return configs[status] || { text: '未知', class: 'default', icon: '?' };
  };

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  const getRemainingTime = () => {
    if (auction.status !== 1 && auction.status !== 2) return null;

    const diff = new Date(auction.end_time).getTime() - Date.now();
    if (diff <= 0) return '已结束';

    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);
    const seconds = Math.floor((diff % 60000) / 1000);

    if (hours > 0) return `${hours}小时${minutes}分钟`;
    if (minutes > 0) return `${minutes}分${seconds}秒`;
    return `${seconds}秒`;
  };

  const statusConfig = getStatusConfig(auction.status);
  const remainingTime = getRemainingTime();

  return (
    <div className="data-table-wrapper">
      <div className="data-table-header">
        <h3 className="data-table-title">竞拍基本信息</h3>
      </div>

      <div style={{ padding: '24px' }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '24px' }}>
          {/* 竞拍ID */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              竞拍ID
            </div>
            <div style={{ fontSize: '18px', fontWeight: 600, color: 'var(--accent-primary)' }}>
              #{auction.id}
            </div>
          </div>

          {/* 商品名称 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              商品名称
            </div>
            <div style={{ fontSize: '18px', fontWeight: 600 }}>
              {auction.product_name || `商品 #${auction.product_id}`}
            </div>
          </div>

          {/* 状态 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              状态
            </div>
            <span className={`status-badge ${statusConfig.class}`}>
              {statusConfig.icon} {statusConfig.text}
            </span>
          </div>

          {/* 当前价格 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              当前价格
            </div>
            <div className="price-display large">
              ¥{auction.current_price.toLocaleString()}
            </div>
          </div>

          {/* 起拍价 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              起拍价
            </div>
            <div style={{ fontSize: '16px', fontWeight: 500 }}>
              ¥{auction.start_price.toLocaleString()}
            </div>
          </div>

          {/* 加价幅度 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              加价幅度
            </div>
            <div style={{ fontSize: '16px', fontWeight: 500 }}>
              ¥{auction.increment.toLocaleString()}
            </div>
          </div>

          {/* 封顶价 */}
          {auction.cap_price && (
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
                封顶价
              </div>
              <div style={{ fontSize: '16px', fontWeight: 500, color: 'var(--gold)' }}>
                ¥{auction.cap_price.toLocaleString()}
              </div>
            </div>
          )}

          {/* 剩余时间 */}
          {remainingTime && (
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
                剩余时间
              </div>
              <div style={{ fontSize: '16px', fontWeight: 600, color: 'var(--error)' }}>
                {remainingTime}
              </div>
            </div>
          )}

          {/* 延时次数 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              延时次数
            </div>
            <div style={{ fontSize: '16px', fontWeight: 500 }}>
              {auction.delay_used} 次
            </div>
          </div>

          {/* 开始时间 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              开始时间
            </div>
            <div style={{ fontSize: '14px' }}>
              {formatTime(auction.start_time)}
            </div>
          </div>

          {/* 结束时间 */}
          <div>
            <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
              结束时间
            </div>
            <div style={{ fontSize: '14px' }}>
              {formatTime(auction.end_time)}
            </div>
          </div>

          {/* 中标者 */}
          {auction.winner_name && (
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>
                中标者
              </div>
              <div style={{ fontSize: '16px', fontWeight: 600, color: 'var(--gold)' }}>
                🏆 {auction.winner_name}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AuctionInfo;
