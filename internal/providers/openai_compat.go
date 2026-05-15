package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/proxy"
)

// OpenAICompatProvider handles any OpenAI-compatible API (Groq, Cerebras, DeepSeek, etc.)
type OpenAICompatProvider struct {
	name    string
	baseURL string
	models  []string
	tier    string
	client  *http.Client
	timeout time.Duration
}

func init() {
	Register("openai-compat", NewOpenAICompat)
	Register("groq", NewOpenAICompat)
	Register("cerebras", NewOpenAICompat)
	Register("deepseek", NewOpenAICompat)
	Register("novita", NewOpenAICompat)
	Register("chutes", NewOpenAICompat)
	Register("nanogpt", NewOpenAICompat)
	Register("nvidia", NewOpenAICompat)
	Register("openrouter", NewOpenAICompat)
	Register("opencode-zen", NewOpenAICompat)
	Register("opencode-go", NewOpenAICompat)
	Register("ollama-cloud", NewOpenAICompat) // Ollama Cloud is OpenAI-compat
	Register("vercel-ai-gateway", NewOpenAICompat) // Vercel AI Gateway is OpenAI-compat
	Register("cloudflare-gateway", NewOpenAICompat) // Cloudflare AI Gateway is OpenAI-compat
}

func NewOpenAICompat(cfg config.ProviderConfig, proxyURL string) (Provider, error) {
	if proxyURL == "" {
		proxyURL = cfg.Proxy
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &OpenAICompatProvider{
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		models:  cfg.Models,
		tier:    cfg.Tier,
		client:  proxy.NewStandardHTTPClient(proxy.DialerConfig{ProxyURL: proxyURL, Timeout: timeout, KeepAlive: 30 * time.Second}),
		timeout: timeout,
	}, nil
}

func (p *OpenAICompatProvider) Name() string     { return p.name }
func (p *OpenAICompatProvider) Models() []string  { return p.models }
func (p *OpenAICompatProvider) BaseURL() string   { return p.baseURL }
func (p *OpenAICompatProvider) IsFree() bool      { return p.tier == "free" }
func (p *OpenAICompatProvider) IsHealthy() bool   { return true } // circuit breaker manages this

func (p *OpenAICompatProvider) Chat(ctx context.Context, req *ChatRequest, key string) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", p.name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB max
	if err != nil {
		return nil, fmt.Errorf("%s: read body: %w", p.name, err)
	}

	if resp.StatusCode == 429 {
		return nil, &RateLimitError{Provider: p.name}
	}
	if resp.StatusCode >= 400 {
		return nil, &ProviderError{Provider: p.name, Status: resp.StatusCode, Body: string(respBody)}
	}

	var cr ChatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("%s: unmarshal response: %w\nbody: %s", p.name, err, string(respBody[:min(len(respBody), 500)]))
	}
	return &cr, nil
}

func (p *OpenAICompatProvider) ChatStream(ctx context.Context, req *ChatRequest, key string) (<-chan StreamChunk, error) {
	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: stream request failed: %w", p.name, err)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, &RateLimitError{Provider: p.name}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10000))
		resp.Body.Close()
		return nil, &ProviderError{Provider: p.name, Status: resp.StatusCode, Body: string(body)}
	}

	ch := make(chan StreamChunk, 10)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := scanner.Text()
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := line[6:]
			if data == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}
			ch <- StreamChunk{Data: []byte(data)}
		}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

// Error types
type RateLimitError struct{ Provider string }
func (e *RateLimitError) Error() string { return e.Provider + ": rate limited (429)" }

type ProviderError struct {
	Provider string
	Status   int
	Body     string
}
func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s: HTTP %d: %s", e.Provider, e.Status, e.Body[:min(len(e.Body), 200)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
