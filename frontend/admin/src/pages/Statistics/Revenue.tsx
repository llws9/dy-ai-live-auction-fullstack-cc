import React, { useState, useEffect } from 'react';
import StatCard from '../../components/Charts/StatCard';
import LineChart from '../../components/Charts/LineChart';
import BarChart from '../../components/Charts/BarChart';
import PieChart from '../../components/Charts/PieChart';

interface RevenueStats {
  total_revenue: number;
  today_revenue: number;
  month_revenue: number;
  avg_order_amount: number;
  total_orders: number;
  completed_orders: number;
  pending_orders: number;
  refunded_orders: number;
}

interface DailyRevenueStats {
  date: string;
  revenue: number;
  orders: number;
  avg_amount: number;
}

interface CategoryRevenue {
  category: string;
  revenue: number;
  count: number;
  avg_price: number;
}

interface OrderStatusDistribution {
  name: string;
  value: number;
  color?: string;
}

const RevenueStatistics: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<RevenueStats | null>(null);
  const [dailyStats, setDailyStats] = useState<DailyRevenueStats[]>([]);
  const [categoryStats, setCategoryStats] = useState<CategoryRevenue[]>([]);
  const [orderStatus, setOrderStatus] = useState<OrderStatusDistribution[]>([]);

  useEffect(() => {
    fetchRevenueData();
  }, []);

  const fetchRevenueData = async () => {
    setLoading(true);
    setError(null);

    try {
      // 获取收入统计数据
      const statsRes = await fetch('/api/v1/statistics/revenue');
      if (statsRes.ok) {
        const statsResult = await statsRes.json();
        if (statsResult.code === 0 && statsResult.data) {
          setStats(statsResult.data);
          setDailyStats(statsResult.data.daily_stats || []);
          setCategoryStats(statsResult.data.category_stats || []);
          setOrderStatus(statsResult.data.order_status_distribution || []);
        }
      }
    } catch (err) {
      console.error('Failed to fetch revenue statistics:', err);
      setError('加载数据失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '400px'
      }}>
        <div className="loading-spinner"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">⚠️</div>
        <div className="empty-state-text">{error}</div>
        <button className="btn btn-primary" style={{ marginTop: '16px' }} onClick={fetchRevenueData}>
          重新加载
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">收入统计</h1>
          <p className="page-subtitle">查看收入趋势、订单金额、类目分布等财务数据</p>
        </div>
        <button className="btn btn-secondary" onClick={fetchRevenueData}>
          刷新数据
        </button>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <StatCard
          title="总收入"
          value={`¥${((stats?.total_revenue || 0) / 100).toFixed(2)}`}
          subtitle="累计金额"
          icon="💰"
          iconColor="gold"
          trend={{
            value: 15.2,
            label: '较上月',
          }}
        />
        <StatCard
          title="今日收入"
          value={`¥${((stats?.today_revenue || 0) / 100).toFixed(2)}`}
          subtitle="今日成交"
          icon="📈"
          iconColor="gold"
        />
        <StatCard
          title="本月收入"
          value={`¥${((stats?.month_revenue || 0) / 100).toFixed(2)}`}
          subtitle="本月累计"
          icon="📊"
          iconColor="gold"
        />
        <StatCard
          title="平均订单金额"
          value={`¥${((stats?.avg_order_amount || 0) / 100).toFixed(2)}`}
          subtitle="客单价"
          icon="💎"
          iconColor="blue"
        />
        <StatCard
          title="总订单数"
          value={stats?.total_orders || 0}
          subtitle="累计订单"
          icon="🧾"
          iconColor="blue"
        />
        <StatCard
          title="已完成"
          value={stats?.completed_orders || 0}
          subtitle="成功订单"
          icon="✅"
          iconColor="green"
        />
        <StatCard
          title="待处理"
          value={stats?.pending_orders || 0}
          subtitle="待发货"
          icon="⏳"
          iconColor="gold"
        />
        <StatCard
          title="已退款"
          value={stats?.refunded_orders || 0}
          subtitle="退款订单"
          icon="↩️"
          iconColor="red"
        />
      </div>

      {/* 图表区域 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(500px, 1fr))', gap: '20px', marginBottom: '32px' }}>
        {/* 收入趋势 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">📈 日收入趋势（最近30天）</h3>
          </div>
          <div className="card-body">
            {dailyStats.length > 0 ? (
              <LineChart
                data={dailyStats}
                xAxisKey="date"
                lines={[
                  { dataKey: 'revenue', name: '收入 (元)', color: '#00d4ff' },
                  { dataKey: 'orders', name: '订单数', color: '#10b981' },
                ]}
                height={280}
              />
            ) : (
              <div className="empty-state">
                <div className="empty-state-text">暂无数据</div>
              </div>
            )}
          </div>
        </div>

        {/* 类目收入分布 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">🏷️ 类目收入分布</h3>
          </div>
          <div className="card-body">
            {categoryStats.length > 0 ? (
              <BarChart
                data={categoryStats}
                xAxisKey="category"
                bars={[
                  { dataKey: 'revenue', name: '收入 (元)', color: '#fbbf24' },
                ]}
                height={280}
                showLegend={false}
              />
            ) : (
              <div className="empty-state">
                <div className="empty-state-text">暂无数据</div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 订单状态分布 */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">🧾 订单状态分布</h3>
        </div>
        <div className="card-body">
          {orderStatus.length > 0 ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '24px' }}>
              <PieChart
                data={orderStatus.map(item => ({
                  ...item,
                  color: item.name === '已完成' ? '#10b981' :
                         item.name === '待处理' ? '#fbbf24' :
                         item.name === '已取消' ? '#ef4444' : '#3b82f6'
                }))}
                height={300}
                innerRadius={60}
                outerRadius={100}
              />
              <div style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                gap: '12px',
              }}>
                {orderStatus.map((item, index) => (
                  <div key={index} style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '12px 16px',
                    background: 'var(--bg-tertiary)',
                    borderRadius: 'var(--radius-md)',
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <div style={{
                        width: '12px',
                        height: '12px',
                        borderRadius: '50%',
                        background: item.name === '已完成' ? '#10b981' :
                                   item.name === '待处理' ? '#fbbf24' :
                                   item.name === '已取消' ? '#ef4444' : '#3b82f6'
                      }} />
                      <span style={{ color: 'var(--text-secondary)' }}>{item.name}</span>
                    </div>
                    <span style={{
                      color: 'var(--text-primary)',
                      fontWeight: '600',
                      fontSize: '16px',
                    }}>
                      {item.value.toLocaleString()}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="empty-state">
              <div className="empty-state-text">暂无数据</div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default RevenueStatistics;
