# 接口功能清单

> 本文档梳理了项目所有后端API接口及其对应的前端功能点（细化到按钮维度）
> 生成时间：2026-05-27

---

## 一、服务架构概览

| 服务 | 端口 | 职责 |
|------|------|------|
| Gateway Service | 8080 | API网关，JWT认证，限流，请求转发 |
| Product Service | 8081 | 商品、竞拍规则、订单、统计管理 |
| Auction Service | 8082 | 竞拍、出价、用户、通知、关注管理 |
| WebSocket Service | 8083 | 实时出价更新、倒计时推送 |

---

## 二、接口功能清单（按业务模块）

### 1. 认证模块

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 用户登录 | POST | `/api/v1/auth/login` | 用户认证登录 | Admin登录页「登录按钮」、H5登录页「登录按钮」 | 提交表单(email/phone + password) |
| 用户注册 | POST | `/api/v1/auth/register` | 用户账号注册 | H5登录页「注册按钮」 | 提交表单(name + email/phone + password) |
| 获取当前用户 | GET | `/api/v1/users/me` | 获取登录用户信息 | 页面加载时自动调用 | 需JWT Token |

---

### 2. 商品模块

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 商品列表 | GET | `/api/v1/products` | 查询商品列表 | Admin商品列表页「页面加载」 | 分页参数(page, page_size, status) |
| 商品详情 | GET | `/api/v1/products/:id` | 查询商品详情 | Admin编辑页「页面加载」、规则配置页「页面加载」 | 商品ID |
| 创建商品 | POST | `/api/v1/products` | 新建商品记录 | Admin新建页「提交按钮」 | 商品信息(name, description, images[]) |
| 更新商品 | PUT | `/api/v1/products/:id` | 更新商品信息 | Admin编辑页「保存修改按钮」 | 商品ID + 更新数据 |
| 删除商品 | DELETE | `/api/v1/products/:id` | 删除商品记录 | Admin商品列表「删除按钮」 | 商品ID（需确认弹窗） |
| 发布商品 | POST | `/api/v1/products/:id/publish` | 上架商品到直播间 | Admin商品列表「发布按钮」 | 商品ID |
| 下架商品 | POST | `/api/v1/products/:id/unpublish` | 从直播间下架 | Admin商品列表「下架按钮」 | 商品ID + reason |
| 获取规则 | GET | `/api/v1/products/:id/rules` | 获取竞拍规则配置 | Admin规则配置页「页面加载」 | 商品ID |
| 创建规则 | POST | `/api/v1/products/:id/rules` | 配置竞拍规则 | Admin规则配置页「保存配置按钮」 | 规则参数(start_price, increment, cap_price, duration, delay_duration, max_delay_time, trigger_delay_before) |

---

### 3. 竞拍模块

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 竞拍列表 | GET | `/api/v1/auctions` | 查询竞拍列表 | Admin竞拍列表页「页面加载」、H5首页「页面加载」 | 状态筛选(status)、直播间ID/名称搜索 |
| 竞拍详情 | GET | `/api/v1/auctions/:id` | 查询竞拍详情 | Admin竞拍详情页「页面加载」、H5竞拍详情页「页面加载」 | 竞拍ID |
| 出价记录 | GET | `/api/v1/auctions/:id/bids` | 查询出价历史 | Admin竞拍详情页「页面加载」、H5竞拍详情页「页面加载」 | 竞拍ID |
| 用户出价 | POST | `/api/v1/auctions/:id/bids` | 提交出价 | H5竞拍详情页「出价按钮」、H5竞拍详情页「确认出价按钮」 | 竞拍ID + 出价金额（需JWT） |
| 竞拍排名 | GET | `/api/v1/auctions/:id/ranking` | 查询出价排名 | H5竞拍详情页（实时展示） | 竞拍ID |
| 取消竞拍 | PUT | `/api/v1/auctions/:id/cancel` | 取消竞拍场次 | Admin竞拍列表「取消竞拍按钮」、Admin竞拍详情页「取消竞拍按钮」 | 竞拍ID（需确认弹窗） |
| 竞拍结果 | GET | `/api/v1/auctions/:id/result` | 查询竞拍结果 | H5结果页「页面加载」 | 竞拍ID |

