# A5 一口价秒杀 - M3 前端 H5 + 监控接入 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 M1+M2 后端就绪基础上，落地观众端 H5 一口价模块（卡片 / 抢购弹窗 / 飘屏）+ 主播端管理（上下架）+ Prometheus / Grafana 接入。

**Tech Stack:** React 18 + Vite + TS + CSS Modules + Tailwind + 现有 LiveStreamSocket（B1 产物）+ shopspring/decimal（仅前端字符串处理）+ Prometheus client_golang + Grafana

**前置依赖：** M1 + M2 全绿；B1 LiveStreamSocket 提供 `subscribe(type, handler)` 钩子

**Spec：** [2026-06-01-fixed-price-sale-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-01-fixed-price-sale-design.md) §4.6 §4.7 §6 §7

---

## File Structure

**Create (frontend H5):**
- `frontend/h5/src/api/fixedPrice.ts` + `.test.ts` - REST 客户端 + 幂等 key 生成
- `frontend/h5/src/hooks/useFixedPriceItems.ts` + `.test.tsx` - 列表 + WS 订阅 reducer
- `frontend/h5/src/components/FixedPriceCard/index.tsx` + `.module.css` + `.test.tsx`
- `frontend/h5/src/components/FixedPricePurchaseModal/index.tsx` + `.module.css` + `.test.tsx`
- `frontend/h5/src/components/FixedPriceFlair/index.tsx` + `.module.css` + `.test.tsx` - 飘屏

**Create (frontend admin):**
- `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx` - 上架表单 + 列表

**Create (observability):**
- `backend/auction/metrics/fixed_price_metrics.go` - Prometheus counters/histograms
- `ops/grafana/fixed-price-dashboard.json` - Dashboard 定义

**Modify:**
- `frontend/h5/src/pages/LiveStream/index.tsx` - 挂载 FixedPriceCard / Modal / Flair
- `backend/auction/service/fixed_price.go` - 在关键路径埋点

---

### Task 1: H5 API 客户端 + 幂等 key

**Files:**
- Test: `frontend/h5/src/api/fixedPrice.test.ts`
- Create: `frontend/h5/src/api/fixedPrice.ts`

- [ ] **Step 1.1: 写失败测试**

```ts
// frontend/h5/src/api/fixedPrice.test.ts
import { describe, it, expect, vi } from 'vitest';
import { fetchItems, purchase, generateIdempotencyKey } from './fixedPrice';
import { request } from './request';

vi.mock('./request');

describe('fixedPrice API', () => {
  it('generateIdempotencyKey 返回符合 UUIDv4 的字符串', () => {
    const k = generateIdempotencyKey();
    expect(k).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i);
  });

  it('purchase 把 idempotencyKey 放到 X-Idempotency-Key header', async () => {
    const spy = vi.mocked(request).mockResolvedValue({ data: { order_id: 9 } });
    await purchase({ liveStreamId: 1001, itemId: 7001, idempotencyKey: 'abc' });
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({
      method: 'POST',
      url: '/api/v1/live-streams/1001/fixed-price-items/7001/purchase',
      headers: { 'X-Idempotency-Key': 'abc' },
    }));
  });

  it('fetchItems GET 路径正确', async () => {
    vi.mocked(request).mockResolvedValue({ data: { items: [] } });
    await fetchItems(1001);
    expect(request).toHaveBeenCalledWith(expect.objectContaining({
      method: 'GET',
      url: '/api/v1/live-streams/1001/fixed-price-items',
    }));
  });
});
```

- [ ] **Step 1.2: 跑测试确认失败**

Run: `cd frontend/h5 && pnpm vitest run src/api/fixedPrice`
Expected: FAIL

- [ ] **Step 1.3: 写实现**

