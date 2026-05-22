import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';

interface Order {
  id: number;
  auction_id: number;
  product_id: number;
  winner_id: number;
  final_price: number;
  status: number;
  paid_at?: string;
  shipped_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

const OrderDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchOrder();
    }
  }, [id]);

  const fetchOrder = async () => {
    try {
      const response = await fetch(`/api/v1/orders/${id}`);
      const data = await response.json();
      setOrder(data);
    } catch (error) {
      console.error('获取订单详情失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleStatusUpdate = async (newStatus: number) => {
    try {
      const response = await fetch(`/api/v1/orders/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: newStatus }),
      });

      if (response.ok) {
        fetchOrder();
        alert('状态更新成功');
      } else {
        alert('状态更新失败');
      }
    } catch (error) {
      console.error('更新订单状态失败:', error);
      alert('更新失败');
    }
  };

  const getStatusText = (status: number) => {
    const statusMap: { [key: number]: string } = {
      0: '待支付',
      1: '已支付',
      2: '已发货',
      3: '已完成',
    };
    return statusMap[status] || '未知';
  };

  const getStatusColor = (status: number) => {
    const colorMap: { [key: number]: string } = {
      0: '#faad14',
      1: '#52c41a',
      2: '#1890ff',
      3: '#8c8c8c',
    };
    return colorMap[status] || '#8c8c8c';
  };

  if (loading) {
    return <div style={{ padding: '20px' }}>加载中...</div>;
  }

  if (!order) {
    return (
      <div style={{ padding: '20px' }}>
        <h2>订单不存在</h2>
        <button onClick={() => navigate('/orders')}>返回订单列表</button>
      </div>
    );
  }

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ marginBottom: '20px' }}>
        <button onClick={() => navigate('/orders')}>← 返回订单列表</button>
      </div>

      <h1>订单详情 #{order.id}</h1>

      <div style={{
        background: '#fff',
        padding: '24px',
        borderRadius: '8px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
        marginBottom: '20px'
      }}>
        <div style={{ marginBottom: '16px' }}>
          <strong>订单状态：</strong>
          <span style={{
            padding: '4px 12px',
            borderRadius: '4px',
            background: getStatusColor(order.status),
            color: '#fff',
            marginLeft: '8px'
          }}>
            {getStatusText(order.status)}
          </span>
        </div>

        <div style={{ marginBottom: '16px' }}>
          <strong>竞拍ID：</strong> {order.auction_id}
        </div>

        <div style={{ marginBottom: '16px' }}>
          <strong>商品ID：</strong> {order.product_id}
        </div>

        <div style={{ marginBottom: '16px' }}>
          <strong>中标者ID：</strong> {order.winner_id}
        </div>

        <div style={{ marginBottom: '16px' }}>
          <strong>成交价格：</strong>
          <span style={{ color: '#ff4d4f', fontSize: '20px', fontWeight: 'bold' }}>
            ¥{order.final_price.toFixed(2)}
          </span>
        </div>

        <div style={{ marginBottom: '16px' }}>
          <strong>创建时间：</strong> {new Date(order.created_at).toLocaleString()}
        </div>

        {order.paid_at && (
          <div style={{ marginBottom: '16px' }}>
            <strong>支付时间：</strong> {new Date(order.paid_at).toLocaleString()}
          </div>
        )}

        {order.shipped_at && (
          <div style={{ marginBottom: '16px' }}>
            <strong>发货时间：</strong> {new Date(order.shipped_at).toLocaleString()}
          </div>
        )}

        {order.completed_at && (
          <div style={{ marginBottom: '16px' }}>
            <strong>完成时间：</strong> {new Date(order.completed_at).toLocaleString()}
          </div>
        )}
      </div>

      <div style={{
        background: '#fff',
        padding: '24px',
        borderRadius: '8px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.1)'
      }}>
        <h3>订单操作</h3>

        {order.status === 0 && (
          <button
            onClick={() => handleStatusUpdate(1)}
            style={{
              padding: '10px 20px',
              background: '#52c41a',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              marginRight: '10px'
            }}
          >
            标记为已支付
          </button>
        )}

        {order.status === 1 && (
          <button
            onClick={() => handleStatusUpdate(2)}
            style={{
              padding: '10px 20px',
              background: '#1890ff',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              marginRight: '10px'
            }}
          >
            标记为已发货
          </button>
        )}

        {order.status === 2 && (
          <button
            onClick={() => handleStatusUpdate(3)}
            style={{
              padding: '10px 20px',
              background: '#8c8c8c',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              marginRight: '10px'
            }}
          >
            标记为已完成
          </button>
        )}
      </div>
    </div>
  );
};

export default OrderDetail;
