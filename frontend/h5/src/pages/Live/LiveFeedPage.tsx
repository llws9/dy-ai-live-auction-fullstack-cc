import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { auctionApi, liveStreamApi, productReminderApi } from '@/services/api';
import { useAuth } from '@/store/authContext';
import { useToast } from '../../components/Toast';
import LiveEmptyState, { UpcomingAuctionItem } from './LiveEmptyState';
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

const extractList = <T = LiveStreamFeedItem,>(response: any): T[] => {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  return [];
};

const extractTotal = (response: any): number | undefined => {
  if (typeof response?.total === 'number') return response.total;
  if (typeof response?.data?.total === 'number') return response.data.total;
  return undefined;
};

const hasCurrentAuction = (room: LiveStreamFeedItem): boolean => {
  const auctionId = Number(room.current_auction_id);
  return Number.isFinite(auctionId) && auctionId > 0;
};

const extractReminderProductIds = (response: any) =>
  new Set(
    extractList<{ product_id?: number; productId?: number }>(response)
      .map((item) => item.product_id ?? item.productId)
      .filter((id): id is number => typeof id === 'number')
  );

const LiveFeedPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { showToast } = useToast();
  const { isAuthenticated } = useAuth();
  const idParam = searchParams.get('id');

  const [loading, setLoading] = useState(true);
  const [rooms, setRooms] = useState<LiveStreamFeedItem[]>([]);
  const [upcomingAuctions, setUpcomingAuctions] = useState<UpcomingAuctionItem[]>([]);
  const [upcomingFailed, setUpcomingFailed] = useState(false);
  const [subscribedProductIds, setSubscribedProductIds] = useState<Set<number>>(() => new Set());
  const [reminderPendingProductId, setReminderPendingProductId] = useState<number | null>(null);
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

  const auctionRooms = useMemo(() => rooms.filter(hasCurrentAuction), [rooms]);
  const shouldLoadEmptyState = !loading && auctionRooms.length === 0;

  useEffect(() => {
    if (!shouldLoadEmptyState) return;
    let cancelled = false;
    setUpcomingFailed(false);
    auctionApi
      .list({ status: '0', upcoming: true, page: 1, page_size: 2 })
      .then((res) => {
        if (cancelled) return;
        setUpcomingAuctions(extractList<UpcomingAuctionItem>(res).slice(0, 2));
      })
      .catch(() => {
        if (cancelled) return;
        setUpcomingAuctions([]);
        setUpcomingFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [shouldLoadEmptyState]);

  useEffect(() => {
    if (!shouldLoadEmptyState || !isAuthenticated) {
      setSubscribedProductIds(new Set());
      return;
    }
    let cancelled = false;
    productReminderApi
      .list()
      .then((response) => {
        if (cancelled) return;
        setSubscribedProductIds(extractReminderProductIds(response));
      })
      .catch(() => {
        if (cancelled) return;
        setSubscribedProductIds(new Set());
      });
    return () => {
      cancelled = true;
    };
  }, [shouldLoadEmptyState, isAuthenticated]);

  const currentIndex = useMemo(() => {
    if (idParam == null) return 0;
    const targetId = Number(idParam);
    const idx = auctionRooms.findIndex((room) => room.id === targetId);
    return idx >= 0 ? idx : 0;
  }, [idParam, auctionRooms]);

  const hasMore = useMemo(() => {
    if (typeof total === 'number') return rooms.length < total;
    return lastPageCount === FEED_PAGE_SIZE;
  }, [total, rooms.length, lastPageCount]);

  // 接近末尾预拉下一页
  useEffect(() => {
    if (rooms.length === 0 || !hasMore) return;
    if (currentIndex < auctionRooms.length - 2) return;
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
      loadingMoreRef.current = false;
    };
  }, [currentIndex, rooms.length, auctionRooms.length, hasMore, page]);

  const goToRoom = (index: number) => {
    const next = auctionRooms[index];
    if (!next) return;
    navigate(`/live?id=${next.id}&auction_id=${next.current_auction_id ?? ''}`, { replace: true });
  };

  const touchStartRef = useRef<{ x: number; y: number } | null>(null);
  const mouseStartRef = useRef<{ x: number; y: number } | null>(null);

  const handleSwipeDelta = (deltaX: number, deltaY: number) => {
    if (bidPending) return; // 出价 pending 锁房：禁止 feed 切房
    if (Math.abs(deltaY) <= Math.abs(deltaX)) return; // 纵向必须占主导

    if (deltaY <= -SWIPE_THRESHOLD_PX) {
      // 上滑 → 下一个
      if (currentIndex >= auctionRooms.length - 1) {
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

  const handleTouchStart = (e: React.TouchEvent) => {
    const t = e.touches[0];
    if (!t) return;
    touchStartRef.current = { x: t.clientX, y: t.clientY };
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    const start = touchStartRef.current;
    touchStartRef.current = null;
    if (!start) return;
    const t = e.changedTouches[0];
    if (!t) return;
    handleSwipeDelta(t.clientX - start.x, t.clientY - start.y);
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    mouseStartRef.current = { x: e.clientX, y: e.clientY };
  };

  const handleMouseUp = (e: React.MouseEvent) => {
    const start = mouseStartRef.current;
    mouseStartRef.current = null;
    if (!start) return;
    handleSwipeDelta(e.clientX - start.x, e.clientY - start.y);
  };

  const handleUpcomingClick = (auctionId: number) => {
    navigate(`/detail?id=${auctionId}`);
  };

  const handleSubscribeReminder = async (productId?: number) => {
    if (!productId) return;
    if (!isAuthenticated) {
      navigate(`/login?redirect=${encodeURIComponent('/live')}`);
      return;
    }

    setReminderPendingProductId(productId);
    try {
      await productReminderApi.subscribe(productId);
      setSubscribedProductIds((current) => {
        const next = new Set(current);
        next.add(productId);
        return next;
      });
    } catch (error: any) {
      if (typeof error?.message === 'string' && error.message.includes('已经订阅')) {
        setSubscribedProductIds((current) => {
          const next = new Set(current);
          next.add(productId);
          return next;
        });
      } else {
        showToast('订阅失败，请稍后重试');
      }
    } finally {
      setReminderPendingProductId(null);
    }
  };

  if (loading) {
    return <div>加载中...</div>;
  }

  if (rooms.length === 0 || auctionRooms.length === 0) {
    return (
      <LiveEmptyState
        upcomingAuctions={upcomingFailed ? [] : upcomingAuctions}
        subscribedProductIds={subscribedProductIds}
        pendingProductId={reminderPendingProductId}
        onAuctionClick={handleUpcomingClick}
        onSubscribe={handleSubscribeReminder}
      />
    );
  }

  const currentRoom = auctionRooms[currentIndex];
  const urlAuctionIdRaw = Number(searchParams.get('auction_id'));
  const urlAuctionId = Number.isFinite(urlAuctionIdRaw) && urlAuctionIdRaw > 0 ? urlAuctionIdRaw : undefined;

  return (
    <div
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      onMouseDown={handleMouseDown}
      onMouseUp={handleMouseUp}
    >
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