```ts
// frontend/h5/src/api/fixedPrice.ts
import { request } from './request';

export interface ProductBrief { id: number; title: string; cover_image?: string }
export interface FixedPriceItem {
  id: number; product_id: number;
  price: string; total_stock: number; remaining_stock: number;
  status: 'live' | 'sold_out' | 'offline';
  product_brief: ProductBrief;
}

export function generateIdempotencyKey(): string {
  // RFC4122 v4
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export function fetchItems(liveStreamId: number) {
  return request<{ items: FixedPriceItem[] }>({
    method: 'GET',
    url: `/api/v1/live-streams/${liveStreamId}/fixed-price-items`,
  });
}

export function purchase(params: { liveStreamId: number; itemId: number; idempotencyKey: string }) {
  return request<{ order_id: number; price: string; remaining_stock: number }>({
    method: 'POST',
    url: `/api/v1/live-streams/${params.liveStreamId}/fixed-price-items/${params.itemId}/purchase`,
    headers: { 'X-Idempotency-Key': params.idempotencyKey },
  });
}
```

- [ ] **Step 1.4: 跑测试通过**

Run: `cd frontend/h5 && pnpm vitest run src/api/fixedPrice`
Expected: PASS

- [ ] **Step 1.5: Commit**

```bash
git add frontend/h5/src/api/fixedPrice*
git commit -m "feat(fixed-price): h5 API client with idempotency key (M3.T1)"
```

---

### Task 2: useFixedPriceItems hook (列表 + WS reducer)

**Files:**
- Test: `frontend/h5/src/hooks/useFixedPriceItems.test.tsx`
- Create: `frontend/h5/src/hooks/useFixedPriceItems.ts`

> **职责：** 初次拉 REST 列表 → 订阅 5 种 WS 消息 → 用 reducer 增量更新；返回 `items` + `byId` 索引。

- [ ] **Step 2.1: 写失败测试（reducer 纯函数）**

```tsx
// frontend/h5/src/hooks/useFixedPriceItems.test.tsx
import { describe, it, expect } from 'vitest';
import { reduceItems } from './useFixedPriceItems';

const baseItem = {
  id: 7001, product_id: 5001, price: '99.00',
  total_stock: 100, remaining_stock: 100, status: 'live',
  product_brief: { id: 5001, title: '翡翠' },
} as const;

describe('reduceItems', () => {
  it('fixed_price_listed 追加新 item', () => {
    const next = reduceItems([], { type: 'fixed_price_listed', payload: { item: baseItem } });
    expect(next).toHaveLength(1);
  });
  it('fixed_price_stock 更新 remaining_stock', () => {
    const next = reduceItems([baseItem], {
      type: 'fixed_price_stock', payload: { item_id: 7001, remaining_stock: 87 }
    });
    expect(next[0].remaining_stock).toBe(87);
  });
  it('fixed_price_sold_out 设置 status=sold_out 且 stock=0', () => {
    const next = reduceItems([baseItem], {
      type: 'fixed_price_sold_out', payload: { item_id: 7001 }
    });
    expect(next[0].status).toBe('sold_out');
    expect(next[0].remaining_stock).toBe(0);
  });
  it('fixed_price_offline 移除 item', () => {
    const next = reduceItems([baseItem], {
      type: 'fixed_price_offline', payload: { item_id: 7001 }
    });
    expect(next).toHaveLength(0);
  });
  it('未知 type 不改变 state', () => {
    const next = reduceItems([baseItem], { type: 'noop' as any, payload: {} });
    expect(next).toBe(next); // 同引用即可（实现可返回 prev）
  });
});
```

- [ ] **Step 2.2: 写实现**

