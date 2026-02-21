package gobo

import (
	"net/http"
	"strings"
)

// Register configures the Gobo instance to intercept a specific method and path prefix.
// The responseSchema argument should be a sample JSON-marshalable struct or map representing the expected output.
// The LLM will use this parameter to infer the structure of the JSON it must return.
func (g *Gobo) Register(method, pathPrefix string, responseSchema any) {
	method = strings.ToUpper(method)

	// Ensure the path prefix plays nicely with comparisons
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}

	g.routes = append(g.routes, &routeSchema{
		Method:         method,
		PathPrefix:     pathPrefix,
		ResponseSchema: responseSchema,
	})

	g.logf("Registered mock schema for %s %s", method, pathPrefix)
}

// match tries to find a registered schema for the incoming request's method and path.
// Matches by exact method (or "ANY") and string prefix matching on the path.
func (g *Gobo) match(r *http.Request) *routeSchema {
	for _, route := range g.routes {
		if (route.Method == "ANY" || route.Method == r.Method) && strings.HasPrefix(r.URL.Path, route.PathPrefix) {
			return route
		}
	}
	return nil
}
