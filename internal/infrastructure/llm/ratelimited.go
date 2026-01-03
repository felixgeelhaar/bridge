package llm

import (
	"context"
	"sync"
	"time"

	"github.com/felixgeelhaar/bolt"
)

// RateLimitedProvider wraps a Provider with rate limiting.
type RateLimitedProvider struct {
	provider Provider
	logger   *bolt.Logger
	limiter  *tokenBucket
}

// RateLimitConfig configures rate limiting.
type RateLimitConfig struct {
	RequestsPerMinute int
	TokensPerMinute   int
	BurstSize         int
}

// DefaultRateLimitConfig returns sensible defaults for rate limiting.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 60,
		TokensPerMinute:   100000,
		BurstSize:         10,
	}
}

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket(requestsPerMinute int, burstSize int) *tokenBucket {
	maxTokens := float64(burstSize)
	if burstSize == 0 {
		maxTokens = float64(requestsPerMinute) / 6 // Allow 10 seconds of burst
	}

	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: float64(requestsPerMinute) / 60.0, // Convert to per-second
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) wait(ctx context.Context) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	// Check if we have tokens available
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return nil
	}

	// Calculate wait time
	waitDuration := time.Duration((1.0 - tb.tokens) / tb.refillRate * float64(time.Second))

	tb.mu.Unlock()

	// Wait with context
	select {
	case <-ctx.Done():
		tb.mu.Lock()
		return ctx.Err()
	case <-time.After(waitDuration):
		tb.mu.Lock()
		tb.tokens = 0
		return nil
	}
}

// NewRateLimitedProvider wraps a provider with rate limiting.
func NewRateLimitedProvider(provider Provider, logger *bolt.Logger, cfg RateLimitConfig) *RateLimitedProvider {
	return &RateLimitedProvider{
		provider: provider,
		logger:   logger,
		limiter:  newTokenBucket(cfg.RequestsPerMinute, cfg.BurstSize),
	}
}

// Name returns the wrapped provider name.
func (p *RateLimitedProvider) Name() string {
	return p.provider.Name()
}

// Models returns the wrapped provider models.
func (p *RateLimitedProvider) Models() []string {
	return p.provider.Models()
}

// Complete sends a completion request with rate limiting applied.
func (p *RateLimitedProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Wait for rate limit
	if err := p.limiter.wait(ctx); err != nil {
		p.logger.Warn().
			Str("provider", p.provider.Name()).
			Err(err).
			Msg("Rate limit wait cancelled")
		return nil, err
	}

	return p.provider.Complete(ctx, req)
}

// Ensure RateLimitedProvider implements Provider.
var _ Provider = (*RateLimitedProvider)(nil)

// ProviderFactory creates providers with all resilience patterns applied.
type ProviderFactory struct {
	logger       *bolt.Logger
	resilientCfg ResilientConfig
	rateLimitCfg RateLimitConfig
}

// NewProviderFactory creates a new provider factory.
func NewProviderFactory(logger *bolt.Logger) *ProviderFactory {
	return &ProviderFactory{
		logger:       logger,
		resilientCfg: DefaultResilientConfig(),
		rateLimitCfg: DefaultRateLimitConfig(),
	}
}

// WithResilientConfig sets the resilient configuration.
func (f *ProviderFactory) WithResilientConfig(cfg ResilientConfig) *ProviderFactory {
	f.resilientCfg = cfg
	return f
}

// WithRateLimitConfig sets the rate limit configuration.
func (f *ProviderFactory) WithRateLimitConfig(cfg RateLimitConfig) *ProviderFactory {
	f.rateLimitCfg = cfg
	return f
}

// CreateAnthropic creates an Anthropic provider with all resilience patterns.
func (f *ProviderFactory) CreateAnthropic(apiKey string, model string) Provider {
	base := NewAnthropicProvider(AnthropicConfig{
		ProviderConfig: ProviderConfig{
			APIKey: apiKey,
			Model:  model,
		},
	})
	resilient := NewResilientProvider(base, f.resilientCfg)
	return NewRateLimitedProvider(resilient, f.logger, f.rateLimitCfg)
}

// CreateOpenAI creates an OpenAI provider with all resilience patterns.
func (f *ProviderFactory) CreateOpenAI(apiKey string, model string) Provider {
	base := NewOpenAIProvider(OpenAIConfig{
		ProviderConfig: ProviderConfig{
			APIKey: apiKey,
			Model:  model,
		},
	})
	resilient := NewResilientProvider(base, f.resilientCfg)
	return NewRateLimitedProvider(resilient, f.logger, f.rateLimitCfg)
}

// CreateGemini creates a Gemini provider with all resilience patterns.
func (f *ProviderFactory) CreateGemini(apiKey string, model string) Provider {
	base := NewGeminiProvider(GeminiConfig{
		ProviderConfig: ProviderConfig{
			APIKey: apiKey,
			Model:  model,
		},
	})
	resilient := NewResilientProvider(base, f.resilientCfg)
	return NewRateLimitedProvider(resilient, f.logger, f.rateLimitCfg)
}

// CreateOllama creates an Ollama provider with all resilience patterns.
func (f *ProviderFactory) CreateOllama(baseURL string, model string) Provider {
	base := NewOllamaProvider(OllamaConfig{
		ProviderConfig: ProviderConfig{
			BaseURL: baseURL,
			Model:   model,
		},
	})
	resilient := NewResilientProvider(base, f.resilientCfg)
	// No rate limiting for local Ollama
	return resilient
}
