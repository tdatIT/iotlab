// Package httpclient wraps resty with timeouts, retries and an optional circuit breaker for outbound REST calls.
package httpclient

import (
	"net"
	"net/http"
	"time"

	"resty.dev/v3"
)

// Caller wraps a resty client and exposes MakeRequest() to build outbound requests.
type Caller interface {
	MakeRequest() *resty.Request
	GetClient() *resty.Client
}

// CBConfig configures resty's built-in (v3) circuit breaker. It is applied only when Config.CircuitBreaker is non-nil.
type CBConfig struct {
	// FailureThreshold is the number of failures that opens the circuit.
	FailureThreshold uint64
	// SuccessThreshold is the number of consecutive successful probes in the half-open state needed to close the circuit again.
	SuccessThreshold uint64
	// ResetTimeout is how long the circuit stays open before moving to half-open.
	ResetTimeout time.Duration
	// Policies decide whether a response counts as a failure. Empty defaults to resty.CircuitBreaker5xxPolicy (HTTP status >= 500).
	Policies []resty.CircuitBreakerPolicy
}

// Config configures an outbound HTTP caller. A nil CircuitBreaker disables the breaker.
type Config struct {
	BaseURL        string
	Timeout        time.Duration
	KeepAlive      time.Duration // idle connection keep-alive duration
	RetryCount     int
	RetryWait      time.Duration // wait between retries on error
	RetryCondition resty.RetryConditionFunc
	Debug          bool
	CircuitBreaker *CBConfig // nil disables the circuit breaker
}

type caller struct {
	client *resty.Client
}

// New builds a Caller from cfg, wiring retries, keep-alive and the optional circuit breaker.
func New(cfg Config) Caller {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
		IdleConnTimeout: cfg.KeepAlive,
	}

	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTimeout(cfg.Timeout).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(cfg.RetryWait).
		SetTransport(transport).
		SetDebug(cfg.Debug)

	if cfg.RetryCondition != nil {
		client.AddRetryConditions(cfg.RetryCondition)
	}

	if cb := cfg.CircuitBreaker; cb != nil {
		client.SetCircuitBreaker(resty.NewCircuitBreakerCount(
			cb.FailureThreshold,
			cb.SuccessThreshold,
			cb.ResetTimeout,
			cb.Policies...,
		))
	}

	return &caller{client: client}
}

func (c *caller) MakeRequest() *resty.Request {
	return c.client.R()
}

func (c *caller) GetClient() *resty.Client {
	return c.client
}
