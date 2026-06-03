import { useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  FixedPriceItem,
  generateIdempotencyKey,
  purchase,
} from '../../api/fixedPrice';
import { useToast } from '../Toast';
import styles from './index.module.css';

interface FixedPricePurchaseModalProps {
  item: FixedPriceItem;
  liveStreamId: number;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
  onInsufficientBalance?: () => void;
}

type PurchaseErrorLike = {
  status?: number;
  code?: string;
  response?: {
    status?: number;
    data?: {
      code?: string;
      message?: string;
    };
  };
};

function getProductTitle(item: FixedPriceItem): string {
  return item.product_brief?.title ?? item.product?.title ?? '一口价商品';
}

function getErrorStatus(error: PurchaseErrorLike): number | undefined {
  return error.status ?? error.response?.status;
}

function getErrorCode(error: PurchaseErrorLike): string | undefined {
  return error.code ?? error.response?.data?.code;
}

function isNetworkError(error: unknown): boolean {
  const typed = error as PurchaseErrorLike;
  return !getErrorStatus(typed);
}

function getConflictMessage(code?: string): string {
  if (code === 'FP_ALREADY_BOUGHT' || code === 'ALREADY_PURCHASED' || code === 'ALREADY_BOUGHT') {
    return '您已购买过';
  }

  return '已售罄';
}

export default function FixedPricePurchaseModal({
  item,
  liveStreamId,
  open,
  onClose,
  onSuccess,
  onInsufficientBalance,
}: FixedPricePurchaseModalProps) {
  const navigate = useNavigate();
  const { showToast } = useToast();
  const [submitting, setSubmitting] = useState(false);
  const [showBalanceDialog, setShowBalanceDialog] = useState(false);
  const idempotencyKeyRef = useRef<string | null>(null);

  if (!open) {
    return null;
  }

  const handlePurchase = async () => {
    if (submitting) {
      return;
    }

    setSubmitting(true);
    const idempotencyKey = idempotencyKeyRef.current ?? generateIdempotencyKey();
    idempotencyKeyRef.current = idempotencyKey;

    for (let attempt = 0; attempt < 2; attempt += 1) {
      try {
        await purchase({ itemId: item.id, idempotencyKey });
        showToast('抢到了！', 'success', 2500);
        setSubmitting(false);
        onSuccess();
        return;
      } catch (error) {
        const typedError = error as PurchaseErrorLike;
        const status = getErrorStatus(typedError);
        const code = getErrorCode(typedError);

        if (status === 402) {
          setShowBalanceDialog(true);
          break;
        }

        if (status === 409) {
          showToast(getConflictMessage(code), 'warning', 2500);
          onClose();
          break;
        }

        if (isNetworkError(error) && attempt === 0) {
          continue;
        }

        showToast('网络异常，请稍后重试', 'error', 2500);
        break;
      }
    }

    setSubmitting(false);
  };

  const handleRecharge = () => {
    setShowBalanceDialog(false);
    onInsufficientBalance?.();
    navigate('/wallet/recharge');
  };

  return (
    <div className={styles.backdrop} role="presentation">
      <section
        className={styles.modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby="fixed-price-purchase-title"
      >
        <button
          type="button"
          className={styles.closeButton}
          onClick={onClose}
          aria-label="关闭抢购弹窗"
        >
          ×
        </button>

        <div className={styles.badge}>直播间 {liveStreamId} 限时一口价</div>
        <h2 id="fixed-price-purchase-title" className={styles.title}>
          确认抢购
        </h2>
        <p className={styles.productName}>{getProductTitle(item)}</p>

        <div className={styles.pricePanel}>
          <span className={styles.priceLabel}>一口价</span>
          <strong className={styles.price}>¥{item.price}</strong>
          <span className={styles.stock}>剩 {item.remaining_stock} / {item.total_stock}</span>
        </div>

        <button
          type="button"
          className={styles.purchaseButton}
          disabled={submitting}
          onClick={handlePurchase}
        >
          {submitting ? '抢购中...' : '确认抢购'}
        </button>
      </section>

      {showBalanceDialog && (
        <section
          className={styles.balanceDialog}
          role="alertdialog"
          aria-modal="true"
          aria-labelledby="fixed-price-balance-title"
        >
          <h3 id="fixed-price-balance-title">余额不足，去充值</h3>
          <p>当前余额不足以完成抢购，充值后可继续参与。</p>
          <div className={styles.balanceActions}>
            <button type="button" className={styles.secondaryButton} onClick={() => setShowBalanceDialog(false)}>
              稍后再说
            </button>
            <button type="button" className={styles.rechargeButton} onClick={handleRecharge}>
              去充值
            </button>
          </div>
        </section>
      )}
    </div>
  );
}
