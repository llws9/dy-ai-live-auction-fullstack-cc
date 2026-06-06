# 商品与竞拍生命周期解耦 Spec

- **创建日期**：2026-06-07
- **作者**：Brainstorming session（用户 + Assistant）
- **状态**：待评审
- **执行分支建议**：`feat/auction-product-lifecycle`
- **核心方案**：方案 1：每次竞拍创建独立 `auction` 记录，并保证同一商品同一时间最多一个活跃竞拍。

---

## 1. 背景与问题

Admin 当前存在两个容易混淆的业务动作：

1. 商品列表里的「发布」按钮。
2. 竞拍管理里的「创建竞拍场次」表单。

从业务本质看，「发布商品」不应直接创建竞拍。竞拍还需要选择规则模板、确定时长，并受“同一商品同一时间只能有一个活跃竞拍”的约束。因此商品发布应只表示：该商品进入可排期商品池，可以在创建竞拍场次时被选择。

同时，竞拍列表中存在的拍品，在商家的商品列表里应能追溯到对应商品。若竞拍列表可见但商品列表不可见，说明 `auction.product_id`、`auction.creator_id`、`product.owner_id` 或商品查询作用域存在不一致，必须从数据归属和创建校验上修复，不能靠前端隐藏或兜底文案掩盖。

### 1.1 已确认现状

- 商品状态当前为 `0=草稿, 1=已发布, 2=已下架`，定义在 `backend/product/model/product.go`。
- 竞拍状态当前为 `0=待开始, 1=进行中, 2=延时中, 3=已结束, 4=已取消`，定义在 `backend/auction/model/auction.go`。
- Admin 创建竞拍时，前端当前加载 `productApi.list({ status: 1 })`，再选择规则模板并调用 `productApi.applyRuleTemplate(productID, templateID)`，最后调用 `auctionApi.create({ product_id, duration })`。
- `auction-service` 的 `CreateAuction` 当前会创建独立 `auction` 记录，并写入 `creator_id`。
- `product-service` 的 Admin 商品列表已按商家 `owner_id` 做作用域过滤。
- `auction-service` 的 Admin 竞拍列表已按商家 `creator_id` 做作用域过滤。

### 1.2 目标

- 商品发布不自动创建竞拍，只让商品进入「可排期」池。
- 创建竞拍必须显式选择商品和规则模板。
- 同一商品同一时间最多存在一个活跃竞拍。
- 流拍后允许该商品再次创建新竞拍，但历史竞拍记录不可复用、不可覆盖。
- 商品列表能展示派生业务状态：可排期、竞拍中、已拍卖、流拍。
- 竞拍列表状态筛选符合商家语义：竞拍中、已拍卖、流拍、已取消。
- 修复“竞拍列表可见但商品列表无对应商品”的数据一致性问题。

### 1.3 非目标

- 不把商品状态扩展成竞拍状态的镜像。
- 不复用已结束或流拍的 `auction` 记录。
- 不允许跨服务直接查库；商品与竞拍之间仍通过 Gateway/API/RPC 或内部 API 通信。
- 不在前端绕过权限或硬编码商家身份。

---

## 2. 核心业务决策

### 2.1 商品状态只表达经营可用性

商品的持久化状态保持为：

| 状态 | 建议 UI 文案 | 含义 |
|---|---|---|
| `0 Draft` | 草稿 | 尚未进入竞拍池，不可创建竞拍 |
| `1 Published` | 可排期 | 可被创建竞拍场次选择 |
| `2 Unpublished` | 已下架 | 商家主动下架，不可创建竞拍 |

原 UI 中「发布」建议改名为：

- 按钮：`设为可排期`
- Tooltip：`进入竞拍池`
- 成功提示：`商品已进入竞拍池，可创建竞拍场次`
- 失败提示：`设为可排期失败`

原因：如果点击后并不会创建真实竞拍，也不会立刻开拍，对商家说“发布”会造成误解。

### 2.2 竞拍状态表达一次交易过程