```ts
// frontend/h5/src/hooks/useFixedPriceItems.ts
import { useEffect, useReducer, useMemo } from 'react';
import { fetchItems, FixedPriceItem } from '../api/fixedPrice';
import { useLiveStreamSocket } from './useLiveStreamSocket'; // B1 产物

type Action =
  | { type: 'init'; payload: { items: FixedPriceItem[] } }
  | { type: 'fixed_price_listed'; payload: { item: FixedPriceItem } }
  | { type: 'fixed_price_stock'; payload: { item_id: number; remaining_stock: number } }
  | { type: 'fixed_price_sold_out'; payload: { item_id: number } }
  | { type: 'fixed_price_offline'; payload: { item_id: number } };

export function reduceItems(state: FixedPriceItem[], action: Action): FixedPriceItem[] {
  switch (action.type) {
    case 'init': return action.payload.items;
    case 'fixed_price_listed':
      return state.find(i => i.id === action.payload.item.id) ? state : [...state, action.payload.item];
    case 'fixed_price_stock':
      return state.map(i => i.id === action.payload.item_id
        ? { ...i, remaining_stock: action.payload.remaining_stock } : i);
    case 'fixed_price_sold_out':
      return state.map(i => i.id === action.payload.item_id
        ? { ...i, remaining_stock: 0, status: 'sold_out' } : i);
    case 'fixed_price_offline':
      return state.filter(i => i.id !== action.payload.item_id);
    default: return state;
  }
}

export function useFixedPriceItems(liveStreamId: number) {
  const [items, dispatch] = useReducer(reduceItems, [] as FixedPriceItem[]);
  const sock = useLiveStreamSocket(liveStreamId);

  useEffect(() => {
    fetchItems(liveStreamId).then(r => dispatch({ type: 'init', payload: { items: r.data.items } }));
  }, [liveStreamId]);

  useEffect(() => {
    const types = ['fixed_price_listed', 'fixed_price_stock', 'fixed_price_sold_out', 'fixed_price_offline'] as const;
    const offs = types.map(t => sock.subscribe(t, (env: any) => dispatch({ type: t, payload: env.payload } as any)));
    return () => offs.forEach(off => off());
  }, [sock]);

  const byId = useMemo(() => Object.fromEntries(items.map(i => [i.id, i])), [items]);
  return { items, byId };
}
```

- [ ] **Step 2.3: 跑测试通过**

Run: `cd frontend/h5 && pnpm vitest run src/hooks/useFixedPriceItems`
Expected: PASS

- [ ] **Step 2.4: Commit**

```bash
git add frontend/h5/src/hooks/useFixedPriceItems*
git commit -m "feat(fixed-price): h5 hook with reducer for ws+rest sync (M3.T2)"
```

---

### Task 3: FixedPriceCard 卡片组件

**Files:**
- Create: `frontend/h5/src/components/FixedPriceCard/index.tsx`
- Create: `frontend/h5/src/components/FixedPriceCard/index.module.css`
- Test: `frontend/h5/src/components/FixedPriceCard/index.test.tsx`

> **UI 要求（iOS-like）：** 卡片高度 ≥ 88px，按钮触控区 ≥ 44px；状态分 live / sold_out / offline；CSS 变量化。

- [ ] **Step 3.1: 写失败测试**

```tsx
// frontend/h5/src/components/FixedPriceCard/index.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import FixedPriceCard from './index';

const item = {
  id: 7001, product_id: 5001, price: '99.00',
  total_stock: 100, remaining_stock: 87, status: 'live' as const,
  product_brief: { id: 5001, title: '翡翠手镯', cover_image: 'cdn://a.jpg' },
};

describe('FixedPriceCard', () => {
  it('显示价格和剩余库存', () => {
    render(<FixedPriceCard item={item} onPurchase={() => {}} />);
    expect(screen.getByText('¥99.00')).toBeInTheDocument();
    expect(screen.getByText(/剩.*87/)).toBeInTheDocument();
  });

  it('点击按钮触发 onPurchase', () => {
    const cb = vi.fn();
    render(<FixedPriceCard item={item} onPurchase={cb} />);
    fireEvent.click(screen.getByRole('button', { name: /立即抢/ }));
    expect(cb).toHaveBeenCalledWith(7001);
  });

  it('sold_out 状态按钮禁用并显示文案', () => {
    render(<FixedPriceCard item={{ ...item, status: 'sold_out', remaining_stock: 0 }} onPurchase={() => {}} />);
    const btn = screen.getByRole('button');
    expect(btn).toBeDisabled();
    expect(btn).toHaveTextContent(/已售罄/);
  });
});
```

- [ ] **Step 3.2: 写实现（节选要点）**

