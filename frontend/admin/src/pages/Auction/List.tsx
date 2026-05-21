// pages/Auction/List.tsx

import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

interface Auction {
  id: number;
  product_id: number;
  product_name: string;
  status: number;
  current_price: number;
  start_price: number;
  cap_price: number;
  increment: number;
  winner_id?: number;
  winner_name?: string;
  start_time: string;
  end_time: string;
  delay_used: number;
  bid_count: number;
}

const AuctionList: React.FC = () => {
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'ongoing' | 'ended'>('all');

  useEffect(() => {
    fetchAuctions();
  }, [filter]);

  const fetchAuctions = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/auctions');
      const data = await response.json();
      setAuctions(data.auctions || []);
    } catch (error) {
      console.error('获取竞拍列表失败:', error);
      // 模拟数据
      setAuctions([
        {
          id: 1,
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
          bid_count: 12,
        },
        {
          id: 2,
          product_id: 2,
          product_name: '签名版限量球鞋',
          status: 1,
          current_price: 280,
          start_price: 100,
          cap_price: 500,
          increment: 20,
          start_time: new Date(Date.now() - 1800000).toISOString(),
          end_time: new Date(Date.now() + 1800000).toISOString(),
          delay_used: 30,
          bid_count: 8,
        },
        {
          id: 3,
          product_id: 3,
          product_name: '古董怀表收藏品',
          status: 3,
          current_price: 520,
          start_price: 200,
          cap_price: 800,
          increment: 10,
          start_time: new Date(Date.now() - 7200000).toISOString(),
          end_time: new Date(Date.now() - 3600000).toISOString(),
          delay_used: 60,
          bid_count: 25,
          winner_id: 1,
          winner_name: '用户A',
        },
        {
          id: 4,
          product_id: 4,
          product_name: '限定款奢侈品包包',
          status: 0,
          current_price: 0,
          start_price: 500,
          cap_price: 2000,
          increment: 50,
          start_time: new Date(Date.now() + 86400000).toISOString(),
          end_time: new Date(Date.now() + 90000000).toISOString(),
          delay_used: 0,
          bid_count: 0,
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    const configs: Record<number, { text: string; class: string }> = {
      0: { text: '待开始', class: 'info' },
      1: { text: '进行中', class: 'success' },
      2: { text: '延时中', class: 'warning' },
      3: { text: '已结束', class: 'default' },
      4: { text: '已取消', class: 'error' },
    };
    return configs[status] || { text: '未知', class: 'default' };
  };

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getRemainingTime = (endTime: string) => {
    const diff = new Date(endTime).getTime() - Date.now();
    if (diff <= 0) return '已结束';
    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);
    if (hours > 0) return `${hours}小时${minutes}分钟`;
    return `${minutes}分钟`;
  };

  const filteredAuctions = auctions.filter((auction) => {
    if (filter === 'ongoing') return auction.status === 1 || auction.status === 2;
    if (filter === 'ended') return auction.status === 3 || auction.status === 4;
    return true;
  });

  // 统计数据
  const stats = {
    total: auctions.length,
    ongoing: auctions.filter(a => a.status === 1 || a.status === 2).length,
    ended: auctions.filter(a => a.status === 3).length,
    totalRevenue: auctions
      .filter(a => a.status === 3)
      .reduce((sum, a) => sum + a.current_price, 0),
  };

  if (loading) {
    return (
      <div className="empty-state">
        <div className="loading-spinner"></div>
        <p style={{ marginTop: '16px' }}>加载中...</p>
      </div>
    );
  }

  return (
    <div>
      {/* 页面标题 */}
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">🎯 竞拍管理</h1>
          <p className="page-subtitle">实时监控所有竞拍状态，管理竞拍流程</p>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon blue">🎯</div>
          </div>
          <div className="stat-card-value">{stats.total}</div>
          <div className="stat-card-label">竞拍总数</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon green">⚡</div>
          </div>
          <div className="stat-card-value">{stats.ongoing}</div>
          <div className="stat-card-label">进行中</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon gold">🏆</div>
          </div>
          <div className="stat-card-value">{stats.ended}</div>
          <div className="stat-card-label">已成交</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon gold">💰</div>
          </div>
          <div className="stat-card-value">¥{stats.totalRevenue.toLocaleString()}</div>
          <div className="stat-card-label">总成交额</div>
        </div>
      </div>

      {/* 筛选标签 */}
      <div className="data-table-wrapper">
        <div className="data-table-header">
          <div style={{ display: 'flex', gap: '8px' }}>
            {(['all', 'ongoing', 'ended'] as const).map((f) => (
              <button
                key={f}
                className={`btn btn-sm ${filter === f ? 'btn-primary' : 'btn-secondary'}`}
                onClick={() => setFilter(f)}
              >
                {f === 'all' ? '全部' : f === 'ongoing' ? '进行中' : '已结束'}
              </button>
            ))}
          </div>
        </div>

        <table className="data-table">
          <thead>
            <tr>
              <th>竞拍ID</th>
              <th>商品名称</th>
              <th>当前价</th>
              <th>出价次数</th>
              <th>状态</th>
              <th>剩余时间</th>
              <th>中标者</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {filteredAuctions.map((auction) => {
              const statusConfig = getStatusConfig(auction.status);
              return (
                <tr key={auction.id}>
                  <td style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>
                    #{auction.id}
                  </td>
                  <td style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                    {auction.product_name}
                  </td>
                  <td>
                    <span className="price-display medium">
                      ¥{auction.current_price.toLocaleString()}
                    </span>
                  </td>
                  <td>
                    <span style={{
                      padding: '4px 10px',
                      background: 'var(--bg-tertiary)',
                      borderRadius: '12px',
                      fontSize: '13px',
                    }}>
                      {auction.bid_count} 次
                    </span>
                  </td>
                  <td>
                    <span className={`status-badge ${statusConfig.class}`}>
                      {statusConfig.text}
                    </span>
                  </td>
                  <td>
                    {auction.status === 1 || auction.status === 2
                      ? getRemainingTime(auction.end_time)
                      : auction.status === 0
                      ? formatTime(auction.start_time) + ' 开始'
                      : '-'}
                  </td>
                  <td>
                    {auction.winner_name ? (
                      <span style={{ color: 'var(--gold)' }}>{auction.winner_name}</span>
                    ) : (
                      <span style={{ color: 'var(--text-muted)' }}>-</span>
                    )}
                  </td>
                  <td>
                    <div className="action-buttons">
                      <Link to={`/auctions/${auction.id}`}>
                        <button className="btn btn-secondary btn-sm">查看详情</button>
                      </Link>
                      {(auction.status === 0 || auction.status === 1) && (
                        <button className="btn btn-danger btn-sm">取消竞拍</button>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {filteredAuctions.length === 0 && (
          <div className="empty-state">
            <div className="empty-state-icon">📭</div>
            <div className="empty-state-text">暂无竞拍数据</div>
          </div>
        )}
      </div>
    </div>
  );
};

export default AuctionList;
