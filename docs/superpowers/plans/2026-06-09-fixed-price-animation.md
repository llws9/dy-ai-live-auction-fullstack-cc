# 一口价上架动画实施计划 (方案 B)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现商家上架一口价商品时的引导性动画（方案B：顶部滑入停留、向右下角缩放渐隐，以及目标卡片的脉冲反馈）。

**Architecture:** 
1. 在 `LiveRoomSlide.tsx` 中监听 `fixedPriceItems` 的变化，识别出新添加的商品并将其存入 `animatingFixedPriceItems` 状态。
2. 新增 `FixedPriceIntroAnimation` 组件，利用 CSS `@keyframes` 完成滑入、停留、向右下角飞出的三阶段动画。动画结束时触发回调并从状态中移除。
3. 动画结束后，将对应商品的 ID 设为 `pulsingItemId`，并传递给 `FixedPriceCard`，通过 CSS 动画实现接收反馈（Pulse）效果。
4. 样式使用现有的 CSS 变量实现自动亮/暗色主题适配。

**Tech Stack:** React, CSS Modules, CSS Keyframes

---

### Task 1: 创建动画组件与样式

**Files:**
- Create: `frontend/h5/src/components/auction/FixedPriceIntroAnimation.tsx`
- Create: `frontend/h5/src/components/auction/FixedPriceIntroAnimation.module.css`

- [ ] **Step 1: 编写动画样式**

创建 `frontend/h5/src/components/auction/FixedPriceIntroAnimation.module.css` 并填入以下内容：

```css
.container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 100;
  pointer-events: none;
}

.card {
  position: absolute;
  top: -100px;
  left: calc(50% - 120px);
  width: 240px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: var(--spacing-3);
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  box-shadow: var(--shadow-key);
  animation: 
    slideDown 0.4s ease-out forwards,
    stayCenter 1.5s 0.4s forwards,
    flyToBottomRight 0.6s 1.9s ease-in forwards;
}

@keyframes slideDown {
  0% { opacity: 0; top: -100px; transform: scale(1); }
  100% { opacity: 1; top: 20%; transform: scale(1); }
}

@keyframes stayCenter {
  0% { opacity: 1; top: 20%; transform: scale(1); }
  100% { opacity: 1; top: 20%; transform: scale(1); }
}

@keyframes flyToBottomRight {
  0% { 
    opacity: 1; 
    top: 20%; 
    left: calc(50% - 120px);
    transform: scale(1); 
  }
  100% { 
    opacity: 0; 
    top: calc(100% - 120px); 
    left: calc(100% - 80px); 
    transform: scale(0.2); 
  }
}

.badge {
  position: absolute;
  top: -10px;
  left: -10px;
  background: var(--text-brand);
  color: var(--bg-page);
  padding: 4px 8px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-bold);
  box-shadow: var(--shadow-sm);
}

.cover {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-lg);
  object-fit: cover;
}

.coverFallback {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-lg);
  background: var(--bg-elevated);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.info {
  flex: 1;
  min-width: 0;
}

.title {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.price {
  display: block;
  margin-top: var(--spacing-1);
  color: var(--text-brand);
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-bold);
}
```

- [ ] **Step 2: 编写组件代码**

创建 `frontend/h5/src/components/auction/FixedPriceIntroAnimation.tsx` 并填入以下内容：

```tsx
import React from 'react';
import type { FixedPriceItem } from '@/api/fixedPrice';
import styles from './FixedPriceIntroAnimation.module.css';

interface FixedPriceIntroAnimationProps {
  item: FixedPriceItem;
  onComplete: (itemId: number) => void;
}

export function FixedPriceIntroAnimation({ item, onComplete }: FixedPriceIntroAnimationProps) {
  const product = item.product_brief ?? item.product ?? { title: item.product_title ?? '一口价商品' };

  return (
    <div className={styles.container}>
      <div 
        className={styles.card}
        onAnimationEnd={(e) => {
          if (e.animationName.includes('flyToBottomRight')) {
            onComplete(item.id);
          }
        }}
      >
        <div className={styles.badge}>新上架 一口价</div>
        {product.cover_image ? (
          <img className={styles.cover} src={product.cover_image} alt={product.title} />
        ) : (
          <div className={styles.coverFallback}>无图</div>
        )}
        <div className={styles.info}>
          <h3 className={styles.title}>{product.title}</h3>
          <span className={styles.price}>¥{item.price}</span>
        </div>
      </div>
    </div>
  );
}
```

