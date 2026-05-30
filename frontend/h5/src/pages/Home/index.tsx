import React, { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { auctionApi, productApi } from '@/services/api';
import PageHeader from '@/components/shared/PageHeader';
import styles from './Home.module.css';

type HomeTab = '全部' | '收藏' | '珠宝腕表' | '艺术品' | '奢侈品' | '收藏品';

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

const tabs: HomeTab[] = ['全部', '收藏', '珠宝腕表', '艺术品', '奢侈品', '收藏品'];

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

const matchesTab = (auction: HomeAuction, activeTab: HomeTab) => {
  if (activeTab === '全部') return true;
  if (activeTab === '收藏') return false;

  const category = auction.product?.category_name || auction.product?.category;
  if (!category) return true;
  return category === activeTab;
};

const HomePage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<HomeTab>('全部');
  const [auctions, setAuctions] = useState<HomeAuction[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchAuctions = useCallback(async () => {
    if (activeTab === '收藏') {
      setAuctions([]);
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const response = await auctionApi.list({ page: 1, page_size: 20 });
      const rawAuctions = extractList(response);

      const products = await Promise.all(
        rawAuctions.map(async (auction) => {
          if (auction.product || !auction.product_id) return undefined;

          try {
            return await productApi.get(auction.product_id);
          } catch (error) {
            console.warn('获取商品详情失败:', auction.product_id, error);
            return undefined;
          }
        })
      );

      setAuctions(rawAuctions.map((auction, index) => normalizeAuction(auction, products[index])));
    } catch (error) {
      console.error('获取竞拍列表失败:', error);
      setAuctions([]);
    } finally {
      setLoading(false);
    }
  }, [activeTab]);

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

  const visibleAuctions = auctions.filter((auction) => matchesTab(auction, activeTab));

  return (
    <section className={styles.page}>
      <PageHeader
        classes={{ header: styles.header, title: styles.title }}
        title="奢华竞拍"
        actions={
          <>
            <span className={styles.iconButton} aria-label="搜索暂未开放" title="搜索暂未开放">
              搜
            </span>
            <Link className={styles.iconButton} to="/following" aria-label="我的关注">
              关
            </Link>
            <Link className={styles.iconButton} to="/notifications" aria-label="消息通知">
              铃
            </Link>
          </>
        }
      />

      <nav className={styles.tabs} aria-label="首页分类">
        {tabs.map((tab) => (
          <button
            key={tab}
            type="button"
            className={`${styles.tab} ${activeTab === tab ? styles.tabActive : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab}
          </button>
        ))}
      </nav>

      <main className={styles.content} id="content-area">
        {loading ? (
          <div className={styles.loading} role="status" aria-live="polite">
            <span className={styles.loadingSpinner} />
            <span className={styles.loadingText}>加载竞拍中...</span>
          </div>
        ) : visibleAuctions.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>◇</span>
            <p className={styles.emptyText}>{activeTab === '收藏' ? '暂无收藏竞拍' : '暂无竞拍数据'}</p>
            {activeTab === '收藏' && <p className={styles.emptyHint}>收藏接口待后端开放后接入。</p>}
          </div>
        ) : (
          <div className={styles.grid}>
            {visibleAuctions.map((auction) => {
              const statusInfo = getStatusInfo(auction.status);
              const productImage = getFirstImage(auction.product);
              const productName = auction.product?.name || `竞拍场次 #${auction.id}`;
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
