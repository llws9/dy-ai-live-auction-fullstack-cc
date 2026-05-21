// pages/History/index.tsx

import React, { useState, useEffect } from 'react';

interface Order {
  id: number;
  auction_id: number;
  product_id: number;
  winner_id: number;
  final_price: number;
  status: number;
  created_at: string;
}

const HistoryPage: React.FC = () => {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);

  useEffect(() => {
    fetchOrders();
  }, [page]);

  const fetchOrders = async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/v1/orders?page=${page}&page_size=10`);
      const data = await response.json();
      setOrders(data.items || []);
      setTotal(data.total || 0);
    } catch (error) {
      console.error('获取历史记录失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusText = (status: number): string => {
    const statusMap: Record<number, string> = {
      0: '待支付',
      1: '已支付',
      2: '已发货',
      3: '已完成',
    };
    return statusMap[status] || '未知状态';
  };

  const getStatusColor = (status: number): string => {
    const colorMap: Record<number, string> = {
      0: '#faad14',
      1: '#1890ff',
      2: '#52c41a',
      3: '#8c8c8c',
    };
    return colorMap[status] || '#666';
  };

  if (loading) {
    return <div style={{ padding: '20px', textAlign: 'center' }}>加载中...</div>;
  }

  return (
    <div style={{ padding: '20px', maxWidth: '600px', margin: '0 auto' }}>
      <h1>竞拍历史</h1>

      {orders.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: '#999' }}>
          暂无竞拍记录
        </div>
      ) : (
        <>
          {orders.map((order) => (
            <div
              key={order.id}
              style={{
                padding: '15px',
                backgroundColor: '#fff',
                borderRadius: '8px',
                marginBottom: '10px',
                boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
                <span style={{ fontWeight: 'bold' }}>订单 #{order.id}</span>
                <span style={{
                  padding: '2px 8px',
                  borderRadius: '4px',
                  backgroundColor: `${getStatusColor(order.status)}20`,
                  color: getStatusColor(order.status),
                }}>
                  {getStatusText(order.status)}
                </span>
              </div>

              <div style={{ fontSize: '14px', color: '#666' }}>
                <p style={{ margin: '5px 0' }}>竞拍ID: {order.auction_id}</p>
                <p style={{ margin: '5px 0' }}>成交价: ¥{order.final_price.toFixed(2)}</p>
                <p style={{ margin: '5px 0' }}>
                  创建时间: {new Date(order.created_at).toLocaleString()}
                </p>
              </div>

              {order.status === 0 && (
                <button
                  style={{
                    marginTop: '10px',
                    padding: '8px 20px',
                    backgroundColor: '#1890ff',
                    color: 'white',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                  }}
                  onClick={async () => {
                    try {
                      await fetch(`/api/v1/orders/${order.id}/pay`, { method: 'POST' });
                      fetchOrders();
                    } catch (error) {
                      console.error('支付失败:', error);
                    }
                  }}
                >
                  立即支付
                </button>
              )}
            </div>
          ))}

          <div style={{ marginTop: '20px', textAlign: 'center' }}>
            <button
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
              style={{ padding: '8px 16px', marginRight: '10px' }}
            >
              上一页
            </button>
            <span>第 {page} 页</span>
            <button
              disabled={page * 10 >= total}
              onClick={() => setPage(page + 1)}
              style={{ padding: '8px 16px', marginLeft: '10px' }}
            >
              下一页
            </button>
          </div>
        </>
      )}
    </div>
  );
};

export default HistoryPage;
