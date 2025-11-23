# Stash Project

Simple key-value configuration service - a minimal alternative to Consul KV or etcd.

## Project Structure

- **app/main.go** - Entry point with CLI options, logging, signal handling
- **app/server/** - HTTP server (TODO)
- **app/store/** - Storage layer (TODO)

## Key Dependencies

- **CLI**: `github.com/umputun/go-flags`
- **Logging**: `github.com/go-pkgz/lgr`
- **HTTP**: `github.com/go-pkgz/routegroup` (planned)
- **Testing**: `github.com/stretchr/testify`

## Build & Test

```bash
make build    # build binary
make test     # run tests
make lint     # run linter
make run      # run with logging enabled
```

## API Design (Planned)

```
GET    /kv/{key}           # get value
PUT    /kv/{key}           # set value
DELETE /kv/{key}           # delete key
GET    /kv?prefix={prefix} # list keys by prefix
```

## Development Notes

- Follow patterns from cronn project
- Use consumer-side interfaces
- Return concrete types, accept interfaces
- Keep it simple - no over-engineering
