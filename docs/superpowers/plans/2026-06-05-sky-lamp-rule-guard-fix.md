# Sky Lamp Rule Guard Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix sky lamp activation failures caused by auctions without matching `auction_rules`.

**Architecture:** Apply a one-time local data repair for the known bad auction, add service-layer guards so missing rules return business errors instead of panics, and fix seed generation so demo auctions have matching rules. The frontend continues to call the gateway `/api/v1` entrypoint unchanged.

**Tech Stack:** Go, GORM, MySQL, Hertz, `shopspring/decimal`.

---

### Task 1: Local Data Repair

**Files:**
- Database only: local MySQL `auction.auction_rules`

- [x] **Step 1: Verify the missing rule**

Run:

```bash
mysql --protocol=TCP -uroot -proot -hlocalhost -P3306 auction \
  -e "SELECT id, auction_id, product_id FROM auction_rules WHERE product_id=993205 OR auction_id=993305;"
```

Expected: no rows before repair.

- [x] **Step 2: Insert the missing rule idempotently**

Run:

```sql
INSERT INTO auction_rules (
  auction_id, product_id, start_price, increment, cap_price, duration,
  delay_duration, max_delay_time, trigger_delay_before, created_at
)
SELECT 993305, 993205, 0.00, 100.00, NULL, 9000, 30, 180, 30, NOW(3)
WHERE NOT EXISTS (
  SELECT 1 FROM auction_rules WHERE product_id = 993205 OR auction_id = 993305
);
```

Expected: one row for `auction_id=993305`, `product_id=993205`.

### Task 2: Backend Rule Guards

**Files:**
- Modify: `backend/auction/service/sky_lamp.go`
- Modify: `backend/auction/service/bid.go`
- Test: `backend/auction/service/sky_lamp_rule_guard_test.go`
- Test: `backend/auction/service/bid_rule_guard_test.go`

- [ ] **Step 1: Write failing tests**

Add tests proving missing auction rules return clear errors instead of panics.

- [ ] **Step 2: Verify tests fail**

Run:

```bash
cd backend/auction
go test ./service -run 'Test(SkyLampStartSubscription|BidServicePlaceBid)_MissingRule' -count=1
```

Expected: FAIL before implementation.

- [ ] **Step 3: Add minimal rule nil guards**

In `SkyLampService.StartSubscription`, return `竞拍规则不存在` when `rule == nil`.

In `BidService.PlaceBid`, return `竞拍规则不存在` when `rule == nil`.

- [ ] **Step 4: Verify tests pass**

Run:

```bash
cd backend/auction
go test ./service -run 'Test(SkyLampStartSubscription|BidServicePlaceBid)_MissingRule' -count=1
```

Expected: PASS.

### Task 3: Seed Data Source Repair

**Files:**
- Modify: `backend/seed/generators.go`
- Test: `backend/seed/generators_test.go`

- [ ] **Step 1: Write failing seed test**

Add a test proving `GenerateAuctionRules` creates at most one rule per published product and covers all published products when the configured count is higher than the published product count.

- [ ] **Step 2: Verify test fails**

Run:

```bash
cd backend/seed
go test . -run TestGenerateAuctionRules_CoversPublishedProductsOnce -count=1
```

Expected: FAIL before implementation.

- [ ] **Step 3: Update seed generator**

Generate one rule for each published product first, then only add additional random rules for products without duplicate conflicts if the model allows it. The current database shape has a unique `auction_id` but `product_id` must still remain semantically unique for rule lookup.

- [ ] **Step 4: Verify seed tests pass**

Run:

```bash
cd backend/seed
go test . -run TestGenerateAuctionRules_CoversPublishedProductsOnce -count=1
```

Expected: PASS.

### Task 4: Final Verification

**Files:**
- No additional files.

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd backend/auction
go test ./service -count=1
cd ../seed
go test . -count=1
```

Expected: PASS.

- [ ] **Step 2: Verify local API no longer returns 500 for repaired auction**

Run authenticated local request through gateway with a valid buyer token and `auction_id=993305`.

Expected: status is not `500`; successful subscription or clear business error.
