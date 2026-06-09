# H5 Profile Center Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the selected A方案 UI for the H5 profile center, including a bare identity header, auction command card, localStorage-based live-room footprints, and compact service grid.

**Architecture:** Keep the profile page as the composition root while extracting footprint persistence into a small utility module. Live-room entry records a normalized footprint after the room data is available; the profile page only reads the latest local records and renders them. All styling stays in CSS modules and uses existing H5 semantic tokens for dark/light themes.

**Tech Stack:** React 18, TypeScript, CSS Modules, React Router, Jest, Testing Library, browser `localStorage`.

---

## File Map

- Create: `frontend/h5/src/utils/liveRoomFootprints.ts`，负责 localStorage 读写、去重、置顶、截断。
- Create: `frontend/h5/src/utils/__tests__/liveRoomFootprints.test.ts`，覆盖足迹纯逻辑。
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`，在进入直播间时写入足迹。
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`，覆盖进房写足迹。
- Modify: `frontend/h5/src/pages/User/Index.tsx`，实现 A 方案结构。
- Modify: `frontend/h5/src/pages/User/Profile.module.css`，实现 A 方案视觉和双主题 token 适配。
- Modify: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`，更新 UI 断言。

---

## Task 1: Footprint Storage Utility

**Files:**
- Create: `frontend/h5/src/utils/liveRoomFootprints.ts`
- Create: `frontend/h5/src/utils/__tests__/liveRoomFootprints.test.ts`

- [ ] **Step 1: Write failing tests**

Create `frontend/h5/src/utils/__tests__/liveRoomFootprints.test.ts`:

```ts
import {
  LIVE_ROOM_FOOTPRINTS_KEY,
  getLiveRoomFootprints,
  recordLiveRoomFootprint,
} from '../liveRoomFootprints';

