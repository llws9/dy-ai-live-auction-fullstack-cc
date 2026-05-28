# Tasks: 用户功能开发

**Feature**: `20260523-product-auction-live` - 用户端功能开发
**Input**: Design documents from `/specs/20260523-product-auction-live/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Focus**: 前端H5用户功能开发（后端已完成）

**Tests**: 前端测试任务标记为可选

**Organization**: 任务按用户故事分组，每个故事可独立实施和测试。

## Format: `[ID] [P?] [Story] Description`
- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属用户故事（US2.6, US2.5）
- 包含准确的文件路径

## Path Conventions
- **Frontend H5**: `frontend/h5/src/`
- **Backend**: `backend/auction/`, `backend/gateway/` (已完成，仅供参考)

---

## Overview

**Total Tasks**: 23 tasks
**Focus**: 前端H5用户功能
- **Foundational**: 认证系统 - 3 tasks
- **User Story 2.6**: 用户出价竞拍 - 10 tasks
- **User Story 2.5**: 用户关注直播间 - 8 tasks
- **Polish**: 优化和测试 - 2 tasks

**Backend Status**: ✅ 已完成（Handler、Service、DAO、Model、测试均已实现）

---

## Phase 1: Foundational (阻塞性前置条件)

**Purpose**: 认证系统 - 所有用户功能的前置条件

**⚠️ CRITICAL**: 用户出价和关注功能依赖认证系统，必须先完成

### 认证系统实现

- [x] T001 [P] 创建认证服务 - `frontend/h5/src/services/auth.ts`
  - 实现login(), logout(), getCurrentUser()方法
  - 处理JWT token存储和读取（localStorage）
  - 错误处理和状态管理

- [x] T002 [P] 创建认证上下文 - `frontend/h5/src/store/authContext.tsx`
  - 创建AuthContext和AuthProvider
  - 管理全局认证状态（isAuthenticated, user, token）
  - 提供useAuth hook给子组件使用

- [x] T003 实现API拦截器配置 - `frontend/h5/src/services/api.ts`
  - 添加请求拦截器，自动携带JWT token到Authorization header
  - 添加响应拦截器，处理401错误跳转登录页
  - 依赖T001, T002完成

**Checkpoint**: ✅ 认证系统就绪 - 用户可以登录/登出，API请求自动携带token

---

## Phase 2: User Story 2.6 - 用户出价竞拍 (Priority: P0) 🎯 MVP

**Goal**: 用户可以在竞拍中出价，系统验证用户登录状态，实时更新排名

**Independent Test**:
1. 未登录用户点击出价 → 提示"请先登录"
2. 已登录用户输入出价金额 → 验证金额合法性
3. 出价成功 → 实时更新排名，WebSocket推送通知

### WebSocket实时推送服务

- [x] T004 创建WebSocket服务 - `frontend/h5/src/services/websocket.ts`
  - 实现WebSocket连接管理（connect, disconnect, reconnect）
  - 消息处理和分发机制
  - 自动重连逻辑（指数退避，最多5次）
  - 错误处理和状态管理

### 出价UI组件

- [x] T005 [P] 创建出价输入组件 - `frontend/h5/src/components/BidInput.tsx`
  - 出价金额输入框（支持小数点后2位）
  - 实时验证逻辑（最小出价金额、封顶价）
  - 出价按钮和加载状态
  - 错误提示显示（红色文字）

- [x] T006 [P] 创建排名列表组件 - `frontend/h5/src/components/RankingList.tsx`
  - 显示当前竞拍排名（前10名）
  - 实时更新排名（从WebSocket接收消息）
  - 显示用户自己的排名高亮
  - 显示出价金额和出价时间

### 出价功能集成

- [x] T007 集成出价功能到直播间页面 - `frontend/h5/src/pages/Live/index.tsx`
  - 在竞拍详情区域添加BidInput组件
  - 在竞拍详情区域添加RankingList组件
  - 连接WebSocket服务（传入auctionId）
  - 处理出价成功/失败反馈（Toast通知）
  - 依赖T004, T005, T006

- [x] T008 实现出价API调用 - `frontend/h5/src/services/api.ts`
  - 添加placeBid(auctionId, amount)方法
  - 处理API响应和错误（401未认证、400参数错误、409竞拍已结束）
  - 出价成功后返回更新后的竞拍信息

- [x] T009 添加出价相关通知处理 - `frontend/h5/src/pages/Live/index.tsx`
  - 处理WebSocket消息类型：排名更新、竞拍结束通知
  - 显示Toast通知（出价成功、出价被超越）
  - 更新竞拍状态（当前价格、剩余时间）

- [x] T010 添加登录状态检查 - `frontend/h5/src/components/BidInput.tsx`
  - 未登录用户点击出价 → 显示Toast"请先登录"
  - 跳转到登录页（或弹出登录框）
  - 已登录用户显示当前用户余额或提示

**Checkpoint**: ✅ 用户出价功能完整可用，可以登录、出价、查看实时排名

---

## Phase 3: User Story 2.5 - 用户关注直播间 (Priority: P1)

**Goal**: 用户可以关注/取消关注直播间，查看关注列表，接收通知推送

**Independent Test**:
1. 用户点击关注按钮 → 按钮状态立即改变（乐观更新）
2. 关注成功 → 显示Toast提示
3. 访问我的关注页面 → 显示关注的直播间列表

### 关注按钮组件

- [x] T011 [P] 创建关注按钮组件 - `frontend/h5/src/components/FollowButton.tsx`
  - 关注/取消关注按钮UI（带图标）
  - 乐观更新逻辑（立即改变状态，失败后回滚）
  - 加载状态和禁用状态
  - 显示当前关注数量

### 关注列表页面

- [x] T012 [P] 创建我的关注页面 - `frontend/h5/src/pages/Follow/index.tsx`
  - 显示关注的直播间列表（卡片式布局）
  - 分页加载（每页20条，滚动加载）
  - 搜索功能（按直播间名称）
  - 空状态提示（"暂无关注的直播间"）

- [x] T013 添加关注列表路由 - `frontend/h5/src/App.tsx`
  - 添加 `/follow` 路由
  - 配置页面导航（底部Tab栏或侧边栏）

### 关注API集成

- [x] T014 实现关注API调用 - `frontend/h5/src/services/api.ts`
  - 添加followLiveStream(liveStreamId)方法
  - 添加unfollowLiveStream(liveStreamId)方法
  - 添加getFollowedLiveStreams(page, pageSize)方法
  - 处理API响应和错误（401未认证、409重复关注）
  - 添加followLiveStream(liveStreamId)方法
  - 添加unfollowLiveStream(liveStreamId)方法
  - 添加getFollowedLiveStreams(page, pageSize)方法
  - 处理API响应和错误（401未认证、409重复关注）

- [x] T015 集成关注功能到直播间页面 - `frontend/h5/src/pages/Live/index.tsx`
  - 在直播间详情区域添加FollowButton组件
  - 调用关注/取消关注API
  - 更新直播间关注状态（关注数+1/-1）
  - 依赖T011

- [x] T016 添加关注相关通知处理 - `frontend/h5/src/pages/Follow/index.tsx`
  - 处理WebSocket消息类型：新商品发布、竞拍开始通知
  - 显示通知徽标（红点或数字）
  - 更新直播间活跃状态（当前竞拍数）

- [x] T017 添加关注列表入口 - `frontend/h5/src/pages/Home/index.tsx` 或导航组件
  - 在首页或导航栏添加"我的关注"入口
  - 显示关注数量徽标

**Checkpoint**: ✅ 用户关注功能完整可用，可以关注直播间、查看关注列表、接收通知

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: 优化用户体验和测试覆盖

### 性能优化

- [x] T018 [P] 添加图片懒加载 - `frontend/h5/src/pages/Follow/index.tsx`
  - 使用Intersection Observer API实现懒加载
  - 减少首屏加载时间
  - 添加占位图

- [x] T019 [P] 添加WebSocket消息节流 - `frontend/h5/src/services/websocket.ts`
  - 限制消息处理频率（100ms节流）
  - 避免频繁渲染影响性能
  - 使用lodash.throttle或自定义节流函数

### 错误处理

- [x] T020 添加错误边界组件 - `frontend/h5/src/components/ErrorBoundary.tsx`
  - 捕获组件渲染错误
  - 显示友好的错误提示
  - 提供重试按钮

### 测试（可选）

- [x] T021 [P] 前端组件测试 - `frontend/h5/src/components/__tests__/`
  - BidInput组件测试（验证逻辑、错误提示）
  - FollowButton组件测试（乐观更新、状态管理）
  - 使用Jest + React Testing Library

- [x] T022 [P] E2E测试 - `e2e/user-features.spec.ts`
  - 用户完整出价流程测试（登录 → 出价 → 查看排名）
  - 用户关注流程测试（关注 → 查看列表 → 取消关注）
  - 使用Playwright或Cypress

- [x] T023 添加登录页面（如果不存在） - `frontend/h5/src/pages/Login/index.tsx`
  - 手机号/邮箱登录表单
  - 密码输入框（显示/隐藏切换）
  - 登录按钮（加载状态）
  - 登录成功后跳转原页面

**Checkpoint**: ✅ 所有用户功能已实现、优化并测试通过

---

## Dependencies & Execution Strategy

### Story Completion Order

```
Phase 1 (认证系统)
    ↓
