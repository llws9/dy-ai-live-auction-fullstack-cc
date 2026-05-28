import React, { useState, useEffect } from 'react';
import StatCard from '../../components/Charts/StatCard';
import LineChart from '../../components/Charts/LineChart';
import BarChart from '../../components/Charts/BarChart';
import PieChart from '../../components/Charts/PieChart';

interface UserStats {
  total_users: number;
  new_users_today: number;
  new_users_month: number;
  active_users: number;
  avg_participation_rate: number;
  avg_bids_per_user: number;
  total_bidders: number;
  total_sellers: number;
}

interface DailyUserStats {
  date: string;
  new_users: number;
  active_users: number;
  total_users: number;
}

interface UserLevelDistribution {
  level: string;
  count: number;
}

interface UserActivityDistribution {
  name: string;
  value: number;
}

const UserStatistics: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<UserStats | null>(null);
  const [dailyStats, setDailyStats] = useState<DailyUserStats[]>([]);
  const [levelDistribution, setLevelDistribution] = useState<UserLevelDistribution[]>([]);
  const [activityDistribution, setActivityDistribution] = useState<UserActivityDistribution[]>([]);

  useEffect(() => {
    fetchUserData();
  }, []);

  const fetchUserData = async () => {
    setLoading(true);
    setError(null);

    try {
      // 获取用户统计数据
      const statsRes = await fetch('/api/v1/statistics/users');
      if (statsRes.ok) {
        const statsResult = await statsRes.json();
        if (statsResult.code === 0 && statsResult.data) {
          setStats(statsResult.data);
          setDailyStats(statsResult.data.daily_stats || []);
          setLevelDistribution(statsResult.data.level_distribution || []);
          setActivityDistribution(statsResult.data.activity_distribution || []);
        }
      }
    } catch (err) {
      console.error('Failed to fetch user statistics:', err);
      setError('加载数据失败,请稍后重试');
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
        <button className="btn btn-primary" style={{ marginTop: '16px' }} onClick={fetchUserData}>
          重新加载
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">用户统计</h1>
          <p className="page-subtitle">查看用户增长、活跃度、参与情况等用户数据</p>
        </div>
        <button className="btn btn-secondary" onClick={fetchUserData}>
          刷新数据
        </button>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <StatCard
          title="总用户数"
          value={stats?.total_users || 0}
          subtitle="注册用户"
          icon="👥"
          iconColor="blue"
          trend={{
            value: 12.5,
            label: '较上月',
          }}
        />
        <StatCard
          title="今日新增"
          value={stats?.new_users_today || 0}
          subtitle="新用户"
          icon="🆕"
          iconColor="green"
        />
        <StatCard
          title="本月新增"
          value={stats?.new_users_month || 0}
          subtitle="本月注册"
          icon="📅"
          iconColor="green"
        />
        <StatCard
          title="活跃用户"
          value={stats?.active_users || 0}
          subtitle="近7日活跃"
          icon="⚡"
          iconColor="blue"
        />
        <StatCard
          title="参与率"
          value={`${(stats?.avg_participation_rate || 0).toFixed(1)}%`}
          subtitle="竞拍参与"
          icon="🎯"
          iconColor="gold"
        />
        <StatCard
          title="平均出价"
          value={stats?.avg_bids_per_user || 0}
          subtitle="每用户"
          icon="💎"
          iconColor="blue"
        />
        <StatCard
          title="买家数"
          value={stats?.total_bidders || 0}
          subtitle="参与出价"
          icon="🛒"
          iconColor="green"
        />
        <StatCard
          title="卖家数"
          value={stats?.total_sellers || 0}
          subtitle="发布商品"
          icon="🏪"
          iconColor="gold"
        />
      </div>

      {/* 图表区域 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(500px, 1fr))', gap: '20px', marginBottom: '32px' }}>
        {/* 用户增长趋势 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">📈 用户增长趋势（最近30天）</h3>
          </div>
          <div className="card-body">
            {dailyStats.length > 0 ? (
              <LineChart
                data={dailyStats}
                xAxisKey="date"
                lines={[
                  { dataKey: 'new_users', name: '新增用户', color: '#10b981' },
                  { dataKey: 'active_users', name: '活跃用户', color: '#00d4ff' },
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

        {/* 用户等级分布 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">⭐ 用户等级分布</h3>
          </div>
          <div className="card-body">
            {levelDistribution.length > 0 ? (
              <BarChart
                data={levelDistribution}
                xAxisKey="level"
                bars={[
                  { dataKey: 'count', name: '用户数量', color: '#fbbf24' },
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

      {/* 用户活跃度分布 */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">🔥 用户活跃度分布</h3>
        </div>
        <div className="card-body">
          {activityDistribution.length > 0 ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '24px' }}>
              <PieChart
                data={activityDistribution.map(item => ({
                  ...item,
                  color: item.name === '高活跃' ? '#10b981' :
                         item.name === '中活跃' ? '#fbbf24' :
                         item.name === '低活跃' ? '#3b82f6' : '#64748b'
                }))}
                height={300}
                outerRadius={100}
              />
              <div style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                gap: '12px',
              }}>
                {activityDistribution.map((item, index) => (
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
                        background: item.name === '高活跃' ? '#10b981' :
                                   item.name === '中活跃' ? '#fbbf24' :
                                   item.name === '低活跃' ? '#3b82f6' : '#64748b'
                      }} />
                      <span style={{ color: 'var(--text-secondary)' }}>{item.name}</span>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{
                        color: 'var(--text-primary)',
                        fontWeight: '600',
                        fontSize: '16px',
                      }}>
                        {item.value.toLocaleString()}
                      </div>
                      <div style={{
                        fontSize: '12px',
                        color: 'var(--text-muted)',
                      }}>
                        {((item.value / (stats?.total_users || 1)) * 100).toFixed(1)}%
                      </div>
                    </div>
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

export default UserStatistics;
