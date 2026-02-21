package gobo

import (
	"encoding/json"
	"net/http"
)

// Stub returns an http.Handler for routes that have no real backend.
// When a generator is configured, it uses the generator to produce a response
// based on the request context and schema. When no generator is set, it
// marshals the schema struct as-is (static stub fallback).
func (g *Gobo) Stub(schema any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.logf("Stub handling %s %s", r.Method, r.URL.Path)

		if g.client != nil {
			g.generateAndWrite(w, r, schema)
			return
		}

		// No generator — return the schema struct as static JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(schema)
	})
}

// Intercept wraps a real http.Handler. When a generator is active, it
// intercepts the request and generates a response using the schema. When no
// generator is configured, it passes through to the real handler unchanged.
func (g *Gobo) Intercept(handler http.Handler, schema any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.client != nil {
			g.logf("Intercepting %s %s", r.Method, r.URL.Path)
			g.generateAndWrite(w, r, schema)
			return
		}

		// No generator — pass through to real handler
		handler.ServeHTTP(w, r)
	})
}

// generateAndWrite extracts request context, calls the generator, and writes
// the JSON response. Shared by Stub, Intercept, and Middleware.
func (g *Gobo) generateAndWrite(w http.ResponseWriter, r *http.Request, schema any) {
	reqContext := extractRequestContext(r)

	responseBytes, err := g.client.GenerateResponse(r.Context(), reqContext, schema)
	if err != nil {
		g.logf("Error generating response: %v", err)
		http.Error(w, "Gobo Mock Generation Failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBytes)
}
