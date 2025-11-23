# Ticket Service Constitution

## Core Principles

### I. Auth Simplicity, Trust, and Accountability
The only authentication path is email plus password backed by bcrypt (or a comparable battle-tested hashing algorithm); no feature or experiment may add OAuth, SSO, or social auth. Passwords, reset tokens, and session secrets are never logged and must be stored only after hashing, with authenticated routes always enforcing explicit role checks before continuing.

### II. Layered Hexagonal Architecture
All code follows a strict transport → service → repository layering: HTTP handlers in Fiber own routing, validation, DTOs, and JSON encoding; services hold business rules independent of Fiber or storage; repositories encapsulate PostgreSQL and Redis access. Domain models stay pure and are never coupled to DB schemas or HTTP payloads.

### III. Domain Ownership & Permissions
User, StaffMember, Department, Team, Ticket, TicketMessage, TicketHistory, and AttachmentReference are first-class domain entities with authoritative definitions. Roles END_USER, AGENT, TEAM_LEAD, and ADMIN have documented permissions, and guardrails ensure end-users only read or mutate their own tickets while staff access is limited by department/team context.

### IV. Quality as a Gatekeeper
Every change must uphold idiomatic Go: context-aware functions, explicit errors, and small focused methods. Database migrations are source-controlled artifacts required for schema changes. Services and auth logic need unit tests; repositories and critical HTTP flows require integration tests, all of which must pass in CI before release.

### V. Observability and Operational Readiness
Structured logs capture every request, auth event, and material ticket lifecycle change without leaking secrets. We emit metrics for request rates, error counts, and ticket status transitions, and maintain health/readiness endpoints that actively verify PostgreSQL and Redis connectivity before advertising availability.

## Architecture & Platform Directives

- **Stack Lock-in**: HTTP APIs are written in Go using Fiber. PostgreSQL is the primary relational store and the source of truth; Redis provides caching plus token/session storage only. No frontend is shipped—this is a REST/JSON backend exclusively.
- **Configuration Discipline**: All settings are supplied via environment variables with documented safe defaults. Secrets never ship in source control and must be overridable for tests.
- **Data Boundaries**: Domain models, DB schemas, and DTOs remain separate types. Mappers convert between layers to preserve invariants and avoid leaking persistence or transport concerns into the core logic.
- **Ticket Lifecycle**: Tickets honor canonical statuses (OPEN, IN_PROGRESS, PENDING_USER, RESOLVED, CLOSED, CANCELLED) and priorities (LOW, MEDIUM, HIGH, URGENT). History and message trails are append-only for forensic clarity.
- **Caching Rules**: Redis caches only non-sensitive aggregates and ephemeral tokens. Any cached data must respect TTLs and be invalidated on writes touching affected entities.

## Delivery Workflow & Quality Gates

- **Planning First**: Any new feature or change must be drafted in the spec and delivery plan before implementation, explicitly referencing impacted entities, roles, and APIs.
- **Testing Matrix**: Unit tests target service-layer business logic and authentication flows; integration tests cover repository adapters and the most important HTTP endpoints (e.g., ticket creation, assignment, resolution). Failing or absent tests block merges.
- **API Consistency**: All handlers share a consistent JSON envelope for success and errors, including traceable error codes. Validation failures report structured fields and never expose raw internal errors.
- **Security Reviews**: Input validation and sanitization are mandatory. Role enforcement is reviewed per route, ensuring end-users can reach only their tickets and staff access respects departmental scoping. Sensitive values (passwords, tokens, personally identifiable content) are redacted from logs and metrics.
- **Operational Checks**: Health/readiness endpoints must cover DB and Redis reachability. Metrics dashboards and log pipelines are part of the Definition of Done for any major feature touching ticket processing.

## Governance
This constitution overrides conflicting practices. Amendments require a documented proposal detailing architectural, security, and testing impacts plus a migration plan for existing data or services. All pull requests must cite the relevant constitutional clauses they satisfy, and reviewers are accountable for rejecting work that bypasses email+password auth, role checks, migrations, or the layered architecture.

**Version**: 1.0.0 | **Ratified**: 2025-11-23 | **Last Amended**: 2025-11-23
