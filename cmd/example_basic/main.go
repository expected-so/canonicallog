package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/expected-so/canonicallog"
)

// Operation defines what we want to do with the logger context
type Operation func(ctx context.Context) error

// ExecuteWithLog wraps an operation with logging
func ExecuteWithLog(operation string, fn Operation) {
	// Setup base context with operation info
	ctx := canonicallog.NewLogLine(context.Background())
	canonicallog.LogAttr(ctx, slog.String("operation", operation))

	// Execute the operation
	startTime := time.Now()
	if err := fn(ctx); err != nil {
		canonicallog.LogError(ctx, err)
	}
	canonicallog.LogDuration(ctx, time.Since(startTime))

	// Log is handled by the wrapper, not the operation
	canonicallog.PrintLine(ctx, operation)
}

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With(
		"service", "demo",
		"env", "local",
	)
	canonicallog.DefaultLoggerFunc = func() *slog.Logger { return logger }

	// Example operations that only handle business logic and attributes
	validateUser := func(ctx context.Context) error {
		canonicallog.LogAttr(ctx, slog.String("user_id", "123"))
		canonicallog.LogAttr(ctx, slog.Bool("is_valid", true))
		return nil
	}

	processPayment := func(ctx context.Context) error {
		canonicallog.LogAttr(ctx, slog.String("payment_id", "PAY123"))
		canonicallog.LogAttr(ctx, slog.Float64("amount", 99.99))
		return nil
	}

	processWithError := func(ctx context.Context) error {
		return errors.New("something went wrong")
	}

	// Execute operations with logging handled by wrapper
	ExecuteWithLog("validate_user", validateUser)
	ExecuteWithLog("process_payment", processPayment)
	ExecuteWithLog("process_with_error", processWithError)
}
