package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SearXNGClient wraps self-hosted SearXNG for web search
type SearXNGClient struct {
	baseURL string
	client  *http.Client
}

// SearchResult from SearXNG
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Content     string `json:"content"`
	Engine      string `json:"engine"`
	Score       float64 `json:"score"`
	PublishedDate string `json:"publishedDate,omitempty"`
}

// SearchResponse wraps SearXNG JSON response
type SearchResponse struct {
	Query           string         `json:"query"`
	NumberOfResults int            `json:"number_of_results"`
	Results         []SearchResult `json:"results"`
	Answers         []string       `json:"answers"`
}

func NewSearXNGClient(searxngURL string) *SearXNGClient {
	return &SearXNGClient{
		baseURL: searxngURL,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Search performs a web search via SearXNG
func (s *SearXNGClient) Search(ctx context.Context, query string, engines []string, limit int) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	if limit > 0 {
		params.Set("pageno", "1")
	}
	if len(engines) > 0 {
		for _, e := range engines {
			params.Add("engines", e)
		}
	}

	searchURL := fmt.Sprintf("%s/search?%s", s.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("searxng: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searxng: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))

	var results SearchResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("searxng: parse response: %w", err)
	}

	// Limit results
	if limit > 0 && len(results.Results) > limit {
		results.Results = results.Results[:limit]
	}

	return &results, nil
}

// SearchAndFormat returns results as a formatted string for LLM context
func (s *SearXNGClient) SearchAndFormat(ctx context.Context, query string) (string, error) {
	resp, err := s.Search(ctx, query, nil, 8)
	if err != nil {
		return "", err
	}

	formatted := fmt.Sprintf("Web search results for: %s\n\n", query)
	for i, r := range resp.Results {
		formatted += fmt.Sprintf("[%d] %s\n    URL: %s\n    %s\n\n", i+1, r.Title, r.URL, r.Content)
	}
	return formatted, nil
}
