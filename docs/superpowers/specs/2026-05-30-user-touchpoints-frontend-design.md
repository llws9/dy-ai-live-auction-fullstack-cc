# 用户触达体系（一期）·前端设计文档

**日期**：2026-05-30
**适配仓库**：`dy-ai-live-auction-fullstack-cc`
**范围**：移动端 H5 `frontend/h5/src/`，纯前端独立交付，数据先以 Mock 落地

---

## 1. 背景与目标

现有移动端 H5 已有登录、底部导航、个人中心、直播间、消息通知与全局 Toast 能力，但用户触达层次不完整：竞拍相关提醒没有统一入口，红点提醒缺失，关注主播开播弹窗的触发边界也需要收敛。

一期目标是做一个最小可验证触达闭环：
- 红点徽标：让用户在「我的」与「我的竞拍」入口看到待处理数量。
- 顶部 Toast：承载 B/C/D 三个高转化实时提醒场景。
- 中央弹窗：只在用户重新登录后触发一次，不因刷新重复打扰。

二期不做：悬浮气泡、Toast A/E 场景、通用消息中心、勿扰偏好、真实后端接口改造。

---

## 2. 当前仓库事实

外部设计文档引用的是 `src/mobile/`，当前仓库实际移动端入口为 `frontend/h5/src/`。

关键现状：
- 全局 App 结构在 [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx)，已挂载 `ToastProvider`、`AuthProvider`、`MobileContainer`。
- 移动端壳层在 [MobileContainer.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/MobileShell/MobileContainer.tsx)，当前只包裹内容与底部导航。
- 底部导航在 [BottomNav.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/MobileShell/BottomNav.tsx)，「我的」入口是 `/profile`。
- 登录状态在 [authContext.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/store/authContext.tsx)，登录后调用 `authService.login()` 并写入状态。
- 凭据持久化在 [auth.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/auth.ts)，主存储键为 `auth_token` / `auth_user`。
- 已存在全局 Toast 在 [components/Toast/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/Toast/index.tsx)，另有 shared Toast 与页面级 Toast，不能再新增第四套并行机制。
- 中央开播弹窗组件在 [LiveReminderModal](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/LiveReminderModal/index.tsx)，当前未在 `MobileContainer` 中挂载。
- 个人中心入口在 [User/Index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/User/Index.tsx)，「我的竞拍」跳转 `/history`。

---

## 3. 推进方案

### 方案 A：照搬外部文档

按 `src/mobile/` 与新建 `Toast/ToastContainer` 的方式实现。

问题：当前仓库没有 `src/mobile/`，且已有全局 Toast；照搬会制造重复架构。

### 方案 B：适配当前仓库并复用现有能力（推荐）

将需求落到 `frontend/h5/src/`，新增 `BadgeDot` 与 Mock hook，升级现有全局 `ToastProvider`，在 `MobileContainer` 统一挂载 `LiveReminderModal`。

优点：改动最少、边界清晰、不会新增重复 Toast 体系。

### 方案 C：先做触达平台抽象

统一抽象 Notification Center、Toast、Modal、Badge、WebSocket 事件分发。

问题：一期需求只需验证触达闭环，这会过度设计，推迟可验证结果。

结论：采用方案 B。

---

## 4. 设计原则

1. **SSOT**：触达状态、Mock 数据、Toast 展示入口各自只有一个事实源。
2. **不重复造轮子**：升级现有 `components/Toast`，不新增并行 Toast Provider。
3. **前端独立交付**：一期不依赖后端，后续真实接口只替换 hook/service 内部实现。
4. **低打扰**：中央弹窗只在重新登录后触发一次；实时提醒使用顶部 Toast；红点只表达待处理数量。
5. **YAGNI**：不做二期能力，不预设完整消息中心。

---

## 5. 模块拆分

