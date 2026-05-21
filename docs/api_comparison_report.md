# 前后端接口对比报告

## 一、后端已实现的 API 接口

### 1. 商品服务 (Product Service) - 通过 Gateway 代理

| 方法 | 路径 | 功能 | 状态 |
|------|------|------|------|
| GET | `/api/v1/products` | 获取商品列表 | ✅ 已实现 |
| GET | `/api/v1/products/:id` | 获取商品详情 | ✅ 已实现 |
| POST | `/api/v1/products` | 创建商品 | ✅ 已实现 |
| PUT | `/api/v1/products/:id` | 更新商品 | ✅ 已实现 |
| DELETE | `/api/v1/products/:id` | 删除商品 | ✅ 已实现 |
| POST | `/api/v1/products/:id/rules` | 创建竞拍规则 | ✅ 已实现 |
| GET | `/api/v1/products/:id/rules` | 获取竞拍规则 | ✅ 已实现 |

### 2. 竞拍服务 (Auction Service) - 通过 Gateway 代理

| 方法 | 路径 | 功能 | 状态 |
|------|------|------|------|
| POST | `/api/v1/auctions` | 创建竞拍场次 | ✅ 已实现 |
| GET | `/api/v1/auctions/:id` | 获取竞拍详情 | ✅ 已实现 |
| PUT | `/api/v1/auctions/:id/cancel` | 取消竞拍 | ✅ 已实现 |
| GET | `/api/v1/auctions/:id/result` | 获取竞拍结果 | ✅ 已实现 |
| POST | `/api/v1/auctions/:id/bids` | 出价 | ✅ 已实现 |
| GET | `/api/v1/auctions/:id/ranking` | 获取出价排名 | ✅ 已实现 |

### 3. 订单服务 (Order Service)

| 方法 | 路径 | 功能 | 状态 |
|------|------|------|------|
| GET | `/api/v1/orders` | 获取订单列表 | ✅ 已实现 |
| GET | `/api/v1/orders/:id` | 获取订单详情 | ✅ 已实现 |
| POST | `/api/v1/orders/:id/pay` | 支付订单 | ✅ 已实现 |

### 4. WebSocket

| 方法 | 路径 | 功能 | 状态 |
|------|------|------|------|
| GET | `/api/v1/ws` | WebSocket 连接 | ✅ 已实现 |

### 5. 用户服务 (Auction Service 内)

| 方法 | 路径 | 功能 | 状态 |
|------|------|------|------|
| POST | `/api/v1/users` | 创建用户 | ✅ 已实现 |
| POST | `/api/v1/users/batch` | 批量创建用户 | ✅ 已实现 |

---

## 二、前端调用的 API 分析

### 1. Admin 管理后台

| 页面 | 调用的 API | 后端支持 |
|------|-----------|---------|
| 商品管理 | `GET /api/v1/products` | ✅ |
| 商品管理 | `POST /api/v1/products` | ✅ |
| 商品管理 | `PUT /api/v1/products/:id` | ✅ |
| 商品管理 | `DELETE /api/v1/products/:id` | ✅ |
| 商品管理 | `GET /api/v1/products/:id/rules` | ✅ |
| 商品管理 | `POST /api/v1/products/:id/rules` | ✅ |
| 竞拍管理 | `GET /api/v1/auctions` | ❌ **缺失** |
| 竞拍管理 | `PUT /api/v1/auctions/:id/cancel` | ✅ |
| 竞拍管理 | `GET /api/v1/auctions/:id/result` | ✅ |
| 订单管理 | `GET /api/v1/orders` | ✅ |
| 订单管理 | `POST /api/v1/orders/:id/pay` | ✅ |

### 2. H5 移动端

| 页面 | 调用的 API | 后端支持 |
|------|-----------|---------|
| 首页 | `GET /api/v1/auctions` | ❌ **缺失** |
| 历史记录 | `GET /api/v1/orders` | ✅ |
| 直播间 | (模拟数据) | ❌ 需要直播间 API |

---

## 三、发现的问题

### 🔴 严重问题

1. **`GET /api/v1/auctions` 接口缺失**
   - 前端 Admin 和 H5 都调用此接口获取竞拍列表
   - 后端 Gateway 未注册此路由
   - Auction Service 也未实现 ListAuction 方法
   - **影响**: 竞拍列表页面无法正常工作

### 🟡 中等问题

2. **直播间相关 API 缺失**
   - 无 `GET /api/v1/live-rooms` 或 `GET /api/v1/live-rooms/:id/products`
   - 新设计的直播间页面需要获取直播间及商品数据
   - **影响**: 直播间页面只能使用模拟数据

3. **用户出价历史 API 缺失**
   - 无 `GET /api/v1/users/:id/bids` 或 `GET /api/v1/my/bids`
   - 用户无法查看自己的出价记录
   - **影响**: 用户无法追踪自己的出价状态

### 🟢 低优先级问题

4. **订单状态更新 API 缺失**
   - 无 `PUT /api/v1/orders/:id/status`
   - Admin 后台订单管理中"确认支付"、"标记发货"等操作无法正常工作
   - **影响**: 订单状态管理功能不完整

5. **用户认证 API 缺失**
   - 无用户登录、注册、Token 验证等接口
   - 当前使用模拟用户数据
   - **影响**: 无法区分真实用户

---

## 四、建议修复方案

### 优先级 P0 - 必须修复

```go
// 1. 添加竞拍列表接口
// backend/auction/handler/auction.go
func (h *AuctionHandler) List(ctx context.Context, c *app.RequestContext) {
    statusStr := c.Query("status")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    
    auctions, total, err := h.auctionService.ListAuctions(ctx, status, page, pageSize)
    // ...
}

// 2. 注册路由
// backend/auction/main.go
v1.GET("/auctions", auctionHandler.List)

// 3. Gateway 注册
// backend/gateway/router/router.go
v1.GET("/auctions", auctionProxy.Forward)
```

### 优先级 P1 - 建议修复

```go
// 直播间商品列表
GET /api/v1/live-rooms/:id/products

// 用户出价历史
GET /api/v1/users/me/bids

// 订单状态更新
PUT /api/v1/orders/:id/status
```

---

## 五、移动端功能测试结果

### 测试环境
- H5 前端: http://localhost:5178
- 视口尺寸: 375x812 (iPhone X)

### 测试项目

| 功能 | 测试结果 | 备注 |
|------|---------|------|
| 首页加载 | ✅ 通过 | 使用模拟数据 |
| 直播间视频背景 | ✅ 通过 | 视频正常播放 |
| 直播间头部信息 | ✅ 通过 | 显示主播、观看人数 |
| 底部浮窗收起状态 | ✅ 通过 | 显示商品数量和价格 |
| 底部浮窗展开 | ✅ 通过 | 触摸事件正常响应 |
| 商品列表显示 | ✅ 通过 | 5个商品正常显示 |
| 商品状态标签 | ✅ 通过 | 竞拍中/即将开始/已结束 |
| 价格标签 | ✅ 通过 | 当前最高价/起拍价/落槌价 |
| 倒计时显示 | ✅ 通过 | 每个商品独立倒计时 |
| 出价面板打开 | ✅ 通过 | 点击商品卡片打开 |
| 出价面板内容 | ✅ 通过 | 商品信息、排名、输入框 |
| 出价输入 | ✅ 通过 | 输入金额正常 |
| 出价提交 | ✅ 通过 | 模拟成功 |
| 出价面板关闭 | ✅ 通过 | 遮罩点击关闭 |

### 结论
移动端 UI 和交互功能完整，主要问题是 API 接口缺失导致无法使用真实数据。
