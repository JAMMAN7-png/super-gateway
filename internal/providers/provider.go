package providers

import (
	"context"

	"github.com/goozway/super-gateway/internal/config"
)

// ChatRequest is the unified OpenAI-compatible chat completion request
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []Message     `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	Tools       []Tool        `json:"tools,omitempty"`
	ToolChoice  interface{}   `json:"tool_choice,omitempty"`
}

type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []ContentPart
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function FunctionDef  `json:"function"`
}

type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse is the unified OpenAI-compatible response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider interface — every LLM backend implements this
type Provider interface {
	Name() string
	Models() []string
	Chat(ctx context.Context, req *ChatRequest, key string) (*ChatResponse, error)
	ChatStream(ctx context.Context, req *ChatRequest, key string) (<-chan StreamChunk, error)
	BaseURL() string
	IsFree() bool
	IsHealthy() bool
}

type StreamChunk struct {
	Data  []byte
	Error error
	Done  bool
}

// Factory creates a provider from config
type Factory func(cfg config.ProviderConfig, proxyURL string) (Provider, error)

var registry = map[string]Factory{}

// Register adds a provider factory
func Register(name string, f Factory) {
	registry[name] = f
}

// Create instantiates a provider
func Create(name string, cfg config.ProviderConfig, proxyURL string) (Provider, error) {
	f, ok := registry[name]
	if !ok {
		return nil, &UnknownProviderError{Name: name}
	}
	return f(cfg, proxyURL)
}

type UnknownProviderError struct{ Name string }
func (e *UnknownProviderError) Error() string { return "unknown provider: " + e.Name }
