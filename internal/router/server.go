package router

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/goozway/super-gateway/internal/cache"
	"github.com/goozway/super-gateway/internal/compress"
	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/fallback"
	"github.com/goozway/super-gateway/internal/guardrails"
	"github.com/goozway/super-gateway/internal/keys"
	"github.com/goozway/super-gateway/internal/logging"
	"github.com/goozway/super-gateway/internal/metrics"
	"github.com/goozway/super-gateway/internal/providers"
	"github.com/goozway/super-gateway/internal/proxy"
	"github.com/goozway/super-gateway/internal/search"
	"github.com/goozway/super-gateway/internal/vkeys"
	"github.com/sony/gobreaker/v2"
)

// Server is the main HTTP gateway
type Server struct {
	app        *fiber.App
	config     config.GatewayConfig
	providers  map[string]providers.Provider
	keyPools   map[string]*keys.Pool
	tieredPools map[string]*keys.TieredPool
	fallback   *fallback.Engine
	cache      *cache.InMemoryCache
	semCache   *cache.SemanticCache
	search     *search.SearXNGClient
	searchMulti *search.MultiEngine
	breakers   map[string]*gobreaker.CircuitBreaker[any]
	logStore   *logging.Store
	vkeyMgr    *vkeys.Manager
	spendTrack *vkeys.SpendTracker
	guard      *guardrails.Guardrails
	metrics    *metrics.Metrics
	balancer   *AdaptiveBalancer
	masterKey  string
}


