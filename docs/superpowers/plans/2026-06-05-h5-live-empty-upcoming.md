# H5 Live Empty Upcoming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the H5 `/live` empty text with an actionable upcoming-auction empty state that shows at most two nearest upcoming auctions, supports product reminder subscription, and falls back to “去首页看拍品”.

**Architecture:** Backend owns upcoming-auction query semantics through `GET /api/v1/auctions?upcoming=true&page=1&page_size=2`, routed through Gateway as today. H5 keeps `LiveFeedPage` responsible for feed orchestration, but moves empty-state UI into `LiveEmptyState` so the feed logic and visual empty state stay separated.

**Tech Stack:** Go 1.25, Hertz, GORM, React 18, TypeScript, CSS Modules, Jest, React Testing Library.

---

## File Structure

- Modify: `backend/auction/dao/auction.go`
  - Add `Upcoming bool` to `AuctionFilters`.
  - When `Upcoming` is true, query only pending auctions with `start_time > now` and order by `start_time ASC, id ASC`.
- Modify: `backend/auction/dao/auction_current_test.go`
  - Add DAO regression coverage for upcoming filtering, ordering, and limit behavior.
- Modify: `backend/auction/handler/auction.go`
  - Parse `upcoming=true` from query string and pass it into `ListParams`.
- Modify: `backend/auction/handler/auction_list.go`
  - Add `Upcoming bool` to `ListParams` and propagate it into `dao.AuctionFilters`.
- Modify: `backend/auction/handler/auction_list_test.go`
  - Add orchestration test proving `BuildAuctionListResponse` forwards `Upcoming`.
- Modify: `frontend/h5/src/services/api.ts`
  - Extend `auctionApi.list` params with `upcoming?: boolean`.
- Create: `frontend/h5/src/pages/Live/LiveEmptyState.tsx`
  - Render upcoming empty state and fallback empty state.
  - Own card click, subscribe click, and event bubbling boundaries.
- Modify: `frontend/h5/src/pages/Live/LiveFeedPage.tsx`
  - Fetch upcoming auctions and reminder state only when live feed has no active auction room.
  - Wire subscription and navigation behavior.
- Modify: `frontend/h5/src/pages/Live/Live.module.css`
  - Add day/night-compatible empty-state styles using existing theme tokens.
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx`
  - Cover upcoming state, fallback state, subscribe behavior, and card navigation.
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveLayoutCss.test.ts`
  - Lock empty-state styles to theme tokens.

## Task 1: Backend Upcoming Query Semantics

**Files:**
- Modify: `backend/auction/dao/auction.go`
- Test: `backend/auction/dao/auction_current_test.go`

- [ ] **Step 1: Write the failing DAO test**

Append this test to `backend/auction/dao/auction_current_test.go`:

```go
func TestListWithFiltersUpcomingReturnsFuturePendingByStartTime(t *testing.T) {
	db := newCurrentTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	now := time.Now()
	rows := []model.Auction{
		{ID: 401, ProductID: 401, Status: model.AuctionStatusPending, StartTime: now.Add(3 * time.Hour), EndTime: now.Add(4 * time.Hour)},
		{ID: 402, ProductID: 402, Status: model.AuctionStatusPending, StartTime: now.Add(30 * time.Minute), EndTime: now.Add(2 * time.Hour)},
		{ID: 403, ProductID: 403, Status: model.AuctionStatusPending, StartTime: now.Add(-30 * time.Minute), EndTime: now.Add(time.Hour)},
		{ID: 404, ProductID: 404, Status: model.AuctionStatusOngoing, StartTime: now.Add(15 * time.Minute), EndTime: now.Add(90 * time.Minute)},
		{ID: 405, ProductID: 405, Status: model.AuctionStatusPending, StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	got, total, err := dao.ListWithFilters(ctx, &AuctionFilters{Upcoming: true}, 1, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, got, 2)
	require.Equal(t, int64(402), got[0].ID)
	require.Equal(t, int64(405), got[1].ID)
}
```

- [ ] **Step 2: Run the DAO test and verify it fails**

Run:

```bash
cd backend/auction && go test ./dao -run TestListWithFiltersUpcomingReturnsFuturePendingByStartTime -count=1
```

