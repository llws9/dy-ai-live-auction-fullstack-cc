import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { userApi } from '../../services/api';
import styles from './Wallet.module.css';

interface BalanceData {
  available_amount?: number | string;
  available?: number | string;
  balance?: number | string;
  frozen_amount?: number | string;
}

type LedgerKind = 'negative' | 'positive' | 'frozen';

interface LedgerRow {
  id: string;
  title: string;
  detail: string;
  time: string;
  amount: string;
  kind: LedgerKind;
}

function toNumber(value: number | string | undefined, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function formatCurrency(value: number | string | undefined) {
  return `¥${toNumber(value).toLocaleString('zh-CN', { maximumFractionDigits: 2 })}`;
}

function pickAvailable(balance: BalanceData | null) {
  if (!balance) return 0;
  return balance.available_amount ?? balance.available ?? balance.balance ?? 0;
}

function pickFrozen(balance: BalanceData | null) {
  if (!balance) return 0;
  return balance.frozen_amount ?? 0;
}

function buildDemoLedger(balance: BalanceData | null): LedgerRow[] {
  const available = toNumber(pickAvailable(balance));
  const frozen = toNumber(pickFrozen(balance));
  const orderPayment = Math.min(Math.max(available * 0.1, 120), 1280);
  const frozenAmount = frozen > 0 ? frozen : 300;

  return [
    {
      id: 'order-payment',
      title: '订单支付',
      detail: '中标订单 · 等待用户确认支付',
      time: '今天 12:20',
      amount: `- ${formatCurrency(orderPayment)}`,
      kind: 'negative',
    },
    {
      id: 'auction-freeze',
      title: '竞拍冻结',
      detail: '出价保证金 · 竞拍结束后自动结算',
      time: '昨天 21:06',
      amount: `- ${formatCurrency(frozenAmount)}`,
      kind: 'frozen',
    },
    {
      id: 'freeze-release',
      title: '冻结释放',
      detail: '未中标保证金释放',
      time: '昨天 20:40',
      amount: `+ ${formatCurrency(frozenAmount)}`,
      kind: 'positive',
    },
  ];
}

const Wallet: React.FC = () => {
  const navigate = useNavigate();
  const [balance, setBalance] = useState<BalanceData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadBalance = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await userApi.getBalance();
      setBalance(response || {});
    } catch (err) {
      setError(err instanceof Error ? err.message : '钱包信息加载失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBalance();
  }, [loadBalance]);

  const available = pickAvailable(balance);
  const frozen = pickFrozen(balance);
  const ledgerRows = useMemo(() => buildDemoLedger(balance), [balance]);

  if (loading) {
    return (
      <main className={styles.statePage}>
        <div className={styles.spinner} />
        <p>钱包加载中...</p>
      </main>
    );
  }

  if (error) {
    return (
      <main className={styles.statePage}>
        <p className={styles.errorText}>钱包信息加载失败</p>
        <button type="button" className={styles.retryButton} onClick={loadBalance}>
          重试
        </button>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.navbar}>
        <button type="button" className={styles.backButton} aria-label="返回" onClick={() => navigate(-1)}>
          ‹
        </button>
        <h1>钱包</h1>
        <button type="button" className={styles.filterButton}>
          筛选
        </button>
      </header>

      <section className={styles.balanceCard} aria-label="钱包余额">
        <p className={styles.cardLabel}>Available</p>
        <strong className={styles.amount}>{formatCurrency(available)}</strong>
        <span>钱包余额 · 记录所有竞拍资金流</span>
      </section>

      <section className={styles.ledgerCard} aria-label="最近流水">
        <div className={styles.sectionHeader}>
          <div>
            <p className={styles.cardLabel}>Ledger</p>
            <h2>最近流水</h2>
          </div>
          <span className={styles.pill}>全部</span>
        </div>
        <p className={styles.demoNotice}>前端派生演示流水</p>
        <div className={styles.timeline}>
          {ledgerRows.map((row) => (
            <article key={row.id} className={styles.ledgerRow}>
              <div className={styles.rowText}>
                <strong>{row.title}</strong>
                <span>{row.detail} · {row.time}</span>
              </div>
              <b className={row.kind === 'positive' ? styles.positiveAmount : styles.negativeAmount}>
                {row.amount}
              </b>
            </article>
          ))}
        </div>
      </section>

      <section className={styles.statusCard} aria-label="资金状态">
        <div className={styles.sectionHeader}>
          <div>
            <p className={styles.cardLabel}>Status</p>
            <h2>资金状态</h2>
          </div>
        </div>
        <div className={styles.statusGrid}>
          <div>
            <strong>{formatCurrency(available)}</strong>
            <span>可用余额</span>
          </div>
          <div>
            <strong>{formatCurrency(frozen)}</strong>
            <span>冻结金额</span>
          </div>
        </div>
      </section>
    </main>
  );
};

export default Wallet;