### Task 2: 增加目标卡片的脉冲反馈 (Pulse)

**Files:**
- Modify: `frontend/h5/src/components/FixedPriceCard/index.tsx`
- Modify: `frontend/h5/src/components/FixedPriceCard/index.module.css`

- [ ] **Step 1: 在 CSS 中添加脉冲动画**

在 `frontend/h5/src/components/FixedPriceCard/index.module.css` 底部添加：

```css
.pulsing {
  animation: cardPulse 0.5s ease-in-out;
}

@keyframes cardPulse {
  0% { transform: scale(1); box-shadow: var(--shadow-key); }
  50% { transform: scale(1.05); box-shadow: 0 0 15px var(--text-brand); }
  100% { transform: scale(1); box-shadow: var(--shadow-key); }
}
```

- [ ] **Step 2: 修改 FixedPriceCard 支持 isPulsing 属性**

修改 `frontend/h5/src/components/FixedPriceCard/index.tsx`：

```tsx
// 在 FixedPriceCardProps 中增加 isPulsing
interface FixedPriceCardProps {
  item: FixedPriceItem;
  purchased?: boolean;
  isPulsing?: boolean;
  onPurchase: (itemId: number) => void;
}

// 修改组件声明
export default function FixedPriceCard({ item, purchased = false, isPulsing = false, onPurchase }: FixedPriceCardProps) {

// 修改根元素的 className
  return (
    <article className={`${styles.card} ${isPulsing ? styles.pulsing : ''}`} aria-label={`${product.title} 一口价商品`}>
// ...
```

### Task 3: 在直播间页面集成动画逻辑

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`

- [ ] **Step 1: 导入动画组件**

在 `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` 顶部适当位置（如其他 import 附近）添加：

```tsx
import { FixedPriceIntroAnimation } from '@/components/auction/FixedPriceIntroAnimation';
```

- [ ] **Step 2: 增加状态与状态跟踪**

在 `LiveRoomSlide` 组件内部，找到 `useState` 声明区域，添加以下状态：

```tsx
  const [animatingFixedPriceItems, setAnimatingFixedPriceItems] = useState<FixedPriceItem[]>([]);
  const [pulsingItemId, setPulsingItemId] = useState<number | null>(null);
  const prevFixedPriceItemsRef = useRef<Set<number>>(new Set());
  const isInitialLoadRef = useRef(true);
```

- [ ] **Step 3: 添加检测新上架商品的 useEffect**

在组件内部适当位置（如其他 `useEffect` 附近）添加检测逻辑：

```tsx
  useEffect(() => {
    const currentIds = new Set(fixedPriceItems.map(i => i.id));
    if (isInitialLoadRef.current) {
      prevFixedPriceItemsRef.current = currentIds;
      isInitialLoadRef.current = false;
      return;
    }

    const addedItems = fixedPriceItems.filter(i => !prevFixedPriceItemsRef.current.has(i.id));
    if (addedItems.length > 0) {
      setAnimatingFixedPriceItems(prev => [...prev, ...addedItems]);
    }
    
    prevFixedPriceItemsRef.current = currentIds;
  }, [fixedPriceItems]);

  const handleIntroAnimationComplete = useCallback((itemId: number) => {
    setAnimatingFixedPriceItems(prev => prev.filter(i => i.id !== itemId));
    setPulsingItemId(itemId);
    window.setTimeout(() => {
      setPulsingItemId(current => current === itemId ? null : current);
    }, 500);
  }, []);
```

- [ ] **Step 4: 渲染动画组件与传递 isPulsing**

在 JSX 的 return 树中，找到 `<FixedPriceFlair socket={fixedPriceSocket} />` 附近（页面底部），渲染入场动画：

```tsx
      {animatingFixedPriceItems.map(item => (
        <FixedPriceIntroAnimation 
          key={`anim-${item.id}`} 
          item={item} 
          onComplete={handleIntroAnimationComplete} 
        />
      ))}
```

找到渲染 `FixedPriceCard` 的地方，将 `isPulsing` 传进去：

```tsx
            <FixedPriceCard
              key={item.id}
              item={item}
              purchased={purchasedFixedPriceItemIds.has(item.id)}
              isPulsing={pulsingItemId === item.id}
              onPurchase={() => {
```
