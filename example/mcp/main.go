package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/feang/gobo"
	"github.com/feang/gobo/mcp"
)

// To try this MCP Server:
// 1. Build this file: `go build -o gobo-mcp main.go`
// 2. Configure your MCP Client (like Claude Desktop) to use the `gobo-mcp` executable.
// 3. When the MCP Client starts it, Gobo will intercept requests and expose the tools via stdio!
func main() {
	// 1. We create the AsyncBroker
	broker := gobo.NewAsyncBroker()

	// 2. We initialize Gobo using the broker
	mock := gobo.New(gobo.Config{
		Generator: broker,
		Debug:     true, // Will print to stderr so it doesn't mess with MCP stdio
	})

	// 3. Register a sample application endpoint we want to intercept
	type SampleSchema struct {
		Message string `json:"message" gobo:"A fun greeting from the AI"`
		Status  int    `json:"status" gobo:"Always 200"`
	}
	mock.Register("GET", "/api/greet", SampleSchema{})

	// 4. Start the mocked server in the background
	ts := httptest.NewServer(mock.Middleware(http.NewServeMux()))
	defer ts.Close()

	log.SetOutput(os.Stderr)
	log.Printf("Gobo is intercepting requests at: %s", ts.URL)
	log.Printf("To test: curl %s/api/greet", ts.URL)
	log.Println("Starting Gobo MCP Server on stdio...")

	// 5. Wrap the broker in the MCP Server and block forever!
	mcpSrv := mcp.NewServer(broker)
	if err := mcpSrv.Start(); err != nil {
		log.Fatalf("MCP Server crashed: %v", err)
	}
}