Phase 2 (US2.6 出价功能) ← MVP核心
    ↓
Phase 3 (US2.5 关注功能)
    ↓
Phase 4 (优化和测试)
```

### Parallel Execution Opportunities

**Phase 1 (Foundational)**:
- T001, T002可并行（不同文件）

**Phase 2 (US2.6)**:
- T005, T006可并行（不同组件）
- T004需先完成（WebSocket服务）

**Phase 3 (US2.5)**:
- T011, T012可并行（不同文件）
- T013, T014可并行（不同文件）

**Phase 4 (Polish)**:
- T018, T019, T021, T022均可并行

### MVP Scope

**Minimum Viable Product**:
- Phase 1 (认证系统) - 必须
- Phase 2 (US2.6 出价功能) - 核心

**Estimated Time**:
- Phase 1: 2-3小时
- Phase 2: 6-8小时
- Phase 3: 4-5小时
- Phase 4: 2-3小时

**Total**: 14-19小时（约2-3个工作日）

---

## Implementation Notes

### Authentication Flow
1. 用户访问需要认证的功能 → 检查localStorage中的token
2. Token不存在或已过期 → 跳转登录页
3. 登录成功 → 存储token到localStorage，更新AuthContext
4. 后续API请求自动携带token（通过拦截器）

### WebSocket Flow
1. 用户进入竞拍详情页 → 建立WebSocket连接
2. 收到消息 → 更新排名列表、显示通知
3. 连接断开 → 自动重连（指数退避）
4. 用户离开页面 → 断开连接

### Optimistic Update (关注功能)
1. 用户点击关注 → 立即更新按钮状态为"已关注"
2. 发送API请求 → 成功则保持状态，失败则回滚
3. 提供即时的视觉反馈，提升用户体验

---

## Testing Strategy

### Manual Testing Checklist

**认证系统**:
- [ ] 未登录用户访问需要认证的页面 → 跳转登录页
- [ ] 登录成功 → token存储到localStorage
- [ ] API请求自动携带token
- [ ] Token过期 → 自动跳转登录页

**出价功能**:
- [ ] 未登录用户点击出价 → 提示"请先登录"
- [ ] 已登录用户输入非法金额 → 显示错误提示
- [ ] 出价成功 → 更新排名，显示Toast
- [ ] WebSocket断开 → 自动重连
- [ ] 实时排名更新 → 正确显示

**关注功能**:
- [ ] 点击关注按钮 → 按钮状态立即改变
- [ ] 关注失败 → 按钮状态回滚，显示错误提示
- [ ] 访问我的关注页面 → 显示关注的直播间
- [ ] 分页加载 → 正确加载下一页

---

## File Summary

**新建文件** (9个):
1. `frontend/h5/src/services/auth.ts` - 认证服务
2. `frontend/h5/src/store/authContext.tsx` - 认证上下文
3. `frontend/h5/src/services/websocket.ts` - WebSocket服务
4. `frontend/h5/src/components/BidInput.tsx` - 出价输入组件
5. `frontend/h5/src/components/RankingList.tsx` - 排名列表组件
6. `frontend/h5/src/components/FollowButton.tsx` - 关注按钮组件
7. `frontend/h5/src/pages/Follow/index.tsx` - 我的关注页面
8. `frontend/h5/src/components/ErrorBoundary.tsx` - 错误边界组件
9. `frontend/h5/src/pages/Login/index.tsx` - 登录页面（如果不存在）

**修改文件** (4个):
1. `frontend/h5/src/services/api.ts` - 添加认证拦截器和API方法
2. `frontend/h5/src/pages/Live/index.tsx` - 集成出价和关注功能
3. `frontend/h5/src/App.tsx` - 添加关注页面路由和登录页路由
4. `frontend/h5/src/pages/Home/index.tsx` - 添加关注列表入口

---

## Backend API Reference (已完成)

**认证相关**:
- POST `/api/v1/auth/login` - 用户登录
- POST `/api/v1/auth/logout` - 用户登出
- GET `/api/v1/auth/me` - 获取当前用户信息

**出价相关**:
- POST `/api/v1/auctions/:id/bids` - 用户出价（需JWT认证）
- GET `/api/v1/auctions/:id/ranking` - 获取竞拍排名

**关注相关**:
- POST `/api/v1/live-streams/:id/follow` - 关注直播间（需JWT认证）
- DELETE `/api/v1/live-streams/:id/follow` - 取消关注
- GET `/api/v1/user/followed-live-streams` - 获取关注列表（需JWT认证）

**WebSocket端点**:
- `ws://localhost:8080/ws/auction/:id` - 竞拍实时更新

