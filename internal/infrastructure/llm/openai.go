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

const openaiAPIURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider implements the Provider interface for OpenAI GPT.
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     ProviderConfig
}

// OpenAIConfig contains OpenAI-specific configuration.
type OpenAIConfig struct {
	ProviderConfig
	Organization string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openaiAPIURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	return &OpenAIProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config: cfg.ProviderConfig,
	}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Models returns available OpenAI models.
func (p *OpenAIProvider) Models() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		"o1",
		"o1-mini",
	}
}

// openaiRequest is the request structure for OpenAI API.
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openaiMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type openaiTool struct {
	Type     string           `json:"type"`
	Function openaiFunction   `json:"function"`
}

type openaiFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openaiToolFunc `json:"function"`
}

type openaiToolFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// openaiResponse is the response structure from OpenAI API.
type openaiResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiError struct {
	Error openaiErrorDetail `json:"error"`
}

type openaiErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Complete sends a completion request to OpenAI.
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	// Build request
	apiReq := openaiRequest{
		Model:       req.Model,
		Messages:    make([]openaiMessage, 0, len(req.Messages)+1),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.StopSequences,
	}

	if apiReq.Model == "" {
		apiReq.Model = p.config.Model
	}
	if apiReq.Model == "" {
		apiReq.Model = "gpt-4o"
	}

	// Add system prompt
	if req.SystemPrompt != "" {
		apiReq.Messages = append(apiReq.Messages, openaiMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		apiReq.Messages = append(apiReq.Messages, openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Convert tools
	for _, tool := range req.Tools {
		apiReq.Tools = append(apiReq.Tools, openaiTool{
			Type: "function",
			Function: openaiFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
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
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		var apiErr openaiError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
			return nil, NewProviderError("openai", resp.StatusCode, apiErr.Error.Message, retryable)
		}
		return nil, NewProviderError("openai", resp.StatusCode, string(respBody), resp.StatusCode >= 500)
	}

	// Parse response
	var apiResp openaiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, NewProviderError("openai", 0, "no choices returned", false)
	}

	choice := apiResp.Choices[0]

	// Build response
	result := &CompletionResponse{
		Content: choice.Message.Content,
		Model:   apiResp.Model,
		Latency: time.Since(start),
		Usage: Usage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:  apiResp.Usage.TotalTokens,
		},
	}

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			args = map[string]any{"raw": tc.Function.Arguments}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	// Map finish reason
	switch choice.FinishReason {
	case "stop":
		result.FinishReason = FinishReasonStop
	case "length":
		result.FinishReason = FinishReasonMaxTokens
	case "tool_calls":
		result.FinishReason = FinishReasonToolUse
	default:
		result.FinishReason = FinishReasonStop
	}

	return result, nil
}

// Ensure OpenAIProvider implements Provider.
var _ Provider = (*OpenAIProvider)(nil)