describe('liveRoomFootprints', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.spyOn(Date, 'now').mockReturnValue(1781020000000);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('records a normalized live room footprint', () => {
    recordLiveRoomFootprint({
      live_stream_id: 3,
      name: '玉石夜拍',
      cover: 'https://example.com/cover.jpg',
    });

    expect(getLiveRoomFootprints()).toEqual([
      {
        live_stream_id: 3,
        name: '玉石夜拍',
        cover: 'https://example.com/cover.jpg',
        enteredAt: 1781020000000,
      },
    ]);
  });

  it('deduplicates by live_stream_id and moves the latest entry to the top', () => {
    recordLiveRoomFootprint({ live_stream_id: 1, name: '旧直播', cover: '' });
    jest.spyOn(Date, 'now').mockReturnValue(1781020005000);
    recordLiveRoomFootprint({ live_stream_id: 2, name: '新直播', cover: '' });
    jest.spyOn(Date, 'now').mockReturnValue(1781020010000);
    recordLiveRoomFootprint({ live_stream_id: 1, name: '旧直播更新', cover: 'next.jpg' });

    expect(getLiveRoomFootprints().map((item) => item.live_stream_id)).toEqual([1, 2]);
    expect(getLiveRoomFootprints()[0]).toMatchObject({
      live_stream_id: 1,
      name: '旧直播更新',
      cover: 'next.jpg',
      enteredAt: 1781020010000,
    });
  });

  it('keeps only the latest 10 records', () => {
    for (let i = 1; i <= 12; i += 1) {
      jest.spyOn(Date, 'now').mockReturnValue(1781020000000 + i);
      recordLiveRoomFootprint({ live_stream_id: i, name: `直播 ${i}`, cover: '' });
    }

    const records = getLiveRoomFootprints();
    expect(records).toHaveLength(10);
    expect(records[0].live_stream_id).toBe(12);
    expect(records[9].live_stream_id).toBe(3);
  });

  it('fails closed when stored JSON is invalid', () => {
    localStorage.setItem(LIVE_ROOM_FOOTPRINTS_KEY, '{bad json');
    expect(getLiveRoomFootprints()).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npm test -- liveRoomFootprints --runInBand
```

Expected: FAIL because `../liveRoomFootprints` does not exist.

- [ ] **Step 3: Implement minimal utility**

Create `frontend/h5/src/utils/liveRoomFootprints.ts`:

```ts
export const LIVE_ROOM_FOOTPRINTS_KEY = 'h5.liveRoomFootprints';
const FOOTPRINT_LIMIT = 10;

export interface LiveRoomFootprint {
  live_stream_id: number;
  name: string;
  cover: string;
  enteredAt: number;
}

export type LiveRoomFootprintInput = Omit<LiveRoomFootprint, 'enteredAt'>;

function canUseLocalStorage() {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function normalizeRecord(value: unknown): LiveRoomFootprint | null {
  if (!value || typeof value !== 'object') return null;
  const record = value as Partial<LiveRoomFootprint>;
  const liveStreamID = Number(record.live_stream_id);
  const enteredAt = Number(record.enteredAt);
  if (!Number.isFinite(liveStreamID) || liveStreamID <= 0 || !Number.isFinite(enteredAt)) return null;
  return {
    live_stream_id: liveStreamID,
    name: String(record.name || '直播间'),
    cover: String(record.cover || ''),
    enteredAt,
  };
}

export function getLiveRoomFootprints(): LiveRoomFootprint[] {
  if (!canUseLocalStorage()) return [];
  try {
    const raw = window.localStorage.getItem(LIVE_ROOM_FOOTPRINTS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map(normalizeRecord)
      .filter((record): record is LiveRoomFootprint => Boolean(record))
      .sort((a, b) => b.enteredAt - a.enteredAt)
      .slice(0, FOOTPRINT_LIMIT);
  } catch {
    return [];
  }
}

export function recordLiveRoomFootprint(input: LiveRoomFootprintInput) {
  if (!canUseLocalStorage()) return;
  const liveStreamID = Number(input.live_stream_id);
  if (!Number.isFinite(liveStreamID) || liveStreamID <= 0) return;

  const nextRecord: LiveRoomFootprint = {
    live_stream_id: liveStreamID,
    name: input.name || '直播间',
    cover: input.cover || '',
    enteredAt: Date.now(),
  };

  const records = [
    nextRecord,
    ...getLiveRoomFootprints().filter((record) => record.live_stream_id !== liveStreamID),
  ].slice(0, FOOTPRINT_LIMIT);

  try {
    window.localStorage.setItem(LIVE_ROOM_FOOTPRINTS_KEY, JSON.stringify(records));
  } catch {
    // localStorage may be full or disabled; footprints are optional UI state.
  }
}
```

- [ ] **Step 4: Verify tests pass**

Run:

```bash
cd frontend/h5 && npm test -- liveRoomFootprints --runInBand
```

Expected: PASS.

---

## Task 2: Record Footprints On Live Room Entry

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`

- [ ] **Step 1: Write failing component test**

In `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`, import the utility mock:

```ts
import { recordLiveRoomFootprint } from '../../../utils/liveRoomFootprints';
```

Add mock near existing mocks:

```ts
jest.mock('../../../utils/liveRoomFootprints', () => ({
  recordLiveRoomFootprint: jest.fn(),
}));
```

Add typed mock:

```ts
const mockedRecordLiveRoomFootprint = recordLiveRoomFootprint as jest.MockedFunction<typeof recordLiveRoomFootprint>;
```

Add test:

```tsx
it('records a local footprint when entering an active live room', async () => {
  renderSlide();

  await waitFor(() => expect(mockedLiveStreamApi.get).toHaveBeenCalledWith(3));

  await waitFor(() =>
    expect(mockedRecordLiveRoomFootprint).toHaveBeenCalledWith({
      live_stream_id: 3,
      name: '测试直播间',
      cover: 'https://example.com/product.jpg',
    })
  );
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npm test -- LiveRoomSlide --runInBand
```

Expected: FAIL because `LiveRoomSlide` does not call `recordLiveRoomFootprint`.

- [ ] **Step 3: Implement minimal recording**

Modify imports in `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`:

```ts
import { recordLiveRoomFootprint } from '../../utils/liveRoomFootprints';
```

Add effect after `liveCoverImage` / `roomName` are derived:

```ts
useEffect(() => {
  if (!active || effectiveLiveStreamId <= 0 || !liveStream) return;
  recordLiveRoomFootprint({
    live_stream_id: effectiveLiveStreamId,
    name: roomName,
    cover: liveCoverImage,
  });
}, [active, effectiveLiveStreamId, liveStream, liveCoverImage, roomName]);
```

- [ ] **Step 4: Verify tests pass**

Run:

```bash
cd frontend/h5 && npm test -- LiveRoomSlide --runInBand
```

Expected: PASS.

---

## Task 3: Profile Page A方案 Structure

**Files:**
- Modify: `frontend/h5/src/pages/User/Index.tsx`
- Modify: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`

- [ ] **Step 1: Update failing profile tests**

In `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`, import footprint helper:

```ts
import { getLiveRoomFootprints } from '../../../utils/liveRoomFootprints';
```

Add mock:

```ts
jest.mock('../../../utils/liveRoomFootprints', () => ({
  getLiveRoomFootprints: jest.fn(),
}));
```

Add typed mock:

```ts
const mockedGetLiveRoomFootprints = getLiveRoomFootprints as jest.MockedFunction<typeof getLiveRoomFootprints>;
```

Set default footprints in `beforeEach`:

```ts
mockedGetLiveRoomFootprints.mockReturnValue([
  {
    live_stream_id: 3,
    name: '玉石夜拍',
    cover: 'https://example.com/live.jpg',
    enteredAt: 1781020000000,
  },
]);
```

Update the first test expectations:

```tsx
expect(screen.getByRole('link', { name: /竞拍记录/ })).toHaveAttribute('href', '/history');
expect(screen.queryByRole('link', { name: /2\s*中标/ })).not.toBeInTheDocument();
expect(screen.getByText('玉石夜拍')).toBeInTheDocument();
expect(screen.getByRole('link', { name: /个人卖家申请/ })).toHaveAttribute('href', '/');
expect(screen.getByRole('link', { name: /企业商家入驻/ })).toHaveAttribute('href', '/');
```

- [ ] **Step 2: Run profile tests to verify failure**

Run:

```bash
cd frontend/h5 && npm test -- Profile --runInBand
```

Expected: FAIL because Profile still renders the old stats grid, menu, and no footprint section.

- [ ] **Step 3: Implement A方案 JSX**

Modify `frontend/h5/src/pages/User/Index.tsx`:

```tsx
import { getLiveRoomFootprints } from '../../utils/liveRoomFootprints';
```

Add:

```tsx
const footprints = useMemo(() => getLiveRoomFootprints(), []);
const pendingAuctionCount = wonNotPaid;
```

Replace old stats/wallet/order/menu body with:

```tsx
<section className={styles.auctionCommandCard} aria-label="我的竞拍">
  <div className={styles.sectionHeader}>
    <div>
      <p className={styles.cardLabel}>Auction</p>
      <h2>我的竞拍</h2>
    </div>
    <span>记录含中标</span>
  </div>
  <Link to="/history" className={styles.primaryAuctionCta} onClick={trackAuctionHistoryClick}>
    <div>
      <strong>{pendingAuctionCount > 0 ? `${pendingAuctionCount} 件中标待支付` : '查看竞拍记录'}</strong>
      <span>从竞拍记录查看全部中标与出价</span>
    </div>
    <b>›</b>
  </Link>
  <div className={styles.auctionMetrics}>
    <Link to="/history" className={styles.metricCard} onClick={trackAuctionHistoryClick}>
      {pendingAuctionCount > 0 && <BadgeDot count={pendingAuctionCount} className={styles.metricBadge} />}
      <strong>{statValue(stats?.auction_history_count)}</strong>
      <span>竞拍记录</span>
    </Link>
    <div className={styles.metricCard} aria-label="中标数量">
      <strong>{statValue(stats?.won_count)}</strong>
      <span>中标</span>
    </div>
    <Link to="/following" className={styles.metricCard}>
      <strong>{statValue(stats?.following_count)}</strong>
      <span>收藏</span>
    </Link>
  </div>
</section>
```

Add footprint and service sections according to the final preview:

```tsx
<section className={styles.footprintCard} aria-label="最近浏览直播间">
  <div className={styles.sectionHeader}>
    <div>
      <p className={styles.cardLabel}>Footprints</p>
      <h2>足迹</h2>
    </div>
    <span>最近 10 个直播间</span>
  </div>
  {footprints.length > 0 ? (
    <div className={styles.footprintList}>
      {footprints.map((item) => (
        <Link key={item.live_stream_id} to={`/live?live_stream_id=${item.live_stream_id}`} className={styles.footprintItem}>
          <div className={styles.footprintCover} style={item.cover ? { backgroundImage: `url(${item.cover})` } : undefined} />
          <strong>{item.name}</strong>
          <span>最近浏览</span>
        </Link>
      ))}
    </div>
  ) : (
    <p className={styles.emptyText}>暂无直播间浏览足迹</p>
  )}
</section>

<section className={styles.serviceGrid} aria-label="账户与服务">
  <Link to="/orders" className={styles.serviceItem}>
    <span className={styles.serviceIcon}>¥</span>
    <span className={styles.serviceText}><strong>钱包</strong><small>{balance ? `可用 ${formatCurrency(pickAvailable(balance))}` : '可用 ¥0'}</small></span>
  </Link>
  <Link to="/addresses" className={styles.serviceItem}>
    <span className={styles.serviceIcon}>D</span>
    <span className={styles.serviceText}><strong>收货地址</strong><small>管理配送</small></span>
  </Link>
  <Link to="/" className={styles.serviceItem}>
    <span className={styles.newBadge}>新</span>
    <span className={styles.serviceIcon}>S</span>
    <span className={styles.serviceText}><strong>个人卖家申请</strong><small>暂未开放</small></span>
  </Link>
  <Link to="/" className={styles.serviceItem}>
    <span className={styles.newBadge}>新</span>
    <span className={styles.serviceIcon}>B</span>
    <span className={styles.serviceText}><strong>企业商家入驻</strong><small>暂未开放</small></span>
  </Link>
</section>
```

- [ ] **Step 4: Verify profile tests pass**

Run:

```bash
cd frontend/h5 && npm test -- Profile --runInBand
```

Expected: PASS.

---

## Task 4: Profile A方案 Styling

**Files:**
- Modify: `frontend/h5/src/pages/User/Profile.module.css`

- [ ] **Step 1: Implement CSS modules**

Update `frontend/h5/src/pages/User/Profile.module.css`:

```css
.auctionCommandCard,
.footprintCard {
  margin-top: var(--spacing-5);
  border: 1px solid var(--card-border-accent);
  border-radius: 24px;
  background: var(--bg-surface);
  box-shadow: var(--shadow-key);
  padding: var(--spacing-5);
}

.primaryAuctionCta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-3);
  border-radius: 20px;
  background: var(--gradient-accent);
  color: var(--text-inverse);
  padding: var(--spacing-4);
}

.primaryAuctionCta strong,
.primaryAuctionCta span {
  display: block;
}

.primaryAuctionCta span {
  margin-top: 4px;
  color: rgba(255, 255, 255, 0.78);
  font-size: var(--font-size-xs);
}

.primaryAuctionCta b {
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.18);
  padding: 6px 10px;
  font-size: 20px;
  line-height: 1;
}

.auctionMetrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--spacing-3);
  margin-top: var(--spacing-4);
}

