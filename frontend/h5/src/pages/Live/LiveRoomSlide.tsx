import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { fetchMyPurchase, type FixedPriceItem } from '@/api/fixedPrice';
import { BidSuccessAnimation } from '@/components/auction/BidSuccessAnimation';
import FixedPriceCard from '@/components/FixedPriceCard';
import FixedPriceFlair from '@/components/FixedPriceFlair';
import FixedPricePurchaseModal from '@/components/FixedPricePurchaseModal';
import { useFixedPriceItems } from '@/hooks/useFixedPriceItems';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi, skyLampApi } from '@/services/api';
import WebSocketService from '@/services/websocket';
import { useAuth } from '@/store/authContext';
import { useDemo } from '@/store/demoContext';
import { useToast } from '../../components/Toast';
import { ChatPanel } from '../../components/LiveChat/ChatPanel';
import { useLiveChatStore } from '../../store/liveChatStore';
import { trackBusinessEvent } from '../../utils/businessEvent';
import { repairUtf8Mojibake } from '../../utils/textEncoding';
import BidDock from './BidDock';
import styles from './Live.module.css';

interface Auction {
  id: number;
  product_id?: number;
  live_stream_id?: number;
  status?: number;
  current_price?: number | string;
  start_price?: number | string;
  end_time?: string;
  rules?: ProductRules;
  rule?: ProductRules;
  auction_rule?: ProductRules;
  product?: Product;
}

interface ProductRules {
  start_price?: number | string;
  increment?: number | string;
}

interface Product {
  id?: number;
  name?: string;
  description?: string;
  images?: string[] | string;
  cover_image?: string;
  rules?: ProductRules;
}

interface LiveStream {
  id?: number;
  name?: string;
  host_name?: string;
  creator_name?: string;
  host_avatar?: string;
  avatar?: string;
  viewer_count?: number;
  followers_count?: number;
  is_following?: boolean;
  cover_image?: string;
  video_url?: string;
}

interface RankingItem {
  rank?: number;
  id?: number;
  user_id?: number;
  user_name?: string;
  amount: number;
  created_at?: string;
}

interface BidSuccessFlair {
  id: number;
  amount: number;
  userName: string;
}

interface WonAnimationState {
  productName: string;
  price: number;
  imageUrl?: string;
}

interface SkyLampNoticeState {
  id: number;
  message: string;
}

export interface LiveRoomSlideProps {
  liveStreamId: number;
  currentAuctionId?: number | null;
  urlAuctionId?: number;
  active: boolean;
  onBidPendingChange?: (pending: boolean) => void;
}

const extractList = (response: any): any[] => {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.ranking)) return response.ranking;
  if (Array.isArray(response?.bids)) return response.bids;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.data?.ranking)) return response.data.ranking;
  if (Array.isArray(response?.data?.bids)) return response.data.bids;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  return [];
};

const getFirstImage = (product?: Product) => {
  if (!product) return '';
  if (product.cover_image) return product.cover_image;
  if (Array.isArray(product.images)) return product.images[0] || '';
  return product.images || '';
};

const formatMoney = (amount: number) => amount.toLocaleString('zh-CN', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});
const BID_SUCCESS_FLAIR_DELAY_MS = 240;
const BID_SUCCESS_FLAIR_VISIBLE_MS = 2800;

const toAmount = (value: unknown, fallback = 0) => {
  const amount = Number(value);
  return Number.isFinite(amount) ? amount : fallback;
};

const toEndTimeIso = (value: unknown): string | undefined => {
  if (value == null) return undefined;

  const text = String(value).trim();
  const parsed = typeof value === 'number'
    ? value
    : /^\d+$/.test(text)
      ? Number(text)
      : new Date(text).getTime();

  if (!Number.isFinite(parsed)) return undefined;
  return new Date(parsed).toISOString();
};

