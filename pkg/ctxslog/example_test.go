package ctxslog_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mycujoo/go-stdlib/pkg/ctxslog"
)

func ExampleToContext() {
	th := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   true,
		ReplaceAttr: RemoveTimeAndBaseSource,
	})
	logger := slog.New(th)

	ctx := ctxslog.ToContext(context.Background(), logger)

	ctxslog.AddArgs(ctx, slog.String("name", "mycujoo"))

	// This line doesn't appear in the output because default logger is not visible in tests.
	ctxslog.Warn(context.Background(), "this is a warning")

	// This one is not logged because the level is not enabled.
	ctxslog.Debug(ctx, "debug msg")

	err := fmt.Errorf("failed to read data: %w", os.ErrPermission)
	ctxslog.Error(ctx, "failed to read data", "error", err)

	ctxslog.Info(ctx, "additional event")

	l := ctxslog.Extract(ctx)
	l.WithGroup("group").Info("this is a log", "test", "a")
	// Output:
	// level=ERROR source=example_test.go:31 msg="failed to read data" name=mycujoo error="failed to read data: permission denied"
	// level=INFO source=example_test.go:33 msg="additional event" name=mycujoo
	// level=INFO source=example_test.go:36 msg="this is a log" name=mycujoo group.test=a
}

// RemoveTimeAndBaseSource removes the top-level time attribute and simplifies the source file path.
// It is intended to be used as a ReplaceAttr function,
// to make example output deterministic.
func RemoveTimeAndBaseSource(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	if a.Key == slog.SourceKey {
		s := a.Value.Any().(*slog.Source)
		s.File = filepath.Base(s.File)
		return slog.Any(a.Key, s)
	}
	return a
}
