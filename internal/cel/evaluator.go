package cel

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
)

// EvalContext provides context variables for CEL expression evaluation.
type EvalContext struct {
	// Request context
	Request RequestContext `json:"request"`

	// JWT claims (if present)
	JWT JWTContext `json:"jwt"`

	// API Key metadata (if present)
	APIKey APIKeyContext `json:"apiKey"`

	// LLM request context
	LLM LLMContext `json:"llm"`

	// Source address
	Source SourceContext `json:"source"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata"`
}

// RequestContext contains request information.
type RequestContext struct {
	Headers map[string]string `json:"headers"`
	Path    string            `json:"path"`
	Method  string            `json:"method"`
	Host    string            `json:"host"`
}

// JWTContext contains JWT claims.
type JWTContext struct {
	Claims map[string]interface{} `json:"claims"`
	Issuer string                 `json:"issuer"`
}

// APIKeyContext contains API key metadata.
type APIKeyContext struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

// LLMContext contains LLM request context.
type LLMContext struct {
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	Streaming bool   `json:"streaming"`
}

// SourceContext contains source address information.
type SourceContext struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

// Evaluator provides CEL expression evaluation.
type Evaluator struct {
	env   *cel.Env
	cache sync.Map // map[string]cel.Program
}

// NewEvaluator creates a new CEL evaluator with the standard environment.
func NewEvaluator() (*Evaluator, error) {
	env, err := cel.NewEnv(
		// Request context with nested headers map
		cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),

		// JWT context
		cel.Variable("jwt", cel.MapType(cel.StringType, cel.DynType)),

		// API Key context
		cel.Variable("apiKey", cel.MapType(cel.StringType, cel.DynType)),

		// LLM context
		cel.Variable("llm", cel.MapType(cel.StringType, cel.DynType)),

		// Source context
		cel.Variable("source", cel.MapType(cel.StringType, cel.DynType)),

		// Metadata
		cel.Variable("metadata", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Evaluator{env: env}, nil
}

// Compile compiles a CEL expression and caches the program.
func (e *Evaluator) Compile(expression string) (cel.Program, error) {
	// Check cache first
	if cached, ok := e.cache.Load(expression); ok {
		return cached.(cel.Program), nil
	}

	// Parse and check the expression
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("invalid CEL expression")
	}

	// Create the program
	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	// Cache the program
	e.cache.Store(expression, prg)

	return prg, nil
}

// Evaluate evaluates a CEL expression against the given context.
func (e *Evaluator) Evaluate(expression string, ctx *EvalContext) (bool, error) {
	prg, err := e.Compile(expression)
	if err != nil {
		return false, err
	}

	// Convert headers to map[string]interface{} for CEL compatibility
	headers := make(map[string]interface{}, len(ctx.Request.Headers))
	for k, v := range ctx.Request.Headers {
		headers[k] = v
	}

	// Build activation
	activation := map[string]interface{}{
		"request": map[string]interface{}{
			"headers": headers,
			"path":    ctx.Request.Path,
			"method":  ctx.Request.Method,
			"host":    ctx.Request.Host,
		},
		"jwt": map[string]interface{}{
			"claims": ctx.JWT.Claims,
			"issuer": ctx.JWT.Issuer,
		},
		"apiKey": map[string]interface{}{
			"id":       ctx.APIKey.ID,
			"name":     ctx.APIKey.Name,
			"metadata": ctx.APIKey.Metadata,
		},
		"llm": map[string]interface{}{
			"model":     ctx.LLM.Model,
			"provider":  ctx.LLM.Provider,
			"streaming": ctx.LLM.Streaming,
		},
		"source": map[string]interface{}{
			"address": ctx.Source.Address,
			"port":    ctx.Source.Port,
		},
		"metadata": ctx.Metadata,
	}

	// Evaluate
	out, _, err := prg.Eval(activation)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	// Convert result to bool
	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expression did not return a boolean")
	}

	return result, nil
}

// ValidateExpression validates a CEL expression without evaluating it.
func (e *Evaluator) ValidateExpression(expression string) error {
	_, err := e.Compile(expression)
	return err
}