// NewServer creates and initializes the gateway
func NewServer(cfg config.GatewayConfig) (*Server, error) {
	s := &Server{
		config:      cfg,
		providers:   make(map[string]providers.Provider),
		keyPools:    make(map[string]*keys.Pool),
		tieredPools: make(map[string]*keys.TieredPool),
		breakers:    make(map[string]*gobreaker.CircuitBreaker[any]),
	}

	// Initialize providers
	for name, pcfg := range cfg.Providers {
		prov, err := providers.Create(name, pcfg, cfg.GlobalProxy)
		if err != nil {
			log.Printf("WARN: skipping provider %s: %v", name, err)
			continue
		}
		s.providers[name] = prov
		log.Printf("Loaded provider: %s (%d models, tier=%s)", name, len(prov.Models()), pcfg.Tier)

		// Build key pool
		resolvedKeys := pcfg.ResolveKeys()
		if len(resolvedKeys) > 0 {
			if cfg.KeyRotation.Tiered {
				// Tiered (escalating cooldown) pool from LLM-API-Key-Proxy
				s.tieredPools[name] = keys.NewTieredPool(
					resolvedKeys,
					cfg.KeyRotation.CoolDown,
					cfg.KeyRotation.MaxCoolDown,
				)
				log.Printf("  Tiered key pool: %d keys", len(resolvedKeys))
			} else {
				s.keyPools[name] = keys.NewPool(
					resolvedKeys,
					cfg.KeyRotation.Strategy,
					cfg.KeyRotation.CoolDown,
					cfg.KeyRotation.MaxCoolDown,
				)
				log.Printf("  Key pool: %d keys, strategy=%s", len(resolvedKeys), cfg.KeyRotation.Strategy)
			}
		}

		// Circuit breaker per provider
		s.breakers[name] = gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
			Name:        name,
			MaxRequests: 3,
			Interval:    10 * time.Second,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures > 5
			},
		})
	}

	// Initialize fallback engine
	s.fallback = fallback.NewEngine(s.providers, s.keyPools, cfg.Fallback)

	// Initialize exact cache (SHA-256)
	if cfg.Cache.Enabled {
		s.cache = cache.NewInMemoryCache(cfg.Cache)
		log.Printf("Response cache: enabled, TTL=%v, max=%d entries", cfg.Cache.TTL, cfg.Cache.MaxSize)
	}

	// Initialize semantic cache (similarity-based from Bifrost)
	if cfg.SemanticCache.Enabled {
		s.semCache = cache.NewSemanticCache(cfg.SemanticCache)
		log.Printf("Semantic cache: enabled, threshold=%.2f, max=%d", cfg.SemanticCache.Similarity, cfg.SemanticCache.MaxSize)
	}

	// Initialize search (SearXNG)
	if cfg.Search.Enabled && cfg.Search.SearXNGURL != "" {
		s.search = search.NewSearXNGClient(cfg.Search.SearXNGURL)
		log.Printf("SearXNG web search: %s", cfg.Search.SearXNGURL)
	}

	// Initialize multi-engine search (Exa + Parallel + Firecrawl)
	if cfg.Search.ExaAPIKey != "" || cfg.Search.ParallelAPIKey != "" || cfg.Search.FirecrawlURL != "" {
		s.searchMulti = search.NewMultiEngine(cfg.Search)
		log.Printf("Multi-engine search: enabled (exa=%v parallel=%v firecrawl=%v)",
			cfg.Search.ExaAPIKey != "", cfg.Search.ParallelAPIKey != "", cfg.Search.FirecrawlURL != "")
	}

	// Initialize guardrails/PII redaction (from Portkey)
	if cfg.Guardrails.Enabled {
		s.guard = guardrails.New(guardrails.Config{
			Enabled:       cfg.Guardrails.Enabled,
			PIIRedact:     cfg.Guardrails.PIIRedact,
			ContentFilter: cfg.Guardrails.ContentFilter,
			MaxInputLen:   cfg.Guardrails.MaxInputLen,
		})
		log.Printf("Guardrails: enabled (pii=%v filter=%v maxlen=%d)",
			cfg.Guardrails.PIIRedact, cfg.Guardrails.ContentFilter, cfg.Guardrails.MaxInputLen)
	}

	// Initialize adaptive load balancer (from Bifrost)
	if cfg.AdaptiveRouting.Enabled {
		s.balancer = NewAdaptiveBalancer(cfg.AdaptiveRouting)
		log.Printf("Adaptive routing: enabled, window=%d prefer_low_latency=%v",
			cfg.AdaptiveRouting.WindowSize, cfg.AdaptiveRouting.PreferLowLatency)
	}

	// Initialize metrics
	s.metrics = metrics.Get()
	if cfg.Metrics.Enabled {
		log.Printf("Prometheus metrics: enabled at %s", cfg.Metrics.Path)
	}

	// Initialize request logging
	store, err := logging.NewStore("./data", 10000)
	if err != nil {
		log.Printf("WARN: logging store init failed: %v (logging disabled)", err)
	} else {
		s.logStore = store
		log.Printf("Request logging: enabled (SQLite + ring buffer)")
	}

	// Initialize virtual key manager
	vkm, err := vkeys.NewManager("./data")
	if err != nil {
		log.Printf("WARN: virtual keys init failed: %v", err)
	} else {
		s.vkeyMgr = vkm
		log.Printf("Virtual key management: enabled")

		// Initialize spend tracker
		if cfg.SpendTracking.Enabled && store != nil {
			st, err := vkeys.NewSpendTracker(store.DB(), cfg.SpendTracking.DailyBudget)
			if err != nil {
				log.Printf("WARN: spend tracker init failed: %v", err)
			} else {
				s.spendTrack = st
				log.Printf("Spend tracking: enabled, daily_budget=%d tokens", cfg.SpendTracking.DailyBudget)
			}
		}
	}

	// Master key from env or generate one
	s.masterKey = os.Getenv("MASTER_KEY")
	if s.masterKey == "" {
		s.masterKey = "sk-master-change-me"
		log.Printf("WARN: Using default master key. Set MASTER_KEY env var!")
	}

	// Build Fiber app
	s.app = fiber.New(fiber.Config{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
		BodyLimit:    10 * 1024 * 1024,
	})

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"providers": len(s.providers),
			"version":   "2.0.0",
		})
	})

	// List models
	s.app.Get("/v1/models", func(c fiber.Ctx) error {
		type ModelEntry struct {
			ID       string `json:"id"`
			Object   string `json:"object"`
			Provider string `json:"owned_by"`
		}
		var models []ModelEntry
		for name, prov := range s.providers {
			for _, m := range prov.Models() {
				models = append(models, ModelEntry{
					ID:       m,
					Object:   "model",
					Provider: name,
				})
			}
		}
		for name := range s.config.MetaModels {
			models = append(models, ModelEntry{
				ID:       name,
				Object:   "model",
				Provider: "meta",
			})
		}
		return c.JSON(fiber.Map{"object": "list", "data": models})
	})

	// Chat completions
	s.app.Post("/v1/chat/completions", s.handleChatCompletion)

	// Web search (multi-engine)
	s.app.Post("/v1/search", s.handleSearch)

	// Dashboard stats
	s.app.Get("/v1/stats", s.handleStats)

	// Request logs
	s.app.Get("/v1/logs", s.handleLogs)

	// Prometheus metrics (from Bifrost/Portkey)
	if s.config.Metrics.Enabled {
		s.app.Get(s.config.Metrics.Path, s.handleMetrics)
	}

	// Dashboard page (SPA - Mantine at root and /dashboard)
	s.app.Get("/", s.handleDashboard)
	s.app.Get("/dashboard", s.handleDashboard)

	// Virtual key management
	keysGroup := s.app.Group("/v1/keys", s.authMiddleware)
	keysGroup.Post("/", s.handleCreateKey)
	keysGroup.Get("/", s.handleListKeys)
	keysGroup.Delete("/:id", s.handleDeleteKey)
}

