import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { auctionApi, bidApi, followApi, liveStreamApi, productApi } from '@/services/api';
import WebSocketService from '@/services/websocket';
import { useAuth } from '@/store/authContext';
import { useToast } from '../../components/Toast';
import BidDock from './BidDock';
import styles from './Live.module.css';

interface Auction {
  id: number;
  product_id?: number;
  live_stream_id?: number;
  status?: number;
  current_price?: number;
  start_price?: number;
  end_time?: string;
  product?: Product;
}

interface ProductRules {
  start_price?: number;
  increment?: number;
}

interface Product {
  id?: number;
  name?: string;
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

function toastPayloadFromNotification(notification: any) {
  switch (notification.type) {
    case 'bid_outbid':
      return {
        type: 'danger' as const,
        title: notification.title || '您已被超价',
        message: notification.content || '当前最高价已更新',
        actionText: '重新出价',
      };
    case 'auction_won':
      return {
        type: 'success' as const,
        title: notification.title || '恭喜中标',
        message: notification.content || '请尽快完成支付',
        actionText: '去支付',
      };
    case 'auction_starting':
      return {
        type: 'warning' as const,
        title: notification.title || '截拍预警',
        message: notification.content || '拍品即将截拍',
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
  const { isAuthenticated, token } = useAuth();

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
  const [following, setFollowing] = useState(false);
  const [followersCount, setFollowersCount] = useState(0);
  const [followingPending, setFollowingPending] = useState(false);
  const [connected, setConnected] = useState(false);
  const [toast, setToast] = useState('');
  const [now, setNow] = useState(() => Date.now());
  const { showToast: showGlobalToast } = useToast();

  const currentPrice = auction?.current_price ?? 0;
  const increment = product?.rules?.increment ?? 100;
  const startPrice = product?.rules?.start_price ?? auction?.start_price ?? 0;
  const minBid = Math.max(currentPrice, startPrice) + increment;
  const isActive = auction?.status === 1 || auction?.status === 2;
  const effectiveLiveStreamId = liveStreamId || auction?.live_stream_id || liveStream?.id || 0;
  const productImage = getFirstImage(product || auction?.product);

  const timeLeft = useMemo(() => {
    if (!auction?.end_time) return 0;
    return Math.max(0, Math.floor((new Date(auction.end_time).getTime() - now) / 1000));
  }, [auction?.end_time, now]);

  const showToast = useCallback((message: string) => {
    setToast(message);
    window.setTimeout(() => setToast(''), 2200);
  }, []);

  // sheet 状态由 URL searchParams 单源驱动（spec §14.4）：
  // 打开 sheet 时 push 一条新 history entry，浏览器返回键即可消费抽屉态（先收起抽屉、不离开直播页）。
  const openSheet = useCallback((next: 'bid' | 'info') => {
    if (!active) return; // 非活跃 slide 不读写 URL sheet
    const params = new URLSearchParams(searchParams);
    params.set('sheet', next);
    setSearchParams(params, { replace: false });
  }, [active, searchParams, setSearchParams]);

  // 程序关闭 sheet（onClose / 出价成功）：去除 sheet 参数并 replace，避免再点返回多退一步。
  const closeSheet = useCallback(() => {
    if (!active) return;
    const params = new URLSearchParams(searchParams);
    if (!params.has('sheet')) return;
    params.delete('sheet');
    setSearchParams(params, { replace: true });
  }, [active, searchParams, setSearchParams]);

  const normalizeRanking = useCallback((items: any[]): RankingItem[] => {
    return items
      .map((item, index) => ({
        rank: item.rank ?? index + 1,
        id: item.id ?? item.user_id ?? index,
        user_id: item.user_id,
        user_name: item.user_name || item.username || `用户${item.user_id ?? index + 1}`,
        amount: Number(item.amount ?? item.bid_amount ?? 0),
        created_at: item.created_at,
      }))
      .filter((item) => item.amount > 0)
      .slice(0, 10);
  }, []);

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
        setFollowersCount(Number(followersStats?.count ?? followersStats?.followers_count ?? liveStreamData?.followers_count ?? 0));
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

    const ws = new WebSocketService(auctionId, token ?? undefined);
    const shownNotificationIds = new Set<number | string>();

    // WS 消息归属校验（spec §14.3）：携带 auction_id/live_stream_id 且与当前房间不一致的消息直接丢弃，
    // 不更新价格/排行，避免跨房污染。
    const belongsToThisRoom = (data: any): boolean => {
      if (!data) return true;
      if (data.auction_id != null && Number(data.auction_id) !== auctionId) return false;
      if (data.live_stream_id != null && Number(data.live_stream_id) !== liveStreamId) return false;
      return true;
    };

    ws.on('rank_update', (data) => {
      if (!belongsToThisRoom(data)) return;
      setRanking(normalizeRanking(extractList(data)));
    });
    ws.on('bid_placed', (data) => {
      if (!belongsToThisRoom(data)) return;
      const nextPrice = Number(data?.current_price ?? data?.amount ?? 0);
      if (nextPrice > 0) {
        setAuction((previous) => previous ? { ...previous, current_price: nextPrice } : previous);
      }
      if (data?.ranking) {
        setRanking(normalizeRanking(extractList(data)));
      }
    });
    ws.on('sync_response', (data) => {
      if (!belongsToThisRoom(data)) return;
      if (data?.current_price || data?.status || data?.end_time) {
        setAuction((previous) => previous ? {
          ...previous,
          current_price: data.current_price ?? previous.current_price,
          status: data.status ?? previous.status,
          end_time: data.end_time ?? previous.end_time,
        } : previous);
      }
      if (data?.ranking) {
        setRanking(normalizeRanking(extractList(data)));
      }
    });
    ws.on('auction_ended', (data) => {
      if (!belongsToThisRoom(data)) return;
      setAuction((previous) => previous ? { ...previous, status: 3, current_price: data?.final_price ?? previous.current_price } : previous);
    });
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
      ws.disconnect();
      setConnected(false);
    };
  }, [auctionId, active, liveStreamId, normalizeRanking, token, showGlobalToast, navigate]);

