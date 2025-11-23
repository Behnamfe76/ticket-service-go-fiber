# Feature Specification: Support Ticket Service (Backend Only)

**Feature Branch**: `[000-support-ticket-service]`  
**Created**: 2025-11-23  
**Status**: Draft  
**Input**: Backend-only Support Ticket System using Go + Fiber, PostgreSQL, Redis, REST/JSON APIs, email+password auth only.

## User Scenarios & Testing

### User Story 1 - End-User Ticket Lifecycle (Priority: P1)
An end-user registers with email/password, logs in, files a ticket with attachments, exchanges public replies with staff, and closes the ticket after resolution while seeing only their tickets and relevant history.

**Why this priority**: Provides the core customer value; without it there is no product.

**Independent Test**: API tests calling `/auth/users/register`, `/auth/users/login`, `/tickets`, `/tickets/{id}`, `/tickets/{id}/messages`, and `/tickets/{id}/close` using a single user and verifying ticket status transitions and response payloads.

**Acceptance Scenarios**:
1. **Given** a new user registers and logs in, **When** they submit `POST /tickets` with valid data, **Then** the system creates a ticket with status `OPEN`, generates an `external_key`, stores attachments references, and publishes a `TicketCreated` event.
2. **Given** a ticket is `RESOLVED`, **When** the requester calls `POST /tickets/{id}/close`, **Then** the ticket moves to `CLOSED`, emits history entries, and becomes read-only for further user edits.

---

### User Story 2 - Staff Ticket Operations (Priority: P1)
An authenticated staff agent filters tickets assigned to their department/team, views complete ticket details (messages, history, attachments), posts public replies or internal notes, and updates status/priority with audit logs.

**Why this priority**: Staff productivity determines time-to-resolution and SLA compliance.

**Independent Test**: Integration suite hitting `/staff/tickets`, `/staff/tickets/{id}`, `/staff/tickets/{id}/messages`, `/staff/tickets/{id}/status`, and `/staff/tickets/{id}/history` under different roles and verifying RBAC enforcement and audit logs.

**Acceptance Scenarios**:
1. **Given** an AGENT authenticated for Department A, **When** they list `/staff/tickets?department_id=A`, **Then** only tickets within authorized scope are returned, paginated, and filterable by status/priority.
2. **Given** a TEAM_LEAD posts an `INTERNAL_NOTE` on `/staff/tickets/{id}/messages`, **Then** the note is hidden from end-users but visible to staff, with `TicketHistory` capturing the action.

---

### User Story 3 - Administrative Organization Management (Priority: P2)
An ADMIN configures departments, teams, and staff accounts, assigns team leads, and ensures only active entities participate in routing/assignment.

**Why this priority**: Proper org structure is prerequisite for controlled staff access and workload distribution.

**Independent Test**: Admin-only API tests for `/staff/departments`, `/staff/teams`, `/staff/members` verifying create/update/list operations, uniqueness, and downstream references (e.g., team must belong to an existing department).

**Acceptance Scenarios**:
1. **Given** a department exists, **When** an ADMIN creates a team via `POST /staff/teams`, **Then** the team references the department, defaults to active, and becomes selectable for ticket routing.
2. **Given** a staff member is deactivated, **When** auto-assignment runs, **Then** the inactive staff member is skipped and history reflects reassignment.

---

### User Story 4 - Security, Audit, and Observability (Priority: P2)
Operators rely on structured logs, metrics, and health endpoints to monitor authentication, ticket lifecycle events, and infrastructure health while enforcing email+password-only authentication and RBAC.

**Why this priority**: Ensures safety, compliance, and operational readiness.

**Independent Test**: Automated checks hitting `/health/live` and `/health/ready`, verifying logs redact secrets, validating audit entries on status/assignee changes, and ensuring errors follow the JSON envelope.

**Acceptance Scenarios**:
1. **Given** an invalid login attempt occurs, **When** `/auth/users/login` receives wrong credentials, **Then** the response is a consistent JSON error, bcrypt comparison is performed, and logs contain only anonymized context.
2. **Given** PostgreSQL is unreachable, **When** `/health/ready` is requested, **Then** the endpoint responds `503` with reasons and metrics increment readiness failures.

### Edge Cases

