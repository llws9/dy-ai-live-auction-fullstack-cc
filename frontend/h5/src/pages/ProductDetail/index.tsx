import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { auctionApi, bidApi, productApi } from '@/services/api';
import { useAuth } from '@/store/authContext';
import styles from './ProductDetail.module.css';

interface AuctionDetail {
  id: number;
  product_id?: number;
  product?: ProductDetailData;
  live_stream_id?: number | null;
  status?: number;
  current_price?: number;
  start_price?: number;
  increment?: number;
  cap_price?: number;
  start_time?: string;
  end_time?: string;
}

interface AuctionRules {
  start_price?: number;
  increment?: number;
  cap_price?: number;
  trigger_delay_before?: number;
}

interface ProductDetailData {
  id?: number;
  name?: string;
  description?: string;
  images?: string[] | string;
  rules?: AuctionRules;
  start_price?: number;
  increment?: number;
  cap_price?: number;
}

interface BidRecord {
  id?: number;
  user_id?: number;
  user_name?: string;
  amount?: number;
  created_at?: string;
}

const extractList = (response: any): BidRecord[] => {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.bids)) return response.bids;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.data?.bids)) return response.data.bids;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  return [];
};

const getFirstImage = (product?: ProductDetailData | null) => {
  if (!product?.images) return '';
  if (Array.isArray(product.images)) return product.images[0] || '';
  return product.images;
};

const getStatusInfo = (status?: number) => {
  switch (status) {
    case 1:
      return { label: '进行中', active: true };
    case 2:
      return { label: '延时中', active: true };
    case 3:
      return { label: '已结束', active: false };
    case 4:
      return { label: '已取消', active: false };
    case 0:
      return { label: '待开始', active: false };
    default:
      return { label: '未知状态', active: false };
  }
};

const formatPrice = (value?: number) => `¥${Number(value ?? 0).toLocaleString()}`;

