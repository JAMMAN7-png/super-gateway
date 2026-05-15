package guardrails

import (
	"regexp"
	"strings"
	"sync"
)

// Pre-compiled regex patterns for PII detection
var (
	reEmail     = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	rePhone     = regexp.MustCompile(`\b(\+?\d{1,3}[-.\s]?)?\(?\d{2,4}\)?[-.\s]?\d{3,4}[-.\s]?\d{3,4}\b`)
	reSSN       = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	reCreditCard = regexp.MustCompile(`\b(?:\d{4}[- ]?){3}\d{4}\b`)
	reAPIKey    = regexp.MustCompile(`\b(sk-[a-zA-Z0-9]{20,}|[A-Za-z0-9]{32,}|eyJ[a-zA-Z0-9_-]{10,}\.(eyJ[a-zA-Z0-9]{10,}|[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}))\b`)
	reIP        = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
)

// PIIMatch describes a detected PII element
type PIIMatch struct {
	Type     string `json:"type"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Redacted string `json:"redacted"`
}

// Result of guardrail check
type Result struct {
	Safe       bool       `json:"safe"`
	Blocked    bool       `json:"blocked"`
	Messages   []string   `json:"messages,omitempty"`
	PIIMatches []PIIMatch `json:"pii_matches,omitempty"`
	Cleaned    string     `json:"cleaned,omitempty"`
}

// Config for guardrail behavior
type Config struct {
	Enabled       bool
	PIIRedact     bool
	ContentFilter bool
	MaxInputLen   int
}

// Guardrails engine
type Guardrails struct {
	mu     sync.RWMutex
	config Config
}

// New creates a guardrails engine
func New(cfg Config) *Guardrails {
	return &Guardrails{
		config: cfg,
	}
}

// UpdateConfig updates guardrail configuration at runtime
func (g *Guardrails) UpdateConfig(cfg Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.config = cfg
}

// CheckInput runs all enabled guardrails on input text
func (g *Guardrails) CheckInput(text string) *Result {
	g.mu.RLock()
	cfg := g.config
	g.mu.RUnlock()

	result := &Result{Safe: true, Cleaned: text}

	if !cfg.Enabled {
		return result
	}

	// Max input length check
	if cfg.MaxInputLen > 0 && len(text) > cfg.MaxInputLen {
		result.Cleaned = text[:cfg.MaxInputLen]
		result.Messages = append(result.Messages, "input truncated to max length")
	}

	// PII redaction
	if cfg.PIIRedact {
		cleaned, matches := redactPII(result.Cleaned)
		if len(matches) > 0 {
			result.PIIMatches = matches
			result.Cleaned = cleaned
			result.Messages = append(result.Messages, "PII redacted")
		}
	}

	// Content filtering (basic harmful content detection)
	if cfg.ContentFilter {
		if containsHarmfulContent(result.Cleaned) {
			result.Safe = false
			result.Blocked = true
			result.Messages = append(result.Messages, "content blocked by filter")
			result.Cleaned = ""
			return result
		}
	}

	return result
}

// CheckOutput runs guardrails on LLM output (typically PII leak prevention)
func (g *Guardrails) CheckOutput(text string) *Result {
	g.mu.RLock()
	cfg := g.config
	g.mu.RUnlock()

	result := &Result{Safe: true, Cleaned: text}

	if !cfg.Enabled || !cfg.PIIRedact {
		return result
	}

	// Redact PII that might leak from model output
	cleaned, matches := redactPII(text)
	if len(matches) > 0 {
		result.PIIMatches = matches
		result.Cleaned = cleaned
		result.Messages = append(result.Messages, "PII redacted from output")
	}

	return result
}

func redactPII(text string) (string, []PIIMatch) {
	type pattern struct {
		re     *regexp.Regexp
		piiType string
	}
	patterns := []pattern{
		{reEmail, "email"},
		{rePhone, "phone"},
		{reSSN, "ssn"},
		{reCreditCard, "credit_card"},
		{reAPIKey, "api_key"},
		{reIP, "ip_address"},
	}

	var matches []PIIMatch
	result := text

	for _, p := range patterns {
		locs := p.re.FindAllStringIndex(result, -1)
		// Process in reverse order to preserve positions
		for i := len(locs) - 1; i >= 0; i-- {
			loc := locs[i]
			original := result[loc[0]:loc[1]]
			redacted := redactByType(original, p.piiType)
			result = result[:loc[0]] + redacted + result[loc[1]:]

			matches = append(matches, PIIMatch{
				Type:     p.piiType,
				Start:    loc[0],
				End:      loc[0] + len(redacted),
				Redacted: redacted,
			})
		}
	}

	return result, matches
}

func redactByType(original, piiType string) string {
	switch piiType {
	case "email":
		parts := strings.SplitN(original, "@", 2)
		if len(parts) == 2 {
			return parts[0][:1] + "***@" + parts[1]
		}
		return "[REDACTED]"
	case "phone":
		if len(original) > 4 {
			return original[:len(original)-4] + "****"
		}
		return "[REDACTED]"
	case "ssn":
		return "***-**-****"
	case "credit_card":
		return "****-****-****-" + original[len(original)-4:]
	case "api_key":
		if len(original) > 8 {
			return original[:4] + "****" + original[len(original)-4:]
		}
		return "[REDACTED]"
	case "ip_address":
		return original[:strings.LastIndex(original, ".")+1] + "***"
	default:
		return "[REDACTED]"
	}
}

// containsHarmfulContent checks for basic harmful patterns
func containsHarmfulContent(text string) bool {
	lower := strings.ToLower(text)

	harmfulPatterns := []string{
		"ignore previous instructions",
		"ignore all instructions",
		"forget your instructions",
		"disregard previous",
		"you are now",
		"pretend you are",
		"act as if",
		"system prompt",
		"your system prompt",
		"jailbreak",
		"dan ",
		"do anything now",
	}

	for _, p := range harmfulPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	return false
}

// TruncateToLimit truncates string to max bytes
func TruncateToLimit(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}
