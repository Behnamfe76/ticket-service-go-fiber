# Implementation Tasks: Support Ticket Service

Version: 1.0 (2025-11-23)

---

## Phase 1 — Project & Infrastructure Setup

1. **Initialize Go Module & Skeleton**
   - Description: Create `go.mod`, basic folder tree, and placeholder files per layout.
   - Files/Packages: repo root, `cmd/api/main.go`, directories under `internal/`.
   - Dependencies: none.

2. **Config Loader**
   - Description: Implement `internal/config/config.go` to read env/.env with defaults (server port, DB/Redis, JWT secret, bcrypt cost, pagination, attachment limits).
   - Files: `internal/config/config.go`, `configs/config.example.yaml`, `.env.example`.
   - Dependencies: Task 1.

3. **Observability Bootstrap**
   - Description: Implement structured logging (`internal/observability/logging.go`, `metrics.go`, `tracing.go` stub) and request logging middleware in `internal/api/http/middleware.go`.
   - Dependencies: Task 1.

4. **Fiber App & Health Routes**
   - Description: Wire Fiber app in `cmd/api/main.go`, include config loading, logger setup, basic middleware, health handlers in `internal/api/http/handlers/health_handler.go`, and router definitions in `router.go`.
   - Dependencies: Tasks 1–3.

---

## Phase 2 — Database & Persistence

5. **Postgres Connection Pool**
   - Description: Implement `internal/persistence/postgres.go` using `pgxpool`, reading config, exposing `ConnectPostgres()` + health checks.
   - Dependencies: Task 2.

6. **Initial Migrations (Users/Staff/Org)**
   - Description: Create `/migrations/0001_init_schema.sql` covering `users`, `staff_members`, `departments`, `teams`, enums, constraints.
   - Dependencies: Task 5 (for schema decisions).

7. **Ticket & History Migrations**
   - Description: Add `/migrations/0002_tickets.sql` for `tickets`, `ticket_messages`, `ticket_history`, `attachment_references`, indexes.
   - Dependencies: Task 6.

8. **Auth Auxiliary Tables**
   - Description: Add `/migrations/0003_auth_tokens.sql` for `password_reset_tokens` and optional `sessions`.
   - Dependencies: Task 7.

9. **Migration Runner**
   - Description: Implement `internal/persistence/migrations.go` plus scripts `scripts/migrate_up.sh`, `migrate_down.sh`; integrate into `cmd/api/main.go` (config-flag controlled).
   - Dependencies: Tasks 5–8.

---

## Phase 3 — Redis & Caching

10. **Redis Client Setup**
    - Description: Implement `internal/persistence/redis.go` using go-redis, config-based connection, health check helper.
    - Dependencies: Tasks 2,4.

11. **Reference Cache**
    - Description: Implement `internal/cache/reference_cache.go` for departments/teams with get/set/invalidate functions and TTL config.
    - Dependencies: Tasks 5–8,10.

12. **Ticket Summary Cache**
    - Description: Implement `internal/cache/ticket_cache.go` storing ticket summary entries and invalidation helpers; optional stub if deferred.
    - Dependencies: Task 11 (shared patterns).

---

## Phase 4 — Domain Models & Repositories

13. **Domain Entities**
    - Description: Define structs and core methods in `internal/domain/*.go` (`user.go`, `staff_member.go`, `department.go`, `team.go`, `ticket.go`, `ticket_message.go`, `ticket_history.go`, `auth.go`).
    - Dependencies: Tasks 6–8.

14. **User Repository**
    - Description: Create `internal/repository/user_repository.go` with interface + Postgres implementation (CRUD, find by email, status updates).
    - Dependencies: Tasks 5–8,13.

15. **Staff Repository**
    - Description: Implement `internal/repository/staff_repository.go` with methods for CRUD, listing by department/team/role, activation toggles.
    - Dependencies: Task 14.

16. **Department & Team Repositories**
    - Description: Implement `department_repository.go` and `team_repository.go` with create/update/list, active filtering, uniqueness enforcement.
    - Dependencies: Task 14.

17. **Ticket Repository**
    - Description: Implement `ticket_repository.go` covering create, get by id, list for user, staff filters (status, priority, department, team, assignee, date range, text search); include transaction helpers.
    - Dependencies: Tasks 13–16.

18. **Message & History Repositories**
    - Description: Implement `ticket_message_repository.go`, `ticket_history_repository.go`, and `attachment_repository.go` for inserts, queries, and attachments.
    - Dependencies: Task 17.

---

## Phase 5 — Authentication & Authorization

19. **Auth Utilities**
    - Description: Implement bcrypt helpers and token utilities (`internal/auth/jwt.go` or `token.go`), including JWT claims/opaque option.
    - Dependencies: Tasks 2,13.

20. **Auth Middleware & RBAC**
    - Description: Implement `internal/auth/middleware.go` for token parsing/context injection and `roles.go` for role/scope checks.
    - Dependencies: Task 19.

21. **Auth Service**
    - Description: Build `internal/service/auth_service.go` handling registration, login, logout, password reset/change; integrate repositories, token utils, Redis blacklist/session storage.
    - Dependencies: Tasks 14–18,19–20.

