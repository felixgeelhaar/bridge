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

const ollamaAPIURL = "http://localhost:11434/api/chat"

// OllamaProvider implements the Provider interface for Ollama (local LLMs).
type OllamaProvider struct {
	baseURL    string
	httpClient *http.Client
	config     ProviderConfig
}

// OllamaConfig contains Ollama-specific configuration.
type OllamaConfig struct {
	ProviderConfig
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(cfg OllamaConfig) *OllamaProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ollamaAPIURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute // Longer timeout for local models
	}

	return &OllamaProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config: cfg.ProviderConfig,
	}
}

// Name returns the provider name.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Models returns commonly used Ollama models.
func (p *OllamaProvider) Models() []string {
	return []string{
		"llama3.2",
		"llama3.1",
		"llama3",
		"codellama",
		"mistral",
		"mixtral",
		"qwen2.5-coder",
		"deepseek-coder-v2",
		"phi3",
	}
}

// ollamaRequest is the request structure for Ollama API.
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ollamaToolCall struct {
	Function ollamaToolFunc `json:"function"`
}

type ollamaToolFunc struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ollamaOptions struct {
	Temperature float64  `json:"temperature,omitempty"`
	TopP        float64  `json:"top_p,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// ollamaResponse is the response structure from Ollama API.
type ollamaResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            ollamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration"`
	LoadDuration       int64         `json:"load_duration"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	PromptEvalDuration int64         `json:"prompt_eval_duration"`
	EvalCount          int           `json:"eval_count"`
	EvalDuration       int64         `json:"eval_duration"`
}

type ollamaError struct {
	Error string `json:"error"`
}

// Complete sends a completion request to Ollama.
func (p *OllamaProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" {
		model = "llama3.2"
	}

	// Build request
	apiReq := ollamaRequest{
		Model:    model,
		Messages: make([]ollamaMessage, 0, len(req.Messages)+1),
		Stream:   false,
		Options: &ollamaOptions{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			NumPredict:  req.MaxTokens,
			Stop:        req.StopSequences,
		},
	}

	// Add system prompt
	if req.SystemPrompt != "" {
		apiReq.Messages = append(apiReq.Messages, ollamaMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		apiReq.Messages = append(apiReq.Messages, ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Convert tools
	for _, tool := range req.Tools {
		apiReq.Tools = append(apiReq.Tools, ollamaTool{
			Type:     "function",
			Function: ollamaFunction(tool),
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

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var apiErr ollamaError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			retryable := resp.StatusCode >= 500
			return nil, NewProviderError("ollama", resp.StatusCode, apiErr.Error, retryable)
		}
		return nil, NewProviderError("ollama", resp.StatusCode, string(respBody), resp.StatusCode >= 500)
	}

	// Parse response
	var apiResp ollamaResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build response
	result := &CompletionResponse{
		Content:      apiResp.Message.Content,
		Model:        apiResp.Model,
		Latency:      time.Since(start),
		FinishReason: FinishReasonStop,
		Usage: Usage{
			InputTokens:  apiResp.PromptEvalCount,
			OutputTokens: apiResp.EvalCount,
			TotalTokens:  apiResp.PromptEvalCount + apiResp.EvalCount,
		},
	}

	// Extract tool calls
	for _, tc := range apiResp.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        "", // Ollama doesn't provide IDs
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
		result.FinishReason = FinishReasonToolUse
	}

	return result, nil
}

// Ensure OllamaProvider implements Provider.
var _ Provider = (*OllamaProvider)(nil)
