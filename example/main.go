package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gabriel-feang/gobo"
	_ "github.com/gabriel-feang/gobo/mcp" // auto-registers MCP server
)

// UserResponse is the schema we expect the mock to generate
type UserResponse struct {
	ID       string `json:"id" gobo:"A UUID v4 format string"`
	Username string `json:"username" gobo:"A creative internet handle"`
	Email    string `json:"email" gobo:"A valid email address for a tech company"`
	Role     string `json:"role" gobo:"Must be exactly one of: 'admin', 'user', 'guest', or 'moderator'"`
}

func main() {
	gobo.Start(gobo.WithDebug())

	mux := http.NewServeMux()

	// A stubbed route — no real handler, Gobo generates the response
	mux.Handle("GET /users/{id}", gobo.Stub(UserResponse{}))

	// A real route — not intercepted by Gobo
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "pong"}`))
	})

	fmt.Println("Starting example server on :8080...")
	fmt.Println("Try running: curl http://localhost:8080/users/123")
	fmt.Println("Try running: curl http://localhost:8080/ping")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