.metricCard {
  position: relative;
  display: flex;
  min-height: 70px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 18px;
  background: var(--item-subtle-bg);
  color: var(--text-primary);
}

.metricBadge {
  top: -8px;
  right: -7px;
}

.metricCard strong {
  color: var(--text-brand);
  font-size: var(--font-size-lg);
}

.metricCard span {
  margin-top: 2px;
  color: var(--text-secondary);
  font-size: 11px;
}

.footprintList {
  display: flex;
  gap: var(--spacing-3);
  overflow-x: auto;
  padding-bottom: 2px;
}

.footprintItem {
  width: 92px;
  flex: 0 0 auto;
  color: var(--text-primary);
}

.footprintCover {
  height: 62px;
  border-radius: 16px;
  background:
    radial-gradient(circle at 70% 20%, rgba(201, 169, 110, 0.30), transparent 32%),
    var(--item-subtle-bg);
  background-position: center;
  background-size: cover;
}

.footprintItem strong {
  display: block;
  overflow: hidden;
  margin-top: var(--spacing-2);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.footprintItem span {
  color: var(--text-secondary);
  font-size: 10px;
}

.serviceGrid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--spacing-3);
  margin-top: var(--spacing-5);
}

.serviceItem {
  position: relative;
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  column-gap: var(--spacing-3);
  border: 1px solid var(--card-border-accent);
  border-radius: 18px;
  background: var(--bg-surface);
  color: var(--text-primary);
  padding: var(--spacing-3);
  box-shadow: var(--shadow-key);
}

