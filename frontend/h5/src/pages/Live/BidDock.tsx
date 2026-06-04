import React from 'react';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
import styles from './Live.module.css';

interface BidDockProps {
  product?: { name?: string; description?: string } | null;
  productImage?: string;
  roomName?: string;
  currentPrice: number;
  sheet: 'bid' | 'info' | null;
  isAuthenticated: boolean;
  bidDisabled?: boolean;
  bidDisabledText?: string;
  skyLampActive?: boolean;
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
  bidDisabled = false,
  bidDisabledText = '已结束',
  skyLampActive = false,
  onOpen,
  onClose,
  onRequireLogin,
  children,
}) => {
  const productName = repairUtf8Mojibake(product?.name) || '竞拍商品';
  const productIntro = repairUtf8Mojibake(product?.description || roomName);

  const handleBidClick = (event: React.MouseEvent) => {
    event.stopPropagation();
    if (bidDisabled) return;
    if (!isAuthenticated) {
      onRequireLogin();
      return;
    }
    onOpen('bid');
  };

  return (
    <>
      <div
        className={`${styles.dock} ${skyLampActive ? styles.dockSkyLampActive : ''}`}
        role="group"
        data-testid="bid-dock"
        data-sky-lamp-active={skyLampActive ? 'true' : 'false'}
        onClick={() => onOpen('info')}
      >
        <div className={styles.dockProduct}>
          {productImage ? (
            <span className={styles.dockImageWrap}>
              <img src={productImage} alt={productName} />
              {skyLampActive && (
                <i className={`${styles.skyLampIcon} ${styles.dockSkyLampIcon}`} data-testid="dock-sky-lamp-icon" aria-hidden="true">
                  <span />
                </i>
              )}
            </span>
          ) : (
            <div className={styles.dockFallback}>品</div>
          )}
          <div className={styles.dockInfo}>
            <p>{productName}</p>
            <span>{productIntro}</span>
            <span className={styles.dockPrice}>当前最高价 ¥{formatMoney(currentPrice)}</span>
          </div>
        </div>
        <button className={styles.dockButton} type="button" onClick={handleBidClick} disabled={bidDisabled}>
          {bidDisabled ? bidDisabledText : '出价'}
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
