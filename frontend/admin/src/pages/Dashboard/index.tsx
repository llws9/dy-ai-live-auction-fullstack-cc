import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import StatCard from '../../components/Charts/StatCard';
import LineChart from '../../components/Charts/LineChart';
import BarChart from '../../components/Charts/BarChart';

interface OverviewData {
  totalAuctions: number;
  activeAuctions: number;
  totalRevenue: number;
  todayRevenue: number;
  totalUsers: number;
  newUsersToday: number;
  successRate: number;
  avgBidPrice: number;
}

interface RevenueTrend {
  date: string;
  revenue: number;
  orders: number;
}

interface CategoryRevenue {
  category: string;
  revenue: number;
  count: number;
}

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [overviewData, setOverviewData] = useState<OverviewData | null>(null);
  const [revenueTrend, setRevenueTrend] = useState<RevenueTrend[]>([]);
  const [categoryRevenue, setCategoryRevenue] = useState<CategoryRevenue[]>([]);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    setLoading(true);
    setError(null);

    try {
      // 获取总览数据
      const overviewRes = await fetch('/api/v1/statistics/overview');
      if (overviewRes.ok) {
        const overviewResult = await overviewRes.json();
        if (overviewResult.code === 0 && overviewResult.data) {
          setOverviewData(overviewResult.data);
        }
      }

      // 获取收入趋势（最近7天）
      const today = new Date();
      const sevenDaysAgo = new Date(today.getTime() - 6 * 24 * 60 * 60 * 1000);
      const revenueRes = await fetch(
        `/api/v1/statistics/revenue?start_date=${sevenDaysAgo.toISOString().split('T')[0]}&end_date=${today.toISOString().split('T')[0]}&group_by=day`
      );
      if (revenueRes.ok) {
        const revenueResult = await revenueRes.json();
        if (revenueResult.code === 0 && revenueResult.data?.daily_stats) {
          setRevenueTrend(revenueResult.data.daily_stats);
        }
      }

      // 获取类目收入分布
      const categoryRes = await fetch('/api/v1/statistics/revenue?group_by=category');
      if (categoryRes.ok) {
        const categoryResult = await categoryRes.json();
        if (categoryResult.code === 0 && categoryResult.data?.category_stats) {
          setCategoryRevenue(categoryResult.data.category_stats);
        }
      }
    } catch (err) {
      console.error('Failed to fetch dashboard data:', err);
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
        <button className="btn btn-primary" style={{ marginTop: '16px' }} onClick={fetchDashboardData}>
          重新加载
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">数据大屏</h1>
          <p className="page-subtitle">实时查看系统运营数据概览</p>
        </div>
        <Link to="/statistics" className="btn btn-secondary">
          查看详细报表 →
        </Link>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <StatCard
          title="总竞拍数"
          value={overviewData?.totalAuctions || 0}
          subtitle="累计场次"
          icon="🎯"
          iconColor="blue"
          trend={{
            value: 12.5,
            label: '较上周',
          }}
        />
        <StatCard
          title="进行中竞拍"
          value={overviewData?.activeAuctions || 0}
          subtitle="当前活跃"
          icon="⚡"
          iconColor="green"
        />
        <StatCard
          title="总收入"
          value={`¥${((overviewData?.totalRevenue || 0) / 100).toFixed(2)}`}
          subtitle="累计金额"
          icon="💰"
          iconColor="gold"
          trend={{
            value: 18.3,
            label: '较上月',
          }}
        />
        <StatCard
          title="今日收入"
          value={`¥${((overviewData?.todayRevenue || 0) / 100).toFixed(2)}`}
          subtitle="今日成交"
          icon="📈"
          iconColor="gold"
        />
        <StatCard
          title="总用户数"
          value={overviewData?.totalUsers || 0}
          subtitle="注册用户"
          icon="👥"
          iconColor="blue"
          trend={{
            value: 8.7,
            label: '较上周',
          }}
        />
        <StatCard
          title="今日新增"
          value={overviewData?.newUsersToday || 0}
          subtitle="新用户"
          icon="🆕"
          iconColor="green"
        />
        <StatCard
          title="竞拍成功率"
          value={`${(overviewData?.successRate || 0).toFixed(1)}%`}
          subtitle="成交率"
          icon="✅"
          iconColor="green"
        />
        <StatCard
          title="平均出价"
          value={`¥${((overviewData?.avgBidPrice || 0) / 100).toFixed(2)}`}
          subtitle="平均金额"
          icon="💎"
          iconColor="blue"
        />
      </div>

      {/* 图表区域 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(500px, 1fr))', gap: '20px', marginBottom: '32px' }}>
        {/* 收入趋势 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">📈 收入趋势（最近7天）</h3>
          </div>
          <div className="card-body">
            {revenueTrend.length > 0 ? (
              <LineChart
                data={revenueTrend}
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
            {categoryRevenue.length > 0 ? (
              <BarChart
                data={categoryRevenue}
                xAxisKey="category"
                bars={[
                  { dataKey: 'revenue', name: '收入 (元)', color: '#00d4ff' },
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
      </div>

      {/* 快捷入口 */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">📊 详细报表</h3>
        </div>
        <div className="card-body">
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
            <Link
              to="/statistics/auction"
              className="btn btn-secondary"
              style={{ textDecoration: 'none', justifyContent: 'flex-start' }}
            >
              🎯 竞拍统计
            </Link>
            <Link
              to="/statistics/revenue"
              className="btn btn-secondary"
              style={{ textDecoration: 'none', justifyContent: 'flex-start' }}
            >
              💰 收入统计
            </Link>
            <Link
              to="/statistics/user"
              className="btn btn-secondary"
              style={{ textDecoration: 'none', justifyContent: 'flex-start' }}
            >
              👥 用户统计
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
