// pages/User/Index.tsx

import React, { useState, useEffect } from 'react';
import UserInfo from '../../components/UserInfo';
import UserStats from '../../components/UserStats';
import { useNavigate } from 'react-router-dom';

interface UserData {
  id: number;
  name: string;
  avatar: string;
  created_at: string;
}

interface UserStatsData {
  participated: number;      // 参与竞拍数
  won: number;              // 中标数
  successRate: number;      // 成功率
  totalSpent: number;       // 总消费金额
  recentAuctions: Array<{   // 最近竞拍记录
    id: number;
    product_name: string;
    final_price: number;
    status: string;
    created_at: string;
  }>;
}

const UserCenter: React.FC = () => {
  const navigate = useNavigate();
  const [userData, setUserData] = useState<UserData | null>(null);
  const [statsData, setStatsData] = useState<UserStatsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchUserData();
    fetchUserStats();
  }, []);

  const fetchUserData = async () => {
    try {
      const token = localStorage.getItem('token');
      if (!token) {
        navigate('/login');
        return;
      }

      const response = await fetch('/api/v1/users/me', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!response.ok) {
        throw new Error('获取用户信息失败');
      }

      const data = await response.json();
      setUserData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取用户信息失败');
    }
  };

  const fetchUserStats = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/v1/users/me/stats', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setStatsData(data);
      }
    } catch (err) {
      console.error('获取用户统计失败:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 20px' }}>
        <div className="loading-spinner"></div>
        <p style={{ color: '#fff', marginTop: '20px' }}>加载中...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 20px' }}>
        <p style={{ color: '#ff4d4f' }}>❌ {error}</p>
        <button
          onClick={() => window.location.reload()}
          style={{
            marginTop: '20px',
            padding: '10px 30px',
            backgroundColor: '#1890ff',
            color: '#fff',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer'
          }}
        >
          重试
        </button>
      </div>
    );
  }

  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: '#0f0f1e',
      color: '#fff',
      padding: '20px'
    }}>
      {/* 头部 */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        marginBottom: '30px',
        padding: '20px',
        backgroundColor: '#1a1a2e',
        borderRadius: '12px'
      }}>
        <button
          onClick={() => navigate(-1)}
          style={{
            background: 'none',
            border: 'none',
            color: '#fff',
            fontSize: '24px',
            cursor: 'pointer',
            marginRight: '15px'
          }}
        >
          ←
        </button>
        <h1 style={{ margin: 0, fontSize: '24px' }}>个人中心</h1>
      </div>

      {/* 用户信息卡片 */}
      {userData && <UserInfo user={userData} />}

      {/* 用户统计 */}
      {statsData && <UserStats stats={statsData} />}

      {/* 最近竞拍记录 */}
      {statsData && statsData.recentAuctions && statsData.recentAuctions.length > 0 && (
        <div style={{
          backgroundColor: '#1a1a2e',
          borderRadius: '12px',
          padding: '20px',
          marginTop: '20px'
        }}>
          <h2 style={{ fontSize: '18px', marginBottom: '20px' }}>最近竞拍</h2>
          {statsData.recentAuctions.map((auction) => (
            <div
              key={auction.id}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '15px',
                backgroundColor: '#252538',
                borderRadius: '8px',
                marginBottom: '10px'
              }}
            >
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '5px' }}>
                  {auction.product_name}
                </div>
                <div style={{ fontSize: '12px', color: '#999' }}>
                  {new Date(auction.created_at).toLocaleDateString()}
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{
                  color: auction.status === 'won' ? '#52c41a' : '#ff4d4f',
                  marginBottom: '5px'
                }}>
                  {auction.status === 'won' ? '中标' : '未中标'}
                </div>
                <div style={{ fontWeight: 'bold' }}>
                  ¥{auction.final_price.toFixed(2)}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 功能菜单 */}
      <div style={{
        backgroundColor: '#1a1a2e',
        borderRadius: '12px',
        padding: '20px',
        marginTop: '20px'
      }}>
        <h2 style={{ fontSize: '18px', marginBottom: '20px' }}>功能菜单</h2>

        <div
          onClick={() => navigate('/history')}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '15px',
            backgroundColor: '#252538',
            borderRadius: '8px',
            marginBottom: '10px',
            cursor: 'pointer'
          }}
        >
          <span>📦 我的订单</span>
          <span>→</span>
        </div>

        <div
          onClick={() => navigate('/notifications')}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '15px',
            backgroundColor: '#252538',
            borderRadius: '8px',
            marginBottom: '10px',
            cursor: 'pointer'
          }}
        >
          <span>🔔 我的消息</span>
          <span>→</span>
        </div>

        <div
          onClick={() => {
            localStorage.removeItem('token');
            navigate('/login');
          }}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '15px',
            backgroundColor: '#252538',
            borderRadius: '8px',
            cursor: 'pointer',
            color: '#ff4d4f'
          }}
        >
          <span>退出登录</span>
          <span>→</span>
        </div>
      </div>
    </div>
  );
};

export default UserCenter;
