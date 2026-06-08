import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { auctionApi, followApi, productApi, productReminderApi } from '@/services/api';
import { notificationApi } from '@/services/notification';
import { useTouchpointNotifications } from '@/hooks/useTouchpointNotifications';
import { useAuth } from '@/store/authContext';
import PageHeader from '@/components/shared/PageHeader';
import BadgeDot from '@/components/BadgeDot';
import { trackEvent } from '@/utils/trackEvent';
import { notifyTouchpointSummaryInvalidated } from '@/utils/touchpointSummaryEvents';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
import PriceFilterSheet, { PriceRange } from './PriceFilterSheet';
import styles from './Home.module.css';

// 固定 tab：「全部」「收藏」无需 category_id；动态 tab 来自 GET /categories
type SpecialTab = '全部' | '收藏';

interface CategoryTab {
  id: number;
  name: string;
}

interface ProductSummary {
  id?: number;
  name?: string;
  image?: string;
  images?: string[] | string;
  category?: string;
  category_name?: string;
  start_price?: number | string;
  rules?: {
    start_price?: number | string;
  };
}

interface RawAuction {
  id: number;
  product_id?: number;
  product?: ProductSummary;
  live_stream_id?: number | null;
  status?: number;
  current_price?: number | string;
  winner_id?: number | string | null;
  start_price?: number | string;
  rules?: {
    start_price?: number | string;
  };
  rule?: {
    start_price?: number | string;
  };
  auction_rule?: {
    start_price?: number | string;
  };
  bid_count?: number;
  bidder_count?: number;
  start_time?: string;
  end_time?: string;
}

interface HomeAuction {
  id: number;
  productId?: number;
  liveStreamId?: number;
  status: number;
  currentPrice: number;
  bidCount: number;
  sold: boolean;
  startTime?: string;
  endTime?: string;
  product?: ProductSummary;
}

interface LiveStream {
  id?: number | string;
  live_stream_id?: number | string;
  name?: string;
  title?: string;
  live_stream_name?: string;
  creator_name?: string;
  host_name?: string;
  status?: string | number;
  current_auctions_count?: number | string;
  auction_count?: number | string;
  followers_count?: number | string;
  viewer_count?: number | string;
  cover_image?: string;
  image?: string;
}

const SPECIAL_TABS: SpecialTab[] = ['全部', '收藏'];
const DEFAULT_PRODUCT_COVER_IMAGE =
  'https://copilot-cn.bytedance.net/api/ide/v1/text_to_image?prompt=premium%20auction%20product%20display%20on%20warm%20neutral%20background%2C%20realistic%20product%20photography%2C%20jade%20jewelry%20and%20luxury%20watch%2C%20soft%20studio%20lighting%2C%20mobile%20ecommerce%20card%20cover&image_size=landscape_4_3';

const SearchIcon = () => (
  <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <path
      d="M10.8 18.1a7.3 7.3 0 1 1 5.16-2.14l4.04 4.04-1.42 1.42-4.04-4.04a7.27 7.27 0 0 1-3.74.72Zm0-2a5.3 5.3 0 1 0 0-10.6 5.3 5.3 0 0 0 0 10.6Z"
      fill="currentColor"
    />
  </svg>
);

const HeartIcon = () => (
  <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <path
      d="M12 20.7 10.74 19.6C5.98 15.44 3 12.84 3 9.28 3 6.36 5.28 4 8.16 4c1.62 0 3.18.76 4.19 1.95A5.48 5.48 0 0 1 16.54 4C19.42 4 21.7 6.36 21.7 9.28c0 3.56-2.98 6.16-7.74 10.32L12 20.7Zm.02-2.66.62-.54c4.24-3.7 6.96-6.08 6.96-8.22C19.6 7.46 18.24 6 16.54 6c-1.32 0-2.6.84-3.06 2.02h-2.24C10.78 6.84 9.48 6 8.16 6 6.46 6 5.1 7.46 5.1 9.28c0 2.14 2.72 4.52 6.96 8.22l-.04.54Z"
      fill="currentColor"
    />
  </svg>
);

