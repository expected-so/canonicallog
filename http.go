package canonicallog

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

var _ http.Hijacker = (*responseWriter)(nil)

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func LogHttpRequest(ctx context.Context, method string) {
	LogAttr(ctx, slog.String("http.method", method))
}

func LogHttpPath(ctx context.Context, path string) {
	LogAttr(ctx, slog.String("http.path", path))
}

func LogHttpStatusCode(ctx context.Context, code int) {
	LogAttr(ctx, slog.Int("http.status_code", code))
}

func HttpHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logContext := NewLogLine(r.Context())

		startAt := time.Now()
		res := &responseWriter{ResponseWriter: w}
		LogHttpRequest(logContext, r.Method)
		LogHttpPath(logContext, r.RequestURI)

		handler.ServeHTTP(res, r.WithContext(logContext))

		LogHttpStatusCode(logContext, res.statusCode)
		LogDuration(logContext, time.Now().Sub(startAt))
		PrintLine(logContext, "http-request")
	})
}
