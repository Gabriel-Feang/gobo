package gobo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// llmClient interacts with the configured Ollama instance.
type llmClient struct {
	config     Config
	httpClient *http.Client
}

// newLLMClient initializes a new client.
func newLLMClient(cfg Config) *llmClient {
	return &llmClient{
		config: cfg,
		httpClient: &http.Client{
			// Give the LLM enough time to generate a response
			Timeout: 2 * time.Minute,
		},
	}
}

// GenerateResponse queries the LLM and attempts to parse its output back as JSON matching the schema.
func (c *llmClient) GenerateResponse(ctx context.Context, reqCtx RequestContext, schema any) ([]byte, error) {
	prompt, err := c.buildPrompt(reqCtx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	ollamaReq := map[string]any{
		"model":  c.config.Model,
		"prompt": prompt,
		// Force JSON mode in Ollama
		"format": "json",
		"stream": false,
		"options": map[string]any{
			"temperature": 0.2, // Keep it low for structural adherence
		},
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	endpoint := strings.TrimRight(c.config.OllamaURL, "/") + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama responded with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	// Validate the returned bytes represent a valid JSON structure
	outBytes := []byte(ollamaResp.Response)
	if !json.Valid(outBytes) {
		return nil, fmt.Errorf("llm returned invalid json")
	}

	return outBytes, nil
}

// buildPrompt constructs the prompt containing the HTTP request details and the required JSON schema.
func (c *llmClient) buildPrompt(reqCtx RequestContext, schema any) (string, error) {
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	reqCtxBytes, err := json.MarshalIndent(reqCtx, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal request context: %w", err)
	}

	// Dynamically parse the struct tags to provide explicit field guidance to the LLM
	fields := reflectSchema(schema)
	fieldInstructions := formatFieldInstructions(fields)

	prompt := fmt.Sprintf(`You are a smart mock server named Gobo. 
Your job is to intercept HTTP requests and generate realistic JSON responses based strictly on the provided Response Schema.
You must use the Request Context to understand what the user is asking for, and generate a fitting response that perfectly matches the Response Schema.

=== Request Context ===
%s

=== Expected Output JSON Structure ===
%s

=== Field Instructions ===
Pay close attention to these explicit data-generation instructions for specific fields:
%s

IMPORTANT: 
- Return ONLY valid JSON.
- The root of your output must match the Expected Output JSON Structure exactly.
- Provide realistic and contextually appropriate fake data.
- If the Request Context provides IDs or names, try to reuse them in the response if the schema permits.
- Strictly adhere to any custom instructions provided in the Field Instructions section.
`, string(reqCtxBytes), string(schemaBytes), fieldInstructions)

	return prompt, nil
}
