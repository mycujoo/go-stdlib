package gcplog

// This package is based on code from https://github.com/jussi-kalliokoski/slogdriver
// Changes:
// Integrated with open telemetry directly.
// Trace context is optional.
// Labels removed.
// Added service context.
// Richer error reporting.

// The license for that code is:
// Copyright 2023 Jussi Kalliokoski
//
// Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	"cloud.google.com/go/compute/metadata"
	"github.com/jussi-kalliokoski/goldjson"
	"github.com/phsym/console-slog"
	"go.opentelemetry.io/otel/trace"
)

// Value for this variable can be set during build.
// go build -ldflags "-X github.com/mycujoo/go-stdlib/pkg/gcplog.serviceVersion=$(git rev-parse HEAD)" -o ./bin/server ./cmd/server
var serviceVersion = ""

type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Minimal log level to log, defaults to slog.LevelInfo
	Level slog.Leveler

	// Service name and version to add to the log
	ServiceName    string
	ServiceVersion string

	// If this is set to true, errors will be reported to GCP error reporting.
	ReportErrors bool

	// GCP project ID to use for trace context
	GCPProjectID string
}

// NewAutoHandler returns slog.Handler that writes to w using GCP structured logging format.
// It automatically detects GCP project ID.
// If the program is not running on GCE, it returns console handler.
func NewAutoHandler(w io.Writer, opts *HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &HandlerOptions{}
	}
	if !metadata.OnGCE() {
		o := &console.HandlerOptions{
			Level:     opts.Level,
			AddSource: opts.AddSource,
		}

		return console.NewHandler(os.Stderr, o)
	}
	if opts.GCPProjectID == "" {
		// Detect project ID
		opts.GCPProjectID, _ = metadata.ProjectID()
	}
	if opts.ServiceVersion == "" {
		// If service version is not set, use the value provided during linking
		opts.ServiceVersion = serviceVersion
	}
	return NewHandler(w, opts)
}

// NewHandler returns slog.Handler that writes to w using GCP structured logging format.
func NewHandler(w io.Writer, opts *HandlerOptions) *Handler {
	if opts == nil {
		opts = &HandlerOptions{}
	}
	encoder := goldjson.NewEncoder(w)
	encoder.PrepareKey(fieldMessage)
	encoder.PrepareKey(fieldTimestamp)
	encoder.PrepareKey(fieldSeverity)
	if opts.AddSource {
		encoder.PrepareKey(fieldSourceLocation)
		encoder.PrepareKey(fieldSourceFile)
		encoder.PrepareKey(fieldSourceLine)
		encoder.PrepareKey(fieldSourceFunction)
	}
	if opts.GCPProjectID != "" {
		encoder.PrepareKey(fieldTraceID)
		encoder.PrepareKey(fieldTraceSpanID)
		encoder.PrepareKey(fieldTraceSampled)
	}
	if opts.ServiceName != "" {
		encoder.PrepareKey(fieldServiceContext)
		encoder.PrepareKey(fieldService)
		encoder.PrepareKey(fieldVersion)
	}
	if opts.ReportErrors {
		encoder.PrepareKey(fieldContext)
	}
	return &Handler{
		opts:    *opts,
		encoder: encoder,
	}
}

type Handler struct {
	opts         HandlerOptions
	encoder      *goldjson.Encoder
	attrBuilders []func(ctx context.Context, h *Handler, l *goldjson.LineWriter, next func(context.Context) error) error
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	l := h.encoder.NewLine()

	// Add message
	l.AddString(fieldMessage, r.Message)

	// Add timestamp
	time := r.Time.Round(0) // strip monotonic to match Attr behavior
	_ = l.AddTime(fieldTimestamp, time)

	// Add severity
	switch {
	case r.Level >= slog.LevelError:
		l.AddString(fieldSeverity, severityError)
	case r.Level >= slog.LevelWarn:
		l.AddString(fieldSeverity, severityWarn)
	case r.Level >= slog.LevelInfo:
		l.AddString(fieldSeverity, severityInfo)
	default:
		l.AddString(fieldSeverity, severityDebug)
	}

	if h.opts.AddSource {
		addSourceLocation(l, &r)
	}

	if h.opts.GCPProjectID != "" {
		addTrace(ctx, l, h.opts.GCPProjectID)
	}

	if h.opts.ServiceName != "" {
		addServiceContext(l, h.opts.ServiceName, h.opts.ServiceVersion)
	}

	// Error reporting doesn't work without a service name
	if h.opts.ServiceName != "" && h.opts.ReportErrors && r.Level >= slog.LevelError {
		var hasReport bool
		r.Attrs(func(attr slog.Attr) bool {
			if attr.Key == fieldContext {
				// We already have context as Attr
				hasReport = true
				return false
			}
			return true
		})
		if !hasReport {
			r.AddAttrs(NewReportContext(r.PC))
		}
	}

	// Add attributes
	err := h.addAttrs(ctx, l, &r)
	err = errors.Join(err, l.End())

	return err
}

