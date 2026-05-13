package extproc

import (
	"encoding/json"
	"strings"
)

// extractModel extracts the LLM model name from the request.
// Strategy: body-first (graceful non-JSON), then header fallback.
//
// TODO: Gateway should pass model as dynamic metadata in the future.
// This body parsing is a workaround until that feature exists in kgateway CRDs.
// Once supported, remove body buffering and read from req.MetadataContext instead.
func extractModel(body []byte, headers map[string]string, modelHeader string) string {
	// Try body first - gracefully handle non-JSON payloads
	if len(body) > 0 {
		var req struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(body, &req); err == nil && req.Model != "" {
			return req.Model
		}
	}

	// Fallback to configurable header
	if modelHeader != "" {
		if model, ok := headers[strings.ToLower(modelHeader)]; ok && model != "" {
			return model
		}
	}

	return ""
}
