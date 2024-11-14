package canonicallog

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type spyHandler struct {
	attrs []slog.Attr
	level slog.Level
	msg   string
}

func (h *spyHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *spyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)
	return h
}
func (h *spyHandler) WithGroup(_ string) slog.Handler { return h }
func (h *spyHandler) Handle(_ context.Context, r slog.Record) error {
	h.level = r.Level
	h.msg = r.Message
	r.Attrs(func(a slog.Attr) bool {
		h.attrs = append(h.attrs, a)
		return true
	})
	return nil
}

func newTestLogger() (*slog.Logger, *spyHandler) {
	h := &spyHandler{attrs: make([]slog.Attr, 0)}
	return slog.New(h), h
}

func TestLogger(t *testing.T) {
	t.Run("default logger behavior", func(t *testing.T) {
		handler := &spyHandler{attrs: make([]slog.Attr, 0)}
		logger := slog.New(handler)
		DefaultLoggerFunc = func() *slog.Logger { return logger }

		ctx := NewLogLine(context.Background())
		LogAttr(ctx, slog.String("test", "value"))
		PrintLine(ctx, "test message")

		assert.Equal(t, slog.LevelInfo, handler.level)
		assert.Equal(t, "test message", handler.msg)
		assert.Equal(t, "value", findAttr(handler.attrs, "test"))
	})

	t.Run("custom logger with preset attributes", func(t *testing.T) {
		logger, handler := newTestLogger()

		ctx := NewLogLine(context.Background())
		AttachLogger(ctx, logger.With("custom", "value"))
		LogAttr(ctx, slog.String("user_id", "123"))
		PrintLine(ctx, "test message")

		assert.Equal(t, "value", findAttr(handler.attrs, "custom"))
		assert.Equal(t, "123", findAttr(handler.attrs, "user_id"))
	})

	t.Run("error logging changes level", func(t *testing.T) {
		logger, handler := newTestLogger()

		ctx := NewLogLine(context.Background())
		AttachLogger(ctx, logger)
		LogError(ctx, errors.New("test error"))
		PrintLine(ctx, "error message")

		assert.Equal(t, slog.LevelError, handler.level)
		assert.Equal(t, "test error", findAttr(handler.attrs, "error"))
	})

	t.Run("duration logging", func(t *testing.T) {
		logger, handler := newTestLogger()

		ctx := NewLogLine(context.Background())
		AttachLogger(ctx, logger)
		duration := 100 * time.Millisecond
		LogDuration(ctx, duration)
		PrintLine(ctx, "duration test")

		assert.Equal(t, duration.String(), findAttr(handler.attrs, "duration"))
	})

	t.Run("nil context handling", func(t *testing.T) {
		logger, handler := newTestLogger()
		DefaultLoggerFunc = func() *slog.Logger { return logger }

		PrintLine(nil, "nil context test")
		assert.Equal(t, "nil context test", handler.msg)
		assert.Equal(t, slog.LevelInfo, handler.level)
	})

	t.Run("multiple attributes accumulation", func(t *testing.T) {
		logger, handler := newTestLogger()

		ctx := NewLogLine(context.Background())
		AttachLogger(ctx, logger)
		LogAttr(ctx, slog.String("key1", "value1"))
		LogAttr(ctx, slog.String("key2", "value2"))
		LogAttr(ctx, slog.String("key3", "value3"))
		PrintLine(ctx, "multiple attributes")

		assert.Equal(t, "value1", findAttr(handler.attrs, "key1"))
		assert.Equal(t, "value2", findAttr(handler.attrs, "key2"))
		assert.Equal(t, "value3", findAttr(handler.attrs, "key3"))
	})

	t.Run("context reuse", func(t *testing.T) {
		logger, handler := newTestLogger()

		ctx := NewLogLine(context.Background())
		AttachLogger(ctx, logger)
		LogAttr(ctx, slog.String("persistent", "value"))

		PrintLine(ctx, "first message")
		assert.Equal(t, "value", findAttr(handler.attrs, "persistent"))

		LogAttr(ctx, slog.String("additional", "value2"))
		PrintLine(ctx, "second message")
		assert.Equal(t, "value", findAttr(handler.attrs, "persistent"))
		assert.Equal(t, "value2", findAttr(handler.attrs, "additional"))
	})
}

// Helper function to find attribute value by key
func findAttr(attrs []slog.Attr, key string) string {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Value.String()
		}
	}
	return ""
}
