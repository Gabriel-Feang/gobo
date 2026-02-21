package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gabriel-feang/gobo"
	gobomcp "github.com/gabriel-feang/gobo/mcp"
)

// To try this MCP Server:
// 1. Build this file: `go build -o gobo-mcp main.go`
// 2. Configure your MCP Client (like Claude Desktop) to use the `gobo-mcp` executable.
// 3. When the MCP Client starts it, Gobo will intercept requests and expose the tools via stdio!
func main() {
	log.SetOutput(os.Stderr)

	// One-liner: creates AsyncBroker + MCP server on stdio automatically
	g := gobo.New(gobomcp.WithMCP(), gobo.WithDebug())

	// Register routes using the handler-based API
	type SampleSchema struct {
		Message string `json:"message" gobo:"A fun greeting from the AI"`
		Status  int    `json:"status" gobo:"Always 200"`
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/greet", g.Stub(SampleSchema{}))

	// HTTP server — MCP is already running in the background
	addr := ":8080"
	fmt.Fprintf(os.Stderr, "Gobo HTTP server on http://localhost%s\n", addr)
	fmt.Fprintf(os.Stderr, "To test: curl http://localhost%s/api/greet\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
