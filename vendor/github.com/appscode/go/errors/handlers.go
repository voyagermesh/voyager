package errors

import (
	"log"
	"sync"
)

// Defines error-handling middleware that receives the traceableError instance
// and handles it gracefully. Caller do not depends on the response of Handle()
// Handle should resolves everything gracefully, and must prevent panics.
// Multiple Handler can be defined.
// In case of error first all the default handlers will be called and then
// the handler of that instance will be called.
type Handler interface {
	// Handler receives the traceableError instance it was called upon as the
	// first parameter. Handler can parse the error via calling Parse(). Additionally
	// handler can be defined as that can only perform upon specific error condition
	// .Is(error, dtypes.Status) can resolve the error types, and be used to handle only
	// on specific types of error. Handler must explicitly handler those. Caller
	// do not performs those types of actions.
	Handle(error)
}

// Collection of global error handlers. Those will be called on every error
// to handle or do tasks based on errors.
var Handlers = newErrorHandlers()

type errorHandlers struct {
	lock     sync.RWMutex
	handlers []Handler
}

func newErrorHandlers() *errorHandlers {
	return &errorHandlers{
		handlers: make([]Handler, 0),
	}
}

func (e *errorHandlers) Add(handlers ...Handler) {
	e.lock.Lock()
	e.handlers = append(e.handlers, handlers...)
	e.lock.Unlock()
}

// Run the error handlers with traceableError instance
func (e *errorHandlers) run(err error) {
	e.lock.Lock()
	defer e.lock.Unlock()
	runHandlers(err, e.handlers)
}

func handleCrash() {
	if r := recover(); r != nil {
		log.Println("recovered from error handler panic, reason:", r)
	}
}

func runHandlers(err error, handlers []Handler) {
	for _, handler := range handlers {
		go func() {
			defer handleCrash()
			handler.Handle(err)
		}()
	}
}
