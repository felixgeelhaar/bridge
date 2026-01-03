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

const geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiProvider implements the Provider interface for Google Gemini.
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     ProviderConfig
}

// GeminiConfig contains Gemini-specific configuration.
type GeminiConfig struct {
	ProviderConfig
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(cfg GeminiConfig) *GeminiProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = geminiAPIURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	return &GeminiProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config: cfg.ProviderConfig,
	}
}

// Name returns the provider name.
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Models returns available Gemini models.
func (p *GeminiProvider) Models() []string {
	return []string{
		"gemini-2.0-flash-exp",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.5-flash-8b",
		"gemini-1.0-pro",
	}
}

// geminiRequest is the request structure for Gemini API.
type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// geminiResponse is the response structure from Gemini API.
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsage       `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiError struct {
	Error geminiErrorDetail `json:"error"`
}

type geminiErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Complete sends a completion request to Gemini.
func (p *GeminiProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" {
		model = "gemini-1.5-flash"
	}

	// Build request
	apiReq := geminiRequest{
		Contents: make([]geminiContent, 0),
		GenerationConfig: &geminiGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			StopSequences:   req.StopSequences,
		},
	}

	if apiReq.GenerationConfig.MaxOutputTokens == 0 {
		apiReq.GenerationConfig.MaxOutputTokens = p.config.MaxTokens
	}

	// Add system instruction
	if req.SystemPrompt != "" {
		apiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		role := string(msg.Role)
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			continue // Handled separately
		}

		apiReq.Contents = append(apiReq.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	// Convert tools
	if len(req.Tools) > 0 {
		declarations := make([]geminiFunctionDecl, 0, len(req.Tools))
		for _, tool := range req.Tools {
			declarations = append(declarations, geminiFunctionDecl(tool))
		}
		apiReq.Tools = []geminiTool{{FunctionDeclarations: declarations}}
	}

	// Marshal request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL with API key
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
		var apiErr geminiError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
			return nil, NewProviderError("gemini", resp.StatusCode, apiErr.Error.Message, retryable)
		}
		return nil, NewProviderError("gemini", resp.StatusCode, string(respBody), resp.StatusCode >= 500)
	}

	// Parse response
	var apiResp geminiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Candidates) == 0 {
		return nil, NewProviderError("gemini", 0, "no candidates returned", false)
	}

	candidate := apiResp.Candidates[0]

	// Build response
	result := &CompletionResponse{
		Model:   model,
		Latency: time.Since(start),
		Usage: Usage{
			InputTokens:  apiResp.UsageMetadata.PromptTokenCount,
			OutputTokens: apiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  apiResp.UsageMetadata.TotalTokenCount,
		},
	}

	// Extract content and tool calls
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			result.Content = part.Text
		}
		if part.FunctionCall != nil {
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        "", // Gemini doesn't provide IDs
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}

	// Map finish reason
	switch candidate.FinishReason {
	case "STOP":
		result.FinishReason = FinishReasonStop
	case "MAX_TOKENS":
		result.FinishReason = FinishReasonMaxTokens
	case "TOOL_USE":
		result.FinishReason = FinishReasonToolUse
	default:
		result.FinishReason = FinishReasonStop
	}

	return result, nil
}

// Ensure GeminiProvider implements Provider.
var _ Provider = (*GeminiProvider)(nil)