---

**Generated**: 2026-05-23
**Status**: Ready for implementation
**Next Command**: `/adk:sdd:implement` to begin execution

---

## Phase 3: User Story 1 - 商品发布到直播间 (Priority: P1) 🎯 MVP

**Goal**: 商家发布草稿状态商品到直播间，自动创建竞拍记录

**Independent Test**: 创建测试商品 → 点击发布 → 验证商品状态变为已发布，竞拍记录创建成功，关联直播间

### Implementation for User Story 1

- [ ] T013 [P] [US1] 创建 `LiveStreamDAO` (`backend/product/dao/live_stream.go`) - 包含 `GetByCreatorID`, `Create` 方法
- [ ] T014 [P] [US1] 创建 `LiveStreamService` (`backend/product/service/live_stream.go`) - 包含 `GetOrCreateLiveStream` 业务逻辑
- [ ] T015 [US1] 修改 `ProductDAO` 添加 `UpdateStatus` 方法 (`backend/product/dao/product.go`)
- [ ] T016 [US1] 修改 `ProductService` 添加 `Publish` 方法 (`backend/product/service/product.go`) - 实现发布逻辑（验证状态、获取直播间、创建竞拍、更新状态）
- [ ] T017 [US1] 创建发布商品 API handler (`backend/product/handler/product.go` - 新增 `Publish` handler)
- [ ] T018 [US1] 在 Gateway 注册路由 `POST /api/v1/products/:id/publish` (`backend/gateway/router/router.go`)
- [ ] T019 [P] [US1] 前端：在商品管理列表添加"发布"按钮 (`frontend/admin/src/pages/Product/List.tsx`)
- [ ] T020 [US1] 前端：实现发布按钮点击逻辑，调用 API 并刷新列表 (`frontend/admin/src/pages/Product/List.tsx`)