- Ticket creation fails if `department_id` references an inactive or nonexistent department, or if `team_id` does not belong to that department.
- Password reset tokens expire based on configurable TTL and are single-use; replay attempts return `error.code = "TOKEN_INVALID"`.
- End-users may not close tickets that are `OPEN` or `IN_PROGRESS`; request returns validation error referencing allowed states.
- Staff cannot post `INTERNAL_NOTE` messages unless they are AGENT+ and assigned to the ticket's department/team.
- Attachment metadata exceeding configured size or MIME policy is rejected before persistence.
- Auto-assignment gracefully handles empty eligible staff pools by leaving the ticket unassigned and flagging history.

## Requirements

### Functional Requirements

- **FR-001**: System MUST support email+password registration and login for end-users only via `/auth/users/register` and `/auth/users/login`.
- **FR-002**: System MUST support staff login via `/auth/staff/login`, enforcing bcrypt hashing and audit logging.
- **FR-003**: All authenticated endpoints MUST require a valid auth token (JWT or opaque) issued after login and revocable via `/auth/logout`.
- **FR-004**: Password reset flows MUST require requesting a token (`/auth/password/reset/request`) and confirming via token + new password (`/auth/password/reset/confirm`).
- **FR-005**: End-users MUST be able to create, list, view, reply to, and close their own tickets only.
- **FR-006**: Staff members MUST manage tickets (list, filter, view, assign, change status/priority, reply, add notes) based on role and department/team scope.
- **FR-007**: ADMINs MUST CRUD departments, teams, and staff members via dedicated endpoints, enforcing uniqueness and active flags.
- **FR-008**: Ticket history MUST automatically record all status, priority, assignee, department, team, and tag changes with JSON payloads.
- **FR-009**: Ticket messages MUST capture author type (USER/STAFF/SYSTEM), message type (PUBLIC_REPLY/INTERNAL_NOTE/SYSTEM_EVENT), body, attachments, and timestamps.
- **FR-010**: AttachmentReference entries MUST only reference external storage (no binary data) with file metadata validation.
- **FR-011**: Notification hooks MUST emit internal events (TicketCreated, StaffReplied, StatusChanged) and integrate with a `NotificationService` interface (stubbed logging in MVP).
- **FR-012**: Health endpoints `/health/live` and `/health/ready` MUST exist; readiness checks PostgreSQL + Redis connections.
- **FR-013**: Observability MUST include structured request logs, domain event logs, and metrics (requests, errors, ticket lifecycle counts).
- **FR-014**: RBAC MUST enforce END_USER, AGENT, TEAM_LEAD, ADMIN capabilities detailed below; violations return `403`.
- **FR-015**: API responses MUST follow a consistent JSON envelope for success and errors.
- **FR-016**: Pagination MUST be available on all list endpoints; default `page=1`, `page_size` configurable with caps.
- **FR-017**: Input validation MUST enforce required fields, string lengths, enum membership, email format, and sanitized text bodies.
- **FR-018**: Auto-assignment MUST support a simple round-robin per team (skipping inactive staff) when `team_id` provided on ticket creation.
- **FR-019**: Redis MUST store session tokens/refresh tokens and cache reference data (departments, teams) with TTLs and invalidation on updates.
- **FR-020**: PostgreSQL migrations MUST define and evolve the schema for all entities (users, staff, departments, teams, tickets, ticket_messages, ticket_history, attachments).

### Key Entities

- **User**: Represents external requester; fields `id`, `name`, `email (unique)`, `password_hash`, `status (ACTIVE|SUSPENDED)`, timestamps.
- **StaffMember**: Internal operator with `role (AGENT|TEAM_LEAD|ADMIN)`, `active_flag`, optional team associations, hashed password.
- **Department**: Organizational grouping for tickets; `is_active` controls availability.
- **Team**: Sub-group under a department; used for assignment and scope checks; inherits department for RBAC.
- **Ticket**: Primary workload with `external_key`, requester reference, optional assignment fields, `status`, `priority`, `tags[]`, timestamps.
- **TicketMessage**: Conversation entries tied to tickets with author metadata, message type, body, attachments.
- **TicketHistory**: Audit log entries referencing change types and old/new JSON values.
- **AttachmentReference**: Links message to stored file location (e.g., S3 key) and metadata.
- **NotificationEndpoint** (future-ready): Admin-configured webhooks for events; stored but minimal behavior in MVP.

### Architecture Overview

