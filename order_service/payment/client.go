package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PaymentRequest struct {
	OrderID        string  `json:"order_id"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	Method         string  `json:"method"`
	CustomerID     string  `json:"customer_id"`
	IdempotencyKey string  `json:"idempotency_key"`
}

// Updated response structure to match new payment service format
type PaymentResponse struct {
	Data    *Payment `json:"data,omitempty"`    // Success response
	Error   string   `json:"error,omitempty"`   // Error response
	Payment Payment  `json:"payment,omitempty"` // Backwards compatibility
}

type Payment struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Method        string    `json:"method"`
	CustomerID    string    `json:"customer_id"`
	TransactionID string    `json:"transaction_id"`
	ErrorMessage  string    `json:"error_message"`
	CreatedAt     time.Time `json:"created_at"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) ProcessPayment(req PaymentRequest) (*PaymentResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	url := c.baseURL + "/payments"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var paymentResp PaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Handle new response format with "data" wrapper
	if paymentResp.Data != nil {
		paymentResp.Payment = *paymentResp.Data
	}

	// Check for error responses
	if resp.StatusCode >= 400 {
		if paymentResp.Error != "" {
			return nil, fmt.Errorf("payment failed: %s", paymentResp.Error)
		}
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	return &paymentResp, nil
}

func (c *Client) GetPaymentStatus(paymentID string) (*PaymentResponse, error) {
	url := c.baseURL + "/payments/" + paymentID
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var paymentResp PaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Handle new response format
	if paymentResp.Data != nil {
		paymentResp.Payment = *paymentResp.Data
	}

	if resp.StatusCode != http.StatusOK {
		if paymentResp.Error != "" {
			return nil, fmt.Errorf("payment service error: %s", paymentResp.Error)
		}
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	return &paymentResp, nil
}
