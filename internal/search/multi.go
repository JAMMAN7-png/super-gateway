package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/goozway/super-gateway/internal/config"
)

// MultiEngine aggregates results from multiple search backends
type MultiEngine struct {
	searxng    *SearXNGClient
	exaKey     string
	parallelKey string
	firecrawlURL string
	firecrawlKey string
	client     *http.Client
	mu         sync.RWMutex
}

// NewMultiEngine creates a multi-engine search aggregator
func NewMultiEngine(cfg config.SearchConfig) *MultiEngine {
	return &MultiEngine{
		searxng:      NewSearXNGClient(cfg.SearXNGURL),
		exaKey:       cfg.ExaAPIKey,
		parallelKey:  cfg.ParallelAPIKey,
		firecrawlURL: strings.TrimRight(cfg.FirecrawlURL, "/"),
		firecrawlKey: cfg.FirecrawlAPIKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AggregatedResult from a search engine
type AggregatedResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
	Source  string  `json:"source"` // "searxng", "exa", "parallel", "firecrawl"
}

// AggregatedResponse combines results from all engines
type AggregatedResponse struct {
	Results  []AggregatedResult `json:"results"`
	Total    int                `json:"total"`
	TimeMs   int64              `json:"time_ms"`
}

// SearchAll runs search across all configured engines and merges results
func (m *MultiEngine) SearchAll(ctx context.Context, query string, limit int) (*AggregatedResponse, error) {
	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allResults []AggregatedResult

	engines := 0

	// SearXNG
	if m.searxng != nil {
		engines++
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := m.searxng.Search(ctx, query, nil, limit)
			if err != nil || resp == nil {
				return
			}
			mu.Lock()
			for _, r := range resp.Results {
				allResults = append(allResults, AggregatedResult{
					Title:   r.Title,
					URL:     r.URL,
					Snippet: r.Content,
					Score:   float64(len(allResults) + 1),
					Source:  "searxng",
				})
			}
			mu.Unlock()
		}()
	}

	// Exa.ai
	if m.exaKey != "" {
		engines++
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := m.searchExa(ctx, query, limit)
			if err != nil {
				return
			}
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}()
	}

	// Parallel.ai
	if m.parallelKey != "" {
		engines++
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := m.searchParallel(ctx, query)
			if err != nil {
				return
			}
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}()
	}

	// Firecrawl
	if m.firecrawlURL != "" {
		engines++
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := m.searchFirecrawl(ctx, query, limit)
			if err != nil {
				return
			}
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// If only one engine, return its results directly
	if engines == 0 {
		return &AggregatedResponse{
			Results: []AggregatedResult{},
			Total:   0,
			TimeMs:  time.Since(start).Milliseconds(),
		}, nil
	}

	// Deduplicate by URL
	seen := make(map[string]bool)
	deduped := make([]AggregatedResult, 0, len(allResults))
	for _, r := range allResults {
		if r.URL == "" || seen[r.URL] {
			continue
		}
		seen[r.URL] = true
		deduped = append(deduped, r)
	}

	// Sort by score descending
	for i := 0; i < len(deduped); i++ {
		for j := i + 1; j < len(deduped); j++ {
			if deduped[j].Score > deduped[i].Score {
				deduped[i], deduped[j] = deduped[j], deduped[i]
			}
		}
	}

	// Limit
	if limit > 0 && len(deduped) > limit {
		deduped = deduped[:limit]
	}

	return &AggregatedResponse{
		Results: deduped,
		Total:   len(deduped),
		TimeMs:  time.Since(start).Milliseconds(),
	}, nil
}

// searchExa queries the Exa.ai search API
func (m *MultiEngine) searchExa(ctx context.Context, query string, limit int) ([]AggregatedResult, error) {
	body := map[string]interface{}{
		"query": query,
		"numResults": limit,
	}
	if limit <= 0 {
		body["numResults"] = 5
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.exa.ai/search", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", m.exaKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exa search: %w", err)
	}
	defer resp.Body.Close()

	var exaResp struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Snippet string  `json:"snippet"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&exaResp); err != nil {
		return nil, err
	}

	results := make([]AggregatedResult, 0, len(exaResp.Results))
	for i, r := range exaResp.Results {
		results = append(results, AggregatedResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
			Score:   r.Score * 100 + float64(len(exaResp.Results)-i),
			Source:  "exa",
		})
	}
	return results, nil
}

// searchParallel queries Parallel.ai search
func (m *MultiEngine) searchParallel(ctx context.Context, query string) ([]AggregatedResult, error) {
	body := map[string]interface{}{
		"inputs": []map[string]string{
			{"query": query},
		},
		"output_type": "text",
		"output": "Return a list of 5 search results. For each result include title, url, and a brief snippet.",
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://task-mcp.parallel.ai/api/task-group", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.parallelKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("parallel search: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	// Extract results as aggregated entries
	var results []AggregatedResult
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			results = append(results, AggregatedResult{
				Snippet: strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "),
				Score:   50.0,
				Source:  "parallel",
			})
		}
	}

	return results, nil
}

// searchFirecrawl queries the self-hosted Firecrawl API
func (m *MultiEngine) searchFirecrawl(ctx context.Context, query string, limit int) ([]AggregatedResult, error) {
	body := map[string]interface{}{
		"query": query,
		"limit": limit,
	}
	if limit <= 0 {
		body["limit"] = 5
	}

	data, _ := json.Marshal(body)
	apiURL := m.firecrawlURL + "/v1/search"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.firecrawlKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.firecrawlKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firecrawl search: %w", err)
	}
	defer resp.Body.Close()

	var fcResp struct {
		Success bool `json:"success"`
		Data    []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&fcResp); err != nil {
		return nil, err
	}

	results := make([]AggregatedResult, 0, len(fcResp.Data))
	for i, r := range fcResp.Data {
		snippet := r.Content
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		results = append(results, AggregatedResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: snippet,
			Score:   r.Score * 100 + float64(len(fcResp.Data)-i),
			Source:  "firecrawl",
		})
	}
	return results, nil
}
