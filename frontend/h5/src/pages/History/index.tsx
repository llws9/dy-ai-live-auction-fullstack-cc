import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { orderApi } from '../../services/api';
import PageHeader from '@/components/shared/PageHeader';
import styles from './AuctionHistory.module.css';

interface HistoryRecord {
  auction_id?: number | string;
  id?: number | string;
  product_name?: string;
  product?: {
    name?: string;
    image?: string;
    images?: string[];
  };
  image?: string;
  product_image?: string;
  final_price?: number | string;
  my_highest_bid?: number | string;
  bid_count?: number | string;
  is_winner?: boolean | number;
  result?: string;
  status?: string | number;
  ended_at?: string;
  created_at?: string;
}

type FilterKey = 'all' | 'won' | 'lost';

function extractList<T>(response: any): T[] {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  return [];
}

function toNumber(value: number | string | undefined, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function formatCurrency(value: number | string | undefined) {
  return `¥${toNumber(value).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}`;
}

function formatTime(value?: string) {
  if (!value) return '时间待确认';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function getRecordId(record: HistoryRecord) {
  return record.auction_id ?? record.id ?? '';
}

function getProductName(record: HistoryRecord) {
  return record.product_name || record.product?.name || `竞拍场次 #${getRecordId(record)}`;
}

function getProductImage(record: HistoryRecord) {
  return record.product_image || record.image || record.product?.image || record.product?.images?.[0] || '';
}

function isWon(record: HistoryRecord) {
  if (typeof record.is_winner === 'boolean') return record.is_winner;
  if (typeof record.is_winner === 'number') return record.is_winner === 1;
  const normalized = String(record.result ?? record.status ?? '').toLowerCase();
  return ['success', 'won', 'winner', 'win'].includes(normalized);
}

function bidSummary(record: HistoryRecord) {
  if (record.my_highest_bid !== undefined) return `最高出价 ${formatCurrency(record.my_highest_bid)}`;
  return `出价 ${toNumber(record.bid_count)} 次`;
}

const filters: Array<{ key: FilterKey; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'won', label: '竞拍成功' },
  { key: 'lost', label: '未中标' },
];

function readFilter(value: string | null): FilterKey {
  if (value === 'won' || value === 'lost') return value;
  return 'all';
}

const AuctionHistoryPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [records, setRecords] = useState<HistoryRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeFilter, setActiveFilter] = useState<FilterKey>(() => readFilter(searchParams.get('filter')));

  useEffect(() => {
    let alive = true;

    async function loadHistory() {
      setLoading(true);
      setError(null);

      try {
        const response = await orderApi.history({ page: 1, page_size: 20 });
        if (!alive) return;
        setRecords(extractList<HistoryRecord>(response));
      } catch (err) {
        if (!alive) return;
        console.error('获取竞拍历史失败:', err);
        setRecords([]);
        setError('竞拍历史暂时无法加载');
      } finally {
        if (alive) setLoading(false);
      }
    }

    loadHistory();

    return () => {
      alive = false;
    };
  }, []);

  const stats = useMemo(() => {
    const won = records.filter(isWon);
    return {
      total: records.length,
      won: won.length,
      spent: won.reduce((sum, record) => sum + toNumber(record.final_price), 0),
    };
  }, [records]);

  const filteredRecords = useMemo(() => {
    if (activeFilter === 'won') return records.filter(isWon);
    if (activeFilter === 'lost') return records.filter((record) => !isWon(record));
    return records;
  }, [activeFilter, records]);

  return (
    <section className={styles.page}>
      <PageHeader
        classes={{
          header: styles.header,
          backButton: styles.backButton,
          eyebrow: styles.eyebrow,
        }}
        back={{ onClick: () => navigate(-1) }}
        eyebrow="Auction Ledger"
        title="我的竞拍记录"
        actions={<Link className={styles.homeLink} to="/">首页</Link>}
      />

      <div className={styles.summaryGrid}>
        <div className={styles.summaryCard}>
          <span>{stats.total}</span>
          <p>参与场次</p>
        </div>
        <div className={styles.summaryCard}>
          <span>{stats.won}</span>
          <p>竞拍成功</p>
        </div>
        <div className={styles.summaryCard}>
          <span>{formatCurrency(stats.spent)}</span>
          <p>成交总额</p>
        </div>
      </div>

      <div className={styles.filterBar} aria-label="竞拍记录筛选">
        {filters.map((filter) => (
          <button
            key={filter.key}
            className={activeFilter === filter.key ? styles.filterActive : styles.filterButton}
            type="button"
            onClick={() => setActiveFilter(filter.key)}
          >
            {filter.label}
          </button>
        ))}
      </div>

      <main className={styles.content} aria-live="polite">
        {loading ? (
          <div className={styles.statePage}>
            <div className={styles.spinner} />
            <p>加载竞拍记录...</p>
          </div>
        ) : error ? (
          <div className={styles.statePage}>
            <p>{error}</p>
            <button type="button" onClick={() => window.location.reload()}>重试</button>
          </div>
        ) : filteredRecords.length === 0 ? (
          <div className={styles.statePage}>
            <div className={styles.emptyIcon}>LOT</div>
            <p>暂无竞拍记录</p>
            <span>参与竞拍后记录将在此显示</span>
            <Link to="/">去参与竞拍</Link>
          </div>
        ) : (
          <div className={styles.recordList}>
            {filteredRecords.map((record) => {
              const recordId = getRecordId(record);
              const won = isWon(record);
              const image = getProductImage(record);

              return (
                <article className={styles.recordCard} key={String(recordId)}>
                  <div className={styles.cardMeta}>
                    <span>LOT {recordId}</span>
                    <time dateTime={record.ended_at || record.created_at}>{formatTime(record.ended_at || record.created_at)}</time>
                  </div>

                  <div className={styles.cardBody}>
                    <div className={styles.imageFrame}>
                      {image ? <img src={image} alt={getProductName(record)} /> : <span>藏品</span>}
                      <strong className={won ? styles.wonBadge : styles.lostBadge}>{won ? '竞拍成功' : '未中标'}</strong>
                    </div>

                    <div className={styles.recordInfo}>
                      <h2>{getProductName(record)}</h2>
                      <dl>
                        <div>
                          <dt>我的参与</dt>
                          <dd>{bidSummary(record)}</dd>
                        </div>
                        <div>
                          <dt>最终成交</dt>
                          <dd>{formatCurrency(record.final_price)}</dd>
                        </div>
                      </dl>
                    </div>
                  </div>

                  <div className={styles.cardActions}>
                    <Link className={won ? styles.primaryAction : styles.secondaryAction} to={won ? `/result?id=${recordId}` : `/detail?id=${recordId}`}>
                      {won ? '查看结果' : '查看详情'}
                    </Link>
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

export default AuctionHistoryPage;