每次竞拍是一条独立 `auction` 记录。即使流拍后重新开拍，也应创建新的 `auction` 记录，而不是复用旧记录。

原因：

- 旧竞拍的开始时间、结束时间、出价、通知、统计都是历史事实。
- 复用旧记录会破坏审计、订单、统计和通知链路。
- “流拍后重拍”本质是同一商品的新一轮交易过程。

### 2.3 活跃竞拍唯一性

同一商品同一时间最多存在一个活跃竞拍。

活跃状态定义：

- `Pending`
- `Ongoing`
- `Delayed`

终态定义：

- `Ended`
- `Cancelled`

成交与流拍的区分**不引入新字段**，直接复用已有的 `auctions.winner_id`：

```text
成交  = status=Ended AND winner_id IS NOT NULL
流拍  = status=Ended AND winner_id IS NULL
已取消 = status=Cancelled
```

原因：

- 现有结算逻辑 `auction_settlement.go` 已基于「有无中标出价」写入 `winner_id`，有出价才创建订单，否则视为流拍。
- 现有统计逻辑 `dao/statistics.go` 已用 `status=Ended AND winner_id IS NULL` 表达流拍。
- 再增加 `result_status` 会与 `winner_id` 表达同一事实，引入双写一致性风险（两字段可能不一致），属于过度设计。

因此不新增列、不新增枚举，不引入数据迁移。

### 2.4 开拍时间与直播间归属

这两项是创建竞拍环节的职责，本 Spec 明确定稿如下，避免边界悬空。

**开拍时间：仅立即开拍，不支持预约。**

- 创建竞拍即开拍：`StartTime = now`，`EndTime = now + duration`，与现状 `auction-service` 行为一致。
- `CreateAuctionRequest` 不新增 `start_time` 字段。
- `Pending` 仅表示「创建完成 → 调度器拉起 Ongoing」之间的瞬时过渡态，不作为商家可长期停留的预约态。
- 商家无法预约未来开拍；若未来需要预约，再单独立项扩展，不在本 Spec 范围。

**直播间：创建竞拍时由 `auction-service` 经 `product-service` API 获取/创建并复用。**

- 直播间归属从「发布商品」环节移除（见 §6.1），改由创建竞拍时确定。
- 直播间数据归 `product-service` 管理，`auction-service` **不得跨服务直查/直写直播间库**。创建竞拍时，`auction-service` 通过 `product-service` 内部 API/RPC 按当前商家「获取或创建 active 直播间」，拿到 `live_stream_id` 后，再在 `auction-service` 本地事务内创建 `auction` 并写入该 `live_stream_id`。
- 因此「获取/创建直播间」与「创建 auction」**不是同一个 DB 事务**：前者是跨服务调用，后者是本地事务。直播间调用失败时，整体 Fail-closed，不创建 auction。
- 商家无感知，无需先手动建直播间，前端创建竞拍表单也不新增直播间选择项。
- 一个商家复用同一个直播间承载其多场竞拍，符合「主播 = 直播间」的现状模型。
- 直播间的 active 校验从「发布商品」前移到「创建竞拍」：直播间被禁用时，创建竞拍 Fail-closed 失败，而不是发布商品失败。

---

## 3. 目标状态模型

### 3.1 商品列表展示状态

商品列表中的状态分为两层：

1. 持久化状态：来自 `products.status`。
2. 派生竞拍状态：来自该商品最新/活跃竞拍。

展示优先级：

| 优先级 | 展示文案 | 判定 |
|---:|---|---|
| 1 | 竞拍中 | 存在 `Pending/Ongoing/Delayed` 竞拍 |
| 2 | 已拍卖 | 最近终态竞拍为 `Ended AND winner_id IS NOT NULL` |
| 3 | 流拍 | 最近终态竞拍为 `Ended AND winner_id IS NULL` |
| 4 | 可排期 | `products.status=Published` 且无活跃竞拍 |
| 5 | 草稿 | `products.status=Draft` |
| 6 | 已下架 | `products.status=Unpublished` |

说明：