**Checkpoint**: User Story 1 完成，商品发布功能可用且可独立测试

---

## Phase 4: User Story 2 - 商品下架功能 (Priority: P1)

**Goal**: 商家下架已发布商品，取消关联竞拍，通知已出价用户

**Independent Test**: 创建已发布商品 → 创建竞拍和出价记录 → 点击下架 → 验证商品状态变为已下架，竞拍取消，通知发送

### Implementation for User Story 2

- [ ] T021 [P] [US2] 创建通知消息结构体 (`backend/auction/mq/notification.go`)
- [ ] T022 [US2] 修改 `ProductService` 添加 `Unpublish` 方法 (`backend/product/service/product.go`) - 实现下架逻辑（验证状态、查询竞拍、取消竞拍、发送通知）
- [ ] T023 [US2] 实现批量推送通知服务 (`backend/auction/service/notification.go`) - 分批查询已出价用户，批量插入通知记录
- [ ] T024 [US2] 创建下架商品 API handler (`backend/product/handler/product.go` - 新增 `Unpublish` handler)
- [ ] T025 [US2] 在 Gateway 注册路由 `POST /api/v1/products/:id/unpublish` (`backend/gateway/router/router.go`)
- [ ] T026 [P] [US2] 前端：在商品管理列表添加"下架"按钮 (`frontend/admin/src/pages/Product/List.tsx`)
- [ ] T027 [US2] 前端：实现下架确认对话框和逻辑 (`frontend/admin/src/pages/Product/List.tsx`)

