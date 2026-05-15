package metrics

import (
	"expvar"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects prometheus-style counters and gauges
type Metrics struct {
	mu sync.RWMutex

	// Counters
	RequestsTotal      int64
	RequestsSuccess    int64
	RequestsFailed     int64
	TokensInputTotal   int64
	TokensOutputTotal  int64
	CacheHits          int64
	CacheMisses        int64
	SemanticCacheHits  int64
	SemanticCacheMisses int64
	StreamingTotal     int64

	// Latency tracking
	latencyBuckets [15]int64 // 0-50ms, 50-100, 100-200, 200-500, 500-1s, 1-2s, 2-5s, 5-10s, >10s
	totalLatencyMs int64
	totalRequests  int64

	// Per-endpoint metrics
	EndpointCounts map[string]int64

	// Provider health
	ProviderLatency map[string]*providerStats
}

type providerStats struct {
	mu         sync.RWMutex
	lastLatency time.Duration
	avgLatency  time.Duration
	totalCalls  int64
	failedCalls int64
	lastError   string
	lastSuccess time.Time
	healthy     bool
}

// Global metrics instance
var (
	global     *Metrics
	globalOnce sync.Once
)

// Get returns the global metrics instance
func Get() *Metrics {
	globalOnce.Do(func() {
		global = &Metrics{
			EndpointCounts:  make(map[string]int64),
			ProviderLatency: make(map[string]*providerStats),
		}
		// Publish to expvar for /debug/vars
		expvar.Publish("gateway", expvar.Func(func() interface{} {
			return global.Snapshot()
		}))
	})
	return global
}

// Snapshot returns a thread-safe copy of all metrics
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := map[string]interface{}{
		"requests_total":       atomic.LoadInt64(&m.RequestsTotal),
		"requests_success":     atomic.LoadInt64(&m.RequestsSuccess),
		"requests_failed":      atomic.LoadInt64(&m.RequestsFailed),
		"tokens_input_total":   atomic.LoadInt64(&m.TokensInputTotal),
		"tokens_output_total":  atomic.LoadInt64(&m.TokensOutputTotal),
		"cache_hits":           atomic.LoadInt64(&m.CacheHits),
		"cache_misses":         atomic.LoadInt64(&m.CacheMisses),
		"semantic_cache_hits":  atomic.LoadInt64(&m.SemanticCacheHits),
		"semantic_cache_misses": atomic.LoadInt64(&m.SemanticCacheMisses),
		"streaming_total":      atomic.LoadInt64(&m.StreamingTotal),
		"avg_latency_ms":       m.AvgLatency(),
	}

	// Copy endpoint counts
	endpoints := make(map[string]int64)
	for k, v := range m.EndpointCounts {
		endpoints[k] = v
	}
	snap["endpoints"] = endpoints

	// Copy provider health
	health := make(map[string]map[string]interface{})
	for k, ps := range m.ProviderLatency {
		ps.mu.RLock()
		health[k] = map[string]interface{}{
			"avg_latency_ms": ps.avgLatency.Milliseconds(),
			"total_calls":    ps.totalCalls,
			"failed_calls":   ps.failedCalls,
			"healthy":        ps.healthy,
			"last_error":     ps.lastError,
		}
		ps.mu.RUnlock()
	}
	snap["providers"] = health

	return snap
}

