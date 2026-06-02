import React from 'react';
import styles from './Live.module.css';

interface BidDockProps {
  product?: { name?: string } | null;
  productImage?: string;
  roomName?: string;
  currentPrice: number;
  sheet: 'bid' | 'info' | null;
  isAuthenticated: boolean;
  onOpen: (sheet: 'bid' | 'info') => void;
  onClose: () => void;
  onRequireLogin: () => void;
  children?: React.ReactNode;
}

const formatMoney = (amount: number) => amount.toLocaleString('zh-CN', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});

const BidDock: React.FC<BidDockProps> = ({
  product,
  productImage,
  roomName,
  currentPrice,
  sheet,
  isAuthenticated,
  onOpen,
  onClose,
  onRequireLogin,
  children,
}) => {
  const handleBidClick = (event: React.MouseEvent) => {
    event.stopPropagation();
    if (!isAuthenticated) {
      onRequireLogin();
      return;
    }
    onOpen('bid');
  };

  return (
    <>
      <div className={styles.dock} role="group" onClick={() => onOpen('info')}>
        <div className={styles.dockProduct}>
          {productImage ? (
            <img src={productImage} alt={product?.name || '竞拍商品'} />
          ) : (
            <div className={styles.dockFallback}>品</div>
          )}
          <div className={styles.dockInfo}>
            <p>{product?.name || '竞拍商品'}</p>
            <span>{roomName}</span>
            <span className={styles.dockPrice}>当前最高价 ¥{formatMoney(currentPrice)}</span>
          </div>
        </div>
        <button className={styles.dockButton} type="button" onClick={handleBidClick}>
          出价
        </button>
      </div>

      {sheet !== null && (
        <>
          <div
            className={styles.mask}
            data-testid="bid-dock-mask"
            onClick={onClose}
          />
          <div
            className={`${styles.sheet} ${styles.sheetOpen}`}
            onClick={(event) => event.stopPropagation()}
          >
            <button
              className={styles.handle}
              type="button"
              aria-label="收起竞拍面板"
              onClick={onClose}
            />
            {children}
          </div>
        </>
      )}
    </>
  );
};

export default BidDock;
