package logging

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// RequestLog records a single API request
type RequestLog struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	LatencyMs    int64     `json:"latency_ms"`
	PromptTokens int       `json:"prompt_tokens"`
	CompTokens   int       `json:"comp_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	Stream       bool      `json:"stream"`
	CacheHit     bool      `json:"cache_hit"`
}

// Store persists and queries request logs
type Store struct {
	mu       sync.RWMutex
	db       *sql.DB
	ringBuf  []*RequestLog
	ringPos  int
	ringSize int
	total    int64
}

// NewStore opens or creates a SQLite store + in-memory ring buffer
func NewStore(dataDir string, ringSize int) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "gateway.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			model TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			latency_ms INTEGER DEFAULT 0,
			prompt_tokens INTEGER DEFAULT 0,
			comp_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			success INTEGER DEFAULT 1,
			error TEXT DEFAULT '',
			stream INTEGER DEFAULT 0,
			cache_hit INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_requests_time ON requests(timestamp);
		CREATE INDEX IF NOT EXISTS idx_requests_model ON requests(model);
		CREATE INDEX IF NOT EXISTS idx_requests_provider ON requests(provider);
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	s := &Store{
		db:       db,
		ringBuf:  make([]*RequestLog, ringSize),
		ringSize: ringSize,
	}

	// Load recent entries into ring buffer
	rows, err := db.Query("SELECT id, timestamp, model, provider, latency_ms, prompt_tokens, comp_tokens, total_tokens, success, error, stream, cache_hit FROM requests ORDER BY id DESC LIMIT ?", ringSize)
	if err == nil {
		defer rows.Close()
		var entries []*RequestLog
		for rows.Next() {
			var r RequestLog
			rows.Scan(&r.ID, &r.Timestamp, &r.Model, &r.Provider, &r.LatencyMs,
				&r.PromptTokens, &r.CompTokens, &r.TotalTokens, &r.Success, &r.Error, &r.Stream, &r.CacheHit)
			entries = append(entries, &r)
		}
		// Reverse to get chronological order
		for i := len(entries) - 1; i >= 0; i-- {
			s.ringBuf[s.ringPos%ringSize] = entries[i]
			s.ringPos++
		}
		s.total = int64(len(entries))
	}

	return s, nil
}

// Log inserts a request record (async-safe via ring buffer, batch-flushed to SQLite)
func (s *Store) Log(r *RequestLog) {
	r.Timestamp = time.Now()

	s.mu.Lock()
	s.ringBuf[s.ringPos%s.ringSize] = r
	s.ringPos++
	s.total++
	s.mu.Unlock()

	// Async persist to SQLite (fire and forget)
	go s.persist(r)
}

func (s *Store) persist(r *RequestLog) {
	_, err := s.db.Exec(
		`INSERT INTO requests (timestamp, model, provider, latency_ms, prompt_tokens, comp_tokens, total_tokens, success, error, stream, cache_hit)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Timestamp, r.Model, r.Provider, r.LatencyMs, r.PromptTokens, r.CompTokens, r.TotalTokens,
		boolToInt(r.Success), r.Error, boolToInt(r.Stream), boolToInt(r.CacheHit),
	)
	if err != nil {
		// SQLite errors in async path are logged to stderr
		os.Stderr.WriteString("logging: persist error: " + err.Error() + "\n")
	}
}

// Recent returns the last N entries from the ring buffer
func (s *Store) Recent(n int) []*RequestLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n > s.ringSize {
		n = s.ringSize
	}

	results := make([]*RequestLog, 0, n)
	start := s.ringPos - n
	if start < 0 {
		start = 0
	}
	for i := start; i < s.ringPos; i++ {
		if entry := s.ringBuf[i%s.ringSize]; entry != nil {
			results = append(results, entry)
		}
	}
	return results
}

// Stats returns aggregate statistics
func (s *Store) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providerStats := make(map[string]map[string]interface{})
	modelCounts := make(map[string]int)
	var totalSuccess, totalFail int64
	var totalLatency int64
	var count int64

	// Scan ring buffer for live stats
	for i := s.ringPos - s.ringSize; i < s.ringPos; i++ {
		if i < 0 {
			continue
		}
		entry := s.ringBuf[i%s.ringSize]
		if entry == nil {
			continue
		}
		count++
		if entry.Success {
			totalSuccess++
		} else {
			totalFail++
		}
		totalLatency += entry.LatencyMs
		modelCounts[entry.Model]++

		if _, ok := providerStats[entry.Provider]; !ok {
			providerStats[entry.Provider] = map[string]interface{}{
				"requests": 0, "success": 0, "fail": 0, "total_latency_ms": int64(0),
			}
		}
		ps := providerStats[entry.Provider]
		ps["requests"] = ps["requests"].(int) + 1
		if entry.Success {
			ps["success"] = ps["success"].(int) + 1
		} else {
			ps["fail"] = ps["fail"].(int) + 1
		}
		ps["total_latency_ms"] = ps["total_latency_ms"].(int64) + entry.LatencyMs
	}

	avgLatency := float64(0)
	if count > 0 {
		avgLatency = float64(totalLatency) / float64(count)
	}

	return map[string]interface{}{
		"total_requests": count,
		"success":        totalSuccess,
		"fail":           totalFail,
		"avg_latency_ms": avgLatency,
		"top_models":     modelCounts,
		"providers":      providerStats,
	}
}

// QueryDB runs a custom SQL query for history beyond ring buffer
func (s *Store) QueryDB(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = vals[i]
		}
		results = append(results, row)
	}
	return results, nil
}

// DB returns the underlying SQLite database handle for external queries
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close shuts down the store
func (s *Store) Close() error {
	return s.db.Close()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
