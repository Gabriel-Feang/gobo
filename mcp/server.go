package mcp

import (
	"context"
	"log"

	"github.com/gabriel-feang/gobo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps a Gobo AsyncBroker and maps its API into MCP tools.
type Server struct {
	broker *gobo.AsyncBroker
	mcp    *mcp.Server
}

// NewServer initializes a new MCP Server tied to the provided broker.
func NewServer(broker *gobo.AsyncBroker) *Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "gobo-mcp", Version: "1.0.0"}, nil)

	srv := &Server{
		broker: broker,
		mcp:    s,
	}

	srv.registerTools()
	return srv
}

// Start begins serving MCP requests over stdio. This method blocks indefinitely.
func (s *Server) Start(ctx context.Context) error {
	transport := &mcp.StdioTransport{}
	return s.mcp.Run(ctx, transport)
}

// WithMCP returns a gobo.Option that wires up an AsyncBroker and starts
// the MCP server on stdio in a background goroutine. This is the simplest
// way to get Gobo + MCP running:
//
//	g := gobo.New(gobomcp.WithMCP())
func WithMCP() gobo.Option {
	return func(g *gobo.Gobo) {
		broker := gobo.NewAsyncBroker()
		g.SetGenerator(broker)

		srv := NewServer(broker)
		go func() {
			log.Println("[gobo-mcp] MCP server starting on stdio")
			if err := srv.Start(context.Background()); err != nil {
				log.Printf("[gobo-mcp] MCP server error: %v", err)
			}
		}()
	}
}

type GetPendingRequestsInput struct{}

type GetPendingRequestsOutput struct {
	Requests []gobo.PendingRequest `json:"requests" jsonschema:"List of pending requests intercepted by Gobo"`
}

type SubmitResponseInput struct {
	RequestID    string `json:"request_id" jsonschema:"The ID of the pending request to fulfill."`
	ResponseJSON string `json:"response_json" jsonschema:"The raw JSON string to return to the blocked HTTP client."`
}

type SubmitResponseOutput struct {
	Message string `json:"message" jsonschema:"Status message indicating success or failure"`
}

func (s *Server) registerTools() {
	// 1. Tool: get_pending_requests
	getReqsTool := &mcp.Tool{
		Name:        "get_pending_requests",
		Description: "Retrieves all HTTP requests currently intercepted and blocked by Gobo that are waiting for an agent to mock a response.",
	}
	mcp.AddTool(s.mcp, getReqsTool, s.handleGetPendingRequests)

	// 2. Tool: submit_response
	submitRespTool := &mcp.Tool{
		Name:        "submit_response",
		Description: "Submits a mocked JSON response to unblock a pending HTTP request intercepted by Gobo.",
	}
	mcp.AddTool(s.mcp, submitRespTool, s.handleSubmitResponse)
}

func (s *Server) handleGetPendingRequests(ctx context.Context, req *mcp.CallToolRequest, input GetPendingRequestsInput) (*mcp.CallToolResult, GetPendingRequestsOutput, error) {
	pending := s.broker.GetPendingRequests()
	// The SDK will automatically marshal this struct into JSON as the StructuredContent
	return nil, GetPendingRequestsOutput{Requests: pending}, nil
}

func (s *Server) handleSubmitResponse(ctx context.Context, req *mcp.CallToolRequest, input SubmitResponseInput) (*mcp.CallToolResult, SubmitResponseOutput, error) {
	err := s.broker.SubmitResponse(input.RequestID, []byte(input.ResponseJSON))
	if err != nil {
		// SDK will auto-wrap this error in a CallToolResult with IsError=true
		return nil, SubmitResponseOutput{}, err
	}

	return nil, SubmitResponseOutput{Message: "Response successfully submitted to Gobo! The blocked HTTP request has been fulfilled."}, nil
}
