# Admin Role Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build backend data ownership, role-aware management APIs, and service-side permission guards for Admin frontend platform-admin vs merchant visibility.

**Architecture:** Keep public H5 APIs unchanged and add/upgrade management APIs under `/api/v1/admin/*` for the Admin frontend. Gateway remains the traffic and JWT entry point; downstream services must still enforce `X-User-ID` and `X-User-Role` so direct service access or route mistakes fail closed. Product service owns product, live stream, order, statistics, and merchant auction-rule-template scope; Auction service owns auction and fixed-price runtime scope.

**Tech Stack:** Go 1.24+, Hertz, GORM, MySQL, Redis, `shopspring/decimal`, existing Gateway/Product/Auction microservices.

---

## 0. Scope And Contracts

### 0.1 Role Semantics

| JWT `user_role` | Gateway header `X-User-Role` | Meaning |
|---:|---|---|
| `0` | `user` | C 端普通用户 |
| `1` | `merchant` | 商家/主播 |
| `2` | `admin` | 平台管理员 |

### 0.2 Data Scope

| Role | Scope | Backend rule |
|---|---|---|
| `admin` | `all` | Can read platform-wide management data, but cannot perform merchant operating actions unless the endpoint is explicitly a platform governance action. |
| `merchant` | `owner_only` | Can read and write only resources owned by `X-User-ID`. |
| `user` | none | Cannot access Admin frontend management APIs. |

### 0.3 API Contract Summary

All endpoints below are under Gateway prefix `/api/v1`.

| Endpoint | Method | Roles | Data scope | Purpose |
|---|---|---|---|---|
| `/admin/products` | `GET` | admin, merchant | admin all; merchant own | Admin frontend product list |
| `/admin/products/:id` | `GET` | admin, merchant | admin all; merchant own | Admin frontend product detail |
| `/admin/products` | `POST` | merchant only | own | Merchant creates product |
| `/admin/products/:id` | `PUT` | merchant only | own | Merchant edits product |
| `/admin/products/:id` | `DELETE` | merchant only | own | Merchant deletes draft product |
| `/admin/auction-rule-templates` | `GET` | merchant only | own | Merchant lists reusable auction parameter templates |
| `/admin/auction-rule-templates/:id` | `GET` | merchant only | own | Merchant reads one template |
| `/admin/auction-rule-templates` | `POST` | merchant only | own | Merchant creates template |
| `/admin/auction-rule-templates/:id` | `PUT` | merchant only | own | Merchant updates template |
| `/admin/auction-rule-templates/:id` | `DELETE` | merchant only | own | Merchant deletes template |
| `/admin/live-streams` | `GET` | admin, merchant | admin all; merchant own | Admin frontend live room list |
| `/admin/live-streams/:id` | `GET` | admin, merchant | admin all; merchant own | Admin frontend live room detail |
| `/admin/live-streams` | `POST` | merchant only | own | Merchant creates live room |
| `/admin/live-streams/:id` | `PUT` | merchant only | own | Merchant edits live room |
| `/admin/live-streams/:id/end` | `PUT` | admin only | all | Platform governance action |
| `/admin/live-streams/:id/ban` | `PUT` | admin only | all | Platform governance action |
| `/admin/orders` | `GET` | admin, merchant | admin all; merchant seller scope | Admin frontend order list |
| `/admin/orders/:id` | `GET` | admin, merchant | admin all; merchant seller scope | Admin frontend order detail |
| `/orders/:id/ship` | `PUT` | merchant only | seller scope | Merchant ships own order |
| `/statistics/overview` | `GET` | admin, merchant | admin all; merchant seller scope | Dashboard overview |
| `/statistics/auctions` | `GET` | admin, merchant | admin all; merchant creator scope | Auction statistics |
| `/statistics/revenue` | `GET` | admin, merchant | admin all; merchant seller scope | Revenue statistics |
| `/statistics/users` | `GET` | admin only | all | Platform user statistics |
| `/admin/auctions` | `GET` | admin, merchant | admin all; merchant creator scope | Admin frontend auction list |
| `/admin/auctions/:id` | `GET` | admin, merchant | admin all; merchant creator scope | Admin frontend auction detail |
| `/admin/auctions` | `POST` | merchant only | own | Merchant creates auction |
| `/admin/auctions/:id/cancel` | `PUT` | merchant only | own | Merchant cancels own auction |
| `/fixed-price/items` | `POST` | merchant only | own | Merchant lists fixed-price item |
| `/fixed-price/items/:id/offline` | `POST` | merchant only | own | Merchant offlines own fixed-price item |

---

## 1. File Structure

### Gateway

- Modify: `backend/gateway/middleware/rbac.go` - add exact-role and role-set middleware.
- Modify: `backend/gateway/router/router.go` - route Admin frontend management endpoints to Product/Auction with correct middleware and internal token.
- Test: `backend/gateway/middleware/rbac_test.go` - verify admin is not accepted by merchant-only middleware.
- Test: `backend/gateway/router/admin_role_routes_test.go` - verify protected route registration and role behavior.

### Product Service

