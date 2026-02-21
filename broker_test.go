package gobo

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestAsyncBroker_Blocking(t *testing.T) {
	broker := NewAsyncBroker()

	g := New(WithGenerator(broker), WithDebug())
	g.Register("GET", "/async", map[string]string{"foo": "bar"})

	// We simulate the HTTP server by calling match and GenerateResponse in a goroutine
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/async"}}
	schema := g.match(req)

	if schema == nil {
		t.Fatalf("Expected match")
	}

	resultCh := make(chan []byte)

	go func() {
		// GenerateResponse will block here
		respBytes, err := broker.GenerateResponse(context.Background(), extractRequestContext(req), schema.ResponseSchema)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		resultCh <- respBytes
	}()

	// Give the goroutine time to register the pending request
	time.Sleep(50 * time.Millisecond)

	pending := broker.GetPendingRequests()
	if len(pending) != 1 {
		t.Fatalf("Expected 1 pending request, got %d", len(pending))
	}

	reqID := pending[0].ID
	if pending[0].Method != "GET" || pending[0].URL != "/async" {
		t.Errorf("Unexpected pending request details: %+v", pending[0])
	}

	// Now the agent fulfills the request
	expectedJSON := `{"async":"success"}`
	err := broker.SubmitResponse(reqID, []byte(expectedJSON))
	if err != nil {
		t.Fatalf("Failed to submit response: %v", err)
	}

	select {
	case res := <-resultCh:
		if string(res) != expectedJSON {
			t.Errorf("Expected result %s, got %s", expectedJSON, string(res))
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for GenerateResponse to unblock")
	}

	// Verify pending request is removed
	time.Sleep(50 * time.Millisecond) // wait for defer lock cleanup
	pendingEnd := broker.GetPendingRequests()
	if len(pendingEnd) != 0 {
		t.Fatalf("Expected 0 pending requests after completion, got %d", len(pendingEnd))
	}
}
