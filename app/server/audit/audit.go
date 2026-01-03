// Package audit provides HTTP middleware for audit logging and query handler for audit log access.
package audit

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/umputun/stash/app/store"
)

//go:generate moq -out mocks/store.go -pkg mocks -skip-ensure -fmt goimports . Store
//go:generate moq -out mocks/auth.go -pkg mocks -skip-ensure -fmt goimports . Auth

// Store defines the interface for audit log storage.
type Store interface {
	LogAudit(ctx context.Context, entry store.AuditEntry) error
	QueryAudit(ctx context.Context, q store.AuditQuery) ([]store.AuditEntry, int, error)
}

// Auth defines the interface for auth operations needed by audit.
type Auth interface {
	GetRequestActor(r *http.Request) (actorType, actorName string)
	IsRequestAdmin(r *http.Request) bool
}

// responseCapture wraps http.ResponseWriter to capture status code and bytes written.
type responseCapture struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

// newResponseCapture creates a responseCapture wrapper.
func newResponseCapture(w http.ResponseWriter) *responseCapture {
	return &responseCapture{ResponseWriter: w, status: http.StatusOK}
}

// WriteHeader captures the status code and delegates to wrapped writer.
func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

// Write captures bytes written and delegates to wrapped writer.
func (rc *responseCapture) Write(b []byte) (int, error) {
	n, err := rc.ResponseWriter.Write(b)
	rc.bytesWritten += n
	if err != nil {
		return n, fmt.Errorf("write: %w", err)
	}
	return n, nil
}

// Unwrap returns the underlying ResponseWriter (for http.ResponseController).
func (rc *responseCapture) Unwrap() http.ResponseWriter {
	return rc.ResponseWriter
}

// Flush implements http.Flusher.
func (rc *responseCapture) Flush() {
	if f, ok := rc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker.
func (rc *responseCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rc.ResponseWriter.(http.Hijacker); ok {
		conn, rw, err := hj.Hijack()
		if err != nil {
			return nil, nil, fmt.Errorf("hijack: %w", err)
		}
		return conn, rw, nil
	}
	return nil, nil, errors.New("ResponseWriter does not implement http.Hijacker")
}

// Middleware creates middleware that logs audit entries after handler completes.
// This is a convenience function that creates a logger and returns its middleware.
func Middleware(auditStore Store, authProvider Auth) func(http.Handler) http.Handler {
	l := newLogger(auditStore, authProvider)
	return l.middleware
}

// NoopMiddleware returns a pass-through middleware (used when audit is disabled).
func NoopMiddleware(next http.Handler) http.Handler {
	return next
}
