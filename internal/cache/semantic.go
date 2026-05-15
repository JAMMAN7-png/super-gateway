package cache

import (
	"container/list"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/providers"
)

// SemanticEntry holds a cached response with its prompt for similarity matching
type SemanticEntry struct {
	Prompt       string
	Response     *providers.ChatResponse
	CreatedAt    time.Time
	AccessCount  int
}

// SemanticCache provides similarity-based response caching (from Bifrost)
type SemanticCache struct {
	mu      sync.RWMutex
	entries map[string]*SemanticEntry
	lru     list.List
	config  config.SemanticCacheConfig
	hits    int64
	misses  int64
}

// NewSemanticCache creates a semantic cache
func NewSemanticCache(cfg config.SemanticCacheConfig) *SemanticCache {
	return &SemanticCache{
		entries: make(map[string]*SemanticEntry, cfg.MaxSize),
		config:  cfg,
	}
}

// ngramTokens extracts character n-gram frequency from a string
func ngramTokens(s string, n int) map[string]int {
	tokens := make(map[string]int)
	runes := []rune(s)
	for i := 0; i <= len(runes)-n; i++ {
		var b strings.Builder
		for j := i; j < i+n; j++ {
			b.WriteRune(unicode.ToLower(runes[j]))
		}
		tokens[b.String()]++
	}
	return tokens
}

// cosineSimilarity computes cosine similarity between two frequency maps
func cosineSimilarity(a, b map[string]int) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for k, v := range a {
		fv := float64(v)
		normA += fv * fv
		if bv, ok := b[k]; ok {
			dot += fv * float64(bv)
		}
	}
	for _, v := range b {
		fv := float64(v)
		normB += fv * fv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// extractPromptText joins all user messages into a single string for comparison
func extractPromptText(req *providers.ChatRequest) string {
	var b strings.Builder
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			if content, ok := msg.Content.(string); ok {
				b.WriteString(content)
				b.WriteByte(' ')
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// cacheKey generates a simple grouping key from model + params
func cacheKey(req *providers.ChatRequest) string {
	var b strings.Builder
	b.WriteString(req.Model)
	if req.Temperature != nil {
		b.WriteString(fmt.Sprintf("|t=%.2f", *req.Temperature))
	}
	if req.MaxTokens > 0 {
		b.WriteString(fmt.Sprintf("|m=%d", req.MaxTokens))
	}
	return b.String()
}

// Get looks up a semantically similar cached response
func (s *SemanticCache) Get(req *providers.ChatRequest) (*providers.ChatResponse, bool, float64) {
	if !s.config.Enabled {
		s.misses++
		return nil, false, 0
	}

	prompt := extractPromptText(req)
	if prompt == "" {
		s.misses++
		return nil, false, 0
	}

	groupKey := cacheKey(req)
	promptTokens := ngramTokens(prompt, 3)

	// First check exact group key match (fast path)
	s.mu.RLock()
	entry, exists := s.entries[groupKey]
	s.mu.RUnlock()

	if exists {
		if time.Since(entry.CreatedAt) < s.config.TTL {
			cachedTokens := ngramTokens(entry.Prompt, 3)
			sim := cosineSimilarity(promptTokens, cachedTokens)
			if sim >= s.config.Similarity {
				entry.AccessCount++
				s.hits++
				return entry.Response, true, sim
			}
		}
	}

	// Linear scan for similar prompts (slower but catches semantic matches)
	s.mu.RLock()
	for _, e := range s.entries {
		if time.Since(e.CreatedAt) >= s.config.TTL {
			continue
		}
		eTokens := ngramTokens(e.Prompt, 3)
		sim := cosineSimilarity(promptTokens, eTokens)
		if sim >= s.config.Similarity {
			s.mu.RUnlock()
			e.AccessCount++
			s.hits++
			return e.Response, true, sim
		}
	}
	s.mu.RUnlock()

	s.misses++
	return nil, false, 0
}

// Set stores a response in the semantic cache
func (s *SemanticCache) Set(req *providers.ChatRequest, resp *providers.ChatResponse) {
	if !s.config.Enabled || resp == nil {
		return
	}

	prompt := extractPromptText(req)
	if prompt == "" {
		return
	}

	key := cacheKey(req)

	s.mu.Lock()
	defer s.mu.Unlock()

	// LRU eviction
	if len(s.entries) >= s.config.MaxSize {
		elem := s.lru.Front()
		if elem != nil {
			evictKey := elem.Value.(string)
			delete(s.entries, evictKey)
			s.lru.Remove(elem)
		}
	}

	s.entries[key] = &SemanticEntry{
		Prompt:    prompt,
		Response:  resp,
		CreatedAt: time.Now(),
	}
	s.lru.PushBack(key)
}

// Stats returns hit/miss counts
func (s *SemanticCache) Stats() (hits, misses int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hits, s.misses
}

// PurgeExpired removes expired entries
func (s *SemanticCache) PurgeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, entry := range s.entries {
		if now.After(entry.CreatedAt.Add(s.config.TTL)) {
			delete(s.entries, key)
		}
	}
}
