# Quickstart: MVP阶段功能完善

**Feature**: `20260522-mvp-notification-report`
**Date**: 2026-05-22

## Prerequisites

1. Go 1.21+ installed
2. Node.js 18+ installed
3. MySQL 8.0 running
4. Redis 7 running
5. Docker (optional, for containerized deployment)

## Development Setup

### 1. Install Swagger CLI

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. Install Frontend Dependencies

```bash
cd frontend/h5 && npm install
cd frontend/admin && npm install
```

### 3. Run Database Migration

```bash
# Connect to MySQL and run:
source specs/20260522-mvp-notification-report/migrations/001_create_notifications.sql
```

## Running the Application

### Backend Services

```bash
# Terminal 1: Auction Service
cd backend/auction && go run .

# Terminal 2: Product Service
cd backend/product && go run .

# Terminal 3: Gateway Service
cd backend/gateway && go run .
```

### Frontend Apps

```bash
# Terminal 4: H5 User App
cd frontend/h5 && npm run dev

# Terminal 5: Admin Dashboard
cd frontend/admin && npm run dev
```

## Integration Scenarios

### Scenario 1: User Receives Bid Outbid Notification

```
1. User A places bid 100 yuan on Auction #1
2. User B places bid 120 yuan on Auction #1
3. System detects User A's bid is outbid
4. NotificationService.SendNotification() called:
   - UserID: User A's ID
   - Type: "bid_outbid"
   - Title: "出价被超越"
   - Content: "您在竞拍「商品名称」中的出价已被超越"
   - Data: {"auction_id": 1, "old_bid": 100, "new_bid": 120}
5. Notification saved to database
6. WebSocket message pushed to User A (if connected)
7. User A sees notification in real-time
```

### Scenario 2: Auction End Notifications

```
1. Auction #1 ends (time expires or manually ended)
2. Winner determined: User B (highest bid 120)
3. NotificationService.SendBatchNotifications() called:
   - Winner notification (auction_won)
   - Loser notifications (auction_lost) for User A, C, D...
4. All notifications saved and pushed via WebSocket
```

### Scenario 3: Order Status Change Notification (Mock)

```
1. User calls Mock PayOrder API
2. Order status changes to "paid"
3. NotificationService.SendNotification() called:
   - Type: "order_paid"
   - Content: "您的订单 #123 已支付成功"
4. ⚠️ Note: This is Mock-triggered. Phase 2 will use OrderEventPublisher.
```

### Scenario 4: Admin Views Statistics Dashboard

```
1. Admin logs in to Admin Dashboard
2. Navigate to Dashboard page
3. Frontend calls GET /api/v1/statistics/overview
4. Backend checks Redis cache
   - Cache hit: Return cached data
   - Cache miss: Query database, cache result (5 min TTL)
5. Dashboard displays: Total auctions, Revenue, Active users
6. Charts render using Recharts
```

### Scenario 5: Swagger Documentation

```
1. Developer adds new API endpoint
2. Developer adds Swagger annotations:
   // @Summary Get notifications
   // @Description Get user's notification list
   // @Tags notification
   // @Security BearerAuth
   // @Param page query int false "Page number"
   // @Success 200 {object} NotificationListResponse
   // @Router /notifications [get]
3. Developer runs: swag init -g gateway/main.go -o ./docs
4. Swagger docs regenerated
5. Access http://localhost:8080/swagger/index.html to view docs
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./backend/... -v

# Run with coverage
go test ./backend/auction/service/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### E2E Tests

```bash
# Run Playwright tests
cd frontend/h5 && npx playwright test
```

### Test Notification Flow

```bash
# 1. Start services
# 2. Connect WebSocket client
# 3. Place bid as User A
# 4. Place higher bid as User B
# 5. Verify User A receives notification via WebSocket
```

## API Endpoints Summary

### Notification APIs

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/notifications | Get user notifications |
| GET | /api/v1/notifications/unread-count | Get unread count |
| PUT | /api/v1/notifications/:id/read | Mark as read |
| PUT | /api/v1/notifications/read-all | Mark all as read |

### Statistics APIs (Admin)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/statistics/overview | Dashboard overview |
| GET | /api/v1/statistics/auctions | Auction statistics |
| GET | /api/v1/statistics/revenue | Revenue statistics |
| GET | /api/v1/statistics/users | User statistics |

### WebSocket Messages

```typescript
// Notification message format
{
  "type": "notification",
  "data": {
    "id": 123,
    "type": "bid_outbid",
    "title": "出价被超越",
    "content": "您的出价已被超越",
    "data": { "auction_id": 1 },
    "created_at": "2026-05-22T10:00:00Z"
  }
}
```

## Troubleshooting

### WebSocket Not Receiving Notifications

1. Check JWT token is valid
2. Verify WebSocket connection is established (check browser devtools)
3. Check user_id in WebSocket connection matches notification recipient

### Statistics API Slow

1. Check Redis is running: `redis-cli ping`
2. Verify database indexes exist
3. Check cache TTL configuration

### Swagger Not Loading

1. Run `swag init` to regenerate docs
2. Check gateway router includes swagger routes
3. Verify `/swagger/*` path is not blocked by auth middleware
