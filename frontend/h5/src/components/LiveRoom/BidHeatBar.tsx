import type { BidHeatLevel } from '@/hooks/useBidHeat';
import styles from './BidHeatBar.module.css';

interface BidHeatBarProps {
  level: BidHeatLevel;
  bidderCount: number;
  viewerCount: number;
}

const HEAT_VIEW = {
  calm: {
    label: '战况冷静',
    value: 24,
  },
  warming: {
    label: '战况升温',
    value: 62,
  },
  blazing: {
    label: '战况白热',
    value: 100,
  },
} satisfies Record<BidHeatLevel, { label: string; value: number }>;

export const BidHeatBar = ({ level, bidderCount, viewerCount }: BidHeatBarProps) => {
  const view = HEAT_VIEW[level];

  return (
    <section
      className={`${styles.bidHeatBar} ${styles[level]}`}
      data-testid="bid-heat-bar"
      aria-label="竞拍战况热度"
    >
      <div className={styles.header}>
        <span className={styles.label}>{view.label}</span>
        <span className={styles.pulseDot} aria-hidden="true" />
      </div>
      <div
        className={styles.meter}
        role="meter"
        aria-label="战况热度"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={view.value}
      >
        <span className={styles.fill} style={{ transform: `scaleX(${view.value / 100})` }} />
        {level === 'blazing' && <span className={styles.shimmer} aria-hidden="true" />}
      </div>
      <div className={styles.stats}>
        <span>已有 {bidderCount} 人出价</span>
        <span>{viewerCount} 人围观</span>
      </div>
    </section>
  );
};
