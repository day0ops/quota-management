package extproc

import (
	"context"
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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/agentgateway/quota-management/internal/config"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/metrics"
)

// RateLimitServer implements the ext_proc service for rate limit metadata injection.
// It runs at PreRouting phase, extracts model from request body, looks up rate limit
// allocations, and injects dynamic metadata for the Envoy rate limiter.
type RateLimitServer struct {
	extprocv3.UnimplementedExternalProcessorServer
	cfg           *config.Config
	rateLimitRepo *db.RateLimitRepository
	requestStates sync.Map
}

// rateLimitRequestState holds per-request state for the rate limit ext-proc stream.
type rateLimitRequestState struct {
	RequestID   string
	TeamID      string
	OrgID       string
	HeadersMap  map[string]string
	RequestBody []byte
}

// rateLimitMetadata stores rate limit values to inject as dynamic metadata.
type rateLimitMetadata struct {
	TokenLimit   *rateLimitValue
	RequestLimit *rateLimitValue
}

// rateLimitValue stores a single rate limit configuration.
type rateLimitValue struct {
	RequestsPerUnit int64
	Unit            string
}

// NewRateLimitServer creates a new rate limit ext_proc server.
func NewRateLimitServer(cfg *config.Config, rateLimitRepo *db.RateLimitRepository) *RateLimitServer {
	return &RateLimitServer{
		cfg:           cfg,
		rateLimitRepo: rateLimitRepo,
	}
}

// Register registers the server with a gRPC server.
func (s *RateLimitServer) Register(grpcServer *grpc.Server) {
	extprocv3.RegisterExternalProcessorServer(grpcServer, s)
}