const BellIcon = () => (
  <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <path
      d="M18 10.6c0-3.08-1.64-5.66-4.5-6.34V3a1.5 1.5 0 0 0-3 0v1.26C7.64 4.94 6 7.5 6 10.6v4.9l-1.72 1.72V18h15.44v-.78L18 15.5v-4.9ZM8 16v-5.4C8 8.12 9.5 6 12 6s4 2.12 4 4.6V16H8Zm1.86 3a2.24 2.24 0 0 0 4.28 0H9.86Z"
      fill="currentColor"
    />
  </svg>
);

const extractList = <T,>(response: any): T[] => {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.auctions)) return response.auctions;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  if (Array.isArray(response?.data?.auctions)) return response.data.auctions;
  return [];
};

const extractReminderProductIds = (response: any) =>
  new Set(
    extractList<{ product_id?: number; productId?: number }>(response)
      .map((item) => item.product_id ?? item.productId)
      .filter((id): id is number => typeof id === 'number')
  );

const extractCategories = (response: any): CategoryTab[] => {
  const candidates = [
    response,
    response?.list,
    response?.items,
    response?.data,
    response?.data?.list,
    response?.data?.items,
  ];
  for (const c of candidates) {
    if (Array.isArray(c)) {
      return c
        .filter((item: any) => item && typeof item.id === 'number' && typeof item.name === 'string')
        .filter((item: any) => !SPECIAL_TABS.includes(item.name as SpecialTab))
        .map((item: any) => ({ id: item.id, name: item.name }));
    }
  }
  return [];
};

const getFirstImage = (product?: ProductSummary) => {
  if (!product) return '';
  if (product.image) return product.image;
  if (!product.images) return '';
  if (Array.isArray(product.images)) return product.images[0] || '';
  return product.images;
};

const isPastEndTime = (endTime?: string) => {
  if (!endTime) return false;
  const parsed = new Date(endTime).getTime();
  return Number.isFinite(parsed) && parsed <= Date.now();
};

const getStatusInfo = (status: number, endTime?: string) => {
  if ((status === 1 || status === 2) && isPastEndTime(endTime)) {
    return { label: '已结束', live: false, ended: true };
  }

  switch (status) {
    case 0:
      return { label: '即将开始', live: false, ended: false };
    case 1:
      return { label: '直播中', live: true, ended: false };
    case 2:
      return { label: '延时中', live: true, ended: false };
    case 3:
      return { label: '已结束', live: false, ended: true };
    case 4:
      return { label: '已取消', live: false, ended: true };
    default:
      return { label: '未知', live: false, ended: false };
  }
};

const formatDateTime = (value?: string) => {
  if (!value) return '';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return '';
  return parsed.toLocaleString('zh-CN', { hour12: false });
};

const getAuctionMetaText = (auction: HomeAuction, statusInfo: ReturnType<typeof getStatusInfo>) => {
  if (auction.status === 0) {
    const startTime = formatDateTime(auction.startTime);
    return startTime ? `开拍 ${startTime}` : '即将开拍';
  }

  if (statusInfo.ended) {
    const endTime = formatDateTime(auction.endTime);
    if (auction.sold) {
      return endTime ? `成交时间 ${endTime}` : '已结束';
    }
    return endTime ? `结束时间 ${endTime}` : '已结束';
  }

  if (auction.bidCount > 0) {
    return `${auction.bidCount}次出价`;
  }

  return '暂无出价';
};

