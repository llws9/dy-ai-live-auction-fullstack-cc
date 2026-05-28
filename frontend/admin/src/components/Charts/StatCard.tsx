import React from 'react';

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: string;
  iconColor: 'blue' | 'green' | 'gold' | 'red';
  trend?: {
    value: number;
    label: string;
  };
}

const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  iconColor,
  trend,
}) => {
  return (
    <div className="stat-card">
      <div className="stat-card-header">
        <div className={`stat-card-icon ${iconColor}`}>
          {icon}
        </div>
        {trend && (
          <div className={`stat-card-trend ${trend.value >= 0 ? 'up' : 'down'}`}>
            <span>{trend.value >= 0 ? '↑' : '↓'}</span>
            <span>{Math.abs(trend.value)}%</span>
          </div>
        )}
      </div>
      <div className="stat-card-value">{typeof value === 'number' ? value.toLocaleString() : value}</div>
      <div className="stat-card-label">
        {title}
        {subtitle && <span style={{ marginLeft: '4px', opacity: 0.7 }}>({subtitle})</span>}
      </div>
      {trend && (
        <div style={{
          marginTop: '8px',
          fontSize: '12px',
          color: 'var(--text-muted)'
        }}>
          {trend.label}
        </div>
      )}
    </div>
  );
};

export default StatCard;
