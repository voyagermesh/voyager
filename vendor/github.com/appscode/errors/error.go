// Package errors implements functions to manipulate custom errors
package errors

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/appscode/log"
	"github.com/facebookgo/stack"
)

// NewGoError returns an go defined error that formats as the given text.
func NewGoError(message string) error {
	return errors.New(message)
}

func NewGoErrorf(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}

func New(message ...string) *traceableError {
	return newTraceableError(nil, message...)
}

// Parse and returns an traceableError object to access its contents
// If the error kind is not traceableError then wrap the error into
// traceableError. Other methods can be called upon the returned values.
func Parse(e error) *traceableError {
	if e == nil {
		// error instance is nil. Returning nil value.
		return nil
	}
	return newTraceableError(e)
}

func newTraceableError(err error, message ...string) *traceableError {
	// Checking if provided error is already an traceableError wrapped
	// returning the instance.
	if t, ok := err.(*traceableError); ok {
		t.messages = append(t.messages, message...)
		return t
	}
	e := &traceableError{
		err:       err,
		messages:  make([]string, 0),
		formatter: defaultFormatter,
		trace:     stack.CallersMulti(2),
	}
	e.messages = append(e.messages, message...)
	return e
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
	t.err = err
	return t
}

func (t *traceableError) WithGoContext(c context.Context, f func(context.Context) string) *traceableError {
	t.lock.Lock()
	t.context = &goContextWrapper{
		c:           c,
		contextFunc: f,
	}
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithContext(c Context) *traceableError {
	t.lock.Lock()
	t.context = c
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithMessage(msg ...string) *traceableError {
	t.lock.Lock()
	t.messages = append(t.messages, msg...)
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithMessagef(format string, msg ...interface{}) *traceableError {
	t.lock.Lock()
	t.messages = append(t.messages, fmt.Sprintf(format, msg...))
	t.lock.Unlock()
	return t
}

func (t *traceableError) WithHelp(url, desc string) *traceableError {
	t.lock.Lock()
	t.help = &Help{
		Url:         url,
		Description: desc,
	}
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

// Creates an error with specific status code
func (t *traceableError) Status(code string) error {
	return t.do(code)
}

func (t *traceableError) BadRequest() error {
	return t.do(BadRequest)
}

func (t *traceableError) External() error {
	return t.do(External)
}

func (t *traceableError) Failed() error {
	return t.do(Failed)
}

func (t *traceableError) Internal() error {
	return t.do(Internal)
}

func (t *traceableError) InvalidData() error {
	return t.do(InvalidData)
}

func (t *traceableError) InvalidPaymentInformation() error {
	return t.do(InvalidPaymentInformation)
}

func (t *traceableError) InvalidQuota() error {
	return t.do(InvalidQuota)
}

func (t *traceableError) NotFound() error {
	return t.do(NotFound)
}

func (t *traceableError) PaymentInformationUnavailable() error {
	return t.do(PaymentInformationUnavailable)
}

func (t *traceableError) PermissionDenied() error {
	return t.do(PermissionDenied)
}

func (t *traceableError) QuotaLimitExceed() error {
	return t.do(QuotaLimitExceed)
}

func (t *traceableError) Unauthorized() error {
	return t.do(Unauthorized)
}

func (t *traceableError) Unimplemented() error {
	return t.do(Unimplemented)
}

func (t *traceableError) Unknown() error {
	return t.do(Unknown)
}

func (t *traceableError) do(code string) error {
	// Ensure error to be always have the code that for the first time
	// initializations
	if !t.handled {
		t.lock.Lock()
		t.code = code
		// Set error is Built
		t.handled = true

		// Unlocks t for using inside the Handlers. Prevents Deadlocks.
		t.lock.Unlock()

		// Running all default error handlers
		Handlers.run(t)

		// Running error handlers with regarding this instances.
		runHandlers(t, t.handler)
	}
	return t
}

// Checks an error, Is returns true if the error is of the kind of the code
func Is(err error, code string) bool {
	return reasonForError(err) == code
}

func IsBadRequest(err error) bool {
	return reasonForError(err) == BadRequest
}

func IsExternal(err error) bool {
	return reasonForError(err) == External
}

func IsFailed(err error) bool {
	return reasonForError(err) == Failed
}

func IsInternal(err error) bool {
	return reasonForError(err) == Internal
}

func IsInvalidData(err error) bool {
	return reasonForError(err) == InvalidData
}

func IsInvalidPaymentInformation(err error) bool {
	return reasonForError(err) == InvalidPaymentInformation
}

func IsInvalidQuota(err error) bool {
	return reasonForError(err) == InvalidQuota
}

func IsNotFound(err error) bool {
	return reasonForError(err) == NotFound
}

func IsPaymentInformationUnavailable(err error) bool {
	return reasonForError(err) == PaymentInformationUnavailable
}

func IsPermissionDenied(err error) bool {
	return reasonForError(err) == PermissionDenied
}

func IsQuotaLimitExceed(err error) bool {
	return reasonForError(err) == QuotaLimitExceed
}

func IsUnauthorized(err error) bool {
	return reasonForError(err) == Unauthorized
}

func IsUnimplemented(err error) bool {
	return reasonForError(err) == Unimplemented
}

func IsUnknown(err error) bool {
	return reasonForError(err) == Unknown
}

func reasonForError(err error) string {
	if err == nil {
		return Unknown
	}

	if e, ok := err.(*traceableError); ok {
		e.lock.Lock()
		defer e.lock.Unlock()
		return e.code
	}
	return Unknown
}

func Exit(e error) {
	if e != nil {
		if err := Parse(e); err != nil {
			if err.Err() != nil {
				log.Error("Error:", err.Code()+" "+err.Err().Error())
			}
			if len(err.Messages()) > 0 {
				log.Error(err.Messages())
			}
			if err.Context() != nil {
				log.Error(err.Context().String())
			}
			log.Error("Trace\n", err.TraceString(), "\n")
		} else {
			log.Error("Error:", e.Error())
		}
		os.Exit(1)
	}
}

func Log(e error) bool {
	if e != nil {
		if err := Parse(e); err != nil {
			if err.Err() != nil {
				log.Error("Error:", err.Code()+" "+err.Err().Error())
			}
			if len(err.Messages()) > 0 {
				log.Error(err.Messages())
			}
			if err.Context() != nil {
				log.Error(err.Context().String())
			}
			log.Error("Trace\n", err.TraceString(), "\n")
		} else {
			log.Error("Error:", e.Error())
		}
		return true
	}
	return false
}
