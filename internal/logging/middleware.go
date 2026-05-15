package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/goozway/super-gateway/internal/providers"
)

// Middleware creates a Fiber middleware that logs requests
func Middleware(store *Store) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// Read the request body so we can extract model name
		body := c.Body()
		var req providers.ChatRequest
		model := "unknown"
		if err := json.Unmarshal(body, &req); err == nil {
			model = req.Model
		}

		// Continue to handler
		err := c.Next()

		latency := time.Since(start).Milliseconds()

		// Try to extract response details
		var respBody []byte
		if c.Response().Body() != nil {
			respBody = c.Response().Body()
		}

		var chatResp providers.ChatResponse
		success := c.Response().StatusCode() == 200
		provider := ""
		promptTokens := 0
		compTokens := 0
		totalTokens := 0

		if success && respBody != nil {
			if err := json.Unmarshal(respBody, &chatResp); err == nil {
				provider = chatResp.Model
				promptTokens = chatResp.Usage.PromptTokens
				compTokens = chatResp.Usage.CompletionTokens
				totalTokens = chatResp.Usage.TotalTokens
			}
		}

		errMsg := ""
		if !success {
			errMsg = string(respBody)
			if len(errMsg) > 200 {
				errMsg = errMsg[:200]
			}
		}

		store.Log(&RequestLog{
			Model:        model,
			Provider:     provider,
			LatencyMs:    latency,
			PromptTokens: promptTokens,
			CompTokens:   compTokens,
			TotalTokens:  totalTokens,
			Success:      success,
			Error:        errMsg,
		})

		// Restore request body for downstream handlers
		c.Request().SetBody(body)

		return err
	}
}

// These prevent unused import warnings
var _ = io.ReadAll
var _ = bytes.NewReader
