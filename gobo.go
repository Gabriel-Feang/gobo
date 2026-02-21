package gobo

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
)

// Generator defines the interface for an AI model that can generate fake data from a schema.
type Generator interface {
	GenerateResponse(ctx context.Context, reqCtx RequestContext, schema any) ([]byte, error)
}

// Config represents the configuration for the Gobo middleware.
type Config struct {
	// OllamaURL is the base address of the Ollama server (e.g., "http://localhost:11434").
	// Used only if Generator is nil.
	OllamaURL string
	// Model is the name of the LLM model to use (e.g., "llama3" or "mistral").
	// Used only if Generator is nil.
	Model string
	// Generator allows providing a custom LLM integration (e.g. OpenAI, Anthropic, MCP).
	// If nil, Gobo defaults to the built-in OllamaGenerator using OllamaURL and Model.
	Generator Generator
	// Debug enables verbose logging if set to true.
	Debug bool
}

// Gobo is the core struct that holds the configuration and registered schemas.
type Gobo struct {
	config Config
	routes []*routeSchema
	client Generator
}

// routeSchema stores an expected schema for a specific HTTP method and path pattern.
type routeSchema struct {
	Method         string
	PathPrefix     string // simple prefix matching for now
	ResponseSchema any    // raw Go struct to be marshaled into a JSON schema
}

// New creates a new Gobo instance.
func New(cfg Config) *Gobo {
	var gen Generator
	if cfg.Generator != nil {
		gen = cfg.Generator
	} else {
		if cfg.OllamaURL == "" {
			cfg.OllamaURL = "http://localhost:11434"
		}
		gen = NewOllamaGenerator(cfg.OllamaURL, cfg.Model)
	}

	return &Gobo{
		config: cfg,
		routes: make([]*routeSchema, 0),
		client: gen,
	}
}

// logf logs messages if Debug is enabled.
func (g *Gobo) logf(format string, args ...any) {
	if g.config.Debug {
		log.Printf("[Gobo] "+format, args...)
	}
}

// Middleware returns a standard net/http middleware.
func (g *Gobo) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attempt to match the request to a registered schema
		schema := g.match(r)

		// If no schema is registered for this route, fallback to the next handler
		if schema == nil {
			g.logf("No schema matched for %s %s. Passing to next handler.", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		g.logf("Intercepted %s %s (matched %s)", r.Method, r.URL.Path, schema.PathPrefix)

		// 1. Extract context from the incoming request (headers, query, body)
		reqContext := extractRequestContext(r)

		// 2. Query the LLM for a mocked response fitting the schema and request context
		responseBytes, err := g.client.GenerateResponse(r.Context(), reqContext, schema.ResponseSchema)
		if err != nil {
			g.logf("Error generating LLM response: %v", err)
			http.Error(w, "Gobo Mock Generation Failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. Write back the LLM generated JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBytes)
	})
}

// RequestContext represents the parts of an HTTP request we'll feed to the LLM.
type RequestContext struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body,omitempty"`
}

// extractRequestContext pulls relevant info from an http.Request.
func extractRequestContext(r *http.Request) RequestContext {
	ctx := RequestContext{
		Method:  r.Method,
		URL:     r.URL.String(),
		Headers: r.Header,
	}

	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			ctx.Body = string(bodyBytes)
		}
		// restore body so it could potentially be read downstream if needed, though Gobo ends the request.
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return ctx
}
