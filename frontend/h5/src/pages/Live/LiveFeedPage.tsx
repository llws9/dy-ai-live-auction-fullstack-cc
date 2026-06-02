import React, { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { liveStreamApi } from '@/services/api';
import LiveRoomSlide from './LiveRoomSlide';
import { FEED_PAGE_SIZE, LIVE_STREAM_STATUS } from './constants';

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

const LiveFeedPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const idParam = searchParams.get('id');

  const [loading, setLoading] = useState(true);
  const [rooms, setRooms] = useState<LiveStreamFeedItem[]>([]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    liveStreamApi
      .list(1, FEED_PAGE_SIZE, LIVE_STREAM_STATUS.LIVE)
      .then((res) => {
        if (cancelled) return;
        setRooms(extractList(res));
      })
      .catch(() => {
        if (cancelled) return;
        setRooms([]);
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
    <div>
      {currentRoom && (
        <LiveRoomSlide
          key={currentRoom.id}
          liveStreamId={currentRoom.id}
          currentAuctionId={currentRoom.current_auction_id}
          urlAuctionId={urlAuctionId}
          active
        />
      )}
    </div>
  );
};

export default LiveFeedPage;
