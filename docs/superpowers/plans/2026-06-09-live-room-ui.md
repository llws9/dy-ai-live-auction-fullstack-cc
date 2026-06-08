# Live Room UI/UX Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement 11 UI/UX design decisions for the Live Room, enhancing atmosphere, social interaction, urgency, and visual polish.

**Architecture:** We will implement the changes primarily in `LiveRoomSlide.tsx` and its associated CSS module. New interactive features (like tap burst hearts and glitch countdown) will be extracted into independent components to avoid bloating the main slide component. State for these features will be managed locally or via existing WebSocket event streams.

**Tech Stack:** React, TypeScript, CSS Modules.

---

### Task 1: Visual Polish - Video Gradient and Top Bar

**Files:**
- Modify: `frontend/h5/src/pages/Live/Live.module.css`
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`

- [ ] **Step 1: Update video gradient mask**

Modify `.videoGradient` in `Live.module.css` to use a linear 3-stop gradient (dark at top and bottom, transparent in middle).

```css
/* In frontend/h5/src/pages/Live/Live.module.css */
.videoGradient {
  position: absolute;
  inset: 0;
  /* V1: Linear 3-Stop */
  background: linear-gradient(180deg, rgba(0,0,0,0.7) 0%, transparent 25%, transparent 70%, rgba(0,0,0,0.8) 100%);
  pointer-events: none;
  z-index: 1;
}
```

- [ ] **Step 2: Update Top Bar / Host Pill styling**

Apply glassmorphism to `.topBar`, `.hostPill`, and `.statusPill`.

```css
/* In frontend/h5/src/pages/Live/Live.module.css */
.hostPill, .statusPill {
  /* ... existing layout styles ... */
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 24px;
  box-shadow: var(--shadow-sm);
}

:global(:root[data-theme='dark']) .hostPill,
:global(:root[data-theme='dark']) .statusPill {
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.1);
}
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat: apply glassmorphism to top bar and optimize video gradient mask"
```

---

### Task 2: Atmosphere - Global Haptic/Audio Toggle (a3)

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`

- [ ] **Step 1: Add state and styles for the toggle**

```css
/* In frontend/h5/src/pages/Live/Live.module.css */
.audioTogglePill {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #fff;
  transition: all 0.2s;
}

:global(:root[data-theme='dark']) .audioTogglePill {
  background: rgba(30, 41, 59, 0.4);
}
```

- [ ] **Step 2: Implement toggle in component**

In `LiveRoomSlide.tsx`, add state and render the pill in `.rightActions`.

```tsx
// Add state
const [hapticsEnabled, setHapticsEnabled] = useState(true);

// Add helper function
const triggerHaptic = useCallback(() => {
  if (!hapticsEnabled) return;
  if (typeof navigator !== 'undefined' && navigator.vibrate) {
    navigator.vibrate(50);
  }
  // TODO: play lightweight sound effect here
}, [hapticsEnabled]);

// Render in .rightActions
<div className={styles.audioTogglePill} onClick={() => setHapticsEnabled(!hapticsEnabled)}>
  🎵 {hapticsEnabled ? 'ON' : 'OFF'}
</div>
```

- [ ] **Step 3: Wire up haptics to bid success**

Locate the bid success handling (e.g., inside `handleBid` or `bidSuccessFlair` effect) and call `triggerHaptic()`.

- [ ] **Step 4: Commit**

```bash
git commit -am "feat: add global haptic/audio toggle pill"
```

---

### Task 3: Atmosphere - Gamified Bid Flair (a1)

**Files:**
- Create: `frontend/h5/src/components/LiveRoom/BidFlairOverlay.tsx`
- Create: `frontend/h5/src/components/LiveRoom/BidFlairOverlay.module.css`
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`

- [ ] **Step 1: Create CSS Module for Gamified Flair**

```css
/* frontend/h5/src/components/LiveRoom/BidFlairOverlay.module.css */
.flairContainer {
  position: absolute;
  top: 40%;
  left: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
  z-index: 50;
}

.flairItem {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid var(--color-violet-5);
  padding: 6px 12px 6px 6px;
  border-radius: 8px;
  color: white;
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 0 10px rgba(167, 139, 250, 0.4);
  animation: slideIn 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards,
             fadeOut 0.3s ease-in 1.7s forwards;
}

.flairItem.isSelf {
  border-color: var(--color-sky-5);
  box-shadow: 0 0 12px rgba(56, 189, 248, 0.6);
}

.avatar { width: 20px; height: 20px; border-radius: 50%; }
.comboText { color: var(--color-violet-4); font-style: italic; font-weight: 800; font-size: 11px; margin-right: 4px; }
.priceText { color: var(--color-amber-4); }