const formatTimeLeft = (seconds: number) => {
  const safeSeconds = Math.max(0, seconds);
  const minutes = Math.floor(safeSeconds / 60);
  const restSeconds = safeSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(restSeconds).padStart(2, '0')}`;
};

const getStatusText = (status?: number) => {
  if (status === 1) return '正在竞拍';
  if (status === 2) return '延时竞拍';
  if (status === 3) return '已结束';
  if (status === 4) return '已取消';
  return '即将开始';
};

const getEffectiveStatusText = (status: number | undefined, expired: boolean) => {
  if (expired && (status === 1 || status === 2)) return '已结束';
  return getStatusText(status);
};

const isAlreadyActiveSkyLampError = (error: any) =>
  typeof error?.message === 'string' && error.message.includes('已有活跃的点天灯订阅');

const extractSkyLampSubscriptions = (response: any) =>
  (response?.subscriptions || response?.data?.subscriptions || []) as Array<{ auction_id?: number | string; status?: number | string }>;

function toastPayloadFromNotification(notification: any) {
  const title = repairUtf8Mojibake(notification.title);
  const content = repairUtf8Mojibake(notification.content);
  switch (notification.type) {
    case 'bid_outbid':
      return {
        type: 'danger' as const,
        title: title || '您已被超价',
        message: content || '当前最高价已更新',
        actionText: '重新出价',
      };
    case 'auction_won':
      return {
        type: 'success' as const,
        title: title || '恭喜中标',
        message: content || '请尽快完成支付',
        actionText: '去支付',
      };
    case 'auction_starting':
      return {
        type: 'warning' as const,
        title: title || '截拍预警',
        message: content || '拍品即将截拍',
      };
    default:
      return null;
  }
}

function auctionResultPathFromNotification(notification: any, fallbackAuctionId: number) {
  const auctionID = notification?.data?.auction_id ?? fallbackAuctionId;
  return auctionID ? `/result?id=${auctionID}` : '/result';
}

const LiveRoomSlide: React.FC<LiveRoomSlideProps> = ({ liveStreamId, currentAuctionId, urlAuctionId, active, onBidPendingChange }) => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { isAuthenticated, token, user } = useAuth();
  const { setCurrentAuctionId, setCurrentLiveStreamId } = useDemo();

  const [auctionId, setAuctionId] = useState(0);
  const [auction, setAuction] = useState<Auction | null>(null);
  const [product, setProduct] = useState<Product | null>(null);
  const [liveStream, setLiveStream] = useState<LiveStream | null>(null);
  const [ranking, setRanking] = useState<RankingItem[]>([]);
  const [loading, setLoading] = useState(true);
  const sheetParam = searchParams.get('sheet');
  const sheet: 'bid' | 'info' | null = sheetParam === 'bid' || sheetParam === 'info' ? sheetParam : null;
  const [bidAmount, setBidAmount] = useState('');
  const [bidding, setBidding] = useState(false);
  const [skyLampConfirmOpen, setSkyLampConfirmOpen] = useState(false);
  const [skyLampPending, setSkyLampPending] = useState(false);
  const [skyLampActive, setSkyLampActive] = useState(false);
  const [skyLampNotice, setSkyLampNotice] = useState<SkyLampNoticeState | null>(null);
  const [following, setFollowing] = useState(false);
  const [followersCount, setFollowersCount] = useState(0);
  const [followingPending, setFollowingPending] = useState(false);
  const [connected, setConnected] = useState(false);
  const [toast, setToast] = useState('');
  const [bidSuccessFlair, setBidSuccessFlair] = useState<BidSuccessFlair | null>(null);
  const [wonAnimation, setWonAnimation] = useState<WonAnimationState | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const { showToast: showGlobalToast } = useToast();
  const wsRef = useRef<WebSocketService | null>(null);
  const enteredRoomsRef = useRef<Set<string>>(new Set());
  const bidFlairDelayTimerRef = useRef<number | null>(null);
  const bidFlairHideTimerRef = useRef<number | null>(null);
  const currentPriceRef = useRef(0);
  const lastBidFlairKeyRef = useRef('');
  const wonAnimationDataRef = useRef<WonAnimationState>({
    productName: '竞拍商品',
    price: 0,
  });
  const [fixedPriceModalItem, setFixedPriceModalItem] = useState<FixedPriceItem | null>(null);
  const [purchasedFixedPriceItemIds, setPurchasedFixedPriceItemIds] = useState<Set<number>>(() => new Set());

  const auctionRules = auction?.rules ?? auction?.rule ?? auction?.auction_rule;
  const currentPrice = toAmount(auction?.current_price);
  const increment = toAmount(auctionRules?.increment ?? product?.rules?.increment, 100);
  const startPrice = toAmount(auctionRules?.start_price ?? product?.rules?.start_price ?? auction?.start_price);
  const minBid = Math.max(currentPrice, startPrice) + increment;
  const timeLeft = useMemo(() => {
    if (!auction?.end_time) return 0;
    return Math.max(0, Math.floor((new Date(auction.end_time).getTime() - now) / 1000));
  }, [auction?.end_time, now]);
  const hasReachedEndTime = Boolean(auction?.end_time && timeLeft <= 0);
  const isActive = (auction?.status === 1 || auction?.status === 2) && !hasReachedEndTime;
  const effectiveLiveStreamId = liveStreamId || auction?.live_stream_id || liveStream?.id || 0;
  const { items: fixedPriceItems, socket: fixedPriceSocket } = useFixedPriceItems(effectiveLiveStreamId);
  const productImage = getFirstImage(product || auction?.product);
  const hostName = liveStream?.host_name || liveStream?.creator_name || '拍卖师';
  const roomName = repairUtf8Mojibake(liveStream?.name) || '竞拍直播间';
  const productName = repairUtf8Mojibake(product?.name || auction?.product?.name) || '竞拍商品';
  const productIntro = repairUtf8Mojibake(product?.description || auction?.product?.description || roomName);
  const liveCoverImage = productImage || liveStream?.cover_image || '';
  const hostAvatar = liveStream?.host_avatar || liveStream?.avatar || '';
  const hasEnded = auction?.status === 3 || hasReachedEndTime;
  const fixedPriceItemIds = useMemo(() => fixedPriceItems.map((item) => item.id), [fixedPriceItems]);
  const currentUserDisplayName = useMemo(() => repairUtf8Mojibake(user?.name) || '当前用户', [user?.name]);
  const showSkyLampNotice = useCallback((message: string) => {
    setSkyLampNotice((previous) => ({
      id: (previous?.id ?? 0) + 1,
      message,
    }));
  }, []);

  useEffect(() => {
    currentPriceRef.current = currentPrice;
  }, [currentPrice]);

  useEffect(() => {
    wonAnimationDataRef.current = {
      productName,
      price: currentPrice || startPrice,
      imageUrl: productImage || undefined,
    };
  }, [currentPrice, productImage, productName, startPrice]);

  useEffect(() => {
    setPurchasedFixedPriceItemIds(new Set());
  }, [effectiveLiveStreamId]);

  useEffect(() => {
    setCurrentAuctionId(active && auctionId > 0 ? auctionId : null);

    return () => {
      setCurrentAuctionId(null);
    };
  }, [active, auctionId, setCurrentAuctionId]);

  useEffect(() => {
    setCurrentLiveStreamId(active && effectiveLiveStreamId > 0 ? effectiveLiveStreamId : null);

    return () => {
      setCurrentLiveStreamId(null);
    };
  }, [active, effectiveLiveStreamId, setCurrentLiveStreamId]);

  useEffect(() => {
    const eventProductId = auction?.product_id ?? auction?.product?.id;
    if (!active || !isAuthenticated || effectiveLiveStreamId <= 0 || auctionId <= 0 || !eventProductId) {
      return;
    }
    const eventKey = `${effectiveLiveStreamId}:${auctionId}:${eventProductId}`;
    if (enteredRoomsRef.current.has(eventKey)) {
      return;
    }
    enteredRoomsRef.current.add(eventKey);
    trackBusinessEvent('live_room_enter', {
      source: 'live_room',
      liveStreamId: effectiveLiveStreamId,
      auctionId,
      productId: eventProductId,
    });
  }, [active, isAuthenticated, effectiveLiveStreamId, auctionId, auction?.product_id, auction?.product?.id]);

  useEffect(() => {
    if (!isAuthenticated || fixedPriceItemIds.length === 0) {
      return undefined;
    }

    let cancelled = false;
    Promise.all(
      fixedPriceItemIds.map(async (itemId) => {
        try {
          const result = await fetchMyPurchase(itemId);
          return result.i_bought ? itemId : null;
        } catch {
          return null;
        }
      })
    ).then((boughtItemIds) => {
      if (cancelled) {
        return;
      }
      setPurchasedFixedPriceItemIds((current) => {
        const next = new Set(current);
        boughtItemIds.forEach((itemId) => {
          if (itemId !== null) {
            next.add(itemId);
          }
        });
        return next;
      });
    });

    return () => {
      cancelled = true;
    };
  }, [fixedPriceItemIds, isAuthenticated]);

  const showToast = useCallback((message: string) => {
    setToast(message);
    window.setTimeout(() => setToast(''), 2200);
  }, []);

  const showBidSuccessFlair = useCallback((amount: number, userName: string) => {
    if (bidFlairDelayTimerRef.current !== null) {
      window.clearTimeout(bidFlairDelayTimerRef.current);
    }
    if (bidFlairHideTimerRef.current !== null) {
      window.clearTimeout(bidFlairHideTimerRef.current);
    }

    bidFlairDelayTimerRef.current = window.setTimeout(() => {
      setBidSuccessFlair({
        id: Date.now(),
        amount,
        userName,
      });
      bidFlairHideTimerRef.current = window.setTimeout(() => {
        setBidSuccessFlair(null);
        bidFlairHideTimerRef.current = null;
      }, BID_SUCCESS_FLAIR_VISIBLE_MS);
      bidFlairDelayTimerRef.current = null;
    }, BID_SUCCESS_FLAIR_DELAY_MS);
  }, []);

  useEffect(() => {
    return () => {
      if (bidFlairDelayTimerRef.current !== null) {
        window.clearTimeout(bidFlairDelayTimerRef.current);
      }
      if (bidFlairHideTimerRef.current !== null) {
        window.clearTimeout(bidFlairHideTimerRef.current);
      }
    };
  }, []);

  // sheet 状态由 URL searchParams 单源驱动（spec §14.4）：
  // 打开 sheet 时 push 一条新 history entry，浏览器返回键即可消费抽屉态（先收起抽屉、不离开直播页）。
  const openSheet = useCallback((next: 'bid' | 'info') => {
    if (!active) return; // 非活跃 slide 不读写 URL sheet
    if (next === 'bid') {
      trackBusinessEvent('bid_button_click', {
        source: 'live_room',
        liveStreamId: effectiveLiveStreamId,
        auctionId,
        productId: auction?.product_id ?? auction?.product?.id,
      });
    }
    const params = new URLSearchParams(searchParams);
    params.set('sheet', next);
    setSearchParams(params, { replace: false });
  }, [active, auction?.product_id, auction?.product?.id, auctionId, effectiveLiveStreamId, searchParams, setSearchParams]);

  // 程序关闭 sheet（onClose / 出价成功）：去除 sheet 参数并 replace，避免再点返回多退一步。
  const closeSheet = useCallback(() => {
    if (!active) return;
    setSkyLampConfirmOpen(false);
    const params = new URLSearchParams(searchParams);
    if (!params.has('sheet')) return;
    params.delete('sheet');
    setSearchParams(params, { replace: true });
  }, [active, searchParams, setSearchParams]);

  useEffect(() => {
    if (hasEnded && sheet !== null) {
      closeSheet();
    }
  }, [closeSheet, hasEnded, sheet]);

  const normalizeRanking = useCallback((items: any[]): RankingItem[] => {
    return items
      .map((item, index) => ({
        rank: item.rank ?? index + 1,
        id: item.id ?? item.user_id ?? index,
        user_id: item.user_id,
        user_name: repairUtf8Mojibake(item.user_name || item.username) || `用户${item.user_id ?? index + 1}`,
        amount: Number(item.amount ?? item.bid_amount ?? 0),
        created_at: item.created_at,
      }))
      .filter((item) => item.amount > 0)
      .slice(0, 10);
  }, []);

  const applyRealtimeBid = useCallback((userID: number, userName: string, amount: number) => {
    if (!userID || amount <= 0) return;

    setAuction((previous) => previous ? {
      ...previous,
      current_price: Math.max(toAmount(previous.current_price), amount),
    } : previous);
    setRanking((previous) => normalizeRanking([
      ...previous.filter((item) => item.user_id !== userID),
      {
        id: userID,
        user_id: userID,
        user_name: userName,
        amount,
      },
    ].sort((left, right) => right.amount - left.amount)));
  }, [normalizeRanking]);

  const showRemoteBidFlair = useCallback((userID: number, userName: string, amount: number) => {
    if (!userID || userID === user?.id || amount <= 0) return;

    const displayName = repairUtf8Mojibake(userName) || `用户${userID}`;
    const flairKey = `${userID}:${amount}`;
    if (lastBidFlairKeyRef.current === flairKey) return;

    lastBidFlairKeyRef.current = flairKey;
    showBidSuccessFlair(amount, displayName);
  }, [showBidSuccessFlair, user?.id]);

  const applyRealtimeRanking = useCallback((data: any) => {
    const nextRanking = normalizeRanking(extractList(data));
    setRanking(nextRanking);

    const leadingBid = nextRanking[0];
    if (!leadingBid || leadingBid.amount <= 0) return;

    setAuction((previous) => previous ? {
      ...previous,
      current_price: Math.max(toAmount(previous.current_price), leadingBid.amount),
    } : previous);

    if (leadingBid.amount > currentPriceRef.current) {
      showRemoteBidFlair(leadingBid.user_id ?? 0, leadingBid.user_name || '', leadingBid.amount);
    }
  }, [normalizeRanking, showRemoteBidFlair]);

  const loadRanking = useCallback(async (targetAuctionId: number) => {
    if (!targetAuctionId) return;

    try {
      const rankingResponse = await bidApi.getRanking(targetAuctionId, 10);
      const rankingItems = normalizeRanking(extractList(rankingResponse));
      if (rankingItems.length > 0) {
        setRanking(rankingItems);
        return;
      }
    } catch (error) {
      console.warn('获取竞拍排名失败，尝试读取出价记录:', error);
    }

    try {
      const bidsResponse = await auctionApi.getBids(targetAuctionId);
      setRanking(normalizeRanking(extractList(bidsResponse)));
    } catch (error) {
      console.warn('获取出价记录失败:', error);
      setRanking([]);
    }
  }, [normalizeRanking]);

  useEffect(() => {
    let cancelled = false;

    const loadLiveRoom = async () => {
      const fallbackId = currentAuctionId && currentAuctionId > 0 ? currentAuctionId : 0;
      const candidateId = urlAuctionId && urlAuctionId > 0 ? urlAuctionId : fallbackId;

      if (!candidateId) {
        setAuctionId(0);
        setLoading(false);
        return;
      }

      setLoading(true);
      try {
        let effectiveId = candidateId;
        let auctionData: Auction = await auctionApi.get(effectiveId);
        if (cancelled) return;

        // auction_id 归属校验（spec §14.3）：
        // 若使用了 urlAuctionId，但加载结果不属于当前直播间，则回退 currentAuctionId 重新加载
        if (
          urlAuctionId && urlAuctionId > 0 && effectiveId === urlAuctionId &&
          auctionData.live_stream_id != null && auctionData.live_stream_id !== liveStreamId &&
          fallbackId && fallbackId !== effectiveId
        ) {
          console.warn('auction_id 不属于当前直播间，回退 current_auction_id', {
            urlAuctionId,
            auctionLiveStreamId: auctionData.live_stream_id,
            liveStreamId,
            currentAuctionId: fallbackId,
          });
          effectiveId = fallbackId;
          auctionData = await auctionApi.get(effectiveId);
          if (cancelled) return;
        }

        setAuctionId(effectiveId);
        setAuction(auctionData);
        const resolvedProductId = auctionData.product_id ?? auctionData.product?.id;
        const resolvedLiveStreamId = liveStreamId || auctionData.live_stream_id;

        const [productData, liveStreamData, followersStats, followStatus] = await Promise.all([
          resolvedProductId ? productApi.get(resolvedProductId).catch(() => auctionData.product ?? null) : Promise.resolve(auctionData.product ?? null),
          resolvedLiveStreamId ? liveStreamApi.get(resolvedLiveStreamId).catch(() => null) : Promise.resolve(null),
          resolvedLiveStreamId ? followApi.getFollowersStats(resolvedLiveStreamId).catch(() => null) : Promise.resolve(null),
          resolvedLiveStreamId && isAuthenticated
            ? followApi.getFollowStatus(resolvedLiveStreamId).catch(() => null)
            : Promise.resolve(null),
        ]);

        if (cancelled) return;
        setProduct(productData);
        setLiveStream(liveStreamData);
        // 登录态优先使用 follow-status 接口的权威值，未登录或失败时回退到详情接口字段
        const authoritativeFollowing =
          followStatus && typeof followStatus.is_following === 'boolean'
            ? followStatus.is_following
            : Boolean(liveStreamData?.is_following);
        setFollowing(authoritativeFollowing);
        setFollowersCount(Number(followersStats?.count ?? followersStats?.followers_count ?? followersStats?.total_count ?? liveStreamData?.followers_count ?? 0));
        await loadRanking(effectiveId);
      } catch (error) {
        console.error('加载直播竞拍失败:', error);
        if (!cancelled) {
          setAuction(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadLiveRoom();

    return () => {
      cancelled = true;
    };
  }, [liveStreamId, currentAuctionId, urlAuctionId, loadRanking, isAuthenticated]);

  useEffect(() => {
    setBidAmount(String(minBid));
  }, [minBid]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!active) return;
    if (!auctionId) return;

    const ws = new WebSocketService(auctionId, token ?? undefined, liveStreamId || undefined);
    wsRef.current = ws;
    const shownNotificationIds = new Set<number | string>();
    const onChatMessage = (data: any) => useLiveChatStore.getState().receive(data);

    // WS 消息归属校验（spec §14.3）：携带 auction_id/live_stream_id 且与当前房间不一致的消息直接丢弃，
    // 不更新价格/排行，避免跨房污染。
    const belongsToThisRoom = (data: any): boolean => {
      if (!data) return true;
      if (data.auction_id != null && Number(data.auction_id) !== auctionId) return false;
      if (data.live_stream_id != null && Number(data.live_stream_id) !== liveStreamId) return false;
      return true;
    };

    const onDelayTriggered = (data: any) => {
      if (!belongsToThisRoom(data)) return;
      const nextEndTime = toEndTimeIso(data?.new_end_time);
      if (!nextEndTime) return;

      setAuction((previous) => previous ? { ...previous, end_time: nextEndTime, status: 2 } : previous);
      showGlobalToast({
        type: 'info',
        title: '触发防狙击',
        message: '已有新出价，竞拍时间自动延长',
      });
    };

    const onTimeSync = (data: any) => {
      if (!belongsToThisRoom(data)) return;
      const nextEndTime = toEndTimeIso(data?.end_time);
      if (!nextEndTime) return;

      setAuction((previous) => previous ? { ...previous, end_time: nextEndTime } : previous);
    };

    ws.on('chat_message', onChatMessage);
    ws.on('delay_triggered', onDelayTriggered);
    ws.on('time_sync', onTimeSync);
    ws.on('rank_update', (data) => {
      if (!belongsToThisRoom(data)) return;
      applyRealtimeRanking(data);
    });
    ws.on('bid_placed', (data) => {
      if (!belongsToThisRoom(data)) return;
      const nextPrice = toAmount(data?.current_price ?? data?.amount);
      if (nextPrice > 0) {
        setAuction((previous) => previous ? { ...previous, current_price: nextPrice } : previous);
      }
      if (data?.ranking) {
        applyRealtimeRanking(data);
      } else {
        const userID = Number(data?.user_id || 0);
        const displayName = repairUtf8Mojibake(data?.user_name) || `用户${userID || '未知'}`;
        applyRealtimeBid(userID, displayName, nextPrice);
        showRemoteBidFlair(userID, displayName, nextPrice);
      }
    });
    const onSkyLampAutoBid = (data: any) => {
      if (!belongsToThisRoom(data)) return;

      const userID = Number(data?.user_id || 0);
      const amount = toAmount(data?.amount);
      const displayName = userID && userID === user?.id ? currentUserDisplayName : `用户${userID || '未知'}`;
      const amountText = amount > 0 ? ` ¥${formatMoney(amount)}` : '';

      setSkyLampActive((previous) => previous || userID === user?.id);
      applyRealtimeBid(userID, displayName, amount);
      showSkyLampNotice(`${displayName} 点天灯自动跟价${amountText}，继续守住领先`);
    };
    ws.on('sky_lamp_auto_bid', onSkyLampAutoBid);
    ws.on('sync_response', (data) => {
      if (!belongsToThisRoom(data)) return;
      if (data?.current_price !== undefined || data?.status !== undefined || data?.end_time !== undefined) {
        const nextEndTime = toEndTimeIso(data?.end_time);
        setAuction((previous) => previous ? {
          ...previous,
          current_price: data.current_price !== undefined ? toAmount(data.current_price, toAmount(previous.current_price)) : previous.current_price,
          status: data.status ?? previous.status,
          end_time: nextEndTime ?? previous.end_time,
        } : previous);
      }
      if (data?.ranking) {
        setRanking(normalizeRanking(extractList(data)));
      }
    });
    const onAuctionEnded = (data: any) => {
      if (!belongsToThisRoom(data)) return;
      setAuction((previous) => previous ? { ...previous, status: 3, current_price: toAmount(data?.final_price, toAmount(previous.current_price)) } : previous);
    };

    ws.on('auction_ended', onAuctionEnded);
    ws.on('auction_end', onAuctionEnded);
    const unsubscribeNotification = ws.onNotification((notification) => {
      const id = notification.id;
      if (id && shownNotificationIds.has(id)) {
        return;
      }
      if (id) {
        shownNotificationIds.add(id);
      }

      const payload = toastPayloadFromNotification(notification);
      if (!payload) {
        return;
      }

      if (notification.type === 'auction_won') {
        const animationData = wonAnimationDataRef.current;
        setWonAnimation({
          productName: animationData.productName,
          price: toAmount(notification?.data?.final_price, animationData.price),
          imageUrl: animationData.imageUrl,
        });
      }

      showGlobalToast({
        ...payload,
        onAction: notification.type === 'auction_won'
          ? () => navigate(auctionResultPathFromNotification(notification, auctionId))
          : () => window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' }),
      });
    });

    ws.connect()
      .then(() => {
        setConnected(true);
        ws.requestSync();
      })
      .catch((error) => {
        console.warn('WebSocket 连接失败:', error);
        setConnected(false);
      });

    return () => {
      unsubscribeNotification();
      ws.off('chat_message', onChatMessage);
      ws.off('delay_triggered', onDelayTriggered);
      ws.off('time_sync', onTimeSync);
      ws.off('sky_lamp_auto_bid', onSkyLampAutoBid);
      ws.off('auction_ended', onAuctionEnded);
      ws.off('auction_end', onAuctionEnded);
      ws.disconnect();
      wsRef.current = null;
      useLiveChatStore.getState().reset();
      setConnected(false);
    };
  }, [auctionId, active, liveStreamId, normalizeRanking, token, showGlobalToast, navigate, user?.id, currentUserDisplayName, showSkyLampNotice, applyRealtimeBid, applyRealtimeRanking, showRemoteBidFlair]);

  const handleBid = async () => {
    if (!isAuthenticated) {
      showToast('请先登录后出价');
      return;
    }
    if (!isActive) {
      showToast('当前竞拍不可出价');
      return;
    }

    const amount = Number(bidAmount);
    if (!Number.isFinite(amount) || amount < minBid) {
      showToast(`最低出价 ¥${formatMoney(minBid)}`);
      return;
    }

    setBidding(true);
    onBidPendingChange?.(true);
    try {
      const result = await bidApi.placeBid(auctionId, amount);
      const nextPrice = Number(result?.current_price ?? amount);
      setAuction((previous) => previous ? { ...previous, current_price: nextPrice } : previous);
      if (result?.ranking) {
        setRanking(normalizeRanking(extractList(result)));
      } else {
        await loadRanking(auctionId);
      }
      setBidAmount(String(nextPrice + increment));
      showToast('出价成功');
      closeSheet();
      showBidSuccessFlair(nextPrice, currentUserDisplayName);
    } catch (error: any) {
      showToast(error?.message || '出价失败，请稍后重试');
    } finally {
      setBidding(false);
      onBidPendingChange?.(false);
    }
  };

  const activateSkyLampUi = () => {
    setSkyLampActive(true);
    showSkyLampNotice(`${currentUserDisplayName} 开启点天灯，自动守住领先`);
    setSkyLampConfirmOpen(false);
    closeSheet();
  };

  useEffect(() => {
    if (!active || !isAuthenticated || !auctionId) return;

    let cancelled = false;
    skyLampApi.listSubscriptions(1)
      .then((response) => {
        if (cancelled) return;
        const matched = extractSkyLampSubscriptions(response).find((subscription) =>
          Number(subscription.auction_id) === auctionId && Number(subscription.status) === 1
        );
        setSkyLampActive(Boolean(matched));
      })
      .catch(() => {
        // 点天灯状态是体验增强，查询失败不应影响直播间主流程。
      });

    return () => {
      cancelled = true;
    };
  }, [active, auctionId, isAuthenticated]);

  const handleStartSkyLamp = async () => {
    if (!isAuthenticated) {
      showToast('请先登录后点天灯');
      return;
    }
    if (!isActive) {
      showToast('当前竞拍不可点天灯');
      return;
    }

    setSkyLampPending(true);
    onBidPendingChange?.(true);
    try {
      await skyLampApi.startSubscription(auctionId);
      activateSkyLampUi();
      showGlobalToast({
        type: 'success',
        title: '点天灯已开启',
        message: '系统将自动为你守住领先出价',
      });
      await loadRanking(auctionId);
    } catch (error: any) {
      if (isAlreadyActiveSkyLampError(error)) {
        activateSkyLampUi();
        showGlobalToast({
          type: 'success',
          title: '点天灯已开启',
          message: '你已有活跃点天灯订阅，系统将继续自动守住领先',
        });
      } else {
        showToast(error?.message || '点天灯开启失败，请稍后重试');
      }
    } finally {
      setSkyLampPending(false);
      onBidPendingChange?.(false);
    }
  };

  const handleFollow = async () => {
    if (!effectiveLiveStreamId) {
      showToast('直播间信息缺失，暂不能收藏');
      return;
    }
    if (!isAuthenticated) {
      showToast('请先登录后收藏');
      return;
    }

    const previousFollowing = following;
    setFollowing(!previousFollowing);
    setFollowersCount((count) => Math.max(0, count + (previousFollowing ? -1 : 1)));
    setFollowingPending(true);

    try {
      if (previousFollowing) {
        await followApi.unfollowLiveStream(effectiveLiveStreamId);
      } else {
        await followApi.followLiveStream(effectiveLiveStreamId);
      }
      showToast(previousFollowing ? '已取消收藏' : '已收藏直播间');
    } catch (error: any) {
      setFollowing(previousFollowing);
      setFollowersCount((count) => Math.max(0, count + (previousFollowing ? 1 : -1)));
      showToast(error?.message || '收藏操作失败');
    } finally {
      setFollowingPending(false);
    }
  };

  if (loading) {
    return (
      <section className={styles.statePage}>
        <span className={styles.spinner} />
        <p>加载直播竞拍中...</p>
      </section>
    );
  }

  if (!auctionId) {
    return (
      <section className={styles.statePage}>
        <h1>请选择竞拍场次</h1>
        <p>需要从首页或商品详情页进入直播间。</p>
        <Link className={styles.stateButton} to="/">返回首页</Link>
      </section>
    );
  }

  if (!auction) {
    return (
      <section className={styles.statePage}>
        <h1>竞拍不存在</h1>
        <p>当前竞拍可能已下线或暂不可访问。</p>
        <Link className={styles.stateButton} to="/">返回首页</Link>
      </section>
    );
  }

  return (
    <section className={styles.page}>
      <div className={`${styles.videoArea} ${sheet !== null ? styles.videoAreaCompact : ''}`}>
        {liveStream?.video_url ? (
          <video className={styles.video} src={liveStream.video_url} poster={liveCoverImage} autoPlay muted loop playsInline />
        ) : liveCoverImage ? (
          <img className={styles.video} src={liveCoverImage} alt={productName || roomName} />
        ) : (
          <div className={styles.videoFallback}>暂无直播画面</div>
        )}
        <div className={styles.videoGradient} />
        <header className={styles.topBar}>
          <div className={styles.hostPill}>
            <Link className={styles.backLink} to="/">‹</Link>
            <div className={styles.avatar}>
              {hostAvatar ? (
                <img src={hostAvatar} alt={hostName} />
              ) : (
                <span>{hostName.slice(0, 1)}</span>
              )}
            </div>
            <div>
              <p className={styles.hostName}>{hostName}</p>
              <p className={styles.viewerCount}>{(liveStream?.viewer_count ?? 0).toLocaleString()} 在线</p>
            </div>
          </div>
          <div className={styles.statusPill}>
            <span className={isActive ? styles.liveDot : styles.statusDot} />
            {getEffectiveStatusText(auction.status, hasReachedEndTime)}
          </div>
        </header>
      </div>

      {fixedPriceItems.length > 0 && (
        <div
          className={`${styles.fixedPriceList} ${sheet !== null ? styles.fixedPriceListHidden : ''}`}
          aria-label="一口价商品列表"
        >
          {fixedPriceItems.map((item) => (
            <FixedPriceCard
              key={item.id}
              item={item}
              purchased={purchasedFixedPriceItemIds.has(item.id)}
              onPurchase={() => {
                trackBusinessEvent('fixed_price_click', {
                  source: 'fixed_price_card',
                  liveStreamId: effectiveLiveStreamId,
                  auctionId,
                  productId: item.product_id ?? item.product?.id ?? item.product_brief?.id,
                  metadata: { item_id: item.id },
                });
                setFixedPriceModalItem(item);
              }}
            />
          ))}
        </div>
      )}

      <section className={styles.liveChatOverlay} aria-label="直播互动">
        <ChatPanel
          currentUserId={user?.id ?? 0}
          onSend={(text, clientMsgId) => wsRef.current?.sendChat(text, clientMsgId) ?? false}
        />
      </section>

      {hasEnded ? (
        <section className={styles.endedSummary} aria-label="竞拍结束摘要">
          <span className={styles.endedEyebrow}>AUCTION CLOSED</span>
          <h1>本场竞拍已结束</h1>
          <p>{productName}</p>
          <strong>成交价 ¥{formatMoney(currentPrice || startPrice)}</strong>
          <Link className={styles.endedResultButton} to={`/result?id=${auctionId}`}>
            查看竞拍结果
          </Link>
        </section>
      ) : (
        <BidDock
          product={product || auction?.product}
          productImage={productImage}
          roomName={roomName}
          currentPrice={currentPrice || startPrice}
          sheet={sheet}
          isAuthenticated={isAuthenticated}
          bidDisabled={!isActive}
          bidDisabledText="不可出价"
          skyLampActive={skyLampActive}
          onOpen={openSheet}
          onClose={closeSheet}
          onRequireLogin={() => showToast('请先登录后出价')}
        >
        <div className={styles.priceBlock}>
          <span className={styles.priceLabel}>当前最高价</span>
          <strong>¥{formatMoney(currentPrice || startPrice)}</strong>
          <div className={styles.priceMeta}>
            <span>起拍价 ¥{formatMoney(startPrice)}</span>
            <span>加价幅度 ¥{formatMoney(increment)}</span>
          </div>
        </div>

        <div className={`${styles.countdown} ${timeLeft < 10 && timeLeft > 0 ? styles.countdownUrgent : ''}`}>
          <span>距离结拍</span>
          <strong>{formatTimeLeft(timeLeft)}</strong>
          <em>{connected ? '实时同步中' : '实时连接中'}</em>
        </div>

        <article className={styles.productCard}>
          {productImage ? <img src={productImage} alt={productName} /> : <div className={styles.productFallback}>暂无图片</div>}
          <div>
            <h1>{productName}</h1>
            <p>{productIntro}</p>
            <div className={styles.followRow}>
              <button className={styles.followButton} disabled={followingPending} onClick={handleFollow} type="button">
                {followingPending ? '处理中...' : following ? '已收藏' : '收藏'}
              </button>
              <span>{followersCount.toLocaleString()} 人收藏</span>
            </div>
          </div>
        </article>

        <section className={styles.rankingBlock}>
          <div className={styles.rankingGlow}></div>
          <h2 className={styles.rankingBlockTitle}>
            <span className={styles.rankingTrophy}>🏆</span> 出价排行
          </h2>
          <div className={styles.rankingList}>
            {[0, 1, 2].map((index) => {
              const item = ranking[index];
              const isFirst = index === 0;
              const isSecond = index === 1;
              const isEmpty = !item;
              const isMe = isAuthenticated && item?.user_id === user?.id;
              
              return (
                <div 
                  className={`${styles.rankingItem} ${isFirst && !isEmpty ? styles.rankingItemFirst : ''} ${isEmpty ? styles.rankingItemEmpty : ''} ${isMe ? styles.rankingItemMe : ''}`} 
                  key={item ? `${item.user_id ?? item.id}-${index}` : `empty-${index}`}
                >
                  <div className={styles.rankingItemLeft}>
                    <span className={`${styles.rank} ${
                      isFirst && !isEmpty ? styles.rankFirst : 
                      isSecond && !isEmpty ? styles.rankSecond : 
                      !isEmpty ? styles.rankThird :
                      styles.rankEmpty
                    }`}>
                      {index + 1}
                    </span>
                    <span className={`${styles.rankingName} ${isEmpty ? styles.rankingNameEmpty : ''} ${isMe ? styles.rankingNameMe : ''}`}>
                      {item ? (isMe ? '我自己 (当前领先)' : item.user_name) : '虚位以待'}
                    </span>
                  </div>
                  <strong className={`${styles.rankingAmount} ${isFirst && !isEmpty ? styles.rankingAmountFirst : ''} ${isEmpty ? styles.rankingAmountEmpty : ''} ${isMe && !isFirst ? styles.rankingAmountMe : ''}`}>
                    {item ? `¥${formatMoney(item.amount)}` : '-'}
                  </strong>
                </div>
              );
            })}
          </div>
          
          {/* 我的出价状态 - 方案A 悬浮轻量卡片 */}
          <div className={styles.myBidSection}>
            <div className={styles.myBidCard}>
              <div className={styles.myBidLeft}>
                {isAuthenticated ? (
                  <>
                    <div className={styles.myBidRankCircle}>
                      <span className={styles.myBidRank}>
                        {ranking.findIndex(r => r.user_id === user?.id) > -1 
                          ? ranking.findIndex(r => r.user_id === user?.id) + 1 
                          : '-'}
                      </span>
                    </div>
                    <span className={styles.myBidLabel}>当前我的排位</span>
                  </>
                ) : (
                  <span className={styles.myBidLabel}>请登录后查看出价状态</span>
                )}
              </div>
              <strong className={styles.myBidAmount}>
                {isAuthenticated 
                  ? `¥${formatMoney(ranking.find(r => r.user_id === user?.id)?.amount || 0)}` 
                  : '-'}
              </strong>
            </div>
          </div>
        </section>

        <section className={styles.bidBox}>
          <label htmlFor="live-bid-input">输入出价金额</label>
          <div className={styles.bidInputRow}>
            <span>¥</span>
            <input
              id="live-bid-input"
              type="number"
              min={minBid}
              step="0.01"
              value={bidAmount}
              onChange={(event) => setBidAmount(event.target.value)}
              disabled={!isActive || bidding || skyLampPending}
            />
          </div>
          <div className={styles.quickBids}>
            {[0, 1, 5].map((multiplier) => (
              <button
                key={multiplier}
                type="button"
                onClick={() => setBidAmount(String(minBid + increment * multiplier))}
                disabled={!isActive || bidding || skyLampPending}
              >
                {multiplier === 0 ? '最低价' : `+${formatMoney(increment * multiplier)}`}
              </button>
            ))}
          </div>
          <div className={styles.bidActionBar}>
            <button
              className={styles.skyLampButton}
              disabled={!isActive || bidding || skyLampPending || skyLampActive}
              onClick={() => setSkyLampConfirmOpen(true)}
              type="button"
            >
              <i
                className={`${styles.skyLampIcon} ${skyLampActive ? styles.skyLampIconFloating : ''}`}
                data-testid={skyLampActive ? 'sky-lamp-floating-icon' : undefined}
                aria-hidden="true"
              >
                <span />
              </i>
              {skyLampActive ? '守护中' : skyLampPending ? '开启中' : '点天灯'}
            </button>
            <button className={styles.bidButton} disabled={!isActive || bidding || skyLampPending} onClick={handleBid} type="button">
              {bidding ? '出价中...' : isActive ? '立即出价' : '竞拍已结束'}
            </button>
          </div>
          {skyLampConfirmOpen && (
            <div className={styles.skyLampConfirm} role="dialog" aria-modal="false" aria-labelledby="sky-lamp-confirm-title">
              <h3 id="sky-lamp-confirm-title">确认开启点天灯？</h3>
              <p>系统将先出价 ¥{formatMoney(minBid)}，并在别人超过你时自动跟价。你可在订阅中停止。</p>
              <div className={styles.skyLampConfirmActions}>
                <button
                  className={styles.skyLampCancelButton}
                  disabled={skyLampPending}
                  onClick={() => setSkyLampConfirmOpen(false)}
                  type="button"
                >
                  取消
                </button>
                <button
                  className={styles.skyLampConfirmButton}
                  disabled={skyLampPending}
                  onClick={handleStartSkyLamp}
                  type="button"
                >
                  {skyLampPending ? '开启中...' : '确认开启'}
                </button>
              </div>
            </div>
          )}
          {!isAuthenticated && <p className={styles.authHint}>请先登录后出价</p>}
          {!isActive && <p className={styles.authHint}>当前竞拍不可出价</p>}
        </section>
        </BidDock>
      )}

      {skyLampNotice && (
        <div key={skyLampNotice.id} className={styles.skyLampNotice} role="status">
          <i className={styles.skyLampNoticeIcon} aria-hidden="true"><span /></i>
          <strong>{skyLampNotice.message}</strong>
        </div>
      )}
      {bidSuccessFlair && (
        <div
          className={styles.bidSuccessFlair}
          data-testid="bid-success-flair"
          key={bidSuccessFlair.id}
          role="status"
          aria-live="polite"
        >
          <span className={styles.bidSuccessAvatar} aria-hidden="true">
            {bidSuccessFlair.userName.slice(0, 1)}
          </span>
          <span className={styles.bidSuccessCopy}>
            <span>{bidSuccessFlair.userName} 刚刚出价</span>
            <strong>¥{formatMoney(bidSuccessFlair.amount)}</strong>
          </span>
        </div>
      )}
      {wonAnimation && (
        <BidSuccessAnimation
          productName={wonAnimation.productName}
          price={wonAnimation.price}
          imageUrl={wonAnimation.imageUrl}
          onAnimationEnd={() => setWonAnimation(null)}
        />
      )}
      {toast && <div className={styles.toast} role="status">{toast}</div>}
      {fixedPriceModalItem && (
        <FixedPricePurchaseModal
          item={fixedPriceModalItem}
          liveStreamId={effectiveLiveStreamId}
          open={true}
          onClose={() => setFixedPriceModalItem(null)}
          onSuccess={() => {
            trackBusinessEvent('purchase_success', {
              source: 'fixed_price_card',
              liveStreamId: effectiveLiveStreamId,
              auctionId,
              productId: fixedPriceModalItem.product_id ?? fixedPriceModalItem.product?.id ?? fixedPriceModalItem.product_brief?.id,
              metadata: { item_id: fixedPriceModalItem.id },
            });
            setPurchasedFixedPriceItemIds((current) => {
              const next = new Set(current);
              next.add(fixedPriceModalItem.id);
              return next;
            });
            setFixedPriceModalItem(null);
          }}
        />
      )}
      <FixedPriceFlair socket={fixedPriceSocket} />
    </section>
  );
};

export default LiveRoomSlide;