- 「已拍卖」和「流拍」不是商品终态，只是最近一次竞拍结果。
- 流拍商品仍可再次创建竞拍。
- 已成交商品默认不可再次创建竞拍；若未来支持同款多库存，应通过库存/批次模型表达，而不是复用同一个商品记录反复拍卖。

### 3.2 竞拍列表筛选

竞拍列表应保留底层状态细节，但默认提供商家可理解的聚合筛选。由于已定稿「立即开拍、不支持预约」，`Pending` 是瞬时过渡态，不单独作为商家主筛选项，而是并入「竞拍中」：

| UI 筛选 | 后端查询语义 |
|---|---|
| 全部场次 | 不限制状态 |
| 竞拍中 | `status IN (Pending, Ongoing, Delayed)` |
| 已拍卖 | `status=Ended AND winner_id IS NOT NULL` |
| 流拍 | `status=Ended AND winner_id IS NULL` |
| 已取消 | `status=Cancelled` |

列表卡片文案（卡片仍保留 `Pending` 的独立文案，仅筛选层做聚合）：

- `Pending`：待开始
- `Ongoing`：竞拍中
- `Delayed`：竞拍中（延时）
- `Ended + winner_id != NULL`：已拍卖
- `Ended + winner_id == NULL`：流拍
- `Cancelled`：已取消

---

## 4. 创建竞拍场次规则

### 4.1 可选商品池

创建竞拍表单的「竞拍商品」下拉框只展示满足以下条件的商品：

- `products.status=Published`
- `products.owner_id = 当前商家 X-User-ID`
- 不存在活跃竞拍：`auction.product_id = product.id AND auction.status IN (Pending, Ongoing, Delayed)`
- 最近一次终态竞拍不得是已成交（不存在 `auction.product_id = product.id AND status=Ended AND winner_id IS NOT NULL` 作为最新终态记录）；流拍 / 已取消的商品仍可进入下拉。

如果无可选商品，UI 显示：

> 暂无可排期商品。请先将商品设为可排期，或等待当前竞拍结束。

### 4.2 创建竞拍前置校验

后端必须在 `auction-service` 创建竞拍时做 Fail-closed 校验：

1. 当前用户必须是商家。
2. 商品必须存在。
3. 商品必须归属当前商家。
4. 商品必须是 `Published`。
5. 商品不得存在活跃竞拍（`status IN (Pending, Ongoing, Delayed)`）。
6. 商品最近一次终态竞拍不得是已成交（`status=Ended AND winner_id IS NOT NULL`）；流拍（`Ended AND winner_id IS NULL`）或取消（`Cancelled`）后允许再次创建新竞拍。
7. 商品必须已绑定有效竞拍规则，且该规则归属当前商家。规则的应用在创建竞拍**之前**由前端单独调用 `applyRuleTemplate` 完成（见 §4.3），`auction-service` 在创建时只校验「商品已有归属当前商家的有效规则」，不接收 `template_id`、不负责应用模板。
8. 当前商家的直播间必须处于 active 状态（直播间被禁用时 Fail-closed 失败）。

其中 2、3、4 需要通过 `product-service` API/RPC 获取，不允许 `auction-service` 跨服务直查产品库。

### 4.3 创建竞拍的编排与写入

创建竞拍是一个跨服务编排，不是单一本地事务：

1. （前端，创建竞拍之前）调用 `applyRuleTemplate(productID, templateID)`，把商家选定的规则模板应用到商品。该步骤校验模板归属当前商家。
2. （`auction-service`）通过 `product-service` 内部 API/RPC 获取/创建当前商家的 active 直播间，拿到 `live_stream_id`（active 校验见 §4.2 第 8 条）。
3. （`auction-service`，本地事务）写入新 `auction` 记录：`product_id`、`creator_id = 当前商家`、`live_stream_id = 直播间 ID`、`status = Pending`、`StartTime = now`、`EndTime = now + duration`。

说明：

