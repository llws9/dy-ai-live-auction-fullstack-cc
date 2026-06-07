# H5 演示控制面板 (Demo Console) 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 H5 直播间端增加一个常驻的 AssistiveTouch 风格悬浮球演示控制面板，一键触发「切换身份 / 他人跟价 / 充值 / 并发压测 / 竞拍延时提示」，用于向评委做无操作负担的业务闭环演示。

**Architecture:** 前端在 `App.tsx` Provider 树内、`<Routes>` 同级挂载常驻浮层组件（不随路由卸载），通过 URL searchParams 跨组件读取当前 `auctionId`；后端在 `test-service` 的 `/api/test/*` 前缀下新增 demo 接口（gateway 已 `Any("/*path")` 自动透传，无需改 gateway），复用既有 auction SDK client 的 `PlaceBid`（X-User-ID + JWT）与 `/internal/test/user-balance` 充值能力。纯演示定位，不做环境隔离。

**Tech Stack:** Go (Hertz, GORM)、React 18 (Context, react-router-dom)、既有 auction SDK client、Jest（前端）、go test（后端）。

**前置依赖（均已在 main 就绪）：**
- 统一 seed：四个 138 账号（`13800138001` 买家A / `13800138004` 买家B / `13800138002` 商家 / `13800138003` 管理员，`Demo@123456`）已可登录（commit `02777b7f`/`603b1ca0`）。
- 防狙击延时可见：`delay_triggered` 广播链路已打通（merge `a3f696b1`），「竞拍延时」演示无需新增后端，靠手动出价触发真实延时。