func (s *Server) handleChatCompletion(c fiber.Ctx) error {
	startTime := time.Now()

	var req providers.ChatRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		s.metrics.RecordRequest("chat", 0, false)
		return c.Status(400).JSON(fiber.Map{"error": "invalid request: " + err.Error()})
	}

	// ---- Guardrails: PII redaction / content filter (from Portkey) ----
	if s.guard != nil {
		for i, msg := range req.Messages {
			if msg.Role == "user" {
				content, ok := msg.Content.(string)
				if !ok {
					continue
				}
				result := s.guard.CheckInput(content)
				if result.Blocked {
					s.metrics.RecordRequest("chat", time.Since(startTime), false)
					return c.Status(400).JSON(fiber.Map{
						"error":   "request blocked by guardrails",
						"details": result.Messages,
					})
				}
				req.Messages[i].Content = result.Cleaned
			}
		}
	}

	// ---- Token Compression: RTK-style input compression (from 9Router) ----
	if s.config.Compression.InputCompression {
		level := compress.Level(s.config.Compression.Level)
		for i, msg := range req.Messages {
			if msg.Role == "user" {
				content, ok := msg.Content.(string)
				if !ok || content == "" {
					continue
				}
				newContent := compress.CompressInput(content, level)
				if len(newContent) > 0 && len(newContent) < len(content) {
					saved := len(content) - len(newContent)
					log.Printf("[COMPRESS] input saved %d chars", saved)
				}
				if newContent != "" {
					req.Messages[i].Content = newContent
				}
			}
		}
	}

	// ---- Spend Tracking: check budget (from LiteLLM/Portkey) ----
	if s.spendTrack != nil {
		auth := c.Get("Authorization")
		if auth != "" && s.vkeyMgr != nil {
			key := strings.TrimPrefix(auth, "Bearer ")
			vk := s.vkeyMgr.Validate(key)
			if vk != nil && s.spendTrack.IsOverBudget(vk.Prefix) {
				s.metrics.RecordRequest("chat", time.Since(startTime), false)
				return c.Status(429).JSON(fiber.Map{
					"error": "daily spend budget exceeded for this key",
				})
			}
		}
	}

	// ---- Semantic Cache: similarity-based (from Bifrost) ----
	if !req.Stream && s.semCache != nil {
		if cached, ok, sim := s.semCache.Get(&req); ok {
			log.Printf("[SEMCACHE HIT] model=%s similarity=%.3f", req.Model, sim)
			atomic.AddInt64(&s.metrics.SemanticCacheHits, 1)
			if s.logStore != nil {
				s.logStore.Log(&logging.RequestLog{
					Model: req.Model, Provider: "semcache", LatencyMs: 0,
					PromptTokens: cached.Usage.PromptTokens,
					CompTokens:   cached.Usage.CompletionTokens,
					TotalTokens:  cached.Usage.TotalTokens,
					Success: true, CacheHit: true,
				})
			}
			return c.JSON(cached)
		}
		atomic.AddInt64(&s.metrics.SemanticCacheMisses, 1)
	}

	// ---- Exact Cache (SHA-256) ----
	if !req.Stream && s.cache != nil {
		cacheKey := cache.Key(req.Model, &req)
		if cached, ok := s.cache.Get(cacheKey); ok {
			log.Printf("[CACHE HIT] model=%s", req.Model)
			atomic.AddInt64(&s.metrics.CacheHits, 1)
			if s.logStore != nil {
				s.logStore.Log(&logging.RequestLog{
					Model: req.Model, Provider: "cache", LatencyMs: 0,
					PromptTokens: cached.Usage.PromptTokens,
					CompTokens:   cached.Usage.CompletionTokens,
					TotalTokens:  cached.Usage.TotalTokens,
					Success: true, CacheHit: true,
				})
			}
			return c.JSON(cached)
		}
		atomic.AddInt64(&s.metrics.CacheMisses, 1)
	}

	// Resolve targets (handles meta-models)
	targets := s.fallback.ResolveModel(req.Model, s.config.MetaModels)
	if len(targets) == 0 {
		s.metrics.RecordRequest("chat", time.Since(startTime), false)
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("no provider found for model '%s'. Available models at GET /v1/models", req.Model),
		})
	}

	// ---- Adaptive Load Balancing (from Bifrost) ----
	if s.balancer != nil {
		targets = s.balancer.HealthyTargets(targets)
		if len(targets) > 1 {
			s.balancer.SortByScore(targets)
		}
	}

	log.Printf("[REQUEST] model=%s targets=%d stream=%v", req.Model, len(targets), req.Stream)

	// Handle streaming
	if req.Stream {
		s.metrics.RecordStreaming()
		return s.handleStream(c, &req, targets)
	}

	// Non-streaming path
	ctx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
	defer cancel()

	resp, usedProvider, err := s.fallback.ChatWithRetry(ctx, &req, targets)
	latency := time.Since(startTime)

	// Record provider latency for adaptive routing
	if s.balancer != nil && usedProvider != "" {
		s.balancer.RecordResult(usedProvider, latency, err == nil)
	}

	if err != nil {
		log.Printf("[ERROR] model=%s error=%v", req.Model, err)
		s.metrics.RecordRequest("chat", latency, false)
		if s.logStore != nil {
			s.logStore.Log(&logging.RequestLog{
				Model: req.Model, Provider: "", LatencyMs: latency.Milliseconds(),
				Success: false, Error: err.Error(),
			})
		}
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("[SUCCESS] model=%s provider=%s tokens=%d", req.Model, resp.Model, resp.Usage.TotalTokens)

	// ---- Token tracking ----
	s.metrics.RecordRequest("chat", latency, true)
	s.metrics.RecordTokens(int64(resp.Usage.PromptTokens), int64(resp.Usage.CompletionTokens))

	// Spend tracking per key
	if s.spendTrack != nil {
		auth := c.Get("Authorization")
		if auth != "" && s.vkeyMgr != nil {
			key := strings.TrimPrefix(auth, "Bearer ")
			vk := s.vkeyMgr.Validate(key)
			if vk != nil {
				s.spendTrack.RecordUsage(vk.Prefix, int64(resp.Usage.PromptTokens), int64(resp.Usage.CompletionTokens))
			}
		}
	}

	// ---- Caveman output compression (from 9Router) ----
	if s.config.Compression.OutputCompression {
		level := compress.Level(s.config.Compression.Level)
		for i, choice := range resp.Choices {
			content, ok := choice.Message.Content.(string)
			if !ok || content == "" {
				continue
			}
			compressed := compress.CompressOutput(content, level)
			if compressed != "" && len(compressed) < len(content) {
				resp.Choices[i].Message.Content = compressed
			}
		}
	}

	// ---- Guardrails: PII redaction on output (from Portkey) ----
	if s.guard != nil && s.config.Guardrails.PIIRedact {
		for i, choice := range resp.Choices {
			content, ok := choice.Message.Content.(string)
			if !ok {
				continue
			}
			result := s.guard.CheckOutput(content)
			if result.Cleaned != "" {
				resp.Choices[i].Message.Content = result.Cleaned
			}
		}
	}

	// Log the successful request
	if s.logStore != nil {
		s.logStore.Log(&logging.RequestLog{
			Model: req.Model, Provider: usedProvider,
			LatencyMs:    latency.Milliseconds(),
			PromptTokens: resp.Usage.PromptTokens,
			CompTokens:   resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
			Success:      true,
		})
	}

	// ---- Cache the response ----
	if s.cache != nil {
		cacheKey := cache.Key(req.Model, &req)
		s.cache.Set(cacheKey, resp)
	}
	if s.semCache != nil {
		s.semCache.Set(&req, resp)
	}

	return c.JSON(resp)
}