// Process handles the bidirectional stream from AgentGateway (PreRouting phase).
func (s *RateLimitServer) Process(stream extprocv3.ExternalProcessor_ProcessServer) error {
	log.Debug().Msg("ratelimit-extproc: new connection")
	ctx := stream.Context()

	var requestID string
	var state *rateLimitRequestState

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
		start := time.Now()

		switch r := req.Request.(type) {
		case *extprocv3.ProcessingRequest_RequestHeaders:
			log.Debug().Msg("ratelimit-extproc: received request headers")

			// Extract or generate request ID
			requestID = s.getRequestID(r.RequestHeaders)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Build headers map and extract identity headers
			headersMap := make(map[string]string)
			if r.RequestHeaders.Headers != nil {
				for _, h := range r.RequestHeaders.Headers.Headers {
					headersMap[strings.ToLower(h.Key)] = getHeaderValue(h)
				}
			}

			teamID := headersMap[strings.ToLower(s.cfg.TeamIDHeader)]
			orgID := headersMap[strings.ToLower(s.cfg.OrgIDHeader)]
			modelID := headersMap[strings.ToLower(s.cfg.ModelHeader)]
			authHeader := headersMap["authorization"]

			log.Debug().
				Str("request_id", requestID).
				Str("team_id_header", teamID).
				Str("org_id_header", orgID).
				Bool("has_auth_header", authHeader != "").
				Interface("all_headers", headersMap).
				Msg("ratelimit-extproc: headers received")

			// Fallback to JWT claims if headers are not present
			if teamID == "" || orgID == "" {
				if claims := extractJWTClaims(authHeader); claims != nil {
					log.Debug().
						Str("request_id", requestID).
						Str("org_id_claim", s.cfg.OrgIDClaim).
						Str("team_id_claim", s.cfg.TeamIDClaim).
						Interface("claims", claims).
						Msg("ratelimit-extproc: decoded JWT claims from Authorization header")

					if teamID == "" {
						teamID = getClaimString(claims, s.cfg.TeamIDClaim)
					}
					if orgID == "" {
						orgID = getClaimString(claims, s.cfg.OrgIDClaim)
					}
					if teamID != "" || orgID != "" {
						log.Debug().
							Str("request_id", requestID).
							Str("team_id", teamID).
							Str("org_id", orgID).
							Msg("ratelimit-extproc: extracted identity from JWT claims")
					}
				}
			}

			state = &rateLimitRequestState{
				RequestID:  requestID,
				TeamID:     teamID,
				OrgID:      orgID,
				HeadersMap: headersMap,
			}
			s.requestStates.Store(requestID, state)

			log.Debug().
				Str("request_id", requestID).
				Str("team_id", teamID).
				Str("org_id", orgID).
				Msg("ratelimit-extproc: stored request state")

			// Inject dynamic metadata in the headers response.
			// The gateway ignores dynamic_metadata from the body phase, so metadata
			// must be set here. Requires x-gw-llm-model header to be present.
			// Both keys are always injected so rate limit CEL expressions never fail
			// on missing keys — defaults match the static descriptor fallback (10M/min).
			var allocation *rateLimitMetadata
			if teamID != "" && modelID != "" {
				allocation = s.lookupRateLimitAllocation(ctx, teamID, modelID)
			}
			dynamicMeta := s.buildRateLimitDynamicMetadata(allocation)

			metrics.RecordExtProc("request_headers", "success", time.Since(start))

			resp = &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_RequestHeaders{
					RequestHeaders: &extprocv3.HeadersResponse{
						Response: &extprocv3.CommonResponse{
							Status: extprocv3.CommonResponse_CONTINUE,
						},
					},
				},
				DynamicMetadata: dynamicMeta,
			}

		case *extprocv3.ProcessingRequest_RequestBody:
			log.Debug().
				Bool("end_of_stream", r.RequestBody.EndOfStream).
				Int("body_len", len(r.RequestBody.Body)).
				Msg("ratelimit-extproc: received request body")

			// Buffer body chunks
			if state != nil {
				state.RequestBody = append(state.RequestBody, r.RequestBody.Body...)
			}

			if r.RequestBody.EndOfStream && state != nil {
				// Extract model: body-first, header fallback
				// TODO: Gateway should pass model as dynamic metadata in the future.
				// This body parsing is a workaround until that feature exists in kgateway CRDs.
				modelID := extractModel(state.RequestBody, state.HeadersMap, s.cfg.ModelHeader)

				log.Debug().
					Str("request_id", requestID).
					Str("team_id", state.TeamID).
					Str("model", modelID).
					Msg("ratelimit-extproc: model extracted")

				// Build body response with model header and optional rate limit metadata
				resp = s.buildBodyResponse(ctx, r.RequestBody.Body, r.RequestBody.EndOfStream, state, modelID, start)

				// Clean up state
				s.requestStates.Delete(requestID)
			} else {
				// Non-final chunk: pass through
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
			}

		case *extprocv3.ProcessingRequest_ResponseHeaders:
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
			resp = &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_ResponseBody{
					ResponseBody: &extprocv3.BodyResponse{
						Response: &extprocv3.CommonResponse{
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
			log.Warn().Str("type", fmt.Sprintf("%T", req.Request)).Msg("ratelimit-extproc: unknown request type")
			continue
		}

		if err := stream.Send(resp); err != nil {
			log.Error().Err(err).Msg("ratelimit-extproc: failed to send response")
			return status.Errorf(codes.Internal, "failed to send response: %v", err)
		}
	}
}

// buildBodyResponse builds the end-of-stream body response with model header and rate limit metadata.
func (s *RateLimitServer) buildBodyResponse(ctx context.Context, body []byte, endOfStream bool, state *rateLimitRequestState, modelID string, start time.Time) *extprocv3.ProcessingResponse {
	var headerMutation *extprocv3.HeaderMutation

	// Set x-gw-llm-model header so rate limit descriptors can read it
	if modelID != "" {
		headerMutation = &extprocv3.HeaderMutation{
			SetHeaders: []*corev3.HeaderValueOption{
				{
					Header: &corev3.HeaderValue{
						Key:      "x-gw-llm-model",
						RawValue: []byte(modelID),
					},
				},
			},
		}
	}

	metrics.RecordExtProc("request_body", "success", time.Since(start))

	return &extprocv3.ProcessingResponse{
		Response: &extprocv3.ProcessingResponse_RequestBody{
			RequestBody: &extprocv3.BodyResponse{
				Response: &extprocv3.CommonResponse{
					HeaderMutation: headerMutation,
					BodyMutation: &extprocv3.BodyMutation{
						Mutation: &extprocv3.BodyMutation_StreamedResponse{
							StreamedResponse: &extprocv3.StreamedBodyResponse{
								Body:        body,
								EndOfStream: endOfStream,
							},
						},
					},
				},
			},
		},
	}
}

// lookupRateLimitAllocation looks up rate limit allocations for a team/model and merges them.
// Multiple allocations may match (e.g. gpt-4o-mini exact + gpt-4* pattern); this merges
// them by taking the most specific allocation that provides each limit type.
// Returns nil if no enforced allocations found (fail open).
func (s *RateLimitServer) lookupRateLimitAllocation(ctx context.Context, teamID, modelID string) *rateLimitMetadata {
	if s.rateLimitRepo == nil || teamID == "" {
		return nil
	}

	allocations, err := s.rateLimitRepo.GetAllocationsForTeamModel(ctx, teamID, modelID)
	if err != nil {
		log.Error().Err(err).
			Str("team_id", teamID).
			Str("model", modelID).
			Msg("ratelimit-extproc: failed to lookup allocations, failing open")
		return nil
	}
	if len(allocations) == 0 {
		log.Debug().
			Str("team_id", teamID).
			Str("model", modelID).
			Msg("ratelimit-extproc: no allocation found")
		return nil
	}

	// Merge limits across all matching allocations (ordered by specificity).
	// For each limit type, use the first allocation in priority order that provides it
	// and is not in monitoring mode.
	meta := &rateLimitMetadata{}

	for i := range allocations {
		a := &allocations[i]
		if a.Enforcement == "monitoring" {
			continue
		}
		if meta.TokenLimit == nil && a.HasTokenLimit() {
			meta.TokenLimit = &rateLimitValue{
				RequestsPerUnit: a.TokenLimit.Int64,
				Unit:            a.TokenUnit.String,
			}
		}
		if meta.RequestLimit == nil && a.HasRequestLimit() {
			meta.RequestLimit = &rateLimitValue{
				RequestsPerUnit: a.RequestLimit.Int64,
				Unit:            a.RequestUnit.String,
			}
		}
		if meta.TokenLimit != nil && meta.RequestLimit != nil {
			break
		}
	}

	if meta.TokenLimit == nil && meta.RequestLimit == nil {
		log.Info().
			Str("team_id", teamID).
			Str("model", modelID).
			Msg("ratelimit-extproc: all matching allocations in monitoring mode, not enforcing")
		return nil
	}

	log.Debug().
		Str("team_id", teamID).
		Str("model", modelID).
		Interface("token_limit", meta.TokenLimit).
		Interface("request_limit", meta.RequestLimit).
		Msg("ratelimit-extproc: injecting rate limit metadata")

	return meta
}

// buildRateLimitDynamicMetadata builds the dynamic metadata struct for rate limiting.
// Both token_rate_limit and request_rate_limit are always included so that the rate
// limiter's CEL expressions never fail on a missing key. When a limit is not set,
// an empty struct is used so the rate limiter falls back to the static descriptor limit.
func (s *RateLimitServer) buildRateLimitDynamicMetadata(meta *rateLimitMetadata) *structpb.Struct {
	var tokenStruct, requestStruct *structpb.Struct

	if meta != nil && meta.TokenLimit != nil {
		tokenStruct, _ = structpb.NewStruct(map[string]interface{}{
			"requests_per_unit": meta.TokenLimit.RequestsPerUnit,
			"unit":              meta.TokenLimit.Unit,
		})
	} else {
		tokenStruct = &structpb.Struct{}
	}

	if meta != nil && meta.RequestLimit != nil {
		requestStruct, _ = structpb.NewStruct(map[string]interface{}{
			"requests_per_unit": meta.RequestLimit.RequestsPerUnit,
			"unit":              meta.RequestLimit.Unit,
		})
	} else {
		requestStruct = &structpb.Struct{}
	}

	return &structpb.Struct{Fields: map[string]*structpb.Value{
		"token_rate_limit":   structpb.NewStructValue(tokenStruct),
		"request_rate_limit": structpb.NewStructValue(requestStruct),
	}}
}

// getRequestID extracts request ID from headers.
func (s *RateLimitServer) getRequestID(headers *extprocv3.HttpHeaders) string {
	if headers == nil || headers.Headers == nil {
		return ""
	}
	for _, h := range headers.Headers.Headers {
		key := strings.ToLower(h.Key)
		if key == "x-request-id" {
			return getHeaderValue(h)
		}
	}
	return ""
}
