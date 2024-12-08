package logging

import (
	"context"
	"log/slog"
)

type contextKey string

const slogAttrs contextKey = "slogAttrs"

type ContextHandler struct {
	slog.Handler
}

// NewContextHandler constructs a ContextHandler that adds new [slog.Attr] to the log messages from [context.Context]
// to the underlying [slog.Handler].
func NewContextHandler(h slog.Handler) ContextHandler {
	return ContextHandler{Handler: h}
}

// Handle enriches the log record with [slog.Attr] stored in context with [WithAttrs].
func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogAttrs).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs adds [...slog.Attr] to the [context.Context] that enriches the log messages handled by [ContextHandler].
func WithAttrs(ctx context.Context, attr ...slog.Attr) context.Context {
	if v, ok := ctx.Value(slogAttrs).([]slog.Attr); ok {
		v = append(v, attr...)
		return context.WithValue(ctx, slogAttrs, v)
	}
	return context.WithValue(ctx, slogAttrs, attr)
}