@keyframes slideIn {
  from { transform: translateX(-100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
@keyframes fadeOut {
  to { opacity: 0; transform: translateY(-10px); }
}
```

- [ ] **Step 2: Create BidFlairOverlay component**

```tsx
import React, { useEffect, useState } from 'react';
import styles from './BidFlairOverlay.module.css';

interface BidEvent {
  id: string;
  userId: string;
  avatar: string;
  price: string;
  combo: number;
  isSelf: boolean;
}

// TODO: Need real combo data from props/WS. For now, simulate or default to 1.
export const BidFlairOverlay: React.FC<{ latestBid?: BidEvent }> = ({ latestBid }) => {
  const [flairs, setFlairs] = useState<BidEvent[]>([]);

  useEffect(() => {
    if (latestBid) {
      setFlairs(prev => [...prev.slice(-4), latestBid]);
      const timer = setTimeout(() => {
        setFlairs(prev => prev.filter(f => f.id !== latestBid.id));
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [latestBid]);

  return (
    <div className={styles.flairContainer}>
      {flairs.map(f => (
        <div key={f.id} className={`${styles.flairItem} ${f.isSelf ? styles.isSelf : ''}`}>
          <img src={f.avatar} className={styles.avatar} alt="" />
          {f.combo > 1 && <span className={styles.comboText}>x{f.combo} COMBO</span>}
          <span>出价 <span className={styles.priceText}>¥{f.price}</span></span>
        </div>
      ))}
    </div>
  );
};
```

- [ ] **Step 3: Integrate into LiveRoomSlide**

Remove old flair logic and render `<BidFlairOverlay latestBid={...} />` inside `LiveRoomSlide.tsx`.

- [ ] **Step 4: Commit**

```bash
git add frontend/h5/src/components/LiveRoom/
git commit -am "feat: implement gamified bid flair overlay"
```

---

### Task 4: Atmosphere - Floating Status Pill (a2)

**Files:**
- Modify: `frontend/h5/src/pages/Live/Live.module.css`
- Modify: `frontend/h5/src/pages/Live/BidDock.tsx`

- [ ] **Step 1: Add CSS for Floating Status Pill**

```css
/* In frontend/h5/src/pages/Live/Live.module.css */
.floatingStatusPill {
  position: absolute;
  top: -36px;
  left: 50%;
  transform: translateX(-50%);
  padding: 6px 16px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 6px;
  box-shadow: var(--shadow-md);
  z-index: 10;
  animation: popIn 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.statusLeading {
  background: var(--color-mint-1);
  color: var(--color-mint-7);
  border: 1px solid var(--color-mint-3);
}

.statusOvertaken {
  background: var(--color-coral-1);
  color: var(--color-coral-7);
  border: 1px solid var(--color-coral-3);
}

:global(:root[data-theme='dark']) .statusLeading {
  background: rgba(16, 185, 129, 0.2);
  color: var(--color-mint-4);
  border-color: var(--color-mint-6);
}

:global(:root[data-theme='dark']) .statusOvertaken {
  background: rgba(239, 68, 68, 0.2);
  color: var(--color-coral-4);
  border-color: var(--color-coral-6);
}

@keyframes popIn {
  from { transform: translate(-50%, 10px) scale(0.9); opacity: 0; }
  to { transform: translate(-50%, 0) scale(1); opacity: 1; }
}
```

- [ ] **Step 2: Render in BidDock**

In `BidDock.tsx`, add logic to determine if the user is leading based on `myBid` and `currentPrice`.

```tsx
// Inside BidDock render, inside the .dock container but outside the sheet
const isLeading = myBid && myBid >= currentPrice;
const hasBid = !!myBid;

{hasBid && (
  <div className={`${styles.floatingStatusPill} ${isLeading ? styles.statusLeading : styles.statusOvertaken}`}>
    {isLeading ? '👑 当前领先' : '⚠️ 被超越，点击反超'}
  </div>
)}
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat: add floating status pill for bid lead state"
```

---

### Task 5: Social - Tap Burst Hearts (s1)

**Files:**
- Create: `frontend/h5/src/components/LiveRoom/TapBurstHearts.tsx`
- Create: `frontend/h5/src/components/LiveRoom/TapBurstHearts.module.css`
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`

- [ ] **Step 1: CSS for particles**

```css
.container {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 40;
  overflow: hidden;
}

.heart {
  position: absolute;
  font-size: 24px;
  user-select: none;
  filter: drop-shadow(0 0 4px rgba(239, 68, 68, 0.5));
  animation: floatUp 1s ease-out forwards;
}

@keyframes floatUp {
  0% { transform: translate(-50%, -50%) scale(0.5) rotate(0deg); opacity: 1; }
  50% { transform: translate(var(--tx), var(--ty)) scale(1.2) rotate(var(--rot)); opacity: 1; }
  100% { transform: translate(var(--tx2), var(--ty2)) scale(0.8) rotate(var(--rot2)); opacity: 0; }
}
```

- [ ] **Step 2: Create component**

```tsx
import React, { useState, useCallback, useEffect } from 'react';
import styles from './TapBurstHearts.module.css';

interface Heart { id: number; x: number; y: number; tx: string; ty: string; tx2: string; ty2: string; rot: string; rot2: string; }

export const TapBurstHearts: React.FC = () => {
  const [hearts, setHearts] = useState<Heart[]>([]);

  const handleDoubleClick = useCallback((e: MouseEvent) => {
    // Ignore if clicking on buttons or dock
    if ((e.target as HTMLElement).closest('button, .dock, .sheet')) return;

    const newHearts = Array.from({ length: 5 }).map((_, i) => ({
      id: Date.now() + i,
      x: e.clientX,
      y: e.clientY,
      tx: `${(Math.random() - 0.5) * 100}px`,
      ty: `-${Math.random() * 50 + 50}px`,
      tx2: `${(Math.random() - 0.5) * 150}px`,
      ty2: `-${Math.random() * 100 + 100}px`,
      rot: `${(Math.random() - 0.5) * 60}deg`,
      rot2: `${(Math.random() - 0.5) * 120}deg`,
    }));
    setHearts(prev => [...prev.slice(-20), ...newHearts]);
    // TODO: Need WS event to broadcast like
  }, []);

  useEffect(() => {
    window.addEventListener('dblclick', handleDoubleClick);
    return () => window.removeEventListener('dblclick', handleDoubleClick);
  }, [handleDoubleClick]);

  return (
    <div className={styles.container}>
      {hearts.map(h => (
        <div key={h.id} className={styles.heart} style={{
          left: h.x, top: h.y,
          '--tx': h.tx, '--ty': h.ty, '--tx2': h.tx2, '--ty2': h.ty2,
          '--rot': h.rot, '--rot2': h.rot2
        } as React.CSSProperties}>❤️</div>
      ))}
    </div>
  );
};
```

- [ ] **Step 3: Integrate into Slide**

Add `<TapBurstHearts />` to `LiveRoomSlide.tsx`.

- [ ] **Step 4: Commit**

```bash
git add frontend/h5/src/components/LiveRoom/
git commit -am "feat: implement double-tap burst hearts animation"
```

---

### Task 6: Social - Horizontal Quick Chat (s2)

**Files:**
- Modify: `frontend/h5/src/components/LiveChat/ChatPanel.tsx`
- Modify: `frontend/h5/src/components/LiveChat/ChatPanel.module.css`

- [ ] **Step 1: CSS for Quick Chat Scroll**

```css
.quickChatContainer {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 8px 12px;
  margin-bottom: 4px;
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none;  /* IE and Edge */
}
.quickChatContainer::-webkit-scrollbar { display: none; }

.quickChatPill {
  flex-shrink: 0;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  color: var(--text-primary);
  cursor: pointer;
  white-space: nowrap;
}
:global(:root[data-theme='dark']) .quickChatPill {
  background: rgba(255,255,255,0.1);
  border-color: rgba(255,255,255,0.2);
}
```

- [ ] **Step 2: Implement in ChatPanel**

```tsx
const QUICK_PHRASES = ["+1", "冲鸭", "捡漏了", "还能加吗", "太顶了"];

// Above the input box
<div className={styles.quickChatContainer}>
  {QUICK_PHRASES.map(phrase => (
    <div key={phrase} className={styles.quickChatPill} onClick={() => handleSendQuick(phrase)}>
      {phrase}
    </div>
  ))}
</div>
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat: add horizontal scroll quick chat phrases"
```

---

### Task 7: Urgency - Color Shift Countdown (c1) & Marquee Heat (c2)

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`

- [ ] **Step 1: CSS for Color Shift & Marquee**

```css
/* Color Shift Countdown */
.countdownUrgentText {
  color: var(--color-coral-6);
  font-weight: 800;
  /* Ensure no layout shift when going bold */
}

/* Marquee Heat */
.heatMarqueeContainer {
  width: 100%;
  overflow: hidden;
  background: rgba(0,0,0,0.4);
  padding: 4px 8px;
  border-radius: 4px;
  margin-top: 8px;
  display: flex;
}
.heatMarqueeText {
  font-size: 11px;
  color: #fff;
  white-space: nowrap;
  animation: marquee 10s linear infinite;
}
@keyframes marquee {
  0% { transform: translateX(100%); }
  100% { transform: translateX(-100%); }
}
```

- [ ] **Step 2: Apply styles**

Update the countdown render logic to apply `.countdownUrgentText` when `timeLeft < 10000`.
Add the Marquee component below the price block.

```tsx
// TODO: Need real bidderCount/viewerCount from props
const bidderCount = 12;
const viewerCount = 300;

<div className={styles.heatMarqueeContainer}>
  <div className={styles.heatMarqueeText}>
    🔥 已有 {bidderCount} 人出价 · {viewerCount} 人围观
  </div>
</div>
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat: color shift countdown and marquee heat text"
```

---

### Task 8: Urgency - Cyber Glitch 5s Countdown (c3)

**Files:**
- Create: `frontend/h5/src/components/LiveRoom/GlitchCountdown.tsx`
- Create: `frontend/h5/src/components/LiveRoom/GlitchCountdown.module.css`
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`

- [ ] **Step 1: CSS Animation**

```css
.container {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  z-index: 60;
}

.glitchText {
  font-family: var(--font-mono);
  font-size: 80px;
  font-weight: 900;
  color: var(--color-coral-5);
  animation: riseGlitch 1s infinite;
}

@keyframes riseGlitch {
  0%   { transform: translateY(80px) scale(0.5); opacity: 0; }
  20%  { transform: translateY(0) scale(1); opacity: 1; text-shadow: 4px 0 var(--color-sky-5), -4px 0 var(--color-coral-6); }
  25%  { transform: translateY(0) scale(1) skewX(10deg); text-shadow: -4px 0 var(--color-sky-5), 4px 0 var(--color-coral-6); }
  30%  { transform: translateY(0) scale(1) skewX(-10deg); text-shadow: 4px 0 var(--color-sky-5), -4px 0 var(--color-coral-6); }
  35%  { transform: translateY(0) scale(1); opacity: 1; text-shadow: none; color: var(--color-coral-6); }
  70%  { transform: translateY(0) scale(1); opacity: 1; color: var(--color-coral-6); }
  100% { transform: translateY(-40px) scale(1.5); opacity: 0; letter-spacing: 20px; filter: blur(4px); }
}
```

- [ ] **Step 2: Component logic**

```tsx
export const GlitchCountdown: React.FC<{ timeLeft: number, isSheetOpen: boolean }> = ({ timeLeft, isSheetOpen }) => {
  if (isSheetOpen || timeLeft > 5000 || timeLeft <= 0) return null;

  const seconds = Math.ceil(timeLeft / 1000);

  return (
    <div className={styles.container}>
      {/* Keying by seconds restarts animation every second */}
      <div key={seconds} className={styles.glitchText}>{seconds}</div>
    </div>
  );
};
```

- [ ] **Step 3: Integrate into Slide**

Add `<GlitchCountdown timeLeft={timeLeft} isSheetOpen={isSheetOpen} />` to `LiveRoomSlide.tsx`. Make sure you track `isSheetOpen` state (might need to lift state from `BidDock` if not already lifted).

- [ ] **Step 4: Commit**

```bash
git add frontend/h5/src/components/LiveRoom/
git commit -am "feat: add cyber glitch 5s immersive countdown"
```

---

### Task 9: Status Result - Shatter Fade Unsold Animation (u1)

**Files:**
- Create: `frontend/h5/src/components/LiveRoom/UnsoldAnimation.tsx`
- Create: `frontend/h5/src/components/LiveRoom/UnsoldAnimation.module.css`
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`

- [ ] **Step 1: CSS Animation**

```css
.container {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0,0,0,0.6);
  z-index: 100;
  pointer-events: none;
}

.shatterText {
  font-weight: 800;
  font-size: 32px;
  color: var(--color-slate-4);
  letter-spacing: 4px;
  animation: shatterFade 3s forwards;
}

@keyframes shatterFade {
  0%   { opacity: 0; filter: blur(0); transform: translateY(-20px) scale(1.2); }
  10%  { opacity: 1; filter: blur(0); transform: translateY(0) scale(1); }
  40%  { opacity: 1; filter: blur(0); transform: translateY(0) scale(1); letter-spacing: 4px; }
  80%  { opacity: 0.6; filter: blur(4px); transform: translateY(20px) scale(0.95); letter-spacing: 16px; }
  100% { opacity: 0; filter: blur(10px); transform: translateY(60px) scale(0.8); letter-spacing: 40px; }
}
```

- [ ] **Step 2: Component implementation**

```tsx
export const UnsoldAnimation: React.FC<{ isUnsold: boolean }> = ({ isUnsold }) => {
  if (!isUnsold) return null;
  return (
    <div className={styles.container}>
      <div className={styles.shatterText}>遗憾流拍</div>
    </div>
  );
};
```

- [ ] **Step 3: Integrate**

Determine `isUnsold` condition (e.g. `auctionStatus === ENDED && currentPrice === startPrice` or via specific WS event) and render component in `LiveRoomSlide.tsx`. Ensure it takes precedence over or replaces the Success animation.

- [ ] **Step 4: Commit**

```bash
git add frontend/h5/src/components/LiveRoom/
git commit -am "feat: add shatter fade animation for unsold state"
```
