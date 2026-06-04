# Product Reminder Hot Pull Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert subscribed products whose auctions start within 30 minutes into idempotent `auction_starting` unread notifications during hot-pull, without adding a second popup.

**Architecture:** Keep `user_product_reminders` as the subscription SSOT and add a receipt table for `user_id + auction_id` idempotency. Extend notification hot-pull to persist product auction reminders, then let existing unread-count and notification-list APIs drive the bell badge. `LiveReminderModal` remains dedicated to live-room start reminders.

**Tech Stack:** Go, GORM, MySQL, Hertz, React, TypeScript, Jest.

---

### Task 1: Backend Idempotent Hot Pull

**Files:**
- Create: `backend/auction/model/product_reminder_receipt.go`
- Modify: `backend/auction/dao/user_product_reminder.go`
- Modify: `backend/auction/dao/notification.go`
- Modify: `backend/auction/service/notification.go`
- Modify: `backend/auction/main.go`
- Test: `backend/auction/service/notification_test.go`

- [ ] **Step 1: Write failing tests**

Add tests proving `HotPullNotifications` creates one unread `auction_starting` notification for an enabled product reminder starting within 30 minutes, and creates no duplicate on a second hot-pull.

- [ ] **Step 2: Run tests and verify red**

Run:

```bash
cd backend/auction
go test ./service -run TestNotificationServiceHotPullProductReminder -count=1
```

Expected: fails because product reminder hot-pull persistence does not exist.

- [ ] **Step 3: Implement minimal backend support**

Add `ProductReminderReceipt` with unique key `user_id + auction_id`, DAO methods to find starting reminders and claim receipts, and call this path from `HotPullNotifications` before returning.

- [ ] **Step 4: Run tests and verify green**

Run:

```bash
cd backend/auction
go test ./service -run TestNotificationServiceHotPullProductReminder -count=1
```

Expected: pass.

### Task 2: Frontend Hot Pull Then Badge Refresh

**Files:**
- Modify: `frontend/h5/src/pages/Home/index.tsx`
- Test: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx`

- [ ] **Step 1: Write failing frontend test**

Add a test proving authenticated Home mount calls `notificationApi.hotPull()` before or together with unread-count refresh, then renders the badge when unread count becomes positive.

- [ ] **Step 2: Run test and verify red**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/pages/Home/__tests__/Home.test.tsx --runInBand
```

Expected: fails because Home currently only calls `getUnreadCount`.

- [ ] **Step 3: Implement hot-pull refresh**

In Home authenticated refresh flow, call `notificationApi.hotPull()` then `notificationApi.getUnreadCount()`. On hot-pull failure, still refresh unread count.

- [ ] **Step 4: Run test and verify green**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/pages/Home/__tests__/Home.test.tsx --runInBand
```

Expected: pass.

### Task 3: Final Verification

**Files:**
- No additional files.

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd backend/auction
go test ./service ./dao -count=1
```

Expected: pass.

- [ ] **Step 2: Run focused frontend tests**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/pages/Home/__tests__/Home.test.tsx src/services/__tests__/notification.test.ts --runInBand
```

Expected: pass.

- [ ] **Step 3: Manual local verification**

Use an authenticated user with a product reminder whose auction starts within 30 minutes:

```bash
curl -X POST http://localhost:8080/api/v1/notifications/hot-pull -H "Authorization: Bearer $TOKEN"
curl http://localhost:8080/api/v1/notifications/unread-count -H "Authorization: Bearer $TOKEN"
```

Expected: unread count increases once; repeated hot-pull does not duplicate `auction_starting`.
