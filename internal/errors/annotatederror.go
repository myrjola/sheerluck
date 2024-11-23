package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
)

// AnnotatedError includes more context than a plain error that is useful for troubleshooting.
type AnnotatedError struct {
	// msg is the error message.
	msg string
	// pc is the program counter for the location of the error provided by runtime.Callers.
	pc uintptr
	// attrs are slog attributes that are added to the log event to provide more context for the error.
	attrs []slog.Attr
}

// New creates a new AnnotatedError with the given message and attributes.
func New(msg string, attrs ...slog.Attr) AnnotatedError {
	var pcs [1]uintptr
	// Skip runtime.Callers and this function.
	runtime.Callers(2, pcs[:])
	return AnnotatedError{
		msg:   msg,
		pc:    pcs[0],
		attrs: attrs,
	}
}

// Creates a plain error without other context that can be used as sentinel error that can be detected with errors.Is.
func NewSentinel(msg string) error {
	return errors.New(msg)
}

// Wrap is a convenience function for wrapping errors, e.g., adding context to a sentinel error.
func (wrapper AnnotatedError) Wrap(err error) error {
	return fmt.Errorf("%w: %w", wrapper, err)
}

// Error implements error interface.
func (err AnnotatedError) Error() string {
	return err.msg
}

// LogValue formats the error for useful logging.
func (err AnnotatedError) LogValue() slog.Value {
	// Retrieve the source location of the error so that developers can locate it faster.
	frames := runtime.CallersFrames([]uintptr{err.pc})
	source, _ := frames.Next()
	sourceAttr := slog.String("source", fmt.Sprintf("%s:%d", source.File, source.Line))

	attrs := append(
		[]slog.Attr{sourceAttr},
		err.attrs...,
	)

	return slog.GroupValue(attrs...)
}

// As exposes stdlib errors.As.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Is exposes stdlib errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Unwrap exposes stdlib errors.Unwrap.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}
