package circuit_breaker

import (
	"errors"
	"sync"
	"time"
)

var ErrOpenState = errors.New("error open state")

const defaultRetryCount = 5
const waitTime time.Duration = time.Second * 30

type Handler func() (any, error)

type CircuitBreaker struct {
	currentRetries int
	skipUntil      time.Time
	state          State
	mux            sync.RWMutex
}

func New() *CircuitBreaker {
	return &CircuitBreaker{
		skipUntil:      time.Now(),
		currentRetries: defaultRetryCount,
		state:          closed,
	}
}

func (cb *CircuitBreaker) Run(handler Handler) (result any, err error) {
	cb.mux.RLock()
	// if skip time return error
	if time.Now().Before(cb.skipUntil) {
		cb.mux.RUnlock()
		return nil, ErrOpenState
	}

	// if not skip time, but open - set halfOpen
	if cb.state.IsOpen() {
		cb.mux.Lock()
		cb.state = halfOpen
		cb.mux.Unlock()
	}

	if cb.state.IsHalfOpen() {
		cb.mux.Lock()
		cb.currentRetries = 1
		cb.mux.Unlock()
	}

	result, err = handler()
	if err != nil {
		cb.mux.Lock()
		cb.currentRetries -= 1

		if cb.currentRetries == 0 {
			cb.skipUntil = time.Now().Add(waitTime)
			cb.state = open
		}
		cb.mux.Unlock()
	}

	if cb.state.IsHalfOpen() {
		cb.mux.Lock()
		cb.state = closed
		cb.mux.Unlock()
	}

	if cb.currentRetries != defaultRetryCount {
		cb.mux.Lock()
		cb.currentRetries = defaultRetryCount
		cb.mux.Unlock()
	}

	cb.mux.RUnlock()

	return result, nil
}