1. **HTTP Layer (Fiber)**  
   - Routes grouped by context: `/auth`, `/tickets`, `/staff`, `/admin`, `/health`.  
   - DTOs for requests/responses with validation (e.g., email regex, enum checks).  
   - Middleware: authentication (token parsing), authorization (role + scope), logging, correlation IDs.

2. **Service Layer**  
   - Modules: `AuthService`, `UserService`, `StaffService`, `TicketService`, `TicketAssignmentService`, `NotificationService`, `AttachmentService`.  
   - Contains business logic, RBAC enforcement, domain events, interaction orchestration.  
   - Uses context-aware interfaces to repositories and external services (email/webhook stub, metrics, logging).

3. **Repository Layer**  
   - PostgreSQL repositories per aggregate (Users, Staff, Departments, Teams, Tickets, TicketMessages, TicketHistory, Attachments).  
   - Redis repositories for token/session storage, password reset tokens, cached reference data.  
   - Transactions for multi-table writes (ticket create + history + initial message).  
   - Query optimization with indexes on `status`, `priority`, `department_id`, `team_id`, `assignee_staff_id`, `created_at`.

4. **Configuration**  
   - Environment variables define DB/Redis connection strings, JWT secret or signing keys, password policy, pagination defaults, auto-assignment settings, attachment size limits.  
   - Safe defaults for local dev; secrets never stored in code.

### RBAC Matrix

| Capability | END_USER | AGENT | TEAM_LEAD | ADMIN |
|------------|----------|-------|-----------|-------|
| Register/Login | ✅ (users) | ✅ (staff login) | ✅ | ✅ |
| Create tickets | ✅ | ❌ | ❌ | ❌ |
| View tickets | Own tickets only | Dept/team scoped | Dept/team scoped + team mgmt | All |
| Reply publicly | ✅ (own tickets) | ✅ | ✅ | ✅ |
| Internal notes | ❌ | ✅ | ✅ | ✅ |
| Update status/priority | ❌ | ✅ (scoped) | ✅ (scoped) | ✅ |
| Assign tickets | ❌ | Self-assign only | Assign within teams | Global |
| Manage departments/teams/staff | ❌ | ❌ | Limited (team-level) | ✅ |
| Configure notifications | ❌ | ❌ | ❌ | ✅ |

### Ticket Lifecycle & Assignment Behavior

1. **Creation**: Default status `OPEN`, priority `MEDIUM` unless specified. Generates `external_key` (`TCK-<YYYY>-<sequence>`). Optional auto-assignment triggered when `team_id` provided; uses round-robin list of active staff within team.
2. **Status Rules**:
   - `OPEN` → `IN_PROGRESS|CANCELLED`.
   - `IN_PROGRESS` → `PENDING_USER|RESOLVED|CANCELLED`.
   - `PENDING_USER` → `IN_PROGRESS|RESOLVED|CANCELLED`.
   - `RESOLVED` → `CLOSED` (end-user) or `IN_PROGRESS` (staff reopen).
   - `CLOSED` and `CANCELLED` are terminal; only ADMIN can reopen via dedicated maintenance tool (not exposed initially).
3. **Closing by User**: Allowed when `status` is `RESOLVED` or `PENDING_USER` with resolution note; action logs history entry.
4. **Assignment**: 
   - `POST /staff/tickets/{id}/assign/self`: only within scope and only if unassigned or assigned to caller.  
   - `POST /staff/tickets/{id}/assign`: ADMIN or TEAM_LEAD assigning specific staff.  
   - `POST /staff/tickets/{id}/assign/team`: ADMIN or TEAM_LEAD re-route to team (updates `team_id`, optional auto-assign).  
   - Auto-assignment skip rules: inactive staff, staff over configured workload (optional), fallback to unassigned.

### API Endpoints

Each response uses `{ "data": ..., "meta": {...} }` for success or `{ "error": { "code": "...", "message": "...", "details": {...} } }` for failures. `Auth` denotes required token type.

#### Authentication & Account

