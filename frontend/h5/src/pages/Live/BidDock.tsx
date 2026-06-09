import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from 'react';
import { repairUtf8Mojibake } from '@/utils/textEncoding';
import { replaceBrokenImageWithFallback } from '@/utils/imageFallback';
import styles from './Live.module.css';

const SHEET_TRANSITION_MS = 350;

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
  myBidStatus?: 'leading' | 'outbid' | null;
  onOpen: (sheet: 'bid' | 'info') => void;
  onClose: () => void;
  onRequireLogin: () => void;
  topAddon?: ReactNode;
  children?: ReactNode;
}

const formatMoney = (amount: number) => amount.toLocaleString('zh-CN', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});

const BidDock = ({
  product,
  productImage,
  roomName,
  currentPrice,
  sheet,
  isAuthenticated,
  bidDisabled = false,
  bidDisabledText = '已结束',
  skyLampActive = false,
  myBidStatus = null,
  onOpen,
  onClose,
  onRequireLogin,
  topAddon,
  children,
}: BidDockProps) => {
  const productName = repairUtf8Mojibake(product?.name) || '竞拍商品';
  const productIntro = repairUtf8Mojibake(product?.description || roomName);
  const [renderedSheet, setRenderedSheet] = useState<'bid' | 'info' | null>(null);
  const [isSheetOpen, setIsSheetOpen] = useState(false);
  const previousSheetRef = useRef<'bid' | 'info' | null>(null);

  useEffect(() => {
    let frameId: number | null = null;
    let closeTimerId: number | null = null;
    const wasClosed = previousSheetRef.current === null;
    previousSheetRef.current = sheet;

    if (sheet !== null) {
      setRenderedSheet(sheet);
      if (wasClosed) {
        setIsSheetOpen(false);
        frameId = window.requestAnimationFrame(() => {
          setIsSheetOpen(true);
        });
      } else {
        setIsSheetOpen(true);
      }
    } else {
      setIsSheetOpen(false);
      closeTimerId = window.setTimeout(() => {
        setRenderedSheet(null);
      }, SHEET_TRANSITION_MS);
    }

    return () => {
      if (frameId !== null) {
        window.cancelAnimationFrame(frameId);
      }
      if (closeTimerId !== null) {
        window.clearTimeout(closeTimerId);
      }
    };
  }, [sheet]);

  const handleBidClick = (event: MouseEvent) => {
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
        {myBidStatus && (
          <div className={`${styles.myBidCapsule} ${myBidStatus === 'leading' ? styles.myBidCapsuleLeading : styles.myBidCapsuleOutbid}`}>
            {myBidStatus === 'leading' ? '当前领先' : '被超越'}
          </div>
        )}
        <div className={styles.dockProduct}>
          {productImage ? (
            <span className={styles.dockImageWrap}>
              <img src={productImage} alt={productName} onError={replaceBrokenImageWithFallback} />
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

      {renderedSheet !== null && (
        <>
          <div
            className={styles.mask}
            data-testid="bid-dock-mask"
            onClick={onClose}
          />
          {topAddon && (
            <div className={styles.sheetDockAddon} data-testid="bid-dock-top-addon">
              {topAddon}
            </div>
          )}
          <div
            className={`${styles.sheet} ${isSheetOpen ? styles.sheetOpen : ''}`}
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
