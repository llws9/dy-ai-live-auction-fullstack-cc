import React, { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { auctionApi, productApi } from '@/services/api';
import { notificationApi } from '@/services/notification';
import { useAuth } from '@/store/authContext';
import PageHeader from '@/components/shared/PageHeader';
import BadgeDot from '@/components/BadgeDot';
import { trackEvent } from '@/utils/trackEvent';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
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
  images?: string[] | string;
  category?: string;
  category_name?: string;
}

interface RawAuction {
  id: number;
  product_id?: number;
  product?: ProductSummary;
  live_stream_id?: number | null;
  status?: number;
  current_price?: number;
  bid_count?: number;
  bidder_count?: number;
}

interface HomeAuction {
  id: number;
  productId?: number;
  liveStreamId?: number;
  status: number;
  currentPrice: number;
  bidCount: number;
  product?: ProductSummary;
}

const SPECIAL_TABS: SpecialTab[] = ['全部', '收藏'];

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

const extractList = (response: any): RawAuction[] => {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.auctions)) return response.auctions;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  if (Array.isArray(response?.data?.auctions)) return response.data.auctions;
  return [];
};

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
  if (!product?.images) return '';
  if (Array.isArray(product.images)) return product.images[0] || '';
  return product.images;
};

const getStatusInfo = (status: number) => {
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

const normalizeAuction = (auction: RawAuction, product?: ProductSummary): HomeAuction => ({
  id: auction.id,
  productId: auction.product_id ?? auction.product?.id,
  liveStreamId: auction.live_stream_id ?? undefined,
  status: auction.status ?? 0,
  currentPrice: auction.current_price ?? 0,
  bidCount: auction.bid_count ?? auction.bidder_count ?? 0,
  product: auction.product ?? product,
});

const HomePage: React.FC = () => {
  // activeTab 用 string 既能存「全部」/「收藏」也能存动态分类 name
  const [activeTab, setActiveTab] = useState<string>('全部');
  const [categories, setCategories] = useState<CategoryTab[]>([]);
  const [auctions, setAuctions] = useState<HomeAuction[]>([]);
  const [loading, setLoading] = useState(true);
  const [unreadCount, setUnreadCount] = useState(0);
  const { isAuthenticated } = useAuth();

  // F-D2：登录后拉取未读消息数（mount + 回到前台），失败时降级为 0
  useEffect(() => {
    if (!isAuthenticated) {
      setUnreadCount(0);
      return;
    }
    let cancelled = false;
    const refresh = () => {
      notificationApi
        .getUnreadCount()
        .then((res) => {
          if (cancelled) return;
          setUnreadCount(res?.count ?? 0);
        })
        .catch((error) => {
          console.warn('获取未读消息数失败:', error);
        });
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
    if (activeTab === '收藏') {
      setAuctions([]);
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const params: { page: number; page_size: number; category_id?: number } = {
        page: 1,
        page_size: 20,
      };
      if (activeTab !== '全部') {
        const matched = categories.find((c) => c.name === activeTab);
        if (matched) {
          params.category_id = matched.id;
        }
      }

      const response = await auctionApi.list(params);
      const rawAuctions = extractList(response);

      setAuctions(rawAuctions.map((auction) => normalizeAuction(auction)));
    } catch (error) {
      console.error('获取竞拍列表失败:', error);
      setAuctions([]);
    } finally {
      setLoading(false);
    }
  }, [activeTab, categories]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      await fetchAuctions();
      if (cancelled) return;
    };
    load();
    return () => {
      cancelled = true;
    };
  }, [fetchAuctions]);

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
              {unreadCount > 0 && (
                <BadgeDot count={unreadCount} className={styles.notificationBadge} />
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

      <main className={styles.content} id="content-area">
        {loading ? (
          <div className={styles.loading} role="status" aria-live="polite">
            <span className={styles.loadingSpinner} />
            <span className={styles.loadingText}>加载竞拍中...</span>
          </div>
        ) : auctions.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>◇</span>
            <p className={styles.emptyText}>{activeTab === '收藏' ? '暂无收藏竞拍' : '暂无竞拍数据'}</p>
            {activeTab === '收藏' && <p className={styles.emptyHint}>收藏接口待后端开放后接入。</p>}
          </div>
        ) : (
          <div className={styles.grid}>
            {auctions.map((auction) => {
              const statusInfo = getStatusInfo(auction.status);
              const productImage = getFirstImage(auction.product);
              const productName = repairUtf8Mojibake(auction.product?.name) || `竞拍场次 #${auction.id}`;
              const livePath = `/live?id=${auction.liveStreamId ?? ''}&auction_id=${auction.id}`;

              return (
                <article key={auction.id} className={styles.card}>
                  <div className={styles.imageWrapper}>
                    {productImage ? (
                      <img
                        alt={productName}
                        className={`${styles.image} ${!statusInfo.live ? styles.imageMuted : ''}`}
                        src={productImage}
                        loading="lazy"
                      />
                    ) : (
                      <div className={styles.imageFallback}>暂无图片</div>
                    )}
                    <div className={`${styles.statusBadge} ${statusInfo.live ? styles.statusLive : ''}`}>
                      {statusInfo.live && <span className={styles.liveDot} />}
                      {statusInfo.label}
                    </div>
                  </div>

                  <div className={styles.cardBody}>
                    <h2 className={styles.productName}>{productName}</h2>
                    <div className={styles.metaRow}>
                      <span>{auction.bidCount}次出价</span>
                      {statusInfo.ended && <span className={styles.dealText}>成交</span>}
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