func (s *Server) handleStream(c fiber.Ctx, req *providers.ChatRequest, targets []fallback.RouteTarget) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	ctx := c.Context()

	var lastErr error
	for _, target := range targets {
		// Check tiered pool first, then regular pool
		if tp, ok := s.tieredPools[target.ProviderName]; ok {
			key, available := tp.GetKey()
			if !available {
				continue
			}

			ch, err := target.Provider.ChatStream(ctx, req, key)
			if err != nil {
				if _, isRateLimit := err.(*providers.RateLimitError); isRateLimit {
					tp.MarkFailure(key)
				}
				lastErr = err
				continue
			}

			tp.MarkSuccess(key)
			return s.writeStream(ch, c, target.ProviderName)
		}

		pool := s.keyPools[target.ProviderName]
		if pool == nil {
			continue
		}

		key, ok := pool.GetKey()
		if !ok {
			continue
		}

		ch, err := target.Provider.ChatStream(ctx, req, key)
		if err != nil {
			if _, isRateLimit := err.(*providers.RateLimitError); isRateLimit {
				pool.CoolDown(key)
			}
			lastErr = err
			continue
		}

		pool.MarkUsed(key)
		pool.ResetCooldown(key)

		return s.writeStream(ch, c, target.ProviderName)
	}

	return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("stream failed: %v", lastErr)})
}

