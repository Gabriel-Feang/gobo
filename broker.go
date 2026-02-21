package gobo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PendingRequest represents an HTTP request intercepted by Gobo that is waiting for an agent to mock a response.
type PendingRequest struct {
	ID        string         `json:"id"`
	Method    string         `json:"method"`
	URL       string         `json:"url"`
	Context   RequestContext `json:"context"`
	Schema    any            `json:"schema"` // Used by agents to understand what to generate
	Timestamp time.Time      `json:"timestamp"`
}

// responseChannel allows sending the mocked JSON back to the blocked Generation routine.
type responseChannel chan []byte

// AsyncBroker implements the Generator interface. It intentionally blocks the HTTP request
// and exposes an API for external agents to pull pending requests and submit JSON responses asynchronously.
type AsyncBroker struct {
	mu       sync.Mutex
	pending  map[string]PendingRequest
	channels map[string]responseChannel
}

// NewAsyncBroker creates a new broker ready to be passed to Gobo's config.
func NewAsyncBroker() *AsyncBroker {
	return &AsyncBroker{
		pending:  make(map[string]PendingRequest),
		channels: make(map[string]responseChannel),
	}
}

// GenerateResponse implements the Generator interface.
// It parks the incoming HTTP request indefinitely until an external agent submits a response via SubmitResponse,
// or the request context is cancelled.
func (b *AsyncBroker) GenerateResponse(ctx context.Context, reqCtx RequestContext, schema any) ([]byte, error) {
	reqID := uuid.New().String()

	respChan := make(responseChannel, 1)

	// Create and register the pending request
	pr := PendingRequest{
		ID:        reqID,
		Method:    reqCtx.Method,
		URL:       reqCtx.URL,
		Context:   reqCtx,
		Schema:    schema,
		Timestamp: time.Now(),
	}

	b.mu.Lock()
	b.pending[reqID] = pr
	b.channels[reqID] = respChan
	b.mu.Unlock()

	defer func() {
		// Cleanup when the request finishes or aborts
		b.mu.Lock()
		delete(b.pending, reqID)
		delete(b.channels, reqID)
		b.mu.Unlock()
	}()

	// Wait for the agent to answer or the HTTP context to naturally cancel
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("request cancelled by client")
	case respBytes := <-respChan:
		return respBytes, nil
	}
}

// GetPendingRequests returns all currently blocked HTTP requests waiting for an agent.
func (b *AsyncBroker) GetPendingRequests() []PendingRequest {
	b.mu.Lock()
	defer b.mu.Unlock()

	var reqs []PendingRequest
	for _, pr := range b.pending {
		reqs = append(reqs, pr)
	}
	return reqs
}

// SubmitResponse is called by an external agent to immediately flush the generated JSON to the blocked HTTP request.
func (b *AsyncBroker) SubmitResponse(id string, responseJSON []byte) error {
	b.mu.Lock()
	ch, exists := b.channels[id]
	b.mu.Unlock()

	if !exists {
		return fmt.Errorf("no pending request found with id %s", id)
	}

	// Send the JSON bytes back to the blocked HTTP handler
	ch <- responseJSON
	return nil
}
