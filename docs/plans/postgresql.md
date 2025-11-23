# PostgreSQL Support

## Overview

Add PostgreSQL support to Stash as an alternative to SQLite, allowing deployment in environments where PostgreSQL is preferred or required (e.g., Kubernetes, cloud deployments with managed databases).

**Key benefits:**
- Better concurrency handling for high-traffic deployments
- Familiar infrastructure for teams already using PostgreSQL
- Cloud-managed database options (RDS, Cloud SQL, etc.)
- No file-based storage concerns in containerized environments

**Design approach:**
- Single `Store` struct handles both databases (not separate implementations)
- Database type auto-detected from connection URL
- SQLite remains default for simplicity
- Backward compatible - existing `--store` flag works unchanged

## Iterative Development Approach

### Iteration 1: CLI & Configuration Updates

- [x] Rename `--store` flag to `--db` in `app/main.go`
  - Update environment variable to `STASH_DB`
  - Update default from `stash.db` to `stash.db` (unchanged)
- [x] Add database URL detection helper in `app/store/`:
  - SQLite: `file:`, `.db`, `.sqlite`, `:memory:`, or plain filename
  - PostgreSQL: `postgres://` or `postgresql://`
- [x] Update `app/main.go` to pass URL to store constructor
- [x] Run existing tests to verify no regressions

### Iteration 2: Store Refactoring

- [x] Rename `SQLite` struct to `Store` in `app/store/sqlite.go`
- [x] Add `dbType` field to `Store` struct (enum: SQLite, Postgres)
- [x] Add `RWLocker` interface for conditional locking:
  ```go
  type RWLocker interface {
      RLock()
      RUnlock()
      Lock()
      Unlock()
  }
  ```
- [x] Create `noopLocker` for PostgreSQL (no-op implementation)
- [x] Replace `sync.RWMutex` with `RWLocker` interface in `Store`
- [x] Update constructor signature: `New(dbURL string) (*Store, error)`
- [x] Keep `NewSQLite` as alias for backward compatibility
- [x] Run tests to verify SQLite still works

### Iteration 3: PostgreSQL Driver & Connection

- [x] Add `github.com/jackc/pgx/v5/stdlib` dependency
- [x] Implement PostgreSQL connection in `New()`:
  - Parse URL to detect database type
  - For SQLite: existing pragmas and single connection
  - For PostgreSQL: standard connection pool
- [x] Handle PostgreSQL-specific initialization:
  - No pragmas needed
  - Default connection pool settings
- [x] Create schema with PostgreSQL syntax variant:
  ```sql
  -- SQLite
  CREATE TABLE IF NOT EXISTS kv (
      key TEXT PRIMARY KEY,
      value BLOB NOT NULL,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
  )

  -- PostgreSQL
  CREATE TABLE IF NOT EXISTS kv (
      key TEXT PRIMARY KEY,
      value BYTEA NOT NULL,
      created_at TIMESTAMP DEFAULT NOW(),
      updated_at TIMESTAMP DEFAULT NOW()
  )
  ```
- [x] Write test for PostgreSQL connection (skip if no test DB)

### Iteration 4: Query Adaptation

- [x] Create placeholder converter: `?` → `$1, $2, ...`
- [x] Update `Get()` method:
  - Use `$1` placeholder for PostgreSQL
- [x] Update `Set()` method:
  - SQLite: `INSERT ... ON CONFLICT(key) DO UPDATE SET ...`
  - PostgreSQL: Same syntax works (both support it)
  - Use `$1, $2, $3, $4` placeholders for PostgreSQL
- [x] Update `Delete()` method:
  - Use `$1` placeholder for PostgreSQL
- [x] Update `List()` method:
  - `length(value)` works in both (or use `octet_length` for PostgreSQL)
- [x] Run SQLite tests to verify no regressions

### Iteration 5: Testing with Testcontainers

- [x] Add `github.com/go-pkgz/testutils` dependency
- [x] Add PostgreSQL integration tests in `app/store/sqlite_test.go`:
  - Use `containers.NewPostgresTestContainerWithDB()` for automatic container
  - Test all CRUD operations
- [x] Example test setup:
  ```go
  func TestStore_Postgres(t *testing.T) {
      ctx := context.Background()
      pgContainer := containers.NewPostgresTestContainerWithDB(ctx, t, "stash_test")
      defer pgContainer.Close(ctx)

      store, err := New(pgContainer.ConnectionString())
      require.NoError(t, err)
      defer store.Close()

      // run tests...
  }
  ```
- [x] Added `TestDetectDBType` and `TestAdoptQuery` unit tests

### Iteration 6: Documentation

- [x] Update README.md:
  - Document `--db` flag
  - Add PostgreSQL usage examples
  - Add Docker Compose example with PostgreSQL
- [x] Update CLAUDE.md with PostgreSQL notes

---

## Technical Details

### Database Type Detection

```go
func detectDBType(url string) DBType {
    url = strings.ToLower(url)
    if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
        return DBTypePostgres
    }
    return DBTypeSQLite
}
```

### Connection URLs

| Database | URL Format |
|----------|------------|
| SQLite (file) | `stash.db`, `./data/stash.db`, `file:stash.db` |
| SQLite (memory) | `:memory:` |
| PostgreSQL | `postgres://user:pass@host:5432/dbname?sslmode=disable` |

### Query Differences

| Operation | SQLite | PostgreSQL |
|-----------|--------|------------|
| Placeholder | `?` | `$1, $2, ...` |
| BLOB type | `BLOB` | `BYTEA` |
| Timestamp | `DATETIME DEFAULT CURRENT_TIMESTAMP` | `TIMESTAMP DEFAULT NOW()` |
| String length | `length(value)` | `length(value)` or `octet_length(value)` |
| Upsert | `ON CONFLICT(key) DO UPDATE` | Same (PostgreSQL 9.5+) |

### Locking Strategy

| Database | Locking | Reason |
|----------|---------|--------|
| SQLite | `sync.RWMutex` | Single writer, WAL mode |
| PostgreSQL | No-op | MVCC handles concurrency |

### Example Usage

```bash
# SQLite (default)
stash --db stash.db

# SQLite with explicit path
stash --db file:./data/stash.db

# PostgreSQL
stash --db "postgres://stash:secret@localhost:5432/stash?sslmode=disable"

# PostgreSQL with SSL
stash --db "postgres://stash:secret@db.example.com:5432/stash?sslmode=require"
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  stash:
    image: ghcr.io/umputun/stash
    environment:
      - STASH_DB=postgres://stash:secret@postgres:5432/stash?sslmode=disable
    depends_on:
      - postgres
    ports:
      - "8484:8484"

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=stash
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=stash
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

---

## Files to Create/Modify

**Modified files:**
- `app/main.go` - CLI flag rename, URL passing
- `app/store/sqlite.go` → `app/store/store.go` - unified store
- `app/store/store_test.go` - PostgreSQL tests
- `go.mod` - pgx dependency
- `README.md` - documentation
- `CLAUDE.md` - project notes

**New files:**
- `docker-compose.postgres.yml` - example with PostgreSQL (optional)

---

## Out of Scope

- Database migrations between SQLite and PostgreSQL
- Connection pooling configuration options
- Read replicas support
- Multi-database support (single DB per instance)