```tsx
// frontend/h5/src/components/FixedPriceCard/index.tsx
import styles from './index.module.css';
import type { FixedPriceItem } from '../../api/fixedPrice';

interface Props { item: FixedPriceItem; onPurchase: (itemId: number) => void }

export default function FixedPriceCard({ item, onPurchase }: Props) {
  const isSoldOut = item.status === 'sold_out' || item.remaining_stock <= 0;
  return (
    <div className={styles.card}>
      <img src={item.product_brief.cover_image} alt={item.product_brief.title} className={styles.cover}/>
      <div className={styles.info}>
        <div className={styles.title}>{item.product_brief.title}</div>
        <div className={styles.price}>¥{item.price}</div>
        <div className={styles.stock}>剩 {item.remaining_stock} / {item.total_stock}</div>
      </div>
      <button
        className={styles.btn}
        disabled={isSoldOut}
        onClick={() => onPurchase(item.id)}
      >
        {isSoldOut ? '已售罄' : '立即抢'}
      </button>
    </div>
  );
}
```

CSS 关键：
```css
.card { display: flex; min-height: 88px; padding: 12px; background: var(--card-bg); border-radius: 12px; }
.btn { min-width: 88px; min-height: 44px; border-radius: 22px; background: var(--accent); color: #fff; }
.btn:disabled { background: var(--gray-3); }
```

- [ ] **Step 3.3: 跑测试通过**

Run: `cd frontend/h5 && pnpm vitest run src/components/FixedPriceCard`
Expected: PASS

- [ ] **Step 3.4: Commit**

```bash
git add frontend/h5/src/components/FixedPriceCard/
git commit -m "feat(fixed-price): h5 card component (M3.T3)"
```

---

### Task 4: FixedPricePurchaseModal 抢购弹窗

**Files:**
- Create: `frontend/h5/src/components/FixedPricePurchaseModal/index.tsx`
- Create: `frontend/h5/src/components/FixedPricePurchaseModal/index.module.css`
- Test: `frontend/h5/src/components/FixedPricePurchaseModal/index.test.tsx`

> **职责：** 弹窗内点击「确认抢购」→ 生成 idempotencyKey → 调 purchase API → 处理 4 种结果：
> - 200 成功：Toast 「抢到了！」 + 关闭弹窗 + 跳订单详情
> - 402 余额不足：弹二级 Modal 「余额不足，去充值」 + 跳 /wallet/recharge
> - 409 售罄/已购：Toast「已售罄」或「您已购买过」+ 关闭
> - 网络异常：保留 idempotencyKey 重试 1 次（按钮置 loading）

- [ ] **Step 4.1: 写失败测试（覆盖 4 种分支）**