const toNumber = (value: number | string | undefined, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const getAuctionStartPrice = (auction: RawAuction, product?: ProductSummary) =>
  toNumber(
    auction.start_price ??
      auction.rules?.start_price ??
      auction.rule?.start_price ??
      auction.auction_rule?.start_price ??
      auction.product?.start_price ??
      auction.product?.rules?.start_price ??
      product?.start_price ??
      product?.rules?.start_price
  );

const normalizeAuction = (auction: RawAuction, product?: ProductSummary): HomeAuction => {
  const currentPrice = toNumber(auction.current_price);
  const startPrice = getAuctionStartPrice(auction, product);
  const winnerId = auction.winner_id;
  const sold = winnerId !== undefined && winnerId !== null
    ? Number(winnerId) > 0
    : currentPrice > 0;

  return {
    id: auction.id,
    productId: auction.product_id ?? auction.product?.id,
    liveStreamId: auction.live_stream_id ?? undefined,
    status: auction.status ?? 0,
    currentPrice: currentPrice > 0 ? currentPrice : startPrice,
    bidCount: auction.bid_count ?? auction.bidder_count ?? 0,
    sold,
    startTime: auction.start_time,
    endTime: auction.end_time,
    product: auction.product ?? product,
  };
};

const getAuctionSortPriority = (auction: HomeAuction) => {
  if ((auction.status === 1 || auction.status === 2) && !isPastEndTime(auction.endTime)) {
    return 0;
  }
  if (auction.status === 0) {
    return 1;
  }
  return 2;
};

const sortAuctionsForHome = (items: HomeAuction[]) =>
  [...items].sort((a, b) => {
    const priorityDiff = getAuctionSortPriority(a) - getAuctionSortPriority(b);
    if (priorityDiff !== 0) return priorityDiff;
    return b.id - a.id;
  });

const getStreamId = (stream: LiveStream) => stream.id ?? stream.live_stream_id;

const getStreamTitle = (stream: LiveStream) => {
  const title = repairUtf8Mojibake(stream.title || stream.name || stream.live_stream_name);
  if (title) return title;
  const streamId = getStreamId(stream);
  return streamId === undefined || streamId === null ? '直播间' : `直播间 #${streamId}`;
};

const getStreamHostName = (stream: LiveStream) =>
  repairUtf8Mojibake(stream.host_name || stream.creator_name) || '主播';

const getStreamCoverImage = (stream: LiveStream) => stream.cover_image || stream.image || '';

const isLiveStreamActive = (status: LiveStream['status']) => {
  const normalized = String(status ?? '').toLowerCase();
  return status === 1 || ['active', 'live', 'living', 'streaming'].includes(normalized);
};

const hasActiveAuction = (stream: LiveStream) => {
  const count = stream.current_auctions_count ?? stream.auction_count;
  return count === undefined ? true : toNumber(count) > 0;
};

const HomePage: React.FC = () => {
  const navigate = useNavigate();
  // activeTab 用 string 既能存「全部」/「收藏」也能存动态分类 name
  const [activeTab, setActiveTab] = useState<string>('全部');
  const [filterSort, setFilterSort] = useState<'default' | 'hot'>('default');
  const [filterPrice, setFilterPrice] = useState<PriceRange>({});
  const [priceSheetOpen, setPriceSheetOpen] = useState(false);
  const [categories, setCategories] = useState<CategoryTab[]>([]);
  const [auctions, setAuctions] = useState<HomeAuction[]>([]);
  const [favoriteLiveStreams, setFavoriteLiveStreams] = useState<LiveStream[]>([]);
  const [loading, setLoading] = useState(true);
  const auctionRequestSeqRef = useRef(0);
  const [subscribedProductIds, setSubscribedProductIds] = useState<Set<number>>(() => new Set());
  const [reminderPendingProductId, setReminderPendingProductId] = useState<number | null>(null);
  const { isAuthenticated } = useAuth();
  const { unreadTotal } = useTouchpointNotifications();
  const activeCategoryId =
    activeTab === '全部' || activeTab === '收藏'
      ? undefined
      : categories.find((category) => category.name === activeTab)?.id;

  // F-D2：登录后热拉通知（mount + 回到前台），成功后刷新共享触达汇总。
  useEffect(() => {
    if (!isAuthenticated) {
      setSubscribedProductIds(new Set());
      return;
    }
    let cancelled = false;
    const refresh = async () => {
      try {
        await notificationApi.hotPull();
        if (cancelled) return;
        notifyTouchpointSummaryInvalidated();
      } catch (error) {
        console.warn('热拉通知失败:', error);
      }
    };
    refresh();
    const onVisibility = () => {
      if (document.visibilityState === 'visible') {
        refresh();
      }
    };
    document.addEventListener('visibilitychange', onVisibility);
    return () => {
      cancelled = true;
      document.removeEventListener('visibilitychange', onVisibility);
    };
  }, [isAuthenticated]);

  useEffect(() => {
    if (!isAuthenticated) {
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
      .catch((error) => {
        console.warn('获取商品订阅列表失败:', error);
      });

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated]);

  // 启动时拉取分类 tabs（失败不阻塞首屏）
  useEffect(() => {
    let cancelled = false;
    productApi
      .listCategories()
      .then((response) => {
        if (cancelled) return;
        setCategories(extractCategories(response));
      })
      .catch((error) => {
        console.warn('获取分类列表失败:', error);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const fetchAuctions = useCallback(async () => {
    const requestSeq = auctionRequestSeqRef.current + 1;
    auctionRequestSeqRef.current = requestSeq;
    const isLatestRequest = () => requestSeq === auctionRequestSeqRef.current;

    setLoading(true);

    if (activeTab === '收藏') {
      try {
        const response = await followApi.getFollowedLiveStreams(1, 20);
        if (!isLatestRequest()) return;
        setFavoriteLiveStreams(extractList<LiveStream>(response));
        setAuctions([]);
      } catch (error) {
        console.error('获取收藏直播间失败:', error);
        if (!isLatestRequest()) return;
        setFavoriteLiveStreams([]);
      } finally {
        if (isLatestRequest()) {
          setLoading(false);
        }
      }
      return;
    }

    try {
      const params: {
        page: number;
        page_size: number;
        category_id?: number;
        sort?: string;
        price_min?: number;
        price_max?: number;
      } = {
        page: 1,
        page_size: 20,
      };
      if (activeCategoryId !== undefined) {
        params.category_id = activeCategoryId;
      }
      if (filterSort === 'hot') params.sort = 'hot';
      if (filterPrice.min !== undefined) params.price_min = filterPrice.min;
      if (filterPrice.max !== undefined) params.price_max = filterPrice.max;

      const response = await auctionApi.list(params);
      if (!isLatestRequest()) return;
      const rawAuctions = extractList<RawAuction>(response);
      const normalized = rawAuctions.map((auction) => normalizeAuction(auction));

      setFavoriteLiveStreams([]);
      // hot 态保留后端排序，避免客户端 feed 排序覆盖「最热」结果。
      setAuctions(filterSort === 'hot' ? normalized : sortAuctionsForHome(normalized));
    } catch (error) {
      console.error('获取竞拍列表失败:', error);
      if (!isLatestRequest()) return;
      setAuctions([]);
    } finally {
      if (isLatestRequest()) {
        setLoading(false);
      }
    }
  }, [activeTab, activeCategoryId, filterSort, filterPrice.min, filterPrice.max]);

  useEffect(() => {
    fetchAuctions();
    return () => {
      auctionRequestSeqRef.current += 1;
    };
  }, [fetchAuctions]);

  const handleSubscribeReminder = async (productId?: number) => {
    if (!productId) return;
    if (!isAuthenticated) {
      navigate(`/login?redirect=${encodeURIComponent('/')}`);
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
        console.error('订阅开拍提醒失败:', error);
      }
    } finally {
      setReminderPendingProductId(null);
    }
  };

  return (
    <section className={styles.page}>
      <PageHeader
        classes={{ header: styles.header, title: styles.title }}
        title="奢华竞拍"
        actions={
          <>
            <span className={styles.iconButton} aria-label="搜索暂未开放" title="搜索暂未开放">
              <SearchIcon />
            </span>
            <Link className={styles.iconButton} to="/following" aria-label="我的收藏">
              <HeartIcon />
            </Link>
            <Link
              className={styles.iconButton}
              to="/notifications"
              aria-label="消息通知"
              onClick={() =>
                trackEvent('entry_clicked', {
                  source: 'home',
                  entry: 'notification_bell',
                  type: 'notification',
                  result: 'clicked',
                })
              }
            >
              <BellIcon />
              {unreadTotal > 0 && (
                <BadgeDot count={unreadTotal} className={styles.notificationBadge} />
              )}
            </Link>
          </>
        }
      />

      <nav className={styles.tabs} aria-label="首页分类">
        {SPECIAL_TABS.map((tab) => (
          <button
            key={tab}
            type="button"
            className={`${styles.tab} ${activeTab === tab ? styles.tabActive : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab}
          </button>
        ))}
        {categories.map((cat) => (
          <button
            key={cat.id}
            type="button"
            className={`${styles.tab} ${activeTab === cat.name ? styles.tabActive : ''}`}
            onClick={() => setActiveTab(cat.name)}
          >
            {cat.name}
          </button>
        ))}
      </nav>

      {activeTab !== '收藏' && (
        <div className={styles.filters} aria-label="排序与价格筛选">
          <button
            type="button"
            className={`${styles.filterPill} ${filterSort === 'default' ? styles.filterPillActive : ''}`}
            onClick={() => setFilterSort('default')}
          >
            综合
          </button>
          <button
            type="button"
            className={`${styles.filterPill} ${filterSort === 'hot' ? styles.filterPillActive : ''}`}
            onClick={() => setFilterSort('hot')}
          >
            最热
          </button>
          <button
            type="button"
            className={`${styles.filterPill} ${
              filterPrice.min !== undefined || filterPrice.max !== undefined ? styles.filterPillActive : ''
            }`}
            onClick={() => setPriceSheetOpen(true)}
          >
            {filterPrice.min !== undefined || filterPrice.max !== undefined
              ? `¥${filterPrice.min ?? 0}${filterPrice.max !== undefined ? `-${filterPrice.max}` : '+'}`
              : '价格区间'}
          </button>
        </div>
      )}

      <PriceFilterSheet
        open={priceSheetOpen}
        value={filterPrice}
        onClose={() => setPriceSheetOpen(false)}
        onConfirm={(range) => setFilterPrice(range)}
      />

      <main className={styles.content} id="content-area">
        {loading ? (
          <div className={styles.loading} role="status" aria-live="polite">
            <span className={styles.loadingSpinner} />
            <span className={styles.loadingText}>加载竞拍中...</span>
          </div>
        ) : (activeTab === '收藏' ? favoriteLiveStreams.length === 0 : auctions.length === 0) ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>◇</span>
            <p className={styles.emptyText}>{activeTab === '收藏' ? '暂无收藏直播间' : '暂无竞拍数据'}</p>
            {activeTab === '收藏' && <p className={styles.emptyHint}>浏览直播时点击收藏按钮即可添加。</p>}
          </div>
        ) : activeTab === '收藏' ? (
          <div className={styles.grid}>
            {favoriteLiveStreams.map((stream) => {
              const title = getStreamTitle(stream);
              const hostName = getStreamHostName(stream);
              const active = isLiveStreamActive(stream.status) && hasActiveAuction(stream);
              const coverImage = getStreamCoverImage(stream);
              const streamId = getStreamId(stream);

              return (
                <article key={streamId ?? title} className={styles.card}>
                  <div className={styles.imageWrapper}>
                    {coverImage ? (
                      <img
                        alt={title}
                        className={`${styles.image} ${!active ? styles.imageMuted : ''}`}
                        src={coverImage}
                        loading="lazy"
                      />
                    ) : (
                      <div className={styles.imageFallback}>暂无直播画面</div>
                    )}
                    <div className={`${styles.statusBadge} ${active ? styles.statusLive : ''}`}>
                      {active && <span className={styles.liveDot} />}
                      {active ? '直播中' : '已结束'}
                    </div>
                  </div>

                  <div className={styles.cardBody}>
                    <h2 className={styles.productName}>{title}</h2>
                    <div className={styles.metaRow}>
                      <span>{hostName}</span>
                      <span>{toNumber(stream.followers_count)} 人收藏</span>
                    </div>
                    <div className={styles.price}>{toNumber(stream.viewer_count)} 观看</div>
                    <div className={styles.actions}>
                      {active && streamId !== undefined && (
                        <Link to={`/live?id=${streamId}`} className={styles.primaryButton}>
                          进入直播
                        </Link>
                      )}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        ) : (
          <div className={styles.grid}>
            {auctions.map((auction) => {
              const statusInfo = getStatusInfo(auction.status, auction.endTime);
              const productImage = getFirstImage(auction.product) || DEFAULT_PRODUCT_COVER_IMAGE;
              const productName = repairUtf8Mojibake(auction.product?.name) || `竞拍场次 #${auction.id}`;
              const livePath = `/live?id=${auction.liveStreamId ?? ''}&auction_id=${auction.id}`;
              const upcoming = auction.status === 0;
              const subscribed = auction.productId ? subscribedProductIds.has(auction.productId) : false;
              const reminderPending = auction.productId === reminderPendingProductId;
              const metaText = getAuctionMetaText(auction, statusInfo);

              return (
                <article key={auction.id} className={styles.card}>
                  <div className={styles.imageWrapper}>
                    <img
                      alt={productName}
                      className={`${styles.image} ${!statusInfo.live ? styles.imageMuted : ''}`}
                      src={productImage}
                      loading="lazy"
                      onError={(event) => {
                        if (event.currentTarget.src !== DEFAULT_PRODUCT_COVER_IMAGE) {
                          event.currentTarget.src = DEFAULT_PRODUCT_COVER_IMAGE;
                        }
                      }}
                    />
                    <div className={`${styles.statusBadge} ${statusInfo.live ? styles.statusLive : upcoming ? styles.statusUpcoming : ''}`}>
                      {statusInfo.live && <span className={styles.liveDot} />}
                      {upcoming && <span className={styles.upcomingDot} />}
                      {statusInfo.label}
                    </div>
                  </div>

                  <div className={styles.cardBody}>
                    <h2 className={styles.productName}>{productName}</h2>
                    <div className={styles.metaRow}>
                      <span>{metaText}</span>
                      {statusInfo.ended && <span className={styles.dealText}>{auction.sold ? '成交' : '流拍'}</span>}
                    </div>
                    <div className={styles.price}>¥{auction.currentPrice.toLocaleString()}</div>
                    <div className={styles.actions}>
                      <Link to={`/detail?id=${auction.id}`} className={styles.outlineButton}>
                        详情
                      </Link>
                      {statusInfo.ended ? (
                        <Link to={`/result?id=${auction.id}`} className={styles.secondaryButton}>
                          查看结果
                        </Link>
                      ) : upcoming ? (
                        <button
                          type="button"
                          className={styles.primaryButton}
                          disabled={!auction.productId || subscribed || reminderPending}
                          onClick={() => handleSubscribeReminder(auction.productId)}
                        >
                          {reminderPending ? '订阅中...' : subscribed ? '已订阅' : '订阅'}
                        </button>
                      ) : (
                        <Link to={livePath} className={styles.primaryButton}>
                          进入直播
                        </Link>
                      )}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </main>
    </section>
  );
};

export default HomePage;
