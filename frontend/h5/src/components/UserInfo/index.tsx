// components/UserInfo/index.tsx

import React from 'react';

interface User {
  id: number;
  name: string;
  avatar: string;
  created_at: string;
}

interface UserInfoProps {
  user: User;
}

const UserInfo: React.FC<UserInfoProps> = ({ user }) => {
  return (
    <div style={{
      backgroundColor: '#1a1a2e',
      borderRadius: '12px',
      padding: '30px 20px',
      marginBottom: '20px'
    }}>
      <div style={{
        display: 'flex',
        alignItems: 'center'
      }}>
        {/* 头像 */}
        <div style={{
          width: '80px',
          height: '80px',
          borderRadius: '50%',
          backgroundColor: '#1890ff',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '32px',
          marginRight: '20px',
          overflow: 'hidden'
        }}>
          {user.avatar ? (
            <img
              src={user.avatar}
              alt={user.name}
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'cover'
              }}
            />
          ) : (
            <span style={{ color: '#fff' }}>
              {user.name.charAt(0).toUpperCase()}
            </span>
          )}
        </div>

        {/* 用户信息 */}
        <div style={{ flex: 1 }}>
          <h2 style={{
            margin: '0 0 10px 0',
            fontSize: '24px',
            fontWeight: 'bold'
          }}>
            {user.name}
          </h2>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            color: '#999',
            fontSize: '14px'
          }}>
            <span style={{ marginRight: '10px' }}>ID: {user.id}</span>
            <span>
              注册时间: {new Date(user.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>

        {/* 编辑按钮 */}
        <button
          style={{
            padding: '8px 16px',
            backgroundColor: 'transparent',
            border: '1px solid #1890ff',
            borderRadius: '4px',
            color: '#1890ff',
            cursor: 'pointer'
          }}
          onClick={() => {
            alert('编辑功能开发中');
          }}
        >
          编辑资料
        </button>
      </div>

      {/* 会员等级 */}
      <div style={{
        marginTop: '20px',
        padding: '15px',
        backgroundColor: '#252538',
        borderRadius: '8px'
      }}>
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <div>
            <div style={{ fontSize: '14px', color: '#999' }}>会员等级</div>
            <div style={{ fontSize: '20px', fontWeight: 'bold', marginTop: '5px' }}>
              普通会员
            </div>
          </div>
          <div style={{
            padding: '8px 16px',
            backgroundColor: '#faad14',
            borderRadius: '20px',
            color: '#fff',
            fontSize: '14px',
            fontWeight: 'bold'
          }}>
            升级VIP
          </div>
        </div>
      </div>
    </div>
  );
};

export default UserInfo;