```tsx
// frontend/h5/src/components/FixedPricePurchaseModal/index.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FixedPricePurchaseModal from './index';
import * as api from '../../api/fixedPrice';

vi.mock('../../api/fixedPrice');

const item = { id: 7001, product_id: 5001, price: '99.00', total_stock: 100, remaining_stock: 87, status: 'live' as const, product_brief: { id: 5001, title: '翡翠' } };

describe('FixedPricePurchaseModal', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('成功路径：200 → 调用 onSuccess', async () => {
    vi.mocked(api.purchase).mockResolvedValue({ data: { order_id: 9, price: '99.00', remaining_stock: 86 } });
    const onSuccess = vi.fn();
    render(<FixedPricePurchaseModal item={item} liveStreamId={1001} open={true} onClose={() => {}} onSuccess={onSuccess} />);
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));
    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(9));
  });

  it('余额不足 402：触发 onInsufficientBalance', async () => {
    vi.mocked(api.purchase).mockRejectedValue({ response: { status: 402, data: { code: 'INSUFFICIENT_BALANCE' } } });
    const onLow = vi.fn();
    render(<FixedPricePurchaseModal item={item} liveStreamId={1001} open={true} onClose={() => {}} onSuccess={() => {}} onInsufficientBalance={onLow} />);
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));
    await waitFor(() => expect(onLow).toHaveBeenCalled());
  });

  it('售罄 409 SOLD_OUT：Toast 提示', async () => {
    vi.mocked(api.purchase).mockRejectedValue({ response: { status: 409, data: { code: 'SOLD_OUT' } } });
    render(<FixedPricePurchaseModal item={item} liveStreamId={1001} open={true} onClose={() => {}} onSuccess={() => {}} />);
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));
    await waitFor(() => expect(screen.getByText(/已售罄/)).toBeInTheDocument());
  });

  it('网络异常：自动重试 1 次后失败 → Toast', async () => {
    vi.mocked(api.purchase).mockRejectedValue(new Error('Network'));
    render(<FixedPricePurchaseModal item={item} liveStreamId={1001} open={true} onClose={() => {}} onSuccess={() => {}} />);
    fireEvent.click(screen.getByRole('button', { name: /确认抢购/ }));
    await waitFor(() => expect(api.purchase).toHaveBeenCalledTimes(2));
    expect(screen.getByText(/网络异常/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 4.2: 写实现（要点）**

实现要点：
- 单例 `idempotencyKey` 在 onClick 首次生成，后续重试复用
- 重试上限 1 次（即总尝试 2 次）
- loading 状态期间按钮禁用
- 401/403 不在此处理，由 axios interceptor 统一跳登录

- [ ] **Step 4.3: 跑测试通过**

Run: `cd frontend/h5 && pnpm vitest run src/components/FixedPricePurchaseModal`
Expected: PASS（4 cases）

- [ ] **Step 4.4: Commit**

```bash
git add frontend/h5/src/components/FixedPricePurchaseModal/
git commit -m "feat(fixed-price): h5 purchase modal with retry+402+409 (M3.T4)"
```

---

### Task 5: FixedPriceFlair 飘屏

**Files:**
- Create: `frontend/h5/src/components/FixedPriceFlair/index.tsx`
- Test: `frontend/h5/src/components/FixedPriceFlair/index.test.tsx`

> **职责：** 订阅 `fixed_price_flair` WS 消息 → 在屏幕中部右→左飞过 4s → 自动 unmount；同时最多 3 条堆叠。

- [ ] **Step 5.1: 写失败测试**

```tsx
import { render, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import FixedPriceFlair from './index';

vi.useFakeTimers();

it('收到 flair 后渲染并 4s 后消失', () => {
  const sock = { subscribe: vi.fn() };
  const { container } = render(<FixedPriceFlair socket={sock as any} />);
  // 模拟订阅回调被触发
  const cb = (sock.subscribe as any).mock.calls[0][1];
  act(() => cb({ payload: { buyer_nickname: 'Alice', product_title: '翡翠', price: '99.00' } }));
  expect(container.textContent).toContain('Alice');
  act(() => vi.advanceTimersByTime(4100));
  expect(container.textContent).not.toContain('Alice');
});

it('堆叠 ≤ 3 条', () => {
  const sock = { subscribe: vi.fn() };
  const { container } = render(<FixedPriceFlair socket={sock as any} />);
  const cb = (sock.subscribe as any).mock.calls[0][1];
  act(() => {
    for (let i = 0; i < 5; i++) {
      cb({ payload: { buyer_nickname: `U${i}`, product_title: 'X', price: '1.00' } });
    }
  });
  expect(container.querySelectorAll('[data-flair]')).toHaveLength(3);
});
```

- [ ] **Step 5.2: 写实现**

CSS keyframes `translateX(100vw → -100%)` + `opacity 0→1→1→0`，动画 4s linear。

- [ ] **Step 5.3: 跑测试通过 + Commit**

```bash
git add frontend/h5/src/components/FixedPriceFlair/
git commit -m "feat(fixed-price): h5 flair animation overlay (M3.T5)"
```

---

### Task 6: 挂载到 LiveStream 页面

**Files:**
- Modify: `frontend/h5/src/pages/LiveStream/index.tsx`

- [ ] **Step 6.1: 接入 hook + 组件**

```tsx
const { items } = useFixedPriceItems(liveStreamId);
const [modalItem, setModalItem] = useState<FixedPriceItem | null>(null);
const navigate = useNavigate();

return (
  <>
    {/* ...原有直播 UI... */}
    <div className={styles.fixedPriceList}>
      {items.map(it => (
        <FixedPriceCard key={it.id} item={it} onPurchase={() => setModalItem(it)} />
      ))}
    </div>
    {modalItem && (
      <FixedPricePurchaseModal
        item={modalItem} liveStreamId={liveStreamId} open
        onClose={() => setModalItem(null)}
        onSuccess={(orderId) => { setModalItem(null); navigate(`/order/${orderId}`); }}
        onInsufficientBalance={() => navigate('/wallet/recharge')}
      />
    )}
    <FixedPriceFlair socket={liveStreamSocket} />
  </>
);
```

- [ ] **Step 6.2: 手动验证 + Commit**

启动 H5 dev server，连接 staging auction-service，触发管理端上架 + 用户购买，确认 UI 流转 4 种结果均符合 spec §4.6。

```bash
git add frontend/h5/src/pages/LiveStream/index.tsx
git commit -m "feat(fixed-price): mount h5 components into LiveStream page (M3.T6)"
```

---

### Task 7: 管理端上下架页面

**Files:**
- Create: `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx`

> **职责：** 直播间维度的 SKU 列表 + 「新增上架」表单（product_id / 价格 / 库存）+ 「下架」按钮。

- [ ] **Step 7.1: 列表 + 表单实现**

调用：
- `GET /api/v1/admin/live-streams/:id/fixed-price-items`（透传，无 X-User-ID 过滤）
- `POST /api/v1/admin/live-streams/:id/fixed-price-items`（上架）
- `POST /api/v1/admin/fixed-price-items/:id/offline`（下架）

复用 admin 现有 `Form` / `Table` 组件，分页 key 一致用 `items`。

- [ ] **Step 7.2: 单测覆盖核心场景**

- 上架成功 → 列表新增一行
- 下架确认弹窗 → 调用 API → 行 status 变 offline

Run: `cd frontend/admin && pnpm vitest run src/pages/LiveStreamFixedPrice`
Expected: PASS

- [ ] **Step 7.3: Commit**

```bash
git add frontend/admin/src/pages/LiveStreamFixedPrice/
git commit -m "feat(fixed-price): admin list+offline page (M3.T7)"
```

---

### Task 8: Prometheus 指标埋点

**Files:**
- Create: `backend/auction/metrics/fixed_price_metrics.go`
- Modify: `backend/auction/service/fixed_price.go`、`backend/auction/service/fixed_price_broadcaster.go`

> **指标列表（spec §6）：**
> - `fixed_price_purchase_total{result="success|sold_out|insufficient_balance|duplicate|other"}` - Counter
> - `fixed_price_purchase_latency_seconds{stage="lua|db|total"}` - Histogram，buckets: 0.005/0.01/0.025/0.05/0.1/0.25/0.5/1
> - `fixed_price_stock_remaining{item_id}` - Gauge
> - `fixed_price_ws_publish_total{type}` - Counter
> - `fixed_price_compensation_total{reason}` - Counter

- [ ] **Step 8.1: 定义 metrics 单例**

```go
// backend/auction/metrics/fixed_price_metrics.go
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    PurchaseTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "fixed_price_purchase_total"},
        []string{"result"})
    PurchaseLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "fixed_price_purchase_latency_seconds",
            Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
        },
        []string{"stage"})
    StockRemaining = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{Name: "fixed_price_stock_remaining"},
        []string{"item_id"})
    WSPublishTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "fixed_price_ws_publish_total"},
        []string{"type"})
    CompensationTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "fixed_price_compensation_total"},
        []string{"reason"})
)

