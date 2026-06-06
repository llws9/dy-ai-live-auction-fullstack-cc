import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import PageHeader from '@/components/shared/PageHeader';
import { orderApi } from '@/services/api';
import styles from './List.module.css';

type OrderStatus = 0 | 1 | 2 | 3;
type FilterKey = 'all' | OrderStatus;

interface ApiOrder {
  id: number;
  auction_id?: number;
  auctionId?: number;
  product_id?: number;
  productId?: number;
  product_name?: string;
  productName?: string;
  product_image?: string;
  productImage?: string;
  seller_id?: number;
  sellerId?: number;
  seller_name?: string;
  sellerName?: string;
  live_stream_name?: string;
  liveStreamName?: string;
  final_price?: string | number;
  finalPrice?: string | number;
  status: OrderStatus;
  created_at?: string;
  createdAt?: string;
  seller?: string;
}

interface OrderCardModel {
  id: number;
  auctionId: number;
  productId: number;
  productName: string;
  productImage: string;
  finalPrice: number;
  status: OrderStatus;
  createdAt: string;
  seller: string;
}

const pageSize = 20;

const filters: Array<{ key: FilterKey; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 0, label: '待支付' },
  { key: 1, label: '待发货' },
  { key: 2, label: '已发货' },
  { key: 3, label: '已完成' },
];

const statusMeta: Record<OrderStatus, { label: string; tone: string }> = {
  0: { label: '待支付', tone: 'warning' },
  1: { label: '待发货', tone: 'info' },
  2: { label: '已发货', tone: 'success' },
  3: { label: '已完成', tone: 'neutral' },
};

function formatCurrency(value: number) {
  return `¥${value.toLocaleString('zh-CN', { maximumFractionDigits: 0 })}`;
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  });
}

function normalizeOrder(order: ApiOrder): OrderCardModel {
  const auctionId = order.auction_id ?? order.auctionId ?? 0;
  const productId = order.product_id ?? order.productId ?? 0;
  const sellerId = order.seller_id ?? order.sellerId ?? 0;
  const finalPrice = Number(order.final_price ?? order.finalPrice ?? 0);
  const status = statusMeta[order.status] ? order.status : 0;

  return {
    id: order.id,
    auctionId,
    productId,
    productName: order.product_name ?? order.productName ?? `商品 #${productId}`,
    productImage: order.product_image ?? order.productImage ?? '',
    finalPrice: Number.isFinite(finalPrice) ? finalPrice : 0,
    status,
    createdAt: order.created_at ?? order.createdAt ?? '',
    seller: order.seller_name ?? order.sellerName ?? order.seller ?? (sellerId > 0 ? `商家 #${sellerId}` : '商家信息待同步'),
  };
}

function getEmptyCopy(filter: FilterKey) {
  if (filter === 'all') {
    return {
      title: '还没有订单',
      description: '中标后的订单会出现在这里，你可以先去直播间看看正在竞拍的商品。',
    };
  }

  if (filter === 0) {
    return {
      title: '还没有待支付订单',
      description: '你中标后，需要支付的订单会显示在这里。',
    };
  }

  if (filter === 1) {
    return {
      title: '还没有待发货订单',
      description: '订单支付完成后，会进入待发货状态。',
    };
  }

  if (filter === 2) {
    return {
      title: '还没有已发货订单',
      description: '商家发货后，你可以在这里查看物流进度。',
    };
  }

  return {
    title: '还没有已完成订单',
    description: '后续订单完成后，会自动归档到这里。',
  };
}

