package ctxslog

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

type ctxMarker struct{}

type ctxLogger struct {
	logger *slog.Logger
	args   []any
}

var (
	ctxMarkerKey = &ctxMarker{}
)

// AddArgs adds attributes to the context logger.
func AddArgs(ctx context.Context, args ...any) {
	l, ok := ctx.Value(ctxMarkerKey).(*ctxLogger)
	if !ok || l == nil {
		return
	}
	l.args = append(l.args, args...)
}

// Extract returns the context-scoped Logger.
//
// It always returns a Logger.
func Extract(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(ctxMarkerKey).(*ctxLogger)
	if !ok || l == nil {
		return slog.Default()
	}
	return l.logger.With(l.args...)
}

// ToContext adds the slog.Logger to the context for extraction later.
// Returning the new context that has been created.
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	l := &ctxLogger{
		logger: logger,
	}
	return context.WithValue(ctx, ctxMarkerKey, l)
}

// Debug is equivalent to calling Debug on the logger in the context.
func Debug(ctx context.Context, msg string, args ...any) {
	l := Extract(ctx)
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Debug]
	r := slog.NewRecord(time.Now(), slog.LevelDebug, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// Info is equivalent to calling Info on the logger in the context.
func Info(ctx context.Context, msg string, args ...any) {
	l := Extract(ctx)
	if !l.Enabled(context.Background(), slog.LevelInfo) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Info]
	r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// Warn is equivalent to calling Warn on the logger in the context.
func Warn(ctx context.Context, msg string, args ...any) {
	l := Extract(ctx)
	if !l.Enabled(context.Background(), slog.LevelWarn) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Warn]
	r := slog.NewRecord(time.Now(), slog.LevelWarn, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// Error is equivalent to calling Error on the logger in the context.
func Error(ctx context.Context, msg string, args ...any) {
	l := Extract(ctx)
	if !l.Enabled(context.Background(), slog.LevelError) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Error]
	r := slog.NewRecord(time.Now(), slog.LevelError, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}