- Create: `backend/product/model/auction_rule_template.go` - merchant reusable auction parameter template model.
- Modify: `backend/product/model/product.go` - add `OwnerID`.
- Modify: `backend/product/model/order.go` - add `SellerID`.
- Modify: `backend/product/main.go` - include new AutoMigrate model and register role-aware Admin frontend routes.
- Create: `backend/migrations/2026060401_admin_role_scope.up.sql` - explicit schema migration.
- Create: `backend/migrations/2026060401_admin_role_scope.down.sql` - rollback migration.
- Create: `backend/product/handler/auth_context.go` - parse `X-User-ID` and `X-User-Role` from Gateway headers.
- Modify: `backend/product/handler/product.go` - add role-aware admin product handlers.
- Modify: `backend/product/service/product.go` - create/update/list/get/delete with owner scope.
- Modify: `backend/product/dao/product.go` - scoped product queries.
- Create: `backend/product/handler/auction_rule_template.go` - template CRUD handlers.
- Create: `backend/product/service/auction_rule_template.go` - template service with decimal validation.
- Create: `backend/product/dao/auction_rule_template.go` - owner-scoped template DAO.
- Modify: `backend/product/handler/live_stream.go` - role-aware list/detail/create/update plus admin-only governance actions.
- Modify: `backend/product/service/live_stream.go` - owner-scoped live stream operations.
- Modify: `backend/product/dao/live_stream.go` - list/detail/update by owner.
- Modify: `backend/product/handler/order.go` and `backend/product/handler/order_admin.go` - merchant seller-scoped admin order reads and ship guard.
- Modify: `backend/product/service/order.go`, `backend/product/service/order_admin.go`, `backend/product/dao/order.go`, `backend/product/dao/order_admin.go` - seller-scoped order operations.
- Modify: `backend/product/handler/statistics.go`, `backend/product/service/statistics.go`, `backend/product/dao/statistics.go` - role-aware statistics.
- Tests: focused tests in `backend/product/handler/*_test.go`, `backend/product/service/*_test.go`, `backend/product/dao/*_test.go`.

### Auction Service

- Modify: `backend/auction/handler/auction.go` - add Admin frontend role-aware auction handlers or branch existing handlers by header.
- Modify: `backend/auction/service/auction.go` - creator-scoped auction list/detail/cancel.
- Modify: `backend/auction/dao/auction.go` - creator-scoped filters.
- Modify: `backend/auction/handler/fixed_price.go` - reject admin writes defensively.
- Tests: `backend/auction/handler/auction_admin_scope_test.go`, `backend/auction/service/auction_scope_test.go`, `backend/auction/handler/fixed_price_test.go`.

### Documentation

- Modify: `docs/superpowers/specs/2026-06-04-admin-role-visibility-design.md` only if implementation discovers a confirmed contract conflict.
- Create or modify: API contract docs only if the repository already has a backend API doc file for Admin APIs; otherwise this plan remains the SSOT for implementation.

---

## 2. Database Changes

### 2.1 Migration Up SQL

Create `backend/migrations/2026060401_admin_role_scope.up.sql`:

```sql
ALTER TABLE products
  ADD COLUMN owner_id BIGINT NULL COMMENT 'merchant user id owning this product' AFTER id,
  ADD INDEX idx_products_owner_id (owner_id),
  ADD INDEX idx_products_owner_status_created (owner_id, status, created_at);

ALTER TABLE orders
  ADD COLUMN seller_id BIGINT NULL COMMENT 'merchant user id owning the sold product at order creation time' AFTER product_id,
  ADD INDEX idx_orders_seller_id (seller_id),
  ADD INDEX idx_orders_seller_status_created (seller_id, status, created_at);

UPDATE orders o
JOIN products p ON p.id = o.product_id
SET o.seller_id = p.owner_id
WHERE o.seller_id IS NULL AND p.owner_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS auction_rule_templates (
  id BIGINT NOT NULL AUTO_INCREMENT,
  owner_id BIGINT NOT NULL COMMENT 'merchant user id owning this template',
  name VARCHAR(128) NOT NULL,
  start_price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
  increment DECIMAL(10,2) NOT NULL,
  cap_price DECIMAL(10,2) NULL,
  duration INT NOT NULL,
  delay_duration INT NOT NULL DEFAULT 30,
  max_delay_time INT NOT NULL DEFAULT 180,
  trigger_delay_before INT NOT NULL DEFAULT 30,
  is_default TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_rule_templates_owner_id (owner_id),
  KEY idx_rule_templates_owner_default (owner_id, is_default),
  UNIQUE KEY uniq_rule_templates_owner_name (owner_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### 2.2 Migration Down SQL

Create `backend/migrations/2026060401_admin_role_scope.down.sql`:

```sql
DROP TABLE IF EXISTS auction_rule_templates;

ALTER TABLE orders
  DROP INDEX idx_orders_seller_status_created,
  DROP INDEX idx_orders_seller_id,
  DROP COLUMN seller_id;

ALTER TABLE products
  DROP INDEX idx_products_owner_status_created,
  DROP INDEX idx_products_owner_id,
  DROP COLUMN owner_id;
```

### 2.3 Model Changes

Product model:

```go
type Product struct {
	ID          int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	OwnerID     *int64        `json:"owner_id,omitempty" gorm:"index"`
	Name        string        `json:"name" gorm:"type:varchar(128);not null"`
	Description string        `json:"description" gorm:"type:text"`
	Images      JSONArray     `json:"images" gorm:"type:json"`
	CategoryID  *int64        `json:"category_id" gorm:"index"`
	Status      ProductStatus `json:"status" gorm:"type:tinyint;default:0"`
	CreatedAt   time.Time     `json:"created_at" gorm:"autoCreateTime"`
}
```

Order model:

```go
type Order struct {
	ID          int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	AuctionID   int64           `json:"auction_id" gorm:"not null;uniqueIndex"`
	ProductID   int64           `json:"product_id" gorm:"not null;index"`
	SellerID    *int64          `json:"seller_id,omitempty" gorm:"index"`
	WinnerID    int64           `json:"winner_id" gorm:"not null;index"`
	FinalPrice  decimal.Decimal `json:"final_price" gorm:"type:decimal(10,2);not null"`
	Status      OrderStatus     `json:"status" gorm:"type:tinyint;default:0"`
	PaidAt      *time.Time      `json:"paid_at,omitempty"`
	ShippedAt   *time.Time      `json:"shipped_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}
```

Auction rule template model:

```go
package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type AuctionRuleTemplate struct {
	ID                 int64            `json:"id" gorm:"primaryKey;autoIncrement"`
	OwnerID            int64            `json:"owner_id" gorm:"not null;index;uniqueIndex:uniq_rule_templates_owner_name,priority:1"`
	Name               string           `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_rule_templates_owner_name,priority:2"`
	StartPrice         decimal.Decimal  `json:"start_price" gorm:"type:decimal(10,2);not null;default:0"`
	Increment          decimal.Decimal  `json:"increment" gorm:"type:decimal(10,2);not null"`
	CapPrice           *decimal.Decimal `json:"cap_price,omitempty" gorm:"type:decimal(10,2)"`
	Duration           int              `json:"duration" gorm:"not null"`
	DelayDuration      int              `json:"delay_duration" gorm:"not null;default:30"`
	MaxDelayTime        int              `json:"max_delay_time" gorm:"not null;default:180"`
	TriggerDelayBefore int              `json:"trigger_delay_before" gorm:"not null;default:30"`
	IsDefault           bool             `json:"is_default" gorm:"not null;default:false"`
	CreatedAt           time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}