const ProductDetail: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { isAuthenticated, user } = useAuth();
  const auctionId = Number(searchParams.get('id') || searchParams.get('auction_id'));

  const [auction, setAuction] = useState<AuctionDetail | null>(null);
  const [product, setProduct] = useState<ProductDetailData | null>(null);
  const [bids, setBids] = useState<BidRecord[]>([]);
  const [bidAmount, setBidAmount] = useState('');
  const [loading, setLoading] = useState(true);
  const [bidding, setBidding] = useState(false);
  const [toastMessage, setToastMessage] = useState('');

  const rules = useMemo<AuctionRules>(() => {
    const productRules = product?.rules || {};
    return {
      start_price: productRules.start_price ?? product?.start_price ?? auction?.start_price ?? 0,
      increment: productRules.increment ?? product?.increment ?? auction?.increment ?? 100,
      cap_price: productRules.cap_price ?? product?.cap_price ?? auction?.cap_price,
      trigger_delay_before: productRules.trigger_delay_before ?? 30,
    };
  }, [auction, product]);

  const currentPrice = auction?.current_price ?? rules.start_price ?? 0;
  const statusInfo = getStatusInfo(auction?.status);
  const productImage = getFirstImage(product);
  const productName = product?.name || (auction ? `竞拍场次 #${auction.id}` : '商品详情');

  const showToast = (message: string) => {
    setToastMessage(message);
    window.setTimeout(() => setToastMessage(''), 2500);
  };

  const loadDetail = useCallback(async () => {
    if (!auctionId) {
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const auctionData = await auctionApi.get(auctionId);
      setAuction(auctionData);

      const productId = auctionData.product_id ?? auctionData.product?.id;
      const [bidsData, productData] = await Promise.all([
        auctionApi.getBids(auctionId).catch(() => []),
        productId ? productApi.get(productId).catch(() => auctionData.product ?? null) : Promise.resolve(auctionData.product ?? null),
      ]);

      setBids(extractList(bidsData));
      setProduct(productData);
    } catch (error) {
      console.error('获取商品详情失败:', error);
      setAuction(null);
      setProduct(null);
      setBids([]);
    } finally {
      setLoading(false);
    }
  }, [auctionId]);

  useEffect(() => {
    loadDetail();
  }, [loadDetail]);

  const quickBid = (incrementMultiplier: number) => {
    const increment = rules.increment ?? 100;
    setBidAmount(String(currentPrice + increment * incrementMultiplier));
  };

  const handleBid = async () => {
    if (!auction) return;

    if (!isAuthenticated || !user) {
      navigate(`/login?redirect=${encodeURIComponent(`/detail?id=${auction.id}`)}`);
      return;
    }

    const amount = Number(bidAmount);
    const minBid = currentPrice + (rules.increment ?? 100);
    if (!amount) {
      showToast('请输入出价金额');
      return;
    }
    if (amount < minBid) {
      showToast(`最低出价 ${formatPrice(minBid)}`);
      return;
    }

    setBidding(true);
    try {
      const result = await bidApi.placeBid(auction.id, amount);
      setAuction((current) => current ? { ...current, current_price: result?.current_price ?? amount } : current);
      await loadDetail();
      setBidAmount('');
      showToast(`出价成功！${formatPrice(amount)}`);
    } catch (error: any) {
      showToast(error?.message || '出价失败');
    } finally {
      setBidding(false);
    }
  };

  if (loading) {
    return (
      <section className={styles.page}>
        <div className={styles.loading} role="status" aria-live="polite">
          <span className={styles.loadingSpinner} />
          <span className={styles.loadingText}>加载商品详情中...</span>
        </div>
      </section>
    );
  }

  if (!auction) {
    return (
      <section className={styles.page}>
        <div className={styles.empty}>
          <span className={styles.emptyIcon}>◇</span>
          <h1>竞拍不存在</h1>
          <p>需要从首页选择有效竞拍商品进入。</p>
          <Link className={styles.primaryLink} to="/">返回首页</Link>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.page}>
      <header className={styles.header}>
        <Link to="/" className={styles.backButton} aria-label="返回首页">‹</Link>
        <h1 className={styles.headerTitle}>商品详情</h1>
        <span className={styles.sharePlaceholder} aria-label="分享暂未开放" title="分享暂未开放">享</span>
      </header>

      <main className={styles.content}>
        <div className={styles.hero}>
          {productImage ? (
            <img className={styles.heroImage} src={productImage} alt={productName} />
          ) : (
            <div className={styles.heroFallback}>暂无商品图片</div>
          )}
          <span className={`${styles.statusBadge} ${statusInfo.active ? styles.statusActive : ''}`}>
            {statusInfo.label}
          </span>
        </div>

        <section className={styles.summary}>
          <span className={styles.lotNo}>LOT {auction.id}</span>
          <h2 className={styles.productName}>{productName}</h2>
          <div className={styles.priceRow}>
            <div>
              <p className={styles.label}>当前出价</p>
              <strong className={styles.currentPrice}>{formatPrice(currentPrice)}</strong>
            </div>
            <div className={styles.priceMeta}>
              <span>起拍价 {formatPrice(rules.start_price)}</span>
              {auction.end_time && <span>截止 {new Date(auction.end_time).toLocaleString()}</span>}
            </div>
          </div>
        </section>

        <section className={styles.card}>
          <div className={styles.sectionTitleRow}>
            <h3>出价记录</h3>
            <span>{bids.length} 次</span>
          </div>
          {bids.length > 0 ? (
            <div className={styles.bidList}>
              {bids.slice(0, 5).map((bid, index) => (
                <div key={bid.id ?? `${bid.user_id}-${bid.amount}-${index}`} className={styles.bidItem}>
                  <span className={styles.bidRank}>#{index + 1}</span>
                  <span className={styles.bidUser}>{bid.user_name || `用户${bid.user_id ?? ''}`}</span>
                  <strong>{formatPrice(bid.amount)}</strong>
                </div>
              ))}
            </div>
          ) : (
            <p className={styles.emptyText}>暂无出价记录</p>
          )}
        </section>

        <section className={styles.card}>
          <h3>商品描述</h3>
          <p className={styles.description}>{product?.description || '暂无描述'}</p>
        </section>

        <section className={styles.card}>
          <h3>竞拍规则</h3>
          <div className={styles.ruleList}>
            <div><span>加价幅度</span><strong>{formatPrice(rules.increment)}</strong></div>
            <div><span>封顶价</span><strong>{rules.cap_price ? formatPrice(rules.cap_price) : '无封顶价'}</strong></div>
            <div><span>延时规则</span><strong>最后 {rules.trigger_delay_before ?? 30} 秒自动延长</strong></div>
          </div>
        </section>
      </main>

      <footer className={styles.bidBar}>
        {statusInfo.active ? (
          <>
            <div className={styles.quickBids}>
              <button type="button" onClick={() => quickBid(1)}>+{formatPrice(rules.increment)}</button>
              <button type="button" onClick={() => quickBid(5)}>+{formatPrice((rules.increment ?? 100) * 5)}</button>
              <button type="button" onClick={() => quickBid(10)}>+{formatPrice((rules.increment ?? 100) * 10)}</button>
            </div>
            <div className={styles.bidInputRow}>
              <label className={styles.srOnly} htmlFor="product-detail-bid">出价金额</label>
              <input
                id="product-detail-bid"
                inputMode="decimal"
                type="number"
                min={currentPrice + (rules.increment ?? 100)}
                value={bidAmount}
                onChange={(event) => setBidAmount(event.target.value)}
                placeholder="输入出价金额"
              />
              <button type="button" disabled={bidding} onClick={handleBid}>
                {bidding ? '出价中...' : '出价'}
              </button>
            </div>
          </>
        ) : (
          <Link className={styles.resultButton} to={`/result?id=${auction.id}`}>查看竞拍结果</Link>
        )}
      </footer>

      {toastMessage && <div className={styles.toast} role="status">{toastMessage}</div>}
    </section>
  );
};

export default ProductDetail;
