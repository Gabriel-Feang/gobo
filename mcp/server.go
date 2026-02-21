package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gabriel-feang/gobo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps a Gobo AsyncBroker and maps its API into MCP tools.
type Server struct {
	broker *gobo.AsyncBroker
	mcp    *server.MCPServer
}

// NewServer initializes a new MCP Server tied to the provided broker.
func NewServer(broker *gobo.AsyncBroker) *Server {
	s := server.NewMCPServer("gobo-mcp", "1.0.0", server.WithToolCapabilities(true))

	srv := &Server{
		broker: broker,
		mcp:    s,
	}

	srv.registerTools()
	return srv
}

// Start begins serving MCP requests over stdio. This method blocks indefinitely.
func (s *Server) Start() error {
	return server.ServeStdio(s.mcp)
}

func (s *Server) registerTools() {
	// 1. Tool: get_pending_requests
	getReqsTool := mcp.NewTool("get_pending_requests",
		mcp.WithDescription("Retrieves all HTTP requests currently intercepted and blocked by Gobo that are waiting for an agent to mock a response."),
	)
	s.mcp.AddTool(getReqsTool, s.handleGetPendingRequests)

	// 2. Tool: submit_response
	submitRespTool := mcp.NewTool("submit_response",
		mcp.WithDescription("Submits a mocked JSON response to unblock a pending HTTP request intercepted by Gobo."),
		mcp.WithString("request_id", mcp.Required(), mcp.Description("The ID of the pending request to fulfill.")),
		mcp.WithString("response_json", mcp.Required(), mcp.Description("The raw JSON string to return to the blocked HTTP client.")),
	)
	s.mcp.AddTool(submitRespTool, s.handleSubmitResponse)
}

func (s *Server) handleGetPendingRequests(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pending := s.broker.GetPendingRequests()

	bytes, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal pending requests: %v", err)), nil
	}

	return mcp.NewToolResultText(string(bytes)), nil
}

func (s *Server) handleSubmitResponse(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("arguments missing or invalid type"), nil
	}

	reqID, ok := args["request_id"].(string)
	if !ok {
		return mcp.NewToolResultError("request_id is required and must be a string"), nil
	}

	respJSONStr, ok := args["response_json"].(string)
	if !ok {
		return mcp.NewToolResultError("response_json is required and must be a string"), nil
	}

	// Double-check the Agent gave us valid JSON before unblocking Gobo
	if !json.Valid([]byte(respJSONStr)) {
		return mcp.NewToolResultError("response_json is not valid JSON"), nil
	}

	err := s.broker.SubmitResponse(reqID, []byte(respJSONStr))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to submit response: %v", err)), nil
	}

	return mcp.NewToolResultText("Response successfully submitted to Gobo! The blocked HTTP request has been fulfilled."), nil
}
