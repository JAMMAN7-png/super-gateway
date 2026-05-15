package fallback

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/keys"
	"github.com/goozway/super-gateway/internal/providers"
)

// Engine manages multi-provider routing with fallback chains
type Engine struct {
	providers   map[string]providers.Provider
	keyPools    map[string]*keys.Pool     // provider name -> key pool
	cfg         config.FallbackConfig
	modelMap    map[string][]RouteTarget  // model -> ordered provider list
}

type RouteTarget struct {
	ProviderName string
	Provider     providers.Provider
	Priority     int
	IsFree       bool
}

// NewEngine builds the fallback routing engine
func NewEngine(providersMap map[string]providers.Provider, keyPools map[string]*keys.Pool, cfg config.FallbackConfig) *Engine {
	e := &Engine{
		providers: providersMap,
		keyPools:  keyPools,
		cfg:       cfg,
		modelMap:  make(map[string][]RouteTarget),
	}
	e.buildModelMap()
	return e
}

func (e *Engine) buildModelMap() {
	// For each model known by any provider, build ordered route targets
	modelProviders := make(map[string][]RouteTarget)

	for name, prov := range e.providers {
		for _, model := range prov.Models() {
			modelProviders[model] = append(modelProviders[model], RouteTarget{
				ProviderName: name,
				Provider:     prov,
				IsFree:       prov.IsFree(),
			})
		}
	}

	// Sort each model's providers: free first, then paid
	for model, targets := range modelProviders {
		sort.Slice(targets, func(i, j int) bool {
			// Free before paid
			if targets[i].IsFree != targets[j].IsFree {
				return targets[i].IsFree
			}
			return i < j // stable order
		})
		e.modelMap[model] = targets
	}
}

// ResolveModel expands meta-models to concrete model/provider pairs
func (e *Engine) ResolveModel(modelName string, metaModels map[string]config.MetaModelConfig) []RouteTarget {
	// Check if it's a meta-model
	if meta, ok := metaModels[modelName]; ok {
		var targets []RouteTarget
		for _, ref := range meta.Models {
			// ref format: "provider/model" or just "model"
			found := e.resolveRef(ref)
			targets = append(targets, found...)
		}
		return targets
	}

	// Direct model lookup
	if targets, ok := e.modelMap[modelName]; ok {
		return targets
	}

	// Try to match any provider that has this model
	return e.resolveRef(modelName)
}

func (e *Engine) resolveRef(ref string) []RouteTarget {
	// Handle "provider/model" format
	if parts := strings.SplitN(ref, "/", 2); len(parts) == 2 {
		provName, modelName := parts[0], parts[1]
		if prov, ok := e.providers[provName]; ok {
			for _, m := range prov.Models() {
				if m == modelName {
					return []RouteTarget{{
						ProviderName: provName,
						Provider:     prov,
						IsFree:       prov.IsFree(),
					}}
				}
			}
		}
	}
	// Try to match any provider that has this model
	for name, prov := range e.providers {
		for _, m := range prov.Models() {
			if m == ref {
				return []RouteTarget{{
					ProviderName: name,
					Provider:     prov,
					IsFree:       prov.IsFree(),
				}}
			}
		}
	}
	// Fall back to modelMap
	if targets, ok := e.modelMap[ref]; ok {
		return targets
	}
	return nil
}

// Chat sends a request through the fallback chain
func (e *Engine) Chat(ctx context.Context, req *providers.ChatRequest, targets []RouteTarget) (*providers.ChatResponse, string, error) {
	if len(targets) == 0 {
		return nil, "", fmt.Errorf("no provider available for model %s", req.Model)
	}

	// Sort: free first if configured
	if e.cfg.PaidOnlyAfterFreeFailure {
		sort.SliceStable(targets, func(i, j int) bool {
			return targets[i].IsFree && !targets[j].IsFree
		})
	}

	var lastErr error
	for attempt := 0; attempt < e.cfg.MaxAttempts && attempt < len(targets); attempt++ {
		target := targets[attempt]

		// Try all keys for this provider
		pool := e.keyPools[target.ProviderName]
		if pool == nil {
			lastErr = fmt.Errorf("no key pool for %s", target.ProviderName)
			continue
		}

		resp, key, err := e.tryProvider(ctx, req, target.Provider, pool)
		if err == nil {
			return resp, key, nil
		}

		lastErr = err

		// Check if we should continue
		if _, isRateLimit := err.(*providers.RateLimitError); isRateLimit {
			// Try next key or next provider
			continue
		}

		// For non-rate-limit errors, break the key loop but try next provider
		continue
	}

	return nil, "", fmt.Errorf("all providers exhausted: %w", lastErr)
}

func (e *Engine) tryProvider(ctx context.Context, req *providers.ChatRequest, prov providers.Provider, pool *keys.Pool) (*providers.ChatResponse, string, error) {
	// Try each available key
	for i := 0; i < 3; i++ { // max 3 key attempts per provider
		key, ok := pool.GetKey()
		if !ok {
			return nil, "", fmt.Errorf("%s: all keys in cooldown", prov.Name())
		}

		resp, err := prov.Chat(ctx, req, key)
		if err == nil {
			pool.MarkUsed(key)
			pool.ResetCooldown(key)
			return resp, key, nil
		}

		// Rate limit — cooldown this key and try next
		if _, isRateLimit := err.(*providers.RateLimitError); isRateLimit {
			pool.CoolDown(key)
			continue
		}

		// Other error — try next key if available
		if pool.AvailableKeys() > 1 {
			continue
		}

		// Last key failed with non-rate-limit error
		return nil, "", err
	}

	return nil, "", fmt.Errorf("%s: all keys exhausted", prov.Name())
}

// ChatWithRetry does exponential backoff across providers
func (e *Engine) ChatWithRetry(ctx context.Context, req *providers.ChatRequest, targets []RouteTarget) (*providers.ChatResponse, string, error) {
	delay := e.cfg.RetryDelay
	factor := e.cfg.BackoffFactor
	if factor == 0 {
		factor = 2.0
	}
	if delay == 0 {
		delay = 500 * time.Millisecond
	}

	var lastErr error
	for i := 0; i < e.cfg.MaxAttempts; i++ {
		resp, key, err := e.Chat(ctx, req, targets)
		if err == nil {
			return resp, key, nil
		}
		lastErr = err

		if i < e.cfg.MaxAttempts-1 {
			backoff := time.Duration(float64(delay) * math.Pow(factor, float64(i)))
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return nil, "", fmt.Errorf("all retries exhausted after %d attempts: %w", e.cfg.MaxAttempts, lastErr)
}
