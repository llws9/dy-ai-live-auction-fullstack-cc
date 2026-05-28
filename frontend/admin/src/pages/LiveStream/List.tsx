// pages/LiveStream/List.tsx

import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

interface LiveStream {
  id: number;
  creator_id: number;
  creator_name: string;
  name: string;
  description: string;
  status: number; // 0=disabled, 1=active
  followers_count: number;
  active_auctions: number;
  created_at: string;
}

const LiveStreamList: React.FC = () => {
  const [liveStreams, setLiveStreams] = useState<LiveStream[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    fetchLiveStreams();
  }, [page, search]);

  const fetchLiveStreams = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/admin/live-streams', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setLiveStreams(data.data.items || []);
        setTotal(data.data.total || 0);
      } else {
        throw new Error('获取直播间列表失败');
      }
    } catch (error) {
      console.error('获取直播间列表失败:', error);
      // 模拟数据
      setLiveStreams([
        {
          id: 10,
          creator_id: 5,
          creator_name: '张三',
          name: '张三的直播间',
          description: '主营珠宝首饰',
          status: 1,
          followers_count: 1250,
          active_auctions: 3,
          created_at: new Date().toISOString(),
        },
        {
          id: 11,
          creator_id: 6,
          creator_name: '李四',
          name: '李四的直播间',
          description: '奢侈品专场',
          status: 1,
          followers_count: 890,
          active_auctions: 2,
          created_at: new Date(Date.now() - 86400000).toISOString(),
        },
        {
          id: 12,
          creator_id: 7,
          creator_name: '王五',
          name: '王五的直播间',
          description: '古董收藏品',
          status: 0,
          followers_count: 450,
          active_auctions: 0,
          created_at: new Date(Date.now() - 172800000).toISOString(),
        },
      ]);
      setTotal(3);
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    return status === 1
      ? { text: '正常', class: 'success' }
      : { text: '已禁用', class: 'error' };
  };

  const filteredStreams = liveStreams.filter(stream =>
    stream.name.includes(search) || stream.creator_name.includes(search)
  );

  const totalPages = Math.ceil(total / 10);

  if (loading) {
    return (
      <div className="empty-state">
        <div className="loading-spinner"></div>
        <p>加载中...</p>
      </div>
    );
  }

  return (
    <div>
      {/* 页面标题 */}
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">📺 直播间管理</h1>
          <p className="page-subtitle">管理所有商家直播间，查看统计数据</p>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon blue">📺</div>
          </div>
          <div className="stat-card-value">{total}</div>
          <div className="stat-card-label">直播间总数</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon green">✓</div>
          </div>
          <div className="stat-card-value">
            {liveStreams.filter(s => s.status === 1).length}
          </div>
          <div className="stat-card-label">正常运营</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon gold">👥</div>
          </div>
          <div className="stat-card-value">
            {liveStreams.reduce((sum, s) => sum + s.followers_count, 0).toLocaleString()}
          </div>
          <div className="stat-card-label">总关注人数</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon purple">🎯</div>
          </div>
          <div className="stat-card-value">
            {liveStreams.reduce((sum, s) => sum + s.active_auctions, 0)}
          </div>
          <div className="stat-card-label">进行中竞拍</div>
        </div>
      </div>

      {/* 数据表格 */}
      <div className="data-table-wrapper">
        <div className="data-table-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h3 className="data-table-title">直播间列表</h3>
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <input
              type="text"
              placeholder="搜索直播间..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{
                padding: '8px 12px',
                border: '1px solid var(--border-color)',
                borderRadius: '6px',
                fontSize: '14px',
                width: '200px',
              }}
            />
            {search && (
              <button
                className="btn btn-secondary btn-sm"
                onClick={() => setSearch('')}
              >
                清除
              </button>
            )}
          </div>
        </div>

        <table className="data-table">
          <thead>
            <tr>
              <th>直播间ID</th>
              <th>直播间名称</th>
              <th>商家</th>
              <th>状态</th>
              <th>关注人数</th>
              <th>进行中竞拍</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {filteredStreams.map((stream) => {
              const statusConfig = getStatusConfig(stream.status);
              return (
                <tr key={stream.id}>
                  <td style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>
                    #{stream.id}
                  </td>
                  <td style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                    {stream.name}
                  </td>
                  <td>{stream.creator_name}</td>
                  <td>
                    <span className={`status-badge ${statusConfig.class}`}>
                      {statusConfig.text}
                    </span>
                  </td>
                  <td>
                    <span style={{
                      padding: '4px 10px',
                      background: 'var(--bg-tertiary)',
                      borderRadius: '12px',
                      fontSize: '13px',
                    }}>
                      {stream.followers_count.toLocaleString()} 人
                    </span>
                  </td>
                  <td>
                    {stream.active_auctions > 0 ? (
                      <span style={{ color: 'var(--success)' }}>{stream.active_auctions} 个</span>
                    ) : (
                      <span style={{ color: 'var(--text-muted)' }}>0 个</span>
                    )}
                  </td>
                  <td>{new Date(stream.created_at).toLocaleString('zh-CN')}</td>
                  <td>
                    <div className="action-buttons">
                      <Link to={`/live-streams/${stream.id}`}>
                        <button className="btn btn-secondary btn-sm">查看详情</button>
                      </Link>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {filteredStreams.length === 0 && (
          <div className="empty-state">
            <div className="empty-state-icon">📭</div>
            <div className="empty-state-text">暂无直播间数据</div>
          </div>
        )}

        {/* 分页 */}
        {totalPages > 1 && (
          <div className="pagination">
            <button
              className="pagination-btn"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            >
              ← 上一页
            </button>
            <span className="pagination-info">
              第 {page} 页 / 共 {totalPages} 页
            </span>
            <button
              className="pagination-btn"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              下一页 →
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default LiveStreamList;
