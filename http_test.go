package canonicallog

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHttpHandler(t *testing.T) {
	t.Run("successful GET request", func(t *testing.T) {
		handler := &spyHandler{attrs: make([]slog.Attr, 0)}
		logger := slog.New(handler)
		DefaultLoggerFunc = func() *slog.Logger { return logger }

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test?param=value", nil)
		rec := httptest.NewRecorder()

		HttpHandler(testHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		attrMap := make(map[string]string)
		for _, attr := range handler.attrs {
			attrMap[attr.Key] = attr.Value.String()
		}

		assert.Equal(t, "GET", attrMap["http.method"])
		assert.Equal(t, "/test?param=value", attrMap["http.path"])
		assert.Equal(t, "200", attrMap["http.status_code"])
		assert.Contains(t, attrMap, "duration")
		assert.Equal(t, "http-request", handler.msg)
	})

	t.Run("error response", func(t *testing.T) {
		handler := &spyHandler{attrs: make([]slog.Attr, 0)}
		logger := slog.New(handler)
		DefaultLoggerFunc = func() *slog.Logger { return logger }

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})

		req := httptest.NewRequest(http.MethodPost, "/error", nil)
		rec := httptest.NewRecorder()

		HttpHandler(testHandler).ServeHTTP(rec, req)

		attrMap := make(map[string]string)
		for _, attr := range handler.attrs {
			attrMap[attr.Key] = attr.Value.String()
		}

		assert.Equal(t, "POST", attrMap["http.method"])
		assert.Equal(t, "/error", attrMap["http.path"])
		assert.Equal(t, "400", attrMap["http.status_code"])
	})

	t.Run("hijack support", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hijacker, ok := w.(http.Hijacker)
			assert.True(t, ok, "ResponseWriter should support hijacking")
			_, _, err := hijacker.Hijack()
			// This will fail because httptest.ResponseRecorder doesn't support hijacking
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "hijack not supported")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		HttpHandler(testHandler).ServeHTTP(rec, req)
	})

	t.Run("context propagation", func(t *testing.T) {
		handler := &spyHandler{attrs: make([]slog.Attr, 0)}
		logger := slog.New(handler)
		DefaultLoggerFunc = func() *slog.Logger { return logger }

		var capturedCtx context.Context
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedCtx = r.Context()
			// Add custom attribute in handler
			LogAttr(r.Context(), slog.String("custom", "value"))
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		HttpHandler(testHandler).ServeHTTP(rec, req)

		// Verify context was properly set up
		lc, ok := capturedCtx.Value(contextKey).(*logContext)
		assert.True(t, ok, "Context should contain logContext")
		assert.NotNil(t, lc)

		// Verify all attributes were captured including custom one
		attrMap := make(map[string]string)
		for _, attr := range handler.attrs {
			attrMap[attr.Key] = attr.Value.String()
		}

		assert.Equal(t, "GET", attrMap["http.method"])
		assert.Equal(t, "/", attrMap["http.path"])
		assert.Equal(t, "value", attrMap["custom"])
	})

	t.Run("timing accuracy", func(t *testing.T) {
		handler := &spyHandler{attrs: make([]slog.Attr, 0)}
		logger := slog.New(handler)
		DefaultLoggerFunc = func() *slog.Logger { return logger }

		delay := 100 * time.Millisecond
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(delay)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		HttpHandler(testHandler).ServeHTTP(rec, req)

		var duration time.Duration
		for _, attr := range handler.attrs {
			if attr.Key == "duration" {
				duration = attr.Value.Duration()
				break
			}
		}

		assert.InDelta(t, delay.Seconds(), duration.Seconds(), 0.1)
	})

	t.Run("custom logger with preset attributes", func(t *testing.T) {
		spy := &spyHandler{attrs: make([]slog.Attr, 0)}
		customLogger := slog.New(spy).With(
			"service", "api",
			"environment", "test",
		)

		var capturedCtx context.Context
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedCtx = r.Context()

			// Add handler-specific attributes

			AttachLogger(capturedCtx, customLogger.With(
				"handler", "test-handler",
				"version", "v1",
			))

			// Add request-specific attributes
			LogAttr(capturedCtx, slog.String("request_id", "123"))
			LogAttr(capturedCtx, slog.String("user_id", "user_123"))

			w.WriteHeader(http.StatusAccepted)
		})

		req := httptest.NewRequest(http.MethodPost, "/users/123/profile", nil)
		rec := httptest.NewRecorder()

		HttpHandler(testHandler).ServeHTTP(rec, req)

		// Verify context was properly set up
		lc, ok := capturedCtx.Value(contextKey).(*logContext)
		assert.True(t, ok, "Context should contain logContext")
		assert.NotNil(t, lc)

		// Collect all attributes in a map for easier verification
		attrMap := make(map[string]string)
		for _, attr := range spy.attrs {
			attrMap[attr.Key] = attr.Value.String()
		}

		// Verify request attributes
		assert.Equal(t, "POST", attrMap["http.method"])
		assert.Equal(t, "/users/123/profile", attrMap["http.path"])
		assert.Equal(t, "202", attrMap["http.status_code"])

		// Verify preset logger attributes
		assert.Equal(t, "api", attrMap["service"])
		assert.Equal(t, "test", attrMap["environment"])

		// Verify handler-specific attributes
		assert.Equal(t, "test-handler", attrMap["handler"])
		assert.Equal(t, "v1", attrMap["version"])

		// Verify request-specific attributes
		assert.Equal(t, "123", attrMap["request_id"])
		assert.Equal(t, "user_123", attrMap["user_id"])

		// Verify timing
		assert.Contains(t, attrMap, "duration")
		assert.Equal(t, "http-request", spy.msg)
		assert.Equal(t, slog.LevelInfo, spy.level)
	})
}
