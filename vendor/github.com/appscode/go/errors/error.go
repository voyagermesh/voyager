// Package errors implements functions to manipulate custom errors
package errors

import (
	gtx "context"
	"fmt"

	"github.com/facebookgo/stack"
	"strings"
)

// New returns a Status representing c and msg.
func New(msg ...string) *traceableError {
	return &traceableError{
		msg:       strings.Join(msg, " "),
		formatter: defaultFormatter,
		trace:     stack.CallersMulti(1),
	}
}

// Newf returns New(c, fmt.Sprintf(format, a...)).
func Newf(format string, a ...interface{}) *traceableError {
	return &traceableError{
		msg:       fmt.Sprintf(format, a...),
		formatter: defaultFormatter,
		trace:     stack.CallersMulti(1),
	}
}

// Parse and returns an traceableError object to access its contents
// If the error kind is not traceableError then wrap the error into
// traceableError. Other methods can be called upon the returned values.
func FromErr(e error) *traceableError {
	if e == nil {
		// error instance is nil. Returning nil value.
		return nil
	}
	if t, ok := e.(*traceableError); ok {
		return t
	}
	return &traceableError{
		cause:     e,
		formatter: defaultFormatter,
		trace:     stack.CallersMulti(2),
	}
}

func (t *traceableError) WithCause(err error) *traceableError {
	t.lock.Lock()
	defer t.lock.Unlock()
	if e, ok := err.(*traceableError); ok {
		e.lock.Lock()
		defer e.lock.Unlock()

		// copies the provided error values to new instance to stay safe.
		// mutex lock for t remains same.
		*t, t.lock = *e, t.lock
		return t
	}
	t.cause = err
	return t
}

func (t *traceableError) WithContext(c gtx.Context) *traceableError {
	t.lock.Lock()
	t.ctx = c
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithMessage(msg string) *traceableError {
	t.lock.Lock()
	t.msg = msg
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithMessagef(format string, msg ...interface{}) *traceableError {
	t.lock.Lock()
	t.msg = fmt.Sprintf(format, msg...)
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithFormatter(f formatterFunc) *traceableError {
	t.lock.Lock()
	t.formatter = f
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithHandler(h ...Handler) *traceableError {
	t.lock.Lock()
	t.handler = append(t.handler, h...)
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithTrace() *traceableError {
	t.lock.Lock()
	t.trace.AddCallers(1)
	t.lock.Unlock()
	return t
}

func (t *traceableError) Err() error {
	if !t.handled {
		t.lock.Lock()
		t.handled = true
		t.lock.Unlock()
		// Running all default error handlers
		Handlers.run(t)
		// Running error handlers with regarding this instances.
		runHandlers(t, t.handler)
	}
	return t
}
