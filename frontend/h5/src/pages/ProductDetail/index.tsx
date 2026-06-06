import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { auctionApi, productApi, productReminderApi } from '@/services/api';
import { useAuth } from '@/store/authContext';
import PageHeader from '@/components/shared/PageHeader';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
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

const isPastEndTime = (endTime?: string) => {
  if (!endTime) return false;
  const parsed = new Date(endTime).getTime();
  return Number.isFinite(parsed) && parsed <= Date.now();
};

const getStatusInfo = (status?: number, endTime?: string) => {
  if ((status === 1 || status === 2) && isPastEndTime(endTime)) {
    return { label: '已结束', active: false, upcoming: false, ended: true };
  }

  switch (status) {
    case 1:
      return { label: '进行中', active: true, upcoming: false, ended: false };
    case 2:
      return { label: '延时中', active: true, upcoming: false, ended: false };
    case 3:
      return { label: '已结束', active: false, upcoming: false, ended: true };
    case 4:
      return { label: '已取消', active: false, upcoming: false, ended: true };
    case 0:
      return { label: '待开始', active: false, upcoming: true, ended: false };
    default:
      return { label: '未知状态', active: false, upcoming: false, ended: false };
  }
};

const formatPrice = (value?: number) => `¥${Number(value ?? 0).toLocaleString()}`;

const formatDateTime = (value?: string) => {
  if (!value) return '';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return '';
  return parsed.toLocaleString('zh-CN', { hour12: false });
};

const isProductReminderSubscribed = (response: any, productId: number) =>
  extractList(response)
    .map((item: any) => item.product_id ?? item.productId)
    .some((id) => id === productId);

const ProductDetail: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { isAuthenticated, user } = useAuth();
  const auctionId = Number(searchParams.get('id') || searchParams.get('auction_id'));

  const [auction, setAuction] = useState<AuctionDetail | null>(null);
  const [product, setProduct] = useState<ProductDetailData | null>(null);
  const [bids, setBids] = useState<BidRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [reminderPending, setReminderPending] = useState(false);
  const [reminderSubscribed, setReminderSubscribed] = useState(false);
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

  const statusInfo = getStatusInfo(auction?.status, auction?.end_time);
  const displayPrice = statusInfo.upcoming ? rules.start_price ?? 0 : auction?.current_price ?? rules.start_price ?? 0;
  const priceLabel = statusInfo.upcoming ? '起拍价' : statusInfo.ended ? '成交价' : '当前出价';
  const timelineLabel = statusInfo.upcoming
    ? '开拍'
    : statusInfo.ended
      ? '成交时间'
      : '截止';
  const timelineTime = formatDateTime(statusInfo.upcoming ? auction?.start_time : auction?.end_time);
  const productImage = getFirstImage(product);
  const productName = repairUtf8Mojibake(product?.name) || (auction ? `竞拍场次 #${auction.id}` : '商品详情');
  const productDescription = repairUtf8Mojibake(product?.description) || '暂无描述';
  const livePath = auction ? `/live?id=${auction.live_stream_id ?? ''}&auction_id=${auction.id}` : '/';

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

      if (isAuthenticated && productId) {
        const reminders = await productReminderApi.list().catch(() => null);
        setReminderSubscribed(reminders ? isProductReminderSubscribed(reminders, productId) : false);
      } else {
        setReminderSubscribed(false);
      }
    } catch (error) {
      console.error('获取商品详情失败:', error);
      setAuction(null);
      setProduct(null);
      setBids([]);
    } finally {
      setLoading(false);
    }
  }, [auctionId, isAuthenticated]);

  useEffect(() => {
    loadDetail();
  }, [loadDetail]);

  const handleSubscribeReminder = async () => {
    const productId = product?.id ?? auction?.product_id;
    if (!auction || !productId) return;

    if (!isAuthenticated || !user) {
      navigate(`/login?redirect=${encodeURIComponent(`/detail?id=${auction.id}`)}`);
      return;
    }

    setReminderPending(true);
    try {
      await productReminderApi.subscribe(productId);
      setReminderSubscribed(true);
      showToast('订阅成功，开拍前将提醒你');
    } catch (error: any) {
      if (typeof error?.message === 'string' && error.message.includes('已经订阅')) {
        setReminderSubscribed(true);
        showToast('你已订阅该商品');
      } else {
        showToast(error?.message || '订阅失败');
      }
    } finally {
      setReminderPending(false);
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
      <PageHeader
        classes={{
          header: styles.header,
          backButton: styles.backButton,
          title: styles.headerTitle,
        }}
        back={{ to: '/' }}
        title="商品详情"
        actions={
          <span className={styles.sharePlaceholder} aria-label="分享暂未开放" title="分享暂未开放">享</span>
        }
      />

      <main className={styles.content}>
        <div className={styles.hero}>
          {productImage ? (
            <img className={styles.heroImage} src={productImage} alt={productName} />
          ) : (
            <div className={styles.heroFallback}>暂无商品图片</div>
          )}
          <span className={`${styles.statusBadge} ${statusInfo.active ? styles.statusActive : statusInfo.upcoming ? styles.statusUpcoming : ''}`}>
            {statusInfo.active && <span className={styles.liveDot} />}
            {statusInfo.upcoming && <span className={styles.upcomingDot} />}
            {statusInfo.label}
          </span>
        </div>

        <section className={styles.summary}>
          <span className={styles.lotNo}>LOT {auction.id}</span>
          <h2 className={styles.productName}>{productName}</h2>
          <div className={styles.priceRow}>
            <div>
              <p className={styles.label}>{priceLabel}</p>
              <strong className={styles.currentPrice}>{formatPrice(displayPrice)}</strong>
            </div>
            <div className={styles.priceMeta}>
              <span>起拍价 {formatPrice(rules.start_price)}</span>
              {timelineTime && <span>{timelineLabel} {timelineTime}</span>}
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
          <p className={styles.description}>{productDescription}</p>
        </section>

        <section className={styles.card}>
          <h3>竞拍规则</h3>
          <div className={styles.ruleList}>
            <div><span>加价幅度</span><strong>{formatPrice(rules.increment)}</strong></div>
            <div><span>封顶价</span><strong>{rules.cap_price ? formatPrice(rules.cap_price) : '无封顶价'}</strong></div>
            <div><span>延时规则</span><strong>最后 {rules.trigger_delay_before ?? 30} 秒自动延长</strong></div>
          </div>
        </section>

        {statusInfo.upcoming && (
          <button
            type="button"
            className={styles.reminderCta}
            disabled={reminderPending || reminderSubscribed}
            onClick={handleSubscribeReminder}
          >
            {reminderPending ? '订阅中...' : reminderSubscribed ? '已订阅' : '订阅开拍提醒'}
          </button>
        )}

        {statusInfo.active && (
          <Link className={styles.reminderCta} to={livePath}>
            参与竞拍
          </Link>
        )}

        {statusInfo.ended && (
          <Link className={styles.resultButton} to={`/result?id=${auction.id}`}>
            查看竞拍结果
          </Link>
        )}
      </main>

      {toastMessage && <div className={styles.toast} role="status">{toastMessage}</div>}
    </section>
  );
};

export default ProductDetail;
