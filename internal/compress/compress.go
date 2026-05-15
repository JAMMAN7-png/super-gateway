package compress

import (
	"strings"
	"unicode"
)

// Level defines compression intensity
type Level string

const (
	LevelLight      Level = "light"
	LevelMedium     Level = "medium"
	LevelAggressive Level = "aggressive"
)

// Config for compression
type Config struct {
	InputCompression  bool
	OutputCompression bool
	Level             Level
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		InputCompression:  false,
		OutputCompression: false,
		Level:             LevelLight,
	}
}

// ============================================================================
// INPUT COMPRESSION (RTK-style from 9Router)
// Compresses prompts before sending to LLM to save tokens
// ============================================================================

// CompressInput reduces prompt token count by removing filler words and
// compressing verbose patterns while preserving meaning
func CompressInput(text string, level Level) string {
	if text == "" {
		return text
	}

	switch level {
	case LevelLight:
		return compressLight(text)
	case LevelMedium:
		return compressMedium(text)
	case LevelAggressive:
		return compressAggressive(text)
	default:
		return text
	}
}

func compressLight(text string) string {
	// Remove common filler words that add no semantic value
	fillers := []string{
		" essentially ", " basically ", " literally ",
		" actually ", " honestly ", " frankly ",
		" obviously ", " clearly ", " certainly ",
		" definitely ", " absolutely ", " totally ",
	}
	result := text
	for _, f := range fillers {
		result = strings.ReplaceAll(result, f, " ")
	}
	return result
}

func compressMedium(text string) string {
	// Light compression plus more aggressive patterns
	result := compressLight(text)

	// Contract common verbose phrases
	replacements := map[string]string{
		" in order to ":       " to ",
		" in the context of ": " re ",
		" with regard to ":    " re ",
		" with respect to ":   " re ",
		" on the other hand ": " however ",
		" as a result ":       " thus ",
		" in addition ":       " also ",
		" in conclusion ":     " finally ",
		" at this point ":     " now ",
		" due to the fact ":   " because ",
		" in spite of ":       " despite ",
		" regardless of ":     " despite ",
		" a large number of ": " many ",
		" a small number of ": " few ",
		" is able to ":        " can ",
		" has the ability ":   " can ",
		" in the event that ": " if ",
	}

	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

func compressAggressive(text string) string {
	// Medium compression plus structural shortening
	result := compressMedium(text)

	// Remove polite phrases
	politeness := []string{
		" please ", " thank you ", " thanks ",
		" I would like ", " I want ", " could you ",
		" would you mind ", " if possible ", " kindly ",
	}
	for _, p := range politeness {
		result = strings.ReplaceAll(result, p, " ")
	}

	// Compress common English articles and prepositions in non-critical positions
	// (only when surrounded by spaces to avoid breaking code)
	removals := []string{
		" very ", " quite ", " rather ", " somewhat ",
		" just ", " simply ", " merely ",
		" indeed ", " surely ", " surely ",
	}
	for _, r := range removals {
		result = strings.ReplaceAll(result, r, " ")
	}

	// Collapse multiple spaces
	result = collapseSpaces(result)

	return result
}

// CompressMessages applies input compression to all user messages
func CompressMessages(messages []Message, level Level) []Message {
	if len(messages) == 0 {
		return messages
	}

	result := make([]Message, len(messages))
	for i, msg := range messages {
		result[i] = msg
		if msg.Role == "user" {
			result[i].Content = CompressInput(msg.Content, level)
		}
	}
	return result
}

// Message represents a chat message for compression
type Message struct {
	Role    string
	Content string
}

// ============================================================================
// OUTPUT COMPRESSION (Caveman mode from 9Router)
// Compresses LLM responses into concise caveman-style output
// ============================================================================

// CompressOutput compresses LLM response into caveman-speak style
// Saves ~40-65% output tokens per 9Router benchmarks
func CompressOutput(text string, level Level) string {
	if text == "" {
		return text
	}

	switch level {
	case LevelLight:
		return outputLight(text)
	case LevelMedium:
		return outputMedium(text)
	case LevelAggressive:
		return outputAggressive(text)
	default:
		return text
	}
}

func outputLight(text string) string {
	// Minor compression: remove redundant pleasantries
	replacements := map[string]string{
		"I'd be happy to help":                "",
		"Sure! Here":                          "Here",
		"Certainly!":                          "",
		"Of course!":                          "",
		"Absolutely!":                         "",
		"No problem!":                         "",
		"You're welcome!":                     "",
		"I hope this helps!":                  "",
		"Let me know if":                      "If",
		"feel free to ask":                    "ask",
		"don't hesitate":                      "",
		"Please let me know":                  "",
		"As an AI":                            "I",
		"As an AI language model":             "I",
		"I'm here to help":                    "",
	}

	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return strings.TrimSpace(result)
}

func outputMedium(text string) string {
	result := outputLight(text)

	// More aggressive: remove self-references and hedging
	hedging := []string{
		" I think ", " I believe ", " I feel ",
		" it seems ", " it appears ", " it looks like ",
		" in my opinion ", " from my perspective ",
		" I would say ", " I'd say ", " I'd suggest ",
	}
	for _, h := range hedging {
		result = strings.ReplaceAll(result, h, " ")
	}

	return collapseSpaces(result)
}

func outputAggressive(text string) string {
	result := outputMedium(text)

	// Caveman mode: shorten sentences, remove articles where possible
	// Convert to imperative/telegraphic style
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Remove leading "The " / "A " / "An " in sentences
		if len(trimmed) > 4 {
			lower := strings.ToLower(trimmed[:4])
			if lower == "the " || lower == "a " || lower == "an " {
				trimmed = trimmed[4:]
			} else if len(trimmed) > 3 && strings.ToLower(trimmed[:3]) == "an " {
				trimmed = trimmed[3:]
			}
		}

		lines[i] = trimmed
	}
	result = strings.Join(lines, "\n")
	result = collapseSpaces(result)

	return result
}

// collapseSpaces reduces multiple whitespace chars to single space
func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	wasSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !wasSpace {
				b.WriteByte(' ')
				wasSpace = true
			}
		} else {
			b.WriteRune(r)
			wasSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
