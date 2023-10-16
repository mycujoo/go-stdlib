package slogtest

// Copyright 2023 Jussi Kalliokoski
//
// Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// ErrorHandler is used for capturing errors from a slog.Handler as slog.Logger
// swallows them.
type ErrorHandler struct {
	inner        slog.Handler
	errorCapture *errorCapture
}

// NewErrorHandler returns a new ErrorHandler.
func NewErrorHandler(inner slog.Handler) *ErrorHandler {
	return &ErrorHandler{
		inner:        inner,
		errorCapture: &errorCapture{},
	}
}

// NewWithErrorHandler wraps the passed slog.Handler into an ErrorHandler,
// creates a new slog.Logger, then returns the Logger and the ErrorHandler.
func NewWithErrorHandler(h slog.Handler) (*slog.Logger, *ErrorHandler) {
	errorHandler := NewErrorHandler(h)
	logger := slog.New(errorHandler)
	return logger, errorHandler
}

// Enabled implements slog.Handler.
func (h *ErrorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *ErrorHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.errorCapture.capture(h.inner.Handle(ctx, r))
}

// WithAttrs implements slog.Handler.
func (h *ErrorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ErrorHandler{
		inner:        h.inner.WithAttrs(attrs),
		errorCapture: h.errorCapture,
	}
}

// WithGroup implements slog.Handler.
func (h *ErrorHandler) WithGroup(name string) slog.Handler {
	return &ErrorHandler{
		inner:        h.inner.WithGroup(name),
		errorCapture: h.errorCapture,
	}
}

// Err returns the captured error(s).
func (h *ErrorHandler) Err() error {
	return h.errorCapture.Err()
}

type errorCapture struct {
	m   sync.Mutex
	err error
}

func (c *errorCapture) capture(err error) error {
	c.m.Lock()
	defer c.m.Unlock()
	c.err = errors.Join(c.err, err)
	return err
}

func (c *errorCapture) Err() error {
	c.m.Lock()
	defer c.m.Unlock()
	return c.err
}
