// pages/LiveStream/Detail.tsx

import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';

interface LiveStreamDetail {
  id: number;
  creator_id: number;
  creator_name: string;
  name: string;
  description: string;
  status: number;
  created_at: string;
  stats: {
    total_followers: number;
    new_today: number;
    new_this_week: number;
    new_this_month: number;
    active_last_7_days: number;
    active_last_30_days: number;
    participated_count: number;
  };
  recent_auctions: Array<{
    id: number;
    product_name: string;
    status: number;
    current_price: number;
    bid_count: number;
  }>;
}

const LiveStreamDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [liveStream, setLiveStream] = useState<LiveStreamDetail | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchLiveStreamDetail();
    }
  }, [id]);

  const fetchLiveStreamDetail = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/v1/live-streams/${id}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setLiveStream(data.data);
      } else {
        throw new Error('获取直播间详情失败');
      }
    } catch (error) {
      console.error('获取直播间详情失败:', error);
      // 模拟数据
      setLiveStream({
        id: Number(id),
        creator_id: 5,
        creator_name: '张三',
        name: '张三的直播间',
        description: '主营珠宝首饰，专注于高端奢侈品拍卖',
        status: 1,
        created_at: new Date().toISOString(),
        stats: {
          total_followers: 1250,
          new_today: 15,
          new_this_week: 120,
          new_this_month: 450,
          active_last_7_days: 800,
          active_last_30_days: 1000,
          participated_count: 650,
        },
        recent_auctions: [
          { id: 1, product_name: '稀有珠宝', status: 1, current_price: 150, bid_count: 12 },
          { id: 2, product_name: '签名版限量球鞋', status: 3, current_price: 520, bid_count: 25 },
          { id: 3, product_name: '古董怀表收藏品', status: 3, current_price: 380, bid_count: 18 },
        ],
      });
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    return status === 1
      ? { text: '正常运营', class: 'success' }
      : { text: '已禁用', class: 'error' };
  };

  if (loading) {
    return (
      <div className="empty-state">
        <div className="loading-spinner"></div>
        <p>加载中...</p>
      </div>
    );
  }

  if (!liveStream) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">❌</div>
        <div className="empty-state-text">直播间不存在</div>
      </div>
    );
  }

  const statusConfig = getStatusConfig(liveStream.status);

  return (
    <div>
      {/* 页面标题 */}
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">📺 {liveStream.name}</h1>
          <p className="page-subtitle">直播间详情与统计数据</p>
        </div>
        <Link to="/live-streams">
          <button className="btn btn-secondary">返回列表</button>
        </Link>
      </div>

      {/* 基本信息 */}
      <div className="data-table-wrapper" style={{ marginBottom: '24px' }}>
        <div className="data-table-header">
          <h3 className="data-table-title">基本信息</h3>
        </div>
        <div style={{ padding: '20px' }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '20px' }}>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>直播间ID</div>
              <div style={{ fontWeight: 600, color: 'var(--accent-primary)' }}>#{liveStream.id}</div>
            </div>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>商家</div>
              <div style={{ fontWeight: 500 }}>{liveStream.creator_name}</div>
            </div>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>状态</div>
              <span className={`status-badge ${statusConfig.class}`}>
                {statusConfig.text}
              </span>
            </div>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>创建时间</div>
              <div>{new Date(liveStream.created_at).toLocaleString('zh-CN')}</div>
            </div>
          </div>
          {liveStream.description && (
            <div style={{ marginTop: '16px', paddingTop: '16px', borderTop: '1px solid var(--border-color)' }}>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>简介</div>
              <div style={{ color: 'var(--text-secondary)', lineHeight: 1.6 }}>{liveStream.description}</div>
            </div>
          )}
        </div>
      </div>

      {/* 关注统计 */}
      <div className="data-table-wrapper" style={{ marginBottom: '24px' }}>
        <div className="data-table-header">
          <h3 className="data-table-title">👥 关注统计</h3>
        </div>
        <div style={{ padding: '20px' }}>
          <div className="stats-grid" style={{ marginBottom: '24px' }}>
            <div className="stat-card">
              <div className="stat-card-header">
                <div className="stat-card-icon blue">👥</div>
              </div>
              <div className="stat-card-value">{liveStream.stats.total_followers.toLocaleString()}</div>
              <div className="stat-card-label">总关注人数</div>
            </div>
            <div className="stat-card">
              <div className="stat-card-header">
                <div className="stat-card-icon green">📈</div>
              </div>
              <div className="stat-card-value">{liveStream.stats.new_today}</div>
              <div className="stat-card-label">今日新增</div>
            </div>
            <div className="stat-card">
              <div className="stat-card-header">
                <div className="stat-card-icon gold">📊</div>
              </div>
              <div className="stat-card-value">{liveStream.stats.new_this_week}</div>
              <div className="stat-card-label">本周新增</div>
            </div>
            <div className="stat-card">
              <div className="stat-card-header">
                <div className="stat-card-icon purple">🎯</div>
              </div>
              <div className="stat-card-value">{liveStream.stats.participated_count}</div>
              <div className="stat-card-label">参与竞拍人数</div>
            </div>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
            <div style={{ padding: '16px', background: 'var(--bg-secondary)', borderRadius: '8px' }}>
              <div style={{ color: 'var(--text-muted)', fontSize: '13px', marginBottom: '4px' }}>本月新增</div>
              <div style={{ fontSize: '20px', fontWeight: 600 }}>{liveStream.stats.new_this_month}</div>
            </div>
            <div style={{ padding: '16px', background: 'var(--bg-secondary)', borderRadius: '8px' }}>
              <div style={{ color: 'var(--text-muted)', fontSize: '13px', marginBottom: '4px' }}>近7天活跃</div>
              <div style={{ fontSize: '20px', fontWeight: 600 }}>{liveStream.stats.active_last_7_days}</div>
            </div>
            <div style={{ padding: '16px', background: 'var(--bg-secondary)', borderRadius: '8px' }}>
              <div style={{ color: 'var(--text-muted)', fontSize: '13px', marginBottom: '4px' }}>近30天活跃</div>
              <div style={{ fontSize: '20px', fontWeight: 600 }}>{liveStream.stats.active_last_30_days}</div>
            </div>
          </div>
        </div>
      </div>

      {/* 近期竞拍 */}
      <div className="data-table-wrapper">
        <div className="data-table-header">
          <h3 className="data-table-title">🎯 近期竞拍</h3>
        </div>
        {liveStream.recent_auctions.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>竞拍ID</th>
                <th>商品名称</th>
                <th>状态</th>
                <th>当前价</th>
                <th>出价次数</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {liveStream.recent_auctions.map((auction) => {
                const auctionStatus = auction.status === 1 ? '进行中' : auction.status === 3 ? '已结束' : '待开始';
                const statusClass = auction.status === 1 ? 'success' : auction.status === 3 ? 'default' : 'info';
                return (
                  <tr key={auction.id}>
                    <td style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>
                      #{auction.id}
                    </td>
                    <td style={{ fontWeight: 500 }}>{auction.product_name}</td>
                    <td>
                      <span className={`status-badge ${statusClass}`}>
                        {auctionStatus}
                      </span>
                    </td>
                    <td>
                      <span className="price-display medium">
                        ¥{auction.current_price.toLocaleString()}
                      </span>
                    </td>
                    <td>{auction.bid_count} 次</td>
                    <td>
                      <Link to={`/auctions/${auction.id}`}>
                        <button className="btn btn-secondary btn-sm">查看详情</button>
                      </Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <div className="empty-state">
            <div className="empty-state-icon">📭</div>
            <div className="empty-state-text">暂无竞拍记录</div>
          </div>
        )}
      </div>
    </div>
  );
};

export default LiveStreamDetail;