func (s *Server) writeStream(ch <-chan providers.StreamChunk, c fiber.Ctx, provider string) error {
	return c.SendStreamWriter(func(w *bufio.Writer) {
		for chunk := range ch {
			if chunk.Error != nil {
				log.Printf("[STREAM ERROR] provider=%s error=%v", provider, chunk.Error)
				return
			}
			if chunk.Done {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				w.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", string(chunk.Data))
			w.Flush()
		}
	})
}

func (s *Server) handleSearch(c fiber.Ctx) error {
	var req struct {
		Query   string   `json:"query"`
		Engines []string `json:"engines,omitempty"`
		Limit   int      `json:"limit,omitempty"`
		// Multi-engine: "auto" uses all configured, or specific: "searxng", "exa", "parallel", "firecrawl"
		Source string `json:"source,omitempty"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query is required"})
	}
	if req.Limit == 0 {
		req.Limit = 8
	}

	// Multi-engine search (aggregates SearXNG + Exa + Parallel + Firecrawl)
	if req.Source == "auto" || (req.Source == "" && s.searchMulti != nil) {
		results, err := s.searchMulti.SearchAll(c.Context(), req.Query, req.Limit)
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(results)
	}

	// Single-engine search
	if s.search == nil {
		return c.Status(501).JSON(fiber.Map{"error": "web search not configured. Set search.searxng_url in config"})
	}

	results, err := s.search.Search(c.Context(), req.Query, req.Engines, req.Limit)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(results)
}

func (s *Server) handleMetrics(c fiber.Ctx) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(s.metrics.PrometheusText())
}

func (s *Server) handleStats(c fiber.Ctx) error {
	stats := fiber.Map{
		"providers":  len(s.providers),
		"models":     0,
		"free_keys":  0,
		"version":    "2.0.0",
	}

	for _, prov := range s.providers {
		stats["models"] = stats["models"].(int) + len(prov.Models())
	}

	for name, pool := range s.keyPools {
		available := pool.AvailableKeys()
		stats["free_keys"] = stats["free_keys"].(int) + available
		stats["provider_"+name] = fiber.Map{
			"available_keys": available,
		}
	}

	for name, tp := range s.tieredPools {
		statData := tp.Stats()
		stats["free_keys"] = stats["free_keys"].(int) + statData["available_keys"].(int)
		stats["tiered_"+name] = statData
	}

	// Cache stats
	if s.cache != nil {
		hits, misses := s.cache.Stats()
		stats["cache_hits"] = hits
		stats["cache_misses"] = misses
	}

	// Semantic cache stats
	if s.semCache != nil {
		hits, misses := s.semCache.Stats()
		stats["semantic_cache_hits"] = hits
		stats["semantic_cache_misses"] = misses
	}

	// Metrics snapshot
	ms := s.metrics.Snapshot()
	stats["total_requests"] = ms["requests_total"]
	stats["avg_latency_ms"] = ms["avg_latency_ms"]
	stats["tokens_input"] = ms["tokens_input_total"]
	stats["tokens_output"] = ms["tokens_output_total"]

	return c.JSON(stats)
}

func (s *Server) handleLogs(c fiber.Ctx) error {
	if s.logStore == nil {
		return c.JSON(fiber.Map{"error": "logging not enabled"})
	}
	entries := s.logStore.Recent(200)
	if entries == nil {
		entries = []*logging.RequestLog{}
	}
	return c.JSON(fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

func (s *Server) handleDashboard(c fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(dashboardHTML)
}

// ---- Auth Middleware ----

func (s *Server) authMiddleware(c fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return c.Status(401).JSON(fiber.Map{"error": "missing Authorization header"})
	}
	key := strings.TrimPrefix(auth, "Bearer ")
	if key != s.masterKey {
		return c.Status(403).JSON(fiber.Map{"error": "invalid master key"})
	}
	return c.Next()
}

// ---- Virtual Key Management ----

func (s *Server) handleCreateKey(c fiber.Ctx) error {
	if s.vkeyMgr == nil {
		return c.Status(501).JSON(fiber.Map{"error": "virtual key management not enabled"})
	}
	var req struct {
		Label         string   `json:"label"`
		AllowedModels []string `json:"allowed_models,omitempty"`
		RPM           int      `json:"rpm,omitempty"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Label == "" {
		req.Label = "unnamed"
	}

	key, err := s.vkeyMgr.CreateKey(req.Label, req.AllowedModels, req.RPM)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"key":    key,
		"label":  req.Label,
		"prefix": key[:11],
	})
}

