# Authentication System

## Overview

Add optional authentication to Stash, inspired by cronn's simple approach. When enabled via `--password-hash`, all routes require authentication. The system supports:

- **UI authentication**: Password-based form login with session cookies
- **API authentication**: Basic auth (full access) or Bearer tokens (prefix-scoped)
- **Backward compatible**: No `--password-hash` = no auth (current behavior)

## Key Features

- Single password for admin access (UI + Basic auth)
- Optional API tokens with prefix-based read/write permissions
- Session management with secure cookies
- Rate limiting on login attempts
- No database changes required (tokens defined via CLI)

## Configuration

```bash
# No auth (dev mode, current behavior)
stash

# Auth enabled with password
stash --password-hash '$2a$10$...'

# Auth with API tokens for scoped access
stash --password-hash '$2a$10$...' \
      --auth-token "apptoken:app/*:rw" \
      --auth-token "monitor:*:r"
```

**Token format**: `token:prefix:permissions` where:
- `token` - the secret token string
- `prefix` - key prefix pattern (`*` = all, `app/*` = app/ prefix)
- `permissions` - `r` (read), `w` (write), or `rw` (both)

## Access Matrix

| Method | Credential | Scope |
|--------|-----------|-------|
| UI form login | password | full access (cookie session) |
| API Bearer token | token | prefix-scoped (or `*:rw` for full) |

---

## Implementation Plan

**Status: COMPLETED** (2025-11-23)

### Iteration 1: CLI Options & Data Structures ✓

- [x] Add auth options to `app/main.go`:
  - `--auth.password-hash` / `STASH_AUTH_PASSWORD_HASH` - bcrypt hash for admin password
  - `--auth.token` / `STASH_AUTH_AUTH_TOKEN` - repeatable flag for API tokens
  - `--auth.login-ttl` / `STASH_AUTH_LOGIN_TTL` - session duration in minutes (default: 1440)
- [x] Create `app/server/auth.go` with types:
  - `Permission` type (None, Read, Write, ReadWrite)
  - `TokenACL` struct (token, prefixes sorted by length)
  - `session` struct (token, createdAt)
  - `Auth` struct encapsulating all auth state
- [x] Update `Config` struct and `New()` to accept auth settings
- [x] Implement token parser: parse `"token:prefix:rw"` format
  - Validate format, error on malformed input
  - Error on duplicate token+prefix combinations
  - Store prefixes sorted by length (longest first) for deterministic matching
- [x] Write tests for token parser

### Iteration 2: Session Management ✓

- [x] Implement `CreateSession()` - generate secure random token (32 bytes hex encoded)
- [x] Implement `ValidateSession(token)` - check token exists and not expired
- [x] Implement `InvalidateSession(token)` - remove session
- [x] Implement `cleanupExpiredSessions()` - cleanup on session creation
- [x] Write tests for session management

### Iteration 3: Auth Middleware ✓

- [x] Create `SessionAuth` middleware for web UI routes:
  - Check session cookie -> allow (full access)
  - Else redirect to `/login`
- [x] Create `TokenAuth` middleware for API routes:
  - Check session cookie -> allow (full access for UI calling API)
  - Check Bearer token -> allow (scoped access)
  - Else 401 Unauthorized
- [x] Create `NoopAuth` pass-through middleware (when auth disabled)
- [x] Create helper `CheckPermission(token, key, needWrite)`:
  - Match key against token's prefix patterns (longest-prefix-wins)
  - Check if operation (read/write) is allowed
- [x] Add middleware to route groups conditionally
- [x] Write tests for middleware logic

### Iteration 4: Login/Logout Handlers ✓

- [x] Create login template `templates/login.html`:
  - Simple password form
  - Error message display
  - Theme support (consistent with main UI)
- [x] Implement `handleLoginForm()` - render login page
- [x] Implement `handleLogin()`:
  - Parse form, validate password with bcrypt
  - Create session, set secure cookie (HttpOnly, SameSite=Lax)
  - Redirect to `/`
