package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gabriel-feang/gobo"
)

// UserResponse is the schema we expect the mock to generate
type UserResponse struct {
	ID       string `json:"id" gobo:"A UUID v4 format string"`
	Username string `json:"username" gobo:"A creative internet handle"`
	Email    string `json:"email" gobo:"A valid email address for a tech company"`
	Role     string `json:"role" gobo:"Must be exactly one of: 'admin', 'user', 'guest', or 'moderator'"`
}

func main() {
	// Initialize Gobo with a local Ollama instance
	mock := gobo.New(
		gobo.WithOllama("http://localhost:11434", "llama3"),
		gobo.WithDebug(),
	)

	// Register a mock schema for GET /users
	mock.Register("GET", "/users/", UserResponse{})

	// Create a standard mux
	mux := http.NewServeMux()

	// An endpoint that we DID NOT mock, to show the fallback works
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "pong"}`))
	})

	// Wrap the mux with our Gobo middleware
	handler := mock.Middleware(mux)

	fmt.Println("Starting example server on :8080...")
	fmt.Println("Try running: curl http://localhost:8080/users/123")
	fmt.Println("Try running: curl http://localhost:8080/ping")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