**状态枚举：**
- `0` 待开始
- `1` 进行中
- `2` 延时中
- `3` 已结束
- `4` 已取消

---

### 4. 订单模块

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 订单列表 | GET | `/api/v1/orders` | 查询订单列表 | Admin订单列表页「页面加载」、H5历史页「页面加载」 | 无参数 |
| 订单详情 | GET | `/api/v1/orders/:id` | 查询订单详情 | Admin订单操作时 | 订单ID |
| 更新状态 | PUT | `/api/v1/orders/:id/status` | 更新订单状态 | Admin订单列表「确认支付按钮」、Admin订单列表「标记发货按钮」、Admin订单列表「确认完成按钮」 | 订单ID + 新状态 |
| 支付订单 | POST | `/api/v1/orders/:id/pay` | 模拟支付 | H5历史页「立即支付按钮」、H5历史页「确认支付按钮」 | 订单ID |
| 发货订单 | PUT | `/api/v1/orders/:id/ship` | 模拟发货 | Admin订单列表「标记发货按钮」 | 订单ID |
| 用户历史 | GET | `/api/v1/orders/history` | 用户订单历史 | H5历史页「页面加载」 | 需JWT |

---

### 5. 直播间模块

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 直播间列表 | GET | `/api/v1/admin/live-streams` | 管理端直播间列表 | Admin直播间列表页「页面加载」 | 分页参数（需Admin权限） |
| 直播间详情 | GET | `/api/v1/live-streams/:id` | 查询直播间详情 | Admin直播间详情页「页面加载」 | 直播间ID |
| 关注直播间 | POST | `/api/v1/live-streams/:id/follow` | 关注直播间 | H5关注页（自动调用） | 直播间ID（需JWT） |
| 取消关注 | DELETE | `/api/v1/live-streams/:id/follow` | 取消关注直播间 | H5关注页「取消关注按钮」 | 直播间ID（需JWT） |
| 用户关注列表 | GET | `/api/v1/user/followed-live-streams` | 用户关注的直播间 | H5关注页「页面加载」、H5关注页「加载更多按钮」 | 分页参数（需JWT） |

---

### 6. 通知模块

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 通知列表 | GET | `/api/v1/notifications` | 查询用户通知 | （待实现）消息中心 | 分页 + unread_only过滤（需JWT） |
| 未读数量 | GET | `/api/v1/notifications/unread-count` | 获取未读数 | （待实现）消息图标角标 | 无参数（需JWT） |
| 标记已读 | PUT | `/api/v1/notifications/:id/read` | 单条标记已读 | （待实现）点击消息 | 通知ID（需JWT） |
| 全部已读 | PUT | `/api/v1/notifications/read-all` | 批量标记已读 | （待实现）「全部已读按钮」 | 无参数（需JWT） |

---

### 7. 统计模块（Admin专用）

| 接口 | 方法 | 路径 | 功能点 | 前端按钮/操作 | 触发条件 |
|------|------|------|--------|---------------|----------|
| 数据概览 | GET | `/api/v1/statistics/overview` | 首页概览数据 | Admin数据大屏「页面加载」、Admin数据大屏「重新加载按钮」 | 无参数 |
| 竞拍统计 | GET | `/api/v1/statistics/auctions` | 竞拍数据统计 | Admin竞拍统计页「页面加载」、Admin竞拍统计页「刷新数据按钮」 | start_date, end_date |
| 收入统计 | GET | `/api/v1/statistics/revenue` | 收入数据统计 | Admin收入统计页「页面加载」、Admin收入统计页「刷新数据按钮」、Admin数据大屏「页面加载」 | start_date, end_date, category, group_by |
| 用户统计 | GET | `/api/v1/statistics/users` | 用户数据统计 | Admin用户统计页「页面加载」、Admin用户统计页「刷新数据按钮」 | start_date, end_date |

