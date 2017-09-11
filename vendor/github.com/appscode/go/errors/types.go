package errors

import (
	"bytes"
	gtx "context"
	"sync"

	"github.com/facebookgo/stack"
)

// traceableError contains the error types that holds the errors
// operations performs with. It also contains the additional help
// messages and the underlying contexts that provide additional
// support with the error.
type traceableError struct {
	// User facing error message that contains all the messages in order
	msg string

	// Underlying original error
	cause error

	// Stacktrace
	trace *stack.Multi

	// Formatter function formats the underlying message collection
	// as string message that can be used for Error().
	formatter formatterFunc

	// Context values that this error holds to
	ctx gtx.Context

	// Custom error handlers that work with
	handler []Handler

	handled bool

	// Mutex lock to provide thread safety.
	lock sync.Mutex
}

// Formatter functions format error for the error types satisfying
type formatterFunc func(error) string

// defaultFormatter is the formatter that provides default formatting
// of Error() returned values. It accepts the underlying error value and
// the message collection throughout the spec and return an string providing
// required formatting upon the parameter. This can be override on each error
// call via WithFormatter() call, that will overrides the defaultFormatter to
// a user provided formatter
var defaultFormatter = func(e error) string {
	if err, ok := e.(*traceableError); ok {
		var buf bytes.Buffer
		buf.WriteString(err.msg)
		if err.cause != nil {
			buf.WriteString("\nCaused By:\n")
			buf.WriteString(err.cause.Error())
		}
		if err.trace != nil {
			buf.WriteString("\nStack trace:\n")
			buf.WriteString(err.trace.String())
		}
		return buf.String()
	}
	if e != nil {
		return e.Error()
	}
	return ""
}

// Error satisfies the error interface
func (t *traceableError) Error() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.formatter(t)
}

func (t *traceableError) Message() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.msg
}

// Returns underlying original error
func (t *traceableError) Cause() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.cause
}

func (t *traceableError) Trace() *stack.Multi {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.trace
}

func (t *traceableError) TraceString() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.trace.String()
}

func (t *traceableError) Context() gtx.Context {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.ctx
}
