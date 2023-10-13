package gcplog

import (
	"fmt"
	"log/slog"

	"github.com/jussi-kalliokoski/goldjson"
)

// Error wraps an error in a slog.Attr with a standard key.
func Error(err error) slog.Attr {
	return slog.Any("error", err)
}

func addError(l *goldjson.LineWriter, key string, err error) error {
	basic := err.Error()
	l.AddString(key, basic)

	f, isFormatter := err.(fmt.Formatter)
	if isFormatter {
		verbose := fmt.Sprintf("%+v", f)
		if verbose != basic {
			// This is a rich error type, like those produced by
			// github.com/pkg/errors.
			l.AddString(key+"Verbose", verbose)
		}
	}
	return nil
}
