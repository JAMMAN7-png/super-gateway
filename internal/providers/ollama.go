package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/proxy"
)

// OllamaProvider handles both local and cloud Ollama
type OllamaProvider struct {
	name    string
	baseURL string
	models  []string
	tier    string
	client  *http.Client
}

func init() {
	Register("ollama", NewOllamaProvider)
	// ollama-cloud uses OpenAI-compat adapter, not Ollama adapter
}

func NewOllamaProvider(cfg config.ProviderConfig, proxyURL string) (Provider, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second // Ollama local can be slow
	}
	return &OllamaProvider{
		name:    cfg.Name,
		baseURL: cfg.BaseURL, // e.g. http://localhost:11434 or https://api.ollama.com
		models:  cfg.Models,
		tier:    cfg.Tier,
		client:  proxy.NewStandardHTTPClient(proxy.DialerConfig{ProxyURL: proxyURL, Timeout: timeout, KeepAlive: 30 * time.Second}),
	}, nil
}

func (p *OllamaProvider) Name() string    { return p.name }
func (p *OllamaProvider) Models() []string { return p.models }
func (p *OllamaProvider) BaseURL() string  { return p.baseURL }
func (p *OllamaProvider) IsFree() bool     { return true }
func (p *OllamaProvider) IsHealthy() bool  { return true }

type ollamaChatReq struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  struct {
		Temperature float64 `json:"temperature,omitempty"`
		NumPredict  int     `json:"num_predict,omitempty"`
		TopP        float64 `json:"top_p,omitempty"`
	} `json:"options,omitempty"`
}

type ollamaChatResp struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest, key string) (*ChatResponse, error) {
	ollamaReq := ollamaChatReq{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}
	if req.Temperature != nil {
		ollamaReq.Options.Temperature = *req.Temperature
	}
	if req.MaxTokens > 0 {
		ollamaReq.Options.NumPredict = req.MaxTokens
	}

	body, _ := json.Marshal(ollamaReq)
	url := p.baseURL + "/api/chat"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))

	if resp.StatusCode >= 400 {
		return nil, &ProviderError{Provider: "ollama", Status: resp.StatusCode, Body: string(respBody)}
	}

	var ollamaResp ollamaChatResp
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal: %w", err)
	}

	return &ChatResponse{
		ID:      "ollama-" + req.Model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{{
			Index:   0,
			Message: ollamaResp.Message,
			FinishReason: "stop",
		}},
	}, nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, req *ChatRequest, key string) (<-chan StreamChunk, error) {
	ollamaReq := ollamaChatReq{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}
	if req.Temperature != nil {
		ollamaReq.Options.Temperature = *req.Temperature
	}
	if req.MaxTokens > 0 {
		ollamaReq.Options.NumPredict = req.MaxTokens
	}

	body, _ := json.Marshal(ollamaReq)
	url := p.baseURL + "/api/chat"

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama stream: %w", err)
	}

	ch := make(chan StreamChunk, 10)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaChatResp
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					ch <- StreamChunk{Error: err}
				}
				ch <- StreamChunk{Done: true}
				return
			}
			// Convert Ollama chunk to OpenAI SSE format
			openaiChunk := map[string]interface{}{
				"id":      "ollama-" + req.Model,
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]interface{}{{
					"index": 0,
					"delta": map[string]interface{}{
						"role":    chunk.Message.Role,
						"content": chunk.Message.Content,
					},
					"finish_reason": nil,
				}},
			}
			data, _ := json.Marshal(openaiChunk)
			ch <- StreamChunk{Data: data}
			if chunk.Done {
				ch <- StreamChunk{Done: true}
				return
			}
		}
	}()
	return ch, nil
}
