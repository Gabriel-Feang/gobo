# Gobo

AI-powered HTTP mocking for Go. Intercept requests, generate intelligent responses.

**Disabled by default.** Only active when `GOBO=1` is set. Production builds are never affected.

## Install

```bash
go get github.com/gabriel-feang/gobo
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/gabriel-feang/gobo"
    _ "github.com/gabriel-feang/gobo/mcp" // auto-registers MCP server on stdio
)

type UserResponse struct {
    ID    string `json:"id" gobo:"A UUID v4"`
    Name  string `json:"name" gobo:"A creative internet handle"`
    Email string `json:"email" gobo:"A valid email for a tech company"`
}

func main() {
    gobo.Start() // no-op unless GOBO=1

    mux := http.NewServeMux()
    mux.Handle("GET /users", gobo.Intercept(realUsersHandler(), UserResponse{}))
    mux.Handle("GET /health", gobo.Stub(HealthResponse{Status: "ok"}))

    http.ListenAndServe(":8080", mux)
}
```

That's it. Three things:
1. `gobo.Start()` in main
2. `gobo.Intercept(handler, schema)` or `gobo.Stub(schema)` on your routes
3. Run with `GOBO=1` to enable

## How It Works

When `GOBO=1`:
- `gobo.Intercept(handler, schema)` intercepts the request and routes it to an AI generator
- `gobo.Stub(schema)` generates a response from the schema
- The default generator is an `AsyncBroker` that parks requests until an MCP-connected AI agent submits responses

When `GOBO` is not set:
- `gobo.Intercept(handler, schema)` passes through to `handler` unchanged
- `gobo.Stub(schema)` returns the schema as static JSON
- Zero overhead

## Schema Tags

Use `gobo` struct tags to instruct the AI per field:

```go
type Payment struct {
    TransactionID string `json:"transaction_id" gobo:"A valid UUID v4"`
    Status        string `json:"status" gobo:"Must always be 'APPROVED'"`
    Amount        int    `json:"amount" gobo:"Amount in cents, between 100 and 99999"`
}
```

## Running with Mage

Add a target to your `magefile.go`:

```go
func Gobo() error {
    cmd := exec.Command("go", "run", ".")
    cmd.Env = append(os.Environ(), "GOBO=1")
    cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
    return cmd.Run()
}
```

Then: `mage gobo`

## MCP Integration

Import `_ "github.com/gabriel-feang/gobo/mcp"` and the MCP server starts automatically on stdio when `GOBO=1`. It exposes two tools:

- **`get_pending_requests`** — list HTTP requests waiting for mock responses
- **`submit_response`** — submit JSON to unblock a pending request

Configure your MCP client:

```json
{
  "mcpServers": {
    "gobo": {
      "command": "go",
      "args": ["run", "."],
      "env": { "GOBO": "1" }
    }
  }
}
```

## Advanced: Instance API

For custom generators or multiple instances:

```go
g := gobo.New(gobo.WithOllama("http://localhost:11434", "llama3"))
mux.Handle("GET /users", g.Intercept(handler, UserResponse{}))
```

### Options

```go
gobo.WithOllama(url, model)   // Use local Ollama
gobo.WithGenerator(gen)       // Custom Generator implementation
gobo.WithDebug()              // Verbose logging
gobo.WithConfig(cfg)          // Full Config struct
```

### Generator Interface

```go
type Generator interface {
    GenerateResponse(ctx context.Context, reqCtx RequestContext, schema any) ([]byte, error)
}
```

## For AI Agents

If you are an AI agent using MCP: Gobo is designed for you. Start the app with `GOBO=1`, connect via MCP, and use `get_pending_requests` / `submit_response` to fulfill intercepted HTTP calls.