**Checkpoint**: User Story 2 完成，商品下架功能可用且可独立测试

---

## Phase 5: User Story 2.5 - 用户关注直播间功能 (Priority: P1)

**Goal**: 用户关注直播间，接收新商品和竞拍开始通知，实现批量推送策略

**Independent Test**: 用户关注直播间 → 商家发布商品 → 验证用户收到通知，取消关注后不再收到

### Implementation for User Story 2.5

- [ ] T028 [P] [US2.5] 创建 `UserLiveStreamFollowDAO` (`backend/auction/dao/user_live_stream_follow.go`) - 包含 `Create`, `Delete`, `GetByUserAndLiveStream`, `GetFollowers`, `CountByLiveStream` 方法
- [ ] T029 [P] [US2.5] 创建 `FollowService` (`backend/auction/service/follow.go`) - 实现关注、取消关注、更新通知偏好逻辑
- [ ] T030 [US2.5] 修改 `NotificationService` 实现批量推送通知 (`backend/auction/service/notification.go`) - 1万用户/批，间隔3秒，最长10分钟
- [ ] T031 [P] [US2.5] 创建关注直播间 API handler (`backend/auction/handler/follow.go` - 新增 `FollowLiveStream` handler)
- [ ] T032 [P] [US2.5] 创建取消关注 API handler (`backend/auction/handler/follow.go` - 新增 `UnfollowLiveStream` handler)
- [ ] T033 [P] [US2.5] 创建获取用户关注列表 API handler (`backend/auction/handler/follow.go` - 新增 `GetUserFollowedLiveStreams` handler)
- [ ] T034 [P] [US2.5] 创建获取直播间关注统计 API handler (`backend/auction/handler/follow.go` - 新增 `GetLiveStreamFollowersStats` handler)
- [ ] T035 [US2.5] 在 Gateway 注册关注相关路由 (`backend/gateway/router/router.go`) - `POST /live-streams/:id/follow`, `DELETE /live-streams/:id/follow`, `GET /user/followed-live-streams`, `GET /live-streams/:id/followers/stats`
- [ ] T036 [P] [US2.5] 前端(H5)：创建直播间列表页 (`frontend/h5/src/pages/LiveStream/List.tsx`)
- [ ] T037 [P] [US2.5] 前端(H5)：创建直播间详情页 (`frontend/h5/src/pages/LiveStream/Detail.tsx`)
- [ ] T038 [P] [US2.5] 前端(H5)：创建我的关注页面 (`frontend/h5/src/pages/User/Follows.tsx`)
- [ ] T039 [US2.5] 前端(H5)：在直播间详情页添加关注/取消关注按钮 (`frontend/h5/src/pages/LiveStream/Detail.tsx`)
- [ ] T040 [US2.5] 修改 `ProductService.Publish` 发送新商品通知到 RabbitMQ (`backend/product/service/product.go`) - 调用 `producer.SendNewProductNotification`
- [ ] T041 [US2.5] 实现竞拍开始前30分钟延迟通知逻辑 (`backend/auction/mq/producer.go`) - 发送到延迟队列，延迟时间 = 开始时间 - 30分钟

