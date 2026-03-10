/*
Package gobo intercepts HTTP requests and generates intelligent mock responses
using AI, so you never have to hard-code JSON stubs again.

# Quick Start

Add one line to main() and wrap your routes:

	import (
	    "net/http"
	    "github.com/gabriel-feang/gobo"
	    _ "github.com/gabriel-feang/gobo/mcp" // auto-registers MCP server
	)

	type UserResponse struct {
	    ID    string `json:"id" gobo:"A UUID v4"`
	    Name  string `json:"name" gobo:"A creative internet handle"`
	    Email string `json:"email" gobo:"A valid email for a tech company"`
	}

	func main() {
	    gobo.Start()  // only active when GOBO=1 env var is set

	    mux := http.NewServeMux()
	    mux.Handle("GET /users", gobo.Intercept(realUsersHandler, UserResponse{}))
	    mux.Handle("GET /health", gobo.Stub(HealthResponse{Status: "ok"}))

	    http.ListenAndServe(":8080", mux)
	}

# Safety

Gobo is disabled by default. It only activates when the GOBO=1 environment
variable is set. Production builds will never get stuck waiting for interception.

# How It Works

When enabled, [Intercept] and [Stub] intercept HTTP requests and route them to
a [Generator] (by default, an [AsyncBroker] connected to an MCP server on stdio).
An AI agent connected via MCP can then inspect pending requests and submit
mock responses.

When disabled, [Intercept] passes through to the real handler and [Stub]
returns the schema struct as static JSON. Zero overhead.

# Schema Tags

Use the "gobo" struct tag to give field-level instructions to the AI:

	type Payment struct {
	    TransactionID string `json:"transaction_id" gobo:"A valid UUID v4"`
	    Status        string `json:"status" gobo:"Must always be 'APPROVED'"`
	    Amount        int    `json:"amount" gobo:"Amount in cents, between 100 and 99999"`
	}

# Instance API

For advanced use cases, create your own [Gobo] instance:

	g := gobo.New(gobo.WithOllama("http://localhost:11434", "llama3"))
	mux.Handle("GET /users", g.Intercept(handler, UserResponse{}))
*/
package gobo