---

### 8. WebSocket模块

| 连接 | 路径 | 功能点 | 前端触发 |
|------|------|--------|----------|
| WebSocket连接 | `/ws?auction_id=<id>&token=<jwt>` | 实时竞拍数据推送 | H5竞拍详情页「页面加载」自动建立连接 |

**消息类型：**
- `price_update` 实时出价更新
- `countdown` 倒计时推送
- `auction_end` 竞拍结束通知

---

## 三、权限控制

| 角色 | Role值 | 权限范围 |
|------|--------|----------|
| 普通用户 | 0 | 出价、关注直播间、查看竞拍 |
| 商家 | 1 | 商品发布/下架 |
| 主播 | 1 | 创建/取消竞拍 |
| 管理员 | 2 | 全部操作，包括统计、管理所有数据 |

---

## 四、前端页面路由汇总

### Admin管理后台

| 路由 | 页面名称 | 主要功能 |
|------|----------|----------|
| `/admin-login` | 登录页 | 管理员登录 |
| `/dashboard` | 数据大屏 | 概览统计 |
| `/products` | 商品列表 | 商品管理 |
| `/products/create` | 新建商品 | 创建商品 |
| `/products/:id/edit` | 编辑商品 | 编辑商品 |
| `/products/:id/rules` | 规则配置 | 竞拍规则配置 |
| `/auctions` | 竞拍列表 | 竞拍管理 |
| `/auctions/:id` | 竞拍详情 | 竞拍详情查看 |
| `/orders` | 订单列表 | 订单管理 |
| `/live-streams` | 直播间列表 | 直播间管理 |
| `/live-streams/:id` | 直播间详情 | 直播间详情 |
| `/statistics` | 统计首页 | 统计导航 |
| `/statistics/auction` | 竞拍统计 | 竞拍数据报表 |
| `/statistics/revenue` | 收入统计 | 收入数据报表 |
| `/statistics/user` | 用户统计 | 用户数据报表 |

### H5用户端

| 路由 | 页面名称 | 主要功能 |
|------|----------|----------|
| `/login` | 登录页 | 用户登录/注册 |
| `/` | 首页 | 竞拍列表浏览 |
| `/auction/:id` | 竞拍详情 | 竞拍详情、出价 |
| `/live` | 直播间 | 直播间竞拍（模拟） |
| `/follow` | 关注页 | 关注的直播间 |
| `/history` | 历史页 | 订单历史记录 |
| `/result/:id` | 结果页 | 竞拍结果查看 |

---

## 五、未实现/待完善功能

1. **H5直播间页** - 使用模拟数据，未接入真实API
2. **消息通知中心** - 后端接口已实现，前端未展示
3. **商品导出** - Admin商品列表有按钮但未实现
4. **结果页支付** - H5结果页支付功能未实现
5. **用户个人中心** - 用户信息编辑、余额查询待完善

---

## 六、后端代码位置参考

| 服务 | 路由定义 | Handler | Model |
|------|----------|---------|-------|
| Gateway | `backend/gateway/router/router.go` | `backend/gateway/middleware/` | - |
| Auction | `backend/auction/handler/auction.go` | `backend/auction/handler/` | `backend/auction/model/` |
| Product | `backend/product/handler/product.go` | `backend/product/handler/` | `backend/product/model/` |
| WebSocket | `backend/auction/main.go` | WebSocket Handler | - |

---

## 七、Swagger文档

- Auction Service: `backend/auction/docs/swagger.json`
- Product Service: `backend/product/docs/swagger.json`
- Gateway Service: `backend/gateway/docs/swagger.json`