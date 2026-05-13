package extproc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/agentgateway/quota-management/internal/budget"
	"github.com/agentgateway/quota-management/internal/cel"
	"github.com/agentgateway/quota-management/internal/config"
	"github.com/agentgateway/quota-management/internal/metrics"
)

// BudgetServer implements the ext_proc service for budget enforcement.
// Runs at PostRouting phase (checks budget before upstream) and Response phase (actual cost accounting).
type BudgetServer struct {
	extprocv3.UnimplementedExternalProcessorServer
	budgetSvc    *budget.Service
	celEvaluator *cel.Evaluator
	cfg          *config.Config

	requestStates sync.Map // map[string]*budgetRequestState
}

// budgetInfo stores budget details for metrics recording.
type budgetInfo struct {
	EntityType string
	Name       string
	Period     string
}

// budgetRequestState holds state for a single request across the stream.
type budgetRequestState struct {
	RequestID       string
	ModelID         string
	TeamID          string
	HeadersMap      map[string]string // stored for model extraction at body phase
	MatchedBudgets  []budgetInfo
	RateLimitedAt   *uuid.UUID
	EstimatedCost   float64
	StartTime       time.Time
	StreamingBuffer strings.Builder
	EvalContext     *cel.EvalContext
	BudgetChecked   bool
	RequestBody     []byte
}

// NewBudgetServer creates a new budget ext_proc server.
func NewBudgetServer(budgetSvc *budget.Service, celEvaluator *cel.Evaluator, cfg *config.Config) *BudgetServer {
	return &BudgetServer{
		budgetSvc:    budgetSvc,
		celEvaluator: celEvaluator,
		cfg:          cfg,
	}
}

// Register registers the server with a gRPC server.
func (s *BudgetServer) Register(grpcServer *grpc.Server) {
	extprocv3.RegisterExternalProcessorServer(grpcServer, s)
}

