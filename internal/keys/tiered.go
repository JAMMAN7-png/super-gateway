package keys

import (
	"sync"
	"time"
)

// TieredCooldownKey wraps a key with escalating cooldown levels
// (from LLM-API-Key-Proxy tiered locking)
type TieredKey struct {
	Key           string
	Level         int           // 0=normal, 1=short, 2=medium, 3=long, 4=dead
	CooldownUntil time.Time
	Fails         int
	Successes     int
}

// TieredPool manages keys with escalating cooldown tiers
type TieredPool struct {
	mu          sync.Mutex
	keys        map[string]*TieredKey
	tiers       []time.Duration // cooldown durations per level
	maxLevel    int
}

// NewTieredPool creates a tiered key pool
// tiers: escalating cooldown durations for levels 1,2,3,4+
func NewTieredPool(apiKeys []string, baseCooldown, maxCooldown time.Duration) *TieredPool {
	tp := &TieredPool{
		keys:     make(map[string]*TieredKey),
		tiers: []time.Duration{
			baseCooldown,                           // level 1
			baseCooldown * 2,                       // level 2
			maxCooldown,                            // level 3
			maxCooldown * 2,                        // level 4 (nearly dead)
		},
		maxLevel: 4,
	}

	for _, k := range apiKeys {
		tp.keys[k] = &TieredKey{Key: k}
	}

	return tp
}

// GetKey returns the best available key (lowest level, not in cooldown)
func (tp *TieredPool) GetKey() (string, bool) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	now := time.Now()
	var best *TieredKey

	for _, tk := range tp.keys {
		if now.Before(tk.CooldownUntil) {
			continue
		}
		if best == nil || tk.Level < best.Level || (tk.Level == best.Level && tk.Successes > best.Successes) {
			best = tk
		}
	}

	if best == nil {
		// All keys in cooldown: return the one closest to being available
		var soonest *TieredKey
		for _, tk := range tp.keys {
			if soonest == nil || tk.CooldownUntil.Before(soonest.CooldownUntil) {
				soonest = tk
			}
		}
		if soonest != nil {
			return soonest.Key, false
		}
		return "", false
	}

	return best.Key, true
}

// MarkFailure escalates the key's cooldown level
func (tp *TieredPool) MarkFailure(key string) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tk, ok := tp.keys[key]
	if !ok {
		return
	}

	tk.Fails++
	if tk.Fails >= 3 && tk.Level < tp.maxLevel {
		tk.Level++
		tk.Fails = 0
	}
	tk.Successes = 0

	// Set cooldown
	level := tk.Level
	if level >= len(tp.tiers) {
		level = len(tp.tiers) - 1
	}
	tk.CooldownUntil = time.Now().Add(tp.tiers[level])
}

// MarkSuccess reduces cooldown level on success
func (tp *TieredPool) MarkSuccess(key string) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tk, ok := tp.keys[key]
	if !ok {
		return
	}

	tk.Successes++
	tk.Fails = 0

	// Reduce level after N consecutive successes
	if tk.Successes >= 5 && tk.Level > 0 {
		tk.Level--
		tk.Successes = 0
	}

	tk.CooldownUntil = time.Time{} // clear cooldown
}

// AvailableKeys returns count of non-cooldowned keys
func (tp *TieredPool) AvailableKeys() int {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	now := time.Now()
	count := 0
	for _, tk := range tp.keys {
		if !now.Before(tk.CooldownUntil) {
			count++
		}
	}
	return count
}

// Stats returns tier distribution for monitoring
func (tp *TieredPool) Stats() map[string]interface{} {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	levelCounts := make(map[int]int)
	total := len(tp.keys)
	available := 0
	now := time.Now()

	for _, tk := range tp.keys {
		levelCounts[tk.Level]++
		if !now.Before(tk.CooldownUntil) {
			available++
		}
	}

	return map[string]interface{}{
		"total_keys":      total,
		"available_keys":  available,
		"level_distribution": levelCounts,
	}
}
