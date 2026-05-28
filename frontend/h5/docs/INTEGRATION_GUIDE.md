# 出价功能集成指南

本文档说明如何在直播间页面集成出价功能。

## 集成步骤

### 1. 导入组件

在 `frontend/h5/src/pages/Live/index.tsx` 顶部添加：

```typescript
import { useEffect, useState } from 'react';
import { useAuth } from '../../store/authContext';
import BidInput from '../../components/BidInput';
import RankingList from '../../components/RankingList';
import WebSocketService from '../../services/websocket';
import { bidApi } from '../../services/api';
```

### 2. 添加状态管理

在组件内部添加：

```typescript
const { isAuthenticated, user } = useAuth();
const [ws, setWs] = useState<WebSocketService | null>(null);
const [rankings, setRankings] = useState<any[]>([]);
const [selectedAuctionId, setSelectedAuctionId] = useState<number | null>(null);
```

### 3. WebSocket连接

在 `useEffect` 中建立连接：

```typescript
useEffect(() => {
  if (selectedAuctionId) {
    const wsService = new WebSocketService(selectedAuctionId, localStorage.getItem('token') || undefined);

    wsService.on('rank_update', (data: any) => {
      setRankings(data.rankings || []);
    });

    wsService.on('bid_placed', (data: any) => {
      // 更新当前价格
      // 显示通知
      console.log('新的出价:', data);
    });

    wsService.connect();

    setWs(wsService);

    return () => {
      wsService.disconnect();
    };
  }
}, [selectedAuctionId]);
```

### 4. 在出价面板中集成组件

找到出价面板的UI部分，替换为：

```typescript
{bidSheetOpen && selectedProduct && (
  <div style={styles.bidSheet}>
    {/* 现有的商品信息... */}

    {/* 添加出价输入组件 */}
    <BidInput
      auctionId={selectedProduct.id}
      currentPrice={selectedProduct.currentPrice}
      minIncrement={selectedProduct.increment}
      maxPrice={undefined} // 如果有封顶价，传入maxPrice
      onBidSuccess={(result) => {
        // 出价成功后的处理
        console.log('出价成功:', result);
        setToast({ message: '出价成功！', type: 'success' });
      }}
      onBidError={(error) => {
        // 出价失败的处理
        console.error('出价失败:', error);
        setToast({ message: error.message, type: 'error' });
      }}
    />

    {/* 添加排名列表 */}
    <RankingList
      rankings={rankings}
      currentUserId={user?.id}
      loading={false}
    />
  </div>
)}
```

### 5. 处理用户点击出价按钮

```typescript
const handleBidClick = (product: Product) => {
  if (!isAuthenticated) {
    // 未登录，跳转登录页
    window.location.href = '/login';
    return;
  }

  setSelectedProduct(product);
  setSelectedAuctionId(product.id);
  setBidSheetOpen(true);
};
```

### 6. 获取初始排名数据

```typescript
useEffect(() => {
  if (selectedAuctionId) {
    // 获取初始排名
    bidApi.getRanking(selectedAuctionId)
      .then((data) => {
        setRankings(data.items || []);
      })
      .catch((error) => {
        console.error('获取排名失败:', error);
      });
  }
}, [selectedAuctionId]);
```

## 完整示例

```typescript
// 在Live页面组件中
import React, { useState, useEffect } from 'react';
import { useAuth } from '../../store/authContext';
import BidInput from '../../components/BidInput';
import RankingList from '../../components/RankingList';
import WebSocketService from '../../services/websocket';
import { bidApi } from '../../services/api';

const LiveAuctionPage: React.FC = () => {
  const { isAuthenticated, user } = useAuth();
  const [ws, setWs] = useState<WebSocketService | null>(null);
  const [rankings, setRankings] = useState<any[]>([]);
  const [selectedAuctionId, setSelectedAuctionId] = useState<number | null>(null);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [bidSheetOpen, setBidSheetOpen] = useState(false);

  // WebSocket连接
  useEffect(() => {
    if (selectedAuctionId) {
      const wsService = new WebSocketService(
        selectedAuctionId,
        localStorage.getItem('token') || undefined
      );

      // 订阅排名更新
      wsService.on('rank_update', (data: any) => {
        setRankings(data.rankings || []);
      });

      // 订阅出价事件
      wsService.on('bid_placed', (data: any) => {
        console.log('新的出价:', data);
        // 可以在这里更新商品列表的价格
      });

      wsService.connect();
      setWs(wsService);

      // 获取初始排名
      bidApi.getRanking(selectedAuctionId)
        .then((data) => setRankings(data.items || []))
        .catch(console.error);

      return () => {
        wsService.disconnect();
      };
    }
  }, [selectedAuctionId]);

  // 处理出价按钮点击
  const handleBidClick = (product: Product) => {
    if (!isAuthenticated) {
      window.location.href = '/login';
      return;
    }

    setSelectedProduct(product);
    setSelectedAuctionId(product.id);
    setBidSheetOpen(true);
  };

  // 出价成功回调
  const handleBidSuccess = (result: any) => {
    console.log('出价成功:', result);
    // 刷新排名
    if (selectedAuctionId) {
      bidApi.getRanking(selectedAuctionId)
        .then((data) => setRankings(data.items || []))
        .catch(console.error);
    }
  };

  return (
    <div>
      {/* 商品列表 */}
      {liveRoom?.products.map((product) => (
        <div key={product.id}>
          {/* 商品信息 */}
          <button onClick={() => handleBidClick(product)}>
            {getButtonText(product)}
          </button>
        </div>
      ))}

      {/* 出价面板 */}
      {bidSheetOpen && selectedProduct && (
        <div>
          <BidInput
            auctionId={selectedProduct.id}
            currentPrice={selectedProduct.currentPrice}
            minIncrement={selectedProduct.increment}
            onBidSuccess={handleBidSuccess}
            onBidError={(error) => console.error(error)}
          />

          <RankingList
            rankings={rankings}
            currentUserId={user?.id}
          />
        </div>
      )}
    </div>
  );
};

export default LiveAuctionPage;
```

## 注意事项

1. **认证状态检查**: 在用户点击出价按钮时检查是否已登录
2. **WebSocket生命周期**: 确保在组件卸载时断开WebSocket连接
3. **错误处理**: 处理API调用失败和WebSocket错误
4. **状态同步**: 出价成功后更新排名列表和商品价格
5. **性能优化**: 使用消息节流避免频繁更新

## 测试清单

- [ ] 未登录用户点击出价 → 跳转登录页
- [ ] 已登录用户输入出价金额 → 验证通过
- [ ] 出价成功 → 更新排名列表
- [ ] WebSocket断开 → 自动重连
- [ ] 实时排名更新 → 正确显示
