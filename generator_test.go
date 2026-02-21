package gobo

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

type mockGenerator struct {
	Response []byte
}

func (m *mockGenerator) GenerateResponse(ctx context.Context, reqCtx RequestContext, schema any) ([]byte, error) {
	return m.Response, nil
}

func TestCustomGenerator(t *testing.T) {
	expectedJSON := `{"id":"123","username":"testuser"}`

	g := New(Config{
		Generator: &mockGenerator{Response: []byte(expectedJSON)},
		Debug:     true,
	})

	g.Register("GET", "/users", map[string]string{})

	schema := g.match(&http.Request{Method: "GET", URL: &url.URL{Path: "/users"}})
	if schema == nil {
		t.Fatalf("Expected match")
	}

	respBytes, err := g.client.GenerateResponse(context.Background(), RequestContext{}, schema.ResponseSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(respBytes) != expectedJSON {
		t.Errorf("Expected %s, got %s", expectedJSON, string(respBytes))
	}
}
