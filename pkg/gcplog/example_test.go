package gcplog_test

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mycujoo/go-stdlib/pkg/gcplog"
)

func ExampleNewAutoHandler() {
	h := gcplog.NewAutoHandler(os.Stderr, &gcplog.HandlerOptions{
		AddSource:      true,
		ServiceName:    "some-service",
		ServiceVersion: "GITSHA",
		ReportErrors:   true,
	})

	err := fmt.Errorf("storage.Get: %w", os.ErrNotExist)

	logger := slog.New(h)
	logger.Error("operation failed", gcplog.Error(err))
}