**Checkpoint**: User Story 2.5 完成，关注和通知推送功能可用且可独立测试

---

## Phase 6: User Story 4 - 竞拍管理状态筛选优化 (Priority: P1)

**Goal**: 新增"待开始"筛选按钮，管理员视角显示直播间列和搜索功能

**Independent Test**: 管理员登录 → 验证筛选按钮正确 → 验证直播间列显示 → 验证搜索功能

### Implementation for User Story 4

- [ ] T042 [US4] 修改 `AuctionDAO` 添加按状态和直播间查询方法 (`backend/auction/dao/auction.go`) - 包含 `GetByStatus`, `GetByLiveStreamID`, `SearchByLiveStreamName` 方法
- [ ] T043 [US4] 修改 `AuctionService.List` 支持状态筛选和权限过滤 (`backend/auction/service/auction.go`) - 商家只返回自己直播间的竞拍，管理员返回所有竞拍
- [ ] T044 [US4] 修改竞拍列表 API handler (`backend/auction/handler/auction.go` - 修改 `List` handler) - 新增 status, live_stream_id, live_stream_name 查询参数
- [ ] T045 [US4] 前端：在竞拍管理页面添加"待开始"筛选按钮 (`frontend/admin/src/pages/Auction/List.tsx`)
- [ ] T046 [US4] 前端：为管理员视角添加直播间ID和名称列 (`frontend/admin/src/pages/Auction/List.tsx`)
- [ ] T047 [US4] 前端：添加搜索框支持按直播间ID或名称搜索 (`frontend/admin/src/pages/Auction/List.tsx`)

**Checkpoint**: User Story 4 完成，竞拍管理筛选和搜索功能可用且可独立测试

---

## Phase 7: User Story 6 - 权限和数据可见性隔离 (Priority: P1)

**Goal**: 实现角色权限隔离，商家只能访问自己的数据，管理员可访问所有数据

**Independent Test**: 商家登录 → 验证只能看到自己直播间的竞拍 → 尝试越权访问返回403；管理员登录 → 验证可看到所有数据

### Implementation for User Story 6

- [ ] T048 [P] [US6] 创建商家权限中间件 (`backend/gateway/middleware/merchant.go`) - 验证 role=1
- [ ] T049 [P] [US6] 创建管理员权限中间件 (`backend/gateway/middleware/admin.go`) - 验证 role=2
- [ ] T050 [US6] 修改所有新增 API 端点添加权限验证 (`backend/gateway/router/router.go`) - 商品发布/下架需要商家或管理员权限，直播间统计需要商家(自己的直播间)或管理员权限
- [ ] T051 [US6] 修改 `AuctionDAO.GetByCreatorID` 确保只返回该商家直播间的竞拍 (`backend/auction/dao/auction.go`)
- [ ] T052 [US6] 前端：根据用户角色动态显示/隐藏菜单项 (`frontend/admin/src/App.tsx`)
- [ ] T053 [US6] 前端：显示当前用户角色和直播间信息 (`frontend/admin/src/components/Header.tsx` 或导航栏组件)

**Checkpoint**: User Story 6 完成，权限隔离功能可用且可独立测试

---

## Phase 8: User Story 3 - 配置规则表单UI优化 (Priority: P2)

**Goal**: 优化配置规则表单UI，使用统一组件库，添加输入验证

**Independent Test**: 打开配置规则页面 → 验证表单样式符合设计规范 → 输入非法数据验证提示

### Implementation for User Story 3

- [ ] T054 [US3] 前端：移除内联样式，使用统一CSS类 (`frontend/admin/src/pages/Product/RuleConfig.tsx`) - 移除 formItemStyle, labelStyle, inputStyle, buttonStyle
- [ ] T055 [US3] 前端：添加表单分组（基础设置、时间设置、高级设置）(`frontend/admin/src/pages/Product/RuleConfig.tsx`)
- [ ] T056 [US3] 前端：添加输入验证逻辑 (`frontend/admin/src/pages/Product/RuleConfig.tsx`) - 加价幅度>0, 竞拍时长60-3600秒, 封顶价>=0
- [ ] T057 [US3] 前端：优化错误提示方式，使用Toast组件 (`frontend/admin/src/pages/Product/RuleConfig.tsx`)

