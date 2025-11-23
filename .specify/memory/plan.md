# Technical Implementation Plan: Support Ticket Service

**Version**: 1.0  
**Authors**: Codex (GPT-5)  
**Date**: 2025-11-23  
**Sources**: Constitution (2025-11-23) & Product Spec (`.specify/memory/spec.md`)

---

## 1. Architecture & Boundaries

### Layer Overview

| Layer | Packages | Responsibilities | Dependencies |
|-------|----------|------------------|--------------|
| HTTP/API | `internal/api/http`, `internal/api/dto`, `internal/auth` | Fiber app setup, routing, DTOs, validation, middleware (auth, RBAC, logging), HTTP error mapping | Calls services only |
| Service (Domain) | `internal/service`, `internal/domain` | Business logic, entity orchestration, RBAC enforcement, transaction coordination, event emission | Depend on repositories, cache interfaces, event dispatcher |
| Repository | `internal/repository`, `internal/persistence` | DB + Redis access, queries, migrations, data mapping | Depend on persistence clients (`postgres.go`, `redis.go`) |
| Cache | `internal/cache` | Reference data cache, ticket metadata cache, token/session storage (if opaque tokens) | Uses Redis connections |
| Auth & RBAC | `internal/auth` | Token generation/verification, middleware, role-scope utilities | Uses services for user lookup, cache for tokens |
| Events & Workers | `internal/events`, `internal/worker` | Event definitions, dispatcher, notification worker for async delivery | Subscribes to service events |
| Observability | `internal/observability` | Logging, metrics, tracing helpers | Used by middleware + services |

**Dependency Flow**: `handlers -> service -> repository/cache -> persistence`. Services emit events to dispatcher; workers subscribe. Repositories never import services or handlers. DTOs map to/from domain models via translators in `internal/api/dto`.

### Fiber Setup

- `cmd/api/main.go` loads config, init logging/metrics, connect Postgres/Redis, run migrations, compose dependencies, build router, start Fiber with graceful shutdown.  
- `internal/api/http/router.go` registers route groups: `/auth`, `/tickets`, `/staff`, `/admin`, `/health`.  
- `internal/api/http/middleware.go` defines request logging, recovery, validation errors, correlation ID injection.  
- Handlers reside per area (`tickets_handler.go`, `users_handler.go`, `staff_handler.go`, `health_handler.go`). Each handler struct holds required services.

---

## 2. Database Schema (PostgreSQL)

### Tables

1. **users**
   - `id UUID PK`
   - `name varchar(200) NOT NULL`
   - `email citext UNIQUE NOT NULL`
   - `password_hash varchar(255) NOT NULL`
   - `status user_status NOT NULL DEFAULT 'ACTIVE'`
   - `created_at timestamptz NOT NULL DEFAULT now()`
   - `updated_at timestamptz NOT NULL DEFAULT now()`
   - Index: `users_email_idx`

2. **staff_members**
   - `id UUID PK`
   - `name varchar(200) NOT NULL`
   - `email citext UNIQUE NOT NULL`
   - `password_hash varchar(255) NOT NULL`
   - `role staff_role NOT NULL`
   - `department_id UUID REFERENCES departments(id)`
   - `team_id UUID REFERENCES teams(id)`
   - `active_flag boolean NOT NULL DEFAULT true`
   - `created_at`, `updated_at`
   - Indexes: `staff_email_idx`, `staff_department_idx`, `staff_team_idx`

3. **departments**
   - `id UUID PK`
   - `name varchar(120) UNIQUE NOT NULL`
   - `description text`
   - `is_active boolean NOT NULL DEFAULT true`
   - `created_at`, `updated_at`

4. **teams**
   - `id UUID PK`
   - `department_id UUID NOT NULL REFERENCES departments(id)`
   - `name varchar(120) NOT NULL`
   - `description text`
   - `is_active boolean NOT NULL DEFAULT true`
   - `created_at`, `updated_at`
   - Unique `(department_id, name)`
   - Index `teams_department_idx`