func init() {
    prometheus.MustRegister(PurchaseTotal, PurchaseLatency, StockRemaining, WSPublishTotal, CompensationTotal)
}
```

- [ ] **Step 8.2: 埋点位置**

| 位置 | 指标 |
|---|---|
| service.Purchase 入口 | start := time.Now() |
| Lua 调用前后 | PurchaseLatency.WithLabelValues("lua").Observe(...) |
| DB 写入前后 | PurchaseLatency.WithLabelValues("db").Observe(...) |
| 函数返回 | PurchaseTotal.WithLabelValues(result).Inc() + PurchaseLatency("total") |
| broadcaster.Publish* 内 | WSPublishTotal.WithLabelValues(type).Inc() |
| 补偿事务执行 | CompensationTotal.WithLabelValues(reason).Inc() |
| outbox handler 处理 stock 事件 | StockRemaining.WithLabelValues(itemID).Set(remaining) |

- [ ] **Step 8.3: 集成测试断言指标值**

```go
func TestPurchase_EmitsMetrics(t *testing.T) {
    // ... 触发一次成功购买 ...
    val := testutil.ToFloat64(metrics.PurchaseTotal.WithLabelValues("success"))
    assert.Equal(t, 1.0, val)
}
```

Run: `cd backend/auction && go test ./service/ -run TestPurchase_EmitsMetrics -v`

- [ ] **Step 8.4: Commit**

```bash
git add backend/auction/metrics/ backend/auction/service/fixed_price*.go
git commit -m "feat(fixed-price): prometheus metrics for purchase+ws+compensation (M3.T8)"
```

---

### Task 9: Grafana Dashboard

**Files:**
- Create: `ops/grafana/fixed-price-dashboard.json`

> **面板（spec §6）：**
> - Row 1: 抢购成功率（rate(success) / rate(total)） + QPS（rate(total)）
> - Row 2: P50/P95/P99 延迟（histogram_quantile by stage）
> - Row 3: stock 单调递减热图（per item_id）
> - Row 4: WS 推送速率（by type） + 补偿事务速率（by reason）

- [ ] **Step 9.1: 用 Grafana UI 搭建后导出 JSON**

或直接手写 JSON（参考现有 `ops/grafana/auction-dashboard.json` 模板）。

- [ ] **Step 9.2: 验证 alerting rules**

定义两条告警（spec §6）：
- `fixed_price_purchase_p99 > 200ms`（5min）
- `rate(fixed_price_compensation_total[5m]) > 0.01`（补偿率 > 1%）

加入 `ops/prometheus/alert-rules/fixed-price.yml`。

- [ ] **Step 9.3: Commit**

```bash
git add ops/grafana/fixed-price-dashboard.json ops/prometheus/alert-rules/fixed-price.yml
git commit -m "feat(fixed-price): grafana dashboard + alert rules (M3.T9)"
```

---

### Task 10: E2E 烟雾测试 + 验收

**Files:**
- Create: `tests/e2e/fixed-price.spec.ts`（Playwright 或现有 e2e 框架）

- [ ] **Step 10.1: 编写脚本（happy + 边缘）**

场景：
1. 登录用户 A → 进直播间 → 看到卡片 → 点抢购 → 看到「抢到了」Toast → 跳订单详情
2. 用户 B 余额不足 → 抢购 → 弹「去充值」Modal
3. 主播下架 → A 屏幕上卡片消失（≤ 1s）
4. 主播再次上架 → A 看到卡片 + 飘屏前一买家

- [ ] **Step 10.2: 跑通 + Commit**

```bash
git add tests/e2e/fixed-price.spec.ts
git commit -m "test(fixed-price): e2e smoke for happy+edge paths (M3.T10)"
```

---

## M3 验收标准

- [ ] H5 单测：API / hook / 4 个组件 全绿（覆盖率 ≥ 80%）
- [ ] Admin 单测：上下架页面通过
- [ ] E2E：4 个核心场景全绿（在 staging 跑）
- [ ] 监控：在 Grafana 上能看到 5 类指标实时数据；触发 1 次余额不足，看到 `fixed_price_purchase_total{result="insufficient_balance"}` +1
- [ ] 告警：手动触发 P99 > 200ms（用 Toxiproxy 注入延迟），10min 内收到 alert
- [ ] UAT：spec §4.6 4 种结果在真机上 UI 表现符合预期
- [ ] PR：分别提交 H5 / Admin / Backend metrics / Ops 4 个 PR；每个 PR 触发 CI 全绿后再 squash merge

---

## Out of Scope（推迟到 M4+）

- 多 SKU 一次抢购（spec §11）
- 倒计时上架 / 限时秒杀
- 直播间维度的全局售罄统计
- 离线消息重放（用户网络中断恢复后补 stock 状态）

---

## Cross-M Dependencies 提醒

- 任何 M3 任务在没有 M1+M2 全绿前不要开工；如果 M2 stock 节流 P95 > 500ms，需先优化 M2 再做 M3 监控（否则告警噪声大）
- 管理端 Task 7 不依赖 H5 Task 1-6，可并行
- 监控 Task 8-9 不依赖 H5/Admin，可并行