Expected: compile failure containing `unknown field Upcoming in struct literal of type AuctionFilters`.

- [ ] **Step 3: Implement upcoming filtering and ordering**

In `backend/auction/dao/auction.go`, update `AuctionFilters`:

```go
type AuctionFilters struct {
	Status         *model.AuctionStatus
	LiveStreamID   *int64
	LiveStreamName string
	Search         string
	Upcoming       bool
	// ProductIDs 仅在 category_id 过滤时由 handler 层装填，
	// 来自 product-service /internal/products?category_id= 的 id 列表（spec C §5.2）。
	// 为空切片表示无命中（应由调用方提前短路），nil 表示未过滤。
	ProductIDs []int64
}
```

In `ListWithFilters`, after `query := ...`, add upcoming handling before the existing status filter:

```go
	if filters.Upcoming {
		query = query.
			Where("status = ?", model.AuctionStatusPending).
			Where("start_time > ?", time.Now())
	} else if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
```

Replace the existing status block:

```go
	// 状态筛选
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
```

with the code above. Then replace the final query execution:

```go
	err := orderByAuctionFeedPriority(query).
		Offset(offset).
		Limit(pageSize).
		Find(&auctions).Error
```

with:

```go
	listQuery := query
	if filters.Upcoming {
		listQuery = listQuery.Order("start_time ASC, id ASC")
	} else {
		listQuery = orderByAuctionFeedPriority(listQuery)
	}
	err := listQuery.
		Offset(offset).
		Limit(pageSize).
		Find(&auctions).Error
```

- [ ] **Step 4: Run the DAO test and verify it passes**

Run:

```bash
cd backend/auction && go test ./dao -run TestListWithFiltersUpcomingReturnsFuturePendingByStartTime -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add backend/auction/dao/auction.go backend/auction/dao/auction_current_test.go
git commit -m "feat: add upcoming auction query semantics"
```

## Task 2: Backend Handler Contract

**Files:**
- Modify: `backend/auction/handler/auction.go`
- Modify: `backend/auction/handler/auction_list.go`
- Test: `backend/auction/handler/auction_list_test.go`

- [ ] **Step 1: Write the failing handler orchestration test**

Append this subtest inside `TestBuildAuctionListResponse` in `backend/auction/handler/auction_list_test.go`:

```go
	t.Run("upcoming flag is forwarded to auction filters", func(t *testing.T) {
		now := time.Now()
		fl := &fakeLister{
			out: []model.Auction{
				{ID: 300, ProductID: 77, Status: model.AuctionStatusPending, CurrentPrice: decimal.NewFromInt(1200), StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
			},
			outTotal: 1,
		}
		fp := &fakeProductClient{
			batchOut: map[int64]client.ProductSummary{
				77: {ID: 77, Name: "upcoming product", Images: []string{"u77"}},
			},
		}

		items, total, err := BuildAuctionListResponse(ctx, fp, fl.List, ListParams{Upcoming: true, Page: 1, PageSize: 2})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, items, 1)
		require.NotNil(t, fl.gotFilters)
		assert.True(t, fl.gotFilters.Upcoming)
		assert.Nil(t, fl.gotFilters.Status)
		assert.Equal(t, 1, fl.gotPage)
		assert.Equal(t, 2, fl.gotPageSize)
	})
```

- [ ] **Step 2: Run the handler test and verify it fails**

Run:

```bash
cd backend/auction && go test ./handler -run TestBuildAuctionListResponse/upcoming -count=1
```

Expected: compile failure containing `unknown field Upcoming in struct literal of type ListParams`.

- [ ] **Step 3: Add `upcoming=true` parsing and forwarding**

In `backend/auction/handler/auction_list.go`, update `ListParams`:

```go
type ListParams struct {
	Status         *model.AuctionStatus
	LiveStreamID   *int64
	LiveStreamName string
	Search         string
	CategoryID     *int64
	Upcoming       bool
	Page           int
	PageSize       int
}
```

In `BuildAuctionListResponse`, update the filter construction:

```go
	filters := &dao.AuctionFilters{
		Status:         p.Status,
		LiveStreamID:   p.LiveStreamID,
		LiveStreamName: p.LiveStreamName,
		Search:         p.Search,
		Upcoming:       p.Upcoming,
	}
```