// Process handles the bidirectional stream from AgentGateway.
func (s *BudgetServer) Process(stream extprocv3.ExternalProcessor_ProcessServer) error {
	log.Debug().Msg("budget-extproc: new connection from agentgateway")
	ctx := stream.Context()

	var requestID string
	var state *budgetRequestState

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive request: %v", err)
		}

		var resp *extprocv3.ProcessingResponse

		switch r := req.Request.(type) {
		case *extprocv3.ProcessingRequest_RequestHeaders:
			log.Debug().Msg("budget-extproc: received request headers")

			requestID = s.getRequestID(r.RequestHeaders)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Build headers map for use in body phase model extraction
			headersMap := make(map[string]string)
			if r.RequestHeaders.Headers != nil {
				for _, h := range r.RequestHeaders.Headers.Headers {
					headersMap[strings.ToLower(h.Key)] = getHeaderValue(h)
				}
			}

			evalCtx := s.buildEvalContext(r.RequestHeaders, req)
			teamID := s.extractTeamIDFromHeaders(r.RequestHeaders)

			state = &budgetRequestState{
				RequestID:     requestID,
				TeamID:        teamID,
				HeadersMap:    headersMap,
				StartTime:     time.Now(),
				EvalContext:   evalCtx,
				BudgetChecked: false,
			}
			s.requestStates.Store(requestID, state)

			// Budget check always happens at headers phase.
			// ImmediateResponse from body phase is unreliable in agentgateway streaming mode —
			// body chunks are already in-flight to upstream before extproc can reject them.
			// Model from x-gw-llm-model header if available; empty model falls back to default pricing.
			if s.cfg.ModelHeader != "" {
				if modelFromHeader, ok := headersMap[strings.ToLower(s.cfg.ModelHeader)]; ok && modelFromHeader != "" {
					state.ModelID = modelFromHeader
				}
			}
			log.Debug().Str("request_id", requestID).Str("model", state.ModelID).Msg("budget-extproc: checking budget at headers phase")
			if budgetResp := s.doBudgetCheckAndRespond(ctx, state, false); budgetResp != nil {
				resp = budgetResp
				break
			}
			resp = s.buildBudgetHeadersResponse(state)

		case *extprocv3.ProcessingRequest_RequestBody:
			log.Debug().Bool("end_of_stream", r.RequestBody.EndOfStream).Int("body_len", len(r.RequestBody.Body)).Msg("budget-extproc: received request body")

			if state != nil {
				state.RequestBody = append(state.RequestBody, r.RequestBody.Body...)
				// Opportunistically capture model from body for response-phase cost accounting.
				if state.ModelID == "" && r.RequestBody.EndOfStream {
					state.ModelID = extractModel(state.RequestBody, state.HeadersMap, s.cfg.ModelHeader)
					if state.ModelID != "" {
						log.Debug().Str("request_id", requestID).Str("model", state.ModelID).Msg("budget-extproc: captured model from body for cost accounting")
					}
				}
			}

			resp = &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_RequestBody{
					RequestBody: &extprocv3.BodyResponse{
						Response: &extprocv3.CommonResponse{
							BodyMutation: &extprocv3.BodyMutation{
								Mutation: &extprocv3.BodyMutation_StreamedResponse{
									StreamedResponse: &extprocv3.StreamedBodyResponse{
										Body:        r.RequestBody.Body,
										EndOfStream: r.RequestBody.EndOfStream,
									},
								},
							},
						},
					},
				},
			}

		case *extprocv3.ProcessingRequest_ResponseHeaders:
			log.Debug().Msg("budget-extproc: received response headers")
			resp = &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: &extprocv3.HeadersResponse{
						Response: &extprocv3.CommonResponse{
							Status: extprocv3.CommonResponse_CONTINUE,
						},
					},
				},
			}

		case *extprocv3.ProcessingRequest_ResponseBody:
			log.Debug().Bool("end_of_stream", r.ResponseBody.EndOfStream).Int("body_len", len(r.ResponseBody.Body)).Msg("budget-extproc: received response body")

			if state != nil {
				state.StreamingBuffer.Write(r.ResponseBody.Body)
			}

			var headerMutation *extprocv3.HeaderMutation
			if r.ResponseBody.EndOfStream && state != nil {
				responseBody := []byte(state.StreamingBuffer.String())

				inputTokens, outputTokens := s.parseTokenUsage(responseBody)

				if state.ModelID == "" {
					state.ModelID = s.extractModelFromResponse(responseBody)
				}

				result, err := s.budgetSvc.DecrementBudgets(ctx, requestID, state.ModelID, inputTokens, outputTokens, state.RateLimitedAt)
				if err != nil {
					log.Error().Err(err).Str("request_id", requestID).Msg("budget-extproc: failed to decrement budgets")
				}

				if result != nil && result.ActualCost > 0 {
					for _, charge := range result.Charges {
						metrics.RecordCostCharged(charge.EntityType, charge.BudgetName, state.ModelID, charge.ChargeAmount)
					}
					for _, b := range state.MatchedBudgets {
						metrics.RecordTokens(b.EntityType, b.Name, state.ModelID, inputTokens, outputTokens)
					}
					metrics.RecordExtProc("response_body", "success", time.Since(state.StartTime))
				}

				if result != nil {
					headerMutation = &extprocv3.HeaderMutation{
						SetHeaders: []*corev3.HeaderValueOption{
							{
								Header: &corev3.HeaderValue{
									Key:      "x-budget-cost-usd",
									RawValue: []byte(fmt.Sprintf("%.6f", result.ActualCost)),
								},
							},
							{
								Header: &corev3.HeaderValue{
									Key:      "x-budget-remaining-usd",
									RawValue: []byte(fmt.Sprintf("%.6f", result.RemainingBudget)),
								},
							},
						},
					}
				}

				s.requestStates.Delete(requestID)
			}

			resp = &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_ResponseBody{
					ResponseBody: &extprocv3.BodyResponse{
						Response: &extprocv3.CommonResponse{
							HeaderMutation: headerMutation,
							BodyMutation: &extprocv3.BodyMutation{
								Mutation: &extprocv3.BodyMutation_StreamedResponse{
									StreamedResponse: &extprocv3.StreamedBodyResponse{
										Body:        r.ResponseBody.Body,
										EndOfStream: r.ResponseBody.EndOfStream,
									},
								},
							},
						},
					},
				},
			}

		default:
			log.Warn().Str("type", fmt.Sprintf("%T", req.Request)).Msg("budget-extproc: unknown request type")
			continue
		}

		log.Debug().Msg("budget-extproc: sending response")
		if err := stream.Send(resp); err != nil {
			log.Error().Err(err).Msg("budget-extproc: failed to send response")
			return status.Errorf(codes.Internal, "failed to send response: %v", err)
		}
	}
}