22. **Auth Handlers & Routes**
    - Description: Implement HTTP handlers for `/auth/...` in `internal/api/http/handlers/users_handler.go` & `staff_handler.go`, plus DTOs in `internal/api/dto/user_dto.go`; wire in `router.go`.
    - Dependencies: Task 21.

---

## Phase 6 — Organization Management

23. **Org Services**
    - Description: Implement `internal/service/staff_service.go` (staff CRUD, activation, role changes) and optional `OrgService` for departments/teams with cache invalidation.
    - Dependencies: Tasks 15–16,21.

24. **Org DTOs**
    - Description: Define request/response structs in `internal/api/dto/staff_dto.go` (staff, departments, teams).
    - Dependencies: Task 23.

25. **Org Handlers**
    - Description: Extend `staff_handler.go` or add dedicated handler files for department/team/staff endpoints with RBAC enforcement; update `router.go`.
    - Dependencies: Tasks 20,23–24.

---

## Phase 7 — Ticket Management

26. **Assignment Service**
    - Description: Implement `internal/service/assignment_service.go` with auto-assignment (round-robin using Redis), self-assign, assign-to-staff/team logic, ticket history hooks.
    - Dependencies: Tasks 17–18,11–12.

27. **Ticket Service**
    - Description: Implement `internal/service/ticket_service.go` covering end-user and staff workflows (create, list, detail, status/prio updates, close, add messages/history).
    - Dependencies: Tasks 17–18,21,26.

28. **Ticket DTOs**
    - Description: Define request/response structs in `internal/api/dto/ticket_dto.go` (ticket payloads, filters, message payloads).
    - Dependencies: Task 27.

29. **End-User Ticket Handlers**
    - Description: Implement `internal/api/http/handlers/tickets_handler.go` for `/tickets` routes (create/list/get/messages/close) with validation and RBAC.
    - Dependencies: Tasks 20,27–28.

30. **Staff Ticket Handlers**
    - Description: Implement staff ticket endpoints (list/detail/assign/status/priority/messages/history) in `staff_handler.go` or dedicated file; ensure scope enforcement.
    - Dependencies: Tasks 20,26–28.

31. **History Integration**
    - Description: Ensure TicketService writes `TicketHistory` entries for each change and exposes history endpoints; verify DTOs include data.
    - Dependencies: Tasks 27–30.

---

## Phase 8 — Attachments & External References

32. **Attachment Repository Integration**
    - Description: Finalize `attachment_repository.go` and integrate with message creation in services.
    - Dependencies: Task 18,27.

33. **Attachment Validation**
    - Description: Extend DTOs/handlers to validate MIME types and size using config; update `AttachmentService` helper.
    - Dependencies: Tasks 28–29,32.

---

## Phase 9 — Events & Notifications

34. **Event Definitions & Dispatcher**
    - Description: Implement `internal/events/event_types.go` and `dispatcher.go` with publish/subscribe API.
    - Dependencies: Tasks 13,27.

35. **Notification Service & Worker**
    - Description: Implement `internal/service/notification_service.go` subscribing to events and `internal/worker/notification_worker.go` for async processing (logging/webhook stub).
    - Dependencies: Tasks 21,27,34.

36. **Webhook Admin Endpoints**
    - Description: Add repository/table (if needed) and handlers for `/admin/webhooks` (POST/GET) in `staff_handler.go` or new handler; update DTOs.
    - Dependencies: Tasks 23–25,34–35.

---

## Phase 10 — Observability & Error Handling

37. **App Error Utilities**
    - Description: Implement `pkg/util/errorutil.go` defining `AppError`, helpers, and mapping from domain errors to HTTP responses; integrate with middleware.
    - Dependencies: Task 3.

38. **Metrics Instrumentation**
    - Description: Add metrics counters/histograms in middleware and key services; optionally expose `/metrics`.
    - Dependencies: Tasks 3,27.

39. **Health Readiness Checks**
    - Description: Enhance `health_handler.go` to use Postgres/Redis health helpers; ensure readiness route returns dependency map.
    - Dependencies: Tasks 5,10,37.

---

## Phase 11 — Testing

40. **Service Unit Tests**
    - Description: Add `_test.go` files for auth, ticket, assignment, staff services using mocks (e.g., `testify/mock`).
    - Dependencies: Tasks 21,23,26–27.

41. **Repository Integration Tests**
    - Description: Write integration tests under `internal/repository` using test Postgres with migrations; cover query filters and history writes.
    - Dependencies: Tasks 14–18,9.

42. **HTTP Handler Tests**
    - Description: Use Fiber test utilities to cover auth, tickets, staff endpoints verifying RBAC and error responses.
    - Dependencies: Tasks 22,29–30,37.

43. **Cache & Event Tests**
    - Description: Tests for cache helpers (`internal/cache`), assignment round-robin (Redis), and event dispatcher/notification.
    - Dependencies: Tasks 11–12,26,34–35.

---

## Phase 12 — Scripts & Developer Experience

44. **Developer Scripts**
    - Description: Implement `scripts/run_dev.sh`, `scripts/migrate_up.sh`, `scripts/migrate_down.sh`; ensure executable and documented.
    - Dependencies: Tasks 1,9.

45. **README & Docs**
    - Description: Update `README.md` with setup steps, env vars, running tests, core API overview.
    - Dependencies: All prior tasks (for accuracy).

---