5. **tickets**
   - `id UUID PK`
   - `external_key varchar(32) UNIQUE NOT NULL`
   - `requester_user_id UUID NOT NULL REFERENCES users(id)`
   - `department_id UUID NOT NULL REFERENCES departments(id)`
   - `team_id UUID REFERENCES teams(id)`
   - `assignee_staff_id UUID REFERENCES staff_members(id)`
   - `title varchar(200) NOT NULL`
   - `description text NOT NULL`
   - `status ticket_status NOT NULL DEFAULT 'OPEN'`
   - `priority ticket_priority NOT NULL DEFAULT 'MEDIUM'`
   - `tags text[] NOT NULL DEFAULT '{}'`
   - `created_at`, `updated_at`, `closed_at timestamptz`
   - Indexes: composite indexes for `(department_id, status)`, `(team_id, status)`, `(assignee_staff_id, status)`, `(priority)`, `(created_at DESC)`.

6. **ticket_messages**
   - `id UUID PK`
   - `ticket_id UUID NOT NULL REFERENCES tickets(id)`
   - `author_type message_author_type NOT NULL`
   - `author_id UUID NULL` (FK to users or staff when applicable)
   - `message_type ticket_message_type NOT NULL`
   - `body text NOT NULL`
   - `created_at timestamptz NOT NULL DEFAULT now()`
   - Index: `ticket_messages_ticket_idx`

7. **ticket_history**
   - `id UUID PK`
   - `ticket_id UUID NOT NULL REFERENCES tickets(id)`
   - `changed_by_type history_actor_type NOT NULL`
   - `changed_by_id UUID NULL`
   - `change_type ticket_change_type NOT NULL`
   - `old_value jsonb`
   - `new_value jsonb`
   - `created_at timestamptz NOT NULL DEFAULT now()`
   - Index: `ticket_history_ticket_idx`

8. **attachment_references**
   - `id UUID PK`
   - `ticket_message_id UUID NOT NULL REFERENCES ticket_messages(id)`
   - `storage_key varchar(255) NOT NULL`
   - `file_name varchar(255) NOT NULL`
   - `mime_type varchar(128) NOT NULL`
   - `size_bytes bigint NOT NULL`
   - `created_at timestamptz NOT NULL DEFAULT now()`

9. **password_reset_tokens**
   - `id UUID PK`
   - `subject_type reset_subject_type NOT NULL` (`USER`/`STAFF`)
   - `subject_id UUID NOT NULL`
   - `token varchar(128) UNIQUE NOT NULL`
   - `expires_at timestamptz NOT NULL`
   - `used_at timestamptz`
   - Index `password_reset_token_idx` on `(subject_type, subject_id)`

10. **sessions** (optional if opaque tokens)
    - `token varchar(128) PK`
    - `subject_type session_subject_type`
    - `subject_id UUID`
    - `role staff_role NULL` (for staff)
    - `expires_at timestamptz`
    - `created_at timestamptz`

### Enums / Constraints

- `user_status`: `ACTIVE`, `SUSPENDED`.  
- `staff_role`: `AGENT`, `TEAM_LEAD`, `ADMIN`.  
- `ticket_status`: `OPEN`, `IN_PROGRESS`, `PENDING_USER`, `RESOLVED`, `CLOSED`, `CANCELLED`.  
- `ticket_priority`: `LOW`, `MEDIUM`, `HIGH`, `URGENT`.  
- `message_author_type`: `USER`, `STAFF`, `SYSTEM`.  
- `ticket_message_type`: `PUBLIC_REPLY`, `INTERNAL_NOTE`, `SYSTEM_EVENT`.  
- `ticket_change_type`: `STATUS_CHANGE`, `ASSIGNEE_CHANGE`, `PRIORITY_CHANGE`, `TEAM_CHANGE`, `DEPARTMENT_CHANGE`, `TAGS_CHANGE`.  
- Additional check constraints for `team_id` referencing same department as ticket via trigger or service validation.

