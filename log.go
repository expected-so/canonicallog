package canonicallog

import (
	"context"
	"log/slog"
	"time"
)

const contextKey = "canonical_log"

func NewLogLine(ctx context.Context) context.Context {
	attrs := new([]slog.Attr)
	return context.WithValue(ctx, contextKey, attrs)
}

func PrintLine(ctx context.Context, message string) {
	attrs, ok := ctx.Value(contextKey).(*[]slog.Attr)
	if !ok || attrs == nil {
		return
	}

	logLevel := slog.LevelInfo
	logArgs := make([]any, len(*attrs))
	for index, attr := range *attrs {
		logArgs[index] = attr
		if attr.Key == "error" {
			logLevel = slog.LevelError
		}
	}
	slog.Log(ctx, logLevel, message, logArgs...)
}

func LogAttr(ctx context.Context, attr slog.Attr) {
	attrs, ok := ctx.Value(contextKey).(*[]slog.Attr)
	if !ok || attrs == nil {
		return
	}
	*attrs = append(*attrs, attr)
}

func LogDuration(ctx context.Context, duration time.Duration) {
	LogAttr(ctx, slog.Duration("duration", duration))
}

func LogError(ctx context.Context, err error) {
	LogAttr(ctx, slog.Any("error", err))
}
