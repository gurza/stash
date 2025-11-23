package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	_ "github.com/jackc/pgx/v5/stdlib" // postgresql driver
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // sqlite driver
)

// Store implements key-value storage using SQLite or PostgreSQL.
type Store struct {
	db     *sqlx.DB
	dbType DBType
	mu     RWLocker
}

// New creates a new Store with the given database URL.
// Automatically detects database type from URL:
// - postgres:// or postgresql:// -> PostgreSQL
// - everything else -> SQLite
func New(dbURL string) (*Store, error) {
	dbType := detectDBType(dbURL)

	var db *sqlx.DB
	var err error
	var locker RWLocker

	switch dbType {
	case DBTypePostgres:
		db, err = connectPostgres(dbURL)
		locker = noopLocker{}
	default:
		db, err = connectSQLite(dbURL)
		locker = &sync.RWMutex{}
	}

	if err != nil {
		return nil, err
	}

	s := &Store{db: db, dbType: dbType, mu: locker}

	if err := s.createSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	log.Printf("[DEBUG] initialized %s store", s.dbTypeName())
	return s, nil
}

// NewSQLite creates a new SQLite store (backward compatibility).
func NewSQLite(dbPath string) (*Store, error) {
	return New(dbPath)
}

// detectDBType determines database type from URL.
func detectDBType(url string) DBType {
	lower := strings.ToLower(url)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return DBTypePostgres
	}
	return DBTypeSQLite
}

// connectSQLite establishes SQLite connection with pragmas.
func connectSQLite(dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite: %w", err)
	}

	// set pragmas for performance and reliability
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=1000",
		"PRAGMA foreign_keys=ON",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	// limit connections for SQLite (single writer)
	db.SetMaxOpenConns(1)

	return db, nil
}

// connectPostgres establishes PostgreSQL connection.
func connectPostgres(dbURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// set reasonable connection pool defaults
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// createSchema creates the kv table if it doesn't exist.
func (s *Store) createSchema() error {
	var schema string
	switch s.dbType {
	case DBTypePostgres:
		schema = `
			CREATE TABLE IF NOT EXISTS kv (
				key TEXT PRIMARY KEY,
				value BYTEA NOT NULL,
				created_at TIMESTAMP DEFAULT NOW(),
				updated_at TIMESTAMP DEFAULT NOW()
			)`
	default:
		schema = `
			CREATE TABLE IF NOT EXISTS kv (
				key TEXT PRIMARY KEY,
				value BLOB NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`
	}
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}

// dbTypeName returns human-readable database type name.
func (s *Store) dbTypeName() string {
	switch s.dbType {
	case DBTypePostgres:
		return "postgres"
	default:
		return "sqlite"
	}
}

// Get retrieves the value for the given key.
// Returns ErrNotFound if the key does not exist.
func (s *Store) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value []byte
	query := s.adoptQuery("SELECT value FROM kv WHERE key = ?")
	err := s.db.Get(&value, query, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %q: %w", key, err)
	}
	return value, nil
}

// Set stores the value for the given key.
// Creates a new key or updates an existing one.
func (s *Store) Set(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	var query string
	switch s.dbType {
	case DBTypePostgres:
		query = `
			INSERT INTO kv (key, value, created_at, updated_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT(key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`
	default:
		query = `
			INSERT INTO kv (key, value, created_at, updated_at) VALUES (?, ?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`
	}
	if _, err := s.db.Exec(query, key, value, now, now); err != nil {
		return fmt.Errorf("failed to set key %q: %w", key, err)
	}
	return nil
}

// Delete removes the key from the store.
// Returns ErrNotFound if the key does not exist.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := s.adoptQuery("DELETE FROM kv WHERE key = ?")
	result, err := s.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %q: %w", key, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns metadata for all keys, ordered by updated_at descending.
func (s *Store) List() ([]KeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []KeyInfo
	var query string
	switch s.dbType {
	case DBTypePostgres:
		query = `SELECT key, octet_length(value) as size, created_at, updated_at FROM kv ORDER BY updated_at DESC`
	default:
		query = `SELECT key, length(value) as size, created_at, updated_at FROM kv ORDER BY updated_at DESC`
	}
	if err := s.db.Select(&keys, query); err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	return keys, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// adoptQuery converts SQLite placeholders (?) to PostgreSQL ($1, $2, ...).
func (s *Store) adoptQuery(query string) string {
	if s.dbType != DBTypePostgres {
		return query
	}

	result := make([]byte, 0, len(query)+10)
	paramNum := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			result = append(result, '$')
			result = append(result, fmt.Sprintf("%d", paramNum)...)
			paramNum++
		} else {
			result = append(result, query[i])
		}
	}
	return string(result)
}