### Mapping Notes

- Tags stored as `text[]`; service enforces max length and sanitization.  
- `ticket_history.old_value`/`new_value` store JSON fragments describing changed fields.  
- Attachment storage is metadata only; actual files stored externally.  
- Auto-generated external key stored as `varchar`, unique index ensures no duplicates.

---

## 3. Redis Usage & Caching Strategy

### Keys & Namespaces

| Usage | Key Pattern | Value | TTL | Invalidation |
|-------|-------------|-------|-----|--------------|
| Department cache | `dept:{id}` | JSON serialized department | 10m | Delete on department update/delete |
| Team cache | `team:{id}` | JSON serialized team | 10m | Delete when team updated/deactivated |
| Ticket summary | `ticket:summary:{id}` | `{status, priority, assignee_id, updated_at}` | 5m | Delete/update after ticket write |
| Session tokens (opaque) | `session:{token}` | `{subject_id, role, type, expires_at}` | TTL = token expiration | Delete on logout |
| Password reset tokens (alternate to DB) | `pwdreset:{token}` | `{subject_type, subject_id, expires_at}` | TTL configurable | Delete on use |

### Strategy

- **Write-through**: After updating departments/teams in Postgres, services call cache invalidation helpers. Reads fallback to repository if miss; results stored with TTL.  
- **Ticket summary cache**: On ticket status/prio/assignee update, service writes new summary record for quick staff list endpoints; list endpoints may fetch base query from DB and overlay cached summary for freshness.  
- **Session storage**: If JWT chosen, Redis used solely for logout blacklist entries (`blacklist:{jti}` TTL=token expiry). If opaque tokens, Redis is authoritative store.

---

## 4. Authentication & Authorization Design

### Password Handling

- Use `golang.org/x/crypto/bcrypt` with cost configurable via env (`AUTH_BCRYPT_COST`, default 12).  
- Password validation: min length 12, complexity rule enforced in service.

### Token Strategy

- **JWT (default)**:
  - Signing: HMAC SHA-256 with secret from env (`AUTH_JWT_SECRET`).  
  - Claims: `sub`, `role`, `type` (`END_USER` or `STAFF`), `exp`, `jti`, `department_ids`/`team_ids` (for staff scope).  
  - Stateless; logout uses Redis `blacklist:{jti}` TTL=exp.  
- **Opaque Token option**:
  - Random UUID tokens stored in Redis `session:{token}` with TTL; logout deletes key.

### Middleware Flow

1. `AuthMiddleware` (in `internal/auth/middleware.go`):
   - Extracts `Authorization: Bearer <token>`.  
   - Validates token (JWT or Redis lookup).  
   - Checks blacklist in Redis (if JWT).  
   - Loads lightweight subject info (id, role, type, department/team scope).  
   - Injects into `context.Context`.

2. `RBACMiddleware`:
   - Ensures required role(s) present.  
   - For staff endpoints, verifies ticket/department/team scope by calling `StaffScopeChecker` in services.

3. Password reset flows:
   - Tokens generated and stored in DB or Redis with TTL; hashed token stored (optional) for security.

---

## 5. API Design

### DTO Conventions

- Request structs defined in `internal/api/dto/*`.  
- Use `validator` tags to enforce required fields, lengths, enums.  
- Responses wrap domain data: `{ "data": {...}, "meta": {...} }`. Errors use `{ "error": { "code": "...", "message": "...", "details": {...} } }`.

### Endpoint Implementation Notes

(Summaries; refer to spec for exhaustive list.)

#### Auth Endpoints