func (AuctionRuleTemplate) TableName() string {
	return "auction_rule_templates"
}
```

---

## 3. Task 1: Gateway Exact Role Middleware

**Files:**
- Modify: `backend/gateway/middleware/rbac.go`
- Create: `backend/gateway/middleware/rbac_test.go`

- [ ] **Step 1: Write failing middleware tests**

Add tests that prove admin does not pass merchant-only routes:

```go
package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/require"
)

func TestRequireMerchantOnlyRejectsAdmin(t *testing.T) {
	c := app.NewContext(0)
	c.Set("user_role", 2)

	handler := RequireMerchantOnly()
	handler(context.Background(), c)

	require.True(t, c.IsAborted())
	require.Equal(t, 403, c.Response.StatusCode())
}

func TestRequireMerchantOrAdminAcceptsBoth(t *testing.T) {
	for _, role := range []int{1, 2} {
		c := app.NewContext(0)
		c.Set("user_role", role)

		handler := RequireMerchantOrAdmin()
		handler(context.Background(), c)

		require.False(t, c.IsAborted())
	}
}

func TestRequireMerchantOrAdminRejectsUser(t *testing.T) {
	c := app.NewContext(0)
	c.Set("user_role", 0)

	handler := RequireMerchantOrAdmin()
	handler(context.Background(), c)

	require.True(t, c.IsAborted())
	require.Equal(t, 403, c.Response.StatusCode())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend/gateway
go test ./middleware -run 'TestRequireMerchant' -count=1
```

Expected: compile failure because `RequireMerchantOnly` and `RequireMerchantOrAdmin` do not exist.

- [ ] **Step 3: Implement exact-role middleware**

Add to `backend/gateway/middleware/rbac.go`:

```go
func RequireExactRole(role int) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userRole := c.GetInt("user_role")
		if userRole != role {
			c.JSON(403, map[string]interface{}{
				"code":    403,
				"message": "权限不足",
			})
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

func RequireAnyRole(roles ...int) app.HandlerFunc {
	allowed := make(map[int]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(ctx context.Context, c *app.RequestContext) {
		userRole := c.GetInt("user_role")
		if _, ok := allowed[userRole]; !ok {
			c.JSON(403, map[string]interface{}{
				"code":    403,
				"message": "权限不足",
			})
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

func RequireMerchantOnly() app.HandlerFunc {
	return RequireExactRole(1)
}

func RequireMerchantOrAdmin() app.HandlerFunc {
	return RequireAnyRole(1, 2)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd backend/gateway
go test ./middleware -run 'TestRequireMerchant' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/gateway/middleware/rbac.go backend/gateway/middleware/rbac_test.go
git commit -m "feat: add exact role gateway middleware"
```

---

## 4. Task 2: Product Service Auth Context Helpers

**Files:**
- Create: `backend/product/handler/auth_context.go`
- Create: `backend/product/handler/auth_context_test.go`

- [ ] **Step 1: Write failing helper tests**

```go
package handler

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/require"
)

func TestReadAdminActorMerchant(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	actor, ok := readAdminActor(c)

	require.True(t, ok)
	require.Equal(t, int64(1001), actor.UserID)
	require.Equal(t, "merchant", actor.Role)
	require.True(t, actor.IsMerchant())
	require.False(t, actor.IsAdmin())
}

func TestRequireMerchantActorRejectsAdmin(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")

	_, ok := requireMerchantActor(c)

	require.False(t, ok)
	require.Equal(t, 403, c.Response.StatusCode())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/product
go test ./handler -run 'TestReadAdminActor|TestRequireMerchantActor' -count=1
```

Expected: compile failure because helpers do not exist.

- [ ] **Step 3: Implement helpers**

```go
package handler

import (
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	roleAdmin    = "admin"
	roleMerchant = "merchant"
)

type AdminActor struct {
	UserID int64
	Role   string
}

func (a AdminActor) IsAdmin() bool {
	return a.Role == roleAdmin
}

func (a AdminActor) IsMerchant() bool {
	return a.Role == roleMerchant
}

func readAdminActor(c *app.RequestContext) (AdminActor, bool) {
	role := string(c.GetHeader("X-User-Role"))
	userIDRaw := string(c.GetHeader("X-User-ID"))
	userID, err := strconv.ParseInt(userIDRaw, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return AdminActor{}, false
	}
	if role != roleAdmin && role != roleMerchant {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
		return AdminActor{}, false
	}
	return AdminActor{UserID: userID, Role: role}, true
}

func requireMerchantActor(c *app.RequestContext) (AdminActor, bool) {
	actor, ok := readAdminActor(c)
	if !ok {
		return AdminActor{}, false
	}
	if !actor.IsMerchant() {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "平台管理员不具备代运营权限"})
		return AdminActor{}, false
	}
	return actor, true
}

func requireAdminActor(c *app.RequestContext) (AdminActor, bool) {
	actor, ok := readAdminActor(c)
	if !ok {
		return AdminActor{}, false
	}
	if !actor.IsAdmin() {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足：需要管理员权限"})
		return AdminActor{}, false
	}
	return actor, true
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend/product
go test ./handler -run 'TestReadAdminActor|TestRequireMerchantActor' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/product/handler/auth_context.go backend/product/handler/auth_context_test.go
git commit -m "feat: add product admin actor helpers"
```

---

## 5. Task 3: Product Ownership Schema And Scoped Product APIs

**Files:**
- Create: `backend/migrations/2026060401_admin_role_scope.up.sql`
- Create: `backend/migrations/2026060401_admin_role_scope.down.sql`
- Modify: `backend/product/model/product.go`
- Modify: `backend/product/dao/product.go`
- Modify: `backend/product/service/product.go`
- Modify: `backend/product/handler/product.go`
- Modify: `backend/product/main.go`
- Modify: `backend/gateway/router/router.go`
- Tests: `backend/product/handler/product_test.go`, `backend/product/service/product_test.go`, `backend/product/dao/product_test.go`

- [ ] **Step 1: Write failing DAO tests**

Add tests that create products for two owners and verify merchant scope returns only one owner:

```go
func TestProductDAOListAdminScopedMerchantOnlyOwnProducts(t *testing.T) {
	db := setupProductTestDB(t)
	dao := NewProductDAO(db)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	require.NoError(t, dao.Create(ctx, &model.Product{Name: "A", OwnerID: &ownerA, Status: model.ProductStatusDraft}))
	require.NoError(t, dao.Create(ctx, &model.Product{Name: "B", OwnerID: &ownerB, Status: model.ProductStatusDraft}))

	items, total, err := dao.ListAdminScoped(ctx, &ownerA, nil, 1, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/product
go test ./dao -run TestProductDAOListAdminScopedMerchantOnlyOwnProducts -count=1
```

Expected: compile failure because `OwnerID` and `ListAdminScoped` do not exist.

- [ ] **Step 3: Add migration files and model field**

Use SQL from section `2.1` and `2.2`. Add `OwnerID *int64` to `model.Product`.

- [ ] **Step 4: Implement DAO scoped methods**

```go
func (d *ProductDAO) ListAdminScoped(ctx context.Context, ownerID *int64, status *model.ProductStatus, page, pageSize int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64
	query := d.db.WithContext(ctx).Model(&model.Product{})
	if ownerID != nil {
		query = query.Where("owner_id = ?", *ownerID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (d *ProductDAO) GetByIDAndOwnerID(ctx context.Context, id, ownerID int64) (*model.Product, error) {
	var product model.Product
	err := d.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}
```

- [ ] **Step 5: Implement service methods**

Add owner-aware request and methods:

```go
func (s *ProductService) CreateProductForOwner(ctx context.Context, ownerID int64, req *CreateProductRequest) (*model.Product, error) {
	product := &model.Product{
		OwnerID:     &ownerID,
		Name:        req.Name,
		Description: req.Description,
		Images:      req.Images,
		Status:      model.ProductStatusDraft,
	}
	if err := s.productDAO.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) ListAdminProducts(ctx context.Context, actorRole string, actorUserID int64, status *model.ProductStatus, page, pageSize int) ([]model.Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var ownerID *int64
	if actorRole == "merchant" {
		ownerID = &actorUserID
	}
	return s.productDAO.ListAdminScoped(ctx, ownerID, status, page, pageSize)
}
```

- [ ] **Step 6: Implement admin product handlers**

Add methods to `ProductHandler`: `AdminList`, `AdminGet`, `AdminCreate`, `AdminUpdate`, `AdminDelete`. `AdminCreate`, `AdminUpdate`, and `AdminDelete` must call `requireMerchantActor`.

Response format:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [],
    "total": 0,
    "page": 1,
    "page_size": 20
  }
}
```

- [ ] **Step 7: Register Gateway and Product routes**

Gateway `backend/gateway/router/router.go`:

```go
authGroup.GET("/admin/products", middleware.RequireMerchantOrAdmin(), adminProductProxy.Forward)
authGroup.GET("/admin/products/:id", middleware.RequireMerchantOrAdmin(), adminProductProxy.Forward)
authGroup.POST("/admin/products", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.PUT("/admin/products/:id", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.DELETE("/admin/products/:id", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
```

Product `backend/product/main.go`:

```go
v1.GET("/admin/products", internalAuth, productHandler.AdminList)
v1.GET("/admin/products/:id", internalAuth, productHandler.AdminGet)
v1.POST("/admin/products", internalAuth, productHandler.AdminCreate)
v1.PUT("/admin/products/:id", internalAuth, productHandler.AdminUpdate)
v1.DELETE("/admin/products/:id", internalAuth, productHandler.AdminDelete)
```

- [ ] **Step 8: Run tests**

```bash
cd backend/product
go test ./dao ./service ./handler -run 'Product.*Admin|ProductDAOListAdminScoped|CreateProductForOwner' -count=1
cd ../gateway
go test ./middleware -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add backend/migrations/2026060401_admin_role_scope.up.sql backend/migrations/2026060401_admin_role_scope.down.sql backend/product/model/product.go backend/product/dao/product.go backend/product/service/product.go backend/product/handler/product.go backend/product/main.go backend/gateway/router/router.go backend/product/dao/product_test.go backend/product/service/product_test.go backend/product/handler/product_test.go
git commit -m "feat: add owner scoped admin product APIs"
```

---

## 6. Task 4: Merchant Auction Rule Templates

**Files:**
- Create: `backend/product/model/auction_rule_template.go`
- Create: `backend/product/dao/auction_rule_template.go`
- Create: `backend/product/service/auction_rule_template.go`
- Create: `backend/product/handler/auction_rule_template.go`
- Modify: `backend/product/main.go`
- Modify: `backend/gateway/router/router.go`
- Tests: matching DAO/service/handler tests.

- [ ] **Step 1: Write failing service test**

```go
func TestAuctionRuleTemplateServiceRejectsFloatLikeInvalidDecimal(t *testing.T) {
	svc := NewAuctionRuleTemplateService(fakeTemplateDAO{})
	_, err := svc.Create(context.Background(), 1001, CreateAuctionRuleTemplateRequest{
		Name:       "默认模板",
		StartPrice: "10.001",
		Increment:  "1.00",
		Duration:   60,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "金额最多支持两位小数")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/product
go test ./service -run TestAuctionRuleTemplateServiceRejectsFloatLikeInvalidDecimal -count=1
```

Expected: compile failure because service does not exist.

- [ ] **Step 3: Implement model and DAO**

Use model from section `2.3`. DAO must include:

```go
func (d *AuctionRuleTemplateDAO) ListByOwner(ctx context.Context, ownerID int64, page, pageSize int) ([]model.AuctionRuleTemplate, int64, error)
func (d *AuctionRuleTemplateDAO) GetByIDAndOwner(ctx context.Context, id, ownerID int64) (*model.AuctionRuleTemplate, error)
func (d *AuctionRuleTemplateDAO) Create(ctx context.Context, item *model.AuctionRuleTemplate) error
func (d *AuctionRuleTemplateDAO) Update(ctx context.Context, item *model.AuctionRuleTemplate) error
func (d *AuctionRuleTemplateDAO) DeleteByIDAndOwner(ctx context.Context, id, ownerID int64) error
```

- [ ] **Step 4: Implement service with decimal strings**

Request and response:

```go
type CreateAuctionRuleTemplateRequest struct {
	Name               string `json:"name"`
	StartPrice         string `json:"start_price"`
	Increment          string `json:"increment"`
	CapPrice           string `json:"cap_price,omitempty"`
	Duration           int    `json:"duration"`
	DelayDuration      int    `json:"delay_duration,omitempty"`
	MaxDelayTime        int    `json:"max_delay_time,omitempty"`
	TriggerDelayBefore int    `json:"trigger_delay_before,omitempty"`
	IsDefault           bool   `json:"is_default"`
}

type AuctionRuleTemplateResponse struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	StartPrice         string `json:"start_price"`
	Increment          string `json:"increment"`
	CapPrice           string `json:"cap_price,omitempty"`
	Duration           int    `json:"duration"`
	DelayDuration      int    `json:"delay_duration"`
	MaxDelayTime        int    `json:"max_delay_time"`
	TriggerDelayBefore int    `json:"trigger_delay_before"`
	IsDefault           bool   `json:"is_default"`
}
```

Decimal validation rule:

```go
func parseMoney2(raw string, field string) (decimal.Decimal, error) {
	v, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%s 金额格式错误", field)
	}
	if !v.Equal(v.Round(2)) {
		return decimal.Zero, fmt.Errorf("%s 金额最多支持两位小数", field)
	}
	if v.IsNegative() {
		return decimal.Zero, fmt.Errorf("%s 金额不能为负数", field)
	}
	return v, nil
}
```

- [ ] **Step 5: Implement merchant-only handlers**

Endpoints:

```http
GET    /api/v1/admin/auction-rule-templates?page=1&page_size=20
GET    /api/v1/admin/auction-rule-templates/:id
POST   /api/v1/admin/auction-rule-templates
PUT    /api/v1/admin/auction-rule-templates/:id
DELETE /api/v1/admin/auction-rule-templates/:id
```

All handlers must call `requireMerchantActor`; admin receives `403`.

- [ ] **Step 6: Register routes**

Gateway:

```go
authGroup.GET("/admin/auction-rule-templates", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.GET("/admin/auction-rule-templates/:id", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.POST("/admin/auction-rule-templates", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.PUT("/admin/auction-rule-templates/:id", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.DELETE("/admin/auction-rule-templates/:id", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
```

Product:

```go
v1.GET("/admin/auction-rule-templates", internalAuth, auctionRuleTemplateHandler.List)
v1.GET("/admin/auction-rule-templates/:id", internalAuth, auctionRuleTemplateHandler.Get)
v1.POST("/admin/auction-rule-templates", internalAuth, auctionRuleTemplateHandler.Create)
v1.PUT("/admin/auction-rule-templates/:id", internalAuth, auctionRuleTemplateHandler.Update)
v1.DELETE("/admin/auction-rule-templates/:id", internalAuth, auctionRuleTemplateHandler.Delete)
```

- [ ] **Step 7: Run tests**

```bash
cd backend/product
go test ./dao ./service ./handler -run 'AuctionRuleTemplate|RuleTemplate' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/product/model/auction_rule_template.go backend/product/dao/auction_rule_template.go backend/product/service/auction_rule_template.go backend/product/handler/auction_rule_template.go backend/product/main.go backend/gateway/router/router.go backend/product/**/*auction_rule_template*_test.go
git commit -m "feat: add merchant auction rule templates"
```

---

## 7. Task 5: Role-Aware Live Stream Management

**Files:**
- Modify: `backend/product/handler/live_stream.go`
- Modify: `backend/product/service/live_stream.go`
- Modify: `backend/product/dao/live_stream.go`
- Modify: `backend/product/main.go`
- Modify: `backend/gateway/router/router.go`
- Tests: `backend/product/handler/live_stream_test.go`, `backend/product/dao/live_stream_test.go`

- [ ] **Step 1: Write failing tests**

Test cases:

```go
func TestLiveStreamDAOListAdminScopedMerchantOnlyOwnStreams(t *testing.T) {
	db := setupLiveStreamTestDB(t)
	dao := NewLiveStreamDAO(db)
	ctx := context.Background()
	require.NoError(t, dao.Create(ctx, &model.LiveStream{CreatorID: 1001, Name: "A", Status: model.LiveStreamStatusLive}))
	require.NoError(t, dao.Create(ctx, &model.LiveStream{CreatorID: 1002, Name: "B", Status: model.LiveStreamStatusLive}))

	items, total, err := dao.ListAdminScoped(ctx, 0, 20, nil, ptrInt64(1001))

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "A", items[0].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/product
go test ./dao -run TestLiveStreamDAOListAdminScopedMerchantOnlyOwnStreams -count=1
```

Expected: compile failure because `ListAdminScoped` does not exist.

- [ ] **Step 3: Implement scoped DAO**

```go
func (d *LiveStreamDAO) ListAdminScoped(ctx context.Context, offset, limit int, statusFilter *int, creatorID *int64) ([]model.LiveStream, int64, error) {
	var liveStreams []model.LiveStream
	var total int64
	query := d.db.WithContext(ctx).Model(&model.LiveStream{})
	if creatorID != nil {
		query = query.Where("creator_id = ?", *creatorID)
	}
	if statusFilter != nil {
		query = query.Where("status = ?", *statusFilter)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&liveStreams).Error
	return liveStreams, total, err
}

func (d *LiveStreamDAO) GetByIDAndCreatorID(ctx context.Context, id, creatorID int64) (*model.LiveStream, error) {
	var liveStream model.LiveStream
	err := d.db.WithContext(ctx).Where("id = ? AND creator_id = ?", id, creatorID).First(&liveStream).Error
	if err != nil {
		return nil, err
	}
	return &liveStream, nil
}
```

- [ ] **Step 4: Implement role-aware handlers**

`ListAdmin` becomes role-aware:

- admin: no creator filter.
- merchant: `creator_id = actor.UserID`.

Add:

```http
GET  /api/v1/admin/live-streams/:id
POST /api/v1/admin/live-streams
PUT  /api/v1/admin/live-streams/:id
```

Create/update bodies:

```json
{
  "name": "直播间名称",
  "description": "直播间描述",
  "cover_image": "https://example.com/cover.png",
  "video_url": "https://example.com/live.m3u8",
  "streamer_name": "主播昵称",
  "streamer_avatar": "https://example.com/avatar.png"
}
```

- [ ] **Step 5: Keep governance actions admin-only**

`EndAdmin` and `BanAdmin` must call `requireAdminActor`, not `readAdminActor`.

- [ ] **Step 6: Register routes**

Gateway:

```go
authGroup.GET("/admin/live-streams", middleware.RequireMerchantOrAdmin(), adminProductProxy.Forward)
authGroup.GET("/admin/live-streams/:id", middleware.RequireMerchantOrAdmin(), adminProductProxy.Forward)
authGroup.POST("/admin/live-streams", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.PUT("/admin/live-streams/:id", middleware.RequireMerchantOnly(), adminProductProxy.Forward)
authGroup.PUT("/admin/live-streams/:id/end", middleware.RequireAdmin(), adminProductProxy.Forward)
authGroup.PUT("/admin/live-streams/:id/ban", middleware.RequireAdmin(), adminProductProxy.Forward)
```

- [ ] **Step 7: Run tests**

```bash
cd backend/product
go test ./dao ./service ./handler -run 'LiveStream.*Admin|ListAdminScoped|GetByIDAndCreatorID' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/product/handler/live_stream.go backend/product/service/live_stream.go backend/product/dao/live_stream.go backend/product/main.go backend/gateway/router/router.go backend/product/handler/live_stream_test.go
git commit -m "feat: add role aware live stream management"
```

---

## 8. Task 6: Seller-Scoped Orders

**Files:**
- Modify: `backend/product/model/order.go`
- Modify: `backend/product/dao/order.go`
- Modify: `backend/product/dao/order_admin.go`
- Modify: `backend/product/service/order.go`
- Modify: `backend/product/service/order_admin.go`
- Modify: `backend/product/handler/order.go`
- Modify: `backend/product/handler/order_admin.go`
- Modify: `backend/gateway/router/router.go`
- Tests: order DAO/service/handler tests.

- [ ] **Step 1: Write failing DAO test**

```go
func TestOrderDAOListAdminOrdersBySellerID(t *testing.T) {
	db := setupOrderTestDB(t)
	dao := NewOrderDAO(db)
	ctx := context.Background()
	sellerA := int64(1001)
	sellerB := int64(1002)
	require.NoError(t, dao.Create(ctx, &model.Order{AuctionID: 1, ProductID: 11, SellerID: &sellerA, WinnerID: 2001, FinalPrice: decimal.NewFromInt(10)}))
	require.NoError(t, dao.Create(ctx, &model.Order{AuctionID: 2, ProductID: 12, SellerID: &sellerB, WinnerID: 2002, FinalPrice: decimal.NewFromInt(20)}))

	items, total, err := dao.ListAdminBySeller(ctx, nil, &sellerA, 1, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, int64(11), items[0].ProductID)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/product
go test ./dao -run TestOrderDAOListAdminOrdersBySellerID -count=1
```

Expected: compile failure because `SellerID` and `ListAdminBySeller` do not exist.

- [ ] **Step 3: Add `SellerID` model and DAO filters**

`ListAdminOrders` must accept both `status` and `sellerID`.

```go
func (d *OrderDAO) ListAdminBySeller(ctx context.Context, status *model.OrderStatus, sellerID *int64, page, pageSize int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64
	query := d.db.WithContext(ctx).Model(&model.Order{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if sellerID != nil {
		query = query.Where("seller_id = ?", *sellerID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}
```

- [ ] **Step 4: Change Admin order handlers to role-aware**

`GET /admin/orders`:

- admin: `sellerID = nil`.
- merchant: `sellerID = actor.UserID`.

`GET /admin/orders/:id`:

- admin: get by id.
- merchant: get by id and seller id.

- [ ] **Step 5: Restrict shipping to merchant seller only**

Gateway route:

```go
authGroup.PUT("/orders/:id/ship", middleware.RequireMerchantOnly(), productProxy.Forward)
```

Product `Ship` handler must:

- require `X-User-Role=merchant`.
- get `X-User-ID`.
- update only if `orders.seller_id = X-User-ID`.

- [ ] **Step 6: Ensure new orders set `seller_id`**

Where product-service creates orders, load product by `ProductID` and copy `product.OwnerID` to `Order.SellerID`. If `product.OwnerID` is nil, fail with a clear error for new order creation:

```go
if product.OwnerID == nil || *product.OwnerID <= 0 {
	return nil, errors.New("商品缺少商家归属，无法创建订单")
}
order.SellerID = product.OwnerID
```

- [ ] **Step 7: Run tests**

```bash
cd backend/product
go test ./dao ./service ./handler -run 'Order.*Admin|Seller|Ship' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/product/model/order.go backend/product/dao/order.go backend/product/dao/order_admin.go backend/product/service/order.go backend/product/service/order_admin.go backend/product/handler/order.go backend/product/handler/order_admin.go backend/gateway/router/router.go backend/product/**/*order*_test.go
git commit -m "feat: add seller scoped admin orders"
```

---

## 9. Task 7: Role-Aware Statistics

**Files:**
- Modify: `backend/gateway/router/router.go`
- Modify: `backend/product/handler/statistics.go`
- Modify: `backend/product/service/statistics.go`
- Modify: `backend/product/dao/statistics.go`
- Tests: `backend/product/handler/statistics_test.go`, `backend/product/service/statistics_test.go`

- [ ] **Step 1: Write failing handler tests**

Required cases:

```go
func TestStatisticsUsersRejectsMerchant(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	handler := NewStatisticsHandler(fakeStatisticsService{})
	handler.GetUserStatistics(context.Background(), c)

	require.Equal(t, 403, c.Response.StatusCode())
}

func TestStatisticsRevenueAllowsMerchant(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	handler := NewStatisticsHandler(fakeStatisticsService{})
	handler.GetRevenueStatistics(context.Background(), c)

	require.NotEqual(t, 403, c.Response.StatusCode())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/product
go test ./handler -run 'TestStatisticsUsersRejectsMerchant|TestStatisticsRevenueAllowsMerchant' -count=1
```

Expected: revenue still rejects merchant because `requireAdminRole` is used.

- [ ] **Step 3: Update Gateway routes**

```go
authGroup.GET("/statistics/overview", middleware.RequireMerchantOrAdmin(), productProxy.Forward)
authGroup.GET("/statistics/auctions", middleware.RequireMerchantOrAdmin(), productProxy.Forward)
authGroup.GET("/statistics/revenue", middleware.RequireMerchantOrAdmin(), productProxy.Forward)
authGroup.GET("/statistics/users", middleware.RequireAdmin(), productProxy.Forward)
```

- [ ] **Step 4: Update handler/service/DAO signatures**

Use actor scope:

```go
type StatisticsScope struct {
	Role   string
	UserID int64
}

func (s StatisticsScope) SellerID() *int64 {
	if s.Role == "merchant" {
		return &s.UserID
	}
	return nil
}
```

Methods:

```go
func (s *StatisticsService) GetOverview(ctx context.Context, scope dao.StatisticsScope) (*dao.OverviewStatistics, error)
func (s *StatisticsService) GetAuctionStatistics(ctx context.Context, scope dao.StatisticsScope, startDate, endDate *time.Time) (*dao.AuctionStatistics, error)
func (s *StatisticsService) GetRevenueStatistics(ctx context.Context, scope dao.StatisticsScope, startDate, endDate *time.Time, category string) (*dao.RevenueStatistics, error)
```

DAO queries must filter orders by `seller_id` for merchant scope.

- [ ] **Step 5: Keep user statistics admin-only**

`GetUserStatistics` must call `requireAdminActor`. Merchant must receive `403`.

- [ ] **Step 6: Run tests**

```bash
cd backend/product
go test ./dao ./service ./handler -run 'Statistics|Revenue|Overview|Users' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/gateway/router/router.go backend/product/handler/statistics.go backend/product/service/statistics.go backend/product/dao/statistics.go backend/product/handler/statistics_test.go backend/product/service/statistics_test.go
git commit -m "feat: add role scoped statistics"
```

---

## 10. Task 8: Auction Service Admin Frontend Scope

**Files:**
- Modify: `backend/gateway/router/router.go`
- Modify: `backend/auction/handler/auction.go`
- Modify: `backend/auction/service/auction.go`
- Modify: `backend/auction/dao/auction.go`
- Tests: `backend/auction/handler/auction_admin_scope_test.go`, `backend/auction/service/auction_scope_test.go`

- [ ] **Step 1: Write failing DAO test**

```go
func TestAuctionDAOListWithFiltersByCreatorID(t *testing.T) {
	db := setupAuctionTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()
	creatorA := int64(1001)
	creatorB := int64(1002)
	require.NoError(t, dao.Create(ctx, &model.Auction{ProductID: 1, CreatorID: &creatorA, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}))
	require.NoError(t, dao.Create(ctx, &model.Auction{ProductID: 2, CreatorID: &creatorB, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}))

	items, total, err := dao.ListWithFilters(ctx, &AuctionFilters{CreatorID: &creatorA}, 1, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, int64(1), items[0].ProductID)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/auction
go test ./dao -run TestAuctionDAOListWithFiltersByCreatorID -count=1
```

Expected: compile failure because `AuctionFilters.CreatorID` does not exist.

- [ ] **Step 3: Add creator filters**

Add `CreatorID *int64` to `dao.AuctionFilters` and apply:

```go
if filters.CreatorID != nil {
	query = query.Where("creator_id = ?", *filters.CreatorID)
}
```

- [ ] **Step 4: Add admin frontend auction endpoints**

Gateway:

```go
authGroup.GET("/admin/auctions", middleware.RequireMerchantOrAdmin(), auctionProxy.Forward)
authGroup.GET("/admin/auctions/:id", middleware.RequireMerchantOrAdmin(), auctionProxy.Forward)
authGroup.POST("/admin/auctions", middleware.RequireMerchantOnly(), auctionProxy.Forward)
authGroup.PUT("/admin/auctions/:id/cancel", middleware.RequireMerchantOnly(), auctionProxy.Forward)
```

Auction service routes should map these paths to handlers that:

- read `X-User-ID` and `X-User-Role`.
- admin list/detail uses no creator filter.
- merchant list/detail uses `creator_id = X-User-ID`.
- create sets `CreatorID = X-User-ID`.
- cancel updates only if `creator_id = X-User-ID`.

- [ ] **Step 5: Run tests**

```bash
cd backend/auction
go test ./dao ./service ./handler -run 'Auction.*Admin|CreatorID|Cancel.*Owner' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/gateway/router/router.go backend/auction/handler/auction.go backend/auction/service/auction.go backend/auction/dao/auction.go backend/auction/handler/auction_admin_scope_test.go backend/auction/service/auction_scope_test.go
git commit -m "feat: add role scoped admin auctions"
```

---

## 11. Task 9: Fixed-Price Merchant-Only Write Enforcement

**Files:**
- Modify: `backend/gateway/router/router.go`
- Modify: `backend/auction/handler/fixed_price.go`
- Tests: `backend/auction/handler/fixed_price_test.go`

- [ ] **Step 1: Write failing handler test**

```go
func TestFixedPriceListRejectsAdminRole(t *testing.T) {
	c := app.NewContext(0)
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.SetBodyString(`{"live_stream_id":1,"product_id":1,"price":"10.00","total_stock":1,"max_per_user":1}`)

	handler := NewFixedPriceHandler(fakeFixedPriceUsecase{}, nil)
	handler.List(context.Background(), c)

	require.Equal(t, 403, c.Response.StatusCode())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/auction
go test ./handler -run TestFixedPriceListRejectsAdminRole -count=1
```

Expected: currently admin may pass Gateway and handler does not reject admin by role header.

- [ ] **Step 3: Update Gateway routes**

```go
authGroup.POST("/fixed-price/items", middleware.RequireMerchantOnly(), auctionProxy.Forward)
authGroup.POST("/fixed-price/items/:id/offline", middleware.RequireMerchantOnly(), auctionProxy.Forward)
authGroup.GET("/admin/live-streams/:id/fixed-price/items", middleware.RequireMerchantOnly(), auctionProxy.Forward)
```

- [ ] **Step 4: Add downstream defensive merchant check**

In `backend/auction/handler/fixed_price.go`:

```go
func requireFPMerchant(c *app.RequestContext) (int64, bool) {
	userID, ok := requireFPUser(c)
	if !ok {
		return 0, false
	}
	if string(c.GetHeader("X-User-Role")) != "merchant" {
		writeFPErr(c, 403, "FP_FORBIDDEN", "平台管理员不具备一口价代运营权限", nil)
		return 0, false
	}
	return userID, true
}
```

Use `requireFPMerchant` in `List`, `Offline`, and admin all-status list handler.

- [ ] **Step 5: Run tests**

```bash
cd backend/auction
go test ./handler -run 'FixedPrice.*Admin|FixedPrice.*Merchant|TestFixedPriceListRejectsAdminRole' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/gateway/router/router.go backend/auction/handler/fixed_price.go backend/auction/handler/fixed_price_test.go
git commit -m "feat: enforce merchant only fixed price writes"
```

---

## 12. Task 10: Integration Verification And API Smoke Tests

**Files:**
- Create: `backend/test/scenarios/admin_role_visibility_test.go` if the test-service scenario style supports HTTP integration tests.
- Otherwise add focused route tests in Gateway/Product/Auction services.

- [ ] **Step 1: Add smoke test matrix**

Test matrix:

| Actor | Endpoint | Expected |
|---|---|---:|
| admin | `GET /api/v1/admin/products` | `200` |
| admin | `POST /api/v1/admin/products` | `403` |
| merchant | `POST /api/v1/admin/products` | `201` |
| merchant A | `GET /api/v1/admin/products/:merchantBProductID` | `404` or `403` |
| merchant | `GET /api/v1/admin/auction-rule-templates` | `200` |
| admin | `GET /api/v1/admin/auction-rule-templates` | `403` |
| merchant | `GET /api/v1/statistics/users` | `403` |
| admin | `GET /api/v1/statistics/users` | `200` |
| admin | `POST /api/v1/fixed-price/items` | `403` |
| merchant | `POST /api/v1/fixed-price/items` | `200` |

- [ ] **Step 2: Run backend unit tests**

```bash
cd backend/gateway
go test ./... -count=1
cd ../product
go test ./... -count=1
cd ../auction
go test ./... -count=1
```

Expected: all PASS.

- [ ] **Step 3: Run local migration against dev DB**

Use the repository's established local DB process. If there is no migration runner, apply:

```bash
mysql -h localhost -P 3306 -u root -p live_auction < backend/migrations/2026060401_admin_role_scope.up.sql
```

Expected:

```text
Query OK
```

- [ ] **Step 4: Manual curl smoke**

Use real tokens from local login:

```bash
ADMIN_TOKEN="<admin jwt>"
MERCHANT_TOKEN="<merchant jwt>"

curl -i -H "Authorization: Bearer ${ADMIN_TOKEN}" http://localhost:8080/api/v1/admin/products
curl -i -X POST -H "Authorization: Bearer ${ADMIN_TOKEN}" -H "Content-Type: application/json" -d '{"name":"禁止代运营"}' http://localhost:8080/api/v1/admin/products
curl -i -H "Authorization: Bearer ${MERCHANT_TOKEN}" http://localhost:8080/api/v1/statistics/users
```

Expected:

```text
admin product list: HTTP/1.1 200
admin create product: HTTP/1.1 403
merchant user statistics: HTTP/1.1 403
```

- [ ] **Step 5: Commit**

```bash
git add backend/test/scenarios/admin_role_visibility_test.go
git commit -m "test: add admin role visibility smoke coverage"
```

---

## 13. Rollout Notes

- Existing rows with `products.owner_id IS NULL` remain visible only to platform admin until a deliberate backfill assigns a merchant owner.
- Existing rows with `orders.seller_id IS NULL` remain visible only to platform admin until backfilled from products with valid `owner_id`.
- New product creation must always write `owner_id`.
- New order creation must always write `seller_id`.
- Do not change public H5 product, live stream, auction, or fixed-price read APIs as part of this backend plan.
- Do not change Gateway `localhost` configuration for local debugging.

---

## 14. Self-Review Checklist

- Spec coverage: role boundaries, merchant `/auction/rules`, admin no代运营, frontend-visible page permissions, and backend data scope are covered by tasks 1-10.
- Database coverage: `products.owner_id`, `orders.seller_id`, and `auction_rule_templates` are explicitly defined with up/down SQL.
- Interface coverage: every Admin frontend backend route needed by the design has method, path, role, and scope.
- Security coverage: Gateway middleware and downstream service checks both enforce role semantics.
- Financial precision coverage: new auction rule template money fields use `shopspring/decimal` and string API values.
- Residual risk: existing legacy rows without ownership require a business-approved backfill before merchants can see them.

