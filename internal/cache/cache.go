package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/providers"
)

// CacheEntry holds a cached LLM response
type CacheEntry struct {
	Response  *providers.ChatResponse
	ExpiresAt time.Time
}

// InMemoryCache provides SHA-256 prompt-hashed response caching
type InMemoryCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxSize  int
	ttl      time.Duration
	hits     int64
	misses   int64
}

func NewInMemoryCache(cfg config.CacheConfig) *InMemoryCache {
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 10000
	}
	if cfg.TTL == 0 {
		cfg.TTL = 30 * time.Minute
	}
	return &InMemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: cfg.MaxSize,
		ttl:     cfg.TTL,
	}
}

// Key generates a deterministic cache key from model + messages + params
func Key(model string, req *providers.ChatRequest) string {
	// SHA-256 hash of model:temperature:top_p:max_tokens:last_user_msg
	hasher := sha256.New()
	hasher.Write([]byte(model))

	// Include key parameters that affect output
	params := struct {
		Temperature *float64 `json:"t"`
		TopP        *float64 `json:"p"`
		MaxTokens   int      `json:"mt"`
	}{
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
	}
	if data, err := json.Marshal(params); err == nil {
		hasher.Write(data)
	}

	// Include all messages
	for _, msg := range req.Messages {
		hasher.Write([]byte(msg.Role))
		if content, ok := msg.Content.(string); ok {
			hasher.Write([]byte(content))
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// Get retrieves a cached response
func (c *InMemoryCache) Get(key string) (*providers.ChatResponse, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.miss()
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.miss()
		return nil, false
	}
	c.hit()
	return entry.Response, true
}

// Set stores a response in cache with LRU eviction
func (c *InMemoryCache) Set(key string, resp *providers.ChatResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// LRU eviction: if at capacity, remove oldest (first inserted)
	if len(c.entries) >= c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	c.entries[key] = &CacheEntry{
		Response:  resp,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *InMemoryCache) hit()  { c.hits++ }
func (c *InMemoryCache) miss() { c.misses++ }

// Stats returns cache hit/miss ratio
func (c *InMemoryCache) Stats() (hits, misses int64) {
	return c.hits, c.misses
}
