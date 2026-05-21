// pages/Order/List.tsx

import React, { useState, useEffect } from 'react';

interface Order {
  id: number;
  auction_id: number;
  product_name: string;
  winner_id: number;
  winner_name: string;
  final_price: number;
  status: number;
  created_at: string;
  paid_at?: string;
}

const OrderList: React.FC = () => {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'pending' | 'paid'>('all');

  useEffect(() => {
    fetchOrders();
  }, [filter]);

  const fetchOrders = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/orders');
      const data = await response.json();
      setOrders(data.items || []);
    } catch (error) {
      console.error('获取订单列表失败:', error);
      // 模拟数据
      setOrders([
        {
          id: 1001,
          auction_id: 3,
          product_name: '古董怀表收藏品',
          winner_id: 1,
          winner_name: '用户A',
          final_price: 520,
          status: 1,
          created_at: new Date(Date.now() - 3600000).toISOString(),
          paid_at: new Date(Date.now() - 1800000).toISOString(),
        },
        {
          id: 1002,
          auction_id: 5,
          product_name: '艺术画作原稿',
          winner_id: 2,
          winner_name: '用户B',
          final_price: 1200,
          status: 0,
          created_at: new Date(Date.now() - 7200000).toISOString(),
        },
        {
          id: 1003,
          auction_id: 8,
          product_name: '限量版手表',
          winner_id: 3,
          winner_name: '用户C',
          final_price: 3500,
          status: 1,
          created_at: new Date(Date.now() - 86400000).toISOString(),
          paid_at: new Date(Date.now() - 43200000).toISOString(),
        },
        {
          id: 1004,
          auction_id: 12,
          product_name: '签名篮球',
          winner_id: 4,
          winner_name: '用户D',
          final_price: 680,
          status: 0,
          created_at: new Date(Date.now() - 172800000).toISOString(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    const configs: Record<number, { text: string; class: string }> = {
      0: { text: '待支付', class: 'warning' },
      1: { text: '已支付', class: 'success' },
      2: { text: '已发货', class: 'info' },
      3: { text: '已完成', class: 'default' },
    };
    return configs[status] || { text: '未知', class: 'default' };
  };

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const filteredOrders = orders.filter((order) => {
    if (filter === 'pending') return order.status === 0;
    if (filter === 'paid') return order.status >= 1;
    return true;
  });

  // 统计数据
  const stats = {
    total: orders.length,
    pending: orders.filter(o => o.status === 0).length,
    paid: orders.filter(o => o.status >= 1).length,
    totalRevenue: orders
      .filter(o => o.status >= 1)
      .reduce((sum, o) => sum + o.final_price, 0),
    pendingAmount: orders
      .filter(o => o.status === 0)
      .reduce((sum, o) => sum + o.final_price, 0),
  };

  const handleUpdateStatus = async (orderId: number, newStatus: number) => {
    try {
      await fetch(`/api/v1/orders/${orderId}/status`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus }),
      });
      fetchOrders();
    } catch (error) {
      console.error('更新状态失败:', error);
    }
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
          <h1 className="page-title">🧾 订单管理</h1>
          <p className="page-subtitle">查看成交订单，管理支付和发货状态</p>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon blue">🧾</div>
          </div>
          <div className="stat-card-value">{stats.total}</div>
          <div className="stat-card-label">订单总数</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon gold">⏳</div>
          </div>
          <div className="stat-card-value">{stats.pending}</div>
          <div className="stat-card-label">待支付</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon green">✓</div>
          </div>
          <div className="stat-card-value">{stats.paid}</div>
          <div className="stat-card-label">已支付</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon gold">💰</div>
          </div>
          <div className="stat-card-value">¥{stats.totalRevenue.toLocaleString()}</div>
          <div className="stat-card-label">已收款</div>
        </div>
      </div>

      {/* 待支付提醒 */}
      {stats.pendingAmount > 0 && (
        <div style={{
          background: 'var(--warning-bg)',
          border: '1px solid rgba(245, 158, 11, 0.3)',
          borderRadius: 'var(--radius-md)',
          padding: '16px 20px',
          marginBottom: '24px',
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
        }}>
          <span style={{ fontSize: '20px' }}>⚠️</span>
          <div>
            <div style={{ fontWeight: 600, color: 'var(--warning)' }}>
              待支付订单提醒
            </div>
            <div style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
              有 {stats.pending} 笔订单待支付，总计 ¥{stats.pendingAmount.toLocaleString()}
            </div>
          </div>
        </div>
      )}

      {/* 订单列表 */}
      <div className="data-table-wrapper">
        <div className="data-table-header">
          <div style={{ display: 'flex', gap: '8px' }}>
            {(['all', 'pending', 'paid'] as const).map((f) => (
              <button
                key={f}
                className={`btn btn-sm ${filter === f ? 'btn-primary' : 'btn-secondary'}`}
                onClick={() => setFilter(f)}
              >
                {f === 'all' ? '全部订单' : f === 'pending' ? '待支付' : '已支付'}
              </button>
            ))}
          </div>
        </div>

        <table className="data-table">
          <thead>
            <tr>
              <th>订单号</th>
              <th>商品名称</th>
              <th>中标者</th>
              <th>成交价</th>
              <th>状态</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {filteredOrders.map((order) => {
              const statusConfig = getStatusConfig(order.status);
              return (
                <tr key={order.id}>
                  <td style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>
                    #{order.id}
                  </td>
                  <td style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                    {order.product_name}
                  </td>
                  <td>
                    <span style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '6px',
                    }}>
                      <span style={{
                        width: '24px',
                        height: '24px',
                        borderRadius: '50%',
                        background: 'linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '12px',
                      }}>
                        {order.winner_name.charAt(0)}
                      </span>
                      {order.winner_name}
                    </span>
                  </td>
                  <td>
                    <span className="price-display medium">
                      ¥{order.final_price.toLocaleString()}
                    </span>
                  </td>
                  <td>
                    <span className={`status-badge ${statusConfig.class}`}>
                      {statusConfig.text}
                    </span>
                  </td>
                  <td style={{ fontSize: '13px' }}>
                    {formatTime(order.created_at)}
                  </td>
                  <td>
                    <div className="action-buttons">
                      {order.status === 0 && (
                        <button
                          className="btn btn-success btn-sm"
                          onClick={() => handleUpdateStatus(order.id, 1)}
                        >
                          确认支付
                        </button>
                      )}
                      {order.status === 1 && (
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => handleUpdateStatus(order.id, 2)}
                        >
                          标记发货
                        </button>
                      )}
                      {order.status === 2 && (
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => handleUpdateStatus(order.id, 3)}
                        >
                          确认完成
                        </button>
                      )}
                      {order.status === 3 && (
                        <span style={{ color: 'var(--text-muted)', fontSize: '13px' }}>
                          订单已完成
                        </span>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {filteredOrders.length === 0 && (
          <div className="empty-state">
            <div className="empty-state-icon">📭</div>
            <div className="empty-state-text">暂无订单数据</div>
          </div>
        )}
      </div>
    </div>
  );
};

export default OrderList;