.serviceIcon {
  display: inline-flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: var(--icon-tile-bg);
  color: var(--text-brand);
  font-weight: var(--font-weight-bold);
}

.serviceText {
  min-width: 0;
}

.serviceText strong,
.serviceText small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.serviceText strong {
  font-size: var(--font-size-sm);
}

.serviceText small {
  margin-top: 3px;
  color: var(--text-secondary);
  font-size: 10px;
}

.newBadge {
  position: absolute;
  top: -8px;
  right: -7px;
  display: inline-flex;
  min-width: 20px;
  height: 20px;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--bg-page);
  border-radius: var(--radius-full);
  background: var(--color-accent-500);
  color: var(--text-inverse);
  font-size: 10px;
  font-weight: var(--font-weight-bold);
  line-height: 1;
}
```

- [ ] **Step 2: Remove obsolete selectors**

Delete or stop using old selectors for the removed layout:

```css
.statsGrid
.statCard
.walletCard
.orderCard
.menu
.menuItem
.menuIcon
.menuLabel
.menuBadge
.mutedItem
```

Keep shared selectors still used by the new page:

```css
.page
.hero
.avatarFrame
.avatar
.avatarFallback
.identity
.eyebrow
.cardLabel
.badges
.sectionHeader
.emptyText
.logoutButton
.statePage
.spinner
.errorText
.retryButton
```

- [ ] **Step 3: Run focused profile tests**

Run:

```bash
cd frontend/h5 && npm test -- Profile --runInBand
```

Expected: PASS.

---

## Task 5: Final Verification

**Files:**
- No new files.

- [ ] **Step 1: Run all focused tests**

Run:

```bash
cd frontend/h5 && npm test -- liveRoomFootprints Profile LiveRoomSlide --runInBand
```

Expected: PASS.

- [ ] **Step 2: Run H5 build**

Run:

```bash
cd frontend/h5 && npm run build
```

Expected: PASS.

- [ ] **Step 3: Diagnostics**

Run editor diagnostics for modified TSX/CSS files.

Expected: no newly introduced TypeScript or CSS diagnostics.

- [ ] **Step 4: Git status review**

Run:

```bash
git status --short
```

Expected: intentional files from this plan plus pre-existing unrelated changes only.

---

## Self-Review

- Spec coverage: bare header covered by Task 3/4; auction command card covered by Task 3/4; localStorage footprints covered by Task 1/2/3; service grid and badge positioning covered by Task 3/4; dark/light token usage covered by Task 4.
- Placeholder scan: no `TBD`, `TODO`, or unspecified implementation steps remain.
- Type consistency: `LiveRoomFootprint`, `recordLiveRoomFootprint`, `getLiveRoomFootprints`, and storage key names are consistent across tests and implementation.
