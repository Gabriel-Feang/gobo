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

// Option is a functional option for configuring a Gobo instance.
type Option func(*Gobo)

// WithConfig applies a full Config struct (backward-compatible with the old API).
func WithConfig(cfg Config) Option {
	return func(g *Gobo) {
		g.config = cfg
		if cfg.Generator != nil {
			g.client = cfg.Generator
		} else {
			if cfg.OllamaURL == "" {
				cfg.OllamaURL = "http://localhost:11434"
			}
			g.client = NewOllamaGenerator(cfg.OllamaURL, cfg.Model)
		}
	}
}

// WithOllama configures Gobo to use a local Ollama instance.
func WithOllama(url, model string) Option {
	return func(g *Gobo) {
		g.config.OllamaURL = url
		g.config.Model = model
		g.client = NewOllamaGenerator(url, model)
	}
}

// WithDebug enables verbose logging.
func WithDebug() Option {
	return func(g *Gobo) {
		g.config.Debug = true
	}
}

// WithGenerator sets a custom Generator implementation.
func WithGenerator(gen Generator) Option {
	return func(g *Gobo) {
		g.config.Generator = gen
		g.client = gen
	}
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

// New creates a new Gobo instance with functional options.
func New(opts ...Option) *Gobo {
	g := &Gobo{
		routes: make([]*routeSchema, 0),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// SetGenerator replaces the current generator. This is used by external
// packages (like gobo/mcp) that need to wire in a generator after construction.
func (g *Gobo) SetGenerator(gen Generator) {
	g.config.Generator = gen
	g.client = gen
}

// logf logs messages if Debug is enabled.
func (g *Gobo) logf(format string, args ...any) {
	if g.config.Debug {
		log.Printf("[Gobo] "+format, args...)
	}
}

// Middleware returns a standard net/http middleware that intercepts requests
// matching registered schemas via Register(). Unmatched routes pass through.
func (g *Gobo) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		schema := g.match(r)

		if schema == nil {
			g.logf("No schema matched for %s %s. Passing to next handler.", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		g.logf("Intercepted %s %s (matched %s)", r.Method, r.URL.Path, schema.PathPrefix)
		g.generateAndWrite(w, r, schema.ResponseSchema)
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
