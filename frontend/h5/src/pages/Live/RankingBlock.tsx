import React from 'react';
import styles from './Live.module.css';

export interface RankingBlockItem {
  rank?: number;
  id?: number;
  user_id?: number;
  user_name?: string;
  amount: number;
  created_at?: string;
}

export interface RankingBlockProps {
  ranking: RankingBlockItem[];
  isAuthenticated: boolean;
  userId?: number;
  myRankIndex: number;
  myBidAmount?: number;
  formatMoney: (amount: number) => string;
}

const RankingBlock: React.FC<RankingBlockProps> = ({
  ranking,
  isAuthenticated,
  userId,
  myRankIndex,
  myBidAmount,
  formatMoney,
}) => {
  return (
    <section className={styles.rankingBlock}>
      <div className={styles.rankingGlow}></div>
      <h2 className={styles.rankingBlockTitle}>
        <span className={styles.rankingTrophy}>🏆</span> 出价排行
      </h2>
      <div className={styles.rankingList}>
        {[0, 1, 2].map((index) => {
          const item = ranking[index];
          const isFirst = index === 0;
          const isSecond = index === 1;
          const isEmpty = !item;
          const isMe = isAuthenticated && item?.user_id === userId;
          const displayName = item
            ? isMe
              ? isFirst
                ? '我自己 (当前领先)'
                : '我自己'
              : item.user_name
            : '虚位以待';

          return (
            <div
              className={`${styles.rankingItem} ${isFirst && !isEmpty ? styles.rankingItemFirst : ''} ${isEmpty ? styles.rankingItemEmpty : ''} ${isMe ? styles.rankingItemMe : ''}`}
              key={item ? `${item.user_id ?? item.id}-${index}` : `empty-${index}`}
            >
              <div className={styles.rankingItemLeft}>
                <span className={`${styles.rank} ${
                  isFirst && !isEmpty ? styles.rankFirst :
                  isSecond && !isEmpty ? styles.rankSecond :
                  !isEmpty ? styles.rankThird :
                  styles.rankEmpty
                }`}>
                  {index + 1}
                </span>
                <span className={`${styles.rankingName} ${isEmpty ? styles.rankingNameEmpty : ''} ${isMe ? styles.rankingNameMe : ''}`}>
                  {displayName}
                </span>
              </div>
              <strong className={`${styles.rankingAmount} ${isFirst && !isEmpty ? styles.rankingAmountFirst : ''} ${isEmpty ? styles.rankingAmountEmpty : ''} ${isMe && !isFirst ? styles.rankingAmountMe : ''}`}>
                {item ? `¥${formatMoney(item.amount)}` : '-'}
              </strong>
            </div>
          );
        })}
      </div>

      {/* 我的出价状态 - 方案A 悬浮轻量卡片 */}
      <div className={styles.myBidSection}>
        <div className={styles.myBidCard}>
          <div className={styles.myBidLeft}>
            {isAuthenticated ? (
              <>
                <div className={styles.myBidRankCircle}>
                  <span className={styles.myBidRank}>
                    {myRankIndex > -1 ? myRankIndex + 1 : '-'}
                  </span>
                </div>
                <span className={styles.myBidLabel}>当前我的排位</span>
              </>
            ) : (
              <span className={styles.myBidLabel}>请登录后查看出价状态</span>
            )}
          </div>
          <strong className={styles.myBidAmount}>
            {isAuthenticated ? `¥${formatMoney(myBidAmount || 0)}` : '-'}
          </strong>
        </div>
      </div>
    </section>
  );
};

export default RankingBlock;