| Method | Path | Purpose | Auth | Role | Request | Response/Notes |
|--------|------|---------|------|------|---------|----------------|
| POST | `/auth/users/register` | Create end-user | None | N/A | `{name, email, password}` (password min 12 chars) | `201` with `{user_id, name, email, created_at}` |
| POST | `/auth/users/login` | End-user login | None | N/A | `{email, password}` | `200` with `{token, expires_in, user}` |
| POST | `/auth/staff/login` | Staff login | None | N/A | `{email, password}` | `200` with `{token, expires_in, staff}` |
| POST | `/auth/logout` | Logout/invalidate token | Bearer | Any | None | `204`, token invalidated via Redis blacklist |
| POST | `/auth/password/reset/request` | Email reset token | None | N/A | `{email}` | Always `202`, send token stub/log |
| POST | `/auth/password/reset/confirm` | Reset password | None | N/A | `{token, new_password}` | `200`, token revoked |
| POST | `/auth/password/change` | Change password | Bearer | Any | `{current_password, new_password}` | `200`, requires bcrypt verify |

#### Organization Management (ADMIN unless noted)

| Method | Path | Purpose | Auth Role | Key Validations |
|--------|------|---------|-----------|----------------|
| POST | `/staff/departments` | Create department | ADMIN | Name unique, description optional |
| GET | `/staff/departments` | List departments | ADMIN | Pagination, `is_active` filter |
| GET | `/staff/departments/{id}` | Department detail | ADMIN | 404 if missing |
| PUT | `/staff/departments/{id}` | Update | ADMIN | Name unique, toggling `is_active` cascades cache invalidation |
| POST | `/staff/teams` | Create team | ADMIN, TEAM_LEAD (within dept) | Department must exist/active |
| GET | `/staff/teams` | List teams | ADMIN/TEAM_LEAD | Filters: `department_id`, `is_active` |
| GET | `/staff/teams/{id}` | Team detail | ADMIN/TEAM_LEAD | RBAC ensures scope |
| PUT | `/staff/teams/{id}` | Update team | ADMIN/TEAM_LEAD | Cannot move team to inactive dept |
| POST | `/staff/members` | Create staff | ADMIN | Email unique, role validated, optional `team_id` |
| GET | `/staff/members` | List staff | ADMIN | Filters for role/team/department/active |
| GET | `/staff/members/{id}` | Staff detail | ADMIN | Includes team assignments |
| PUT | `/staff/members/{id}` | Update staff | ADMIN | Role transitions logged, password resets optional |

#### End-User Ticket APIs

| Method | Path | Purpose | Auth Role | Request Highlights | Response Highlights |
|--------|------|---------|-----------|--------------------|---------------------|
| POST | `/tickets` | Create ticket | END_USER | `{title, description, department_id, team_id?, priority?, tags?, attachments?}` | `201` with ticket payload, auto history entry |
| GET | `/tickets` | List my tickets | END_USER | Query: `status, priority, page, page_size` | Paginated tickets sorted by `created_at desc` |
| GET | `/tickets/{id}` | View my ticket | END_USER | Path `id` or `external_key` | Includes ticket, visible messages (`PUBLIC_REPLY`, relevant system), simplified history |
| POST | `/tickets/{id}/messages` | Add user reply | END_USER | `{body, attachments?}` (PUBLIC_REPLY only) | `201`, history entry for message |
| POST | `/tickets/{id}/close` | Close ticket | END_USER | No payload or `{comment}` optional | Allowed only `RESOLVED` or `PENDING_USER` |

#### Staff Ticket APIs

| Method | Path | Purpose | Auth Role | Validation |
|--------|------|---------|-----------|-----------|
| GET | `/staff/tickets` | List/filter tickets | AGENT+ | Enforce scope filters; supports `department_id`, `team_id`, `assignee_staff_id`, `status`, `priority`, `created_from`, `created_to`, `updated_from`, `updated_to`, `search`, `tags[]` |
| GET | `/staff/tickets/{id}` | View ticket details | AGENT+ | Returns full ticket, all messages, attachments, history |
| POST | `/staff/tickets/{id}/assign/self` | Self-assign | AGENT+ | Fails if ticket outside scope or already assigned to other active staff |
| POST | `/staff/tickets/{id}/assign` | Assign to staff | TEAM_LEAD+, ADMIN | Body `{assignee_staff_id}`; validates staff active + scoped |
| POST | `/staff/tickets/{id}/assign/team` | Assign to team | TEAM_LEAD+, ADMIN | Body `{team_id}`; updates `team_id` and optional auto-assign |
| POST | `/staff/tickets/{id}/status` | Update status | AGENT+ | Body `{new_status, comment?}`; enforces transition table |
| POST | `/staff/tickets/{id}/priority` | Update priority | AGENT+ | Body `{new_priority}`; logs change |
| POST | `/staff/tickets/{id}/messages` | Staff reply/note | AGENT+ | Body `{body, message_type, attachments?}`; `INTERNAL_NOTE` hidden from users |
| GET | `/staff/tickets/{id}/history` | Full history | AGENT+ | Returns chronological entries with metadata |

