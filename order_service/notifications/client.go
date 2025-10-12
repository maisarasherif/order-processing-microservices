package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SendReceiptRequest is the simplified request to notification service
// Notification service will fetch order and payment details itself
type SendReceiptRequest struct {
	OrderID       string `json:"order_id"`
	CustomerEmail string `json:"customer_email"`
}

// NotificationResponse represents the response from notification service
type NotificationResponse struct {
	Data struct {
		NotificationID string `json:"notification_id"`
		OrderID        string `json:"order_id"`
		Status         string `json:"status"`
		SentAt         string `json:"sent_at"`
	} `json:"data"`
}

// Client handles communication with the notification service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new notification service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Notifications shouldn't block orders
		},
	}
}

// SendReceipt sends a receipt email via notification service
// The notification service fetches order and payment details itself
func (c *Client) SendReceipt(orderID, customerEmail string) error {
	req := SendReceiptRequest{
		OrderID:       orderID,
		CustomerEmail: customerEmail,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := c.baseURL + "/send-receipt"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
