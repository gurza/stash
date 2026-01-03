package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/stash/app/enum"
	"github.com/umputun/stash/app/server/audit/mocks"
)

func TestLogger_MapAction(t *testing.T) {
	l := newLogger(nil, nil)

	tests := []struct {
		method string
		status int
		want   enum.AuditAction
	}{
		{http.MethodGet, http.StatusOK, enum.AuditActionRead},
		{http.MethodGet, http.StatusNotFound, enum.AuditActionRead},
		{http.MethodPut, http.StatusOK, enum.AuditActionUpdate},
		{http.MethodPut, http.StatusCreated, enum.AuditActionCreate},
		{http.MethodDelete, http.StatusNoContent, enum.AuditActionDelete},
		{http.MethodPost, http.StatusOK, enum.AuditActionRead}, // fallback
	}

	for _, tt := range tests {
		name := tt.method + "_" + http.StatusText(tt.status)
		t.Run(name, func(t *testing.T) {
			got := l.mapAction(tt.method, tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogger_MapStatus(t *testing.T) {
	l := newLogger(nil, nil)

	tests := []struct {
		status int
		want   enum.AuditResult
	}{
		{http.StatusOK, enum.AuditResultSuccess},
		{http.StatusCreated, enum.AuditResultSuccess},
		{http.StatusNoContent, enum.AuditResultSuccess},
		{http.StatusNotFound, enum.AuditResultNotFound},
		{http.StatusForbidden, enum.AuditResultDenied},
		{http.StatusUnauthorized, enum.AuditResultDenied},
		{http.StatusBadRequest, enum.AuditResultNotFound},
		{http.StatusInternalServerError, enum.AuditResultNotFound},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			got := l.mapStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogger_ExtractActor(t *testing.T) {
	t.Run("nil auth returns anonymous", func(t *testing.T) {
		l := newLogger(nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/kv/test", http.NoBody)
		actor, actorType := l.extractActor(req)
		assert.Equal(t, "anonymous", actor)
		assert.Equal(t, enum.ActorTypePublic, actorType)
	})

	t.Run("user actor", func(t *testing.T) {
		auth := &mocks.AuthMock{
			GetRequestActorFunc: func(_ *http.Request) (string, string) {
				return "user", "alice"
			},
			IsRequestAdminFunc: func(_ *http.Request) bool { return false },
		}
		l := newLogger(nil, auth)
		req := httptest.NewRequest(http.MethodGet, "/kv/test", http.NoBody)

		actor, actorType := l.extractActor(req)

		assert.Equal(t, "alice", actor)
		assert.Equal(t, enum.ActorTypeUser, actorType)
	})

	t.Run("token actor", func(t *testing.T) {
		auth := &mocks.AuthMock{
			GetRequestActorFunc: func(_ *http.Request) (string, string) {
				return "token", "token:abcd****"
			},
			IsRequestAdminFunc: func(_ *http.Request) bool { return false },
		}
		l := newLogger(nil, auth)
		req := httptest.NewRequest(http.MethodGet, "/kv/test", http.NoBody)

		actor, actorType := l.extractActor(req)

		assert.Equal(t, "token:abcd****", actor)
		assert.Equal(t, enum.ActorTypeToken, actorType)
	})

	t.Run("public actor", func(t *testing.T) {
		auth := &mocks.AuthMock{
			GetRequestActorFunc: func(_ *http.Request) (string, string) {
				return "public", ""
			},
			IsRequestAdminFunc: func(_ *http.Request) bool { return false },
		}
		l := newLogger(nil, auth)
		req := httptest.NewRequest(http.MethodGet, "/kv/test", http.NoBody)

		actor, actorType := l.extractActor(req)

		assert.Equal(t, "anonymous", actor)
		assert.Equal(t, enum.ActorTypePublic, actorType)
	})
}