```text
frontend/h5/src/
├── components/
│   ├── BadgeDot/
│   │   ├── index.tsx
│   │   └── BadgeDot.module.css
│   ├── Toast/
│   │   ├── index.tsx              # 改造：支持顶部卡片、队列、action
│   │   └── Toast.module.css       # 新增/替换：去掉内联样式
│   ├── LiveReminderModal/
│   │   └── index.tsx              # 复用现有组件，补足无占位图风险
│   └── MobileShell/
│       ├── MobileContainer.tsx    # 挂载登录后弹窗
│       ├── BottomNav.tsx          # 「我的」入口挂徽标
│       └── MobileShell.module.css # 增加徽标定位样式
├── hooks/
│   └── useTouchpointNotifications.ts
├── store/
│   └── authContext.tsx            # 登录成功写 pending_live_reminder
└── pages/
    ├── User/Index.tsx             # 「我的竞拍」挂徽标
    ├── Home/index.tsx             # 开发环境 Demo 触发器，可选
    └── Live/index.tsx             # 开发环境 Demo 触发器，可选
```

为什么这么拆：
- `BadgeDot` 是纯展示组件，不绑定业务数据。
- `useTouchpointNotifications` 聚合一期 Mock 触达数字，后续对接后端只改这一处。
- `ToastProvider` 已是全局单例，应增强 API 而不是新增 `ToastContainer`。
- `MobileContainer` 是壳层，适合统一承载登录后弹窗，不污染具体页面。

---

## 6. 功能设计

### 6.1 中央弹窗：仅重新登录触发

当前问题：
- `LiveReminderModal` 已存在，但 `MobileContainer` 没有挂载它。
- 单靠 `isAuthenticated` 无法区分「页面刷新后仍已登录」和「刚完成一次登录」。

设计：
- 在 `authContext.login()` 成功后写入 `localStorage('pending_live_reminder', '1')`。
- `MobileContainer` 首次挂载时读取该标记；如果存在，则清除标记并打开 `LiveReminderModal`。
- 弹窗内容一期使用 Mock 直播间数据，避免依赖后端开播状态查询。
- `authContext.setAuth()` 是否写入该标记：一期不写。原因是 `setAuth()` 可能用于恢复/注入登录态，不一定代表用户主动重新登录。

伪代码：

```ts
const login = async (req: LoginRequest) => {
  const result = await authService.login(req);
  localStorage.setItem('pending_live_reminder', '1');
  setToken(result.token);
  setUser(result.user);
  setIsAuthenticated(true);
};

useEffect(() => {
  if (localStorage.getItem('pending_live_reminder') !== '1') return;
  localStorage.removeItem('pending_live_reminder');
  setReminderOpen(true);
}, []);
```

### 6.2 顶部 Toast：升级现有全局 Provider

一期支持场景：
- B 截拍预警：`type: 'warning'`。
- C 被超价提醒：`type: 'danger'`，实现时可兼容现有 `error` 类型。
- D 中标结果：`type: 'success'`。

API 设计：

```ts
showToast({
  type: 'success' | 'warning' | 'danger' | 'error' | 'info' | 'loading',
  title: '恭喜中标',
  message: '您已成功拍下 XX，请尽快支付',
  duration: 3000,
  actionText: '去支付',
  onAction: () => navigate('/result'),
});
```

兼容策略：
- 保留旧调用：`showToast(message, type, duration)`，避免一次性改动所有调用点。
- 新增对象调用：用于 B/C/D 场景的标题、正文与 action。
- `services/api.ts` 仍可通过旧签名注入错误提示函数。

展示规则：
- 容器位置从居中改为移动端顶部，宽度 90%，最大宽度跟随 H5 容器。
- 同时最多展示 3 条；超过 3 条时保留队列，前一条消失后补位。
- 默认 3s 自动关闭；点击关闭或 action 后立即移除。
- `role="status"`，action 使用 `<button>`，保证基本可访问性。

Demo 触发：
- 只在 `import.meta.env.DEV` 下显示测试按钮。
- 推荐先放在 `Live/index.tsx`，因为 B/C/D 都与竞拍直播间强相关。

### 6.3 红点徽标：组件 + Mock 数据

组件 API：

