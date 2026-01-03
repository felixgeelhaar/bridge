package observability_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/felixgeelhaar/bridge/internal/infrastructure/observability"
)

func TestDefaultConfig(t *testing.T) {
	cfg := observability.DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("expected level 'info', got %s", cfg.Level)
	}

	if cfg.Format != "json" {
		t.Errorf("expected format 'json', got %s", cfg.Format)
	}

	if cfg.Output == nil {
		t.Error("expected non-nil output")
	}
}

func TestDevConfig(t *testing.T) {
	cfg := observability.DevConfig()

	if cfg.Level != "debug" {
		t.Errorf("expected level 'debug', got %s", cfg.Level)
	}

	if cfg.Format != "console" {
		t.Errorf("expected format 'console', got %s", cfg.Format)
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config observability.LogConfig
	}{
		{
			name: "json format",
			config: observability.LogConfig{
				Level:  "info",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "console format",
			config: observability.LogConfig{
				Level:  "debug",
				Format: "console",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "trace level",
			config: observability.LogConfig{
				Level:  "trace",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "debug level",
			config: observability.LogConfig{
				Level:  "debug",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "warn level",
			config: observability.LogConfig{
				Level:  "warn",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "warning level",
			config: observability.LogConfig{
				Level:  "warning",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "error level",
			config: observability.LogConfig{
				Level:  "error",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "fatal level",
			config: observability.LogConfig{
				Level:  "fatal",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "default level for unknown",
			config: observability.LogConfig{
				Level:  "unknown",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "nil output defaults to stdout",
			config: observability.LogConfig{
				Level:  "info",
				Format: "json",
				Output: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := observability.NewLogger(tt.config)
			if logger == nil {
				t.Error("expected non-nil logger")
			}
		})
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := observability.NewDefaultLogger()
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestNewDevLogger(t *testing.T) {
	logger := observability.NewDevLogger()
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestWithLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := observability.NewLogger(observability.LogConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	ctx := context.Background()
	ctx = observability.WithLogger(ctx, logger)

	retrieved := observability.FromContext(ctx)
	if retrieved == nil {
		t.Error("expected non-nil logger from context")
	}
}

func TestFromContext_NoLogger(t *testing.T) {
	ctx := context.Background()
	logger := observability.FromContext(ctx)

	// Should return default logger when none in context
	if logger == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestNewServiceLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := observability.LogConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	}
	fields := observability.LoggerFields{
		Service:     "bridge",
		Version:     "1.0.0",
		Environment: "test",
	}

	logger := observability.NewServiceLogger(cfg, fields)
	if logger == nil {
		t.Error("expected non-nil logger")
	}

	// Log a message to verify fields are included
	logger.Info().Msg("test message")

	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}

func TestWithWorkflowContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := observability.NewLogger(observability.LogConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	workflowLogger := observability.WithWorkflowContext(logger, "wf-123", "run-456")
	if workflowLogger == nil {
		t.Error("expected non-nil logger")
	}

	workflowLogger.Info().Msg("workflow message")

	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}

func TestWithStepContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := observability.NewLogger(observability.LogConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	stepLogger := observability.WithStepContext(logger, "step-123", "analyze")
	if stepLogger == nil {
		t.Error("expected non-nil logger")
	}

	stepLogger.Info().Msg("step message")

	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}

func TestWithAgentContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := observability.NewLogger(observability.LogConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	agentLogger := observability.WithAgentContext(logger, "agent-123", "claude-3-opus")
	if agentLogger == nil {
		t.Error("expected non-nil logger")
	}

	agentLogger.Info().Msg("agent message")

	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}
