package gobo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGobo_Passthrough(t *testing.T) {
	g := New()

	nextHandlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	middleware := g.Middleware(next)

	req := httptest.NewRequest("GET", "/unmatched", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if !nextHandlerCalled {
		t.Errorf("Expected next handler to be called for unmatched route")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "ok" {
		t.Errorf("Expected body 'ok', got %q", rr.Body.String())
	}
}

func TestGobo_Match(t *testing.T) {
	g := New()
	g.Register("POST", "/users", map[string]string{"result": "success"})

	req1 := httptest.NewRequest("POST", "/users", nil)
	if schema := g.match(req1); schema == nil {
		t.Errorf("Expected match for POST /users")
	}

	req2 := httptest.NewRequest("GET", "/users", nil)
	if schema := g.match(req2); schema != nil {
		t.Errorf("Did not expect match for GET /users")
	}

	req3 := httptest.NewRequest("POST", "/unrelated", nil)
	if schema := g.match(req3); schema != nil {
		t.Errorf("Did not expect match for POST /unrelated")
	}
}

func TestExtractRequestContext(t *testing.T) {
	req := httptest.NewRequest("POST", "/test?query=1", strings.NewReader(`{"hello":"world"}`))
	req.Header.Set("Content-Type", "application/json")

	ctx := extractRequestContext(req)

	if ctx.Method != "POST" {
		t.Errorf("Expected method POST, got %s", ctx.Method)
	}
	if ctx.URL != "/test?query=1" {
		t.Errorf("Expected URL /test?query=1, got %s", ctx.URL)
	}
	if ctx.Body != `{"hello":"world"}` {
		t.Errorf("Expected body, got %s", ctx.Body)
	}
	if ct := http.Header(ctx.Headers).Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	// Verify the body wasn't destroyed
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Failed to read restored body: %v", err)
	}
	if string(bodyBytes) != `{"hello":"world"}` {
		t.Errorf("Restored body mismatched: %s", string(bodyBytes))
	}
}
