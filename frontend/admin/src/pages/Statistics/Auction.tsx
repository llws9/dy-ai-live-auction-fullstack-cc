import React, { useState, useEffect } from 'react';
import StatCard from '../../components/Charts/StatCard';
import LineChart from '../../components/Charts/LineChart';
import BarChart from '../../components/Charts/BarChart';
import PieChart from '../../components/Charts/PieChart';

interface AuctionStats {
  total_auctions: number;
  active_auctions: number;
  completed_auctions: number;
  failed_auctions: number;
  total_bids: number;
  avg_bids_per_auction: number;
  success_rate: number;
  avg_final_price: number;
}

interface DailyAuctionStats {
  date: string;
  auctions: number;
  bids: number;
  success_rate: number;
}

interface HourlyDistribution {
  hour: string;
  count: number;
}

interface StatusDistribution {
  name: string;
  value: number;
}

const AuctionStatistics: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<AuctionStats | null>(null);
  const [dailyStats, setDailyStats] = useState<DailyAuctionStats[]>([]);
  const [hourlyDistribution, setHourlyDistribution] = useState<HourlyDistribution[]>([]);
  const [statusDistribution, setStatusDistribution] = useState<StatusDistribution[]>([]);

  useEffect(() => {
    fetchAuctionData();
  }, []);

  const fetchAuctionData = async () => {
    setLoading(true);
    setError(null);

    try {
      // 获取竞拍统计数据
      const statsRes = await fetch('/api/v1/statistics/auctions');
      if (statsRes.ok) {
        const statsResult = await statsRes.json();
        if (statsResult.code === 0 && statsResult.data) {
          setStats(statsResult.data);
          setDailyStats(statsResult.data.daily_stats || []);
          setHourlyDistribution(statsResult.data.hourly_distribution || []);
          setStatusDistribution(statsResult.data.status_distribution || []);
        }
      }
    } catch (err) {
      console.error('Failed to fetch auction statistics:', err);
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
        <button className="btn btn-primary" style={{ marginTop: '16px' }} onClick={fetchAuctionData}>
          重新加载
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">竞拍统计</h1>
          <p className="page-subtitle">查看竞拍场次、参与情况、成交数据等详细统计</p>
        </div>
        <button className="btn btn-secondary" onClick={fetchAuctionData}>
          刷新数据
        </button>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <StatCard
          title="总竞拍数"
          value={stats?.total_auctions || 0}
          subtitle="累计场次"
          icon="🎯"
          iconColor="blue"
        />
        <StatCard
          title="进行中"
          value={stats?.active_auctions || 0}
          subtitle="当前活跃"
          icon="⚡"
          iconColor="green"
        />
        <StatCard
          title="已完成"
          value={stats?.completed_auctions || 0}
          subtitle="成功成交"
          icon="✅"
          iconColor="green"
        />
        <StatCard
          title="已流拍"
          value={stats?.failed_auctions || 0}
          subtitle="未成交"
          icon="❌"
          iconColor="red"
        />
        <StatCard
          title="总出价数"
          value={stats?.total_bids || 0}
          subtitle="累计出价"
          icon="💎"
          iconColor="blue"
        />
        <StatCard
          title="平均出价"
          value={stats?.avg_bids_per_auction || 0}
          subtitle="每场次"
          icon="📊"
          iconColor="blue"
        />
        <StatCard
          title="成功率"
          value={`${(stats?.success_rate || 0).toFixed(1)}%`}
          subtitle="成交率"
          icon="📈"
          iconColor="green"
        />
        <StatCard
          title="平均成交价"
          value={`¥${((stats?.avg_final_price || 0) / 100).toFixed(2)}`}
          subtitle="成交均价"
          icon="💰"
          iconColor="gold"
        />
      </div>

      {/* 图表区域 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(500px, 1fr))', gap: '20px', marginBottom: '32px' }}>
        {/* 竞拍趋势 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">📈 竞拍趋势（最近30天）</h3>
          </div>
          <div className="card-body">
            {dailyStats.length > 0 ? (
              <LineChart
                data={dailyStats}
                xAxisKey="date"
                lines={[
                  { dataKey: 'auctions', name: '竞拍场次', color: '#00d4ff' },
                  { dataKey: 'bids', name: '出价次数', color: '#fbbf24' },
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

        {/* 时段分布 */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">⏰ 竞拍时段分布</h3>
          </div>
          <div className="card-body">
            {hourlyDistribution.length > 0 ? (
              <BarChart
                data={hourlyDistribution}
                xAxisKey="hour"
                bars={[
                  { dataKey: 'count', name: '场次数量', color: '#00d4ff' },
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

      {/* 状态分布 */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">🎯 竞拍状态分布</h3>
        </div>
        <div className="card-body">
          {statusDistribution.length > 0 ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '24px' }}>
              <PieChart
                data={statusDistribution}
                height={300}
                outerRadius={100}
              />
              <div style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                gap: '12px',
              }}>
                {statusDistribution.map((item, index) => (
                  <div key={index} style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '12px 16px',
                    background: 'var(--bg-tertiary)',
                    borderRadius: 'var(--radius-md)',
                  }}>
                    <span style={{ color: 'var(--text-secondary)' }}>{item.name}</span>
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

export default AuctionStatistics;
