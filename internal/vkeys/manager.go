package vkeys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// VirtualKey represents a client API key with permissions
type VirtualKey struct {
	KeyHash      string    `json:"key_hash"`      // SHA-256 of the key
	Label        string    `json:"label"`          // human-readable name
	Prefix       string    `json:"prefix"`         // first 8 chars of key (for display)
	AllowedModels []string `json:"allowed_models"` // empty = all models
	RPM          int       `json:"rpm"`            // requests per minute (0 = unlimited)
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	// Runtime counters
	requestCount atomic.Int64
	windowStart  time.Time
	mu           sync.Mutex
}

// Manager handles virtual key lifecycle
type Manager struct {
	mu       sync.RWMutex
	keys     map[string]*VirtualKey // key_hash → VirtualKey
	filePath string
}

// NewManager loads or creates the virtual key store
func NewManager(dataDir string) (*Manager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(dataDir, "virtual_keys.json")
	m := &Manager{
		keys:     make(map[string]*VirtualKey),
		filePath: filePath,
	}
	m.load()
	return m, nil
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return // no file yet, start fresh
	}
	var keys []*VirtualKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return
	}
	for _, k := range keys {
		m.keys[k.KeyHash] = k
	}
}

func (m *Manager) save() error {
	m.mu.RLock()
	keys := make([]*VirtualKey, 0, len(m.keys))
	for _, k := range m.keys {
		keys = append(keys, k)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0600)
}

// CreateKey generates a new virtual API key
func (m *Manager) CreateKey(label string, allowedModels []string, rpm int) (string, error) {
	// Generate random key: sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	key := "sk-" + hex.EncodeToString(bytes)
	hash := hashKey(key)

	vk := &VirtualKey{
		KeyHash:       hash,
		Label:         label,
		Prefix:        key[:11], // "sk-xxxxxxxx"
		AllowedModels: allowedModels,
		RPM:           rpm,
		CreatedAt:     time.Now(),
	}

	m.mu.Lock()
	m.keys[hash] = vk
	m.mu.Unlock()

	if err := m.save(); err != nil {
		return "", fmt.Errorf("save failed: %w", err)
	}
	return key, nil
}

// Validate checks if a key exists and returns it
func (m *Manager) Validate(key string) *VirtualKey {
	hash := hashKey(key)
	m.mu.RLock()
	vk := m.keys[hash]
	m.mu.RUnlock()
	if vk == nil {
		return nil
	}
	// Check expiry
	if !vk.ExpiresAt.IsZero() && time.Now().After(vk.ExpiresAt) {
		return nil
	}
	return vk
}

// CheckRateLimit returns true if the key has exceeded its RPM
func (vk *VirtualKey) CheckRateLimit() bool {
	if vk.RPM == 0 {
		return true // unlimited
	}
	vk.mu.Lock()
	defer vk.mu.Unlock()

	now := time.Now()
	if now.Sub(vk.windowStart) >= time.Minute {
		// Reset window
		vk.windowStart = now
		vk.requestCount.Store(0)
	}
	count := vk.requestCount.Add(1)
	return count <= int64(vk.RPM)
}

// CanUseModel checks if a model is allowed for this key
func (vk *VirtualKey) CanUseModel(model string) bool {
	if len(vk.AllowedModels) == 0 {
		return true // all models allowed
	}
	for _, m := range vk.AllowedModels {
		if m == model {
			return true
		}
	}
	return false
}

// DeleteKey removes a virtual key
func (m *Manager) DeleteKey(prefix string) error {
	m.mu.Lock()
	var found string
	for hash, vk := range m.keys {
		if vk.Prefix == prefix || vk.Label == prefix {
			found = hash
			break
		}
	}
	if found != "" {
		delete(m.keys, found)
	}
	m.mu.Unlock()

	if found == "" {
		return fmt.Errorf("key not found: %s", prefix)
	}
	return m.save()
}

// List returns all virtual keys (without hashes, just metadata)
func (m *Manager) List() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []map[string]interface{}
	for _, vk := range m.keys {
		results = append(results, map[string]interface{}{
			"label":          vk.Label,
			"prefix":         vk.Prefix,
			"allowed_models": vk.AllowedModels,
			"rpm":            vk.RPM,
			"created_at":     vk.CreatedAt,
			"expires_at":     vk.ExpiresAt,
		})
	}
	return results
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
