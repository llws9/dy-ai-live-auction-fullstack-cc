// pages/Live/index.tsx
// 直播间风格竞拍页面 - 一个直播间多个商品、底部浮窗、半屏出价面板

import React, { useState, useEffect, useRef } from 'react';

interface Product {
  id: number;
  name: string;
  image: string;
  status: 'ongoing' | 'pending' | 'ended';
  hasBids: boolean;
  currentPrice: number;
  startPrice: number;
  finalPrice?: number;
  myBid?: number;
  increment: number;
  endTime: string;
  winner?: string;
  bidCount: number;
}

interface BidRecord {
  id: number;
  userName: string;
  amount: number;
  time: string;
}

interface LiveRoom {
  id: number;
  name: string;
  anchor: string;
  avatar: string;
  viewerCount: number;
  products: Product[];
}

const LiveAuctionPage: React.FC = () => {
  const [bottomSheetExpanded, setBottomSheetExpanded] = useState(false);
  const [bidSheetOpen, setBidSheetOpen] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [liveRoom, setLiveRoom] = useState<LiveRoom | null>(null);
  const [countdown, setCountdown] = useState<Record<number, number>>({});
  const [bidAmount, setBidAmount] = useState('');
  const [bidRecords, setBidRecords] = useState<BidRecord[]>([]);
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const touchStartY = useRef(0);

  useEffect(() => {
    // 模拟直播间数据 - 一个直播间有多个商品
    const mockLiveRoom: LiveRoom = {
      id: 1,
      name: '珍品拍卖专场',
      anchor: '拍卖师小王',
      avatar: 'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=100',
      viewerCount: 12800,
      products: [
        {
          id: 1,
          name: '稀有珠宝 - 限量版钻石项链',
          image: 'https://images.unsplash.com/photo-1515562141207-7a88fb7ce338?w=400',
          status: 'ongoing',
          hasBids: true,
          currentPrice: 5200,
          startPrice: 3000,
          increment: 100,
          endTime: new Date(Date.now() + 1800000).toISOString(),
          myBid: 5000,
          bidCount: 12,
        },
        {
          id: 2,
          name: '签名版限量球鞋',
          image: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400',
          status: 'ongoing',
          hasBids: true,
          currentPrice: 850,
          startPrice: 500,
          increment: 50,
          endTime: new Date(Date.now() + 900000).toISOString(),
          bidCount: 8,
        },
        {
          id: 3,
          name: '古董怀表收藏品',
          image: 'https://images.unsplash.com/photo-1509048191080-d2984bad6ae5?w=400',
          status: 'ended',
          hasBids: true,
          currentPrice: 3500,
          startPrice: 200,
          finalPrice: 3500,
          increment: 50,
          endTime: new Date(Date.now() - 3600000).toISOString(),
          winner: '用户A',
          bidCount: 25,
        },
        {
          id: 4,
          name: '艺术画作原稿',
          image: 'https://images.unsplash.com/photo-1579783902614-a3fb3927b6a5?w=400',
          status: 'pending',
          hasBids: false,
          currentPrice: 0,
          startPrice: 1000,
          increment: 100,
          endTime: new Date(Date.now() + 86400000).toISOString(),
          bidCount: 0,
        },
        {
          id: 5,
          name: '限量版手表',
          image: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=400',
          status: 'ongoing',
          hasBids: false,
          currentPrice: 0,
          startPrice: 2000,
          increment: 100,
          endTime: new Date(Date.now() + 2700000).toISOString(),
          bidCount: 0,
        },
      ],
    };

    setLiveRoom(mockLiveRoom);

    // 模拟出价记录
    setBidRecords([
      { id: 1, userName: '用户A', amount: 5200, time: '刚刚' },
      { id: 2, userName: '用户B', amount: 5100, time: '1分钟前' },
      { id: 3, userName: '用户C', amount: 5000, time: '2分钟前' },
    ]);
  }, []);

  // 倒计时 - 支持多个商品同时竞拍
  useEffect(() => {
    if (!liveRoom) return;

    const interval = setInterval(() => {
      const newCountdown: Record<number, number> = {};
      liveRoom.products.forEach((product) => {
        if (product.status === 'ongoing') {
          const diff = new Date(product.endTime).getTime() - Date.now();
          newCountdown[product.id] = Math.max(0, diff);
        }
      });
      setCountdown(newCountdown);
    }, 100);

    return () => clearInterval(interval);
  }, [liveRoom]);

  const formatCountdown = (ms: number) => {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    const msDisplay = Math.floor((ms % 1000) / 10);
    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}.${String(msDisplay).padStart(2, '0')}`;
  };

  const getPriceLabel = (product: Product) => {
    if (product.status === 'ended') {
      return product.hasBids ? '落槌价' : '起拍价';
    }
    if (product.status === 'pending') return '起拍价';
    return product.hasBids ? '当前最高价' : '起拍价';
  };

  const getPrice = (product: Product) => {
    if (product.status === 'ended' && product.hasBids) {
      return product.finalPrice || product.currentPrice;
    }
    if (!product.hasBids) return product.startPrice;
    return product.currentPrice;
  };

  const getButtonText = (product: Product) => {
    if (product.status === 'ended') return '已结束';
    if (product.status === 'pending') return '商品详情';
    return '立即出价';
  };

  // 获取正在竞拍的商品数量
  const getOngoingCount = () => {
    return liveRoom?.products.filter(p => p.status === 'ongoing').length || 0;
  };

  // 获取最热门的竞拍商品（用于收起状态预览）
  const getHotProduct = () => {
    if (!liveRoom) return null;
    // 优先显示竞拍中且出价最多的
    const ongoing = liveRoom.products.filter(p => p.status === 'ongoing');
    if (ongoing.length > 0) {
      return ongoing.sort((a, b) => b.bidCount - a.bidCount)[0];
    }
    // 其次显示即将开始的
    const pending = liveRoom.products.filter(p => p.status === 'pending');
    if (pending.length > 0) return pending[0];
    // 最后显示已结束的
    return liveRoom.products[0];
  };

  const handleProductClick = (product: Product) => {
    setSelectedProduct(product);
    setBidSheetOpen(true);
  };

  const handleBid = () => {
    if (!selectedProduct || !bidAmount) return;

    const amount = parseFloat(bidAmount);
    const minBid = (selectedProduct.hasBids ? selectedProduct.currentPrice : selectedProduct.startPrice) + selectedProduct.increment;

    if (amount < minBid) {
      setToast({ message: `最低出价 ¥${minBid}`, type: 'error' });
      setTimeout(() => setToast(null), 2000);
      return;
    }

    // 模拟出价成功
    setToast({ message: '出价成功！', type: 'success' });
    setTimeout(() => setToast(null), 2000);
    setBidAmount('');
    setBidSheetOpen(false);
  };

  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartY.current = e.touches[0].clientY;
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    const diff = touchStartY.current - e.changedTouches[0].clientY;
    if (diff > 50) {
      setBottomSheetExpanded(true);
    } else if (diff < -50) {
      setBottomSheetExpanded(false);
    }
  };

  // 点击展开浮窗（支持桌面浏览器）
  const handleSheetClick = () => {
    if (!bottomSheetExpanded) {
      setBottomSheetExpanded(true);
    }
  };

  // 点击视频区域收起浮窗
  const handleVideoClick = () => {
    if (bottomSheetExpanded) {
      setBottomSheetExpanded(false);
    }
    if (bidSheetOpen) {
      setBidSheetOpen(false);
    }
  };

  const hotProduct = getHotProduct();
  const ongoingCount = getOngoingCount();

  return (
    <div style={styles.container}>
      {/* 直播间背景 */}
      <div style={styles.liveContainer} onClick={handleVideoClick}>
        <video
          style={styles.liveVideo}
          autoPlay
          loop
          muted
          playsInline
          poster="https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?w=800"
        >
          <source src="https://www.w3schools.com/html/mov_bbb.mp4" type="video/mp4" />
        </video>

        {/* 直播间头部 */}
        <div style={styles.liveHeader} onClick={(e) => e.stopPropagation()}>
          <img
            src={liveRoom?.avatar}
            alt="主播"
            style={styles.liveAvatar}
          />
          <div style={styles.liveInfo}>
            <div style={styles.liveTitle}>{liveRoom?.name || '竞拍直播间'}</div>
            <div style={styles.liveViewer}>🔥 {(liveRoom?.viewerCount || 0).toLocaleString()}人在看</div>
          </div>
          <div style={styles.liveBadge}>直播中</div>
        </div>
      </div>

      {/* 底部浮窗 */}
      <div
        style={{
          ...styles.bottomSheet,
          height: bottomSheetExpanded ? '80vh' : '100px',
        }}
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        onClick={handleSheetClick}
      >
        <div style={styles.sheetHandle}></div>

        {/* 收起状态 - 显示竞拍商品数量和热门商品预览 */}
        {!bottomSheetExpanded && (
          <div style={styles.sheetPreview}>
            <div style={styles.previewInfo}>
              <div style={styles.previewName}>
                🎯 {liveRoom?.products.length || 0}件竞拍商品
                {ongoingCount > 0 && (
                  <span style={styles.ongoingBadge}>{ongoingCount}件竞拍中</span>
                )}
              </div>
              {hotProduct && (
                <div style={styles.previewPrice}>
                  <span style={styles.previewPriceLabel}>{getPriceLabel(hotProduct)}</span>
                  ¥{getPrice(hotProduct).toLocaleString()}
                </div>
              )}
            </div>
            {hotProduct && hotProduct.status === 'ongoing' && (
              <div style={styles.previewCountdown}>
                <div style={styles.countdownTime}>
                  {formatCountdown(countdown[hotProduct.id] || 0)}
                </div>
                <div style={styles.countdownLabel}>剩余时间</div>
              </div>
            )}
          </div>
        )}

        {/* 展开状态 - 商品列表 */}
        {bottomSheetExpanded && liveRoom && (
          <div style={styles.sheetContent}>
            <h3 style={styles.sheetTitle}>
              竞拍商品
              <span style={styles.sheetCount}>{liveRoom.products.length}件</span>
            </h3>
            {liveRoom.products.map((product) => (
              <div key={product.id} style={styles.productCard} onClick={() => handleProductClick(product)}>
                <div style={styles.productImageWrapper}>
                  <img src={product.image} alt={product.name} style={styles.productImage} />
                  {product.status === 'ongoing' && (
                    <div style={styles.productCountdown}>
                      {formatCountdown(countdown[product.id] || 0)}
                    </div>
                  )}
                </div>
                <div style={styles.productInfo}>
                  <div style={styles.productName}>{product.name}</div>
                  <div style={styles.productStatus}>
                    <span style={{
                      ...styles.statusDot,
                      background: product.status === 'ongoing' ? 'var(--neon-green)' :
                        product.status === 'pending' ? 'var(--neon-blue)' : 'var(--text-muted)'
                    }}></span>
                    <span style={styles.statusText}>
                      {product.status === 'ongoing' ? '竞拍中' :
                        product.status === 'pending' ? '即将开始' : '已结束'}
                    </span>
                    {product.bidCount > 0 && (
                      <span style={styles.bidCountTag}>{product.bidCount}次出价</span>
                    )}
                  </div>
                  <div style={styles.productPrice}>
                    <span style={styles.priceLabel}>{getPriceLabel(product)}</span>
                    <span style={styles.priceValue}>¥{getPrice(product).toLocaleString()}</span>
                  </div>
                  <div style={styles.productAction}>
                    <button style={{
                      ...styles.bidBtn,
                      background: product.status === 'ended' ? 'rgba(255,255,255,0.05)' :
                        product.status === 'ongoing' ? 'var(--gradient-gold)' : 'rgba(255,255,255,0.1)',
                      color: product.status === 'ended' ? 'var(--text-muted)' :
                        product.status === 'ongoing' ? '#000' : 'var(--text-secondary)',
                    }}>
                      {getButtonText(product)}
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 半屏出价面板 */}
      <div style={{
        ...styles.bidSheet,
        bottom: bidSheetOpen ? '0' : '-70vh',
      }}>
        <div style={styles.bidSheetHeader}>
          <h3 style={styles.bidSheetTitle}>
            {selectedProduct?.status === 'ended' ? '成交详情' : '参与竞拍'}
          </h3>
          <button style={styles.bidSheetClose} onClick={() => setBidSheetOpen(false)}>✕</button>
        </div>

        <div style={styles.bidSheetBody}>
          {selectedProduct && (
            <>
              {/* 倒计时 */}
              {selectedProduct.status === 'ongoing' && (
                <div style={styles.countdownSection}>
                  <div style={{
                    ...styles.countdownDisplay,
                    color: (countdown[selectedProduct.id] || 0) < 30000 ? 'var(--neon-red)' : 'var(--neon-blue)',
                  }}>
                    {formatCountdown(countdown[selectedProduct.id] || 0)}
                  </div>
                </div>
              )}

              {/* 商品信息 */}
              <div style={styles.bidProduct}>
                <img src={selectedProduct.image} alt={selectedProduct.name} style={styles.bidProductImage} />
                <div style={styles.bidProductDetails}>
                  <div style={styles.bidProductName}>{selectedProduct.name}</div>
                  <div style={styles.bidPriceRow}>
                    <div style={styles.bidPriceItem}>
                      <div style={styles.bidPriceLabel}>当前价</div>
                      <div style={styles.bidPriceValue}>¥{getPrice(selectedProduct).toLocaleString()}</div>
                    </div>
                    <div style={styles.bidPriceItem}>
                      <div style={styles.bidPriceLabel}>我的出价</div>
                      <div style={{
                        ...styles.bidPriceValue,
                        color: selectedProduct.myBid ? 'var(--neon-green)' : 'var(--text-muted)',
                        fontSize: selectedProduct.myBid ? '20px' : '14px',
                      }}>
                        {selectedProduct.myBid ? `¥${selectedProduct.myBid.toLocaleString()}` : '暂未出价'}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* 出价排名 */}
              <div style={styles.rankingSection}>
                <div style={styles.rankingTitle}>出价排名 TOP 3</div>
                <div style={styles.rankingList}>
                  {bidRecords.slice(0, 3).map((record, index) => (
                    <div key={record.id} style={styles.rankingItem}>
                      <span style={{
                        ...styles.rankingPosition,
                        background: index === 0 ? 'linear-gradient(135deg, #ffd700 0%, #ff8c00 100%)' :
                          index === 1 ? 'linear-gradient(135deg, #c0c0c0 0%, #a0a0a0 100%)' :
                          'linear-gradient(135deg, #cd7f32 0%, #8b4513 100%)',
                        color: index === 2 ? 'white' : '#000',
                      }}>
                        {index + 1}
                      </span>
                      <span style={styles.rankingName}>{record.userName}</span>
                      <span style={styles.rankingAmount}>¥{record.amount.toLocaleString()}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* 出价输入 */}
              {selectedProduct.status === 'ongoing' && (
                <div style={styles.bidInputSection}>
                  <div style={styles.bidInputLabel}>输入出价金额</div>
                  <div style={styles.bidInputWrapper}>
                    <input
                      type="number"
                      value={bidAmount}
                      onChange={(e) => setBidAmount(e.target.value)}
                      placeholder={`最低 ¥${(selectedProduct.hasBids ? selectedProduct.currentPrice : selectedProduct.startPrice) + selectedProduct.increment}`}
                      style={styles.bidInput}
                    />
                  </div>
                  <div style={styles.bidIncrementHint}>
                    每次加价至少 ¥{selectedProduct.increment}
                  </div>
                </div>
              )}

              {/* 出价按钮 */}
              {selectedProduct.status === 'ongoing' && (
                <button style={styles.bidSubmitBtn} onClick={handleBid}>
                  立即竞拍
                </button>
              )}

              {/* 已结束状态 */}
              {selectedProduct.status === 'ended' && (
                <div style={{ textAlign: 'center', padding: '20px 0' }}>
                  {selectedProduct.winner ? (
                    <>
                      <div style={{ fontSize: '16px', color: 'var(--neon-gold)', marginBottom: '8px' }}>
                        🏆 恭喜 {selectedProduct.winner} 以 ¥{selectedProduct.finalPrice?.toLocaleString()} 中标！
                      </div>
                    </>
                  ) : (
                    <div style={{ color: 'var(--text-muted)' }}>本次竞拍未成交</div>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* 遮罩层 */}
      <div style={{
        ...styles.overlay,
        opacity: bidSheetOpen ? 1 : 0,
        visibility: bidSheetOpen ? 'visible' : 'hidden',
      }} onClick={() => setBidSheetOpen(false)}></div>

      {/* Toast 提示 */}
      {toast && (
        <div style={{
          ...styles.toast,
          border: toast.type === 'success' ? '1px solid var(--neon-green)' : '1px solid var(--neon-red)',
        }}>
          {toast.message}
        </div>
      )}
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    height: '100vh',
    width: '100vw',
    overflow: 'hidden',
    position: 'relative',
    background: '#000',
  },
  liveContainer: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
  },
  liveVideo: {
    width: '100%',
    height: '100%',
    objectFit: 'cover',
  },
  liveHeader: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    padding: '50px 16px 16px',
    background: 'linear-gradient(to bottom, rgba(0,0,0,0.6) 0%, transparent 100%)',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    zIndex: 10,
  },
  liveAvatar: {
    width: '40px',
    height: '40px',
    borderRadius: '50%',
    border: '2px solid var(--neon-blue)',
  },
  liveInfo: {
    flex: 1,
  },
  liveTitle: {
    fontSize: '14px',
    fontWeight: 600,
    color: 'white',
  },
  liveViewer: {
    fontSize: '12px',
    color: 'rgba(255,255,255,0.7)',
  },
  liveBadge: {
    padding: '4px 10px',
    background: '#ff4d4f',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: 600,
    color: 'white',
  },
  bottomSheet: {
    position: 'fixed',
    left: 0,
    right: 0,
    bottom: 0,
    background: 'rgba(30, 30, 45, 0.98)',
    borderRadius: '20px 20px 0 0',
    zIndex: 100,
    transition: 'height 0.3s ease',
    boxShadow: '0 -10px 40px rgba(0, 0, 0, 0.5)',
  },
  sheetHandle: {
    width: '40px',
    height: '4px',
    background: 'rgba(255, 255, 255, 0.3)',
    borderRadius: '2px',
    margin: '12px auto',
  },
  sheetPreview: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    padding: '0 16px 16px',
  },
  previewInfo: {
    flex: 1,
  },
  previewName: {
    fontSize: '14px',
    fontWeight: 600,
    marginBottom: '4px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  ongoingBadge: {
    padding: '2px 8px',
    background: 'rgba(0, 255, 136, 0.2)',
    borderRadius: '10px',
    fontSize: '11px',
    color: 'var(--neon-green)',
    fontWeight: 500,
  },
  previewPrice: {
    fontSize: '16px',
    fontWeight: 700,
    color: '#ffd700',
    display: 'flex',
    alignItems: 'baseline',
    gap: '6px',
  },
  previewPriceLabel: {
    fontSize: '11px',
    color: 'rgba(255,255,255,0.5)',
    fontWeight: 400,
  },
  previewCountdown: {
    textAlign: 'right',
  },
  countdownTime: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '18px',
    fontWeight: 700,
    color: '#00d4ff',
  },
  countdownLabel: {
    fontSize: '10px',
    color: 'rgba(255,255,255,0.5)',
  },
  sheetContent: {
    padding: '0 16px 100px',
    overflowY: 'auto',
    height: 'calc(80vh - 30px)',
  },
  sheetTitle: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '18px',
    fontWeight: 700,
    marginBottom: '16px',
    position: 'sticky',
    top: 0,
    background: 'rgba(30, 30, 45, 0.98)',
    padding: '10px 0',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  sheetCount: {
    fontSize: '13px',
    fontWeight: 500,
    color: 'rgba(255,255,255,0.5)',
  },
  productCard: {
    display: 'flex',
    gap: '12px',
    padding: '12px',
    background: 'rgba(40, 40, 60, 1)',
    borderRadius: '16px',
    marginBottom: '12px',
  },
  productImageWrapper: {
    position: 'relative',
    flexShrink: 0,
  },
  productImage: {
    width: '80px',
    height: '80px',
    borderRadius: '12px',
    objectFit: 'cover',
  },
  productCountdown: {
    position: 'absolute',
    bottom: '4px',
    left: '4px',
    right: '4px',
    padding: '2px 6px',
    background: 'rgba(0,0,0,0.7)',
    borderRadius: '6px',
    fontSize: '10px',
    fontWeight: 600,
    color: 'var(--neon-blue)',
    textAlign: 'center',
  },
  productInfo: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
  },
  productName: {
    fontSize: '14px',
    fontWeight: 600,
    lineHeight: 1.3,
    display: '-webkit-box',
    WebkitLineClamp: 2,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden',
  },
  productStatus: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    marginTop: '4px',
    flexWrap: 'wrap' as const,
  },
  statusDot: {
    width: '6px',
    height: '6px',
    borderRadius: '50%',
  },
  statusText: {
    fontSize: '12px',
    color: 'rgba(255,255,255,0.7)',
  },
  bidCountTag: {
    padding: '2px 6px',
    background: 'rgba(255, 215, 0, 0.15)',
    borderRadius: '8px',
    fontSize: '10px',
    color: 'var(--neon-gold)',
  },
  productPrice: {
    display: 'flex',
    alignItems: 'baseline',
    gap: '4px',
  },
  priceLabel: {
    fontSize: '11px',
    color: 'rgba(255,255,255,0.5)',
  },
  priceValue: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '18px',
    fontWeight: 700,
    color: '#ffd700',
  },
  productAction: {
    marginTop: '6px',
  },
  bidBtn: {
    padding: '6px 16px',
    border: 'none',
    borderRadius: '8px',
    fontSize: '12px',
    fontWeight: 600,
    cursor: 'pointer',
  },
  // 半屏面板
  bidSheet: {
    position: 'fixed',
    left: 0,
    right: 0,
    bottom: '-70vh',
    height: '70vh',
    background: 'rgba(30, 30, 45, 0.98)',
    borderRadius: '28px 28px 0 0',
    zIndex: 200,
    transition: 'bottom 0.3s ease',
    boxShadow: '0 -10px 60px rgba(0, 0, 0, 0.8)',
  },
  bidSheetHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '16px 20px',
    borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
  },
  bidSheetTitle: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '18px',
    fontWeight: 700,
    margin: 0,
  },
  bidSheetClose: {
    width: '32px',
    height: '32px',
    background: 'rgba(255, 255, 255, 0.1)',
    border: 'none',
    borderRadius: '50%',
    color: 'rgba(255,255,255,0.7)',
    fontSize: '18px',
    cursor: 'pointer',
  },
  bidSheetBody: {
    padding: '20px',
    overflowY: 'auto',
    height: 'calc(70vh - 60px)',
  },
  countdownSection: {
    textAlign: 'center',
    marginBottom: '20px',
  },
  countdownDisplay: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '36px',
    fontWeight: 800,
    textShadow: '0 0 20px currentColor',
  },
  bidProduct: {
    display: 'flex',
    gap: '16px',
    padding: '16px',
    background: 'rgba(40, 40, 60, 1)',
    borderRadius: '16px',
    marginBottom: '20px',
  },
  bidProductImage: {
    width: '100px',
    height: '100px',
    borderRadius: '12px',
    objectFit: 'cover',
  },
  bidProductDetails: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
  },
  bidProductName: {
    fontSize: '15px',
    fontWeight: 600,
    marginBottom: '8px',
  },
  bidPriceRow: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  bidPriceItem: {
    textAlign: 'center',
  },
  bidPriceLabel: {
    fontSize: '11px',
    color: 'rgba(255,255,255,0.5)',
    marginBottom: '2px',
  },
  bidPriceValue: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '20px',
    fontWeight: 700,
    color: '#ffd700',
  },
  rankingSection: {
    marginBottom: '20px',
  },
  rankingTitle: {
    fontSize: '13px',
    color: 'rgba(255,255,255,0.5)',
    marginBottom: '12px',
  },
  rankingList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  rankingItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    padding: '10px 14px',
    background: 'rgba(40, 40, 60, 1)',
    borderRadius: '10px',
  },
  rankingPosition: {
    width: '24px',
    height: '24px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: '50%',
    fontSize: '12px',
    fontWeight: 700,
  },
  rankingName: {
    flex: 1,
    fontSize: '14px',
  },
  rankingAmount: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '16px',
    fontWeight: 700,
    color: '#ffd700',
  },
  bidInputSection: {
    background: 'rgba(40, 40, 60, 1)',
    borderRadius: '16px',
    padding: '16px',
    marginBottom: '16px',
  },
  bidInputLabel: {
    fontSize: '13px',
    color: 'rgba(255,255,255,0.5)',
    marginBottom: '8px',
  },
  bidInputWrapper: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  bidInput: {
    flex: 1,
    padding: '14px 16px',
    background: 'rgba(20, 20, 30, 0.8)',
    border: '2px solid rgba(255, 255, 255, 0.1)',
    borderRadius: '12px',
    fontSize: '18px',
    fontWeight: 600,
    color: 'white',
    textAlign: 'center',
    outline: 'none',
  },
  bidIncrementHint: {
    fontSize: '12px',
    color: 'rgba(255,255,255,0.5)',
    textAlign: 'center',
    marginTop: '8px',
  },
  bidSubmitBtn: {
    width: '100%',
    padding: '16px',
    background: 'linear-gradient(135deg, #ffd700 0%, #ff8c00 100%)',
    border: 'none',
    borderRadius: '16px',
    fontSize: '18px',
    fontWeight: 700,
    color: '#000',
    cursor: 'pointer',
  },
  overlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'rgba(0, 0, 0, 0.7)',
    zIndex: 150,
    transition: 'opacity 0.3s, visibility 0.3s',
  },
  toast: {
    position: 'fixed',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    padding: '16px 32px',
    background: 'rgba(0, 0, 0, 0.9)',
    borderRadius: '16px',
    color: 'white',
    fontSize: '14px',
    zIndex: 1000,
  },
};

export default LiveAuctionPage;