**Checkpoint**: User Story 3 完成，配置规则表单UI优化可用且可独立测试

---

## Phase 9: User Story 5 - 直播间管理模块 (Priority: P2)

**Goal**: 管理员可查看和管理所有直播间，包括统计数据和状态管理

**Independent Test**: 管理员登录 → 进入直播间管理 → 验证列表展示 → 验证状态操作

### Implementation for User Story 5

- [ ] T058 [P] [US5] 创建 `LiveStreamDAO` 统计方法 (`backend/product/dao/live_stream.go`) - 包含 `GetAll`, `GetByID`, `CountActiveAuctions`, `GetTotalRevenue` 方法
- [ ] T059 [P] [US5] 创建 `LiveStreamService` (`backend/product/service/live_stream.go` - 新增 `List`, `GetDetail`, `UpdateStatus` 方法)
- [ ] T060 [P] [US5] 创建直播间列表 API handler (`backend/product/handler/live_stream.go` - 新增 `List` handler)
- [ ] T061 [P] [US5] 创建直播间详情 API handler (`backend/product/handler/live_stream.go` - 新增 `GetDetail` handler)
- [ ] T062 [P] [US5] 创建更新直播间状态 API handler (`backend/product/handler/live_stream.go` - 新增 `UpdateStatus` handler)
- [ ] T063 [US5] 在 Gateway 注册直播间管理路由 (`backend/gateway/router/router.go`) - `GET /live-streams`, `GET /live-streams/:id`, `PUT /live-streams/:id/status`
- [ ] T064 [P] [US5] 前端：创建直播间列表页 (`frontend/admin/src/pages/LiveStream/List.tsx`)
- [ ] T065 [P] [US5] 前端：创建直播间详情页 (`frontend/admin/src/pages/LiveStream/Detail.tsx`)
- [ ] T066 [US5] 前端：在导航菜单添加"直播间管理"菜单项 (`frontend/admin/src/App.tsx`)

**Checkpoint**: User Story 5 完成，直播间管理功能可用且可独立测试

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: 跨用户故事的改进和优化

- [ ] T067 [P] 数据库查询优化：为 `live_streams.creator_id` 添加索引（如果迁移脚本中未添加）
- [ ] T068 [P] 数据库查询优化：为 `user_live_stream_follows(live_stream_id, notification_enabled)` 添加复合索引
- [ ] T069 [P] 实现直播间统计数据缓存 (`backend/product/service/live_stream.go`) - 使用 Redis 缓存关注数、活跃竞拍数、总成交额
- [ ] T070 [P] 实现直播间统计数据定时更新任务 (`backend/product/service/live_stream.go`) - 每小时从数据库重新计算并更新缓存
- [ ] T071 [P] RabbitMQ 监控配置 - 配置队列堆积告警（消息数>1000）、死信队列告警（消息数>100）
- [ ] T072 [P] 日志记录：添加关键操作日志（商品发布、下架、关注、通知推送）(`backend/product/handler/`, `backend/auction/handler/`)
- [ ] T073 [P] 错误处理优化：统一错误响应格式 (`backend/gateway/middleware/error_handler.go`)
- [ ] T074 启动 RabbitMQ 消费者 Worker (`backend/auction/main.go`) - 在 Auction Service 启动时启动消费者

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖，可立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成 - **阻塞所有用户故事**
- **User Stories (Phase 3-9)**: 全部依赖 Foundational 完成
  - 用户故事之间可以并行执行（如有足够人力）
  - 或按优先级顺序执行（P1 → P2）
- **Polish (Phase 10)**: 依赖所需用户故事完成

### User Story Dependencies

