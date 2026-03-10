package gobo

import (
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	defaultInstance *Gobo
	defaultOnce     sync.Once
	enabled         bool

	// startMCP is set by the gobo/mcp package via RegisterMCPStarter.
	// This avoids a circular import while allowing Start() to auto-launch MCP.
	mcpStarter func(g *Gobo)
)

func init() {
	defaultInstance = New()
	enabled = os.Getenv("GOBO") == "1" || os.Getenv("GOBO") == "true"
}

// Enabled reports whether Gobo interception is active.
// Gobo is enabled by setting the GOBO=1 (or GOBO=true) environment variable.
// When disabled, Stub returns static JSON and Intercept passes through to the real handler.
func Enabled() bool {
	return enabled
}

// Start initializes the default Gobo instance. When GOBO=1 is set, it starts
// the MCP server on stdio so an AI agent can fulfill intercepted requests.
// When GOBO is not set, Start is a no-op and all Stub/Intercept calls pass through.
//
// Call this once at the top of main():
//
//	func main() {
//	    gobo.Start()
//	    mux := http.NewServeMux()
//	    mux.Handle("GET /users", gobo.Intercept(realHandler, UserResponse{}))
//	    http.ListenAndServe(":8080", mux)
//	}
func Start(opts ...Option) {
	if !enabled {
		log.Println("[gobo] disabled (set GOBO=1 to enable)")
		return
	}

	defaultOnce.Do(func() {
		for _, opt := range opts {
			opt(defaultInstance)
		}

		// If no generator was provided, set up the AsyncBroker + MCP server
		if defaultInstance.client == nil && mcpStarter != nil {
			mcpStarter(defaultInstance)
		}

		log.Println("[gobo] enabled — intercepting HTTP requests")
	})
}

// Stub returns an http.Handler for routes that have no real backend.
// When Gobo is enabled and a generator is active, it generates a response.
// When disabled, it marshals the schema struct as static JSON.
//
// Usage:
//
//	mux.Handle("GET /health", gobo.Stub(HealthResponse{Status: "ok"}))
func Stub(schema any) http.Handler {
	return defaultInstance.Stub(schema)
}

// Intercept wraps a real http.Handler. When Gobo is enabled and a generator
// is active, it intercepts the request and generates a response from the schema.
// When disabled, it passes through to the real handler — zero overhead.
//
// Usage:
//
//	mux.Handle("POST /users", gobo.Intercept(realHandler, UserResponse{}))
func Intercept(handler http.Handler, schema any) http.Handler {
	return defaultInstance.Intercept(handler, schema)
}

// InterceptFunc is a convenience wrapper around Intercept for http.HandlerFunc.
//
// Usage:
//
//	mux.Handle("POST /users", gobo.InterceptFunc(handleCreateUser, UserResponse{}))
func InterceptFunc(handler http.HandlerFunc, schema any) http.Handler {
	return defaultInstance.Intercept(handler, schema)
}

// RegisterMCPStarter is called by the gobo/mcp package's init() to register
// the MCP server launcher. This avoids a circular import.
func RegisterMCPStarter(fn func(g *Gobo)) {
	mcpStarter = fn
}

// Default returns the default Gobo instance used by the package-level functions.
// Use this for advanced configuration or to access the instance-based API.
func Default() *Gobo {
	return defaultInstance
}
