# 竞拍成功动画 (Gavel Smash + V1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在前端项目中实现基于“一锤定音（Gavel Smash）+ 经典欢庆（V1）”方案的竞拍成功动效组件。

**Architecture:** 提取在 `bid_success_animations.html` 原型中的核心 DOM 结构与 CSS，转化为一个可复用的 React 组件 (`BidSuccessAnimation.tsx`)。动画逻辑通过纯 CSS (`@keyframes` 及 `animation-delay`) 编排，保障高性能。通过传递 props（如拍品名称、图片、价格等）使其支持动态数据展示。

**Tech Stack:** React, Tailwind CSS (若项目支持，否则使用原生 CSS/CSS Modules), 纯 CSS 动画

---

### Task 1: 创建动画核心样式文件

**Files:**
- Create: `frontend/styles/bid-success-animation.css`

- [ ] **Step 1: 写入核心 CSS 变量与动画 `@keyframes`**

```css
/* frontend/styles/bid-success-animation.css */
:root {
  --accent: #F59E0B;
}

.intro-container {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  pointer-events: none;
}

.gavel-wrapper {
  transform-origin: bottom right;
  animation: gavel-smash 1.2s cubic-bezier(0.2, 0, 0, 1) forwards;
}

@keyframes gavel-smash {
  0% { transform: rotate(0deg) scale(1.5) translateY(-150px) translateX(100px); opacity: 0; }
  15% { transform: rotate(35deg) scale(1.5) translateY(-180px) translateX(120px); opacity: 1; }
  30% { transform: rotate(40deg) scale(1.5) translateY(-180px) translateX(120px); opacity: 1; }
  40% { transform: rotate(-45deg) scale(1.5) translateY(20px) translateX(0px); opacity: 1; }
  70% { transform: rotate(-45deg) scale(1.5) translateY(20px) translateX(0px); opacity: 1; }
  100% { transform: rotate(-45deg) scale(1.5) translateY(20px) translateX(0px); opacity: 0; }
}

.shockwave {
  position: absolute;
  width: 150px;
  height: 40px;
  border-radius: 50%;
  border: 8px solid var(--accent);
  top: calc(50% + 80px);
  left: 50%;
  transform: translate(-50%, -50%) scale(0);
  opacity: 0;
  animation: shockwave-expand 0.6s 0.48s ease-out forwards;
}

@keyframes shockwave-expand {
  0% { transform: translate(-50%, -50%) scale(0.2); opacity: 1; border-width: 20px; }
  100% { transform: translate(-50%, -50%) scale(3.5); opacity: 0; border-width: 0px; }
}

.shake-trigger {
  animation: shake 0.5s 0.48s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
}

@keyframes shake {
  0%, 100% { transform: translate3d(0, 0, 0); }
  10%, 50%, 90% { transform: translate3d(-12px, 10px, 0); }
  30%, 70% { transform: translate3d(12px, -10px, 0); }
}

.ribbon {
  position: absolute;
  top: calc(50% + 40px);
  left: 50%;
  opacity: 0;
  animation: ribbon-burst 1.5s 0.48s cubic-bezier(0.25, 1, 0.5, 1) forwards;
}
.shape-rect { width: 14px; height: 14px; }
.shape-circle { width: 14px; height: 14px; border-radius: 50%; }
.shape-long { width: 8px; height: 28px; border-radius: 4px; }

@keyframes ribbon-burst {
  0% { transform: translate(-50%, -50%) rotate(0deg) scale(1); opacity: 0; }
  1% { opacity: 1; }
  50% { transform: translate(calc(-50% + var(--tx) * 0.8), calc(-50% + var(--ty) - 120px)) rotate(calc(var(--rot) * 0.5)) scale(1); opacity: 1; }
  100% { transform: translate(calc(-50% + var(--tx)), calc(-50% + var(--ty) + 200px)) rotate(var(--rot)) scale(0.5); opacity: 0; }
}

/* V1 卡片动画 */
.card-container {
  position: relative;
  z-index: 10;
}
.auction-card {
  background-color: #FFFFFF;
  border: 1px solid #E5E7EB;
  border-radius: 24px;
  padding: 40px;
  width: 360px;
  text-align: center;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  position: relative;
  overflow: hidden;
  opacity: 0;
}
.v1-anim {
  animation: card-appear 0.6s 0.8s cubic-bezier(0.175, 0.885, 0.32, 1.275) both;
}
@keyframes card-appear {
  0% { transform: scale(0.6) translateY(60px); opacity: 0; }
  100% { transform: scale(1) translateY(0); opacity: 1; }
}
.v1-stamp {
  position: absolute;
  top: 24px;
  right: 24px;
  width: 80px;
  height: 80px;
  border: 4px solid var(--accent);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
  font-weight: 800;
  font-size: 18px;
  transform: rotate(-15deg) scale(2);
  opacity: 0;
  animation: v1-stamp-in 0.5s 1.1s cubic-bezier(0.175, 0.885, 0.32, 1.5) both;
  z-index: 10;
}
@keyframes v1-stamp-in {
  0% { transform: rotate(-30deg) scale(3); opacity: 0; }
  50% { opacity: 1; }
  100% { transform: rotate(-15deg) scale(1); opacity: 1; }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/styles/bid-success-animation.css
git commit -m "style: add css keyframes and styles for bid success animation"
```

