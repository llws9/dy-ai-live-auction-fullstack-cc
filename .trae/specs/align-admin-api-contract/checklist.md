# Checklist - Admin 端接口对齐

> change-id: `align-admin-api-contract`
> 验收标准：每条勾选必须有可执行的"如何验证"，最终全部勾选才算完成。

## 一、契约一致性

- [ ] `GET /products` 响应字段为 `{ list, total, page, page_size }`，无 `items`
- [ ] `GET /orders` 响应字段为 `{ list, total }`，C 端语义不变
- [ ] `GET /admin/orders` 响应字段为 `{ list, total, page, page_size }`，含联表 VO 字段
- [ ] `GET /auctions` 响应字段为 `{ list, total, page, page_size }`，每项含 `product / live_stream_name / bid_count`
- [ ] `GET /auctions/:id` 含 `product / rules / winner_name / bid_count`
- [ ] `GET /auctions/:id/bids` 每项含 `user_name`
- [ ] `GET /admin/live-streams`、`GET /live-streams/:id` 字段为 `streamer_id, streamer_name, streamer_avatar, viewer_count, auction_count, status, ...`
- [ ] `GET /admin/live-streams?status=` 过滤生效
- [ ] `GET /statistics/overview` 含 `total_auctions, ongoing_auctions, total_revenue, today_revenue, total_users, total_orders`
- [ ] `GET /statistics/dashboard` 一次返回 6 项 KPI
- [ ] `GET /statistics/auctions|revenue|users` 返回数组，且按 `group_by` 分支正确
- [ ] `/orders/:id/pay` gateway 与 product 方法均为 `POST`

## 二、新增模块

- [ ] 拍卖规则模板 5 个 endpoint 全部可用：list / get / create / update / delete
- [ ] `POST /products/:id/apply-rule-template` 能 upsert `auction_rules`
- [ ] 角色 CRUD 可用，`PUT /admin/roles/:id/permissions` 全量替换权限
- [ ] 管理员账户 CRUD 可用，密码字段不出现在响应
- [ ] **复用** `POST /live-streams/:id/start` 完成 admin 开播（不重复造接口）
- [ ] `PUT /admin/live-streams/:id/end` 状态置 `ended` + WS 广播
- [ ] `PUT /admin/live-streams/:id/ban` 状态置 `banned` + 写 `ban_reason`
- [ ] `PUT /users/me` 更新基本信息成功
- [ ] `PUT /users/me/password` 旧密码错误返回 400；正确返回 200 后旧密码无法登录
- [ ] `PUT /users/me/preferences` 写入 `users.preferences` JSON
- [ ] `POST /uploads` 上传 jpg/png/webp 成功；超过 5MB 拒绝；mime 不匹配拒绝
- [ ] product 服务删除冗余 `PUT /orders/:id/pay` 注册，仅保留 `POST`

## 三、权限与安全

- [ ] 所有 `/admin/*` 路由对非 admin JWT 返回 403
- [ ] `/admin/orders` 不再读 `X-User-ID`
- [ ] 修改密码使用 bcrypt 验证 + 重写
- [ ] 上传文件名 UUID 化，避免目录穿越

## 四、数据库

- [ ] 新表已创建：`auction_rule_templates / roles / role_permissions`
- [ ] 列变更：`live_streams.streamer_name / streamer_avatar / viewer_count / ban_reason`
- [ ] 列新增：`users.preferences JSON`
- [ ] 迁移脚本可幂等执行，存在 rollback 说明

## 五、前端联动

- [ ] [api/index.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/shared/api/index.ts) 增加新模块封装：`auctionRuleTemplateApi / roleApi / adminUserApi / uploadApi / profileApi`，并补 `liveStreamApi.{start, end, ban}`
- [ ] [Dashboard.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Dashboard.tsx) 改用 `/statistics/dashboard`，6 卡片有真实数据
- [ ] [Dashboard.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Dashboard.tsx) "开启直播"按钮解除 disabled，对接 `POST /live-streams/:id/start`
- [ ] [Dashboard.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Dashboard.tsx) 待办事项卡片仍为静态占位（spec 明确不接入）
- [ ] [OrderList.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/OrderList.tsx) / [OrderDetail.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/OrderDetail.tsx) admin 视角能看到全量订单与完整字段
- [ ] [Stats.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Stats.tsx) 三 Tab 不再 runtime 报错，图表渲染正常
- [ ] [AuctionRules.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/AuctionRules.tsx) 移除 mock，CRUD + 应用模板可用
- [ ] [Permissions.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Permissions.tsx) 移除 mock，角色与管理员账户 CRUD 可用
- [ ] [LiveDetail.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/LiveDetail.tsx) 三个控制按钮可点
- [ ] [Profile.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Profile.tsx) 修改信息/密码/偏好按钮启用并能成功提交
- [ ] [GoodsEdit.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/GoodsEdit.tsx) 图片上传接入新接口

## 六、测试与质量

- [ ] 后端：每个新 handler / 修改 handler 至少 1 个单测；DAO 联表查询有最小集成测试
- [ ] 后端：`go test ./...` 全绿
- [ ] 前端：核心 page 关键路径快照测试通过；`pnpm test` 全绿
- [ ] 端到端：管理员账号登录后 13 个页面无 console 错误，无 4xx/5xx

## 七、文档

- [ ] [docs/api_documentation.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/api_documentation.md) 同步新接口
- [ ] [docs/api-interface-list.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/api-interface-list.md) 同步
- [ ] [docs/DATABASE_SCHEMA.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/DATABASE_SCHEMA.md) 同步表结构
- [ ] [docs/admin-api-audit.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/admin-api-audit.md) 顶部追加"已通过 align-admin-api-contract spec 解决"的 closure 说明
