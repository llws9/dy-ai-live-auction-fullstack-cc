# Admin Rule Template Apply Chain Tasks

## T001 - Product Service Apply Template API

**Scope**

- Add service method to apply an owner-scoped `auction_rule_templates` record to an owner-scoped product.
- Add handler for `POST /api/v1/admin/products/:id/apply-rule-template`.
- Register product-service internal admin route.

**Allowed Files**

- `backend/product/service/auction_rule_template.go`
- `backend/product/service/auction_rule_template_test.go`
- `backend/product/handler/auction_rule_template.go`
- `backend/product/handler/auction_rule_template_test.go`
- `backend/product/main.go`

**Expected Tests**

```bash
cd backend/product && go test ./service ./handler -run 'AuctionRuleTemplate|ApplyRuleTemplate' -count=1
```

## T002 - Gateway Route

**Scope**

- Add merchant-only gateway route for `POST /api/v1/admin/products/:id/apply-rule-template`.
- Ensure route forwards through admin product proxy with internal token and merchant role.

**Allowed Files**

- `backend/gateway/router/router.go`
- `backend/gateway/router/admin_rule_template_route_test.go`

**Expected Tests**

```bash
cd backend/gateway && go test ./router -run 'AdminRuleTemplate|ApplyRuleTemplate' -count=1
```

## T003 - Admin Create Auction UI

**Scope**

- Enable merchant “创建竞拍场次” button.
- Add create form on `AuctionList` with product selector, rule template selector and duration.
- Submit sequence: `productApi.applyRuleTemplate(product_id, template_id)` then `auctionApi.create({ product_id, duration })`.
- Add frontend API wrappers and tests.

**Allowed Files**

- `frontend/admin/src/pages-new/AuctionList.tsx`
- `frontend/admin/src/pages-new/__tests__/AuctionList.createAuction.test.tsx`
- `frontend/admin/src/shared/api/product.ts`
- `frontend/admin/src/shared/api/auction.ts`
- `frontend/admin/src/shared/api/__tests__/product.test.ts`
- `frontend/admin/src/shared/api/__tests__/auction.test.ts`

**Expected Tests**

```bash
cd frontend/admin && npm test -- --runTestsByPath src/pages-new/__tests__/AuctionList.createAuction.test.tsx src/shared/api/__tests__/product.test.ts src/shared/api/__tests__/auction.test.ts --runInBand
```

## Final Verification

```bash
cd backend/product && go test ./service ./handler -run 'AuctionRuleTemplate|ApplyRuleTemplate' -count=1
cd backend/gateway && go test ./router -run 'AdminRuleTemplate|ApplyRuleTemplate' -count=1
cd frontend/admin && npm test -- --runInBand
cd frontend/admin && npm run build
```
