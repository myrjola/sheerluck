package errors

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestAnnotatedError(t *testing.T) {
	annotatedError := New("test error", slog.Int("id", 123))
	require.Equal(t, "test error", annotatedError.Error())

	// Assert that wrapping sentinel errors work as expected.
	sentinel := NewSentinel("test error")
	require.NotErrorIs(t, annotatedError, NewSentinel("test error"))
	wrapped := Join(annotatedError, sentinel)
	require.ErrorIs(t, wrapped, sentinel)
	require.True(t, Is(wrapped, sentinel))
	require.Equal(t, "test error\ntest error", wrapped.Error())

	// Ensure the latest annotations override older ones
	wrapped = Wrap(wrapped, "wrap error 1", slog.String("user", "johndoe"))
	wrapped = Wrap(wrapped, "wrap error 2", slog.Duration("duration", time.Second))
	require.Equal(t, "wrap error 2: wrap error 1: test error\ntest error", wrapped.Error())

	// Assert that we can find the annotated error
	annotated := AnnotatedError{}
	require.True(t, As(wrapped, &annotated))
	require.Equal(t, annotated.Error(), "wrap error 2")

	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, nil))
	l.Info("test", "error", SlogError(wrapped))
	logLine := buf.String()
	require.Contains(t, logLine, "id=123")
	require.Contains(t, logLine, "user=johndoe")
	require.Contains(t, logLine, "duration=1s")
	require.Contains(t, logLine, "annotatederror_test.go:14")
	require.Contains(t, logLine, "annotatederror_test.go:26")
	require.Contains(t, logLine, "annotatederror_test.go:27")
	// Assert we didn't mess up the stack trace skips.
	require.NotContains(t, logLine, "annotatederror.go")

	// Try to break things by passing a nil error and other wonkiness.
	SlogError(Join(nil, nil, NewSentinel("sentinel"), newAnnotatedError("test")))
	SlogError(nil)
	SlogError(fmt.Errorf("test: %w", NewSentinel("sentinel")))
	SlogError(Join(NewSentinel("sentinel1"), NewSentinel("sentinel2")))
	SlogError(Wrap(nil, "wrap error"))
	SlogError(Wrap(errors.Join(nil, nil), "wrap error"))
	_ = Unwrap(Wrap(NewSentinel("sentinel"), "wrap error"))
}
