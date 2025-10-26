package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"ulak/internal/config"
)

func TestApiKeyAuthMiddleware_ValidKey(t *testing.T) {
	cfg := config.Auth{
		ApiKey:       "test-api-key-123",
		ApiHeaderKey: "x-api-key",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := ApiKeyAuthMiddleware(&cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "test-api-key-123")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Expected next handler to be called with valid API key")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestApiKeyAuthMiddleware_MissingKey(t *testing.T) {
	cfg := config.Auth{
		ApiKey:       "test-api-key-123",
		ApiHeaderKey: "x-api-key",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	middleware := ApiKeyAuthMiddleware(&cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No API key header
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if nextCalled {
		t.Error("Expected next handler NOT to be called with missing API key")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestApiKeyAuthMiddleware_EmptyKey(t *testing.T) {
	cfg := config.Auth{
		ApiKey:       "test-api-key-123",
		ApiHeaderKey: "x-api-key",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	middleware := ApiKeyAuthMiddleware(&cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if nextCalled {
		t.Error("Expected next handler NOT to be called with empty API key")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestApiKeyAuthMiddleware_InvalidKey(t *testing.T) {
	cfg := config.Auth{
		ApiKey:       "test-api-key-123",
		ApiHeaderKey: "x-api-key",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	middleware := ApiKeyAuthMiddleware(&cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "wrong-api-key")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if nextCalled {
		t.Error("Expected next handler NOT to be called with invalid API key")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
