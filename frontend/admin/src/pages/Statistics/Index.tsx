import React from 'react';
import { Link } from 'react-router-dom';

const StatisticsIndex: React.FC = () => {
  const reportTypes = [
    {
      path: '/statistics/auction',
      icon: '🎯',
      title: '竞拍统计',
      description: '查看竞拍场次、参与人数、出价记录等详细统计数据',
      color: '#00d4ff',
    },
    {
      path: '/statistics/revenue',
      icon: '💰',
      title: '收入统计',
      description: '查看收入趋势、类目分布、订单金额等财务数据',
      color: '#fbbf24',
    },
    {
      path: '/statistics/user',
      icon: '👥',
      title: '用户统计',
      description: '查看用户增长、活跃度、参与情况等用户数据',
      color: '#10b981',
    },
  ];

  return (
    <div>
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">数据统计</h1>
          <p className="page-subtitle">查看系统运营数据详细报表</p>
        </div>
      </div>

      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
        gap: '24px',
      }}>
        {reportTypes.map((type) => (
          <Link
            key={type.path}
            to={type.path}
            style={{
              display: 'block',
              textDecoration: 'none',
              background: 'var(--bg-card)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-lg)',
              padding: '32px',
              transition: 'all 0.3s ease',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = type.color;
              e.currentTarget.style.boxShadow = `0 0 30px ${type.color}30`;
              e.currentTarget.style.transform = 'translateY(-4px)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-color)';
              e.currentTarget.style.boxShadow = 'none';
              e.currentTarget.style.transform = 'translateY(0)';
            }}
          >
            <div style={{
              fontSize: '48px',
              marginBottom: '16px',
            }}>
              {type.icon}
            </div>
            <h3 style={{
              fontFamily: 'var(--font-display)',
              fontSize: '20px',
              fontWeight: '600',
              color: 'var(--text-primary)',
              marginBottom: '8px',
            }}>
              {type.title}
            </h3>
            <p style={{
              fontSize: '14px',
              color: 'var(--text-muted)',
              lineHeight: '1.6',
            }}>
              {type.description}
            </p>
            <div style={{
              marginTop: '20px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              color: type.color,
              fontSize: '14px',
              fontWeight: '600',
            }}>
              查看报表
              <span>→</span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
};

export default StatisticsIndex;
