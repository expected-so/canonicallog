package canonicallog

import (
	"context"
	"log/slog"
	"time"
)

const contextKey = "canonical_log"

var DefaultLoggerFunc = func() *slog.Logger {
	return slog.Default()
}

type logContext struct {
	attrs  []slog.Attr
	logger *slog.Logger
}

func NewLogLine(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := make([]slog.Attr, 0)
	return context.WithValue(ctx, contextKey, &logContext{
		attrs:  attrs,
		logger: nil,
	})
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	lc := fromContext(ctx)
	lc.logger = logger
	return ctx
}

func PrintLine(ctx context.Context, message string) {
	lc := fromContext(ctx)

	logLevel := slog.LevelInfo
	for _, attr := range lc.attrs {
		if attr.Key == "error" {
			logLevel = slog.LevelError
		}
	}

	if lc.logger != nil {
		lc.logger.LogAttrs(ctx, logLevel, message, lc.attrs...)
	} else {
		DefaultLoggerFunc().LogAttrs(ctx, logLevel, message, lc.attrs...)
	}
}

func LogAttr(ctx context.Context, attr slog.Attr) {
	lc := fromContext(ctx)
	lc.attrs = append(lc.attrs, attr)
}

func LogDuration(ctx context.Context, duration time.Duration) {
	LogAttr(ctx, slog.Duration("duration", duration))
}

func LogError(ctx context.Context, err error) {
	LogAttr(ctx, slog.Any("error", err))
}

func fromContext(ctx context.Context) *logContext {
	if ctx == nil {
		return fromContext(NewLogLine(context.Background()))
	}

	val := ctx.Value(contextKey)
	if val == nil {
		return fromContext(NewLogLine(ctx))
	}

	lc, ok := val.(*logContext)
	if !ok {
		return fromContext(NewLogLine(ctx))
	}

	return lc
}