// doBudgetCheckAndRespond performs an atomic budget check and reservation.
// If isHeadersPhase is true, returns a headers response on success; otherwise returns nil on success.
// Returns an immediate reject response if budget is exceeded.
func (s *BudgetServer) doBudgetCheckAndRespond(ctx context.Context, state *budgetRequestState, isHeadersPhase bool) *extprocv3.ProcessingResponse {
	if state == nil || state.EvalContext == nil {
		return nil
	}

	checkStart := time.Now()
	result, err := s.budgetSvc.CheckAndReserveBudget(ctx, state.EvalContext, state.ModelID, state.RequestID)
	checkDuration := time.Since(checkStart)

	requestPath := ""
	if state.EvalContext != nil {
		requestPath = state.EvalContext.Request.Path
	}

	if err != nil {
		log.Error().Err(err).Str("request_id", state.RequestID).Msg("budget-extproc: failed to check budget")
		return immediateResponse(429, "Budget check failed", 0, requestPath)
	}

	state.BudgetChecked = true

	if len(result.MatchedBudgets) > 0 {
		for _, b := range result.MatchedBudgets {
			log.Debug().
				Str("request_id", state.RequestID).
				Str("model", state.ModelID).
				Str("budget_name", b.Name).
				Str("entity_type", string(b.EntityType)).
				Float64("remaining", b.CalculateRemaining()).
				Float64("estimated_cost", result.EstimatedCost).
				Bool("allowed", result.Allowed).
				Msg("budget-extproc: budget check detail")
		}
	}

	if !result.Allowed {
		metrics.RecordBudgetRequest(false)
		for _, b := range result.MatchedBudgets {
			metrics.RecordBudgetCheck(string(b.EntityType), b.Name, false, checkDuration)
			metrics.RecordRateLimited(string(b.EntityType), b.Name)
			metrics.UpdateBudgetUsage(string(b.EntityType), b.Name, string(b.Period),
				b.CurrentUsageUSD, b.CalculateRemaining(), b.BudgetAmountUSD)
		}
		retryAfter := int(result.RetryAfter.Seconds())
		if retryAfter < 1 {
			retryAfter = 3600
		}
		log.Info().
			Str("request_id", state.RequestID).
			Str("model", state.ModelID).
			Float64("remaining", result.RemainingBudget).
			Int("retry_after_seconds", retryAfter).
			Msg("budget-extproc: request rejected - budget exceeded")

		resetMins := retryAfter / 60
		var budgetName string
		if len(result.MatchedBudgets) > 0 {
			budgetName = result.MatchedBudgets[0].Name
		}
		message := fmt.Sprintf(
			"Budget exceeded for '%s'. Remaining: $%.5f. Resets in ~%d min. Stop current task and notify the user.",
			budgetName, result.RemainingBudget, resetMins,
		)
		return immediateResponse(400, message, retryAfter, requestPath)
	}

	metrics.RecordBudgetRequest(true)
	for _, b := range result.MatchedBudgets {
		metrics.RecordBudgetCheck(string(b.EntityType), b.Name, true, checkDuration)
		metrics.UpdateBudgetUsage(string(b.EntityType), b.Name, string(b.Period),
			b.CurrentUsageUSD, b.CalculateRemaining(), b.BudgetAmountUSD)
	}

	state.MatchedBudgets = make([]budgetInfo, 0, len(result.MatchedBudgets))
	for _, b := range result.MatchedBudgets {
		state.MatchedBudgets = append(state.MatchedBudgets, budgetInfo{
			EntityType: string(b.EntityType),
			Name:       b.Name,
			Period:     string(b.Period),
		})
	}
	state.RateLimitedAt = result.RateLimitedAt
	state.EstimatedCost = result.EstimatedCost

	if isHeadersPhase {
		return s.buildBudgetHeadersResponse(state)
	}
	return nil
}

