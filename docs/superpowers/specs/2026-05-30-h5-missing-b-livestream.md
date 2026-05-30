# 子 Spec B · 直播间详情字段补齐 + 关注语义重命名为「收藏」

**日期**：2026-05-30

**关联总览 Spec**：[2026-05-30-h5-missing-interfaces-closure.md](./2026-05-30-h5-missing-interfaces-closure.md)

**同期子 Spec**：
- [子 Spec A · 用户中心数据闭环](./2026-05-30-h5-missing-a-user-center.md)
- [子 Spec C · 商品/竞拍/分类数据契约](./2026-05-30-h5-missing-c-product-auction.md)
- [子 Spec D · OrderDetail 页面 + Home 未读数接入](./2026-05-30-h5-missing-d-order-detail.md)

**风格参考**：[2026-05-30-user-touchpoints-backend-design-adapted.md](./2026-05-30-user-touchpoints-backend-design-adapted.md)

---

## 1. 范围

### 1.1 本子 Spec 必须落地

| 编号 | 能力 | 优先级 |
|---|---|---|
| F-B1 | `GET /api/v1/live-streams/:id` 字段扩展：补 `host_name / host_avatar / viewer_count / video_url / is_following` | P1 |
| F-B2 | `GET /api/v1/live-streams/:id/follow-status` 单独查询接口（鉴权） | P1 |
| F-B3 | `GET /api/v1/user/followed-live-streams` 列表项扩展：补 `host_avatar / viewer_count / auction_count` | P2 |
| F-B4 | UI 重命名：H5「关注」→「收藏」，仅文案/图标重命名，**不重命名后端 API、不重命名前端组件名** | P1 |

### 1.2 明确不做

- 后端 follow 接口路径/方法/字段重命名为 `collection`：**不做**。
- 新建 `favorites` 表或独立收藏聚合：**不做**。
- 视频流（实际 RTMP/HLS 拉流）接入：**不做**，仅落 `video_url` 字段，前端拿到后置占位/外链播放。
- WebSocket 通道增加 `viewer_count` 实时下推：**不做**，本期靠详情接口轮询/页面进入时拉取。

---

## 2. 接口契约

所有响应统一为 `{"code":200,"data":{...}}`。错误码沿用现有约定：`400` 入参错误、`401` 未鉴权、`404` 资源不存在、`500` 内部错误。

### 2.1 F-B1 · 直播间详情扩展

**Method / Path**：`GET /api/v1/live-streams/:id`

**鉴权**：**可选**。未登录可访问；带合法 `Authorization: Bearer <token>` 时附加 `is_following`。详见 §5.3 `JWTAuthOptional` 中间件。

**当前实现**：[live_stream.go GetDetail](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/live_stream.go) 仅返回 `id / name / description / cover_image / status / creator_id / created_at`。

**扩展后 Response**：

```json
{
  "code": 200,
  "data": {
    "id": 101,
    "name": "顶流主播·古玩专场",
    "description": "...",
    "cover_image": "https://cdn/.../cover.jpg",
    "status": 1,
    "creator_id": 9001,
    "created_at": "2026-05-20T10:00:00Z",

    "host_name": "老王",
    "host_avatar": "https://cdn/.../avatar.jpg",
    "viewer_count": 0,
    "video_url": null,
    "is_following": false
  }
}
```

**字段语义**：

| 字段 | 类型 | 来源 | 未登录 | 缺数据降级 |
|---|---|---|---|---|
| `host_name` | string | 跨服务读 auction-service `users.username` by `creator_id` | 返回 | 空字符串 `""` |
| `host_avatar` | string | 同上 `users.avatar` | 返回 | 空字符串 `""`，前端兜底默认头像 |
| `viewer_count` | int | auction-service WebSocket Hub 房间在线人数（本期降级为 `0`） | 返回 | `0` |
| `video_url` | string \| null | `live_streams.video_url`（新增列） | 返回 | `null` |
| `is_following` | bool | follow service `IsFollowing(user_id, live_stream_id)` | **省略字段或 false**（见 §5.3） | `false` |

### 2.2 F-B2 · 关注状态查询

**Method / Path**：`GET /api/v1/live-streams/:id/follow-status`

**鉴权**：**必需**（`authGroup`）。

**实现位置**：auction-service [follow.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/follow.go) 新增 `GetFollowStatusHandler`。

**Request**：无 body，路径参数 `id` 为 `live_stream_id`。

**Response**：

```json
{ "code": 200, "data": { "is_following": true } }
```

**Gateway 路由新增**（[router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) 「直播间关注路由」段）：

```
authGroup.GET("/live-streams/:id/follow-status", auctionProxy.Forward)
```

**错误码**：
- `400`：`id` 解析失败
- `401`：缺 token / token 非法
- `404`：直播间不存在（可选；本期返回 `is_following=false` 即可，不强校验存在性）
- `500`：DB 异常

### 2.3 F-B3 · 关注列表卡片字段扩展

**Method / Path**：`GET /api/v1/user/followed-live-streams`（路径**不变**）

