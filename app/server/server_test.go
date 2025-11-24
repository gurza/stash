package server

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/stash/app/server/mocks"
	"github.com/umputun/stash/app/store"
)

func TestServer_HandleGet(t *testing.T) {
	st := &mocks.KVStoreMock{
		GetFunc: func(key string) ([]byte, error) {
			switch key {
			case "testkey":
				return []byte("testvalue"), nil
			case "path/to/key":
				return []byte("nested value"), nil
			default:
				return nil, store.ErrNotFound
			}
		},
		ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
	}
	srv := newTestServer(t, st)

	t.Run("get existing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/kv/testkey", http.NoBody)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "testvalue", rec.Body.String())
	})

	t.Run("get nonexistent key returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/kv/nonexistent", http.NoBody)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("get key with slashes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/kv/path/to/key", http.NoBody)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "nested value", rec.Body.String())
	})
}

func TestServer_HandleSet(t *testing.T) {
	t.Run("set new key", func(t *testing.T) {
		st := &mocks.KVStoreMock{
			SetFunc:  func(key string, value []byte) error { return nil },
			ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
		}
		srv := newTestServer(t, st)

		body := bytes.NewBufferString("newvalue")
		req := httptest.NewRequest(http.MethodPut, "/kv/newkey", body)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, st.SetCalls(), 1)
		assert.Equal(t, "newkey", st.SetCalls()[0].Key)
		assert.Equal(t, []byte("newvalue"), st.SetCalls()[0].Value)
	})

	t.Run("update existing key", func(t *testing.T) {
		st := &mocks.KVStoreMock{
			SetFunc:  func(key string, value []byte) error { return nil },
			ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
		}
		srv := newTestServer(t, st)

		body := bytes.NewBufferString("updated")
		req := httptest.NewRequest(http.MethodPut, "/kv/existing", body)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, st.SetCalls(), 1)
		assert.Equal(t, "existing", st.SetCalls()[0].Key)
		assert.Equal(t, []byte("updated"), st.SetCalls()[0].Value)
	})

	t.Run("set key with slashes", func(t *testing.T) {
		st := &mocks.KVStoreMock{
			SetFunc:  func(key string, value []byte) error { return nil },
			ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
		}
		srv := newTestServer(t, st)

		body := bytes.NewBufferString("nested")
		req := httptest.NewRequest(http.MethodPut, "/kv/a/b/c", body)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, st.SetCalls(), 1)
		assert.Equal(t, "a/b/c", st.SetCalls()[0].Key)
		assert.Equal(t, []byte("nested"), st.SetCalls()[0].Value)
	})
}

func TestServer_HandleDelete(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		st := &mocks.KVStoreMock{
			DeleteFunc: func(key string) error { return nil },
			ListFunc:   func() ([]store.KeyInfo, error) { return nil, nil },
		}
		srv := newTestServer(t, st)

		req := httptest.NewRequest(http.MethodDelete, "/kv/todelete", http.NoBody)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		require.Len(t, st.DeleteCalls(), 1)
		assert.Equal(t, "todelete", st.DeleteCalls()[0].Key)
	})

	t.Run("delete nonexistent key returns 404", func(t *testing.T) {
		st := &mocks.KVStoreMock{
			DeleteFunc: func(key string) error { return store.ErrNotFound },
			ListFunc:   func() ([]store.KeyInfo, error) { return nil, nil },
		}
		srv := newTestServer(t, st)

		req := httptest.NewRequest(http.MethodDelete, "/kv/nonexistent", http.NoBody)
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		require.Len(t, st.DeleteCalls(), 1)
		assert.Equal(t, "nonexistent", st.DeleteCalls()[0].Key)
	})
}

func TestServer_Ping(t *testing.T) {
	st := &mocks.KVStoreMock{
		ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
	}
	srv := newTestServer(t, st)

	req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestServer_HandleGet_InternalError(t *testing.T) {
	st := &mocks.KVStoreMock{
		GetFunc:  func(key string) ([]byte, error) { return nil, errors.New("db error") },
		ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
	}
	srv := newTestServer(t, st)

	req := httptest.NewRequest(http.MethodGet, "/kv/testkey", http.NoBody)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Len(t, st.GetCalls(), 1)
	assert.Equal(t, "testkey", st.GetCalls()[0].Key)
}

func TestServer_HandleSet_InternalError(t *testing.T) {
	st := &mocks.KVStoreMock{
		SetFunc:  func(key string, value []byte) error { return errors.New("db error") },
		ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
	}
	srv := newTestServer(t, st)

	body := bytes.NewBufferString("value")
	req := httptest.NewRequest(http.MethodPut, "/kv/testkey", body)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Len(t, st.SetCalls(), 1)
	assert.Equal(t, "testkey", st.SetCalls()[0].Key)
}

func TestServer_HandleDelete_InternalError(t *testing.T) {
	st := &mocks.KVStoreMock{
		DeleteFunc: func(key string) error { return errors.New("db error") },
		ListFunc:   func() ([]store.KeyInfo, error) { return nil, nil },
	}
	srv := newTestServer(t, st)

	req := httptest.NewRequest(http.MethodDelete, "/kv/testkey", http.NoBody)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Len(t, st.DeleteCalls(), 1)
	assert.Equal(t, "testkey", st.DeleteCalls()[0].Key)
}

func TestServer_New_InvalidTokens(t *testing.T) {
	st := &mocks.KVStoreMock{
		ListFunc: func() ([]store.KeyInfo, error) { return nil, nil },
	}
	_, err := New(st, Config{
		Address:      ":8080",
		ReadTimeout:  5 * time.Second,
		PasswordHash: "$2a$10$hash",
		AuthTokens:   []string{"invalid"}, // bad format
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func newTestServer(t *testing.T, st KVStore) *Server {
	t.Helper()
	srv, err := New(st, Config{Address: ":8080", ReadTimeout: 5 * time.Second, Version: "test"})
	require.NoError(t, err)
	return srv
}