#### Attachments

- Attachments are created as part of ticket messages via both user and staff endpoints.  
- Validation: file size <= configured max (default 10MB), MIME whitelist.  
- Storage: only metadata stored; `storage_key` points to object storage handled outside scope.  
- API ensures attachments reference existing `ticket_message_id`; orphan uploads are rejected.

#### Notifications & Webhooks (Admin-only / MVP stub)

| Method | Path | Purpose | Role |
|--------|------|---------|------|
| POST | `/admin/webhooks` | Register outbound webhook endpoint | ADMIN |
| GET | `/admin/webhooks` | List configured webhooks | ADMIN |

Notification events:

- `TicketCreated`: payload includes ticket id, requester, department/team, priority.  
- `StaffPublicReply`: ticket id, staff id, snippet of body.  
- `StatusChanged`: old/new status, actor, timestamp.  
Events pass through `NotificationService` which currently logs structured events and, if webhooks exist, enqueues HTTP POST jobs (best-effort in MVP).

### Validation Rules (examples)

- Emails validated via RFC 5322-compatible regex; stored lowercase.
- Password minimum length 12, must include uppercase/lowercase/numeric.  
- Ticket title 5-200 chars; description up to 10k chars sanitized (strip scripts).  
- Tags limited to 20 entries, each <= 32 chars, alphanumeric + hyphen.  
- Attachment file names <= 255 chars; size limit enforced via metadata.  
- Pagination: `page_size` default 20, max 100.  
- Enum validations: statuses and priorities limited to defined sets; message types validated by role.

### Non-Functional Requirements

- **Pagination**: All list endpoints accept `page`, `page_size`; responses include `meta.total`, `meta.page`, `meta.page_size`.
- **Error Format**: Standard envelope with `error.code`, `error.message`, `error.details`. Example:  
  ```json
  {
    "error": {
      "code": "VALIDATION_FAILED",
      "message": "department_id is inactive",
      "details": { "field": "department_id" }
    }
  }
  ```
- **Logging**: Structured logs (JSON) capturing request id, method, path, status, latency, actor id, role, ticket id (when applicable). Sensitive fields (passwords, tokens, PII content) redacted or hashed.
- **Metrics**: Prometheus-style counters/gauges for request counts, error counts, auth failures, ticket status counts, assignment latency, notification dispatch results.
- **Observability**: `/health/live` returns `200` if process up; `/health/ready` checks DB + Redis; failure details in response `details`.
- **Performance**: Use DB indexes on `tickets(status, priority)`, `tickets(department_id, team_id)`, `tickets(assignee_staff_id)`, `ticket_messages(ticket_id, created_at)`. Query plans reviewed for filtering endpoints.
- **Security**: 
  - Tokens stored in Redis with TTL; logout invalidates token entry.  
  - Password hashes stored using bcrypt with configurable cost (default 12).  
  - Rate limiting for auth endpoints (configurable).  
  - Input sanitization for message bodies/attachments.  
  - End-users restricted to own tickets via repository scoping and service-layer check.
- **Migrations**: Each schema change requires a migration file with up/down scripts; migrations executed before deployment.

### Success Criteria

- **SC-001**: 95% of ticket creations complete < 500ms at P95 latency under nominal load (100 RPS).
- **SC-002**: Authentication failure logs contain zero plaintext passwords or tokens (verified via automated log scan).
- **SC-003**: RBAC scope tests achieve 100% pass rate across roles for all ticket and organization endpoints.
- **SC-004**: Auto-assignment successfully assigns ≥90% of team-tagged tickets to an eligible staff member within 2 seconds.
- **SC-005**: Health/readiness endpoints accurately report dependency outages with <10 seconds detection lag.
- **SC-006**: Notification events emitted for 100% of ticket create, status change, and staff public reply operations (verified via event log).

## Governance Alignment

- Auth is strictly email+password with bcrypt hashing; no alternative auth methods are permitted without constitutional amendment.
- All new features must be added to this spec and linked plans prior to implementation.
- Layered architecture, migration discipline, test coverage, and observability mandates align with the ratified constitution (2025-11-23).
