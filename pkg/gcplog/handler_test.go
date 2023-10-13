package gcplog_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/mycujoo/go-stdlib/pkg/gcplog"
	"github.com/mycujoo/go-stdlib/pkg/gcplog/internal/require"
	"github.com/mycujoo/go-stdlib/pkg/gcplog/internal/slogtest"
	"go.opentelemetry.io/otel/trace"
)

func TestHandler(t *testing.T) {
	t.Run("Enabled", func(t *testing.T) {
		tests := []struct {
			name     string
			leveler  slog.Leveler
			level    slog.Level
			expected bool
		}{
			{"nil info", nil, slog.LevelInfo, true},
			{"nil debug", nil, slog.LevelDebug, false},
			{"debug info", slog.LevelDebug, slog.LevelInfo, true},
			{"debug debug", slog.LevelDebug, slog.LevelDebug, true},
			{"error warn", slog.LevelError, slog.LevelWarn, false},
			{"error error", slog.LevelError, slog.LevelError, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				type Entry struct{}

				ctx := context.Background()
				var capture slogtest.Capture[Entry]
				logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, &gcplog.HandlerOptions{
					Level: tt.leveler,
				}))

				logger.LogAttrs(ctx, tt.level, "level")
				entries := capture.Entries()
				received := len(entries) == 1
				err := errs.Err()

				require.NoError(t, err)
				require.Equal(t, tt.expected, received)
			})
		}
	})

	t.Run("message", func(t *testing.T) {
		type Entry struct {
			Message string `json:"message"`
		}

		var capture slogtest.Capture[Entry]
		logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, &gcplog.HandlerOptions{}))

		logger.Info("hello")
		logger.Warn("world")
		entries := capture.Entries()
		err := errs.Err()

		require.NoError(t, err)
		require.Equal(t, "hello", entries[0].Message)
		require.Equal(t, "world", entries[1].Message)
	})

	t.Run("severity", func(t *testing.T) {
		tests := []struct {
			name     string
			level    slog.Level
			expected string
		}{
			{"debug", slog.LevelDebug, "DEBUG"},
			{"info", slog.LevelInfo, "INFO"},
			{"warn", slog.LevelWarn, "WARNING"},
			{"error", slog.LevelError, "ERROR"},
			{"below debug", slog.LevelDebug - 1, "DEBUG"},
			{"below info", slog.LevelInfo - 1, "DEBUG"},
			{"below warn", slog.LevelWarn - 1, "INFO"},
			{"below error", slog.LevelError - 1, "WARNING"},
			{"above error", slog.LevelError + 1, "ERROR"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				type Entry struct {
					Severity string `json:"severity"`
				}
				ctx := context.Background()
				var capture slogtest.Capture[Entry]
				logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, &gcplog.HandlerOptions{
					Level: slog.Level(-1e6),
				}))

				logger.LogAttrs(ctx, tt.level, "level")
				entries := capture.Entries()
				err := errs.Err()

				require.NoError(t, err)
				require.Equal(t, tt.expected, entries[0].Severity)
			})
		}
	})

	t.Run("source location", func(t *testing.T) {
		type Entry struct {
			SourceLocation struct {
				File     string `json:"file"`
				Line     int    `json:"line"`
				Function string `json:"function"`
			} `json:"logging.googleapis.com/sourceLocation"`
		}

		var capture slogtest.Capture[Entry]
		logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, &gcplog.HandlerOptions{
			AddSource: true,
		}))

		logger.Info("hello")
		fs := runtime.CallersFrames([]uintptr{getPC()})
		expected, _ := fs.Next()
		entries := capture.Entries()
		received := entries[0].SourceLocation
		err := errs.Err()

		require.NoError(t, err)
		require.Equal(t, expected.File, received.File)
		require.Equal(t, expected.Line-1, received.Line)
		require.Equal(t, expected.Function, received.Function)
	})

	t.Run("serviceContext", func(t *testing.T) {
		type ServiceContext struct {
			Service string `json:"service"`
			Version string `json:"version"`
		}
		type ServiceContextEntry struct {
			ServiceContext ServiceContext `json:"serviceContext"`
		}

		tests := []struct {
			name     string
			opts     *gcplog.HandlerOptions
			expected ServiceContext
		}{
			{
				"no service name",
				nil,
				ServiceContext{},
			},
			{
				"with service name",
				&gcplog.HandlerOptions{
					ServiceName: "my-service",
				},
				ServiceContext{
					Service: "my-service",
				},
			},
			{
				"with service version only no context",
				&gcplog.HandlerOptions{
					ServiceVersion: "vvv",
				},
				ServiceContext{},
			},
			{
				"with service name and version",
				&gcplog.HandlerOptions{
					ServiceName:    "my-service",
					ServiceVersion: "xxyyzz",
				},
				ServiceContext{
					Service: "my-service",
					Version: "xxyyzz",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				var capture slogtest.Capture[ServiceContextEntry]
				logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, tt.opts))

				logger.Info("with service")
				entries := capture.Entries()
				received := entries[0]
				err := errs.Err()

				require.NoError(t, err)
				require.Equal(t, tt.expected, received.ServiceContext)
			})
		}
	})

	t.Run("trace", func(t *testing.T) {
		type TraceInfo struct {
			TraceID      *string `json:"logging.googleapis.com/trace"`
			SpanID       *string `json:"logging.googleapis.com/spanId"`
			TraceSampled *bool   `json:"logging.googleapis.com/trace_sampled"`
		}

		tests := []struct {
			name     string
			opts     *gcplog.HandlerOptions
			ctx      context.Context
			expected TraceInfo
		}{
			{
				"no trace info",
				nil,
				context.Background(),
				TraceInfo{},
			},
			{
				"span ID unavailable",
				&gcplog.HandlerOptions{
					GCPProjectID: "my-project",
				},
				trace.ContextWithSpanContext(context.Background(), trace.SpanContext{}),
				TraceInfo{},
			},
			{
				"sampled",
				&gcplog.HandlerOptions{
					GCPProjectID: "my-project",
				},
				trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    [16]byte{1, 1},
					SpanID:     trace.SpanID{2},
					TraceFlags: trace.FlagsSampled,
				})),
				TraceInfo{
					TraceID:      vptr("projects/my-project/traces/01010000000000000000000000000000"),
					SpanID:       vptr("0200000000000000"),
					TraceSampled: vptr(true),
				},
			},
			{
				"not sampled",
				&gcplog.HandlerOptions{
					GCPProjectID: "my-project",
				},
				trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: [16]byte{1, 1},
					SpanID:  trace.SpanID{2},
				})),
				TraceInfo{
					TraceID:      vptr("projects/my-project/traces/01010000000000000000000000000000"),
					SpanID:       vptr("0200000000000000"),
					TraceSampled: vptr(false),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				ctx := tt.ctx
				var capture slogtest.Capture[TraceInfo]
				logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, tt.opts))

				logger.InfoContext(ctx, "trace")
				entries := capture.Entries()
				received := entries[0]
				err := errs.Err()

				require.NoError(t, err)
				require.Equal(t, tt.expected, received)
			})
		}
	})

	t.Run("groups and attrs", func(t *testing.T) {
		t.Run("nested", func(t *testing.T) {
			type Nested2 struct {
				CustomPrepared3 int
				CustomAdded     int
			}

			type Nested1 struct {
				CustomPrepared2 int
				Nested2         Nested2
			}

			type Entry struct {
				CustomPrepared1 int
				Nested1         Nested1
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			var h slog.Handler = gcplog.NewHandler(&capture, nil)
			h = h.WithAttrs([]slog.Attr{slog.Int64("CustomPrepared1", 1)})
			h = h.WithGroup("Nested1")
			h = h.WithAttrs([]slog.Attr{slog.Int64("CustomPrepared2", 2)})
			h = h.WithGroup("Nested2")
			h = h.WithAttrs([]slog.Attr{slog.Int64("CustomPrepared3", 3)})
			logger, errs := slogtest.NewWithErrorHandler(h)
			expected := Entry{
				CustomPrepared1: 1,
				Nested1: Nested1{
					CustomPrepared2: 2,
					Nested2: Nested2{
						CustomPrepared3: 3,
						CustomAdded:     4,
					},
				},
			}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Int64("CustomAdded", 4))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("group", func(t *testing.T) {
			type Group struct {
				Val1 string
				Val2 int
			}

			type Entry struct {
				Group Group
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{
				Group: Group{
					Val1: "abc",
					Val2: 123,
				},
			}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Group("Group",
				slog.String("Val1", "abc"),
				slog.Int64("Val2", 123),
			))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("empty group", func(t *testing.T) {
			type Entry struct {
				Group *struct{}
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{Group: nil}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Group("Group"))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("string", func(t *testing.T) {
			type Entry struct {
				StringVal string
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{"cbd"}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.String("StringVal", "cbd"))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("int64", func(t *testing.T) {
			type Entry struct {
				Int64Val int64
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{-1234}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Int64("Int64Val", -1234))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("uint64", func(t *testing.T) {
			type Entry struct {
				Uint64Val uint64
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{1234}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Uint64("Uint64Val", 1234))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("float64", func(t *testing.T) {
			type Entry struct {
				Float64Val float64
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{12.34}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Float64("Float64Val", 12.34))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("bool", func(t *testing.T) {
			type Entry struct {
				BoolVal1 bool
				BoolVal2 bool
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{true, false}

			logger.LogAttrs(ctx, slog.LevelError, "attrs",
				slog.Bool("BoolVal1", true),
				slog.Bool("BoolVal2", false),
			)
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("duration", func(t *testing.T) {
			type Entry struct {
				DurationVal1 time.Duration
				DurationVal2 time.Duration
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{123456, -234567}

			logger.LogAttrs(ctx, slog.LevelError, "attrs",
				slog.Duration("DurationVal1", 123456),
				slog.Duration("DurationVal2", -234567),
			)
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("time", func(t *testing.T) {
			type Entry struct {
				TimeVal1 string
				TimeVal2 string
			}

			ctx := context.Background()
			now := time.Now()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{
				"2023-06-15T19:24:13.123456789Z",
				now.Round(0).Format(time.RFC3339Nano),
			}

			logger.LogAttrs(ctx, slog.LevelError, "attrs",
				slog.Time("TimeVal1", time.Date(2023, 6, 15, 19, 24, 13, 123456789, time.FixedZone("gcp", 0))),
				slog.Time("TimeVal2", now),
			)
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("error", func(t *testing.T) {
			type Entry struct {
				ErrorVal string
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{"unknown error"}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Any("ErrorVal", errors.New("unknown error")))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("error with formatter", func(t *testing.T) {
			type Entry struct {
				Error        string
				ErrorVerbose string
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{
				Error:        "foo",
				ErrorVerbose: "EXTRA\nfoo",
			}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", gcplog.Error(ExtendedError{"foo"}))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("error with custom marshal", func(t *testing.T) {
			type Entry struct {
				JSONErrorVal JSONError
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{JSONError{"foo"}}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Any("JSONErrorVal", JSONError{"foo"}))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("json value", func(t *testing.T) {
			type JSONVal struct {
				Val1 string
				Val2 int
			}

			type Entry struct {
				JSONVal JSONVal
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{JSONVal{"bcd", 234}}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.Any("JSONVal", JSONVal{"bcd", 234}))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.NoError(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("error", func(t *testing.T) {
			type Entry struct {
				Correct  string
				Erroring *struct{}
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{"correct", nil}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.String("Correct", "correct"), slog.Any("erroring", ErroringMarshal{}))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.Error(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("WithAttrs error", func(t *testing.T) {
			type Entry struct {
				Correct  string
				Erroring *struct{}
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{"correct", nil}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.String("Correct", "correct"), slog.Any("Erroring", ErroringMarshal{}))
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.Error(t, err)
			require.Equal(t, expected, received)
		})

		t.Run("Invalid Attr Kind", func(t *testing.T) {
			type Entry struct {
				Correct  string
				Erroring *struct{}
			}

			type FakeValue struct {
				num uint64
				any any
			}

			ctx := context.Background()
			var capture slogtest.Capture[Entry]
			logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&capture, nil))
			expected := Entry{"correct", nil}
			invalidAttr := slog.Attr{
				Key: "Erroring",
				Value: *(*slog.Value)(unsafe.Pointer(&FakeValue{
					any: slog.Kind(0xDEADBEEF),
				})),
			}

			logger.LogAttrs(ctx, slog.LevelError, "attrs", slog.String("Correct", "correct"), invalidAttr)
			entries := capture.Entries()
			received := entries[0]
			err := errs.Err()

			require.Error(t, err)
			require.Equal(t, expected, received)
		})
	})

	t.Run("Writer error", func(t *testing.T) {
		ctx := context.Background()
		var w ErrorWriter
		logger, errs := slogtest.NewWithErrorHandler(gcplog.NewHandler(&w, nil))

		logger.LogAttrs(ctx, slog.LevelError, "write error")
		err := errs.Err()

		require.Error(t, err)
	})
}

func Benchmark(b *testing.B) {
	w := &IgnoreWriter{}
	level := slog.Level(-1e6)
	slogdriverLogger := slog.New(gcplog.NewHandler(w, &gcplog.HandlerOptions{
		Level: level,
	}))
	jsonLogger := slog.New(NewCloudLoggingJSONHandler(w, level))

	b.Run("gcplog", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			slogdriverLogger.Info("hello world")
		}
	})

	b.Run("cloud logging JSONHandler", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			jsonLogger.Info("hello world")
		}
	})
}

func NewCloudLoggingJSONHandler(w io.Writer, level slog.Leveler) *slog.JSONHandler {
	const (
		fieldMessage        = "message"
		fieldTimestamp      = "timestamp"
		fieldSeverity       = "severity"
		fieldSourceLocation = "logging.googleapis.com/sourceLocation"
	)

	const (
		slogFieldMessage = iota + 1
		slogFieldTimestamp
		slogFieldLevel
		slogFieldSource
	)

	const (
		severityError = 500
		severityWarn  = 400
		severityInfo  = 300
		severityDebug = 200
	)

	mappings := map[string]int{
		slog.MessageKey: slogFieldMessage,
		slog.TimeKey:    slogFieldTimestamp,
		slog.LevelKey:   slogFieldLevel,
		slog.SourceKey:  slogFieldSource,
	}

	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 {
				switch mappings[a.Key] {
				case slogFieldMessage:
					return slog.Attr{Key: fieldMessage, Value: a.Value}
				case slogFieldTimestamp:
					return slog.Attr{Key: fieldTimestamp, Value: a.Value}
				case slogFieldSource:
					return slog.Attr{Key: fieldSourceLocation, Value: a.Value}
				case slogFieldLevel:
					level := a.Value.Any().(slog.Level)
					switch {
					case level >= slog.LevelError:
						return slog.Int64(fieldSeverity, severityError)
					case level >= slog.LevelWarn:
						return slog.Int64(fieldSeverity, severityWarn)
					case level >= slog.LevelInfo:
						return slog.Int64(fieldSeverity, severityInfo)
					default:
						return slog.Int64(fieldSeverity, severityDebug)
					}
				}
			}
			return a
		},
	})
}

type ExtendedError struct {
	Message string
}

func (e ExtendedError) Error() string {
	return e.Message
}

func (e ExtendedError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "EXTRA\n%s", e.Message)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

type JSONError struct {
	Message string
}

func (e JSONError) Error() string {
	return e.Message
}

func (e JSONError) MarshalJSON() ([]byte, error) {
	f := e.jsonFormat()
	f.Message = e.Message
	return json.Marshal(f)
}

func (e *JSONError) UnmarshalJSON(data []byte) error {
	f := e.jsonFormat()
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	e.Message = f.Message
	return nil
}

func (JSONError) jsonFormat() (v struct {
	Message string `json:"message"`
}) {
	return v
}

type IgnoreWriter struct{}

func (*IgnoreWriter) Write(data []byte) (n int, err error) {
	return len(data), nil
}

type ErrorWriter struct{}

func (*ErrorWriter) Write(data []byte) (n int, err error) {
	return 0, fmt.Errorf("error writing")
}

type ErroringMarshal struct{}

func (ErroringMarshal) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("cannot be marshaled")
}

func getPC() uintptr {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	return pcs[0]
}

func vptr[T any](v T) *T {
	return &v
}
