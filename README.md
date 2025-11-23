# Stash

Simple key-value configuration service. A minimal alternative to Consul KV or etcd for storing and retrieving configuration data.

## Features

- HTTP API for key-value operations
- File-based persistent storage
- TTL caching support
- Hierarchical keys (e.g., `app/config/database`)

## Installation

```bash
go install github.com/umputun/stash@latest
```

Or with Docker:

```bash
docker pull ghcr.io/umputun/stash:latest
```

## Usage

```bash
stash --store=/path/to/stash.db --server.address=:8484 --log.enabled
```

### Command Line Options

| Option | Environment | Default | Description |
|--------|-------------|---------|-------------|
| `-s, --store` | `STASH_STORE` | `stash.db` | Path to storage file |
| `--server.address` | `STASH_SERVER_ADDRESS` | `:8484` | Server listen address |
| `--log.enabled` | `STASH_LOG_ENABLED` | `false` | Enable logging |
| `--log.debug` | `STASH_LOG_DEBUG` | `false` | Debug mode |

## API

### Get value
```
GET /kv/{key}
```

### Set value
```
PUT /kv/{key}
Content-Type: application/json

{"value": ...}
```

### List keys
```
GET /kv?prefix={prefix}
```

### Delete key
```
DELETE /kv/{key}
```

## Docker

```bash
docker run -p 8484:8484 -v /data:/srv/data ghcr.io/umputun/stash \
    --store=/srv/data/stash.db --log.enabled
```

## License

MIT License - see [LICENSE](LICENSE) for details.
