package providers

import (
	"bufio"
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

// CloudflareWorkersAI handles the Cloudflare Workers AI REST API
type CloudflareWorkersAI struct {
	name       string
	accountID  string
	apiToken   string
	models     []string
	tier       string
	client     *http.Client
}

func init() {
	Register("cloudflare-workers", NewCloudflareWorkersAI)
}

func NewCloudflareWorkersAI(cfg config.ProviderConfig, proxyURL string) (Provider, error) {
	if proxyURL == "" {
		proxyURL = cfg.Proxy
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	accountID := ""
	if v, ok := cfg.Headers["CF-Account-ID"]; ok {
		accountID = v
	}
	return &CloudflareWorkersAI{
		name:      cfg.Name,
		accountID: accountID,
		apiToken:  "", // set per-request from key pool
		models:    cfg.Models,
		tier:      cfg.Tier,
		client:    proxy.NewStandardHTTPClient(proxy.DialerConfig{ProxyURL: proxyURL, Timeout: timeout, KeepAlive: 30 * time.Second}),
	}, nil
}

func (p *CloudflareWorkersAI) Name() string    { return p.name }
func (p *CloudflareWorkersAI) Models() []string { return p.models }
func (p *CloudflareWorkersAI) BaseURL() string  { return "https://api.cloudflare.com/client/v4/accounts/" + p.accountID + "/ai/run" }
func (p *CloudflareWorkersAI) IsFree() bool     { return p.tier == "free" }
func (p *CloudflareWorkersAI) IsHealthy() bool  { return true }

// cfRequest matches Cloudflare Workers AI chat format
type cfRequest struct {
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type cfResponse struct {
	Result struct {
		Response string `json:"response"`
	} `json:"result"`
	Success bool `json:"success"`
}

func (p *CloudflareWorkersAI) Chat(ctx context.Context, req *ChatRequest, key string) (*ChatResponse, error) {
	cfReq := cfRequest{
		Messages:  req.Messages,
		Stream:    false,
		MaxTokens: req.MaxTokens,
	}

	body, _ := json.Marshal(cfReq)
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/%s",
		p.accountID, req.Model)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cf: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cf: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))

	if resp.StatusCode == 429 {
		return nil, &RateLimitError{Provider: "cloudflare"}
	}
	if resp.StatusCode >= 400 {
		return nil, &ProviderError{Provider: "cloudflare", Status: resp.StatusCode, Body: string(respBody)}
	}

	var cfResp cfResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return nil, fmt.Errorf("cf: unmarshal: %w", err)
	}

	return &ChatResponse{
		ID:      "cf-" + req.Model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: cfResp.Result.Response,
			},
			FinishReason: "stop",
		}},
		Usage: Usage{TotalTokens: 0}, // CF doesn't return token counts reliably
	}, nil
}

func (p *CloudflareWorkersAI) ChatStream(ctx context.Context, req *ChatRequest, key string) (<-chan StreamChunk, error) {
	// Cloudflare Workers AI streaming via SSE
	cfReq := cfRequest{Messages: req.Messages, Stream: true, MaxTokens: req.MaxTokens}
	body, _ := json.Marshal(cfReq)
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/%s", p.accountID, req.Model)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cf stream: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10000))
		resp.Body.Close()
		return nil, &ProviderError{Provider: "cloudflare", Status: resp.StatusCode, Body: string(body)}
	}

	ch := make(chan StreamChunk, 10)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		// CF streams text/event-stream lines with "data: <json>"
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 6 && line[:6] == "data: " {
				data := line[6:]
				if data == "[DONE]" {
					ch <- StreamChunk{Done: true}
					return
				}
				ch <- StreamChunk{Data: []byte(data)}
			}
		}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}
