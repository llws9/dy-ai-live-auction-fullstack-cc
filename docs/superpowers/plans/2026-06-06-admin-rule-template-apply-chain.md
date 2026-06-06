# Admin Rule Template Apply Chain Plan

## Goal

让商家在 Admin 端维护的竞拍规则模板不仅能落 `auction_rule_templates` 表，还能在创建竞拍场次时被选择并应用到商品的真实 `auction_rules`，使后续 auction-service 出价、详情和点天灯链路读取到同一套权威规则。

## Current State

- `frontend/admin/src/pages-new/AuctionRules.tsx` 已对接 `/api/v1/admin/auction-rule-templates`，模板 CRUD 可以落表。
- 后端 product-service 已有 `POST /api/v1/products/:id/rules`，但该公开路由没有商家 owner 约束，不能直接作为 Admin 模板应用入口。
- Admin 端 `AuctionList.tsx` 的“创建竞拍场次”按钮仍为 disabled，没有选择商品、选择模板、创建竞拍的闭环。

## Target Architecture

1. product-service 新增 Admin 内部接口：
   - `POST /api/v1/admin/products/:id/apply-rule-template`
   - 仅允许 merchant actor。
   - 校验商品属于当前商家。
   - 校验模板属于当前商家。
   - 将模板字段转换为 `CreateAuctionRuleRequest` 并写入/更新 `auction_rules`。

2. gateway-service 新增路由：
   - `POST /api/v1/admin/products/:id/apply-rule-template`
   - JWT 后要求 `RequireMerchantOnly()`。
   - 经 `adminProductProxy` 透传 `X-Internal-Token`、`X-User-ID`、`X-User-Role`。

3. Admin 前端新增创建竞拍表单：
   - 商家在 `/auction/list` 点击“创建竞拍场次”。
   - 选择商品、规则模板、竞拍时长。
   - 提交顺序固定为：应用模板到商品规则 -> 创建竞拍。
   - 前端只调用 Gateway `/api/v1`，不直连子服务。

## Contract

### Apply Rule Template

Request:

```http
POST /api/v1/admin/products/:id/apply-rule-template
Authorization: Bearer <merchant-jwt>
Content-Type: application/json

{
  "template_id": 123
}
```

Response:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 456,
    "product_id": 789,
    "start_price": 100,
    "increment": 10,
    "cap_price": 1000,
    "duration": 3600,
    "delay_duration": 30,
    "max_delay_time": 180,
    "trigger_delay_before": 30
  }
}
```

## Decisions

- 不让前端直接调用公开 `POST /products/:id/rules`，因为该路由缺少 Admin owner scope。
- 不在 auction-service 创建竞拍接口中直接接受 `template_id`，因为模板和商品规则归属 product-service；跨服务不直接查库。
- 创建竞拍前先应用模板，保持 auction-service 继续读取 `auction_rules` 的既有运行时模型。

## Verification

- product-service handler/service tests cover owner scope and template conversion.
- gateway router test covers merchant-only proxy and internal token forwarding.
- Admin tests cover create flow ordering: `applyRuleTemplate` before `auctionApi.create`.
- Run affected backend and frontend test suites plus Admin build.
