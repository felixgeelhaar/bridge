package observability

import (
	"context"
	"io"
	"os"

	"github.com/felixgeelhaar/bolt"
)

// LogConfig contains logger configuration.
type LogConfig struct {
	Level  string
	Format string // "json" or "console"
	Output io.Writer
}

// DefaultConfig returns the default logger configuration.
func DefaultConfig() LogConfig {
	return LogConfig{
		Level:  "info",
		Format: "json",
		Output: os.Stdout,
	}
}

// DevConfig returns a development logger configuration.
func DevConfig() LogConfig {
	return LogConfig{
		Level:  "debug",
		Format: "console",
		Output: os.Stdout,
	}
}

// NewLogger creates a new bolt logger with the given configuration.
func NewLogger(cfg LogConfig) *bolt.Logger {
	var handler bolt.Handler

	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	switch cfg.Format {
	case "console":
		handler = bolt.NewConsoleHandler(cfg.Output)
	default:
		handler = bolt.NewJSONHandler(cfg.Output)
	}

	logger := bolt.New(handler)

	// Set log level
	switch cfg.Level {
	case "trace":
		logger = logger.SetLevel(bolt.TRACE)
	case "debug":
		logger = logger.SetLevel(bolt.DEBUG)
	case "warn", "warning":
		logger = logger.SetLevel(bolt.WARN)
	case "error":
		logger = logger.SetLevel(bolt.ERROR)
	case "fatal":
		logger = logger.SetLevel(bolt.FATAL)
	default:
		logger = logger.SetLevel(bolt.INFO)
	}

	return logger
}

// NewDefaultLogger creates a logger with default configuration.
func NewDefaultLogger() *bolt.Logger {
	return NewLogger(DefaultConfig())
}

// NewDevLogger creates a logger for development.
func NewDevLogger() *bolt.Logger {
	return NewLogger(DevConfig())
}

// contextKey is the key for storing logger in context.
type contextKey struct{}

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger *bolt.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the logger from context.
// Returns a default logger if none is found.
func FromContext(ctx context.Context) *bolt.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*bolt.Logger); ok {
		return logger
	}
	return NewDefaultLogger()
}

// LoggerFields are common fields that can be added to a logger.
type LoggerFields struct {
	Service     string
	Version     string
	Environment string
}

// NewServiceLogger creates a logger with service-specific fields.
func NewServiceLogger(cfg LogConfig, fields LoggerFields) *bolt.Logger {
	logger := NewLogger(cfg)

	return logger.With().
		Str("service", fields.Service).
		Str("version", fields.Version).
		Str("environment", fields.Environment).
		Logger()
}

// WithWorkflowContext adds workflow-specific fields to the logger.
func WithWorkflowContext(logger *bolt.Logger, workflowID, runID string) *bolt.Logger {
	return logger.With().
		Str("workflow_id", workflowID).
		Str("run_id", runID).
		Logger()
}

// WithStepContext adds step-specific fields to the logger.
func WithStepContext(logger *bolt.Logger, stepID, stepName string) *bolt.Logger {
	return logger.With().
		Str("step_id", stepID).
		Str("step_name", stepName).
		Logger()
}

// WithAgentContext adds agent-specific fields to the logger.
func WithAgentContext(logger *bolt.Logger, agentID, model string) *bolt.Logger {
	return logger.With().
		Str("agent_id", agentID).
		Str("model", model).
		Logger()
}
