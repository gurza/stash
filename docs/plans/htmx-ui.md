# HTMX UI Implementation

## Overview

Add a web UI using HTMX for managing key-value pairs. Features:
- Table view listing all keys with metadata (size, timestamps) - values shown on-demand via modal
- Full CRUD: create, view, edit, delete keys
- Light/dark theme with system preference detection
- Binary-safe value display (base64 for non-UTF-8)
- Minimal JavaScript, server-driven UI

## Context (from discovery)

**Files involved:**
- `app/server/server.go` - add web routes and template rendering
- `app/server/web.go` - new file for web handlers
- `app/server/static/` - CSS, JS, HTMX library
- `app/server/templates/` - HTML templates
- `app/store/sqlite.go` - need List() method for keys

**Patterns from cronn:**
- `//go:embed` for templates and static files
- HTMX v2 with polling, OOB swaps, modals
- CSS variables for theming (`data-theme` attribute)
- Cookie-based theme preference
- Custom confirm dialog for delete

**New store method needed:**
- `List() ([]KeyInfo, error)` - returns all keys with metadata, ordered by updated_at DESC

## Iterative Development Approach

- Complete each iteration fully before moving to the next
- Write tests alongside implementation
- Run tests and linter after each change

## Progress Tracking

- Mark completed items with `[x]`
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix

## Implementation Steps

### Iteration 1: Store List Method [COMPLETED]

- [x] Add `KeyInfo` struct to `app/store/store.go` (key, size, created_at, updated_at)
- [x] Add `List() ([]KeyInfo, error)` method to `app/store/sqlite.go`
- [x] Add tests for List in `app/store/sqlite_test.go`
- [x] Run tests to verify

### Iteration 2: Static Files & Templates Structure [COMPLETED]

- [x] Create `app/server/static/` directory
- [x] Add `htmx.min.js` (HTMX v2)
- [x] Create `style.css` with CSS variables for light/dark theme
- [x] Create `app.js` for minimal JS (confirm dialog, theme toggle, modal)
- [x] Create `app/server/templates/` directory
- [x] Create `base.html` layout template
- [x] Create `index.html` main page template with search
- [x] Create `partials/` directory for HTMX fragments
- [x] Add `//go:embed` directives in server package
- [x] Verify embedded files compile

### Iteration 3: Web Handlers - List & View [COMPLETED]

- [x] Create `app/server/web.go` with web handlers
- [x] Update `KVStore` interface: add `List() ([]store.KeyInfo, error)`
- [x] Implement `handleIndex` - render main page with key list
- [x] Implement `handleKeyList` - HTMX partial for table refresh with search
- [x] Implement `handleKeyView` - modal with full value (handle binary as base64)
- [x] Add routes: `GET /`, `GET /web/keys`, `GET /web/keys/view/{key...}`
- [x] Add static file serving at `/static/`
- [x] Wire up in `routes()` method

### Iteration 4: Web Handlers - Create, Edit, Delete [COMPLETED]

- [x] Create `partials/form.html` for create/edit modal
- [x] Implement `handleKeyCreate` - POST new key
- [x] Implement `handleKeyEdit` - render edit form
- [x] Implement `handleKeyUpdate` - PUT update key
- [x] Implement `handleKeyDelete` - DELETE with confirm
- [x] Add routes: `POST /web/keys`, `PUT /web/keys/{key...}`, `DELETE /web/keys/{key...}`
- [x] Implement custom confirm dialog in JS

### Iteration 5: Theme Support [COMPLETED]

- [x] Implement `handleThemeToggle` - POST toggle theme
- [x] Add cookie-based theme persistence
- [x] Add system preference detection in CSS
- [x] Add theme toggle button to UI
- [x] Test theme switching

### Iteration 6: Web Tests [COMPLETED]

Tests in `app/server/web_test.go`:
- [x] `TestHandleIndex` - index page renders with key list
- [x] `TestHandleKeyList` - partial returns correct HTML, search filtering works
- [x] `TestHandleKeyCreate` - create new key via form POST
- [x] `TestHandleKeyView` - view modal renders key/value, handles not found
- [x] `TestHandleKeyEdit` - edit form renders with existing value
- [x] `TestHandleKeyUpdate` - update key value via PUT
- [x] `TestHandleKeyDelete` - delete key, handles not found
- [x] `TestHandleThemeToggle` - sets theme cookie, returns HX-Refresh header
- [x] `TestValueForDisplay` - UTF-8 passthrough, binary base64 encoding
- [x] `TestValueFromForm` - text/binary form value decoding
- [x] Run full test suite with race detector
- [x] Manual Playwright testing of all UI features

### Iteration 7: Documentation & Cleanup [IN PROGRESS]

- [ ] Update README.md with UI section
- [ ] Update CLAUDE.md with web package info
- [x] Run linter and fix issues
- [x] Final manual testing

## Technical Details

### KeyInfo Struct

```go
type KeyInfo struct {
    Key       string    `db:"key"`
    Size      int       `db:"size"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

### Template Data

```go
type templateData struct {
    Keys     []store.KeyInfo
    Key      string
    Value    string   // text or base64 depending on IsBinary
    IsBinary bool     // true if value is not valid UTF-8
    IsNew    bool     // true when rendering new key form
    Theme    string   // "light", "dark", or "" (system preference)
    ViewMode string   // "grid" or "cards"
    Search   string   // current search term for highlighting
    Error    string
}
```

### Routes Summary

| Method | Path | Description |
|--------|------|-------------|
| GET | / | Main page with key list |
| GET | /web/keys | HTMX partial: key table (supports ?search=) |
| GET | /web/keys/new | HTMX partial: new key form |
| GET | /web/keys/view/{key...} | HTMX partial: view modal |
| GET | /web/keys/edit/{key...} | HTMX partial: edit form |
| POST | /web/keys | Create new key (form submit) |
| PUT | /web/keys/{key...} | Update key value |
| DELETE | /web/keys/{key...} | Delete key |
| POST | /web/theme | Toggle theme |
| POST | /web/view-mode | Toggle view mode (grid/cards) |

### Existing API (unchanged)

| Method | Path | Description |
|--------|------|-------------|
| GET | /kv/{key...} | Get value |
| PUT | /kv/{key...} | Set value |
| DELETE | /kv/{key...} | Delete key |
| GET | /ping | Health check |

### HTMX Patterns to Use

1. **Refresh after action**: table reloads after create/edit/delete via `hx-swap-oob`
2. **Search with debounce**: `hx-trigger="input changed delay:300ms"`
3. **Modal loading**: `hx-target="#modal-content" hx-on::after-request="showModal()"`
4. **Confirm delete**: custom JS confirm dialog
5. **Theme toggle**: `HX-Refresh: true` header for full page reload

### Binary Value Handling

- Check if value is valid UTF-8 using `utf8.Valid()`
- UTF-8 values: display as text in view/edit
- Non-UTF-8 values: display as base64 with indicator, allow base64 edit

### Key Encoding

- Keys with special characters (spaces, slashes, etc.) are URL-encoded by browser
- Go's `http.PathValue` returns URL-decoded values automatically
- `html/template` escapes values in attributes (XSS protection)

### CSS Theme Variables

```css
:root {
    --color-bg: #ffffff;
    --color-surface: #f5f5f5;
    --color-text: #1a1a1a;
    --color-primary: #2563eb;
    --color-danger: #dc2626;
}

[data-theme="dark"] {
    --color-bg: #1a1a1a;
    --color-surface: #2a2a2a;
    --color-text: #f5f5f5;
}
```
