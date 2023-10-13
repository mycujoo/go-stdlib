package gcplog

import (
	"log/slog"
	"runtime"
	"strconv"
)

const fieldContext = "context"
const fieldReportLocation = "reportLocation"

// NewReportContext creates a new report context.
// see: https://cloud.google.com/error-reporting/docs/formatting-error-messages
func NewReportContext(pc uintptr) slog.Attr {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()

	return slog.Group(fieldContext,
		slog.Group(fieldReportLocation,
			slog.String("filePath", f.File),
			slog.String("lineNumber", strconv.Itoa(f.Line)),
			slog.String("functionName", f.Function),
		),
	)
}
