package vkeys

import (
	"database/sql"
	"sync"
	"time"
)

// SpendRecord tracks token usage for a virtual key
type SpendRecord struct {
	KeyPrefix   string    `json:"key_prefix"`
	Date        string    `json:"date"` // YYYY-MM-DD
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	TotalTokens  int64    `json:"total_tokens"`
	RequestCount int64    `json:"request_count"`
}

// SpendTracker manages per-key token budgets
type SpendTracker struct {
	mu          sync.RWMutex
	db          *sql.DB
	dailyBudget int64
	cache       map[string]*SpendRecord // key: keyPrefix+date
}

// NewSpendTracker creates a spend tracker
func NewSpendTracker(db *sql.DB, dailyBudget int64) (*SpendTracker, error) {
	st := &SpendTracker{
		db:          db,
		dailyBudget: dailyBudget,
		cache:       make(map[string]*SpendRecord),
	}

	if err := st.initDB(); err != nil {
		return nil, err
	}

	return st, nil
}

func (st *SpendTracker) initDB() error {
	_, err := st.db.Exec(`
		CREATE TABLE IF NOT EXISTS spend_records (
			key_prefix TEXT NOT NULL,
			date TEXT NOT NULL,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			request_count INTEGER DEFAULT 0,
			PRIMARY KEY (key_prefix, date)
		)
	`)
	return err
}

// RecordUsage records token usage for a given day
func (st *SpendTracker) RecordUsage(keyPrefix string, inputTokens, outputTokens int64) {
	date := time.Now().Format("2006-01-02")
	cacheKey := keyPrefix + ":" + date

	st.mu.Lock()
	rec, ok := st.cache[cacheKey]
	if !ok {
		rec = &SpendRecord{
			KeyPrefix: keyPrefix,
			Date:      date,
		}
		st.cache[cacheKey] = rec
	}
	rec.InputTokens += inputTokens
	rec.OutputTokens += outputTokens
	rec.TotalTokens += inputTokens + outputTokens
	rec.RequestCount++
	st.mu.Unlock()

	// Async persist to DB
	go func() {
		_, err := st.db.Exec(`
			INSERT INTO spend_records (key_prefix, date, input_tokens, output_tokens, total_tokens, request_count)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(key_prefix, date) DO UPDATE SET
				input_tokens = input_tokens + ?,
				output_tokens = output_tokens + ?,
				total_tokens = total_tokens + ?,
				request_count = request_count + ?
		`, keyPrefix, date, inputTokens, outputTokens, inputTokens+outputTokens, 1,
			inputTokens, outputTokens, inputTokens+outputTokens, 1)
		if err != nil {
			// Log silently - don't block on spend tracking
		}
	}()
}

// GetDailyUsage returns current day's usage for a key
func (st *SpendTracker) GetDailyUsage(keyPrefix string) *SpendRecord {
	date := time.Now().Format("2006-01-02")
	cacheKey := keyPrefix + ":" + date

	st.mu.RLock()
	rec, ok := st.cache[cacheKey]
	st.mu.RUnlock()

	if ok {
		return rec
	}

	// Try DB
	rec = &SpendRecord{KeyPrefix: keyPrefix, Date: date}
	row := st.db.QueryRow(`
		SELECT input_tokens, output_tokens, total_tokens, request_count
		FROM spend_records WHERE key_prefix = ? AND date = ?
	`, keyPrefix, date)
	var input, output, total, count int64
	if err := row.Scan(&input, &output, &total, &count); err == nil {
		rec.InputTokens = input
		rec.OutputTokens = output
		rec.TotalTokens = total
		rec.RequestCount = count
	}

	return rec
}

// IsOverBudget checks if a key has exceeded its daily budget
func (st *SpendTracker) IsOverBudget(keyPrefix string) bool {
	if st.dailyBudget <= 0 {
		return false // no budget limit
	}
	usage := st.GetDailyUsage(keyPrefix)
	return usage.TotalTokens >= st.dailyBudget
}

// GetDailyBudget returns the configured daily budget
func (st *SpendTracker) GetDailyBudget() int64 {
	return st.dailyBudget
}

// SetDailyBudget updates the daily budget
func (st *SpendTracker) SetDailyBudget(budget int64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.dailyBudget = budget
}