- 步骤 2 是跨服务调用，步骤 3 是 `auction-service` 本地事务，二者**不在同一 DB 事务**。任一前置步骤失败则整体 Fail-closed，不创建 auction。
- 直播间创建逻辑从 `product-service` 的 `PublishProduct` 迁移至此（见 §6.1），但实际建库动作仍在 `product-service` 内执行，`auction-service` 只通过 API 触发。

### 4.4 并发约束

仅靠“查询无活跃竞拍后再插入”不够，两个并发请求可能同时通过检查，必须用数据库唯一约束兜底。

**定稿实现：`auctions` 表生成列 + 唯一索引。**

- 在 `auctions` 增加一个存储型生成列 `active_product_key`：
  - 当 `status IN (0,1,2)`（Pending/Ongoing/Delayed）时，值 = `product_id`；
  - 当 `status IN (3,4)`（Ended/Cancelled）时，值 = `NULL`。
- 对 `active_product_key` 建唯一索引 `uk_active_product`。
- MySQL 唯一索引允许多个 `NULL`，因此同一商品的历史终态竞拍互不冲突，但同时只允许一条活跃竞拍存在。

```sql
ALTER TABLE auctions
  ADD COLUMN active_product_key BIGINT AS
    (CASE WHEN status IN (0,1,2) THEN product_id ELSE NULL END) STORED,
  ADD UNIQUE KEY uk_active_product (active_product_key);
```

实现要求：

- `auction-service` 创建竞拍仍在事务中完成，先做 §4.2 应用层 Fail-closed 校验（快速失败、给出友好错误），唯一索引作为并发兜底，二者不互斥。
- 竞拍进入终态（`Ended/Cancelled`）时，生成列自动变为 `NULL`，无需手动释放锁，也不需要额外锁表。

**不采用** `auction_product_locks` 锁表与 Redis 分布式锁：前者需要手动维护释放逻辑、易因漏释放造成死锁；后者只能作弱保证，最终仍以 DB 为准，徒增复杂度。生成列方案以 DB 事实为唯一真相，自动随状态机收敛。

Fail-closed 行为：

- 应用层检测到活跃竞拍时返回业务错误：`该商品已有待开始或进行中的竞拍场次`。
- 并发下命中唯一索引冲突（`Duplicate entry`）时，归一化为同一业务错误，不向上抛 500。

---

## 5. 数据一致性要求

### 5.1 竞拍和商品归属一致

创建竞拍后必须满足：

```text
auction.creator_id == product.owner_id == 当前商家 X-User-ID
```

如果历史数据不满足，应通过数据修复脚本处理，不应在 UI 中隐藏问题。

### 5.2 竞拍列表商品摘要

竞拍列表回填商品摘要时，如果 `product-service` 内部批量接口找不到某个 `product_id`，后端应暴露可观测错误或明确标记异常，不应把它静默渲染成 `竞拍场次 #id` 后继续假装正常。

推荐行为：

- 管理端列表：展示 `商品不存在或无权限` 的异常标签，并在服务日志中记录 `auction_id/product_id/creator_id`。
- 修复脚本：提供只读检查命令，列出 `auction.product_id` 无法在同商家商品池中找到的记录。

### 5.3 对 test-service / fixture 链路的影响与适配

`backend/test` 是一条独立于 Admin 前端的创建链路（压测 `pressure`、防狙击 `antisnipe`、E2E `e2e`），通过 SDK 直连 Gateway API 创建拍品和竞拍。本次改造会打到它，必须同步适配，不能只改 Admin。

现状（已核对）：

- 三条链路均以 `RoleMerchant` + 固定 seller 创建商品（`status=1` 直接 Published），随后 `CreateAuctionRule` → `CreateAuctionAs`，**不经过** `PublishProduct`。
- 因此 §6.1 删除 `PublishProduct` 副作用**不影响** test 链路。
- 现状每个竞拍 fixture 都用「时间戳命名的新商品」，天然满足「一品一活跃竞拍」。

适配要求：