- `POST /auth/users/register`: Validates name/email/password, ensures unique email via repo. Service hashes password, stores user, returns sanitized profile + token (auto-login optional).  
- `POST /auth/users/login`: Validates credentials, issues token, records login event in history/logs.  
- `POST /auth/staff/login`: Similar but checks `active_flag`.  
- `POST /auth/logout`: Requires auth, invalidates token (Redis).  
- `POST /auth/password/reset/request`: Accept email; create token, store in DB/Redis, log event, optionally enqueue notification.  
- `POST /auth/password/reset/confirm`: Validate token, update password hash, mark token used.  
- `POST /auth/password/change`: Auth required; verify current password before updating.

#### Organization Management

- Departments, teams, staff endpoints enforce ADMIN/TEAM_LEAD roles as defined.  
- DTOs include name, description, is_active toggles.  
- On updates, run validators to ensure referential integrity (team belongs to active dept).  
- Cache invalidation triggered post-commit.

#### End-User Tickets

- `POST /tickets`: Validate dept/team existence and active status, optional priority/tags, attachments metadata. Service transaction: insert ticket, initial history, optional auto-assignment, emit `TicketCreated`.  
- `GET /tickets`: Query limited to requester via repo `ListByRequester`.  
- `GET /tickets/{id}`: Service ensures ownership, fetches ticket + messages (PUBLIC_REPLY + appropriate SYSTEM events) + truncated history.  
- `POST /tickets/{id}/messages`: Validate ticket ownership and status (deny if closed). Create message + optional attachments within transaction, emit `TicketMessageAdded`.  
- `POST /tickets/{id}/close`: Verify status (`RESOLVED` or `PENDING_USER`), set to `CLOSED`, record history.

#### Staff Tickets

- `GET /staff/tickets`: Accept filter DTO; repository builds dynamic query with indexes. Additional scope filter applied in service.  
- `POST /staff/tickets/{id}/assign*`: Validate role, scope, and target staff availability; update ticket and history in transaction, emit `TicketAssigned`.  
- `POST /staff/tickets/{id}/status`: Enforce transition table in service; record history, emit `TicketStatusChanged`.  
- `POST /staff/tickets/{id}/priority`: Validate enums; record history.  
- `POST /staff/tickets/{id}/messages`: Allows `PUBLIC_REPLY` or `INTERNAL_NOTE` depending on role; `INTERNAL_NOTE` flagged for user filtering.  
- `GET /staff/tickets/{id}/history`: Returns full history list.

#### Notifications & Webhooks

- `POST /admin/webhooks`: ADMIN only; validate URL, store in DB table `webhook_endpoints` (if added).  
- `GET /admin/webhooks`: List configured endpoints.  
- Notification worker reads event queue and posts to configured endpoints (stubbed logging for MVP).

#### Health/Observability

- `GET /health/live`: Always `200` if server running.  
- `GET /health/ready`: Checks Postgres (`SELECT 1`) and Redis (`PING`); returns detailed JSON structure.

### Error Mapping

- Validation errors → `400` with `VALIDATION_FAILED`.  
- Auth failures → `401` `AUTHENTICATION_FAILED`.  
- RBAC violations → `403` `AUTHORIZATION_FAILED`.  
- Missing resources → `404` `NOT_FOUND`.  
- Business rule violations → `409` `BUSINESS_RULE_VIOLATION`.  
- Internal errors → `500` `INTERNAL_ERROR` with correlation id.

---

## 6. Services & Use Cases

### AuthService
- Methods: `RegisterUser`, `LoginUser`, `LoginStaff`, `Logout`, `RequestPasswordReset`, `ConfirmPasswordReset`, `ChangePassword`.  
- Dependencies: `UserRepository`, `StaffRepository`, `PasswordResetRepository`, `TokenManager`, `EventDispatcher`, `Metrics`.  
- Transactions: Registration + initial user creation; password reset confirmation.

### UserService
- Methods: `GetProfile`, `UpdateProfile`, `ListUserTickets` (delegates to TicketService).  
- Dependencies: `UserRepository`.