```tsx
<BadgeDot count={3} max={99} />
<BadgeDot count={120} max={99} />
<BadgeDot dot />
<BadgeDot count={0} />
```

数据源：

```ts
export function useTouchpointNotifications() {
  return {
    pendingPayment: 1,
    unreadTotal: 3,
  };
}
```

应用位置：
- `BottomNav` 的「我的」入口显示 `unreadTotal`。
- `User/Index.tsx` 的「我的竞拍」入口显示 `pendingPayment`。
- 其他个人中心入口一期不显示，避免把 Mock 数字扩散成伪业务。

---

## 7. 数据流

```text
useTouchpointNotifications()
  ├─ unreadTotal ──────> BottomNav /profile BadgeDot
  └─ pendingPayment ───> User/Index.tsx /history BadgeDot

业务页面 / Demo 触发器
  └─ showToast(config) ─> ToastProvider 顶部队列

authContext.login()
  └─ pending_live_reminder ─> MobileContainer ─> LiveReminderModal
```

---

## 8. 后端依赖

一期无后端阻塞。

二期前需要确认的接口：
- 未读消息汇总接口：替换 `useTouchpointNotifications()` 内部 Mock。
- 实时推送事件：通过 WebSocket 承载 B/C/D 事件，再调用 `showToast(config)`。
- 关注主播开播查询：替换 `LiveReminderModal` 的 Mock 直播间数据。

---

## 9. 测试与验证

自动化测试建议：
- `BadgeDot` 单测：`count=0` 不渲染、`count=3` 显示 `3`、`count=120 max=99` 显示 `99+`、`dot` 显示纯红点。
- `ToastProvider` 单测：旧签名兼容、对象签名渲染 title/action、最多同时显示 3 条、action 点击会关闭并执行回调。
- `authContext` 或集成测试：登录成功写入 `pending_live_reminder`。
- `MobileContainer` 测试：有标记时打开弹窗并清除标记；刷新无标记时不弹。

人工验收：
- 进入移动端首页，底部「我的」入口显示 `3`。
- 进入个人中心，「我的竞拍」显示 `1`。
- 登录后弹出开播提醒；关闭后刷新不再弹；退出后重新登录再次弹。
- 开发环境在直播间触发 success/warning/danger 三类顶部 Toast，验证排队、自动消失、action。
- 底部导航点击行为与现有路由不变。

---

## 10. 边界与风险

1. **Toast 体系重复**：当前已有 `components/Toast`、`components/shared/Toast` 与页面级 Toast。本期只升级全局 `components/Toast`；不主动清理所有页面级 Toast，避免扩大范围。
2. **Mock 数据误上线**：Mock hook 文件名与注释必须明确，一期验收数字固定为 `3` 与 `1`。
3. **弹窗数据真实性**：一期弹窗直播间信息为 Mock，只验证触达时机，不验证真实关注主播开播状态。
4. **视觉占位图**：现有 `LiveReminderModal` 使用外部 placeholder URL，实施时应改为 CSS 占位或本地可控默认视觉，避免外链不稳定。
5. **登录入口差异**：只有 `authContext.login()` 写入重新登录标记；如果未来新增第三方登录或静默续登，需要明确是否属于“重新登录”。

---

## 11. 验收标准

- [ ] `BadgeDot` 可独立使用，覆盖纯点、数字、超限、0 不渲染四种状态。
- [ ] `BottomNav` 的「我的」入口显示 Mock 未读总数 `3`。
- [ ] `User/Index.tsx` 的「我的竞拍」入口显示 Mock 待处理数 `1`。
- [ ] `showToast(message, type, duration)` 旧签名仍可用。
- [ ] `showToast(config)` 新签名支持 `title`、`message`、`actionText`、`onAction`。
- [ ] 顶部 Toast 同时最多展示 3 条，并支持自动关闭。
- [ ] 登录成功后开播提醒弹窗只出现一次；刷新页面不重复触发。
- [ ] 不破坏现有页面布局、底部导航路由、API 错误提示。
