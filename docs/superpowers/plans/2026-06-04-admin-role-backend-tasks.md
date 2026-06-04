# Admin Role Backend Tasks

Source plan: `docs/superpowers/plans/2026-06-04-admin-role-backend-implementation-plan.md`

## Task Checklist

- [ ] T001: Gateway exact role middleware
- [ ] T002: Product service auth context helpers
- [ ] T003: Product ownership schema and scoped product APIs
- [ ] T004: Merchant auction rule templates
- [ ] T005: Role-aware live stream management
- [ ] T006: Seller-scoped orders
- [ ] T007: Role-aware statistics
- [ ] T008: Auction service admin frontend scope
- [ ] T009: Fixed-price merchant-only write enforcement
- [ ] T010: Integration verification and API smoke tests

## Dependencies

| Task ID | Depends On | Reason |
| --- | --- | --- |
| T001 | - | Gateway exact-role middleware is the entry guard for all role-aware routes. |
| T002 | - | Product downstream role parsing is required before scoped Product handlers. |
| T003 | T001,T002 | Product owner scope requires Gateway role middleware and Product actor helpers. |
| T004 | T001,T002 | Merchant rule templates require merchant-only Gateway guard and Product actor helpers. |
| T005 | T001,T002 | Live stream management reuses role-aware Gateway routes and Product actor helpers. |
| T006 | T001,T002,T003 | Order seller scope depends on `products.owner_id` and `orders.seller_id`. |
| T007 | T001,T002,T006 | Merchant revenue/overview statistics depend on seller-scoped orders. |
| T008 | T001 | Auction service needs Gateway role-aware admin routes. |
| T009 | T001 | Fixed-price write enforcement depends on merchant-only Gateway guard. |
| T010 | T003,T004,T005,T006,T007,T008,T009 | Smoke tests verify the full backend contract. |

## Execution Waves

| Wave | Tasks | Parallelism |
| --- | --- | --- |
| W1 | T001,T002 | Can run in parallel; separate services/files. |
| W2 | T003 | Sequential; introduces shared schema and product ownership contract. |
| W3 | T004,T005,T008,T009 | Can run in parallel after W1/W2 where dependencies allow; avoid same-file route conflicts by assigning Gateway route edits carefully. |
| W4 | T006,T007 | Sequential; statistics depend on seller-scoped orders. |
| W5 | T010 | Final integration verification after all implementation tasks. |

## Required Verification

| Area | Command |
| --- | --- |
| Gateway | `cd backend/gateway && go test ./... -count=1` |
| Product | `cd backend/product && go test ./... -count=1` |
| Auction | `cd backend/auction && go test ./... -count=1` |
| SDD Script | `python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --plan docs/superpowers/plans/2026-06-04-admin-role-backend-implementation-plan.md --tasks docs/superpowers/plans/2026-06-04-admin-role-backend-tasks.md --topic admin-role-backend --force` |