In `backend/auction/handler/auction.go`, read the query:

```go
	upcoming := c.Query("upcoming") == "true"
```

Add it to `params`:

```go
	params := ListParams{
		LiveStreamName: liveStreamName,
		Search:         search,
		Upcoming:       upcoming,
		Page:           page,
		PageSize:       pageSize,
	}
```

Update the legacy filter branch condition:

```go
	if statusStr != "" || liveStreamIDStr != "" || liveStreamName != "" || search != "" || upcoming {
```

Update the legacy filter object:

```go
		filters = &dao.AuctionFilters{
			Status:         params.Status,
			LiveStreamID:   params.LiveStreamID,
			LiveStreamName: liveStreamName,
			Search:         search,
			Upcoming:       upcoming,
		}
```

- [ ] **Step 4: Run backend handler and DAO tests**

Run:

```bash
cd backend/auction && go test ./handler -run TestBuildAuctionListResponse -count=1
cd backend/auction && go test ./dao -run 'TestListWithFiltersUpcomingReturnsFuturePendingByStartTime|TestListOrdersByLiveUpcomingEndedPriority' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add backend/auction/handler/auction.go backend/auction/handler/auction_list.go backend/auction/handler/auction_list_test.go
git commit -m "feat: expose upcoming auction list contract"
```

## Task 3: H5 API Contract and Empty-State Tests

**Files:**
- Modify: `frontend/h5/src/services/api.ts`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx`

- [ ] **Step 1: Update the Jest API mock and add failing empty-state tests**

In `frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx`, replace the API mock block with:

```tsx
jest.mock('@/services/api', () => ({
  liveStreamApi: {
    list: jest.fn(),
  },
  auctionApi: {
    list: jest.fn(),
  },
  productReminderApi: {
    list: jest.fn(),
    subscribe: jest.fn(),
  },
}));
```

Replace the router import:

```tsx
import { MemoryRouter } from 'react-router-dom';
```

with:

```tsx
import { MemoryRouter, useLocation } from 'react-router-dom';
```

Replace the API import:

```tsx
import { liveStreamApi } from '@/services/api';
```

with:

```tsx
import { auctionApi, liveStreamApi, productReminderApi } from '@/services/api';
```

Add auth mock near the other mocks:

```tsx
const mockAuthState = {
  isAuthenticated: true,
  loading: false,
};

jest.mock('@/store/authContext', () => ({
  useAuth: () => mockAuthState,
}));
```

Add these constants after `mockedLiveStreamApi`:

```tsx
const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>;
const mockedProductReminderApi = productReminderApi as jest.Mocked<typeof productReminderApi>;
```

Add a location probe above `renderFeed`:

```tsx
const LocationProbe = () => {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}{location.search}</div>;
};
```

Update `renderFeed`:

```tsx
const renderFeed = (entry: string) =>
  render(
    <MemoryRouter initialEntries={[entry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <LiveFeedPage />
      <LocationProbe />
    </MemoryRouter>
  );
```

In `beforeEach`, add:

```tsx
    mockAuthState.isAuthenticated = true;
    mockAuthState.loading = false;
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 2 });
    mockedProductReminderApi.list.mockResolvedValue({ items: [] });
    mockedProductReminderApi.subscribe.mockResolvedValue({ product_id: 501 });