// PrometheusText returns metrics in Prometheus text format
func (m *Metrics) PrometheusText() string {
	snap := m.Snapshot()
	var b strings.Builder

	b.WriteString("# HELP gateway_requests_total Total request count\n")
	b.WriteString("# TYPE gateway_requests_total counter\n")
	b = appendPromCounter(b, "gateway_requests_total", snap["requests_total"])

	b.WriteString("# HELP gateway_requests_success Successful requests\n")
	b.WriteString("# TYPE gateway_requests_success counter\n")
	b = appendPromCounter(b, "gateway_requests_success", snap["requests_success"])

	b.WriteString("# HELP gateway_requests_failed Failed requests\n")
	b.WriteString("# TYPE gateway_requests_failed counter\n")
	b = appendPromCounter(b, "gateway_requests_failed", snap["requests_failed"])

	b.WriteString("# HELP gateway_tokens_input_total Input tokens\n")
	b.WriteString("# TYPE gateway_tokens_input_total counter\n")
	b = appendPromCounter(b, "gateway_tokens_input_total", snap["tokens_input_total"])

	b.WriteString("# HELP gateway_tokens_output_total Output tokens\n")
	b.WriteString("# TYPE gateway_tokens_output_total counter\n")
	b = appendPromCounter(b, "gateway_tokens_output_total", snap["tokens_output_total"])

	b.WriteString("# HELP gateway_cache_hits Cache hits\n")
	b.WriteString("# TYPE gateway_cache_hits counter\n")
	b = appendPromCounter(b, "gateway_cache_hits", snap["cache_hits"])
	b = appendPromCounter(b, "gateway_cache_misses", snap["cache_misses"])
	b = appendPromCounter(b, "gateway_semantic_cache_hits", snap["semantic_cache_hits"])
	b = appendPromCounter(b, "gateway_semantic_cache_misses", snap["semantic_cache_misses"])

	b.WriteString("# HELP gateway_latency_ms Average latency\n")
	b.WriteString("# TYPE gateway_latency_ms gauge\n")
	b = appendPromGauge(b, "gateway_latency_ms", snap["avg_latency_ms"])

	// Per-provider metrics
	if providers, ok := snap["providers"].(map[string]map[string]interface{}); ok {
		for name, stats := range providers {
			b = appendPromGauge(b, `gateway_provider_health{provider="`+name+`"}`, stats["healthy"])
			b = appendPromGauge(b, `gateway_provider_avg_latency{provider="`+name+`"}`, stats["avg_latency_ms"])
			b = appendPromCounter(b, `gateway_provider_calls{provider="`+name+`"}`, stats["total_calls"])
		}
	}

	return b.String()
}

func appendPromCounter(b strings.Builder, name string, val interface{}) strings.Builder {
	b.WriteString(name)
	b.WriteByte(' ')
	switch v := val.(type) {
	case int64:
		b.WriteString(fmt.Sprintf("%d", v))
	case float64:
		b.WriteString(fmt.Sprintf("%.3f", v))
	}
	b.WriteByte('\n')
	return b
}

func appendPromGauge(b strings.Builder, name string, val interface{}) strings.Builder {
	return appendPromCounter(b, name, val)
}

// AvgLatency computes average latency across all requests
func (m *Metrics) AvgLatency() float64 {
	total := atomic.LoadInt64(&m.totalRequests)
	if total == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&m.totalLatencyMs)) / float64(total)
}

// RecordRequest records a completed request
func (m *Metrics) RecordRequest(endpoint string, latency time.Duration, success bool) {
	atomic.AddInt64(&m.RequestsTotal, 1)
	atomic.AddInt64(&m.totalLatencyMs, latency.Milliseconds())
	atomic.AddInt64(&m.totalRequests, 1)
	if success {
		atomic.AddInt64(&m.RequestsSuccess, 1)
	} else {
		atomic.AddInt64(&m.RequestsFailed, 1)
	}

	m.mu.Lock()
	m.EndpointCounts[endpoint]++
	m.mu.Unlock()
}

// RecordTokens records token usage
func (m *Metrics) RecordTokens(input, output int64) {
	atomic.AddInt64(&m.TokensInputTotal, input)
	atomic.AddInt64(&m.TokensOutputTotal, output)
}

// RecordProviderLatency tracks per-provider latency
func (m *Metrics) RecordProviderLatency(provider string, latency time.Duration, success bool, errMsg string) {
	m.mu.Lock()
	ps, ok := m.ProviderLatency[provider]
	if !ok {
		ps = &providerStats{}
		m.ProviderLatency[provider] = ps
	}
	m.mu.Unlock()

	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.totalCalls++
	ps.lastLatency = latency
	if !success {
		ps.failedCalls++
		ps.lastError = errMsg
		ps.healthy = false
	} else {
		ps.lastSuccess = time.Now()
		ps.lastError = ""
		ps.healthy = true
	}

	// Exponential moving average
	if ps.avgLatency == 0 {
		ps.avgLatency = latency
	} else {
		ps.avgLatency = time.Duration(float64(ps.avgLatency)*0.9 + float64(latency)*0.1)
	}
}

// RecordStreaming increments streaming counter
func (m *Metrics) RecordStreaming() {
	atomic.AddInt64(&m.StreamingTotal, 1)
}
