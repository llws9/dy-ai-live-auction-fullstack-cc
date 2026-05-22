# Research: MVP阶段功能完善

**Feature**: `20260522-mvp-notification-report`
**Date**: 2026-05-22

## Technology Decisions

### 1. Notification Delivery Strategy

**Decision**: WebSocket real-time push + Database persistence

**Rationale**:
- Existing WebSocket infrastructure can be reused
- Database persistence ensures no notification loss on disconnect
- Users can retrieve unread notifications on reconnect or page refresh

**Alternatives considered**:
- Server-Sent Events (SSE): Simpler but less efficient for bidirectional communication
- Polling: Higher latency and server load
- Pure WebSocket without persistence: Risk of notification loss

### 2. Statistics Data Aggregation

**Decision**: SQL aggregation queries with Redis caching

**Rationale**:
- MySQL can efficiently handle aggregation for MVP scale (<100k auctions)
- Redis caching for frequently accessed dashboard metrics
- No need for separate analytics pipeline at current scale

**Alternatives considered**:
- Pre-computed tables: Adds complexity, not needed at current scale
- ClickHouse/Doris: Overkill for MVP, can migrate later if needed
- Real-time stream processing: Unnecessary complexity

### 3. API Documentation Generation

**Decision**: swaggo/swag with inline annotations

**Rationale**:
- De facto standard for Go HTTP services
- Annotations stay close to implementation
- Automatic generation via `swag init`
- Works well with Hertz framework

**Alternatives considered**:
- Manual OpenAPI YAML: Maintenance burden, easily outdated
- Postman collections: Separate from code, drift risk
- grpc-gateway: Not applicable (using REST/Hertz)

### 4. Test Strategy

**Decision**: Layered testing approach

| Layer | Tool | Coverage Target |
|-------|------|-----------------|
| Unit | go test + testify | >80% core services |
| Integration | go test | DAO layer |
| E2E | Playwright | 5 core scenarios |

**Rationale**:
- Unit tests for business logic isolation
- Integration tests for database operations
- E2E tests for critical user flows

### 5. Interface Reservation Pattern

**Decision**: Define interfaces for future integration points

```go
// Current: Mock implementation triggers notifications
// Phase 2: Real order system implements OrderEventPublisher

type NotificationSender interface {
    SendNotification(ctx context.Context, req *NotificationRequest) error
    SendBatchNotifications(ctx context.Context, reqs []*NotificationRequest) error
}

type OrderEventPublisher interface {
    PublishOrderEvent(ctx context.Context, event *OrderEvent) error
}
```

**Rationale**:
- Allows current Mock implementation to work
- Clear contract for phase 2 integration
- No notification core code changes needed when real order system arrives

## Dependency Analysis

### Backend New Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/swaggo/swag | v1.16.x | Swagger doc generation |
| github.com/swaggo/hertz-swagger | v1.4.x | Hertz Swagger integration |

### Frontend New Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| recharts | ^2.10.x | Admin dashboard charts |
| @ant-design/charts | ^2.x | Alternative chart library |

## Performance Considerations

### Notification Throughput

- Expected: ~100 notifications/second during peak auction activity
- WebSocket broadcast: O(n) clients in auction room
- Batch notification API for auction end (multiple losers)

### Statistics Query Optimization

- Dashboard overview: Cache 5 minutes in Redis
- Date range queries: Add composite indexes on created_at columns
- Large dataset: Consider pagination for list APIs

## Security Considerations

### Notification Access Control

- Users can only fetch their own notifications (user_id filter)
- WebSocket authenticated via JWT token
- Notification data sanitized before WebSocket broadcast

### Statistics Access Control

- Admin-only endpoints (RoleAdmin check)
- Rate limiting on statistics APIs to prevent abuse

## Migration Path (Phase 2)

### Order System Integration

1. Real order service implements `OrderEventPublisher`
2. Configure event subscription to notification service
3. Remove Mock notification triggers in `order.go`
4. Notification core code unchanged

### Payment System Integration

1. Payment callback triggers `OrderEventPublisher.PublishOrderEvent(event)`
2. Event type: `OrderEventPaid`
3. Notification service handles and sends notification
