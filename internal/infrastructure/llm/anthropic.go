package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude.
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     ProviderConfig
}

// AnthropicConfig contains Anthropic-specific configuration.
type AnthropicConfig struct {
	ProviderConfig
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg AnthropicConfig) *AnthropicProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = anthropicAPIURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	return &AnthropicProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config: cfg.ProviderConfig,
	}
}

// Name returns the provider name.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Models returns available Anthropic models.
func (p *AnthropicProvider) Models() []string {
	return []string{
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

// anthropicRequest is the request structure for Anthropic API.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Stop        []string           `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// anthropicResponse is the response structure from Anthropic API.
type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Error   anthropicErrorDetail `json:"error"`
}

type anthropicErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete sends a completion request to Anthropic.
func (p *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	// Build request
	apiReq := anthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		System:      req.SystemPrompt,
		Messages:    make([]anthropicMessage, 0, len(req.Messages)),
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.StopSequences,
	}

	if apiReq.MaxTokens == 0 {
		apiReq.MaxTokens = p.config.MaxTokens
	}
	if apiReq.MaxTokens == 0 {
		apiReq.MaxTokens = 4096
	}

	if apiReq.Model == "" {
		apiReq.Model = p.config.Model
	}
	if apiReq.Model == "" {
		apiReq.Model = "claude-sonnet-4-20250514"
	}

	// Convert messages
	for _, msg := range req.Messages {
		role := string(msg.Role)
		if role == "system" {
			continue // System prompt is handled separately
		}
		if role == "tool" {
			role = "user" // Tool responses are sent as user messages
		}
		apiReq.Messages = append(apiReq.Messages, anthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Convert tools
	for _, tool := range req.Tools {
		apiReq.Tools = append(apiReq.Tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}

	// Marshal request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
			return nil, NewProviderError("anthropic", resp.StatusCode, apiErr.Error.Message, retryable)
		}
		return nil, NewProviderError("anthropic", resp.StatusCode, string(respBody), resp.StatusCode >= 500)
	}

	// Parse response
	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build response
	result := &CompletionResponse{
		Model:   apiResp.Model,
		Latency: time.Since(start),
		Usage: Usage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
			TotalTokens:  apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	for _, content := range apiResp.Content {
		switch content.Type {
		case "text":
			result.Content = content.Text
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: content.Input,
			})
		}
	}

	// Map stop reason
	switch apiResp.StopReason {
	case "end_turn":
		result.FinishReason = FinishReasonStop
	case "max_tokens":
		result.FinishReason = FinishReasonMaxTokens
	case "tool_use":
		result.FinishReason = FinishReasonToolUse
	default:
		result.FinishReason = FinishReasonStop
	}

	return result, nil
}

// Ensure AnthropicProvider implements Provider.
var _ Provider = (*AnthropicProvider)(nil)
