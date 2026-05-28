import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/authContext';
import { followApi } from '../../services/api';
import LazyImage from '../../components/LazyImage';

interface LiveStream {
  id: number;
  name: string;
  description: string;
  creator_id: number;
  creator_name: string;
  status: string;
  current_auctions_count: number;
  followers_count: number;
  created_at: string;
  cover_image?: string;
}

const FollowPage: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();
  const [liveStreams, setLiveStreams] = useState<LiveStream[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  // 加载关注的直播间列表
  const loadLiveStreams = async (pageNum: number, reset: boolean = false) => {
    if (!isAuthenticated) {
      navigate('/login');
      return;
    }

    try {
      if (reset) {
        setLoading(true);
      } else {
        setLoadingMore(true);
      }

      const response = await followApi.getFollowedLiveStreams(pageNum, 20);

      if (response && response.data) {
        const newList = response.data.items || [];

        if (reset) {
          setLiveStreams(newList);
        } else {
          setLiveStreams([...liveStreams, ...newList]);
        }

        // 检查是否还有更多数据
        setHasMore(newList.length === 20);
      }
    } catch (error) {
      console.error('加载关注列表失败:', error);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  // 初始加载
  useEffect(() => {
    loadLiveStreams(1, true);
  }, []);

  // 加载更多
  const handleLoadMore = () => {
    if (!loadingMore && hasMore) {
      const nextPage = page + 1;
      setPage(nextPage);
      loadLiveStreams(nextPage);
    }
  };

  // 搜索过滤
  const filteredLiveStreams = liveStreams.filter(stream =>
    stream.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // 取消关注
  const handleUnfollow = async (liveStreamId: number) => {
    try {
      await followApi.unfollowLiveStream(liveStreamId);
      // 从列表中移除
      setLiveStreams(liveStreams.filter(s => s.id !== liveStreamId));
    } catch (error) {
      console.error('取消关注失败:', error);
    }
  };

  // 进入直播间
  const handleEnterLiveStream = (liveStreamId: number) => {
    navigate(`/live/${liveStreamId}`);
  };

  if (loading) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>加载中...</div>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      {/* 顶部标题栏 */}
      <div style={styles.header}>
        <h1 style={styles.title}>我的关注</h1>
        <div style={styles.count}>{liveStreams.length} 个直播间</div>
      </div>

      {/* 搜索栏 */}
      <div style={styles.searchBar}>
        <input
          type="text"
          placeholder="搜索直播间名称"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={styles.searchInput}
        />
      </div>

      {/* 直播间列表 */}
      {filteredLiveStreams.length === 0 ? (
        <div style={styles.emptyState}>
          <div style={styles.emptyIcon}>📺</div>
          <p style={styles.emptyText}>
            {searchQuery ? '未找到匹配的直播间' : '暂无关注的直播间'}
          </p>
        </div>
      ) : (
        <div style={styles.list}>
          {filteredLiveStreams.map((stream) => (
            <div key={stream.id} style={styles.card}>
              {/* 封面图 */}
              {stream.cover_image && (
                <LazyImage
                  src={stream.cover_image}
                  alt={stream.name}
                  style={styles.coverImage}
                />
              )}

              {/* 直播间信息 */}
              <div style={styles.cardContent}>
                <div style={styles.cardHeader}>
                  <h3 style={styles.streamName}>{stream.name}</h3>
                  <span
                    style={{
                      ...styles.statusBadge,
                      backgroundColor:
                        stream.status === 'active'
                          ? '#52c41a'
                          : stream.status === 'scheduled'
                          ? '#1890ff'
                          : '#8c8c8c',
                    }}
                  >
                    {stream.status === 'active'
                      ? '直播中'
                      : stream.status === 'scheduled'
                      ? '未开始'
                      : '已结束'}
                  </span>
                </div>

                <p style={styles.description}>{stream.description}</p>

                <div style={styles.metaInfo}>
                  <span style={styles.metaItem}>
                    🏷️ {stream.creator_name}
                  </span>
                  <span style={styles.metaItem}>
                    🔥 {stream.current_auctions_count} 个竞拍
                  </span>
                  <span style={styles.metaItem}>
                    👥 {stream.followers_count} 人关注
                  </span>
                </div>

                {/* 操作按钮 */}
                <div style={styles.actions}>
                  <button
                    onClick={() => handleEnterLiveStream(stream.id)}
                    style={styles.enterButton}
                  >
                    进入直播间
                  </button>
                  <button
                    onClick={() => handleUnfollow(stream.id)}
                    style={styles.unfollowButton}
                  >
                    取消关注
                  </button>
                </div>
              </div>
            </div>
          ))}

          {/* 加载更多按钮 */}
          {hasMore && (
            <button
              onClick={handleLoadMore}
              disabled={loadingMore}
              style={styles.loadMoreButton}
            >
              {loadingMore ? '加载中...' : '加载更多'}
            </button>
          )}
        </div>
      )}
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
    paddingBottom: '20px',
  },
  header: {
    backgroundColor: '#fff',
    padding: '16px 20px',
    borderBottom: '1px solid #f0f0f0',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  title: {
    fontSize: '20px',
    fontWeight: 'bold',
    margin: 0,
  },
  count: {
    fontSize: '14px',
    color: '#666',
  },
  searchBar: {
    padding: '12px 20px',
    backgroundColor: '#fff',
  },
  searchInput: {
    width: '100%',
    padding: '10px 16px',
    border: '1px solid #e0e0e0',
    borderRadius: '20px',
    fontSize: '14px',
    outline: 'none',
  },
  loading: {
    textAlign: 'center',
    padding: '40px 20px',
    color: '#999',
  },
  emptyState: {
    textAlign: 'center',
    padding: '80px 20px',
    backgroundColor: '#fff',
    marginTop: '12px',
  },
  emptyIcon: {
    fontSize: '64px',
    marginBottom: '16px',
  },
  emptyText: {
    fontSize: '16px',
    color: '#999',
  },
  list: {
    padding: '12px 20px',
  },
  card: {
    backgroundColor: '#fff',
    borderRadius: '12px',
    overflow: 'hidden',
    marginBottom: '12px',
    boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
  },
  coverImage: {
    width: '100%',
    height: '180px',
    objectFit: 'cover',
  },
  cardContent: {
    padding: '16px',
  },
  cardHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: '8px',
  },
  streamName: {
    fontSize: '18px',
    fontWeight: 'bold',
    margin: 0,
    flex: 1,
  },
  statusBadge: {
    padding: '4px 12px',
    borderRadius: '12px',
    fontSize: '12px',
    color: '#fff',
    fontWeight: 'bold',
  },
  description: {
    fontSize: '14px',
    color: '#666',
    marginBottom: '12px',
    lineHeight: '1.6',
  },
  metaInfo: {
    display: 'flex',
    gap: '16px',
    marginBottom: '16px',
    fontSize: '13px',
    color: '#8c8c8c',
  },
  metaItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
  },
  actions: {
    display: 'flex',
    gap: '12px',
  },
  enterButton: {
    flex: 1,
    padding: '10px 16px',
    backgroundColor: '#ff4d4f',
    color: '#fff',
    border: 'none',
    borderRadius: '8px',
    fontSize: '14px',
    fontWeight: 'bold',
    cursor: 'pointer',
  },
  unfollowButton: {
    padding: '10px 16px',
    backgroundColor: '#fff',
    color: '#666',
    border: '1px solid #d9d9d9',
    borderRadius: '8px',
    fontSize: '14px',
    cursor: 'pointer',
  },
  loadMoreButton: {
    width: '100%',
    padding: '12px',
    backgroundColor: '#fff',
    border: '1px solid #e0e0e0',
    borderRadius: '8px',
    fontSize: '14px',
    color: '#666',
    cursor: 'pointer',
    marginTop: '12px',
  },
};

export default FollowPage;
