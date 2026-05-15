package router

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/fallback"
)

// AdaptiveBalancer routes to the healthiest/fastest provider based on
// recent latency and error rate data (from Bifrost)
type AdaptiveBalancer struct {
	mu         sync.RWMutex
	windowSize int
	preferLow  bool
	stats      map[string]*ProviderStats
}

// ProviderStats tracks recent performance for a provider
type ProviderStats struct {
	Latencies   []time.Duration
	Errors      int
	TotalCalls  int
	LastUsed    time.Time
	ConsecutiveErrors int
}

// NewAdaptiveBalancer creates a new balancer
func NewAdaptiveBalancer(cfg config.AdaptiveRoutingConfig) *AdaptiveBalancer {
	return &AdaptiveBalancer{
		windowSize: cfg.WindowSize,
		preferLow:  cfg.PreferLowLatency,
		stats:      make(map[string]*ProviderStats),
	}
}

// RecordResult records the outcome of a provider call
func (b *AdaptiveBalancer) RecordResult(providerID string, latency time.Duration, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ps, ok := b.stats[providerID]
	if !ok {
		ps = &ProviderStats{
			Latencies: make([]time.Duration, 0, b.windowSize),
		}
		b.stats[providerID] = ps
	}

	// Append latency
	ps.Latencies = append(ps.Latencies, latency)
	if len(ps.Latencies) > b.windowSize {
		ps.Latencies = ps.Latencies[len(ps.Latencies)-b.windowSize:]
	}

	ps.TotalCalls++
	ps.LastUsed = time.Now()

	if !success {
		ps.Errors++
		ps.ConsecutiveErrors++
	} else {
		ps.ConsecutiveErrors = 0
	}
}

// ScoreTarget computes a routing score for a provider (0-100, higher = better)
func (b *AdaptiveBalancer) ScoreTarget(target fallback.RouteTarget) float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	ps, ok := b.stats[target.ProviderName]
	if !ok {
		// No data yet: give medium score, prefer free tier
		if target.IsFree {
			return 60.0
		}
		return 40.0
	}

	if ps.TotalCalls == 0 {
		if target.IsFree {
			return 60.0
		}
		return 40.0
	}

	// Error rate score (0-40 points)
	errorRate := float64(ps.Errors) / float64(ps.TotalCalls)
	errorScore := 40.0 * (1.0 - errorRate)

	// Consecutive errors penalty
	if ps.ConsecutiveErrors > 2 {
		errorScore -= float64(ps.ConsecutiveErrors) * 10.0
		if errorScore < 0 {
			errorScore = 0
		}
	}

	// Latency score (0-40 points)
	latencyScore := 20.0 // default medium
	if b.preferLow && len(ps.Latencies) > 0 {
		avgLatency := avgDuration(ps.Latencies)
		switch {
		case avgLatency < 500*time.Millisecond:
			latencyScore = 40.0
		case avgLatency < 2*time.Second:
			latencyScore = 30.0
		case avgLatency < 5*time.Second:
			latencyScore = 20.0
		default:
			latencyScore = 10.0
		}
	}

	// Tier bonus (0-20 points)
	tierScore := 10.0
	if target.IsFree {
		tierScore = 20.0
	}

	return errorScore + latencyScore + tierScore
}

// SortByScore sorts targets by adaptive score (highest first)
func (b *AdaptiveBalancer) SortByScore(targets []fallback.RouteTarget) {
	sort.Slice(targets, func(i, j int) bool {
		return b.ScoreTarget(targets[i]) > b.ScoreTarget(targets[j])
	})
}

// HealthyTargets filters targets that haven't been consistently failing
func (b *AdaptiveBalancer) HealthyTargets(targets []fallback.RouteTarget) []fallback.RouteTarget {
	b.mu.RLock()
	defer b.mu.RUnlock()

	healthy := make([]fallback.RouteTarget, 0, len(targets))
	for _, t := range targets {
		ps, ok := b.stats[t.ProviderName]
		if !ok || ps.ConsecutiveErrors < 10 {
			healthy = append(healthy, t)
		}
	}

	if len(healthy) == 0 && len(targets) > 0 {
		// All unhealthy: reset and return all
		for _, ps := range b.stats {
			ps.ConsecutiveErrors = 0
			ps.Errors = 0
		}
		return targets
	}

	return healthy
}

// Reset clears all stats
func (b *AdaptiveBalancer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stats = make(map[string]*ProviderStats)
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	if len(durations) == 0 {
		return 0
	}
	return time.Duration(int64(total) / int64(len(durations)))
}

// ProgressTracker tracks retry progress for exponential backoff
type ProgressTracker struct {
	mu          sync.Mutex
	attempts    map[string]int
	lastSuccess map[string]time.Time
}

// NewProgressTracker creates a progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		attempts:    make(map[string]int),
		lastSuccess: make(map[string]time.Time),
	}
}

// NextDelay computes the next backoff delay
func (pt *ProgressTracker) NextDelay(providerID string, baseDelay time.Duration, backoffFactor float64, maxDelay time.Duration) time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.attempts[providerID]++
	attempts := pt.attempts[providerID]
	delay := float64(baseDelay) * math.Pow(backoffFactor, float64(attempts-1))
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}
	return time.Duration(delay)
}

// RecordSuccess resets the attempt counter for a provider
func (pt *ProgressTracker) RecordSuccess(providerID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.attempts[providerID] = 0
	pt.lastSuccess[providerID] = time.Now()
}
