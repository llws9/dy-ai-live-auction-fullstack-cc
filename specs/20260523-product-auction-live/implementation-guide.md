# Complete Implementation Guide

**Feature**: 20260523-product-auction-live
**Generated**: 2026-05-23 07:00
**Status**: Infrastructure Complete, Ready for Feature Implementation

---

## 🎯 Implementation Checklist

### ✅ Completed (Infrastructure - 100%)

**Database**:
- [x] live_streams table created
- [x] user_live_stream_follows table created
- [x] auctions.live_stream_id field added
- [x] Migration executed on auction database

**RabbitMQ**:
- [x] RabbitMQ 4.3.1 installed and running
- [x] Connection management implemented (DLX + TTL)
- [x] Producer/Consumer pattern implemented
- [x] Delayed queue working without plugin

**Models**:
- [x] LiveStream model
- [x] UserLiveStreamFollow model
- [x] Auction model updated
- [x] Product model updated

**DAOs**:
- [x] LiveStreamDAO
- [x] ProductDAO (updated)

**Services**:
- [x] LiveStreamService
- [x] ProductService (Publish/Unpublish methods added)

**Middleware**:
- [x] Permission middleware created

---

## 📋 Remaining Implementation Tasks

### Phase 3: US1 - 商品发布到直播间 (70% Complete)

**Remaining Files**:

#### 1. Backend Handler
**File**: `backend/product/handler/product.go`

```go
// PublishHandler 发布商品API
func (h *ProductHandler) PublishHandler(ctx context.Context, c *app.RequestContext) {
    var req struct {
        StartTime *time.Time `json:"start_time"`
    }
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, map[string]interface{}{"code": 400, "message": "参数错误"})
        return
    }

    productID := c.Param("id")
    userRole := c.GetInt("user_role")
    userID := c.GetInt64("user_id")

    // 权限检查
    if userRole != 1 && userRole != 2 {
        c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
        return
    }

    product, liveStream, err := h.productService.PublishProduct(ctx, productID, userID, req.StartTime)
    if err != nil {
        c.JSON(500, map[string]interface{}{"code": 500, "message": err.Error()})
        return
    }

    c.JSON(200, map[string]interface{}{
        "code": 200,
        "message": "发布成功",
        "data": map[string]interface{}{
            "product": product,
            "live_stream": liveStream,
        },
    })
}
```

#### 2. Gateway Router
**File**: `backend/gateway/router/router.go`

Add routes:
```go
productGroup := v1.Group("/products")
productGroup.POST("/:id/publish", middleware.RequireMerchant(), productHandler.PublishHandler)
productGroup.POST("/:id/unpublish", middleware.RequireMerchant(), productHandler.UnpublishHandler)
```

#### 3. Frontend (Admin)
**File**: `frontend/admin/src/pages/Product/List.tsx`

Add publish button:
```typescript
const handlePublish = async (productId: number) => {
  try {
    await axios.post(`/api/v1/products/${productId}/publish`);
    message.success('发布成功');
    fetchProducts(); // 刷新列表
  } catch (error) {
    message.error('发布失败');
  }
};

// 在操作列添加按钮
{record.status === 0 && (
  <Button type="primary" onClick={() => handlePublish(record.id)}>
    发布
  </Button>
)}
```

---

### Phase 4: US2 - 商品下架功能 (60% Complete)

**Remaining Files**:

#### 1. Backend Handler
**File**: `backend/product/handler/product.go`

```go
// UnpublishHandler 下架商品API
func (h *ProductHandler) UnpublishHandler(ctx context.Context, c *app.RequestContext) {
    var req struct {
        Reason string `json:"reason"`
    }
    if err := c.BindJSON(&req); err != nil {
        req.Reason = ""
    }

    productID := c.Param("id")
    userID := c.GetInt64("user_id")
    userRole := c.GetInt("user_role")

    if userRole != 1 && userRole != 2 {
        c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
        return
    }

    product, err := h.productService.UnpublishProduct(ctx, productID, userID, req.Reason)
    if err != nil {
        c.JSON(500, map[string]interface{}{"code": 500, "message": err.Error()})
        return
    }

    // TODO: 通过RabbitMQ发送下架通知

    c.JSON(200, map[string]interface{}{
        "code": 200,
        "message": "下架成功",
        "data": product,
    })
}
```

#### 2. Frontend
Add unpublish button similar to publish button.

---

### Phase 5: US2.5 - 用户关注直播间功能 (0% Complete)

**Required Files**:

#### 1. DAO
**File**: `backend/auction/dao/user_live_stream_follow.go`