```

Append these tests:

```tsx
  it('没有正在竞拍直播间时展示最近两条即将开播预告', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        { id: 21, product_id: 501, status: 0, current_price: 1200, start_time: '2026-06-05T20:00:00Z', product: { id: 501, name: '古董腕表专场' } },
        { id: 22, product_id: 502, status: 0, current_price: 680, start_time: '2026-06-05T21:30:00Z', product: { id: 502, name: '潮玩限量专场' } },
        { id: 23, product_id: 503, status: 0, current_price: 990, start_time: '2026-06-05T22:00:00Z', product: { id: 503, name: '不应展示的第三条' } },
      ],
      total: 3,
      page: 1,
      page_size: 2,
    });

    renderFeed('/live');

    expect(await screen.findByText('下一场竞拍正在准备')).toBeInTheDocument();
    expect(screen.getByText('古董腕表专场')).toBeInTheDocument();
    expect(screen.getByText('潮玩限量专场')).toBeInTheDocument();
    expect(screen.queryByText('不应展示的第三条')).not.toBeInTheDocument();
    expect(mockedAuctionApi.list).toHaveBeenCalledWith({ status: '0', upcoming: true, page: 1, page_size: 2 });
  });

  it('点击预告条目非按钮区域进入商品详情页', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    mockedAuctionApi.list.mockResolvedValue({
      list: [{ id: 21, product_id: 501, status: 0, current_price: 1200, start_time: '2026-06-05T20:00:00Z', product: { id: 501, name: '古董腕表专场' } }],
      total: 1,
      page: 1,
      page_size: 2,
    });

    renderFeed('/live');

    fireEvent.click(await screen.findByText('古董腕表专场'));
    expect(screen.getByTestId('location')).toHaveTextContent('/detail?id=21');
  });

  it('点击订阅按钮不触发行跳转', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    mockedAuctionApi.list.mockResolvedValue({
      list: [{ id: 21, product_id: 501, status: 0, current_price: 1200, start_time: '2026-06-05T20:00:00Z', product: { id: 501, name: '古董腕表专场' } }],
      total: 1,
      page: 1,
      page_size: 2,
    });

    renderFeed('/live');

    fireEvent.click(await screen.findByRole('button', { name: '订阅' }));
    await waitFor(() => expect(mockedProductReminderApi.subscribe).toHaveBeenCalledWith(501));
    expect(screen.getByTestId('location')).toHaveTextContent('/live');
  });

  it('无预告或预告接口失败时降级展示去首页看拍品', async () => {
    mockedLiveStreamApi.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 });
    mockedAuctionApi.list.mockRejectedValue(new Error('upstream down'));

    renderFeed('/live');

    expect(await screen.findByText('当前没有竞拍直播')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '去首页看拍品' })).toHaveAttribute('href', '/');
  });
```

- [ ] **Step 2: Run frontend tests and verify they fail**

Run:

```bash
cd frontend/h5 && npm test -- LiveFeedPage.test.tsx --runInBand
```

Expected: FAIL because `auctionApi` and `productReminderApi` are not used by `LiveFeedPage` yet, and the new empty-state text does not exist.

- [ ] **Step 3: Extend `auctionApi.list` params**

In `frontend/h5/src/services/api.ts`, change `auctionApi.list` signature:

```ts
  list: (params?: { status?: string; page?: number; page_size?: number; category_id?: number; upcoming?: boolean }) => {
```

Add this query serialization:

```ts
    if (params?.upcoming !== undefined) query.set('upcoming', String(params.upcoming));
```

- [ ] **Step 4: Commit API/test setup**

Run:

```bash
git add frontend/h5/src/services/api.ts frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx
git commit -m "test: cover h5 live upcoming empty state"
```

## Task 4: H5 Empty-State Component and Feed Integration

**Files:**
- Create: `frontend/h5/src/pages/Live/LiveEmptyState.tsx`
- Modify: `frontend/h5/src/pages/Live/LiveFeedPage.tsx`

- [ ] **Step 1: Create `LiveEmptyState.tsx`**

Create `frontend/h5/src/pages/Live/LiveEmptyState.tsx`:

```tsx
import React from 'react';
import { Link } from 'react-router-dom';
import styles from './Live.module.css';

export interface UpcomingAuctionItem {
  id: number;
  product_id?: number;
  status?: number;
  current_price?: number | string | null;
  start_time?: string;
  product?: {
    id?: number;
    name?: string;
    image?: string;
  };
}

interface LiveEmptyStateProps {
  upcomingAuctions: UpcomingAuctionItem[];
  subscribedProductIds: Set<number>;
  pendingProductId: number | null;
  onAuctionClick: (auctionId: number) => void;
  onSubscribe: (productId?: number, auctionId?: number) => void;
}

const formatStartTime = (value?: string) => {
  if (!value) return '即将';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '即将';
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false });
};

const formatPrice = (value?: number | string | null) => {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return '待公布';
  return `¥${n.toLocaleString('zh-CN')}`;
};

const getProductName = (auction: UpcomingAuctionItem) =>
  auction.product?.name || `竞拍场次 #${auction.id}`;

