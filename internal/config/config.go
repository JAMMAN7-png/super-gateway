package config

import (
	"os"
	"strings"
	"time"
)

// ProviderConfig defines a single LLM provider endpoint
type ProviderConfig struct {
	Name       string            `yaml:"name"`
	BaseURL    string            `yaml:"base_url"`
	APIKey     string            `yaml:"api_key"`     // env var or comma-separated keys
	APIKeys    []string          `yaml:"api_keys"`    // explicit key list
	Models     []string          `yaml:"models"`
	Tier       string            `yaml:"tier"`        // "free", "paid"
	Priority   int               `yaml:"priority"`    // lower = used first
	Timeout    time.Duration     `yaml:"timeout"`
	MaxRetries int               `yaml:"max_retries"`
	Headers    map[string]string `yaml:"headers"`
	Proxy      string            `yaml:"proxy"`       // per-provider proxy override
}

// GatewayConfig is the top-level config
type GatewayConfig struct {
	Port            int                        `yaml:"port"`
	GlobalProxy     string                     `yaml:"global_proxy"`
	Providers       map[string]ProviderConfig  `yaml:"providers"`
	Fallback        FallbackConfig             `yaml:"fallback"`
	Cache           CacheConfig                `yaml:"cache"`
	Search          SearchConfig               `yaml:"search"`
	KeyRotation     KeyRotationConfig          `yaml:"key_rotation"`
	MetaModels      map[string]MetaModelConfig `yaml:"meta_models"`
	SemanticCache   SemanticCacheConfig        `yaml:"semantic_cache"`
	Guardrails      GuardrailsConfig           `yaml:"guardrails"`
	Compression     CompressionConfig          `yaml:"compression"`
	SpendTracking   SpendTrackingConfig        `yaml:"spend_tracking"`
	Metrics         MetricsConfig              `yaml:"metrics"`
	AdaptiveRouting AdaptiveRoutingConfig      `yaml:"adaptive_routing"`
}

type FallbackConfig struct {
	Enabled                   bool          `yaml:"enabled"`
	MaxAttempts               int           `yaml:"max_attempts"`
	RetryDelay                time.Duration `yaml:"retry_delay"`
	BackoffFactor             float64       `yaml:"backoff_factor"`
	PaidOnlyAfterFreeFailure  bool          `yaml:"paid_only_after_free_failure"`
}

type CacheConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
	MaxSize int           `yaml:"max_size"`
	Redis   string        `yaml:"redis"`
}

type SearchConfig struct {
	Enabled           bool   `yaml:"enabled"`
	SearXNGURL        string `yaml:"searxng_url"`
	ExaAPIKey         string `yaml:"exa_api_key"`
	ParallelAPIKey    string `yaml:"parallel_api_key"`
	FirecrawlURL      string `yaml:"firecrawl_url"`
	FirecrawlAPIKey   string `yaml:"firecrawl_api_key"`
}

type KeyRotationConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Strategy    string        `yaml:"strategy"`
	CoolDown    time.Duration `yaml:"cooldown"`
	MaxCoolDown time.Duration `yaml:"max_cooldown"`
	Tiered      bool          `yaml:"tiered"`
}

type SemanticCacheConfig struct {
	Enabled    bool          `yaml:"enabled"`
	MaxSize    int           `yaml:"max_size"`
	Similarity float64       `yaml:"similarity_threshold"`
	TTL        time.Duration `yaml:"ttl"`
}

type GuardrailsConfig struct {
	Enabled       bool   `yaml:"enabled"`
	PIIRedact     bool   `yaml:"pii_redact"`
	ContentFilter bool   `yaml:"content_filter"`
	MaxInputLen   int    `yaml:"max_input_length"`
}

type CompressionConfig struct {
	InputCompression  bool   `yaml:"input_compression"`
	OutputCompression bool   `yaml:"output_compression"`
	Level             string `yaml:"level"`
}

type SpendTrackingConfig struct {
	Enabled     bool  `yaml:"enabled"`
	DailyBudget int64 `yaml:"daily_budget"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type AdaptiveRoutingConfig struct {
	Enabled          bool `yaml:"enabled"`
	WindowSize       int  `yaml:"window_size"`
	PreferLowLatency bool `yaml:"prefer_low_latency"`
}

type MetaModelConfig struct {
	Strategy string   `yaml:"strategy"`
	Models   []string `yaml:"models"`
}

// ResolveKeys expands env vars and comma-separated key strings
func (p *ProviderConfig) ResolveKeys() []string {
	if len(p.APIKeys) > 0 {
		return p.APIKeys
	}
	key := p.APIKey
	if strings.HasPrefix(key, "$") {
		key = os.Getenv(key[1:])
	}
	if key == "" {
		return nil
	}
	keys := strings.Split(key, ",")
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			result = append(result, k)
		}
	}
	return result
}

// DefaultConfig returns sensible defaults
func DefaultConfig() GatewayConfig {
	return GatewayConfig{
		Port: 3000,
		Fallback: FallbackConfig{
			Enabled:                  true,
			MaxAttempts:              5,
			RetryDelay:               500 * time.Millisecond,
			BackoffFactor:            2.0,
			PaidOnlyAfterFreeFailure: true,
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     30 * time.Minute,
			MaxSize: 10000,
		},
		Search: SearchConfig{
			Enabled: true,
		},
		KeyRotation: KeyRotationConfig{
			Enabled:     true,
			Strategy:    "least_used",
			CoolDown:    30 * time.Second,
			MaxCoolDown: 5 * time.Minute,
		},
		SemanticCache: SemanticCacheConfig{
			Enabled:    true,
			MaxSize:    5000,
			Similarity: 0.85,
			TTL:        15 * time.Minute,
		},
		Guardrails: GuardrailsConfig{
			Enabled:       true,
			PIIRedact:     true,
			ContentFilter: false,
			MaxInputLen:   0,
		},
		Compression: CompressionConfig{
			InputCompression:  false,
			OutputCompression: false,
			Level:             "light",
		},
		SpendTracking: SpendTrackingConfig{
			Enabled:     true,
			DailyBudget: 1000000,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
		AdaptiveRouting: AdaptiveRoutingConfig{
			Enabled:          true,
			WindowSize:       100,
			PreferLowLatency: true,
		},
	}
}