  const handleBid = async () => {
    if (!isAuthenticated) {
      showToast('请先登录后出价');
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
    } catch (error: any) {
      showToast(error?.message || '出价失败，请稍后重试');
    } finally {
      setBidding(false);
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

  const hostName = liveStream?.host_name || liveStream?.creator_name || '拍卖师';
  const roomName = liveStream?.name || '竞拍直播间';
  const liveCoverImage = productImage || liveStream?.cover_image || '';
  const hostAvatar = liveStream?.host_avatar || liveStream?.avatar || '';

  return (
    <section className={styles.page}>
      <div className={`${styles.videoArea} ${sheet !== null ? styles.videoAreaCompact : ''}`}>
        {liveStream?.video_url ? (
          <video className={styles.video} src={liveStream.video_url} poster={liveCoverImage} autoPlay muted loop playsInline />
        ) : liveCoverImage ? (
          <img className={styles.video} src={liveCoverImage} alt={product?.name || roomName} />
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
            {getStatusText(auction.status)}
          </div>
        </header>
      </div>

      <BidDock
        product={product || auction?.product}
        productImage={productImage}
        roomName={roomName}
        currentPrice={currentPrice || startPrice}
        sheet={sheet}
        isAuthenticated={isAuthenticated}
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
          {productImage ? <img src={productImage} alt={product?.name || '竞拍商品'} /> : <div className={styles.productFallback}>暂无图片</div>}
          <div>
            <h1>{product?.name || '竞拍商品'}</h1>
            <p>{roomName}</p>
            <div className={styles.followRow}>
              <button className={styles.followButton} disabled={followingPending} onClick={handleFollow} type="button">
                {followingPending ? '处理中...' : following ? '已收藏' : '收藏'}
              </button>
              <span>{followersCount.toLocaleString()} 人收藏</span>
            </div>
          </div>
        </article>

        <section className={styles.rankingBlock}>
          <h2>出价排行</h2>
          {ranking.length === 0 ? (
            <p className={styles.emptyText}>暂无出价记录</p>
          ) : (
            <div className={styles.rankingList}>
              {ranking.slice(0, 5).map((item, index) => (
                <div className={styles.rankingItem} key={`${item.user_id ?? item.id}-${index}`}>
                  <span className={styles.rank}>{item.rank ?? index + 1}</span>
                  <span className={styles.rankingName}>{item.user_name}</span>
                  <strong>¥{formatMoney(item.amount)}</strong>
                </div>
              ))}
            </div>
          )}
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
              disabled={!isActive || bidding}
            />
          </div>
          <div className={styles.quickBids}>
            {[0, 1, 5].map((multiplier) => (
              <button
                key={multiplier}
                type="button"
                onClick={() => setBidAmount(String(minBid + increment * multiplier))}
                disabled={!isActive || bidding}
              >
                {multiplier === 0 ? '最低价' : `+${formatMoney(increment * multiplier)}`}
              </button>
            ))}
          </div>
          <button className={styles.bidButton} disabled={!isActive || bidding} onClick={handleBid} type="button">
            {bidding ? '出价中...' : '立即出价'}
          </button>
          {!isAuthenticated && <p className={styles.authHint}>请先登录后出价</p>}
          {!isActive && <p className={styles.authHint}>当前竞拍不可出价</p>}
        </section>

        <section className={styles.chatBlock}>
          <h2>直播互动</h2>
          <p>聊天协议尚未开放，当前仅保留直播间互动入口。</p>
        </section>
      </BidDock>

      {toast && <div className={styles.toast} role="status">{toast}</div>}
    </section>
  );
};

export default LiveRoomSlide;