```go
package dao

import (
    "context"
    "auction-service/model"
    "gorm.io/gorm"
)

type UserLiveStreamFollowDAO struct {
    db *gorm.DB
}

func NewUserLiveStreamFollowDAO(db *gorm.DB) *UserLiveStreamFollowDAO {
    return &UserLiveStreamFollowDAO{db: db}
}

func (d *UserLiveStreamFollowDAO) Create(ctx context.Context, follow *model.UserLiveStreamFollow) error {
    return d.db.WithContext(ctx).Create(follow).Error
}

func (d *UserLiveStreamFollowDAO) Delete(ctx context.Context, userID, liveStreamID int64) error {
    return d.db.WithContext(ctx).
        Where("user_id = ? AND live_stream_id = ?", userID, liveStreamID).
        Delete(&model.UserLiveStreamFollow{}).Error
}

func (d *UserLiveStreamFollowDAO) GetFollowers(ctx context.Context, liveStreamID int64, offset, limit int) ([]model.UserLiveStreamFollow, error) {
    var follows []model.UserLiveStreamFollow
    err := d.db.WithContext(ctx).
        Where("live_stream_id = ? AND notification_enabled = ?", liveStreamID, true).
        Offset(offset).
        Limit(limit).
        Find(&follows).Error
    return follows, err
}

func (d *UserLiveStreamFollowDAO) CountByLiveStream(ctx context.Context, liveStreamID int64) (int64, error) {
    var count int64
    err := d.db.WithContext(ctx).
        Model(&model.UserLiveStreamFollow{}).
        Where("live_stream_id = ? AND notification_enabled = ?", liveStreamID, true).
        Count(&count).Error
    return count, err
}
```

#### 2. Service
**File**: `backend/auction/service/follow.go`

```go
package service

import (
    "context"
    "auction-service/dao"
    "auction-service/model"
)

type FollowService struct {
    followDAO *dao.UserLiveStreamFollowDAO
}

func NewFollowService(followDAO *dao.UserLiveStreamFollowDAO) *FollowService {
    return &FollowService{followDAO: followDAO}
}

func (s *FollowService) Follow(ctx context.Context, userID, liveStreamID int64) error {
    follow := &model.UserLiveStreamFollow{
        UserID:              userID,
        LiveStreamID:        liveStreamID,
        NotificationEnabled: true,
    }
    return s.followDAO.Create(ctx, follow)
}

func (s *FollowService) Unfollow(ctx context.Context, userID, liveStreamID int64) error {
    return s.followDAO.Delete(ctx, userID, liveStreamID)
}
```

#### 3. Notification Service
**File**: `backend/auction/service/notification.go`

```go
package service

import (
    "context"
    "auction-service/dao"
    "auction-service/mq"
    "time"
)

type NotificationService struct {
    followDAO *dao.UserLiveStreamFollowDAO
    notifyDAO *dao.NotificationDAO
    producer  *mq.NotificationProducer
}

const (
    BatchSize    = 10000
    BatchDelay   = 3 * time.Second
)

func (s *NotificationService) ProcessNotification(msg *mq.NotificationMessage) error {
    // 获取关注用户总数
    totalUsers, err := s.followDAO.CountByLiveStream(context.Background(), msg.LiveStreamID)
    if err != nil {
        return err
    }

    // 分批推送
    batches := (totalUsers + BatchSize - 1) / BatchSize

    for i := 0; i < int(batches); i++ {
        offset := i * BatchSize
        users, err := s.followDAO.GetFollowers(context.Background(), msg.LiveStreamID, offset, BatchSize)
        if err != nil {
            continue
        }

        // 批量创建通知记录
        notifications := make([]*model.Notification, 0, len(users))
        for _, user := range users {
            notifications = append(notifications, &model.Notification{
                UserID:  user.UserID,
                Type:    model.NotificationType(msg.Type),
                Title:   msg.GenerateTitle(),
                Content: msg.GenerateContent(),
                Data:    msg.GenerateData(),
            })
        }

        s.notifyDAO.BatchCreate(context.Background(), notifications)

        if i < int(batches)-1 {
            time.Sleep(BatchDelay)
        }
    }

    return nil
}
```

#### 4. Frontend (H5)
Create pages:
- `frontend/h5/src/pages/LiveStream/List.tsx` - 直播间列表
- `frontend/h5/src/pages/LiveStream/Detail.tsx` - 直播间详情（含关注按钮）
- `frontend/h5/src/pages/User/Follows.tsx` - 我的关注

---

### Phase 6: US4 - 竞拍管理状态筛选优化 (30% Complete)

**Remaining Work**:

#### 1. Backend - Modify AuctionDAO
Add methods to `backend/auction/dao/auction.go`:

```go
func (d *AuctionDAO) GetByStatus(ctx context.Context, status model.AuctionStatus, offset, limit int) ([]model.Auction, int64, error) {
    var auctions []model.Auction
    var total int64

    query := d.db.WithContext(ctx).Model(&model.Auction{})
    if status >= 0 {
        query = query.Where("status = ?", status)
    }

    query.Count(&total)
    err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&auctions).Error

    return auctions, total, err
}

func (d *AuctionDAO) GetByLiveStreamID(ctx context.Context, liveStreamID int64, offset, limit int) ([]model.Auction, int64, error) {
    var auctions []model.Auction
    var total int64

    query := d.db.WithContext(ctx).Model(&model.Auction{}).Where("live_stream_id = ?", liveStreamID)
    query.Count(&total)
    err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&auctions).Error

    return auctions, total, err
}
```

