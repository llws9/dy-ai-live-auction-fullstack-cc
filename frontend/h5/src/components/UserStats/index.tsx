// components/UserStats/index.tsx

import React from 'react';

interface UserStats {
  participated: number;      // 参与竞拍数
  won: number;              // 中标数
  successRate: number;      // 成功率
  totalSpent: number;       // 总消费金额
}

interface UserStatsProps {
  stats: UserStats;
}

const UserStats: React.FC<UserStatsProps> = ({ stats }) => {
  const statsItems = [
    {
      label: '参与竞拍',
      value: stats.participated,
      unit: '次',
      color: '#1890ff'
    },
    {
      label: '中标次数',
      value: stats.won,
      unit: '次',
      color: '#52c41a'
    },
    {
      label: '成功率',
      value: `${stats.successRate.toFixed(1)}%`,
      unit: '',
      color: '#faad14'
    },
    {
      label: '总消费',
      value: `¥${stats.totalSpent.toFixed(2)}`,
      unit: '',
      color: '#ff4d4f'
    }
  ];

  return (
    <div style={{
      backgroundColor: '#1a1a2e',
      borderRadius: '12px',
      padding: '20px'
    }}>
      <h2 style={{ fontSize: '18px', marginBottom: '20px' }}>竞拍统计</h2>

      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(2, 1fr)',
        gap: '15px'
      }}>
        {statsItems.map((item, index) => (
          <div
            key={index}
            style={{
              backgroundColor: '#252538',
              borderRadius: '8px',
              padding: '20px 15px',
              textAlign: 'center'
            }}
          >
            <div style={{
              fontSize: '14px',
              color: '#999',
              marginBottom: '10px'
            }}>
              {item.label}
            </div>
            <div style={{
              fontSize: '28px',
              fontWeight: 'bold',
              color: item.color
            }}>
              {item.value}
              {item.unit && (
                <span style={{ fontSize: '14px', marginLeft: '5px' }}>
                  {item.unit}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* 成功率进度条 */}
      <div style={{
        marginTop: '20px',
        padding: '15px',
        backgroundColor: '#252538',
        borderRadius: '8px'
      }}>
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '10px'
        }}>
          <span style={{ fontSize: '14px', color: '#999' }}>成功率</span>
          <span style={{ fontSize: '16px', fontWeight: 'bold', color: '#faad14' }}>
            {stats.successRate.toFixed(1)}%
          </span>
        </div>
        <div style={{
          width: '100%',
          height: '8px',
          backgroundColor: '#3a3a4a',
          borderRadius: '4px',
          overflow: 'hidden'
        }}>
          <div style={{
            width: `${Math.min(stats.successRate, 100)}%`,
            height: '100%',
            backgroundColor: '#faad14',
            borderRadius: '4px',
            transition: 'width 0.3s ease'
          }}></div>
        </div>
      </div>
    </div>
  );
};

export default UserStats;