**关键现状事实（实施前必读，纠正旧 spec 的错误假设）：**
1. test-service 路由前缀是 `/api/test`（[main.go:116](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go#L116)），**不是** `/api/v1/test/demo`；gateway 已透传（[router.go:212-213](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L212-L213)），新增接口零 gateway 改动。
2. 充值能力已存在：auction DAO `AddAmount` UPSERT（[user_balance.go:66-91](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/dao/user_balance.go#L66-L91)）+ 内部接口 `POST /internal/test/user-balance`（[main.go:481](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go#L481)）+ test 侧 client `TopUpUserBalance`（[client.go:442-460](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go#L442-L460)）。
3. 以买家B出价：复用 SDK client `PlaceBid`（[client.go:343-348](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go#L343-L348)），身份靠 `Actor.UserID`→`X-User-ID`+JWT（[client.go:539-552](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go#L539-L552)），用配 `SetJWTSecret` 的 `bizCli`（走 gateway）。
4. 前端 `services/api.ts` baseURL 写死 `/api/v1`（[api.ts:5](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L5)），demo 接口在 `/api/test/*`，需独立请求函数。
5. 全局浮层拿不到 LiveRoomSlide 内部 `auctionId` state，用 URL searchParams 跨组件传递。
6. 编程式切换身份：`useAuth().login({ phone, password })`（[authContext.tsx:42-54](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/store/authContext.tsx#L42-L54)）。
7. **本轮不做 AI 防作弊 R4**（功能未实现，已与用户确认）。

---

## 文件结构

**后端（test-service）：**
- 新建 `backend/test/handler/demo.go` — demo 接口 handler（follow-bid / recharge），同步执行非异步任务。
- 修改 `backend/test/main.go` — 构造 demo handler 并注册 `/api/test/demo/*` 路由；把已构造的 `bizCli`/`internalCli` 传入。
- 新建 `backend/test/handler/demo_test.go` — handler 单测。

**前端（h5）：**
- 新建 `frontend/h5/src/services/demoApi.ts` — `/api/test/*` 独立请求封装（不走 api.ts 的 /api/v1 base）。
- 新建 `frontend/h5/src/store/demoContext.tsx` — 跨组件共享「当前 auctionId」（供浮层读取 LiveRoomSlide 设置的值）。
- 新建 `frontend/h5/src/components/DemoConsole/index.tsx` — AssistiveTouch 浮球 + 扇形菜单组件。
- 新建 `frontend/h5/src/components/DemoConsole/DemoConsole.css` — 毛玻璃质感样式。
- 修改 `frontend/h5/src/App.tsx` — 挂载 `<DemoProvider>` 与 `<DemoConsole />`。
- 修改 `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` — 把当前 `auctionId` 写入 demoContext。
- 新建 `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx` — 组件交互测试。
- 新建 `frontend/h5/src/services/__tests__/demoApi.test.ts` — api 封装测试。

---

## 后端实施

### Task 1: demo follow-bid 接口（买家B 后台代出价）

**Files:**
- Create: `backend/test/handler/demo.go`
- Modify: `backend/test/main.go`（构造 + 注册路由）
- Test: `backend/test/handler/demo_test.go`

> 设计要点：follow-bid 是**同步**动作（不走 runner 异步任务）。handler 解析 `auction_id`，以买家B（`13800138004`，需其数据库 id）身份调 `bizCli.PlaceBid`。买家B 的数据库 id 由 seed 固定为 `9102`（见统一 seed spec）。出价金额策略：不传具体金额，由 handler 查当前价 + 步长算出「当前价 + 一档」；为降低实施复杂度，第一版接受前端传入 `amount`，缺省时回退一个安全增量。

- [ ] **Step 1: 写失败测试**

`backend/test/handler/demo.go` 需要一个不依赖真实 HTTP 的纯函数来计算跟价金额，先测它。创建 `backend/test/handler/demo_test.go`：

```go
package handler

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestComputeFollowBidAmount(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		incr     string
		override string
		want     string
	}{
		{"override wins", "100", "10", "500", "500"},
		{"current plus increment", "100", "10", "", "110"},
		{"zero current uses increment", "0", "5", "", "5"},
		{"empty increment defaults to 1", "100", "", "", "101"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cur, _ := decimal.NewFromString(c.current)
			incr := decimal.Zero
			if c.incr != "" {
				incr, _ = decimal.NewFromString(c.incr)
			}
			var override *decimal.Decimal
			if c.override != "" {
				v, _ := decimal.NewFromString(c.override)
				override = &v
			}
			got := computeFollowBidAmount(cur, incr, override)
			want, _ := decimal.NewFromString(c.want)
			if !got.Equal(want) {
				t.Fatalf("computeFollowBidAmount(%s,%s,%v)=%s want %s", c.current, c.incr, override, got, want)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/test && go test ./handler/ -run TestComputeFollowBidAmount -v`
Expected: FAIL（`undefined: computeFollowBidAmount`，编译失败）

- [ ] **Step 3: 写最小实现**

创建 `backend/test/handler/demo.go`：

```go
package handler

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/shopspring/decimal"

	auctioncli "test-service/client/auction"
)

// BuyerBUserID 是统一 seed 固定的「买家B」数据库 id（13800138004）。
const BuyerBUserID int64 = 9102

// DemoHandler 处理演示控制面板触发的同步业务动作。
type DemoHandler struct {
	bizCli *auctioncli.Client // 走 gateway，带 JWT，用于以指定用户身份出价
}

func NewDemoHandler(bizCli *auctioncli.Client) *DemoHandler {
	return &DemoHandler{bizCli: bizCli}
}

// computeFollowBidAmount 计算买家B 跟价金额：
// 优先用 override；否则取 current + increment（increment 缺省按 1 计）。
func computeFollowBidAmount(current, increment decimal.Decimal, override *decimal.Decimal) decimal.Decimal {
	if override != nil {
		return *override
	}
	if increment.IsZero() {
		increment = decimal.NewFromInt(1)
	}
	return current.Add(increment)
}

type followBidRequest struct {
	AuctionID int64    `json:"auction_id"`
	Amount    *float64 `json:"amount,omitempty"`
}

// PostFollowBid 以买家B 身份对指定拍卖发起一次跟价出价。
func (h *DemoHandler) PostFollowBid(ctx context.Context, c *app.RequestContext) {
	var req followBidRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil || req.AuctionID <= 0 {
		c.JSON(400, map[string]any{"error": "invalid auction_id"})
		return
	}
	var amount float64
	if req.Amount != nil {
		amount = *req.Amount
	}
	hlog.CtxInfof(ctx, "[demo] follow-bid auction=%d amount=%v as buyerB=%d", req.AuctionID, amount, BuyerBUserID)
	result := h.bizCli.PlaceBid(ctx, BuyerBUserID, req.AuctionID, amount)
	if !result.OK {
		c.JSON(400, map[string]any{"error": result.Error, "status": result.StatusCode})
		return
	}
	c.JSON(200, map[string]any{"ok": true, "auction_id": req.AuctionID, "amount": amount})
}
```

> 注：`StepResult` 字段名（`OK`/`Error`/`StatusCode`）以 [client.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go) 中 `StepResult` 的真实定义为准；实施时打开该文件确认字段后对齐。若 `PlaceBid` 要求 amount>0 而当前版本未查当前价，则第一版要求前端必传 `amount`（见前端 Task 6）。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/test && go test ./handler/ -run TestComputeFollowBidAmount -v`
Expected: PASS（4 个子用例全过）

- [ ] **Step 5: 注册路由**

在 [backend/test/main.go:115-129](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go#L115-L129) 路由注册区，复用已构造的 `bizCli`（见 [main.go:72-77](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go#L72-L77)），新增：

```go
demoHandler := handler.NewDemoHandler(bizCli)
demo := api.Group("/demo")
demo.POST("/follow-bid", demoHandler.PostFollowBid)
```

（`api` 即 [main.go:116](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go#L116) 的 `h.Group("/api/test")`，故最终路径为 `POST /api/test/demo/follow-bid`。）

- [ ] **Step 6: 编译验证**

Run: `cd backend/test && go build ./...`
Expected: 退出码 0，无报错。

- [ ] **Step 7: 提交**

```bash
git add backend/test/handler/demo.go backend/test/handler/demo_test.go backend/test/main.go
git commit -m "feat(test): add demo follow-bid endpoint for buyer B"
```

---

### Task 2: demo recharge 接口（给当前用户充值）

**Files:**
- Modify: `backend/test/handler/demo.go`（新增 handler）
- Modify: `backend/test/main.go`（注入 internalCli + 注册路由）
- Test: `backend/test/handler/demo_test.go`（新增校验测试）

> 设计要点：充值复用 auction `/internal/test/user-balance`，test 侧已有 `internalCli.TopUpUserBalance`（[client.go:442-460](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go#L442-L460)）。handler 接受 `{user_id, amount}`，校验后调 client。金额用 string 传递避免浮点（项目硬约束）。

- [ ] **Step 1: 写失败测试**

在 `backend/test/handler/demo_test.go` 追加请求校验的纯函数测试：

```go
func TestValidateRechargeRequest(t *testing.T) {
	cases := []struct {
		name    string
		userID  int64
		amount  string
		wantErr bool
	}{
		{"valid", 9101, "100.00", false},
		{"zero user", 0, "100.00", true},
		{"empty amount", 9101, "", true},
		{"non-positive amount", 9101, "0", true},
		{"negative amount", 9101, "-5", true},
		{"bad amount", 9101, "abc", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRechargeRequest(c.userID, c.amount)
			if (err != nil) != c.wantErr {
				t.Fatalf("validateRechargeRequest(%d,%q) err=%v wantErr=%v", c.userID, c.amount, err, c.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/test && go test ./handler/ -run TestValidateRechargeRequest -v`
Expected: FAIL（`undefined: validateRechargeRequest`）

- [ ] **Step 3: 写最小实现**

在 `backend/test/handler/demo.go` 顶部 import 增加 `"errors"`，并新增：

```go
// 在 DemoHandler struct 增加 internalCli 字段
// type DemoHandler struct {
// 	bizCli      *auctioncli.Client
// 	internalCli *auctioncli.Client
// }
// 同步改 NewDemoHandler(bizCli, internalCli *auctioncli.Client)

func validateRechargeRequest(userID int64, amount string) error {
	if userID <= 0 {
		return errors.New("invalid user_id")
	}
	if amount == "" {
		return errors.New("amount required")
	}
	v, err := decimal.NewFromString(amount)
	if err != nil {
		return errors.New("invalid amount")
	}
	if !v.IsPositive() {
		return errors.New("amount must be positive")
	}
	return nil
}

type rechargeRequest struct {
	UserID int64  `json:"user_id"`
	Amount string `json:"amount"`
}

// PostRecharge 给指定用户充值（演示用），复用 auction 内部充值接口。
func (h *DemoHandler) PostRecharge(ctx context.Context, c *app.RequestContext) {
	var req rechargeRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid body"})
		return
	}
	if err := validateRechargeRequest(req.UserID, req.Amount); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	hlog.CtxInfof(ctx, "[demo] recharge user=%d amount=%s", req.UserID, req.Amount)
	result := h.internalCli.TopUpUserBalance(ctx, req.UserID, req.Amount)
	if !result.OK {
		c.JSON(400, map[string]any{"error": result.Error, "status": result.StatusCode})
		return
	}
	c.JSON(200, map[string]any{"ok": true, "user_id": req.UserID, "amount": req.Amount})
}
```

> 注：`TopUpUserBalance` 的真实签名与返回类型以 [client.go:442-460](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go#L442-L460) 为准（参数可能是 `(ctx, userID int64, amount string)` 或 `float64`），实施时打开确认并对齐 handler 调用与字段名。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/test && go test ./handler/ -run TestValidateRechargeRequest -v`
Expected: PASS（6 个子用例全过）

- [ ] **Step 5: 更新构造与路由**

在 [main.go:72-77](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go#L72-L77) 确认 `internalCli` 已构造（已有），把 Task 1 的构造改为：

```go
demoHandler := handler.NewDemoHandler(bizCli, internalCli)
demo := api.Group("/demo")
demo.POST("/follow-bid", demoHandler.PostFollowBid)
demo.POST("/recharge", demoHandler.PostRecharge)
```

- [ ] **Step 6: 编译 + 全 handler 测试**

Run: `cd backend/test && go build ./... && go test ./handler/ -v`
Expected: 编译退出码 0；`TestComputeFollowBidAmount` 与 `TestValidateRechargeRequest` 全 PASS。

- [ ] **Step 7: 提交**

```bash
git add backend/test/handler/demo.go backend/test/handler/demo_test.go backend/test/main.go
git commit -m "feat(test): add demo recharge endpoint reusing internal balance topup"
```

---

## 前端实施

### Task 3: demo api 封装（/api/test/* 独立请求）

**Files:**
- Create: `frontend/h5/src/services/demoApi.ts`
- Test: `frontend/h5/src/services/__tests__/demoApi.test.ts`

> 设计要点：不能用 [services/api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) 的 `post`（baseURL 写死 `/api/v1`）。新写一个用 `fetch('/api/test/...')` 的请求函数，自动带 Authorization（与 api.ts 同源读 `auth_token`）。

- [ ] **Step 1: 写失败测试**

创建 `frontend/h5/src/services/__tests__/demoApi.test.ts`：

```ts
import { demoApi } from '../demoApi';

describe('demoApi', () => {
  const originalFetch = global.fetch;
  beforeEach(() => {
    localStorage.setItem('auth_token', 'tk-123');
  });
  afterEach(() => {
    global.fetch = originalFetch;
    localStorage.clear();
    jest.restoreAllMocks();
  });

  it('posts follow-bid to /api/test/demo/follow-bid with auth header', async () => {
    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    });
    global.fetch = mockFetch as any;

    await demoApi.followBid(42, 110);

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url, init] = mockFetch.mock.calls[0];
    expect(url).toBe('/api/test/demo/follow-bid');
    expect(init.method).toBe('POST');
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tk-123');
    expect(JSON.parse(init.body)).toEqual({ auction_id: 42, amount: 110 });
  });

  it('throws on non-ok response', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: 'bad' }),
    }) as any;

    await expect(demoApi.recharge(9101, '100.00')).rejects.toThrow('bad');
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npx jest src/services/__tests__/demoApi.test.ts`
Expected: FAIL（`Cannot find module '../demoApi'`）

- [ ] **Step 3: 写最小实现**

创建 `frontend/h5/src/services/demoApi.ts`：

```ts
const DEMO_BASE = '/api/test/demo';

async function postDemo<T = unknown>(path: string, body: unknown): Promise<T> {
  const token = localStorage.getItem('auth_token') || localStorage.getItem('token') || '';
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${DEMO_BASE}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error((data && (data.error || data.message)) || `请求失败 (${res.status})`);
  }
  return data as T;
}

export const demoApi = {
  followBid(auctionId: number, amount?: number) {
    const body: Record<string, unknown> = { auction_id: auctionId };
    if (typeof amount === 'number') body.amount = amount;
    return postDemo('/follow-bid', body);
  },
  recharge(userId: number, amount: string) {
    return postDemo('/recharge', { user_id: userId, amount });
  },
};
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend/h5 && npx jest src/services/__tests__/demoApi.test.ts`
Expected: PASS（2 个用例全过）

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/services/demoApi.ts frontend/h5/src/services/__tests__/demoApi.test.ts
git commit -m "feat(h5): add demo api client for /api/test/demo endpoints"
```

---

### Task 4: demoContext 跨组件共享当前 auctionId

**Files:**
- Create: `frontend/h5/src/store/demoContext.tsx`
- Test: 在 Task 7 的组件测试中间接覆盖（本任务仅建 Provider + hook）

> 设计要点：全局浮层挂在 App 顶层，拿不到 LiveRoomSlide 的内部 `auctionId` state。建一个极简 Context 存 `currentAuctionId`，LiveRoomSlide 设置它、DemoConsole 读取它。参照 [authContext.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/store/authContext.tsx) 的 Context 写法。

- [ ] **Step 1: 写实现（Context 无独立逻辑分支，随 Task 7 测试覆盖）**

创建 `frontend/h5/src/store/demoContext.tsx`：

```tsx
import React, { createContext, useContext, useMemo, useState, ReactNode } from 'react';

interface DemoContextValue {
  currentAuctionId: number;
  setCurrentAuctionId: (id: number) => void;
}

const DemoContext = createContext<DemoContextValue | undefined>(undefined);

export const DemoProvider = ({ children }: { children: ReactNode }) => {
  const [currentAuctionId, setCurrentAuctionId] = useState(0);
  const value = useMemo(
    () => ({ currentAuctionId, setCurrentAuctionId }),
    [currentAuctionId],
  );
  return <DemoContext.Provider value={value}>{children}</DemoContext.Provider>;
};

export const useDemo = (): DemoContextValue => {
  const ctx = useContext(DemoContext);
  if (!ctx) {
    // 容错：未挂 Provider 时返回空操作，避免演示组件崩溃
    return { currentAuctionId: 0, setCurrentAuctionId: () => {} };
  }
  return ctx;
};
```

- [ ] **Step 2: 编译检查（tsc）**

Run: `cd frontend/h5 && npx tsc --noEmit`
Expected: 退出码 0（无类型错误）。

- [ ] **Step 3: 提交**

```bash
git add frontend/h5/src/store/demoContext.tsx
git commit -m "feat(h5): add demo context for sharing current auction id"
```

---

### Task 5: DemoConsole 浮球组件（账号切换 + 菜单骨架）

**Files:**
- Create: `frontend/h5/src/components/DemoConsole/index.tsx`
- Create: `frontend/h5/src/components/DemoConsole/DemoConsole.css`
- Test: `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`

> 设计要点：AssistiveTouch 悬浮球，点击展开一级菜单（账号 / 演示 / 充值 / 关闭），账号二级（买家A / 商家 / 管理员 / 返回）调 `useAuth().login`。演示二级与充值在 Task 6 接线。本任务先做：渲染浮球、展开/收起、账号切换调用 login。SVG 线框图标 + 毛玻璃样式。账号常量内联定义。

- [ ] **Step 1: 写失败测试**

创建 `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`：

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import DemoConsole from '../index';

const mockLogin = jest.fn().mockResolvedValue(undefined);
jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({ login: mockLogin, logout: jest.fn(), user: { id: 9101 }, isAuthenticated: true }),
}));
jest.mock('../../../store/demoContext', () => ({
  useDemo: () => ({ currentAuctionId: 0, setCurrentAuctionId: jest.fn() }),
}));
const mockShowToast = jest.fn();
jest.mock('../../Toast', () => ({
  useToast: () => ({ showToast: mockShowToast, showLoading: jest.fn() }),
}));

describe('DemoConsole', () => {
  beforeEach(() => jest.clearAllMocks());

  it('toggles menu open when the floating ball is clicked', () => {
    render(<DemoConsole />);
    expect(screen.queryByText('账号')).not.toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('演示控制面板'));
    expect(screen.getByText('账号')).toBeInTheDocument();
  });

  it('logs in as buyer A when 买家A is chosen', async () => {
    render(<DemoConsole />);
    fireEvent.click(screen.getByLabelText('演示控制面板'));
    fireEvent.click(screen.getByText('账号'));
    fireEvent.click(screen.getByText('买家A'));
    await waitFor(() =>
      expect(mockLogin).toHaveBeenCalledWith({ phone: '13800138001', password: 'Demo@123456' }),
    );
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npx jest src/components/DemoConsole/__tests__/DemoConsole.test.tsx`
Expected: FAIL（`Cannot find module '../index'`）

- [ ] **Step 3: 写最小实现**

创建 `frontend/h5/src/components/DemoConsole/index.tsx`：

```tsx
import React, { useState, useCallback } from 'react';
import { useAuth } from '../../store/authContext';
import { useDemo } from '../../store/demoContext';
import { useToast } from '../Toast';
import { demoApi } from '../../services/demoApi';
import './DemoConsole.css';

const DEMO_PASSWORD = 'Demo@123456';
const ACCOUNTS = {
  buyerA: { phone: '13800138001', label: '买家A' },
  merchant: { phone: '13800138002', label: '商家' },
  admin: { phone: '13800138003', label: '管理员' },
};
const BUYER_B_ID = 9102;

type MenuLevel = 'root' | 'account' | 'action';

const DemoConsole: React.FC = () => {
  const [open, setOpen] = useState(false);
  const [level, setLevel] = useState<MenuLevel>('root');
  const { login } = useAuth();
  const { currentAuctionId } = useDemo();
  const { showToast } = useToast();

  const close = useCallback(() => {
    setOpen(false);
    setLevel('root');
  }, []);

  const switchAccount = useCallback(
    async (phone: string, label: string) => {
      try {
        await login({ phone, password: DEMO_PASSWORD });
        showToast(`已切换为${label}`, 'success');
        close();
      } catch (e: any) {
        showToast(e?.message || '切换失败', 'error');
      }
    },
    [login, showToast, close],
  );

  const followBid = useCallback(async () => {
    if (!currentAuctionId) {
      showToast('请先进入直播间', 'warning');
      return;
    }
    try {
      await demoApi.followBid(currentAuctionId);
      showToast('买家B 已跟价', 'success');
      close();
    } catch (e: any) {
      showToast(e?.message || '跟价失败', 'error');
    }
  }, [currentAuctionId, showToast, close]);

  const recharge = useCallback(async () => {
    try {
      await demoApi.recharge(BUYER_B_ID, '10000.00');
      showToast('充值成功', 'success');
      close();
    } catch (e: any) {
      showToast(e?.message || '充值失败', 'error');
    }
  }, [showToast, close]);

  return (
    <div className="demo-console">
      <button
        type="button"
        aria-label="演示控制面板"
        className="demo-console__ball"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="demo-console__ball-core" />
      </button>

      {open && (
        <div className="demo-console__menu" role="menu">
          {level === 'root' && (
            <>
              <button type="button" onClick={() => setLevel('account')}>账号</button>
              <button type="button" onClick={() => setLevel('action')}>演示</button>
              <button type="button" onClick={recharge}>充值</button>
              <button type="button" onClick={close}>关闭</button>
            </>
          )}
          {level === 'account' && (
            <>
              <button type="button" onClick={() => switchAccount(ACCOUNTS.buyerA.phone, ACCOUNTS.buyerA.label)}>买家A</button>
              <button type="button" onClick={() => switchAccount(ACCOUNTS.merchant.phone, ACCOUNTS.merchant.label)}>商家</button>
              <button type="button" onClick={() => switchAccount(ACCOUNTS.admin.phone, ACCOUNTS.admin.label)}>管理员</button>
              <button type="button" onClick={() => setLevel('root')}>返回</button>
            </>
          )}
          {level === 'action' && (
            <>
              <button type="button" onClick={followBid}>他人跟价</button>
              <button type="button" onClick={() => setLevel('root')}>返回</button>
            </>
          )}
        </div>
      )}
    </div>
  );
};

export default DemoConsole;
```

创建 `frontend/h5/src/components/DemoConsole/DemoConsole.css`：

```css
.demo-console {
  position: fixed;
  right: 16px;
  bottom: 96px;
  z-index: 9999;
}
.demo-console__ball {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  border: none;
  background: rgba(20, 20, 20, 0.35);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.demo-console__ball-core {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.85);
  box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.18);
}
.demo-console__menu {
  position: absolute;
  right: 0;
  bottom: 64px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border-radius: 14px;
  background: rgba(28, 28, 30, 0.55);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
}
.demo-console__menu button {
  min-width: 96px;
  padding: 8px 12px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
}
.demo-console__menu button:active {
  background: rgba(255, 255, 255, 0.24);
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend/h5 && npx jest src/components/DemoConsole/__tests__/DemoConsole.test.tsx`
Expected: PASS（2 个用例全过）

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/components/DemoConsole/
git commit -m "feat(h5): add demo console floating ball with account switch"
```

---

### Task 6: 演示动作测试补全（跟价 / 充值）

**Files:**
- Modify: `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`（补跟价/充值用例）

> 设计要点：Task 5 已实现 followBid/recharge 逻辑，本任务补测试锁定行为：无 auctionId 时跟价提示「请先进入直播间」；有 auctionId 时调 `demoApi.followBid`；充值调 `demoApi.recharge`。需把 demoApi mock 掉。

- [ ] **Step 1: 写失败测试**

在 `DemoConsole.test.tsx` 顶部 mock demoApi，并把 useDemo mock 改为可变 auctionId。在文件顶部 mock 区追加：

```tsx
const mockFollowBid = jest.fn().mockResolvedValue({ ok: true });
const mockRecharge = jest.fn().mockResolvedValue({ ok: true });
jest.mock('../../../services/demoApi', () => ({
  demoApi: {
    followBid: (...args: any[]) => mockFollowBid(...args),
    recharge: (...args: any[]) => mockRecharge(...args),
  },
}));

let mockAuctionId = 0;
jest.mock('../../../store/demoContext', () => ({
  useDemo: () => ({ currentAuctionId: mockAuctionId, setCurrentAuctionId: jest.fn() }),
}));
```

> 注意：若 Task 5 测试文件里已有 `jest.mock('../../../store/demoContext', ...)`，需替换为这个可变版本（删除旧的固定 mock，避免重复 mock 报错）。

追加用例：

```tsx
it('warns when following bid without an active auction', () => {
  mockAuctionId = 0;
  render(<DemoConsole />);
  fireEvent.click(screen.getByLabelText('演示控制面板'));
  fireEvent.click(screen.getByText('演示'));
  fireEvent.click(screen.getByText('他人跟价'));
  expect(mockShowToast).toHaveBeenCalledWith('请先进入直播间', 'warning');
  expect(mockFollowBid).not.toHaveBeenCalled();
});

it('calls follow-bid api with current auction id', async () => {
  mockAuctionId = 77;
  render(<DemoConsole />);
  fireEvent.click(screen.getByLabelText('演示控制面板'));
  fireEvent.click(screen.getByText('演示'));
  fireEvent.click(screen.getByText('他人跟价'));
  await waitFor(() => expect(mockFollowBid).toHaveBeenCalledWith(77));
});

it('calls recharge api when 充值 is chosen', async () => {
  render(<DemoConsole />);
  fireEvent.click(screen.getByLabelText('演示控制面板'));
  fireEvent.click(screen.getByText('充值'));
  await waitFor(() => expect(mockRecharge).toHaveBeenCalledWith(9102, '10000.00'));
});
```

- [ ] **Step 2: 运行测试**

Run: `cd frontend/h5 && npx jest src/components/DemoConsole/__tests__/DemoConsole.test.tsx`
Expected: PASS（全部用例，含新增 3 条）。若失败，根据断言信息核对 Task 5 实现的金额/参数是否一致。

- [ ] **Step 3: 提交**

```bash
git add frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx
git commit -m "test(h5): cover demo console follow-bid and recharge actions"
```

---

### Task 7: 接线到 App.tsx 与 LiveRoomSlide

**Files:**
- Modify: `frontend/h5/src/App.tsx`（挂 DemoProvider + DemoConsole）
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`（写入 currentAuctionId）

> 设计要点：`<DemoProvider>` 需包裹 `<MobileContainer>`（让 LiveRoomSlide 与 DemoConsole 共享同一 context）；`<DemoConsole />` 挂在 [App.tsx:78](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx#L78) MobileContainer 内、与 `<Routes>` 同级（参照 ToastInitializer 位置），保证不随路由卸载且能用 useAuth/useToast/useDemo。LiveRoomSlide 在 `setAuctionId(effectiveId)`（[LiveRoomSlide.tsx:437](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L437)）后同步写入 demoContext。

- [ ] **Step 1: 在 App.tsx 挂载**

打开 [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx)，import：

```tsx
import { DemoProvider } from './store/demoContext';
import DemoConsole from './components/DemoConsole';
```

在 Provider 树中，于 `<AuctionProvider>` 与 `<MobileContainer>` 之间插入 `<DemoProvider>`（包裹 MobileContainer），并在 `<MobileContainer>` 内、`<Suspense>`/`<Routes>` 同级处加 `<DemoConsole />`。即把原结构：

```tsx
<AuctionProvider>
  <MobileContainer>
    <Suspense fallback={...}>
      <Routes>...</Routes>
    </Suspense>
  </MobileContainer>
</AuctionProvider>
```

改为：

```tsx
<AuctionProvider>
  <DemoProvider>
    <MobileContainer>
      <DemoConsole />
      <Suspense fallback={...}>
        <Routes>...</Routes>
      </Suspense>
    </MobileContainer>
  </DemoProvider>
</AuctionProvider>
```

> 以文件中真实的 `<Suspense>` fallback 内容为准，不要改动 fallback。

- [ ] **Step 2: LiveRoomSlide 写入 currentAuctionId**

打开 [LiveRoomSlide.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx)，import：

```tsx
import { useDemo } from '../../store/demoContext';
```

在组件内（`useAuth()` 附近，[LiveRoomSlide.tsx:193](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L193)）加：

```tsx
const { setCurrentAuctionId } = useDemo();
```

在 `setAuctionId(effectiveId)`（[LiveRoomSlide.tsx:437](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L437)）之后紧接一行：

```tsx
setCurrentAuctionId(effectiveId);
```

- [ ] **Step 3: 类型检查 + 构建**

Run: `cd frontend/h5 && npx tsc --noEmit && npm run build`
Expected: tsc 退出码 0；build 成功产出（无类型/打包错误）。

- [ ] **Step 4: 回归既有测试**

Run: `cd frontend/h5 && npx jest src/pages/Live/__tests__/LiveRoomSlide.test.tsx`
Expected: PASS（20 个用例全过，确认接线未破坏 LiveRoomSlide）。

> 若 LiveRoomSlide 测试因新增 `useDemo` 报错（未 mock context），在该测试文件已有的 mock 区补 `jest.mock('../../../store/demoContext', () => ({ useDemo: () => ({ currentAuctionId: 0, setCurrentAuctionId: jest.fn() }) }));`。

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/App.tsx frontend/h5/src/pages/Live/LiveRoomSlide.tsx frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx
git commit -m "feat(h5): mount demo console globally and feed current auction id"
```

---

### Task 8: 端到端手动验证（本地）

**Files:** 无（手动验证 + 记录）

> 「并发压测」与「竞拍延时」不新增代码：压测复用现有 `/test/pressure` 平台入口；竞拍延时靠在临近结束的拍卖里手动出价触发防狙击真实延时（已合入 main），由 H5 倒计时回弹 + toast 体现。本任务做一次真机/浏览器闭环确认。

- [ ] **Step 1: 起本地服务**

Run: `bash scripts/deploy-dev.sh`（或既有本地启动流程）
Expected: gateway / auction / test-service / h5 均启动；seed 已注入 138 账号。

- [ ] **Step 2: 验证账号切换**

在 H5 点悬浮球 → 账号 → 买家A，确认 toast「已切换为买家A」，页面进入登录态。依次验证商家、管理员。

- [ ] **Step 3: 验证他人跟价**

以买家A 进入一个进行中的拍卖直播间 → 悬浮球 → 演示 → 他人跟价。
Expected: 当前页面看到被超价（rank_update/bid_placed 广播驱动的动画），若已点天灯则触发自动反击。

- [ ] **Step 4: 验证充值**

点悬浮球 → 充值，确认 toast「充值成功」；进入余额/我的页确认买家B 余额增加（或改为给当前用户充值，见下方风险）。

- [ ] **Step 5: 验证竞拍延时（依赖已合入的防狙击）**

进入一个临近结束（剩 <10s）的拍卖，手动出价。
Expected: 倒计时实时回弹 + 弹「触发防狙击」提示（验证 merge `a3f696b1` 的链路）。

- [ ] **Step 6: 记录验证结果**

把以上结果记入 PR 描述或 state 文档。无独立 commit。

---

## Self-Review 注记

- **充值对象**：Task 5 当前实现给固定 `BUYER_B_ID=9102` 充值。若演示更希望「给当前登录用户充值」，需把当前用户 id 经 demoContext 或 useAuth 传入 recharge（`useAuth().user?.id`）。实施 Task 5 时按演示实际需求二选一，并同步 Task 6 的断言金额/对象。这是**唯一需要执行者按现场决定的开放点**，不影响其余任务。
- **并发压测**：未在 DemoConsole 菜单内做独立按钮（复用现有 `/test/pressure` 平台），如需在浮球内加入口，可在 Task 5 的 action 菜单追加一个跳转 `/test/pressure` 的按钮（react-router `useNavigate`），非必需。
- **SDK 字段名核对**：Task 1/2 依赖 `StepResult`、`PlaceBid`、`TopUpUserBalance` 的真实签名，实施时务必打开 [client.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/client/auction/client.go) 对齐字段（`.OK/.Error/.StatusCode` 等），这是后端两个 Task 的主要风险点。
- **防作弊 R4**：本计划明确不含（功能未实现）。
