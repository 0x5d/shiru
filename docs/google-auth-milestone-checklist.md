## Google Auth + Per-User Isolation Checklist

This checklist defines implementation-ready milestones for adding Google login and enforcing user isolation for settings, stories, vocab, and related resources.

### Milestone 0: Lock Design

- Add `docs/google-auth.md` with final design choices:
  - Google ID token from frontend
  - Backend verification of ID token
  - Signed HTTP-only session cookie
  - Protected `/api/v1/*` endpoints except auth endpoints
  - User-scoped data access in repositories

Acceptance criteria:

- Design doc includes endpoint contracts, cookie policy, and env vars.

### Milestone 1: User Schema

- Add migration `internal/postgres/migrations/003_google_auth.sql`:
  - `users.google_sub TEXT`
  - `users.email TEXT`
  - `users.name TEXT`
  - `users.avatar_url TEXT`
  - `users.last_login_at TIMESTAMPTZ`
  - unique index on `google_sub` (nullable-safe)
- Add repository file `internal/postgres/users.go`:
  - `UpsertGoogleUser(...)`
  - `GetByID(...)`
  - `EnsureUserSettings(...)`

Acceptance criteria:

- Repeated login with same Google `sub` maps to same user ID.

### Milestone 2: Auth Config + Session

- Extend `internal/config/config.go` with:
  - `GOOGLE_CLIENT_ID`
  - `SESSION_SECRET`
  - `SESSION_COOKIE_NAME` (default `shiru_session`)
  - `SESSION_TTL` (duration)
- Add `internal/auth` package:
  - Google token verifier
  - signed session encode/decode
  - expiry validation

Acceptance criteria:

- Unit tests pass for valid, expired, and tampered sessions.

### Milestone 3: Auth Endpoints

- Add `internal/api/auth.go` with routes:
  - `POST /api/v1/auth/google`
  - `GET /api/v1/auth/me`
  - `POST /api/v1/auth/logout`
- Wire routes in `internal/api/server.go`.
- Wire dependencies in `main.go`.

Acceptance criteria:

- Login sets session cookie.
- `/auth/me` returns user when authenticated.
- Logout clears cookie.

### Milestone 4: Auth Middleware

- Add middleware to read/validate session cookie.
- Add context helpers for current user ID.
- Apply middleware to all business endpoints.
- Replace `domain.DefaultUserID` usage in API handlers with context user ID.

Acceptance criteria:

- Unauthenticated requests to protected endpoints return `401`.

### Milestone 5: Repository Ownership Enforcement

- Update repository methods to require `user_id` in queries that fetch/update by resource ID:
  - stories `Get`
  - vocab `GetByID`, `UpdateDetails`
  - audio flow validates story ownership first
- Regenerate mocks with `go generate ./...`.

Acceptance criteria:

- Cross-user resource access by guessed UUID is blocked.

### Milestone 6: Frontend Login Flow

- Add Google GIS integration in frontend.
- Add auth API helpers and include `credentials: 'include'` in fetches.
- Add login page and auth guard in router.
- Add logout control.

Acceptance criteria:

- User must log in before accessing app pages.

### Milestone 7: Data Backfill

- Add one-time script for moving default-user data to a real Google user.
- Include runbook and rollback notes in docs.

Acceptance criteria:

- Existing seed-user data can be reassigned safely.

### Milestone 8: Tests + Verification

- Add/extend tests for:
  - auth endpoints
  - middleware auth enforcement
  - cross-user access denial
  - cookie tampering/expiry
- Run checks:
  - `go generate ./...`
  - `go test -timeout 5m ./...`
  - `golangci-lint run`

Acceptance criteria:

- All checks pass and auth/tenant isolation tests are green.

### Milestone 9: Deployment + Secrets

- Document required env vars in deployment docs and compose setup.
- Add frontend env for Google client ID.

Acceptance criteria:

- Local and deployed environments can run auth flow with documented configuration.