- [x] Implement `handleLogout()`:
  - Invalidate session, clear cookie
  - Redirect to `/login`
- [x] Register routes: `GET /login`, `POST /login`, `POST /logout`
- [x] Write tests for login/logout handlers

### Iteration 5: API Token Authorization ✓

- [x] TokenAuth middleware checks permissions per operation:
  - GET requests require read permission for key prefix
  - PUT/DELETE requests require write permission for key prefix
- [x] Return 403 Forbidden when token lacks permission
- [x] Session cookie grants full access for web UI calling API
- [x] Write tests for prefix matching and permission checks

### Iteration 6: UI Updates ✓

- [x] Add logout button to header (when auth enabled)
- [x] Add login page styles (`.login-container`, `.login-box`, etc.)
- [x] Test UI flow: login -> use -> logout

### Iteration 7: Documentation & Testing ✓

- [x] Update README.md:
  - Document `--auth.password-hash` option
  - Document `--auth.token` format
  - Add examples for generating bcrypt hash
  - Document API authentication methods
- [x] Update CLAUDE.md with auth section
- [x] Integration tests for full auth flow (`TestIntegration_WithAuth`)
- [x] Manual testing completed:
  - [x] No auth mode works as before
  - [x] Password login works
  - [x] Bearer token with full access works
  - [x] Bearer token with read-only works
  - [x] Bearer token with prefix restriction works
  - [x] Logout clears session

### Not Implemented (deferred)

- Rate limiting on login endpoint (future improvement)
- Basic auth support (Bearer tokens only)

---

## Technical Details

### Data Structures

```go
// Permission represents read/write access level
type Permission int

const (
    PermissionNone Permission = iota
    PermissionRead
    PermissionWrite
    PermissionReadWrite
)

// TokenACL defines access control for an API token
type TokenACL struct {
    Token    string
    Prefixes map[string]Permission // "app/*" -> ReadWrite
}

// session represents an active login session
type session struct {
    token     string
    createdAt time.Time
}
```

### Prefix Matching Rules

**Precedence**: Longest prefix wins (most specific match). Prefixes are sorted by length descending before matching.

| Pattern | Key | Match |
|---------|-----|-------|
| `*` | any key | yes |
| `app/*` | `app/config` | yes |
| `app/*` | `app/db/host` | yes |
| `app/*` | `other/key` | no |
| `app/config` | `app/config` | yes (exact) |
| `app/config` | `app/config/sub` | no |

**Multiple prefixes example**: Token with `*:r` and `app/*:rw` accessing `app/config`:
- `app/*` matches (length 5) - wins over `*` (length 1)
- Result: read-write access

**Duplicate handling**: Error at startup if same token+prefix defined twice via multiple `--auth-token` flags.

### Cookie Configuration

```go
http.Cookie{
    Name:     "stash-auth",        // or "__Host-stash-auth" for HTTPS
    Value:    sessionToken,
    Path:     "/",
    MaxAge:   86400,               // 24 hours
    HttpOnly: true,
    SameSite: http.SameSiteStrictMode,
    Secure:   isHTTPS,
}
```

### Error Responses

| Scenario | Browser | API |
|----------|---------|-----|
| No auth | redirect `/login` | 401 Unauthorized |
| Invalid password | re-render form with error | N/A (form only) |
| Invalid token | N/A | 401 Unauthorized |
| No permission for prefix | 403 page | 403 Forbidden |

---

## Files to Create/Modify

**New files:**
- `app/server/auth.go` - auth logic, middleware, session management
- `app/server/auth_test.go` - tests
- `app/server/templates/login.html` - login form template

**Modified files:**
- `app/main.go` - CLI options, pass auth config to server
- `app/server/server.go` - Server struct fields, middleware setup
- `app/server/web.go` - logout button in templates
- `app/server/templates/base.html` - logout link in header
- `README.md` - documentation

---

## Out of Scope (for now)

- Multiple admin users
- Token management UI (create/delete tokens in UI)
- Token storage in database
- Password change UI
- OAuth/OIDC integration
