import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { auctionApi, orderApi, productApi } from '@/services/api';
import { useAuth } from '@/store/authContext';
import styles from './Result.module.css';

interface BidRecord {
  id?: number;
  user_id?: number;
  user_name?: string;
  amount?: number;
  created_at?: string;
}

interface ProductInfo {
  id?: number;
  name?: string;
  images?: string[] | string;
}

interface OrderInfo {
  id?: number;
  status?: number;
  paid_at?: string;
}

interface AuctionResult {
  id?: number;
  auction_id?: number;
  product_id?: number;
  product?: ProductInfo;
  status?: number;
  final_price?: number;
  current_price?: number;
  winner_id?: number;
  order_id?: number;
  order?: OrderInfo;
  won_bid?: BidRecord;
  started_at?: string;
  ended_at?: string;
  delay_used?: number;
}

const formatPrice = (value?: number) => `¥${Number(value ?? 0).toLocaleString()}`;

const getFirstImage = (product?: ProductInfo | null) => {
  if (!product?.images) return '';
  if (Array.isArray(product.images)) return product.images[0] || '';
  return product.images;
};

const getOrderStatusText = (status?: number) => {
  switch (status) {
    case 0:
      return '待支付';
    case 1:
      return '已支付';
    case 2:
      return '已发货';
    case 3:
      return '已完成';
    default:
      return '待确认';
  }
};

const ResultPage: React.FC = () => {
  const navigate = useNavigate();
  const { id: pathId } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const { user } = useAuth();
  const auctionId = Number(pathId ?? searchParams.get('auction_id') ?? searchParams.get('id'));

  const [result, setResult] = useState<AuctionResult | null>(null);
  const [product, setProduct] = useState<ProductInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [paying, setPaying] = useState(false);
  const [toastMessage, setToastMessage] = useState('');

  const showToast = (message: string) => {
    setToastMessage(message);
    window.setTimeout(() => setToastMessage(''), 2500);
  };

  const loadResult = useCallback(async () => {
    if (!auctionId) {
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const resultData = await auctionApi.getResult(auctionId);
      setResult(resultData);

      const embeddedProduct = resultData.product;
      const productId = resultData.product_id ?? embeddedProduct?.id;
      if (embeddedProduct) {
        setProduct(embeddedProduct);
      } else if (productId) {
        const productData = await productApi.get(productId).catch(() => null);
        setProduct(productData);
      } else {
        setProduct(null);
      }
    } catch (error) {
      console.error('获取竞拍结果失败:', error);
      setResult(null);
      setProduct(null);
    } finally {
      setLoading(false);
    }
  }, [auctionId]);

  useEffect(() => {
    loadResult();
  }, [loadResult]);

  const order = result?.order;
  const orderId = order?.id ?? result?.order_id;
  const wonBid = result?.won_bid;
  const auctionNo = result?.auction_id ?? result?.id ?? auctionId;
  const finalPrice = result?.final_price ?? result?.current_price ?? wonBid?.amount ?? 0;
  const isWinner = Boolean(user?.id && result?.winner_id === user.id);
  const productImage = getFirstImage(product);
  const productName = product?.name || (auctionNo ? `竞拍场次 #${auctionNo}` : '竞拍结果');
  const statusText = result?.status === 3 ? '已结束' : '结果已生成';
  const orderStatusText = useMemo(() => getOrderStatusText(order?.status), [order?.status]);

  const handlePay = async () => {
    if (!orderId || paying) return;

    setPaying(true);
    try {
      const paidOrder = await orderApi.pay(orderId);
      setResult((current) => current ? { ...current, order: paidOrder, order_id: paidOrder?.id ?? orderId } : current);
      showToast('支付成功，订单已更新');
    } catch (error: any) {
      showToast(error?.message || '支付失败，请稍后重试');
    } finally {
      setPaying(false);
    }
  };

  if (loading) {
    return (
      <section className={styles.page}>
        <div className={styles.loading} role="status" aria-live="polite">
          <span className={styles.loadingSpinner} />
          <span className={styles.loadingText}>加载竞拍结果中...</span>
        </div>
      </section>
    );
  }

  if (!result) {
    return (
      <section className={styles.page}>
        <div className={styles.empty}>
          <span className={styles.emptyIcon}>◇</span>
          <h1>竞拍结果不存在</h1>
          <p>需要从已结束的竞拍场次进入结果页。</p>
          <Link className={styles.primaryLink} to="/">返回首页</Link>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.page}>
      <header className={styles.header}>
        <button className={styles.backButton} onClick={() => navigate(-1)} aria-label="返回上一页">‹</button>
        <h1 className={styles.headerTitle}>竞拍结果</h1>
        <span className={styles.sharePlaceholder} aria-label="分享暂未开放" title="分享暂未开放">享</span>
      </header>

      <main className={styles.content}>
        <section className={styles.hero}>
          <div className={`${styles.resultBadge} ${isWinner ? styles.resultBadgeWinner : ''}`}>
            <span>{isWinner ? '恭喜中标' : '竞拍已结束'}</span>
          </div>

          <div className={`${styles.productFrame} ${isWinner ? styles.productFrameWinner : ''}`}>
            {productImage ? (
              <img className={styles.productImage} src={productImage} alt={productName} />
            ) : (
              <div className={styles.productFallback}>暂无商品图片</div>
            )}
            <span className={styles.closedTag}>已成交</span>
          </div>

          <p className={styles.lotNo}>LOT {auctionNo}</p>
          <h2 className={styles.productName}>{productName}</h2>
        </section>

        <section className={styles.priceCard}>
          <p className={styles.label}>最终成交价</p>
          <strong className={styles.finalPrice}>{formatPrice(finalPrice)}</strong>
          <div className={styles.winnerRow}>
            <div>
              <span className={styles.avatar}>{(wonBid?.user_name || '中').slice(0, 1)}</span>
            </div>
            <div className={styles.winnerInfo}>
              <strong>{wonBid?.user_name || (result.winner_id ? `用户${result.winner_id}` : '暂无中标者')}</strong>
              <span>中标人</span>
            </div>
            <div className={styles.bidTime}>
              <span>出价时间</span>
              <strong>{wonBid?.created_at ? new Date(wonBid.created_at).toLocaleString() : '---'}</strong>
            </div>
          </div>
        </section>

        <section className={styles.metaCard}>
          <div>
            <span>竞拍状态</span>
            <strong>{statusText}</strong>
          </div>
          <div>
            <span>订单状态</span>
            <strong>{orderId ? orderStatusText : '订单待生成'}</strong>
          </div>
          <div>
            <span>结束时间</span>
            <strong>{result.ended_at ? new Date(result.ended_at).toLocaleString() : '---'}</strong>
          </div>
          {Number(result.delay_used ?? 0) > 0 && (
            <div>
              <span>延时时长</span>
              <strong>{result.delay_used} 秒</strong>
            </div>
          )}
        </section>

        <div className={styles.actions}>
          <Link to="/" className={styles.secondaryLink}>返回首页</Link>
          {isWinner ? (
            <button
              className={styles.primaryButton}
              type="button"
              onClick={handlePay}
              disabled={!orderId || paying || order?.status === 1}
            >
              {order?.status === 1 ? '已支付' : paying ? '支付中...' : orderId ? '立即支付' : '订单待生成'}
            </button>
          ) : (
            <Link to="/" className={styles.primaryLink}>继续竞拍</Link>
          )}
        </div>
      </main>

      {toastMessage && <div className={styles.toast} role="status">{toastMessage}</div>}
    </section>
  );
};

export default ResultPage;