export default function OrderListPage() {
  const navigate = useNavigate();
  const [activeFilter, setActiveFilter] = useState<FilterKey>('all');
  const [orders, setOrders] = useState<OrderCardModel[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadOrders = useCallback(async (nextPage = 1) => {
    setLoading(true);
    setError('');

    try {
      const response = await orderApi.list({ page: nextPage, page_size: pageSize });
      const list = Array.isArray(response?.list) ? response.list : [];
      const normalized = list.map(normalizeOrder);
      setOrders((current) => (nextPage === 1 ? normalized : [...current, ...normalized]));
      setTotal(Number(response?.total ?? list.length));
      setPage(nextPage);
    } catch {
      setError('订单暂时无法加载');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadOrders(1);
  }, [loadOrders]);

  const stats = useMemo(() => {
    const pendingPay = orders.filter((order) => order.status === 0).length;
    const pendingShip = orders.filter((order) => order.status === 1).length;
    const totalAmount = orders.reduce((sum, order) => sum + order.finalPrice, 0);
    return { pendingPay, pendingShip, totalAmount };
  }, [orders]);

  const visibleOrders = useMemo(() => {
    if (activeFilter === 'all') return orders;
    return orders.filter((order) => order.status === activeFilter);
  }, [activeFilter, orders]);

  const hasNextPage = page * pageSize < total;
  const emptyCopy = getEmptyCopy(activeFilter);

  return (
    <section className={styles.page}>
      <PageHeader
        classes={{
          header: styles.header,
          backButton: styles.backButton,
          eyebrow: styles.eyebrow,
        }}
        back={{ onClick: () => navigate(-1) }}
        eyebrow="Order Ledger"
        title="我的订单"
        actions={<Link className={styles.historyLink} to="/history">竞拍记录</Link>}
      />

      <main className={styles.content}>
        <section className={styles.heroCard} aria-label="订单概览">
          <p>竞拍成交后的订单会沉淀在这里，承接支付、发货和售后状态。</p>
          <div className={styles.summaryGrid}>
            <div>
              <strong>{formatCurrency(stats.totalAmount)}</strong>
              <span>成交金额</span>
            </div>
            <div>
              <strong>{stats.pendingPay}</strong>
              <span>待支付</span>
            </div>
            <div>
              <strong>{stats.pendingShip}</strong>
              <span>待发货</span>
            </div>
          </div>
        </section>

        <div className={styles.filterBar} aria-label="订单状态筛选">
          {filters.map((filter) => (
            <button
              key={String(filter.key)}
              type="button"
              className={activeFilter === filter.key ? styles.filterActive : styles.filterButton}
              onClick={() => setActiveFilter(filter.key)}
            >
              {filter.label}
            </button>
          ))}
        </div>

        {loading ? (
          <section className={styles.emptyState}>
            <div className={styles.emptyMark}>ORDER</div>
            <h2>正在加载订单</h2>
            <p>正在同步你的中标订单和履约状态。</p>
          </section>
        ) : error ? (
          <section className={styles.emptyState}>
            <div className={styles.emptyMark}>ORDER</div>
            <h2>{error}</h2>
            <p>网络或服务暂时不可用，请稍后重试。</p>
            <button type="button" onClick={() => void loadOrders(page)}>重试</button>
          </section>
        ) : visibleOrders.length === 0 ? (
          <section className={styles.emptyState}>
            <div className={styles.emptyMark}>ORDER</div>
            <h2>{emptyCopy.title}</h2>
            <p>{emptyCopy.description}</p>
            <Link to="/">去看直播竞拍</Link>
          </section>
        ) : (
          <div className={styles.orderList}>
            {visibleOrders.map((order) => {
              const meta = statusMeta[order.status];
              return (
                <article className={styles.orderCard} key={order.id}>
                  <div className={styles.receiptTop}>
                    <span className={styles.receiptStamp}>AUCTION ORDER</span>
                    <time dateTime={order.createdAt}>{formatDate(order.createdAt)}</time>
                  </div>

                  <div className={styles.cardMain}>
                    {order.productImage ? (
                      <div className={styles.productMedia}>
                        <img src={order.productImage} alt={order.productName} />
                      </div>
                    ) : (
                      <div className={styles.seal} aria-hidden="true">
                        <span>LOT</span>
                        <strong>{order.auctionId}</strong>
                      </div>
                    )}
                    <div className={styles.orderInfo}>
                      <div className={styles.orderTitleRow}>
                        <h2>{order.productName}</h2>
                      </div>
                      <p>{order.seller}</p>
                    </div>
                  </div>

                  <div className={styles.cardFooter}>
                    <div className={styles.priceTag} aria-label={`成交价 ${formatCurrency(order.finalPrice)}`}>
                      <span>成交价</span>
                      <strong className={styles.price}>{formatCurrency(order.finalPrice)}</strong>
                    </div>
                    <div className={styles.cardActions}>
                      <span className={styles.statusPill} data-tone={meta.tone}>
                        {meta.label}
                      </span>
                      <Link className={styles.detailLink} to={`/order/${order.id}`}>
                        查看订单
                      </Link>
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        )}

        {!loading && !error && hasNextPage ? (
          <button
            type="button"
            className={styles.loadMoreButton}
            onClick={() => void loadOrders(page + 1)}
          >
            加载更多
          </button>
        ) : null}
      </main>
    </section>
  );
}