**鉴权**：必需。

**当前返回**（[follow.go GetUserFollowsHandler](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/follow.go)）：直接序列化 `follow` 实体列表。

**扩展后 Response**：

```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 1,
        "live_stream_id": 101,
        "live_stream_name": "顶流主播·古玩专场",
        "cover_image": "https://cdn/.../cover.jpg",
        "status": 1,
        "notification_enabled": true,
        "followed_at": "2026-05-22T08:30:00Z",

        "host_avatar": "https://cdn/.../avatar.jpg",
        "viewer_count": 0,
        "auction_count": 3
      }
    ],
    "total": 12,
    "page": 1,
    "page_size": 20
  }
}
```

**字段来源**：

| 字段 | 来源 | 降级 |
|---|---|---|
| `host_avatar` | 跨服务批量查 `users` by `creator_id` 列表 | `""` |
| `viewer_count` | 同 §2.1，本期固定 `0` | `0` |
| `auction_count` | auction-service 内部查 `auctions` 表，统计某 `live_stream_id` 下 `status IN (进行中)` 的竞拍数 | `0` |

`auction_count` 为同进程内查询，无跨服务调用。`host_avatar` 与 `viewer_count` 复用 §5 跨服务方案。

---

## 3. 数据模型变更

### 3.1 `live_streams` 表新增列

实体：[live_stream.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/model/live_stream.go)

| 列名 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `video_url` | `varchar(512)` | nullable | 直播流地址（HLS / RTMP / 占位 URL） |

**迁移**：写一条 GORM AutoMigrate 即可，不写独立 SQL 脚本（与现有产品库迁移方式一致）。

### 3.2 不新增的内容

- 不新增 `host_name / host_avatar / viewer_count` 列。这些字段是**只读派生字段**，由跨服务调用与运行时统计聚合得出，写入会导致与 `users` 表/WebSocket 状态不一致。
- 不新增 `favorites` 表（与总览 Spec §1 一致）。

---

## 4. 跨服务调用方案

### 4.1 host_name / host_avatar：内部 HTTP

**问题**：详情接口实现在 product-service，但 `users` 表归 auction-service。

**候选方案对比**：

| 方案 | 优点 | 缺点 |
|---|---|---|
| (a) Gateway BFF 聚合 | product-service 零改动 | Gateway 复杂度上升、需为该单点写聚合代码 |
| (b) product-service 内部 HTTP 调 auction-service | 改动局部，符合现有「服务自治」 | 多一次内网调用 |
| (c) 同步冗余 `users` 副本到 product 库 | 读极快 | 一致性维护成本高，本期不值当 |

**结论 · 选 (b)**：auction-service 暴露**仅内网可达**接口 `GET /internal/users/:id`；product-service 在 `GetDetail` 中调用并加 Redis 缓存。

**内部接口契约**（auction-service 内网）：

- 路径：`GET /internal/users/:id`（**不**经过 gateway，**不**注册到 `/api/v1`）
- 鉴权：内网网络隔离 + Header `X-Internal-Token`（与既有内部令牌方案一致；若不存在则先复用环境变量 `INTERNAL_API_TOKEN`）
- 返回：`{"code":200,"data":{"id":9001,"username":"老王","avatar":"https://..."}}`
- 仅暴露 `id / username / avatar`，不返回手机号、密码等敏感字段

**缓存**：
- Key：`user:profile:{user_id}`
- TTL：`5min`
- 失效：用户改名/换头像时，由 auction-service 更新 user profile 后 `DEL` 该 key（本期可不做主动失效，依赖 5 分钟过期）

**批量场景（F-B3）**：新增 `POST /internal/users/batch`（body `{"ids":[...]}`）批量取，避免 N+1。

### 4.2 viewer_count：本期降级

**问题**：实时房间在线数在 auction-service [hub.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/hub.go) 内存中，product-service 不可直接访问。

**长期方案**：auction-service 暴露 `GET /internal/live-streams/:id/viewer-count`。

**本期决策**：
- 后端 `viewer_count` 字段**固定返回 `0`**（占位，保证字段稳定存在）。
- 前端拿到 `0` 时**显示占位 `-`** 而非 `0 人在线`，避免给用户错误反馈。
- 不在本子 Spec 内推进 hub 暴露接口；列入下一期增强。

### 4.3 is_following：详情接口的鉴权可选化

