package app

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabriel-feang/gobo"
)

func TestOrderService_WithGoboMock(t *testing.T) {
	// 1. Initialize Gobo to intercept the PaymentGateway requests
	// We'll use the AsyncBroker so this test doesn't actually require a running Ollama model.
	// In a real project, you would wire a real Generator (like Ollama or OpenAI) to power it!
	broker := gobo.NewAsyncBroker()

	mock := gobo.New(gobo.Config{
		Generator: broker,
		Debug:     true,
	})

	// 2. Define the schema we expect the mocked payment gateway to return
	type ExpectedGatewayResponse struct {
		TransactionID string `json:"transaction_id" gobo:"A UUID v4 representing the bank transaction"`
		Status        string `json:"status" gobo:"Must always be exactly 'APPROVED'"`
	}

	// Register it to intercept POST /charge
	mock.Register("POST", "/charge", ExpectedGatewayResponse{})

	// 3. Start a test server wrapped with our Gobo middleware.
	// Any non-intercepted requests would hit the empty mux,
	// but Gobo will catch our /charge attempt.
	ts := httptest.NewServer(mock.Middleware(http.NewServeMux()))
	defer ts.Close()

	// 4. Initialize our application, pointing it to our Gobo mock server
	service := &OrderService{
		PaymentGatewayURL: ts.URL,
		HTTPClient: &http.Client{
			Timeout: 2 * time.Minute, // Long timeout since Ollama needs time to answer
		},
	}

	order := Order{
		OrderID: "ORD-9999",
		UserID:  "USER-8888",
		Amount:  12500, // $125.00
	}

	log.Println("Submitting order to Payment Gateway (intercepted by Gobo/Ollama)...")
	start := time.Now()

	// Start an async "Agent" that fulfills the stuck request using the broker
	go func() {
		// Wait for the request to be parked
		time.Sleep(100 * time.Millisecond)
		pending := broker.GetPendingRequests()
		if len(pending) > 0 {
			// Simulating the LLM fulfilling the request perfectly
			responseJSON := []byte(`{"transaction_id": "abc-123-def", "status": "APPROVED"}`)
			broker.SubmitResponse(pending[0].ID, responseJSON)
		}
	}()

	// 5. Fire the request!
	paymentResult, err := service.ProcessOrder(order)
	if err != nil {
		t.Fatalf("ProcessOrder failed unexpectedly: %v", err)
	}

	log.Printf("Gobo LLM took %v to mock the payment!", time.Since(start))
	log.Printf("Generated Mock Response: %+v\n", paymentResult)

	// 6. Assert Ollama followed the struct tags
	if paymentResult.TransactionID == "" {
		t.Errorf("Expected LLM to generate a TransactionID")
	}

	if paymentResult.Status != "APPROVED" {
		t.Errorf("Expected LLM to follow the 'APPROVED' status hint, got: %s", paymentResult.Status)
	}
}