1. **一品一活跃竞拍是强约束**：fixture 必须保证每个活跃竞拍对应独立商品；严禁改成复用同一商品创建多个竞拍，否则会命中 §4.4 的 `uk_active_product` 唯一索引并失败。此约束需在 fixture 代码注释与本 Spec 中固化。
2. **直播间自动创建后无需 fixture 介入**：§2.4/§4.3 决定直播间由 `auction-service` 在创建竞拍时经 `product-service` API 获取/创建，因此 `antisnipe`/`pressure`/`e2e` 三条链路**无需新增** `CreateLiveStream` 步骤；固定 seller 第一次创建竞拍时自动建直播间，后续复用。
3. **归属一致性同样适用**：fixture 创建的 `auction.creator_id` 必须等于其商品 `owner_id`（固定 seller），§5.1 的只读检查脚本应能覆盖 test 链路产生的数据。
4. **回归验证**：改造后必须重跑 `antisnipe` / `pressure` / `e2e` 三个场景的既有测试，确认创建竞拍校验收紧（§4.2）和唯一索引（§4.4）落地后链路仍通。

---

## 6. API 与前端改造建议

### 6.1 product-service：清理 PublishProduct 副作用（必须）

现状 `backend/product/service/product.go` 的 `PublishProduct` 与新语义冲突，发布时会：

1. `GetOrCreateByCreatorID` 创建/绑定直播间；
2. 校验直播间 `IsActive()`，否则发布失败；
3. 计算 `now+30min` 默认竞拍开始时间（虽未真正建竞拍，仍是竞拍语义残留）。

按 §2.1「发布 = 仅进入可排期池，不创建竞拍、不绑定直播间、不涉及开拍时间」，实施时必须：

- 移除 `PublishProduct` 中的直播间创建/校验与默认开始时间逻辑，只保留「`Draft → Published` 状态流转」。
- 直播间创建/active 校验与开拍时间迁移到「创建竞拍」环节（`auction-service`，见 §2.4 与 §4.3），不是凭空删除。
- 否则会出现「文案改了、后端没改」：商家点「设为可排期」仍可能因直播间未激活而失败，且 30min 逻辑沦为死代码。

### 6.2 Product API

新增或扩展 Admin 商品列表响应字段：

```json
{
  "id": 1001,
  "status": 1,
  "display_status": "auctioning",
  "display_status_label": "竞拍中",
  "active_auction_id": 993511,
  "latest_auction_id": 993511,
  "latest_auction_result": "unsold"
}
```

- `latest_auction_result` 由后端依据最近终态竞拍的 `winner_id` 派生（`sold` / `unsold`），**不对应任何持久化列**。
- 建议由后端聚合生成，前端不应自行拉全量竞拍后拼装。

### 6.3 Auction API

创建竞拍请求保持以 `product_id` 为核心，但后端不信任前端筛选结果：

```json
{
  "product_id": 1001,
  "duration": 3600
}
```

- 请求体不含 `template_id`、`live_stream_id`、`start_time`：规则在创建前由前端 `applyRuleTemplate` 应用到商品（§4.3 步骤 1），直播间由 `auction-service` 经 `product-service` API 获取/创建（§4.3 步骤 2），开拍时间固定为 `now`（§2.4）。
- 创建竞拍的最终校验（含规则已绑定且归属当前商家）必须在 `auction-service` 内完成，不信任前端筛选结果。

### 6.4 UI 文案

商品列表：

- 主按钮：`新增商品`
- 草稿行操作：`设为可排期`
- 可排期行操作：`创建竞拍`
- 竞拍中行操作：`查看竞拍`
- 流拍行操作：`重新创建竞拍`
- 已拍卖行操作：`查看结果`
- 已下架行操作：`重新上架`

创建竞拍表单：

- 标题：`创建竞拍场次`
- 说明：`选择可排期商品和规则模板，创建一场真实竞拍。`
- 商品下拉：`竞拍商品`
- 规则下拉：`规则模板`
- 提交按钮：`确认创建竞拍`

错误提示：

- 无商品：`暂无可排期商品`
- 活跃冲突：`该商品已有待开始或进行中的竞拍场次`
- 商品归属异常：`商品不存在或不属于当前商家`
- 规则异常：`规则模板不存在或不属于当前商家`

