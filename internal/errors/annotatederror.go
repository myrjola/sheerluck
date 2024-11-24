package errors

import (
	"errors"
	"fmt"
	"iter"
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

// New creates a new [AnnotatedError] with the given message and attributes.
func New(msg string, attrs ...slog.Attr) AnnotatedError {
	return newAnnotatedError(msg, attrs...)
}

// newAnnotatedError is a constructor that ensures that the program counter is set correctly.
//
// It must always be called directly by an exported function or method
// because it uses a fixed call depth to obtain the pc.
func newAnnotatedError(msg string, attrs ...slog.Attr) AnnotatedError {
	var pcs [1]uintptr
	// Skip runtime.Callers, this function, and the function calling this function.
	runtime.Callers(3, pcs[:])
	return AnnotatedError{
		msg:   msg,
		pc:    pcs[0],
		attrs: attrs,
	}
}

// Wrap is a convenience function for wrapping errors, e.g., adding context to a sentinel error.
func Wrap(err error, msg string, attrs ...slog.Attr) error {
	wrapper := newAnnotatedError(msg, attrs...)
	return fmt.Errorf("%w: %w", wrapper, err)
}

// Creates a plain error without other context that can be used as sentinel error that can be detected with errors.Is.
func NewSentinel(msg string) error {
	return errors.New(msg)
}

// Error implements error interface.
func (err AnnotatedError) Error() string {
	return err.msg
}

// AllAnnotated returns an iterator that iterates over all the [AnnotatedError] in err
func AllAnnotated(err error) iter.Seq[AnnotatedError] {
	return func(yield func(annotatedError AnnotatedError) bool) {
		traverseAnnotated(err, yield)
	}
}

// traverseAnnotated recursively yields all the [AnnotatedError] in err.
func traverseAnnotated(err error, yield func(annotatedError AnnotatedError) bool) bool {
	if err == nil {
		return true
	}

	// If the error is an AnnotatedError, yield it and return since it is a leaf node.
	if annotated, ok := err.(AnnotatedError); ok {
		return yield(annotated)
	}

	// Recurse through the wrapped errors accounting for multi errors created with e.g.
	// errors.Join and fmt.Errorf("%w: %w", err1, err2).
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return traverseAnnotated(wrapped, yield)
	}
	multi, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return true
	}
	for _, suberr := range multi.Unwrap() {
		if !traverseAnnotated(suberr, yield) {
			return false
		}
	}

	return true
}

// SlogError compiles the annotations to a slog.Attr for logging.
func SlogError(err error) slog.Attr {
	var (
		attrs           []slog.Attr
		programCounters []uintptr
	)
	for annotated := range AllAnnotated(err) {
		attrs = append(attrs, annotated.attrs...)
		programCounters = append([]uintptr{annotated.pc}, programCounters...)
	}

	if err == nil {
		return slog.String("error", "<nil>")
	}

	// Retrieve the sources of the errors so that developers can locate them faster.
	frames := runtime.CallersFrames(programCounters)
	var sources []string
	for {
		frame, more := frames.Next()
		// file:// prefix is used to make it clickable in Goland terminal.
		sources = append(sources, fmt.Sprintf("file://%s:%d", frame.File, frame.Line))
		if !more {
			break
		}
	}

	return slog.Group("error",
		slog.String("msg", err.Error()),
		slog.Any("sources", sources),
		slog.Any("annotations", slog.GroupValue(attrs...)),
	)
}

// As exposes stdlib [errors.As].
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Is exposes stdlib [errors.Is].
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Unwrap exposes stdlib [errors.Unwrap].
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Join exposes stdlib [errors.Join].
func Join(errs ...error) error {
	return errors.Join(errs...)
}
