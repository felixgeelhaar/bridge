package llm

import (
	"context"
	"time"

	"github.com/felixgeelhaar/fortify/circuitbreaker"
	"github.com/felixgeelhaar/fortify/retry"
	"github.com/felixgeelhaar/fortify/timeout"
)

// ResilientProvider wraps a Provider with resilience patterns.
type ResilientProvider struct {
	provider       Provider
	circuitBreaker circuitbreaker.CircuitBreaker[*CompletionResponse]
	retry          retry.Retry[*CompletionResponse]
	timeout        timeout.Timeout[*CompletionResponse]
	config         ResilientConfig
}

// ResilientConfig configures resilience patterns.
type ResilientConfig struct {
	// Circuit breaker settings
	CBMaxFailures      int           // Max consecutive failures before opening circuit
	CBResetTimeout     time.Duration // Time to wait before attempting to close circuit
	CBHalfOpenRequests int           // Number of requests to allow in half-open state

	// Retry settings
	RetryMaxAttempts  int           // Max retry attempts
	RetryInitialDelay time.Duration // Initial retry delay
	RetryMaxDelay     time.Duration // Maximum retry delay
	RetryMultiplier   float64       // Backoff multiplier

	// Timeout settings
	Timeout time.Duration // Request timeout
}

// DefaultResilientConfig returns sensible defaults for resilience.
func DefaultResilientConfig() ResilientConfig {
	return ResilientConfig{
		CBMaxFailures:      5,
		CBResetTimeout:     30 * time.Second,
		CBHalfOpenRequests: 2,

		RetryMaxAttempts:  3,
		RetryInitialDelay: 500 * time.Millisecond,
		RetryMaxDelay:     10 * time.Second,
		RetryMultiplier:   2.0,

		Timeout: 2 * time.Minute,
	}
}

// NewResilientProvider wraps a provider with resilience patterns.
func NewResilientProvider(provider Provider, cfg ResilientConfig) *ResilientProvider {
	// Circuit breaker
	cb := circuitbreaker.New[*CompletionResponse](circuitbreaker.Config{
		MaxRequests: uint32(cfg.CBHalfOpenRequests),
		Interval:    cfg.CBResetTimeout,
		Timeout:     cfg.CBResetTimeout,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(cfg.CBMaxFailures)
		},
		OnStateChange: func(from, to circuitbreaker.State) {
			// State change logged silently - could add logger later
		},
	})

	// Retry with exponential backoff
	r := retry.New[*CompletionResponse](retry.Config{
		MaxAttempts:  cfg.RetryMaxAttempts,
		InitialDelay: cfg.RetryInitialDelay,
		MaxDelay:     cfg.RetryMaxDelay,
		Multiplier:   cfg.RetryMultiplier,
		Jitter:       true,
		IsRetryable: func(err error) bool {
			if providerErr, ok := err.(*ProviderError); ok {
				return providerErr.IsRetryable()
			}
			return false
		},
		OnRetry: func(attempt int, err error) {
			// Retry logged silently - could add logger later
		},
	})

	// Timeout
	t := timeout.New[*CompletionResponse](timeout.Config{
		DefaultTimeout: cfg.Timeout,
	})

	return &ResilientProvider{
		provider:       provider,
		circuitBreaker: cb,
		retry:          r,
		timeout:        t,
		config:         cfg,
	}
}

// Name returns the wrapped provider name.
func (p *ResilientProvider) Name() string {
	return p.provider.Name()
}

// Models returns the wrapped provider models.
func (p *ResilientProvider) Models() []string {
	return p.provider.Models()
}

// Complete sends a completion request with resilience patterns applied.
func (p *ResilientProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Apply patterns: Timeout -> CircuitBreaker -> Retry -> Provider
	resp, err := p.timeout.Execute(ctx, p.config.Timeout, func(ctx context.Context) (*CompletionResponse, error) {
		return p.circuitBreaker.Execute(ctx, func(ctx context.Context) (*CompletionResponse, error) {
			return p.retry.Do(ctx, func(ctx context.Context) (*CompletionResponse, error) {
				return p.provider.Complete(ctx, req)
			})
		})
	})

	return resp, err
}

// CircuitState returns the current circuit breaker state.
func (p *ResilientProvider) CircuitState() string {
	return p.circuitBreaker.State().String()
}

// Ensure ResilientProvider implements Provider.
var _ Provider = (*ResilientProvider)(nil)