// buildBudgetHeadersResponse builds the headers response with budget tracking header.
func (s *BudgetServer) buildBudgetHeadersResponse(state *budgetRequestState) *extprocv3.ProcessingResponse {
	return &extprocv3.ProcessingResponse{
		Response: &extprocv3.ProcessingResponse_RequestHeaders{
			RequestHeaders: &extprocv3.HeadersResponse{
				Response: &extprocv3.CommonResponse{
					Status: extprocv3.CommonResponse_CONTINUE,
					HeaderMutation: &extprocv3.HeaderMutation{
						SetHeaders: []*corev3.HeaderValueOption{
							{
								Header: &corev3.HeaderValue{
									Key:      "x-budget-request-id",
									RawValue: []byte(state.RequestID),
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildEvalContext builds a CEL evaluation context from request information.
func (s *BudgetServer) buildEvalContext(headers *extprocv3.HttpHeaders, req *extprocv3.ProcessingRequest) *cel.EvalContext {
	ctx := &cel.EvalContext{
		Request: cel.RequestContext{
			Headers: make(map[string]string),
		},
		JWT: cel.JWTContext{
			Claims: make(map[string]interface{}),
		},
		APIKey: cel.APIKeyContext{
			Metadata: make(map[string]interface{}),
		},
		LLM:      cel.LLMContext{},
		Source:   cel.SourceContext{},
		Metadata: make(map[string]interface{}),
	}

	if headers != nil && headers.Headers != nil {
		for _, h := range headers.Headers.Headers {
			key := strings.ToLower(h.Key)
			value := getHeaderValue(h)
			ctx.Request.Headers[key] = value

			switch key {
			case ":path":
				ctx.Request.Path = value
			case ":method":
				ctx.Request.Method = value
			case ":authority", "host":
				ctx.Request.Host = value
			case s.cfg.OrgIDHeader:
				ctx.Metadata["org_id"] = value
			case s.cfg.TeamIDHeader:
				ctx.Metadata["team_id"] = value
			case s.cfg.UserIDHeader:
				ctx.Metadata["user_id"] = value
			}
		}
	}

	if req.MetadataContext != nil && req.MetadataContext.FilterMetadata != nil {
		for ns, md := range req.MetadataContext.FilterMetadata {
			if md.Fields != nil {
				for k, v := range md.Fields {
					ctx.Metadata[ns+"."+k] = v.AsInterface()
				}
			}
		}
	}

	return ctx
}

// getRequestID extracts request ID from headers.
func (s *BudgetServer) getRequestID(headers *extprocv3.HttpHeaders) string {
	if headers == nil || headers.Headers == nil {
		return ""
	}
	for _, h := range headers.Headers.Headers {
		key := strings.ToLower(h.Key)
		if key == "x-request-id" || key == "x-budget-request-id" {
			return getHeaderValue(h)
		}
	}
	return ""
}

// extractTeamIDFromHeaders extracts team ID from request headers.
func (s *BudgetServer) extractTeamIDFromHeaders(headers *extprocv3.HttpHeaders) string {
	if headers == nil || headers.Headers == nil {
		return ""
	}
	teamHeader := s.cfg.TeamIDHeader
	if teamHeader == "" {
		teamHeader = "x-gw-team-id"
	}
	for _, h := range headers.Headers.Headers {
		if strings.EqualFold(h.Key, teamHeader) {
			return getHeaderValue(h)
		}
	}
	return ""
}

// extractModelFromResponse extracts the model from the LLM response body.
func (s *BudgetServer) extractModelFromResponse(body []byte) string {
	var resp struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &resp); err == nil && resp.Model != "" {
		return resp.Model
	}
	return ""
}

// parseTokenUsage parses token usage from the LLM response.
// Handles both non-streaming JSON responses and SSE streaming responses (data: {...} lines).
func (s *BudgetServer) parseTokenUsage(body []byte) (inputTokens, outputTokens int64) {
	bodyStr := string(body)

	// Check if this is an SSE stream (contains "data: " prefix)
	if strings.Contains(bodyStr, "data: ") {
		return s.parseSSETokenUsage(bodyStr)
	}

	// Non-streaming: try OpenAI format first
	var openaiResp struct {
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &openaiResp); err == nil {
		if openaiResp.Usage.PromptTokens > 0 || openaiResp.Usage.CompletionTokens > 0 {
			return openaiResp.Usage.PromptTokens, openaiResp.Usage.CompletionTokens
		}
	}

	// Non-streaming: try Anthropic format
	var anthropicResp struct {
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &anthropicResp); err == nil {
		if anthropicResp.Usage.InputTokens > 0 || anthropicResp.Usage.OutputTokens > 0 {
			return anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens
		}
	}

	return 0, 0
}

// parseSSETokenUsage parses token usage from SSE streaming responses.
// Anthropic: message_start has input_tokens, message_delta has output_tokens.
// OpenAI: last chunk before [DONE] has usage.prompt_tokens / usage.completion_tokens.
func (s *BudgetServer) parseSSETokenUsage(body string) (inputTokens, outputTokens int64) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		// Anthropic message_start: {"type":"message_start","message":{...,"usage":{"input_tokens":N,"output_tokens":M}}}
		var anthropicStart struct {
			Type    string `json:"type"`
			Message struct {
				Usage struct {
					InputTokens  int64 `json:"input_tokens"`
					OutputTokens int64 `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &anthropicStart); err == nil && anthropicStart.Type == "message_start" {
			if anthropicStart.Message.Usage.InputTokens > 0 {
				inputTokens = anthropicStart.Message.Usage.InputTokens
			}
			if anthropicStart.Message.Usage.OutputTokens > 0 {
				outputTokens += anthropicStart.Message.Usage.OutputTokens
			}
			continue
		}

		// Anthropic message_delta: {"type":"message_delta","delta":{...},"usage":{"output_tokens":N}}
		var anthropicDelta struct {
			Type  string `json:"type"`
			Usage struct {
				OutputTokens int64 `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &anthropicDelta); err == nil && anthropicDelta.Type == "message_delta" {
			if anthropicDelta.Usage.OutputTokens > 0 {
				outputTokens += anthropicDelta.Usage.OutputTokens
			}
			continue
		}

		// OpenAI chunk: {"object":"chat.completion.chunk","usage":{"prompt_tokens":N,"completion_tokens":M}}
		var openaiChunk struct {
			Object string `json:"object"`
			Usage  *struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &openaiChunk); err == nil && openaiChunk.Usage != nil {
			if openaiChunk.Usage.PromptTokens > 0 {
				inputTokens = openaiChunk.Usage.PromptTokens
			}
			if openaiChunk.Usage.CompletionTokens > 0 {
				outputTokens = openaiChunk.Usage.CompletionTokens
			}
		}
	}

	log.Debug().
		Int64("actual_input_tokens", inputTokens).
		Int64("actual_output_tokens", outputTokens).
		Msg("budget-extproc: parsed SSE token usage")

	return inputTokens, outputTokens
}
