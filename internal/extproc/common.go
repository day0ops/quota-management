package extproc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
)

// getHeaderValue extracts the value from a header, preferring RawValue over Value.
func getHeaderValue(h *corev3.HeaderValue) string {
	if len(h.RawValue) > 0 {
		return string(h.RawValue)
	}
	return h.Value
}

// extractJWTClaims extracts claims from a JWT token without validating the signature.
// The JWT is expected to already be validated by the gateway's JWT filter.
// Returns nil if the token is invalid or missing.
func extractJWTClaims(authHeader string) map[string]interface{} {
	if authHeader == "" {
		return nil
	}

	// Extract Bearer token
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return nil
	}
	token := strings.TrimSpace(authHeader[7:])

	// Split JWT into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}

	// Decode payload (second part)
	payload := parts[1]
	// Add padding if needed for base64 decoding
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try standard base64 with URL-safe replacements
		payload = strings.ReplaceAll(parts[1], "-", "+")
		payload = strings.ReplaceAll(payload, "_", "/")
		switch len(payload) % 4 {
		case 2:
			payload += "=="
		case 3:
			payload += "="
		}
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil
		}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil
	}

	return claims
}

// getClaimString extracts a string claim from JWT claims map.
func getClaimString(claims map[string]interface{}, key string) string {
	if claims == nil || key == "" {
		return ""
	}
	if val, ok := claims[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case float64:
			return fmt.Sprintf("%.0f", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// immediateResponse creates an immediate response to reject a request.
// requestPath is used to detect the API format and shape the error body accordingly.
func immediateResponse(statusCode int, message string, retryAfter int, requestPath string) *extprocv3.ProcessingResponse {
	var headers []*corev3.HeaderValueOption

	headers = append(headers, &corev3.HeaderValueOption{
		Header: &corev3.HeaderValue{
			Key:      "content-type",
			RawValue: []byte("application/json"),
		},
	})

	if statusCode == 429 && retryAfter > 0 {
		headers = append(headers, &corev3.HeaderValueOption{
			Header: &corev3.HeaderValue{
				Key:      "retry-after",
				RawValue: []byte(strconv.Itoa(retryAfter)),
			},
		})
	}

	msgJSON, _ := json.Marshal(message)

	var body string
	if strings.Contains(requestPath, "/v1/messages") {
		// Anthropic format
		body = fmt.Sprintf(`{"type":"error","error":{"type":"insufficient_quota","message":%s}}`, msgJSON)
	} else {
		// OpenAI-compatible format (default for /v1/chat/completions and others)
		body = fmt.Sprintf(`{"error":{"message":%s,"type":"insufficient_quota","code":"budget_exceeded"}}`, msgJSON)
	}

	return &extprocv3.ProcessingResponse{
		Response: &extprocv3.ProcessingResponse_ImmediateResponse{
			ImmediateResponse: &extprocv3.ImmediateResponse{
				Status: &typev3.HttpStatus{
					Code: typev3.StatusCode(statusCode),
				},
				Headers: &extprocv3.HeaderMutation{
					SetHeaders: headers,
				},
				Body:    []byte(body),
				Details: "budget-rate-limit: " + message,
			},
		},
	}
}
