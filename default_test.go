package gobo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackageLevelStub_Disabled(t *testing.T) {
	// GOBO is not set, so Stub should return static JSON
	type Response struct {
		Status string `json:"status"`
	}

	handler := Stub(Response{Status: "ok"})
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected application/json, got %s", ct)
	}
}

func TestPackageLevelIntercept_Disabled(t *testing.T) {
	// GOBO is not set, so Intercept should pass through to real handler
	realCalled := false
	real := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("real"))
	})

	handler := Intercept(real, struct{}{})
	req := httptest.NewRequest("GET", "/users", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !realCalled {
		t.Error("Expected real handler to be called when Gobo is disabled")
	}
	if rr.Body.String() != "real" {
		t.Errorf("Expected 'real', got %q", rr.Body.String())
	}
}

func TestEnabled_DefaultFalse(t *testing.T) {
	if Enabled() {
		t.Error("Expected Enabled() to be false by default")
	}
}
