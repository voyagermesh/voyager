package errors

import (
	"context"
	"strings"
	"sync"

	"github.com/facebookgo/stack"
)

// Basic error codes supported by appscode.
const (
	BadRequest                    = "BADREQUEST"
	External                      = "EXTERNAL"
	Failed                        = "FAILED"
	Internal                      = "INTERNAL"
	InvalidData                   = "INVALIED_DATA"
	InvalidPaymentInformation     = "INVALID_PAYMENT_INFORMATION"
	InvalidQuota                  = "INVALID_QUOTA"
	NotFound                      = "NOT_FOUND"
	PaymentInformationUnavailable = "PAYMENT_INFORMATION_UNAVAILABLE"
	PermissionDenied              = "PERMISSION_DENIED"
	QuotaLimitExceed              = "QUOTA_LIMIT_EXCEED"
	Unauthorized                  = "UNAUTHORIZED"
	Unimplemented                 = "UNIMPLEMENTED"
	Unknown                       = "UNKNOWN_ERROR"
)

// traceableError contains the error types that holds the errors
// operations performs with. It also contains the additional help
// messages and the underlying contexts that provide additional
// support with the error.
type traceableError struct {
	// represents the error error code as string literal
	code string

	// User facing error message that contains all the messages in order
	messages []string

	// Underlying original error
	err error

	// Stacktrace
	trace *stack.Multi

	// Additional help messages
	help *Help

	// Formatter function formats the underlying message collection
	// as string message that can be used for Error().
	formatter formatterFunc

	// Context values that this error holds to
	context Context

	// Custom error handlers that work with
	handler []Handler

	// Mutex lock to provide thread safety.
	lock sync.Mutex

	handled bool
}

// Provides links to documentation or for performing an out of band action.
type Help struct {
	// Describe what link offers
	Description string `json:"description,omitempty"`
	// The URL of The link.
	Url string `json:"url,omitempty"`
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
		if err.err != nil {
			return err.code + ":" + err.err.Error()
		}
		if len(err.messages) > 0 {
			return err.code + ":" + strings.Join(err.messages, ";")
		}
		return err.code
	}
	if e != nil {
		return e.Error()
	}
	return ""
}

// Context stands for an context object that holds additional information
// regarding the error an can be used to parse the values.
type Context interface {
	// String returns the string value associated with this context or empty
	// string. Successive calls to Value returns the same result.
	String() string
}

// Error satisfies the error interface
func (t *traceableError) Error() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.formatter(t)
}

// Error satisfies the error interface
func (t *traceableError) Handled() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.handled
}

func (t *traceableError) Code() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.code
}

func (t *traceableError) Message() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	last := len(t.messages) - 1
	if last >= 0 {
		return t.messages[last]
	}
	return ""
}

func (t *traceableError) Messages() []string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.messages
}

// Returns underlying original error
func (t *traceableError) Err() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.err
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

func (t *traceableError) Help() *Help {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.help
}

func (t *traceableError) Context() Context {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.context
}

type goContextWrapper struct {
	c           context.Context
	contextFunc func(c context.Context) string
}

func (g *goContextWrapper) String() string {
	return g.contextFunc(g.c)
}