---

### Task 2: 实现 IntroAnimation 组件 (锤子与彩带)

**Files:**
- Create: `frontend/components/auction/IntroAnimation.tsx`

- [ ] **Step 1: 编写组件代码**

```tsx
import React, { useEffect, useState } from 'react';
import '../../styles/bid-success-animation.css';

export const IntroAnimation: React.FC = () => {
  const [ribbons, setRibbons] = useState<Array<{
    id: number; color: string; shape: string; tx: string; ty: string; rot: string;
  }>>([]);

  useEffect(() => {
    const colors = ['#F59E0B', '#EF4444', '#10B981', '#3B82F6', '#8B5CF6', '#EC4899', '#FCD34D'];
    const shapes = ['rect', 'circle', 'long'];
    
    const newRibbons = Array.from({ length: 80 }).map((_, i) => {
      const angle = (Math.random() * Math.PI * 2);
      const velocity = 300 + Math.random() * 500;
      const shape = shapes[Math.floor(Math.random() * shapes.length)];
      
      return {
        id: i,
        color: colors[Math.floor(Math.random() * colors.length)],
        shape,
        tx: `${Math.cos(angle) * velocity}px`,
        ty: `${Math.sin(angle) * velocity}px`,
        rot: `${(Math.random() - 0.5) * 1080}deg`,
      };
    });
    setRibbons(newRibbons);
  }, []);

  return (
    <div className="intro-container">
      <div className="gavel-wrapper">
        <svg width="200" height="200" viewBox="0 0 100 100" fill="none" style={{ filter: 'drop-shadow(0 15px 25px rgba(0,0,0,0.3))' }}>
          {/* Handle */}
          <rect x="44" y="30" width="12" height="55" rx="4" fill="#8B5A2B" />
          <rect x="44" y="30" width="6" height="55" rx="2" fill="#6B4423" />
          {/* Head */}
          <rect x="15" y="15" width="70" height="34" rx="8" fill="var(--accent)" />
          <rect x="10" y="20" width="12" height="24" rx="4" fill="#D97706" />
          <rect x="78" y="20" width="12" height="24" rx="4" fill="#D97706" />
          {/* Golden Band */}
          <rect x="42" y="15" width="16" height="34" fill="#FDE68A" />
        </svg>
      </div>
      <div className="shockwave" />
      {ribbons.map(r => (
        <div 
          key={r.id}
          className={`ribbon shape-${r.shape}`}
          style={{
            backgroundColor: r.color,
            '--tx': r.tx,
            '--ty': r.ty,
            '--rot': r.rot,
          } as React.CSSProperties}
        />
      ))}
    </div>
  );
};
```

- [ ] **Step 2: Commit**

```bash
git add frontend/components/auction/IntroAnimation.tsx
git commit -m "feat: implement IntroAnimation component with gavel smash and ribbons"
```

---

### Task 3: 实现 V1 卡片与外层调度组件

**Files:**
- Create: `frontend/components/auction/BidSuccessAnimation.tsx`

- [ ] **Step 1: 编写组件代码**

```tsx
import React, { useState, useEffect } from 'react';
import { IntroAnimation } from './IntroAnimation';
import '../../styles/bid-success-animation.css';

interface BidSuccessAnimationProps {
  productName: string;
  price: number;
  imageUrl?: string;
  onAnimationEnd?: () => void;
}

export const BidSuccessAnimation: React.FC<BidSuccessAnimationProps> = ({
  productName,
  price,
  imageUrl,
  onAnimationEnd
}) => {
  const [show, setShow] = useState(true);

  // 动画总时长大概在 2.5s 左右，3 秒后可触发结束回调
  useEffect(() => {
    const timer = setTimeout(() => {
      if (onAnimationEnd) onAnimationEnd();
    }, 3000);
    return () => clearTimeout(timer);
  }, [onAnimationEnd]);

  if (!show) return null;

  return (
    <div className="shake-trigger" style={{ position: 'fixed', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9999 }}>
      <IntroAnimation />
      
      <div className="card-container">
        <div className="auction-card v1-anim">
          <div className="v1-stamp">成交</div>
          
          <div style={{
            width: 120, height: 120, borderRadius: 16, margin: '0 auto 24px',
            background: '#F3F4F6', display: 'flex', alignItems: 'center', justifyContent: 'center',
            overflow: 'hidden'
          }}>
            {imageUrl ? (
              <img src={imageUrl} alt={productName} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
            ) : (
              <span style={{ color: '#6B7280', fontSize: 14 }}>[ 拍品图 ]</span>
            )}
          </div>
          
          <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 4, color: '#111827' }}>
            {productName}
          </div>
          <div style={{ fontSize: 14, color: '#6B7280' }}>
            最终成交价
          </div>
          <div style={{ fontSize: 32, fontWeight: 800, color: '#F59E0B', marginBottom: 8 }}>
            ¥ {price.toLocaleString()}
          </div>
        </div>
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Commit**

```bash
git add frontend/components/auction/BidSuccessAnimation.tsx
git commit -m "feat: implement BidSuccessAnimation orchestrator component with V1 card"
```
