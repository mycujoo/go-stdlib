package connectlog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/mycujoo/go-stdlib/pkg/ctxslog"
)

var errInternal = errors.New("internal error")

// NewLoggingInterceptor returns unary interceptor that logs response errors.
// It injects context logger with method and trace context.
func NewLoggingInterceptor(logger *slog.Logger, opts ...Option) connect.UnaryInterceptorFunc {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			l := logger.With(methodFields(request.Spec())...)

			// Inject logger into context
			ctx = ctxslog.ToContext(ctx, l)

			resp, err := next(ctx, request)

			// Extract logger from context with all added attributes
			l = ctxslog.Extract(ctx)

			if err != nil {
				var level slog.Level
				var msg string
				var attrs []slog.Attr
				originalErr := err

				if connectErr := new(connect.Error); errors.As(err, &connectErr) {
					level = codeToLevel(connectErr.Code())
					msg = fmt.Sprintf("handler error: %s", connectErr.Message())
					attrs = append(attrs, slog.String("code", connectErr.Code().String()))
				} else {
					level = slog.LevelError
					msg = fmt.Sprintf("handler error: %s", err.Error())
					// Hide the internal error from the client
					err = connect.NewError(connect.CodeInternal, errInternal)
				}

				attrs = append(attrs, slog.Any("error", originalErr))
				l.LogAttrs(ctx, level, msg, attrs...)
			} else if o.logSuccess {
				l.Log(ctx, slog.LevelInfo, "handler ok")
			}

			return resp, err
		}
	}
}

func codeToLevel(code connect.Code) slog.Level {
	switch code {
	case connect.CodeCanceled:
		return slog.LevelInfo
	case connect.CodeUnknown:
		return slog.LevelError
	case connect.CodeInvalidArgument:
		return slog.LevelInfo
	case connect.CodeDeadlineExceeded:
		return slog.LevelWarn
	case connect.CodeNotFound:
		return slog.LevelInfo
	case connect.CodeAlreadyExists:
		return slog.LevelInfo
	case connect.CodePermissionDenied:
		return slog.LevelWarn
	case connect.CodeResourceExhausted:
		return slog.LevelWarn
	case connect.CodeFailedPrecondition:
		return slog.LevelWarn
	case connect.CodeAborted:
		return slog.LevelWarn
	case connect.CodeOutOfRange:
		return slog.LevelWarn
	case connect.CodeUnimplemented:
		return slog.LevelError
	case connect.CodeInternal:
		return slog.LevelError
	case connect.CodeUnavailable:
		return slog.LevelWarn
	case connect.CodeDataLoss:
		return slog.LevelError
	case connect.CodeUnauthenticated:
		return slog.LevelInfo
	default:
		return slog.LevelError
	}
}

func methodFields(spec connect.Spec) []any {
	name := strings.TrimLeft(spec.Procedure, "/")
	parts := strings.SplitN(name, "/", 2)
	var fields []any
	switch len(parts) {
	case 0:
		return nil // invalid
	case 1:
		// fall back to treating the whole string as the method
		if method := parts[0]; method != "" {
			fields = append(fields, slog.String("method", method))
		}
	default:
		// some.package.v1.Service/Method
		if svc := parts[0]; svc != "" {
			fields = append(fields, slog.String("service", svc))
		}
		if method := parts[1]; method != "" {
			fields = append(fields, slog.String("method", method))
		}
	}
	return fields
}

// NewLoggingRecoverHandler returns a recover handler that logs panics.
func NewLoggingRecoverHandler(logger *slog.Logger) func(context.Context, connect.Spec, http.Header, any) error {
	return func(ctx context.Context, spec connect.Spec, header http.Header, val any) error {
		// remove authorization header from logs
		header.Del("authorization")
		attrs := append(methodFields(spec),
			slog.Any("headers", header),
			slog.Any("val", val),
		)
		logger.ErrorContext(ctx,
			"handler panic",
			attrs,
		)
		return connect.NewError(connect.CodeInternal, errInternal)
	}
}
