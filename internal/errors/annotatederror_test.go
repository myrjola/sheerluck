package errors

import (
	"github.com/stretchr/testify/require"
	"log/slog"
	"slices"
	"testing"
)

func TestAnnotatedError(t *testing.T) {
	err := New("test error", slog.String("id", "123"))
	require.Equal(t, "test error", err.Error())

	// Assert that wrapping sentinel errors work as expected.
	sentinel := NewSentinel("test error")
	require.NotErrorIs(t, err, NewSentinel("test error"))
	wrapped := err.Wrap(sentinel)
	require.ErrorIs(t, wrapped, sentinel)

	// Ensure log values are coming through.
	group := err.LogValue().Group()
	require.Contains(t, group, slog.String("id", "123"))

	// Assert there's a valid source
	sourceIdx := slices.IndexFunc(group, func(attr slog.Attr) bool {
		return attr.Key == "source"
	})
	source := group[sourceIdx]
	require.Contains(t, source.Value.String(), "annotatederror_test.go")
}
