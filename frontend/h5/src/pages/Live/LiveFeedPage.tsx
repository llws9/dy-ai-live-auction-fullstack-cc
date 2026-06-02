import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { liveStreamApi } from '@/services/api';
import { useToast } from '../../components/Toast';
import LiveRoomSlide from './LiveRoomSlide';
import { FEED_PAGE_SIZE, LIVE_STREAM_STATUS, SWIPE_THRESHOLD_PX } from './constants';

interface LiveStreamFeedItem {
  id: number;
  name?: string;
  cover_image?: string;
  status?: number;
  host_name?: string;
  host_avatar?: string;
  viewer_count?: number;
  current_auction_id?: number | null;
  current_product_id?: number | null;
  current_price?: string | null;
}

const extractList = (response: any): LiveStreamFeedItem[] => {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  return [];
};

const extractTotal = (response: any): number | undefined => {
  if (typeof response?.total === 'number') return response.total;
  if (typeof response?.data?.total === 'number') return response.data.total;
  return undefined;
};

const LiveFeedPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { showToast } = useToast();
  const idParam = searchParams.get('id');

  const [loading, setLoading] = useState(true);
  const [rooms, setRooms] = useState<LiveStreamFeedItem[]>([]);
  const [total, setTotal] = useState<number | undefined>(undefined);
  const [lastPageCount, setLastPageCount] = useState(0);
  const [page, setPage] = useState(1);
  const [bidPending, setBidPending] = useState(false);
  const loadingMoreRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    liveStreamApi
      .list(1, FEED_PAGE_SIZE, LIVE_STREAM_STATUS.LIVE)
      .then((res) => {
        if (cancelled) return;
        const list = extractList(res);
        setRooms(list);
        setTotal(extractTotal(res));
        setLastPageCount(list.length);
        setPage(1);
      })
      .catch(() => {
        if (cancelled) return;
        setRooms([]);
        setTotal(undefined);
        setLastPageCount(0);
        setPage(1);
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const currentIndex = useMemo(() => {
    if (idParam == null) return 0;
    const targetId = Number(idParam);
    const idx = rooms.findIndex((room) => room.id === targetId);
    return idx >= 0 ? idx : 0;
  }, [idParam, rooms]);

  const hasMore = useMemo(() => {
    if (typeof total === 'number') return rooms.length < total;
    return lastPageCount === FEED_PAGE_SIZE;
  }, [total, rooms.length, lastPageCount]);

  // 接近末尾预拉下一页
  useEffect(() => {
    if (rooms.length === 0 || !hasMore) return;
    if (currentIndex < rooms.length - 2) return;
    if (loadingMoreRef.current) return;
    loadingMoreRef.current = true;
    const nextPage = page + 1;
    let cancelled = false;
    liveStreamApi
      .list(nextPage, FEED_PAGE_SIZE, LIVE_STREAM_STATUS.LIVE)
      .then((res) => {
        if (cancelled) return;
        const list = extractList(res);
        setRooms((prev) => {
          const existing = new Set(prev.map((r) => r.id));
          const appended = list.filter((r) => !existing.has(r.id));
          return appended.length > 0 ? [...prev, ...appended] : prev;
        });
        const nextTotal = extractTotal(res);
        if (typeof nextTotal === 'number') setTotal(nextTotal);
        setLastPageCount(list.length);
        setPage(nextPage);
      })
      .catch(() => {
        /* 预拉失败保持现状，下次滑动可重试 */
      })
      .finally(() => {
        if (cancelled) return;
        loadingMoreRef.current = false;
      });
    return () => {
      cancelled = true;
    };
  }, [currentIndex, rooms.length, hasMore, page]);

  const goToRoom = (index: number) => {
    const next = rooms[index];
    if (!next) return;
    navigate(`/live?id=${next.id}&auction_id=${next.current_auction_id ?? ''}`, { replace: true });
  };

  const touchStartRef = useRef<{ x: number; y: number } | null>(null);

  const handleTouchStart = (e: React.TouchEvent) => {
    const t = e.touches[0];
    if (!t) return;
    touchStartRef.current = { x: t.clientX, y: t.clientY };
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (bidPending) return; // 出价 pending 锁房：禁止 feed 切房
    const start = touchStartRef.current;
    touchStartRef.current = null;
    if (!start) return;
    const t = e.changedTouches[0];
    if (!t) return;
    const deltaX = t.clientX - start.x;
    const deltaY = t.clientY - start.y;
    if (Math.abs(deltaY) <= Math.abs(deltaX)) return; // 纵向必须占主导

    if (deltaY <= -SWIPE_THRESHOLD_PX) {
      // 上滑 → 下一个
      if (currentIndex >= rooms.length - 1) {
        if (!hasMore) showToast('没有更多直播间');
        return;
      }
      goToRoom(currentIndex + 1);
    } else if (deltaY >= SWIPE_THRESHOLD_PX) {
      // 下滑 → 上一个
      if (currentIndex <= 0) return;
      goToRoom(currentIndex - 1);
    }
  };

  if (loading) {
    return <div>加载中...</div>;
  }

  if (rooms.length === 0) {
    return <div>暂无直播中房间</div>;
  }

  const currentRoom = rooms[currentIndex];
  const urlAuctionIdRaw = Number(searchParams.get('auction_id'));
  const urlAuctionId = Number.isFinite(urlAuctionIdRaw) && urlAuctionIdRaw > 0 ? urlAuctionIdRaw : undefined;

  return (
    <div onTouchStart={handleTouchStart} onTouchEnd={handleTouchEnd}>
      {currentRoom && (
        <LiveRoomSlide
          key={currentRoom.id}
          liveStreamId={currentRoom.id}
          currentAuctionId={currentRoom.current_auction_id}
          urlAuctionId={urlAuctionId}
          active
          onBidPendingChange={setBidPending}
        />
      )}
    </div>
  );
};

export default LiveFeedPage;
