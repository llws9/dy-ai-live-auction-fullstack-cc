import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { followApi } from '../../services/api';
import PageHeader from '@/components/shared/PageHeader';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
import styles from './Following.module.css';

interface LiveStream {
  id?: number | string;
  live_stream_id?: number | string;
  name?: string;
  title?: string;
  live_stream_name?: string;
  description?: string;
  creator_name?: string;
  host_name?: string;
  host_avatar?: string;
  avatar?: string;
  status?: string | number;
  current_auctions_count?: number | string;
  auction_count?: number | string;
  followers_count?: number | string;
  viewer_count?: number | string;
  cover_image?: string;
  image?: string;
}

function extractList<T>(response: any): T[] {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  return [];
}

function toNumber(value: number | string | undefined, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function isLive(status: LiveStream['status']) {
  const normalized = String(status ?? '').toLowerCase();
  return status === 1 || ['active', 'live', 'living', 'streaming'].includes(normalized);
}

function hasActiveAuction(stream: LiveStream) {
  const count = stream.current_auctions_count ?? stream.auction_count;
  return count === undefined ? true : toNumber(count) > 0;
}

function getStreamId(stream: LiveStream) {
  return stream.id ?? stream.live_stream_id;
}

function getTitle(stream: LiveStream) {
  const title = repairUtf8Mojibake(stream.title || stream.name || stream.live_stream_name);
  if (title) return title;

  const streamId = getStreamId(stream);
  return streamId === undefined || streamId === null ? '直播间' : `直播间 #${streamId}`;
}

function getHostName(stream: LiveStream) {
  return repairUtf8Mojibake(stream.host_name || stream.creator_name) || '主播';
}

function getCoverImage(stream: LiveStream) {
  return stream.cover_image || stream.image || '';
}

function getAvatar(stream: LiveStream) {
  return stream.host_avatar || stream.avatar || '';
}

const pageSize = 20;

const FollowPage: React.FC = () => {
  const navigate = useNavigate();
  const [liveStreams, setLiveStreams] = useState<LiveStream[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | number | null>(null);
  const [pendingUnfollowIds, setPendingUnfollowIds] = useState<Array<string | number>>([]);

  useEffect(() => {
    let alive = true;

    async function loadFollowedLiveStreams() {
      setLoading(true);
      setError(null);

      try {
        const response = await followApi.getFollowedLiveStreams(1, pageSize);
        if (!alive) return;
        setLiveStreams(extractList<LiveStream>(response));
      } catch (err) {
        if (!alive) return;
        console.error('加载收藏列表失败:', err);
        setLiveStreams([]);
        setError('收藏列表暂时无法加载');
      } finally {
        if (alive) setLoading(false);
      }
    }

    loadFollowedLiveStreams();

    return () => {
      alive = false;
    };
  }, []);

  const liveCount = useMemo(
    () => liveStreams.filter((stream) => isLive(stream.status) && hasActiveAuction(stream)).length,
    [liveStreams]
  );

  const handleUnfollow = async (liveStreamId: number | string) => {
    setPendingUnfollowIds((ids) => [...ids, liveStreamId]);
    try {
      await followApi.unfollowLiveStream(Number(liveStreamId));
      setLiveStreams((streams) => streams.filter((stream) => getStreamId(stream) !== liveStreamId));
      if (expandedId === liveStreamId) setExpandedId(null);
    } catch (error) {
      console.error('取消收藏失败:', error);
      setError('取消收藏失败，请稍后重试');
    } finally {
      setPendingUnfollowIds((ids) => ids.filter((id) => id !== liveStreamId));
    }
  };

  const handleEnterLiveStream = (liveStreamId: number | string) => {
    navigate(`/live?id=${liveStreamId}`);
  };

  return (
    <div className={styles.page}>
      <PageHeader
        classes={{
          header: styles.header,
          backButton: styles.backButton,
          eyebrow: styles.eyebrow,
        }}
        back={{ onClick: () => navigate(-1) }}
        eyebrow="FAVORITES"
        title="我的收藏"
        actions={<span className={styles.countBadge}>{liveStreams.length} 个收藏</span>}
      />

      <section className={styles.summaryGrid} aria-label="收藏概览">
        <div className={styles.summaryCard}>
          <strong>{liveStreams.length}</strong>
          <span>已收藏</span>
        </div>
        <div className={styles.summaryCard}>
          <strong>{liveCount}</strong>
          <span>直播中</span>
        </div>
        <div className={styles.summaryCard}>
          <strong>
            {liveStreams.reduce(
              (sum, stream) => sum + toNumber(stream.current_auctions_count ?? stream.auction_count),
              0
            )}
          </strong>
          <span>竞拍场</span>
        </div>
      </section>

      {error && <div className={styles.errorBanner}>{error}</div>}

      <main className={styles.content}>
        {loading ? (
          <div className={styles.loading}>加载收藏列表...</div>
        ) : liveStreams.length === 0 ? (
          <div className={styles.emptyState}>
            <div className={styles.emptyMark}>♡</div>
            <p>暂无收藏的直播间</p>
            <span>浏览直播时点击收藏按钮即可添加</span>
          </div>
        ) : (
          <div className={styles.streamList}>
            {liveStreams.map((stream) => {
              const streamId = getStreamId(stream);
              const title = getTitle(stream);
              const hostName = getHostName(stream);
              const active = isLive(stream.status) && hasActiveAuction(stream);
              const expanded = expandedId === streamId;
              const pending = streamId !== undefined && pendingUnfollowIds.includes(streamId);
              const coverImage = getCoverImage(stream);
              const avatar = getAvatar(stream);

              return (
                <article key={streamId ?? title} className={styles.streamCard}>
                  <button
                    className={styles.cardToggle}
                    type="button"
                    onClick={() => setExpandedId(expanded ? null : streamId ?? null)}
                    aria-expanded={expanded}
                  >
                    <div className={styles.coverFrame}>
                      {coverImage ? <img src={coverImage} alt={title} /> : <span>暂无直播画面</span>}
                      <span className={active ? styles.liveBadge : styles.offlineBadge}>
                        {active ? '直播中' : '已结束'}
                      </span>
                    </div>

                    <div className={styles.cardBody}>
                      <div className={active ? styles.avatarFrameActive : styles.avatarFrame}>
                        {avatar ? <img src={avatar} alt={hostName} /> : <span>{hostName.slice(0, 1)}</span>}
                      </div>
                      <div className={styles.streamInfo}>
                        <h2>{hostName}</h2>
                        <p>{title}</p>
                        <div className={styles.metrics}>
                          <span>{toNumber(stream.viewer_count)} 观看</span>
                          <span>{toNumber(stream.followers_count)} 人收藏</span>
                        </div>
                      </div>
                      <span className={expanded ? styles.chevronOpen : styles.chevron}>⌄</span>
                    </div>
                  </button>

                  <div className={styles.detailPanel}>
                    {expanded && (
                      <p>{stream.description || '直播间简介待补充'}</p>
                    )}
                    <div className={styles.actions}>
                      <button
                        className={styles.primaryButton}
                        type="button"
                        aria-label={`进入直播间 ${title}`}
                        disabled={!active || streamId === undefined}
                        onClick={() => active && streamId !== undefined && handleEnterLiveStream(streamId)}
                      >
                        进入直播间
                      </button>
                      <button
                        className={styles.secondaryButton}
                        type="button"
                        aria-label={`取消收藏 ${title}`}
                        disabled={pending || streamId === undefined}
                        onClick={() => streamId !== undefined && handleUnfollow(streamId)}
                      >
                        {pending ? '取消中...' : '取消收藏'}
                      </button>
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </main>
    </div>
  );
};

export default FollowPage;
