package middleware

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shrek82/jorm/core"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreakerMiddleware struct {
	Threshold    int           // Number of failures before opening
	ResetTimeout time.Duration // Time to wait before half-open

	mu             sync.Mutex
	state          State
	failures       int
	lastFailure    time.Time
	halfOpenPassed bool
}

func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		Threshold:    threshold,
		ResetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

func (m *CircuitBreakerMiddleware) Name() string {
	return "CircuitBreaker"
}

func (m *CircuitBreakerMiddleware) Init(db *core.DB) error {
	return nil
}

func (m *CircuitBreakerMiddleware) Shutdown() error {
	return nil
}

func (m *CircuitBreakerMiddleware) Process(ctx context.Context, query *core.Query, next core.QueryFunc) (*core.Result, error) {
	m.mu.Lock()
	switch m.state {
	case StateOpen:
		if time.Since(m.lastFailure) > m.ResetTimeout {
			m.state = StateHalfOpen
			m.halfOpenPassed = false // Reset for new half-open attempt
		} else {
			m.mu.Unlock()
			return &core.Result{Error: ErrCircuitOpen}, ErrCircuitOpen
		}
	case StateHalfOpen:
		if m.halfOpenPassed {
			// Already one request in flight or processed, reject others until we know result?
			// Simple implementation: allow one, reject others.
			m.mu.Unlock()
			return &core.Result{Error: ErrCircuitOpen}, ErrCircuitOpen
		}
		m.halfOpenPassed = true
	}
	m.mu.Unlock()

	res, err := next(ctx, query)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		m.recordFailure()
	} else {
		m.recordSuccess()
	}

	return res, err
}

func (m *CircuitBreakerMiddleware) recordFailure() {
	m.failures++
	m.lastFailure = time.Now()

	if m.state == StateClosed {
		if m.failures >= m.Threshold {
			m.state = StateOpen
		}
	} else if m.state == StateHalfOpen {
		m.state = StateOpen
		m.halfOpenPassed = false
	}
}

func (m *CircuitBreakerMiddleware) recordSuccess() {
	if m.state == StateHalfOpen {
		m.state = StateClosed
		m.failures = 0
		m.halfOpenPassed = false
	} else if m.state == StateClosed {
		// Optional: reset failures on success? 
		// Usually we reset only on consecutive successes or after some time.
		// For simple breaker, let's reset on any success in Closed state to track CONSECUTIVE failures.
		m.failures = 0
	}
}
