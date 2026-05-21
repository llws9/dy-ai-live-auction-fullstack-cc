// pages/History/index.tsx

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

interface Order {
  id: number;
  auction_id: number;
  product_id: number;
  product_name: string;
  product_image: string;
  winner_id: number;
  final_price: number;
  status: number;
  created_at: string;
  paid_at?: string;
}

const HistoryPage: React.FC = () => {
  const navigate = useNavigate();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'all' | 'won' | 'pending'>('all');
  const [paymentModal, setPaymentModal] = useState<Order | null>(null);

  useEffect(() => {
    fetchOrders();
  }, []);

  const fetchOrders = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/orders');
      const data = await response.json();
      setOrders(data.items || []);
    } catch (error) {
      console.error('获取历史记录失败:', error);
      // 模拟数据
      setOrders([
        {
          id: 1001,
          auction_id: 3,
          product_id: 3,
          product_name: '古董怀表收藏品',
          product_image: 'https://images.unsplash.com/photo-1509048191080-d2984bad6ae5?w=400',
          winner_id: 1,
          final_price: 520,
          status: 1,
          created_at: new Date(Date.now() - 3600000).toISOString(),
          paid_at: new Date(Date.now() - 1800000).toISOString(),
        },
        {
          id: 1002,
          auction_id: 5,
          product_id: 5,
          product_name: '艺术画作原稿',
          product_image: 'https://images.unsplash.com/photo-1579783902614-a3fb3927b6a5?w=400',
          winner_id: 1,
          final_price: 1200,
          status: 0,
          created_at: new Date(Date.now() - 7200000).toISOString(),
        },
        {
          id: 1003,
          auction_id: 8,
          product_id: 8,
          product_name: '限量版手表',
          product_image: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=400',
          winner_id: 1,
          final_price: 3500,
          status: 3,
          created_at: new Date(Date.now() - 86400000).toISOString(),
          paid_at: new Date(Date.now() - 43200000).toISOString(),
        },
        {
          id: 1004,
          auction_id: 12,
          product_id: 12,
          product_name: '签名篮球',
          product_image: 'https://images.unsplash.com/photo-1519861531473-9200262188bf?w=400',
          winner_id: 1,
          final_price: 680,
          status: 2,
          created_at: new Date(Date.now() - 172800000).toISOString(),
          paid_at: new Date(Date.now() - 144000000).toISOString(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusConfig = (status: number) => {
    const configs: Record<number, { text: string; tag: string; icon: string }> = {
      0: { text: '待支付', tag: 'pending', icon: '⏳' },
      1: { text: '已支付', tag: 'paid', icon: '✓' },
      2: { text: '已发货', tag: 'paid', icon: '📦' },
      3: { text: '已完成', tag: 'ended', icon: '🏆' },
    };
    return configs[status] || { text: '未知', tag: 'ended', icon: '?' };
  };

  const handlePay = async (order: Order) => {
    try {
      await fetch(`/api/v1/orders/${order.id}/pay`, { method: 'POST' });
      setOrders(orders.map(o =>
        o.id === order.id ? { ...o, status: 1, paid_at: new Date().toISOString() } : o
      ));
      setPaymentModal(null);
    } catch (error) {
      console.error('支付失败:', error);
      // 模拟支付成功
      setOrders(orders.map(o =>
        o.id === order.id ? { ...o, status: 1, paid_at: new Date().toISOString() } : o
      ));
      setPaymentModal(null);
    }
  };

  const filteredOrders = orders.filter((order) => {
    if (activeTab === 'won') return order.status >= 1;
    if (activeTab === 'pending') return order.status === 0;
    return true;
  });

  const stats = {
    total: orders.length,
    pending: orders.filter(o => o.status === 0).length,
    totalSpent: orders.filter(o => o.status >= 1).reduce((sum, o) => sum + o.final_price, 0),
  };

  return (
    <div style={styles.container}>
      {/* 头部 */}
      <div style={styles.header}>
        <button style={styles.backBtn} onClick={() => navigate('/')}>
          ←
        </button>
        <h1 style={styles.title}>竞拍历史</h1>
        <div style={{ width: '40px' }}></div>
      </div>

      {/* 统计卡片 */}
      <div style={styles.statsSection}>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{stats.total}</div>
          <div style={styles.statLabel}>参与竞拍</div>
        </div>
        <div style={styles.statCard}>
          <div style={{ ...styles.statValue, color: 'var(--neon-gold)' }}>
            {stats.pending}
          </div>
          <div style={styles.statLabel}>待支付</div>
        </div>
        <div style={styles.statCard}>
          <div style={{ ...styles.statValue, color: 'var(--neon-green)' }}>
            ¥{stats.totalSpent.toLocaleString()}
          </div>
          <div style={styles.statLabel}>已消费</div>
        </div>
      </div>

      {/* 标签筛选 */}
      <div style={styles.tabs}>
        {(['all', 'pending', 'won'] as const).map((tab) => (
          <button
            key={tab}
            style={{
              ...styles.tab,
              ...(activeTab === tab ? styles.tabActive : {}),
            }}
            onClick={() => setActiveTab(tab)}
          >
            {tab === 'all' ? '全部' : tab === 'pending' ? '待支付' : '已中标'}
          </button>
        ))}
      </div>

      {/* 订单列表 */}
      <div style={styles.content}>
        {loading ? (
          <div style={styles.loading}>
            <div style={styles.spinner}></div>
          </div>
        ) : filteredOrders.length === 0 ? (
          <div style={styles.empty}>
            <div style={styles.emptyIcon}>📭</div>
            <p style={styles.emptyText}>暂无竞拍记录</p>
            <button style={styles.emptyBtn} onClick={() => navigate('/')}>
              去参与竞拍
            </button>
          </div>
        ) : (
          <div style={styles.orderList}>
            {filteredOrders.map((order) => {
              const statusConfig = getStatusConfig(order.status);
              return (
                <div key={order.id} style={styles.orderCard}>
                  {/* 商品图片 */}
                  <div style={styles.orderImageWrapper}>
                    <img
                      src={order.product_image}
                      alt={order.product_name}
                      style={styles.orderImage}
                    />
                    <span className={`status-tag ${statusConfig.tag}`} style={styles.orderBadge}>
                      {statusConfig.icon} {statusConfig.text}
                    </span>
                  </div>

                  {/* 订单信息 */}
                  <div style={styles.orderInfo}>
                    <h3 style={styles.orderTitle}>{order.product_name}</h3>
                    <div style={styles.orderMeta}>
                      <span style={styles.orderId}>订单 #{order.id}</span>
                      <span style={styles.orderTime}>
                        {new Date(order.created_at).toLocaleDateString('zh-CN')}
                      </span>
                    </div>

                    <div style={styles.orderPriceRow}>
                      <div>
                        <span style={styles.priceLabel}>成交价</span>
                        <span style={styles.priceValue}>
                          ¥{order.final_price.toLocaleString()}
                        </span>
                      </div>
                    </div>

                    {/* 操作按钮 */}
                    <div style={styles.orderActions}>
                      {order.status === 0 && (
                        <button
                          style={styles.payBtn}
                          onClick={() => setPaymentModal(order)}
                        >
                          💳 立即支付
                        </button>
                      )}
                      {order.status >= 1 && (
                        <button
                          style={styles.detailBtn}
                          onClick={() => navigate(`/result/${order.auction_id}`)}
                        >
                          查看详情
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* 支付弹窗 */}
      {paymentModal && (
        <div style={styles.modalOverlay}>
          <div style={styles.modal}>
            <div style={styles.modalHeader}>
              <h3 style={styles.modalTitle}>确认支付</h3>
              <button
                style={styles.modalClose}
                onClick={() => setPaymentModal(null)}
              >
                ✕
              </button>
            </div>

            <div style={styles.modalBody}>
              <div style={styles.modalProduct}>
                <img
                  src={paymentModal.product_image}
                  alt={paymentModal.product_name}
                  style={styles.modalImage}
                />
                <div>
                  <div style={styles.modalProductName}>
                    {paymentModal.product_name}
                  </div>
                  <div style={styles.modalOrderId}>
                    订单 #{paymentModal.id}
                  </div>
                </div>
              </div>

              <div style={styles.modalPrice}>
                <span>支付金额</span>
                <span style={styles.modalPriceValue}>
                  ¥{paymentModal.final_price.toLocaleString()}
                </span>
              </div>

              <div style={styles.modalNote}>
                ⚠️ 这是模拟支付，点击确认后将直接完成支付
              </div>
            </div>

            <div style={styles.modalActions}>
              <button
                style={styles.cancelBtn}
                onClick={() => setPaymentModal(null)}
              >
                取消
              </button>
              <button
                style={styles.confirmBtn}
                onClick={() => handlePay(paymentModal)}
              >
                确认支付
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    minHeight: '100vh',
    backgroundColor: 'var(--bg-primary)',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '16px 20px',
    background: 'var(--bg-secondary)',
    borderBottom: '1px solid rgba(255,255,255,0.05)',
    position: 'sticky',
    top: 0,
    zIndex: 100,
  },
  backBtn: {
    width: '40px',
    height: '40px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'var(--bg-tertiary)',
    border: 'none',
    borderRadius: '50%',
    color: 'var(--text-primary)',
    fontSize: '18px',
    cursor: 'pointer',
  },
  title: {
    fontFamily: 'var(--font-display)',
    fontSize: '18px',
    fontWeight: 700,
    margin: 0,
  },
  statsSection: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '12px',
    padding: '20px',
  },
  statCard: {
    background: 'var(--bg-card)',
    borderRadius: 'var(--radius-lg)',
    padding: '16px',
    textAlign: 'center',
  },
  statValue: {
    fontFamily: 'var(--font-display)',
    fontSize: '24px',
    fontWeight: 700,
    color: 'var(--text-primary)',
  },
  statLabel: {
    fontSize: '12px',
    color: 'var(--text-muted)',
    marginTop: '4px',
  },
  tabs: {
    display: 'flex',
    gap: '8px',
    padding: '0 20px 16px',
  },
  tab: {
    flex: 1,
    padding: '12px',
    background: 'var(--bg-card)',
    border: '1px solid rgba(255,255,255,0.05)',
    borderRadius: 'var(--radius-md)',
    color: 'var(--text-secondary)',
    fontSize: '14px',
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  tabActive: {
    background: 'linear-gradient(135deg, rgba(0, 212, 255, 0.2) 0%, rgba(139, 92, 246, 0.2) 100%)',
    border: '1px solid rgba(0, 212, 255, 0.3)',
    color: 'var(--neon-blue)',
    boxShadow: '0 0 20px rgba(0, 212, 255, 0.2)',
  },
  content: {
    padding: '0 20px 100px',
  },
  loading: {
    display: 'flex',
    justifyContent: 'center',
    padding: '60px',
  },
  spinner: {
    width: '40px',
    height: '40px',
    border: '3px solid var(--bg-tertiary)',
    borderTopColor: 'var(--neon-blue)',
    borderRadius: '50%',
    animation: 'spin 1s linear infinite',
  },
  empty: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: '60px 20px',
  },
  emptyIcon: {
    fontSize: '64px',
    marginBottom: '16px',
    opacity: 0.5,
  },
  emptyText: {
    fontSize: '16px',
    color: 'var(--text-muted)',
    marginBottom: '20px',
  },
  emptyBtn: {
    padding: '12px 24px',
    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    color: 'white',
    border: 'none',
    borderRadius: 'var(--radius-lg)',
    fontSize: '14px',
    fontWeight: 600,
    cursor: 'pointer',
  },
  orderList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  orderCard: {
    background: 'var(--bg-card)',
    borderRadius: 'var(--radius-lg)',
    overflow: 'hidden',
    border: '1px solid rgba(255,255,255,0.05)',
  },
  orderImageWrapper: {
    position: 'relative',
    paddingTop: '50%',
    background: 'var(--bg-tertiary)',
  },
  orderImage: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: '100%',
    height: '100%',
    objectFit: 'cover',
  },
  orderBadge: {
    position: 'absolute',
    top: '12px',
    right: '12px',
  },
  orderInfo: {
    padding: '16px',
  },
  orderTitle: {
    fontSize: '16px',
    fontWeight: 600,
    margin: '0 0 8px 0',
    color: 'var(--text-primary)',
  },
  orderMeta: {
    display: 'flex',
    gap: '12px',
    marginBottom: '12px',
  },
  orderId: {
    fontSize: '12px',
    color: 'var(--neon-blue)',
  },
  orderTime: {
    fontSize: '12px',
    color: 'var(--text-muted)',
  },
  orderPriceRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-end',
    marginBottom: '12px',
  },
  priceLabel: {
    fontSize: '12px',
    color: 'var(--text-muted)',
    marginRight: '8px',
  },
  priceValue: {
    fontFamily: 'var(--font-display)',
    fontSize: '24px',
    fontWeight: 700,
    color: 'var(--neon-gold)',
  },
  orderActions: {
    display: 'flex',
    gap: '12px',
  },
  payBtn: {
    flex: 1,
    padding: '12px',
    background: 'linear-gradient(135deg, #ffd700 0%, #ff8c00 100%)',
    color: '#1a1a2e',
    border: 'none',
    borderRadius: 'var(--radius-md)',
    fontSize: '14px',
    fontWeight: 600,
    cursor: 'pointer',
  },
  detailBtn: {
    flex: 1,
    padding: '12px',
    background: 'var(--bg-tertiary)',
    color: 'var(--text-primary)',
    border: '1px solid rgba(255,255,255,0.1)',
    borderRadius: 'var(--radius-md)',
    fontSize: '14px',
    fontWeight: 500,
    cursor: 'pointer',
  },
  // 支付弹窗样式
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'rgba(0, 0, 0, 0.8)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '20px',
    zIndex: 1000,
  },
  modal: {
    background: 'var(--bg-card)',
    borderRadius: 'var(--radius-xl)',
    width: '100%',
    maxWidth: '400px',
    overflow: 'hidden',
  },
  modalHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '20px',
    borderBottom: '1px solid rgba(255,255,255,0.05)',
  },
  modalTitle: {
    fontFamily: 'var(--font-display)',
    fontSize: '18px',
    fontWeight: 700,
    margin: 0,
  },
  modalClose: {
    width: '32px',
    height: '32px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'var(--bg-tertiary)',
    border: 'none',
    borderRadius: '50%',
    color: 'var(--text-secondary)',
    cursor: 'pointer',
  },
  modalBody: {
    padding: '20px',
  },
  modalProduct: {
    display: 'flex',
    gap: '12px',
    marginBottom: '20px',
  },
  modalImage: {
    width: '60px',
    height: '60px',
    borderRadius: 'var(--radius-md)',
    objectFit: 'cover',
  },
  modalProductName: {
    fontSize: '14px',
    fontWeight: 600,
    marginBottom: '4px',
  },
  modalOrderId: {
    fontSize: '12px',
    color: 'var(--text-muted)',
  },
  modalPrice: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '16px',
    background: 'var(--bg-tertiary)',
    borderRadius: 'var(--radius-md)',
    marginBottom: '16px',
  },
  modalPriceValue: {
    fontFamily: 'var(--font-display)',
    fontSize: '24px',
    fontWeight: 700,
    color: 'var(--neon-gold)',
  },
  modalNote: {
    fontSize: '12px',
    color: 'var(--neon-gold)',
    textAlign: 'center',
  },
  modalActions: {
    display: 'flex',
    gap: '12px',
    padding: '20px',
    borderTop: '1px solid rgba(255,255,255,0.05)',
  },
  cancelBtn: {
    flex: 1,
    padding: '14px',
    background: 'var(--bg-tertiary)',
    color: 'var(--text-secondary)',
    border: 'none',
    borderRadius: 'var(--radius-md)',
    fontSize: '14px',
    fontWeight: 600,
    cursor: 'pointer',
  },
  confirmBtn: {
    flex: 1,
    padding: '14px',
    background: 'linear-gradient(135deg, #00ff88 0%, #00d4ff 100%)',
    color: '#1a1a2e',
    border: 'none',
    borderRadius: 'var(--radius-md)',
    fontSize: '14px',
    fontWeight: 600,
    cursor: 'pointer',
  },
};

export default HistoryPage;
