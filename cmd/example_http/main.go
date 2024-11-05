package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/expected-so/canonicallog"
)

const port = ":8080"

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With(
		"service", "demo",
		"env", "local",
	)
	canonicallog.DefaultLoggerFunc = func() *slog.Logger { return logger }

	// Setup server
	mux := http.NewServeMux()

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		canonicallog.LogAttr(ctx, slog.String("user_id", "123"))
		w.WriteHeader(http.StatusOK)
	})

	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		canonicallog.LogAttr(ctx, slog.String("error", "demo_error"))
		w.WriteHeader(http.StatusInternalServerError)
	})

	mux.Handle("/users", canonicallog.HttpHandler(userHandler))
	mux.Handle("/error", canonicallog.HttpHandler(errorHandler))

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// Start server in background
	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond) // Wait for server to start

	// Make sample requests
	http.Get("http://localhost" + port + "/users")
	http.Get("http://localhost" + port + "/error")

	// Cleanup
	server.Shutdown(context.Background())
}
