import React, { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { auctionApi, productApi } from '@/services/api';
import { useAuth } from '@/store/authContext';
import PageHeader from '@/components/shared/PageHeader';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
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
  final_price?: number | string;
  current_price?: number | string;
  winner_id?: number | string | null;
  order_id?: number;
  order?: OrderInfo;
  won_bid?: BidRecord;
  started_at?: string;
  ended_at?: string;
  delay_used?: number;
}

const toPositiveAmount = (value?: number | string) => {
  const amount = Number(value ?? 0);
  return Number.isFinite(amount) && amount > 0 ? amount : undefined;
};

const formatPrice = (value?: number | string) => `¥${Number(value ?? 0).toLocaleString()}`;

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
  const [showPaymentDialog, setShowPaymentDialog] = useState(false);

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
  const finalPrice = toPositiveAmount(result?.final_price)
    ?? toPositiveAmount(result?.current_price)
    ?? toPositiveAmount(wonBid?.amount);
  const winnerId = Number(result?.winner_id ?? 0);
  const sold = Boolean(winnerId > 0 || wonBid || finalPrice !== undefined);
  const isWinner = Boolean(user?.id && winnerId === user.id);
  const productImage = getFirstImage(product);
  const productName = repairUtf8Mojibake(product?.name) || (auctionNo ? `竞拍场次 #${auctionNo}` : '竞拍结果');
  const winnerName = sold ? repairUtf8Mojibake(wonBid?.user_name) || (winnerId > 0 ? `用户${winnerId}` : '暂无中标者') : '';
  const statusText = sold ? (result?.status === 3 ? '已结束' : '结果已生成') : '流拍';
  const orderStatusText = getOrderStatusText(order?.status);

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
      <PageHeader
        classes={{
          header: styles.header,
          backButton: styles.backButton,
          title: styles.headerTitle,
        }}
        back={{ onClick: () => navigate(-1) }}
        title="竞拍结果"
        actions={
          <span className={styles.sharePlaceholder} aria-label="分享暂未开放" title="分享暂未开放">享</span>
        }
      />

      <main className={styles.content}>
        <section className={styles.hero}>
          <div className={`${styles.resultBadge} ${isWinner ? styles.resultBadgeWinner : ''}`}>
            <span>{!sold ? '流拍' : isWinner ? '恭喜中标' : '竞拍已结束'}</span>
          </div>

          <div className={`${styles.productFrame} ${isWinner ? styles.productFrameWinner : ''}`}>
            {productImage ? (
              <img className={styles.productImage} src={productImage} alt={productName} />
            ) : (
              <div className={styles.productFallback}>暂无商品图片</div>
            )}
            <span className={styles.closedTag}>{sold ? '已成交' : '流拍'}</span>
          </div>

          <p className={styles.lotNo}>LOT {auctionNo}</p>
          <h2 className={styles.productName}>{productName}</h2>
        </section>

        <section className={styles.priceCard}>
          <p className={styles.label}>{sold ? '最终成交价' : '未成交'}</p>
          <strong className={styles.finalPrice}>{sold ? formatPrice(finalPrice) : '流拍'}</strong>
          {sold && (
            <div className={styles.winnerRow}>
              <div>
                <span className={styles.avatar}>{winnerName.slice(0, 1)}</span>
              </div>
              <div className={styles.winnerInfo}>
                <strong>{winnerName}</strong>
                <span>中标人</span>
              </div>
              <div className={styles.bidTime}>
                <strong>{wonBid?.created_at ? new Date(wonBid.created_at).toLocaleString() : '---'}</strong>
                <span>出价时间</span>
              </div>
            </div>
          )}
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

        <div className={`${styles.actions} ${styles.actionsSingle}`}>
          {isWinner ? (
            <button className={styles.primaryButton} type="button" onClick={() => setShowPaymentDialog(true)}>
              去支付
            </button>
          ) : (
            <Link to="/" className={styles.primaryLink}>返回首页</Link>
          )}
        </div>
      </main>

      {showPaymentDialog && (
        <div className={styles.dialogBackdrop}>
          <div className={styles.dialog} role="dialog" aria-modal="true" aria-labelledby="payment-dialog-title">
            <h2 id="payment-dialog-title">支付链路待完善</h2>
            <p>当前支付链路仍在建设中，暂时无法在 H5 内完成支付。</p>
            <button className={styles.dialogButton} type="button" onClick={() => setShowPaymentDialog(false)}>
              我知道了
            </button>
          </div>
        </div>
      )}
    </section>
  );
};

export default ResultPage;
