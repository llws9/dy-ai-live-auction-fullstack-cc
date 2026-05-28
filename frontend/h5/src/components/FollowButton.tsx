import React, { useState } from 'react';
import { useAuth } from '../store/authContext';
import { followApi } from '../services/api';

interface FollowButtonProps {
  liveStreamId: number;
  initialFollowed?: boolean;
  initialCount?: number;
  onFollowSuccess?: (followed: boolean) => void;
  onFollowError?: (error: Error) => void;
  size?: 'small' | 'medium' | 'large';
}

const FollowButton: React.FC<FollowButtonProps> = ({
  liveStreamId,
  initialFollowed = false,
  initialCount = 0,
  onFollowSuccess,
  onFollowError,
  size = 'medium',
}) => {
  const { isAuthenticated } = useAuth();
  const [followed, setFollowed] = useState(initialFollowed);
  const [count, setCount] = useState(initialCount);
  const [loading, setLoading] = useState(false);

  const handleFollow = async () => {
    // 检查登录状态
    if (!isAuthenticated) {
      alert('请先登录');
      window.location.href = '/login';
      return;
    }

    // 乐观更新
    const previousFollowed = followed;
    const previousCount = count;
    setFollowed(!followed);
    setCount(followed ? count - 1 : count + 1);
    setLoading(true);

    try {
      if (previousFollowed) {
        // 取消关注
        await followApi.unfollowLiveStream(liveStreamId);
      } else {
        // 关注
        await followApi.followLiveStream(liveStreamId);
      }

      // 成功回调
      if (onFollowSuccess) {
        onFollowSuccess(!previousFollowed);
      }
    } catch (err: any) {
      // 失败回滚
      setFollowed(previousFollowed);
      setCount(previousCount);

      const errorMsg = err.message || '操作失败，请重试';
      alert(errorMsg);

      if (onFollowError) {
        onFollowError(err);
      }
    } finally {
      setLoading(false);
    }
  };

  const getSizeStyles = () => {
    switch (size) {
      case 'small':
        return {
          padding: '6px 12px',
          fontSize: '12px',
          iconSize: '14px',
        };
      case 'large':
        return {
          padding: '12px 24px',
          fontSize: '18px',
          iconSize: '22px',
        };
      default:
        return {
          padding: '8px 16px',
          fontSize: '14px',
          iconSize: '18px',
        };
    }
  };

  const sizeStyles = getSizeStyles();

  return (
    <button
      onClick={handleFollow}
      disabled={loading}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '6px',
        padding: sizeStyles.padding,
        fontSize: sizeStyles.fontSize,
        fontWeight: 'bold',
        color: followed ? '#666' : '#fff',
        backgroundColor: followed ? '#f0f0f0' : '#ff4d4f',
        border: followed ? '1px solid #d9d9d9' : 'none',
        borderRadius: '4px',
        cursor: loading ? 'not-allowed' : 'pointer',
        transition: 'all 0.3s',
        opacity: loading ? 0.6 : 1,
      }}
    >
      {/* 关注图标 */}
      <svg
        width={sizeStyles.iconSize}
        height={sizeStyles.iconSize}
        viewBox="0 0 24 24"
        fill={followed ? 'currentColor' : 'none'}
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        {followed ? (
          <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z" />
        ) : (
          <>
            <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z" />
            <line x1="12" y1="7" x2="12" y2="13" />
            <line x1="9" y1="10" x2="15" y2="10" />
          </>
        )}
      </svg>

      <span>
        {loading ? '处理中...' : followed ? '已关注' : '关注'}
      </span>

      {/* 关注数量 */}
      {count > 0 && (
        <span style={{ marginLeft: '4px', fontSize: '0.9em' }}>
          {count}
        </span>
      )}
    </button>
  );
};

export default FollowButton;