### StaffService
- Methods: `CreateStaff`, `ListStaff`, `GetStaff`, `UpdateStaff`, `SetActiveStatus`.  
- Dependencies: `StaffRepository`, `DepartmentRepository`, `TeamRepository`, `Cache`.  
- Transactions: Staff creation/update (ensuring consistent team assignments).

### TicketService
- Methods: `CreateTicket`, `ListUserTickets`, `GetUserTicket`, `AddUserMessage`, `CloseTicket`, `ListStaffTickets`, `GetStaffTicket`, `AddStaffMessage`, `UpdateStatus`, `UpdatePriority`, `AssignSelf`, `AssignStaff`, `AssignTeam`.  
- Dependencies: Ticket-related repositories, `AssignmentService`, `NotificationService`, `Cache`.  
- Transactions: Ticket creation, message posting (with attachments), status/priority updates, assignment changes.

### AssignmentService
- Methods: `AutoAssign(ticketID)`, `AssignToStaff(ticketID, staffID)`, `AssignToTeam(ticketID, teamID)`, `SelfAssign(ticketID, staffID)`.  
- Dependencies: `TicketRepository`, `StaffRepository`, `TeamRepository`, `TicketHistoryRepository`.  
- Implements round-robin per team using Redis list `team:{id}:queue` storing staff IDs; rotates on each assignment.

### NotificationService
- Methods: `HandleTicketCreated`, `HandleTicketMessage`, `HandleStatusChange`, `DispatchWebhooks`.  
- Dependencies: `EventDispatcher`, `Worker`, `Observability`.  
- MVP behavior: log structured event; if webhook endpoints exist, enqueue HTTP POST to worker; worker currently logs success/failure.

### Supporting Services
- `DepartmentService`, `TeamService` for clarity (wrapping repositories + cache).  
- `AttachmentService` handles metadata validation.

---

## 7. Events & Notifications

### Event Model

- Define `Event` struct: `{Type string, Timestamp time.Time, Payload interface{}, Metadata map[string]string}` in `internal/events/event_types.go`.  
- Event types: `TicketCreated`, `TicketStatusChanged`, `TicketPriorityChanged`, `TicketAssigned`, `TicketMessageAdded`, `StaffPublicReply`, `TicketClosed`.  
- Payload examples: `TicketStatusChangedPayload{TicketID, OldStatus, NewStatus, ActorID, ActorType}`.

### Dispatcher

- Interface: `Dispatch(ctx context.Context, evt Event) error`.  
- Implementation: in-memory channel-based dispatcher with subscriber registry.  
- Services call `dispatcher.Dispatch`.  
- `internal/worker/notification_worker.go` subscribes to relevant events and processes sequentially or via goroutines.

### Notification Flow

1. Service emits event.  
2. Dispatcher pushes to worker queue (buffered channel).  
3. Worker handles event, e.g., for `TicketCreated` logs message, optionally sends HTTP POST to each webhook URL (stored in DB/repo).  
4. Retries: simple retry with exponential backoff limited attempts, logged failures.

---

## 8. Observability & Error Handling

### Logging

- Use structured logger (e.g., `zerolog` or `zap`).  
- Middleware logs: request id, method, path, status, latency, user/staff id (if available).  
- Services log key events (`ticket_id`, `action`, `actor`).  
- Sensitive data redacted (password, tokens, message bodies for internal notes optionally hashed).

### Metrics

- Use Prometheus client. Metrics:  
  - `http_requests_total{path, method, status}`  
  - `http_request_duration_seconds` histogram  
  - `auth_failures_total`  
  - `tickets_created_total`, `ticket_status_transitions_total{from,to}`  
  - `notification_dispatch_total{result}`  
- `/metrics` endpoint optional (if exposing).

### Health Checks

- `health_handler.go` implements `/health/live` (returns uptime, version) and `/health/ready` ( Postgres `db.PingContext`, Redis `Ping`).  
- Response JSON includes dependency status map.

### Error Handling

