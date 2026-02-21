/*
Package gobo provides an AI-powered HTTP middleware for simulating downstream services
in integration tests.

Instead of meticulously hard-coding JSON stubs for every edge case in your tests,
you can use Gobo to define an expected struct schema for a given route. Gobo intercepts
the network call, sends the request's context and your schema to a local LLM
(like Ollama), and seamlessly unmarshals the realistic AI-generated JSON back to
your application.

# Key Features

  - Schema-driven mocking: Register routes with expected response schemas.
  - Dynamic Prompt Engineering: Use the `gobo` struct tag to provide explicit
    instructions to the LLM for specific fields.
  - Extensible: Provide your own `Generator` implementation to use OpenAI, Anthropic,
    or other AI models instead of the default local Ollama instance.
  - AsyncBroker: A specialized `Generator` that allows MCP-connected test agents
    to manually fulfill intercepted requests asynchronously without tripping HTTP timeouts.

# Basic Usage

	mock := gobo.New(gobo.Config{
		OllamaURL: "http://localhost:11434",
		Model:     "llama3",
	})

	type PaymentResponse struct {
		TransactionID string `json:"transaction_id" gobo:"A valid UUID v4"`
		Status        string `json:"status" gobo:"Must always be 'APPROVED'"`
	}

	mock.Register("POST", "/v1/charge", PaymentResponse{})

	// Wrap your standard mux or test server
	ts := httptest.NewServer(mock.Middleware(http.NewServeMux()))
	defer ts.Close()

# The AsyncBroker (For Agents)

If testing via an AI agent, or if you need to manually inspect and orchestrate mocked
responses without LLM timeouts, use the `AsyncBroker`.

	broker := gobo.NewAsyncBroker()
	mock := gobo.New(gobo.Config{
		Generator: broker,
	})

	// ... Meanwhile, in another Go routine or via an Agent MCP Tool:
	pending := broker.GetPendingRequests()
	if len(pending) > 0 {
		broker.SubmitResponse(pending[0].ID, []byte(`{"status": "APPROVED"}`))
	}
*/
package gobo