---

## 7. 测试要求

### 7.1 后端单测

- 商家只能用自己的 `Published` 商品创建竞拍。
- 草稿商品、下架商品、他人商品创建竞拍均失败。
- 同一商品已有 `Pending/Ongoing/Delayed` 竞拍时，创建失败。
- 同一商品最近竞拍为流拍时，可创建新的竞拍记录。
- 同一商品最近竞拍为已成交（`winner_id IS NOT NULL`）时，创建失败。
- 商品未绑定归属当前商家的有效规则时，创建失败。
- 并发创建同一商品竞拍时，最多一条成功（验证生成列唯一索引兜底）。
- 竞拍进入终态后，生成列释放，同一商品可再次创建竞拍。
- 创建竞拍时经 `product-service` API 获取/创建直播间并写入 `auction.live_stream_id`；直播间被禁用时创建失败。
- Admin 商品列表能返回派生展示状态。
- 竞拍列表 `竞拍中` 筛选覆盖 `Pending + Ongoing + Delayed`。
- 竞拍列表能区分 `已拍卖` 与 `流拍`。

### 7.2 前端单测

- 商品列表显示 `设为可排期` 而不是误导性的 `发布`。
- 创建竞拍下拉只渲染后端返回的可选商品。
- 无可选商品时展示空态文案。
- 竞拍状态 Badge 正确映射：竞拍中、竞拍中（延时）、已拍卖、流拍。
- 活跃冲突错误展示中文提示。

### 7.3 数据检查

需要提供一条只读数据检查脚本或命令，验证：

- `auctions.product_id` 是否都能在 `products.id` 找到。
- `auctions.creator_id` 是否等于对应 `products.owner_id`。
- 每个商品是否最多存在一个活跃竞拍。

### 7.4 test-service 链路回归

- 重跑 `antisnipe` / `pressure` / `e2e` 三个场景，确认 §4.2 校验收紧与 §4.4 唯一索引落地后链路仍通。
- 确认 fixture 不复用商品创建多个活跃竞拍（否则命中 `uk_active_product`）。

---

## 8. 验收标准

- 点击商品列表「设为可排期」后，商品进入创建竞拍下拉框，但不会自动创建竞拍。
- 创建竞拍必须选择商品和规则模板。
- 同一商品已有待开始/进行中/延时中竞拍时，无法再次创建竞拍。
- 流拍后，该商品可以再次创建新竞拍，旧竞拍记录保留。
- 商品列表能看到竞拍列表中每个场次对应的商品，或明确暴露异常数据。
- 商品列表和竞拍列表的状态文案不再混用。
- 所有前端流量仍经 `gateway-service` 的 `/api/v1` 入口。

---

## 9. 风险与取舍

- 若只做前端筛选，不做后端活跃唯一约束，并发下仍会出现一品多拍，不能接受。
- 若复用流拍竞拍记录，会破坏历史审计和统计，不能接受。
- 若把 `竞拍中/已拍卖/流拍` 写入 `products.status`，会导致商品经营状态与竞拍过程状态耦合，不利于重拍和历史追踪。
- 成交与流拍直接复用 `winner_id` 判定，不新增 `result_status`：现有结算与统计已以 `winner_id` 为事实来源，新增字段只会引入双写一致性风险。
- 生成列唯一索引依赖 MySQL「唯一索引允许多 `NULL`」特性；若未来迁移到不支持该语义的存储，需改用部分索引或等价机制，此为已知约束。
- 开拍时间定为「仅立即开拍、不支持预约」：放弃预约能力换取最小改动与现状一致；`Pending` 退化为瞬时过渡态。若后续要预约，需扩展 `CreateAuctionRequest` 与调度，属新立项。
- 直播间定为「创建竞拍时自动建/复用」：商家无感、迁移成本最低，但意味着「主播=直播间」模型被固化，一个商家的所有竞拍共享一个直播间；若未来要多直播间/选直播间，需在创建竞拍表单与 API 显式引入直播间选择。