#### 2. Frontend - Modify List Component
Update `frontend/admin/src/pages/Auction/List.tsx`:

```typescript
const [filterStatus, setFilterStatus] = useState<string>('all');
const [searchLiveStream, setSearchLiveStream] = useState('');

const filterButtons = [
  { key: 'all', label: '全部' },
  { key: '0', label: '待开始' },
  { key: '1', label: '进行中' },
  { key: '3', label: '已结束' },
];

// 管理员额外显示直播间列
const columns = userRole === 2 ? [
  ...baseColumns,
  {
    title: '直播间ID',
    dataIndex: 'live_stream_id',
    key: 'live_stream_id',
  },
  {
    title: '直播间名称',
    dataIndex: 'live_stream_name',
    key: 'live_stream_name',
  },
] : baseColumns;
```

---

### Phase 7-9: US3, US5 (UI优化和管理模块 - 20% Complete)

**US3 - UI优化**:
- Remove inline styles from `frontend/admin/src/pages/Product/RuleConfig.tsx`
- Add form validation
- Use project CSS classes

**US5 - 直播间管理模块**:
- Create `frontend/admin/src/pages/LiveStream/List.tsx`
- Create `frontend/admin/src/pages/LiveStream/Detail.tsx`
- Add navigation menu item

---

## 🧪 Testing Strategy

### 1. API Testing (Manual)

**Use Postman or curl**:

```bash
# 登录获取token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'

# 发布商品
curl -X POST http://localhost:8080/api/v1/products/1/publish \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"start_time":"2026-05-23T15:00:00Z"}'

# 关注直播间
curl -X POST http://localhost:8080/api/v1/live-streams/1/follow \
  -H "Authorization: Bearer <token>"
```

### 2. Integration Testing

**Create test file**: `backend/product/service/product_test.go`

```go
package service

import (
    "context"
    "testing"
    "product-service/model"
    "github.com/stretchr/testify/assert"
)

func TestPublishProduct(t *testing.T) {
    // Setup test database
    // Create test product
    // Call PublishProduct
    // Verify status changed to Published
    // Verify live stream created/returned
}

func TestUnpublishProduct(t *testing.T) {
    // Setup test database
    // Create published product
    // Call UnpublishProduct
    // Verify status changed to Unpublished
}
```

---

## 📖 API Documentation Generation

### Install Swagger

```bash
cd backend/product
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g main.go -o ./docs
```

### Add Swagger Annotations

```go
// PublishHandler 发布商品到直播间
// @Summary 发布商品
// @Description 将草稿状态的商品发布到直播间，创建竞拍记录
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Param body body PublishRequest true "发布参数"
// @Success 200 {object} PublishResponse
// @Failure 400 {object} ErrorResponse
// @Router /products/{id}/publish [post]
func (h *ProductHandler) PublishHandler(ctx context.Context, c *app.RequestContext) {
    // ...
}
```

### Access Documentation

After generation, access at: `http://localhost:8080/swagger/index.html`

---

## 🚀 Final Steps

### 1. Start All Services

```bash
# Terminal 1: Start Product Service
cd backend/product
go run main.go

# Terminal 2: Start Auction Service
cd backend/auction
go run main.go

# Terminal 3: Start Gateway
cd backend/gateway
go run main.go

# Terminal 4: Start Frontend Admin
cd frontend/admin
npm run dev

# Terminal 5: Start Frontend H5
cd frontend/h5
npm run dev
```

### 2. Run Tests

```bash
# Backend tests
cd backend/product
go test ./...

cd backend/auction
go test ./...

# Frontend tests
cd frontend/admin
npm test
```

### 3. Verify All Features

- [ ] 商品发布功能正常
- [ ] 商品下架功能正常
- [ ] 用户可以关注直播间
- [ ] 管理员可以看到所有竞拍
- [ ] 商家只能看到自己的竞拍
- [ ] 竞拍筛选功能正常
- [ ] 直播间管理模块正常

### 4. Performance Testing

- Test with 100 concurrent users
- Verify notification delivery within 10 minutes
- Check database query performance
- Monitor RabbitMQ message throughput

---

## 📊 Progress Summary

**Total Tasks**: 74
**Completed**: 22 (Infrastructure + Models + Basic DAOs)
**Remaining**: 52 (User Story Implementation)

**Estimated Time**:
- Backend APIs: 15-20 hours
- Frontend Pages: 10-15 hours
- Testing: 5-8 hours
- Documentation: 2-3 hours
- **Total**: 32-46 hours

---

## 🎓 Key Learnings

1. **RabbitMQ Delayed Queue**: Standard DLX + TTL pattern is more reliable than plugin
2. **Database Naming**: Always verify database name before running migrations
3. **Permission Middleware**: Implement early to ensure security
4. **Batch Processing**: Essential for large-scale notifications

---

**Next Session**: Start with implementing US1 and US2 handlers, then proceed to US2.5 (follow feature) which is the largest remaining task.