const LiveEmptyState: React.FC<LiveEmptyStateProps> = ({
  upcomingAuctions,
  subscribedProductIds,
  pendingProductId,
  onAuctionClick,
  onSubscribe,
}) => {
  const visibleAuctions = upcomingAuctions.slice(0, 2);

  if (visibleAuctions.length === 0) {
    return (
      <section className={styles.liveEmptyPage} aria-live="polite">
        <div className={styles.liveEmptyPanel}>
          <div className={styles.liveEmptyIcon} aria-hidden="true">
            <span className={styles.liveEmptyIconRing} />
            <span className={styles.liveEmptyIconCamera} />
          </div>
          <h1 className={styles.liveEmptyTitle}>当前没有竞拍直播</h1>
          <p className={styles.liveEmptyText}>可以先看看正在预热的拍品，开拍提醒会第一时间通知你。</p>
          <Link className={styles.liveEmptyPrimaryLink} to="/">去首页看拍品</Link>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.liveEmptyPage} aria-live="polite">
      <div className={styles.liveEmptyPanel}>
        <div className={styles.liveEmptyIcon} aria-hidden="true">
          <span className={styles.liveEmptyIconRing} />
          <span className={styles.liveEmptyIconCamera} />
        </div>
        <h1 className={styles.liveEmptyTitle}>下一场竞拍正在准备</h1>
        <p className={styles.liveEmptyText}>当前没有正在竞拍的直播间。先订阅感兴趣的预告场次，开拍前会提醒你回来。</p>
        <div className={styles.upcomingHeader}>即将开播</div>
        <div className={styles.upcomingList}>
          {visibleAuctions.map((auction) => {
            const productId = auction.product_id ?? auction.product?.id;
            const subscribed = productId ? subscribedProductIds.has(productId) : false;
            const pending = productId === pendingProductId;
            return (
              <article
                key={auction.id}
                className={styles.upcomingCard}
                role="button"
                tabIndex={0}
                onClick={() => onAuctionClick(auction.id)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    onAuctionClick(auction.id);
                  }
                }}
              >
                <div className={styles.upcomingTime}>{formatStartTime(auction.start_time)}</div>
                <div className={styles.upcomingInfo}>
                  <strong>{getProductName(auction)}</strong>
                  <span>起拍 {formatPrice(auction.current_price)} · 点击查看详情</span>
                </div>
                <button
                  type="button"
                  className={styles.upcomingSubscribe}
                  disabled={!productId || subscribed || pending}
                  onClick={(event) => {
                    event.stopPropagation();
                    onSubscribe(productId, auction.id);
                  }}
                >
                  {pending ? '订阅中...' : subscribed ? '已订阅' : '订阅'}
                </button>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
};

export default LiveEmptyState;
```

- [ ] **Step 2: Integrate empty-state data flow in `LiveFeedPage.tsx`**

Modify imports in `frontend/h5/src/pages/Live/LiveFeedPage.tsx`:

```tsx
import { auctionApi, liveStreamApi, productReminderApi } from '@/services/api';
import { useAuth } from '@/store/authContext';
import LiveEmptyState, { UpcomingAuctionItem } from './LiveEmptyState';
```

Add helper below `extractTotal`:

```tsx
const extractReminderProductIds = (response: any) =>
  new Set(
    extractList<{ product_id?: number; productId?: number }>(response)
      .map((item) => item.product_id ?? item.productId)
      .filter((id): id is number => typeof id === 'number')
  );
```

Inside `LiveFeedPage`, add auth and state:

```tsx
  const { isAuthenticated } = useAuth();
  const [upcomingAuctions, setUpcomingAuctions] = useState<UpcomingAuctionItem[]>([]);
  const [upcomingFailed, setUpcomingFailed] = useState(false);
  const [subscribedProductIds, setSubscribedProductIds] = useState<Set<number>>(() => new Set());
  const [reminderPendingProductId, setReminderPendingProductId] = useState<number | null>(null);
```

Add effect after `auctionRooms`:

```tsx
  const shouldLoadEmptyState = !loading && auctionRooms.length === 0;

  useEffect(() => {
    if (!shouldLoadEmptyState) return;
    let cancelled = false;
    setUpcomingFailed(false);
    auctionApi
      .list({ status: '0', upcoming: true, page: 1, page_size: 2 })
      .then((res) => {
        if (cancelled) return;
        setUpcomingAuctions(extractList<UpcomingAuctionItem>(res).slice(0, 2));
      })
      .catch(() => {
        if (cancelled) return;
        setUpcomingAuctions([]);
        setUpcomingFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [shouldLoadEmptyState]);

  useEffect(() => {
    if (!shouldLoadEmptyState || !isAuthenticated) {
      setSubscribedProductIds(new Set());
      return;
    }
    let cancelled = false;
    productReminderApi
      .list()
      .then((response) => {
        if (cancelled) return;
        setSubscribedProductIds(extractReminderProductIds(response));
      })
      .catch(() => {
        if (cancelled) return;
        setSubscribedProductIds(new Set());
      });
    return () => {
      cancelled = true;
    };
  }, [shouldLoadEmptyState, isAuthenticated]);
```

Add handlers before render branches:

```tsx
  const handleUpcomingClick = (auctionId: number) => {
    navigate(`/detail?id=${auctionId}`);
  };

  const handleSubscribeReminder = async (productId?: number) => {
    if (!productId) return;
    if (!isAuthenticated) {
      navigate(`/login?redirect=${encodeURIComponent('/live')}`);
      return;
    }
    setReminderPendingProductId(productId);
    try {
      await productReminderApi.subscribe(productId);
      setSubscribedProductIds((current) => {
        const next = new Set(current);
        next.add(productId);
        return next;
      });
    } catch (error: any) {
      if (typeof error?.message === 'string' && error.message.includes('已经订阅')) {
        setSubscribedProductIds((current) => {
          const next = new Set(current);
          next.add(productId);
          return next;
        });
      } else {
        showToast('订阅失败，请稍后重试');
      }
    } finally {
      setReminderPendingProductId(null);
    }
  };
```

Replace:

```tsx
  if (rooms.length === 0) {
    return <div>暂无直播中房间</div>;
  }

  if (auctionRooms.length === 0) {
    return <div>暂无正在竞拍的直播间</div>;
  }
```

with:

```tsx
  if (rooms.length === 0 || auctionRooms.length === 0) {
    return (
      <LiveEmptyState
        upcomingAuctions={upcomingFailed ? [] : upcomingAuctions}
        subscribedProductIds={subscribedProductIds}
        pendingProductId={reminderPendingProductId}
        onAuctionClick={handleUpcomingClick}
        onSubscribe={handleSubscribeReminder}
      />
    );
  }
```

- [ ] **Step 3: Run frontend tests**

Run:

```bash
cd frontend/h5 && npm test -- LiveFeedPage.test.tsx --runInBand
```

Expected: new behavior tests pass; existing feed navigation tests still pass.

- [ ] **Step 4: Commit**

Run:

```bash
git add frontend/h5/src/pages/Live/LiveEmptyState.tsx frontend/h5/src/pages/Live/LiveFeedPage.tsx frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx frontend/h5/src/services/api.ts
git commit -m "feat: add h5 live upcoming empty state"
```

## Task 5: Empty-State Styling and CSS Regression

**Files:**
- Modify: `frontend/h5/src/pages/Live/Live.module.css`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveLayoutCss.test.ts`

- [ ] **Step 1: Add failing CSS regression test**

Append to `frontend/h5/src/pages/Live/__tests__/LiveLayoutCss.test.ts`:

```ts
  it('uses theme tokens for the live empty upcoming state', () => {
    const css = readFileSync(join(__dirname, '..', 'Live.module.css'), 'utf8');
    const pageCss = css.match(/\.liveEmptyPage\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const titleCss = css.match(/\.liveEmptyTitle\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const primaryLinkCss = css.match(/\.liveEmptyPrimaryLink\s*\{[\s\S]*?\n\}/)?.[0] ?? '';
    const cardCss = css.match(/\.upcomingCard\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(pageCss).toContain('background: var(--bg-page);');
    expect(pageCss).toContain('color: var(--text-primary);');
    expect(titleCss).toContain('color: var(--text-primary);');
    expect(primaryLinkCss).toContain('color:');
    expect(cardCss).toContain('background: var(--bg-elevated);');
  });
```

- [ ] **Step 2: Run CSS test and verify it fails**

Run:

```bash
cd frontend/h5 && npm test -- LiveLayoutCss.test.ts --runInBand
```

Expected: FAIL because the new classes do not exist.

- [ ] **Step 3: Add styles**

Append to `frontend/h5/src/pages/Live/Live.module.css`:

```css
.liveEmptyPage {
  display: flex;
  width: 100%;
  min-height: calc(100vh - var(--live-bottom-nav-height));
  min-height: calc(100dvh - var(--live-bottom-nav-height));
  align-items: center;
  justify-content: center;
  padding: calc(var(--spacing-6) + env(safe-area-inset-top, 0px)) var(--spacing-5) calc(var(--spacing-8) + env(safe-area-inset-bottom, 0px));
  background: var(--bg-page);
  color: var(--text-primary);
}

.liveEmptyPanel {
  width: 100%;
  max-width: 360px;
  text-align: center;
}

.liveEmptyIcon {
  position: relative;
  width: 112px;
  height: 112px;
  margin: 0 auto var(--spacing-5);
  border: 1px solid rgba(201, 169, 110, 0.22);
  border-radius: 34px;
  background: var(--bg-elevated);
  box-shadow: var(--shadow-md);
}

.liveEmptyIconRing {
  position: absolute;
  inset: -10px;
  border: 1px dashed rgba(201, 169, 110, 0.34);
  border-radius: 40px;
}

.liveEmptyIconCamera {
  position: absolute;
  left: 27px;
  top: 38px;
  width: 58px;
  height: 38px;
  border-radius: 13px;
  background: #b88a2f;
  box-shadow: 0 10px 20px rgba(184, 138, 47, 0.24);
}

.liveEmptyIconCamera::before {
  position: absolute;
  left: 18px;
  top: 9px;
  width: 19px;
  height: 19px;
  border-radius: var(--radius-full);
  background: var(--bg-page);
  content: '';
}

.liveEmptyIconCamera::after {
  position: absolute;
  right: -13px;
  top: 10px;
  border-top: 9px solid transparent;
  border-bottom: 9px solid transparent;
  border-left: 16px solid #b88a2f;
  content: '';
}

.liveEmptyTitle {
  margin: 0 0 var(--spacing-3);
  color: var(--text-primary);
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-bold);
  letter-spacing: -0.04em;
}

.liveEmptyText {
  margin: 0 auto;
  max-width: 290px;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  line-height: 1.7;
}

.liveEmptyPrimaryLink {
  display: inline-flex;
  min-width: 172px;
  height: 46px;
  align-items: center;
  justify-content: center;
  margin-top: var(--spacing-6);
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, #b88a2f, #e0b767);
  color: #fffaf2;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-bold);
  text-decoration: none;
  box-shadow: 0 12px 24px rgba(184, 138, 47, 0.24);
}

.upcomingHeader {
  margin: var(--spacing-7) 0 var(--spacing-3);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-bold);
  text-align: left;
}

.upcomingList {
  display: grid;
  gap: var(--spacing-3);
}

.upcomingCard {
  display: grid;
  grid-template-columns: 48px 1fr auto;
  gap: var(--spacing-3);
  align-items: center;
  min-height: 78px;
  padding: var(--spacing-3);
  border: 1px solid rgba(201, 169, 110, 0.18);
  border-radius: var(--radius-xl);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
  box-shadow: var(--shadow-sm);
}

.upcomingTime {
  display: flex;
  width: 48px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  background: rgba(201, 169, 110, 0.12);
  color: var(--text-brand);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-bold);
}

.upcomingInfo {
  min-width: 0;
}

.upcomingInfo strong,
.upcomingInfo span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upcomingInfo strong {
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}

.upcomingInfo span {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
}

.upcomingSubscribe {
  min-width: 64px;
  height: 34px;
  border: none;
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, #b88a2f, #e0b767);
  color: #fffaf2;
  cursor: pointer;
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-bold);
}

.upcomingSubscribe:disabled {
  background: rgba(201, 169, 110, 0.14);
  color: var(--text-secondary);
  cursor: not-allowed;
  box-shadow: none;
}
```

- [ ] **Step 4: Run CSS and page tests**

Run:

```bash
cd frontend/h5 && npm test -- LiveLayoutCss.test.ts LiveFeedPage.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add frontend/h5/src/pages/Live/Live.module.css frontend/h5/src/pages/Live/__tests__/LiveLayoutCss.test.ts
git commit -m "style: polish h5 live empty upcoming state"
```

## Task 6: Business Event Reuse

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveFeedPage.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx`

This task does not add new event names or Gateway whitelist entries. It reuses existing `reminder_subscribe` with `source: 'live_room'` and metadata `entry: 'live_empty_upcoming'`.

- [ ] **Step 1: Add failing tracking assertion**

In `LiveFeedPage.test.tsx`, add this mock:

```tsx
const mockTrackBusinessEvent = jest.fn();
jest.mock('@/utils/businessEvent', () => ({
  trackBusinessEvent: (...args: any[]) => mockTrackBusinessEvent(...args),
}));
```

In the subscribe test, after the subscribe assertion, add:

```tsx
    expect(mockTrackBusinessEvent).toHaveBeenCalledWith('reminder_subscribe', {
      source: 'live_room',
      auctionId: 21,
      productId: 501,
      metadata: { entry: 'live_empty_upcoming' },
    });
```

In `beforeEach`, add:

```tsx
    mockTrackBusinessEvent.mockClear();
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
cd frontend/h5 && npm test -- LiveFeedPage.test.tsx --runInBand
```

Expected: FAIL because no business event is emitted yet.

- [ ] **Step 3: Emit existing business event on subscribe click**

In `LiveFeedPage.tsx`, import:

```tsx
import { trackBusinessEvent } from '@/utils/businessEvent';
```

Change handler signature:

```tsx
  const handleSubscribeReminder = async (productId?: number, auctionId?: number) => {
```

After successful `productReminderApi.subscribe(productId);`, add:

```tsx
      trackBusinessEvent('reminder_subscribe', {
        source: 'live_room',
        auctionId,
        productId,
        metadata: { entry: 'live_empty_upcoming' },
      });
```

Also add the same tracking call in the `"已经订阅"` branch after local state is updated, so duplicate subscriptions still produce a click-event record.

- [ ] **Step 4: Run frontend tests**

Run:

```bash
cd frontend/h5 && npm test -- LiveFeedPage.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add frontend/h5/src/pages/Live/LiveFeedPage.tsx frontend/h5/src/pages/Live/__tests__/LiveFeedPage.test.tsx
git commit -m "feat: track live empty reminder subscriptions"
```

## Task 7: Full Verification

**Files:**
- Verify only.

- [ ] **Step 1: Run backend targeted tests**

Run:

```bash
cd backend/auction && go test ./dao ./handler -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend targeted tests**

Run:

```bash
cd frontend/h5 && npm test -- LiveFeedPage.test.tsx LiveLayoutCss.test.ts --runInBand
```

Expected: PASS.

- [ ] **Step 3: Run frontend type/build check**

Run:

```bash
cd frontend/h5 && npm run build
```

Expected: TypeScript and Vite build complete successfully.

- [ ] **Step 4: Inspect final diff**

Run:

```bash
git status --short
git diff --stat HEAD~6..HEAD
```

Expected:

- Only intended source/test files are changed by the feature commits.
- Existing unrelated `frontend/h5/node_modules/.vite/deps/_metadata.json` may still appear dirty and must not be committed.

## Self-Review

Spec coverage:

- Empty state replaces pure text: Tasks 3-5.
- Shows at most two nearest upcoming auctions: Tasks 1-4.
- Subscribe button with `订阅/订阅中.../已订阅`: Task 4.
- Day/night token-safe styling: Task 5.
- Fallback to `去首页看拍品`: Tasks 3-4.
- Gateway-only traffic: Task 2 reuses `/api/v1/auctions`; Task 4 uses existing API client.
- No new “全部预告” entry: Task 4 component has no such link.

Placeholder scan: no placeholder tasks or deferred implementation steps.

Type consistency:

- Backend uses `Upcoming bool` consistently in `ListParams` and `dao.AuctionFilters`.
- Frontend uses `UpcomingAuctionItem` with `product_id` and `product.id` fallback.
- Subscribe callback signature is `(productId?: number, auctionId?: number)` in both component and page.