- Define `type AppError struct { Code string; Message string; Status int; Details map[string]any }`.  
- Services return `AppError`; handlers map to HTTP responses.  
- Validation errors aggregated using `validator` library; convert to `details.field`.  
- Panic recovery middleware logs stack traces and returns `500`.

---

## 9. Testing Strategy

- **Unit Tests** (`internal/service`, `internal/auth`): Use Go test with mocks (e.g., `testify/mock`). Cover business rules (status transitions, RBAC enforcement, assignment logic).  
- **Repository Integration Tests** (`internal/repository`): Run against test Postgres using Docker/local DB; apply migrations before suite and teardown afterwards. Use transaction rollback per test or schema reset.  
- **HTTP Integration Tests** (`internal/api/http`): Use Fiber test server; mock services or run with in-memory repos. Cover core flows (`POST /tickets`, `POST /staff/tickets/{id}/status`).  
- **Cache Tests**: Use Redis test instance; verify TTL/invalidation.  
- **Event/Worker Tests**: Use in-memory dispatcher verifying subscribers invoked.  
- Test data via fixtures builder functions in `internal/testsupport`.  
- CI pipeline: run `go test ./...`, plus linting (golangci-lint) and migrations check.

---

## 10. Deployment & Configuration

- **Configuration Loading**: `internal/config/config.go` reads from `.env` + YAML (optional) + environment variables, populating `Config` struct (server port, DB DSN, Redis address, JWT secret, bcrypt cost, pagination defaults, attachment limits).  
- Provide `.env.example` with placeholders.  
- `cmd/api/main.go` uses config to init Postgres (`internal/persistence/postgres.go` using `pgxpool`), Redis client, logger, metrics, router.

- **Migrations**: SQL files in `/migrations`. Scripts `scripts/migrate_up.sh` and `migrate_down.sh` call migration tool (e.g., `golang-migrate`). `internal/persistence/migrations.go` optionally runs migrations at startup based on config flag.

- **Runtime**: 
  - Fiber server listens on `CONFIG.HTTP_PORT`, graceful shutdown via context cancellation.  
  - Connection pools: Postgres (max connections config), Redis (Go-redis).  
  - Background worker (notification) runs via goroutine, stops on shutdown.

- **Local Dev**: Provide `docker-compose` (future) or docs for starting Postgres + Redis. `scripts/run_dev.sh` sets env vars and runs server with live reload (air) optionally.

---

## 11. Implementation Sequencing

1. **Bootstrap Project**: go.mod, folder structure, logging/metrics scaffolding.  
2. **Config & Persistence**: config loader, Postgres/Redis clients, migrations for core schema.  
3. **Domain & Repository Layer**: domain models, repositories with CRUD, integration tests.  
4. **Auth Layer**: password hashing utilities, token manager, middleware, auth handlers.  
5. **Ticket Service & Handlers**: implement core ticket flows, user endpoints, history, assignments.  
6. **Staff & Org Management**: departments/teams/staff services + handlers + cache.  
7. **Notifications/Event System**: event dispatcher, notification worker, webhook stub.  
8. **Observability & Health**: finalize logging, metrics, health endpoints.  
9. **Testing & Hardening**: complete unit/integration tests, load basic data, run lint/tests.  
10. **Docs & Scripts**: README, scripts for running/migrating, .env example.

---

## 12. Risk & Mitigation

- **RBAC Complexity**: Mitigate with centralized scope checker utility and exhaustive tests.  
- **Auto-assignment fairness**: Use Redis queues and ensure failure handling falls back gracefully.  
- **Eventual Consistency between cache and DB**: Provide cache-busting helpers invoked post-transaction commit.  
- **Error proliferation**: Use consistent `AppError` type and middleware for mapping.  
- **Testing flakiness**: Use deterministic fixtures, isolated DB schemas per test run.

---

This plan aligns with the constitution and product spec, detailing architecture, schema, services, and operational practices required to implement the Support Ticket Service using Go/Fiber, PostgreSQL, and Redis.
