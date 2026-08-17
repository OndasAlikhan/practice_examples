package circuit_breaker

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrOpenState = errors.New("error open state")

const defaultRetryCount = 5
const waitTime time.Duration = time.Second * 30

type Handler[T any] func() (T, error)

type CircuitBreaker[T any] struct {
	currentRetries int
	skipUntil      time.Time
	state          State
	mux            sync.RWMutex
}

func New[T any]() *CircuitBreaker[T] {
	return &CircuitBreaker[T]{
		skipUntil:      time.Now(),
		currentRetries: defaultRetryCount,
		state:          closed,
	}
}

func (cb *CircuitBreaker[T]) Run(handler Handler[T]) (result T, err error) {
	err = cb.beforeRequest()
	if err != nil {
		var defaultVal T
		return defaultVal, err
	}

	result, err = handler()
	return cb.afterRequest(result, err)
}

func (cb *CircuitBreaker[T]) beforeRequest() error {
	cb.mux.Lock()
	defer cb.mux.Unlock()

	// if skip time return error
	if time.Now().Before(cb.skipUntil) {
		return ErrOpenState
	}

	// if not skip time, but open - set halfOpen
	if cb.state.IsOpen() {
		cb.state = halfOpen
	}

	if cb.state.IsHalfOpen() {
		cb.currentRetries = 1
	}

	return nil
}

func (cb *CircuitBreaker[T]) afterRequest(result T, err error) (T, error) {
	cb.mux.Lock()
	defer cb.mux.Unlock()

	if err != nil {
		cb.currentRetries -= 1

		if cb.currentRetries == 0 {
			cb.skipUntil = time.Now().Add(waitTime)
			cb.state = open

			var defVal T
			return defVal, fmt.Errorf("%w: %w", ErrOpenState, err)
		}

		var defVal T
		return defVal, err
	}

	if cb.state.IsHalfOpen() {
		cb.state = closed
	}

	if cb.currentRetries != defaultRetryCount {
		cb.currentRetries = defaultRetryCount
	}

	return result, nil
}
