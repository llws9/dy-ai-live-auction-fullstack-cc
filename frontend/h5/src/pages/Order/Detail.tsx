import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ApiError, orderApi } from '@/services/api';
import PageHeader from '@/components/shared/PageHeader';
import styles from './Detail.module.css';

interface OrderRecord {
  id: number;
  auction_id: number;
  product_id: number;
  winner_id: number;
  final_price: number;
  status: number;
  paid_at?: string | null;
  shipped_at?: string | null;
  completed_at?: string | null;
  created_at: string;
  updated_at?: string;
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'success'; data: OrderRecord }
  | { kind: 'not-found' }
  | { kind: 'error'; message: string };

const STATUS_META: Record<number, { label: string; tone: string }> = {
  0: { label: '待支付', tone: 'warning' },
  1: { label: '已支付', tone: 'info' },
  2: { label: '已发货', tone: 'success' },
  3: { label: '已完成', tone: 'neutral' },
};

const formatPrice = (value?: number) => `¥${Number(value ?? 0).toLocaleString()}`;

const formatTime = (value?: string | null) => {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
};

const OrderDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  // 严格校验：必须是正整数，否则直接归 not-found，不发请求
  const parsed = Number(id);
  const isValidId = Number.isInteger(parsed) && parsed > 0;
  const orderId = isValidId ? parsed : 0;

  const [state, setState] = useState<LoadState>(
    isValidId ? { kind: 'loading' } : { kind: 'not-found' }
  );
  const [toast, setToast] = useState('');
  const toastTimerRef = useRef<number | null>(null);

  const showToast = (msg: string) => {
    setToast(msg);
    if (toastTimerRef.current !== null) {
      window.clearTimeout(toastTimerRef.current);
    }
    toastTimerRef.current = window.setTimeout(() => {
      setToast('');
      toastTimerRef.current = null;
    }, 2500);
  };

  // 卸载时清理 toast timer，避免 setState on unmounted
  useEffect(() => {
    return () => {
      if (toastTimerRef.current !== null) {
        window.clearTimeout(toastTimerRef.current);
        toastTimerRef.current = null;
      }
    };
  }, []);

  const load = useCallback(async () => {
    if (!isValidId) {
      setState({ kind: 'not-found' });
      return;
    }
    setState({ kind: 'loading' });
    try {
      const data = await orderApi.get(orderId);
      // 防御性：后端返回空 body 或缺关键字段时归 not-found，避免渲染 NaN
      if (!data || typeof (data as OrderRecord).id !== 'number') {
        setState({ kind: 'not-found' });
        return;
      }
      setState({ kind: 'success', data: data as OrderRecord });
    } catch (err: unknown) {
      if (err instanceof ApiError && err.status === 404) {
        setState({ kind: 'not-found' });
        return;
      }
      const message = err instanceof Error ? err.message : '加载失败';
      setState({ kind: 'error', message });
    }
  }, [orderId, isValidId]);

  useEffect(() => {
    load();
  }, [load]);

  if (state.kind === 'loading') {
    return (
      <section className={styles.page}>
        <div className={styles.center} role="status" aria-live="polite">
          <span className={styles.spinner} />
          <span>加载订单详情中...</span>
        </div>
      </section>
    );
  }

  if (state.kind === 'not-found') {
    return (
      <section className={styles.page}>
        <PageHeader
          classes={{ header: styles.header, backButton: styles.backButton, title: styles.headerTitle }}
          back={{ onClick: () => navigate(-1) }}
          title="订单详情"
        />
        <div className={styles.center}>
          <p className={styles.emptyText}>订单不存在</p>
          <button type="button" className={styles.primaryButton} onClick={() => navigate(-1)}>
            返回
          </button>
        </div>
      </section>
    );
  }

  if (state.kind === 'error') {
    return (
      <section className={styles.page}>
        <PageHeader
          classes={{ header: styles.header, backButton: styles.backButton, title: styles.headerTitle }}
          back={{ onClick: () => navigate(-1) }}
          title="订单详情"
        />
        <div className={styles.center}>
          <p className={styles.emptyText}>加载失败：{state.message}</p>
          <button type="button" className={styles.primaryButton} onClick={load}>
            重试
          </button>
        </div>
      </section>
    );
  }

  const order = state.data;
  const status = STATUS_META[order.status] ?? { label: '未知', tone: 'neutral' };

  return (
    <section className={styles.page}>
      <PageHeader
        classes={{ header: styles.header, backButton: styles.backButton, title: styles.headerTitle }}
        back={{ onClick: () => navigate(-1) }}
        title="订单详情"
      />

      <main className={styles.content}>
        <div className={styles.statusBadge} data-tone={status.tone}>
          {status.label}
        </div>

        <section className={styles.card} aria-label="商品摘要">
          <h2 className={styles.cardTitle}>商品摘要</h2>
          <dl className={styles.kv}>
            <div>
              <dt>竞拍场次</dt>
              <dd>#{order.auction_id}</dd>
            </div>
            <div>
              <dt>商品 ID</dt>
              <dd>#{order.product_id}</dd>
            </div>
            <div>
              <dt>成交价</dt>
              <dd className={styles.priceText}>{formatPrice(order.final_price)}</dd>
            </div>
          </dl>
        </section>

        <section className={styles.card} aria-label="订单金额">
          <h2 className={styles.cardTitle}>订单金额</h2>
          <strong className={styles.amount}>{formatPrice(order.final_price)}</strong>
        </section>

        <section className={styles.card} aria-label="订单时间线">
          <h2 className={styles.cardTitle}>订单时间线</h2>
          <ol className={styles.timeline}>
            <li>
              <span className={styles.timelineLabel}>创建</span>
              <time>{formatTime(order.created_at)}</time>
            </li>
            <li>
              <span className={styles.timelineLabel}>支付</span>
              <time>{formatTime(order.paid_at)}</time>
            </li>
            <li>
              <span className={styles.timelineLabel}>发货</span>
              <time>{formatTime(order.shipped_at)}</time>
            </li>
            {order.status === 3 && (
              <li>
                <span className={styles.timelineLabel}>完成</span>
                <time>{formatTime(order.completed_at)}</time>
              </li>
            )}
          </ol>
        </section>

        <div className={styles.actions}>
          <button type="button" className={styles.secondaryButton} onClick={() => navigate(-1)}>
            返回
          </button>
          <button
            type="button"
            className={styles.primaryButton}
            onClick={() => showToast('客服功能即将上线')}
          >
            联系客服
          </button>
        </div>
      </main>

      {toast && (
        <div className={styles.toast} role="status">
          {toast}
        </div>
      )}
    </section>
  );
};

export default OrderDetailPage;