func (h *Handler) WithAttrs(as []slog.Attr) slog.Handler {
	clone := *h
	staticFields, w := goldjson.NewStaticFields()
	var err error
	for _, attr := range as {
		err = errors.Join(err, addAttr(w, attr))
	}
	clone.attrBuilders = cloneAppend(
		h.attrBuilders,
		func(ctx context.Context, h *Handler, l *goldjson.LineWriter, next func(context.Context) error) error {
			l.AddStaticFields(staticFields)
			return errors.Join(err, next(ctx))
		},
	)
	err = w.End()
	return &clone
}

func (h *Handler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.encoder = h.encoder.Clone()
	clone.encoder.PrepareKey(name)
	clone.attrBuilders = cloneAppend(
		h.attrBuilders,
		func(ctx context.Context, h *Handler, l *goldjson.LineWriter, next func(context.Context) error) error {
			l.StartRecord(name)
			defer l.EndRecord()
			return next(ctx)
		},
	)
	return &clone
}

func addSourceLocation(l *goldjson.LineWriter, r *slog.Record) {
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()

	l.StartRecord(fieldSourceLocation)
	defer l.EndRecord()

	l.AddString(fieldSourceFile, f.File)
	l.AddInt64(fieldSourceLine, int64(f.Line))
	l.AddString(fieldSourceFunction, f.Function)
}

func addTrace(ctx context.Context, l *goldjson.LineWriter, projectName string) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return
	}

	l.AddString(fieldTraceID, fmt.Sprintf("projects/%s/traces/%s", projectName, sc.TraceID().String()))
	l.AddString(fieldTraceSpanID, sc.SpanID().String())
	l.AddBool(fieldTraceSampled, sc.IsSampled())
}

func addServiceContext(l *goldjson.LineWriter, name, version string) {
	l.StartRecord(fieldServiceContext)
	defer l.EndRecord()

	l.AddString(fieldService, name)
	l.AddString(fieldVersion, version)
}

func (h *Handler) addAttrs(ctx context.Context, l *goldjson.LineWriter, r *slog.Record) error {
	if len(h.attrBuilders) == 0 {
		return addAttrsRaw(l, r)
	}

	b := func(ctx context.Context) error {
		return addAttrsRaw(l, r)
	}

	for i := range h.attrBuilders {
		attrBuilder := h.attrBuilders[len(h.attrBuilders)-1-i]
		next := b
		b = func(ctx context.Context) error {
			return attrBuilder(ctx, h, l, next)
		}
	}

	return b(ctx)
}

func addAttrsRaw(l *goldjson.LineWriter, r *slog.Record) error {
	var err error
	r.Attrs(func(attr slog.Attr) bool {
		err = errors.Join(err, addAttr(l, attr))
		return true
	})
	return err
}

func addAttr(l *goldjson.LineWriter, a slog.Attr) error {
	a.Value.Resolve()
	switch a.Value.Kind() {
	case slog.KindGroup:
		return addGroup(l, a)
	case slog.KindString:
		l.AddString(a.Key, a.Value.String())
		return nil
	case slog.KindInt64:
		l.AddInt64(a.Key, a.Value.Int64())
		return nil
	case slog.KindUint64:
		l.AddUint64(a.Key, a.Value.Uint64())
		return nil
	case slog.KindFloat64:
		l.AddFloat64(a.Key, a.Value.Float64())
		return nil
	case slog.KindBool:
		l.AddBool(a.Key, a.Value.Bool())
		return nil
	case slog.KindDuration:
		l.AddInt64(a.Key, int64(a.Value.Duration()))
		return nil
	case slog.KindTime:
		return l.AddTime(a.Key, a.Value.Time())
	case slog.KindAny:
		return addAny(l, a)
	}
	return fmt.Errorf("bad kind: %s", a.Value.Kind())
}

func addGroup(l *goldjson.LineWriter, a slog.Attr) error {
	attrs := a.Value.Group()
	if len(attrs) == 0 {
		return nil
	}
	l.StartRecord(a.Key)
	defer l.EndRecord()
	var err error
	for _, a := range attrs {
		err = errors.Join(err, addAttr(l, a))
	}
	return err
}

func addAny(l *goldjson.LineWriter, a slog.Attr) error {
	v := a.Value.Any()
	_, jm := v.(json.Marshaler)
	if err, ok := v.(error); ok && !jm {
		return addError(l, a.Key, err)
	}
	return l.AddMarshal(a.Key, v)
}

const (
	fieldMessage        = "message"
	fieldTimestamp      = "time"
	fieldSeverity       = "severity"
	fieldSourceLocation = "logging.googleapis.com/sourceLocation"
	fieldSourceFile     = "file"
	fieldSourceLine     = "line"
	fieldSourceFunction = "function"
	fieldTraceID        = "logging.googleapis.com/trace"
	fieldTraceSpanID    = "logging.googleapis.com/spanId"
	fieldTraceSampled   = "logging.googleapis.com/trace_sampled"
	fieldServiceContext = "serviceContext"
	fieldService        = "service"
	fieldVersion        = "version"
)

const (
	severityError = "ERROR"
	severityWarn  = "WARNING"
	severityInfo  = "INFO"
	severityDebug = "DEBUG"
)

func cloneSlice[T any](slice []T, extraCap int) []T {
	return append(make([]T, 0, len(slice)+extraCap), slice...)
}

func cloneAppend[T any](slice []T, values ...T) []T {
	return append(cloneSlice(slice, len(values)), values...)
}