**问题**：当前 `v1.GET("/live-streams/:id", productProxy.Forward)`（[router.go#L104](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go)）在 `v1` 公开组下，没有 user_id 上下文。

**候选**：
- (a) 详情路由迁入 `authGroup` —— 破坏未登录用户浏览直播间的能力，**否决**。
- (b) 新增 `JWTAuthOptional` 中间件 —— 有 token 校验通过则注入 `user_id`；无 token 或 token 无效则放行但不注入。

**结论 · 选 (b)**。

**`JWTAuthOptional` 设计**：
- 位置：`backend/gateway/middleware/jwt_optional.go`（新文件，与现有 `jwt.go` 同目录）。
- 行为：
  - 无 `Authorization` header → 放行，不写 `user_id`。
  - 有 header 但解析失败 → 放行，不写 `user_id`，不返回 401。
  - 解析成功 → `c.Set("user_id", uid)` 后放行。
- 路由挂载：详情路由不再用 `v1.GET`，改为 `v1.GET(..., middleware.JWTAuthOptional(), productProxy.Forward)`。
- Forward 时：gateway 通过 header 透传 `X-User-Id`（沿用现有 `RequireAuth` 透传约定）；product-service 收到 header 即可。

**product-service 调用 follow service**：
- 新增内部接口 `GET /internal/follows/check?user_id=xxx&live_stream_id=yyy`（auction-service 暴露）。
- product-service `GetDetail` 检测到 `X-User-Id` 不为空时调用一次，否则跳过。
- 缓存：`follow:{user_id}:{live_stream_id}` TTL `60s`，关注/取消关注操作触发主动 `DEL`。

---

## 5. UI 重命名清单（F-B4）

**总原则**：仅文案/图标层重命名，组件名、文件名、props、API 名、followApi 方法名**全部不动**。

| 文件 | 改动点 |
|---|---|
| [FollowButton.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/FollowButton.tsx) | 内部文本 `关注` → `收藏`、`已关注` → `已收藏`；图标从 `+` 改为**心形 ❤** 语义图标（关注后填充实心，未关注空心） |
| [Follow/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Follow/index.tsx) | 页面标题 `关注` / `我的关注` → `我的收藏` |
| [BottomNav.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/MobileShell/BottomNav.tsx) | 「关注」入口 label → `收藏`；图标同步换为心形 |
| 直播间详情页关注按钮所在容器 | 复用 `FollowButton`，文案随组件自动更新；如有外层 `aria-label` / 文案需同步改 |

**图标统一选择**：**心形 ❤️**（不选书签）。理由：
- 与电商「收藏商品」普遍心形心智一致；
- 书签在 H5 场景偏「稍后阅读」语义，与直播间不匹配。

**保持不变**：
- 组件文件名 `FollowButton.tsx`、导出名 `FollowButton`。
- props 名（如 `isFollowing`、`onFollowChange`）。
- 测试文件中的描述若引用了「关注」语义，按 §7 要求同步替换断言文本，但不改文件名。

---

## 6. 前端集成点

### 6.1 LiveRoom 详情 adapter

- 位置：`frontend/h5/src/pages/Live/LiveRoom.tsx` 或对应 service 适配层。
- adapter 在解析 `GET /live-streams/:id` 响应时，对新增字段做兼容：
  - 字段缺失（旧后端） → `host_name=""`, `host_avatar=""`, `viewer_count=0`, `video_url=null`, `is_following=false`。
  - `viewer_count===0` 时 UI 显示 `-`（与 §4.2 对齐）。

### 6.2 followApi 新增方法

[api.ts followApi](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) 现有方法 `followLiveStream / unfollowLiveStream / getFollowedLiveStreams` 全部**保留命名**。

新增：

```
followApi.getFollowStatus(liveStreamId: number): Promise<{ is_following: boolean }>
```

- 内部调用 `GET /api/v1/live-streams/:id/follow-status`。
- 401 时返回 `{ is_following: false }`，不抛错（与未登录场景一致）。

**调用时机**：
- 详情页打开时，若用户已登录但响应中 `is_following` 字段缺失（旧后端兼容场景）→ 调一次 `getFollowStatus` 兜底。
- 正常情况下详情接口已带 `is_following`，**无需**额外调用。

### 6.3 Following 列表 adapter

- 处理 §2.3 新增字段：`host_avatar` 缺失走默认头像；`viewer_count===0` 显示 `-`；`auction_count` 显示 `N 个竞拍中`，为 `0` 时显示 `暂无在售`。

---

## 7. 测试要求

### 7.1 后端单测

| 测试 | 位置 | 要点 |
|---|---|---|
| `GetDetail` 未登录返回扩展字段 | `backend/product/handler/live_stream_test.go` | `is_following` 不出现或为 `false`；`host_name/avatar` 走 mock 内部 HTTP |
| `GetDetail` 已登录返回 `is_following=true` | 同上 | 通过注入 `X-User-Id` header + mock follow check |
| `GetFollowStatus` 鉴权 | `backend/auction/handler/follow_test.go` | 已关注/未关注两条用例 |
| `GetUserFollows` 列表附加字段 | 同上 | 校验 `host_avatar / viewer_count / auction_count` 存在 |
| `JWTAuthOptional` 中间件 | `backend/gateway/middleware/jwt_optional_test.go` | 三分支：无 token / 非法 token / 合法 token |
| 内部 `/internal/users/:id` | `backend/auction/handler/internal_user_test.go` | 鉴权失败 401、成功只返回 id/username/avatar |

### 7.2 前端单测调整

| 测试 | 调整 |
|---|---|
| [FollowButton.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/__tests__/FollowButton.test.tsx) | 断言文本 `关注/已关注` → `收藏/已收藏`；新增心形图标存在性断言 |
| [Following