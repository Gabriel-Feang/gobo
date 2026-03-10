# Gobo — AI-Powered HTTP Mocking for Go

Gobo intercepts HTTP requests and lets an AI agent generate mock responses. It is **disabled by default** and only activates when `GOBO=1` is set.

## Setup Instructions

When a user asks you to "setup gobo", follow these steps exactly:

### 1. Add the dependency

```bash
go get github.com/gabriel-feang/gobo
```

### 2. Add gobo.Start() to main.go

Add these two lines to the imports and one line to the top of `main()`:

```go
import (
    "github.com/gabriel-feang/gobo"
    _ "github.com/gabriel-feang/gobo/mcp" // registers MCP server for AI agent communication
)

func main() {
    gobo.Start() // no-op unless GOBO=1 is set
    // ... rest of main
}
```

### 3. Wrap HTTP handlers with gobo.Intercept

Find routes that call external services or return data you want to mock. Wrap them:

```go
// Before:
mux.Handle("GET /api/users", getUsersHandler())

// After:
type UserResponse struct {
    ID    string `json:"id" gobo:"A UUID v4"`
    Name  string `json:"name" gobo:"A realistic full name"`
    Email string `json:"email" gobo:"A valid email address"`
}
mux.Handle("GET /api/users", gobo.Intercept(getUsersHandler(), UserResponse{}))
```

For routes with no real handler yet, use `gobo.Stub`:

```go
mux.Handle("GET /api/health", gobo.Stub(HealthResponse{Status: "ok"}))
```

### 4. Create a Mage target (if user wants `mage gobo`)

Create or edit `magefile.go`:

```go
//go:build mage

package main

import (
    "fmt"
    "os"
    "os/exec"
)

// Gobo runs the application with Gobo interception enabled.
// The MCP server starts on stdio for AI agent communication.
func Gobo() error {
    fmt.Println("Starting with Gobo enabled...")
    cmd := exec.Command("go", "run", ".")
    cmd.Env = append(os.Environ(), "GOBO=1")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin
    return cmd.Run()
}
```

Then run: `mage gobo`

### 5. Configure MCP client (Claude Desktop, Claude Code, etc.)

For Claude Code, add to `.mcp.json`:

```json
{
  "mcpServers": {
    "gobo": {
      "command": "mage",
      "args": ["gobo"]
    }
  }
}
```

Or if running the binary directly:

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

## API Reference

### Package-level functions (use the default instance)

| Function | Description |
|----------|-------------|
| `gobo.Start(opts ...Option)` | Initialize gobo. Only active when `GOBO=1`. Starts MCP server on stdio. |
| `gobo.Stub(schema any) http.Handler` | Handler for mock-only routes. Returns generated or static JSON. |
| `gobo.Intercept(handler, schema) http.Handler` | Wraps a real handler. Intercepts when enabled, passes through when disabled. |
| `gobo.InterceptFunc(fn, schema) http.Handler` | Same as Intercept but accepts `http.HandlerFunc`. |
| `gobo.Enabled() bool` | Reports whether gobo is active. |
| `gobo.Default() *Gobo` | Returns the default instance for advanced use. |

### Instance-level (advanced)

```go
g := gobo.New(gobo.WithOllama("http://localhost:11434", "llama3"))
g := gobo.New(gobo.WithGenerator(customGen))
g := gobo.New(gobo.WithDebug())
```

### Schema tags

Use `gobo` struct tags to give the AI instructions per field:

```go
type Order struct {
    ID     string `json:"id" gobo:"A UUID v4"`
    Status string `json:"status" gobo:"One of: pending, approved, rejected"`
    Total  int    `json:"total" gobo:"Amount in cents, always positive"`
}
```

### Generator interface

Implement this to use any AI backend:

```go
type Generator interface {
    GenerateResponse(ctx context.Context, reqCtx RequestContext, schema any) ([]byte, error)
}
```

### AsyncBroker (for agent-driven testing)

The default generator when using MCP. It parks HTTP requests until an agent submits a response:

```go
broker := gobo.NewAsyncBroker()
// broker.GetPendingRequests() — list blocked requests
// broker.SubmitResponse(id, jsonBytes) — unblock a request
```

## Key Design Principles

1. **Disabled by default** — `GOBO=1` env var required. Production is never affected.
2. **Zero overhead when disabled** — `Intercept` returns the original handler, `Stub` returns static JSON.
3. **Schema-driven** — Define expected response shapes as Go structs with `gobo` tags.
4. **MCP-native** — Import `_ "github.com/gabriel-feang/gobo/mcp"` and the MCP server auto-starts on stdio.

## File Structure

- `gobo.go` — Core Gobo struct, options, middleware, request context
- `default.go` — Package-level API (Start, Stub, Intercept) using default instance
- `handler.go` — Stub and Intercept handler implementations
- `schema.go` — Route registration and matching (for middleware API)
- `reflect.go` — Struct reflection and gobo tag extraction
- `llm.go` — Ollama LLM generator
- `broker.go` — AsyncBroker for agent-driven responses
- `mcp/server.go` — MCP server with get_pending_requests and submit_response tools
- `doc.go` — Package documentation
