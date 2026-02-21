package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// OrderService creates orders and charges users via a third-party payment gateway.
type OrderService struct {
	PaymentGatewayURL string
	HTTPClient        *http.Client
}

// Order defines a customer order.
type Order struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
	Amount  int    `json:"amount"` // in cents
}

// PaymentResponse is what the external gateway returns.
type PaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

// ProcessOrder simulate placing an order and calling an external API for payment.
func (s *OrderService) ProcessOrder(order Order) (*PaymentResponse, error) {
	reqBody, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, s.PaymentGatewayURL+"/charge", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment gateway returned status: %d", resp.StatusCode)
	}

	var paymentResp PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return nil, err
	}

	return &paymentResp, nil
}