func (s *Server) handleListKeys(c fiber.Ctx) error {
	if s.vkeyMgr == nil {
		return c.Status(501).JSON(fiber.Map{"error": "virtual key management not enabled"})
	}
	keys := s.vkeyMgr.List()
	return c.JSON(fiber.Map{"keys": keys, "count": len(keys)})
}

func (s *Server) handleDeleteKey(c fiber.Ctx) error {
	if s.vkeyMgr == nil {
		return c.Status(501).JSON(fiber.Map{"error": "virtual key management not enabled"})
	}
	id := c.Params("id")
	if err := s.vkeyMgr.DeleteKey(id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"deleted": id})
}

// Listen starts the HTTP server
func (s *Server) Listen(addr string) error {
	log.Printf("Super Gateway v2.0.0 starting on %s", addr)
	log.Printf("Providers: %d", len(s.providers))
	freeCount := 0
	for _, p := range s.providers {
		if p.IsFree() {
			freeCount++
		}
	}
	log.Printf("Free providers: %d, Paid providers: %d", freeCount, len(s.providers)-freeCount)

	totalKeys := 0
	for _, pool := range s.keyPools {
		totalKeys += pool.AvailableKeys()
	}
	for _, tp := range s.tieredPools {
		totalKeys += tp.AvailableKeys()
	}
	log.Printf("Total available keys: %d", totalKeys)
	log.Printf("Meta-models: %d", len(s.config.MetaModels))
	for name := range s.config.MetaModels {
		log.Printf("  meta: %s", name)
	}

	// Start background semantic cache purger
	if s.semCache != nil {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				s.semCache.PurgeExpired()
			}
		}()
	}

	return s.app.Listen(addr)
}

// ProxyDialer helper — kept for external use
var _ = proxy.NewFastHTTPClient
var _ = strings.TrimSpace