- **US1 (商品发布)**: 无依赖，可独立实施
- **US2 (商品下架)**: 无依赖，可独立实施（但依赖 US1 的商品发布功能测试）
- **US2.5 (关注直播间)**: 无依赖，可独立实施（但依赖 US1 的商品发布测试通知功能）
- **US3 (UI优化)**: 无依赖，可独立实施
- **US4 (竞拍筛选)**: 无依赖，可独立实施
- **US5 (直播间管理)**: 无依赖，可独立实施
- **US6 (权限隔离)**: 应在 US1, US2, US4, US5 之前或同时完成，因为所有这些故事都需要权限验证

**推荐顺序**: US6 → US1 → US2 → US2.5 → US4 → US3 → US5

### Within Each User Story

- Models/DAOs before Services
- Services before Handlers
- Backend before Frontend
- Core implementation before integration

### Parallel Opportunities

- **Phase 1**: T002, T003, T004 可并行
- **Phase 2**: T006-T011 可并行
- **User Stories**: 不同用户故事可由不同团队成员并行开发
- **Within Stories**: 标记 [P] 的任务可并行（通常是在不同文件中的任务）

---

## Parallel Example: User Story 2.5 (关注直播间)

```bash
# 可并行启动的任务（不同文件，无依赖）:
Task T028: "创建 UserLiveStreamFollowDAO"
Task T029: "创建 FollowService"
Task T031: "创建关注直播间 API handler"
Task T032: "创建取消关注 API handler"
Task T033: "创建获取用户关注列表 API handler"
Task T034: "创建获取直播间关注统计 API handler"
Task T036: "前端(H5)：创建直播间列表页"
Task T037: "前端(H5)：创建直播间详情页"
Task T038: "前端(H5)：创建我的关注页面"
```

---

## Implementation Strategy

### MVP First (User Story 1 + 6)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (**CRITICAL** - 阻塞所有故事)
3. Complete Phase 7: User Story 6 (权限隔离)
4. Complete Phase 3: User Story 1 (商品发布)
5. **STOP and VALIDATE**: 测试权限隔离和商品发布功能
6. Deploy/demo if ready

**MVP Scope**: 权限隔离 + 商品发布功能

### Incremental Delivery

1. Complete Setup + Foundational → 基础设施就绪
2. Add US6 + US1 → 测试独立功能 → 部署/演示 (MVP!)
3. Add US2 (商品下架) → 测试独立功能 → 部署/演示
4. Add US2.5 (关注和通知) → 测试独立功能 → 部署/演示
5. Add US4 (竞拍筛选) → 测试独立功能 → 部署/演示
6. Add US3 (UI优化) + US5 (直播间管理) → 部署/演示
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US6 (权限隔离)
   - Developer B: US1 (商品发布)
   - Developer C: US2 (商品下架)
3. After US6 and US1 complete:
   - Developer A: US2.5 (关注直播间)
   - Developer B: US4 (竞拍筛选)
   - Developer C: US5 (直播间管理)
4. Developer D (if available): US3 (UI优化)
5. Stories complete and integrate independently

---

## Task Summary

**Total Tasks**: 74
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundational): 8 tasks
- Phase 3 (US1 - 商品发布): 8 tasks
- Phase 4 (US2 - 商品下架): 7 tasks
- Phase 5 (US2.5 - 关注直播间): 14 tasks
- Phase 6 (US4 - 竞拍筛选): 6 tasks
- Phase 7 (US6 - 权限隔离): 6 tasks
- Phase 8 (US3 - UI优化): 4 tasks
- Phase 9 (US5 - 直播间管理): 9 tasks
- Phase 10 (Polish): 8 tasks

**Parallel Opportunities**: 35 tasks marked [P] can run in parallel within their phases

**MVP Scope**: 22 tasks (Phase 1 + Phase 2 + US6 + US1)

---

## Notes

- [P] 标记的任务位于不同文件，无依赖，可并行执行
- [Story] 标签映射任务到具体用户故事，便于追溯
- 每个用户故事应可独立完成和测试
- 每个任务或逻辑组完成后提交代码
- 任何检查点均可停止并独立验证故事
- 避免：模糊任务描述、同文件冲突、跨故事依赖导致独立性破坏
