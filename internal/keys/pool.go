package keys

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// KeyState tracks a single API key's usage and cooldown status
type KeyState struct {
	Key          string
	UseCount     atomic.Int64
	LastUsed     time.Time
	CoolDownUntil time.Time
	mu           sync.Mutex
}

// Pool manages a set of keys for one provider with rotation strategy
type Pool struct {
	keys      []*KeyState
	strategy  string
	cooldown  time.Duration
	maxCDown  time.Duration
	mu        sync.RWMutex
	rrCounter atomic.Int64
}

// NewPool creates a key pool
func NewPool(apiKeys []string, strategy string, cooldown, maxCooldown time.Duration) *Pool {
	keys := make([]*KeyState, len(apiKeys))
	for i, k := range apiKeys {
		keys[i] = &KeyState{Key: k}
	}
	return &Pool{
		keys:     keys,
		strategy: strategy,
		cooldown: cooldown,
		maxCDown: maxCooldown,
	}
}

// GetKey returns the next available key based on strategy
func (p *Pool) GetKey() (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.keys) == 0 {
		return "", false
	}

	now := time.Now()
	var candidates []*KeyState

	// Filter out keys in cooldown
	for _, ks := range p.keys {
		if now.After(ks.CoolDownUntil) {
			candidates = append(candidates, ks)
		}
	}

	if len(candidates) == 0 {
		return "", false // all in cooldown
	}

	switch p.strategy {
	case "least_used":
		return p.leastUsed(candidates), true
	case "weighted":
		return p.weightedPick(candidates), true
	case "random":
		return candidates[rand.Intn(len(candidates))].Key, true
	default: // round_robin
		idx := p.rrCounter.Add(1) % int64(len(candidates))
		return candidates[idx].Key, true
	}
}

func (p *Pool) leastUsed(candidates []*KeyState) string {
	best := candidates[0]
	for _, ks := range candidates[1:] {
		if ks.UseCount.Load() < best.UseCount.Load() {
			best = ks
		}
	}
	return best.Key
}

func (p *Pool) weightedPick(candidates []*KeyState) string {
	// Inverse weight based on usage count
	totalWeight := 0.0
	weights := make([]float64, len(candidates))
	for i, ks := range candidates {
		w := 1.0 / (float64(ks.UseCount.Load()) + 1.0)
		weights[i] = w
		totalWeight += w
	}
	r := rand.Float64() * totalWeight
	cum := 0.0
	for i, w := range weights {
		cum += w
		if r <= cum {
			return candidates[i].Key
		}
	}
	return candidates[len(candidates)-1].Key
}

// MarkUsed increments usage counter
func (p *Pool) MarkUsed(key string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, ks := range p.keys {
		if ks.Key == key {
			ks.UseCount.Add(1)
			ks.LastUsed = time.Now()
			return
		}
	}
}

// CoolDown puts a key in cooldown (exponential backoff on repeated 429s)
func (p *Pool) CoolDown(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ks := range p.keys {
		if ks.Key == key {
			// Exponential backoff: double cooldown each time, cap at max
			currentCooldown := p.cooldown
			if !ks.CoolDownUntil.IsZero() && ks.CoolDownUntil.After(time.Now()) {
				// Already in cooldown — double it
				currentCooldown = time.Until(ks.CoolDownUntil) * 2
				if currentCooldown > p.maxCDown {
					currentCooldown = p.maxCDown
				}
			}
			ks.CoolDownUntil = time.Now().Add(currentCooldown)
			return
		}
	}
}

// ResetCooldown clears cooldown for a key (e.g. after successful request)
func (p *Pool) ResetCooldown(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ks := range p.keys {
		if ks.Key == key {
			ks.CoolDownUntil = time.Time{}
			return
		}
	}
}

// AvailableKeys returns count of non-cooldowned keys
func (p *Pool) AvailableKeys() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	now := time.Now()
	count := 0
	for _, ks := range p.keys {
		if now.After(ks.CoolDownUntil) {
			count++
		}
	}
	return count
}
