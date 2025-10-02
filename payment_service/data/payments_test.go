package data

import "testing"

func TestPaymentValidation(t *testing.T) {
	// Valid payment
	p := &Payment{
		OrderID:        "order_123",
		Amount:         120,
		Currency:       "EGP",
		Method:         "credit_card",
		CustomerID:     "cust_456",
		IdempotencyKey: "idempotency_test_1",
	}

	err := p.Validate()
	if err != nil {
		t.Fatalf("Valid payment failed validation: %v", err)
	}
}

func TestInvalidCurrency(t *testing.T) {
	p := &Payment{
		OrderID:        "order_123",
		Amount:         120,
		Currency:       "XYZ", // Invalid currency
		Method:         "credit_card",
		CustomerID:     "cust_456",
		IdempotencyKey: "idempotency_test_2",
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("Expected validation error for invalid currency, got none")
	}
}

func TestInvalidAmount(t *testing.T) {
	p := &Payment{
		OrderID:        "order_123",
		Amount:         0, // Invalid amount
		Currency:       "USD",
		Method:         "credit_card",
		CustomerID:     "cust_456",
		IdempotencyKey: "idempotency_test_3",
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("Expected validation error for zero amount, got none")
	}
}

func TestCreatePayment(t *testing.T) {
	p := &Payment{
		OrderID:        "order_789",
		Amount:         49.99,
		Currency:       "EUR",
		Method:         "paypal",
		CustomerID:     "cust_999",
		IdempotencyKey: "idempotency_test_4",
	}

	err := CreatePayment(p)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	if p.ID == "" {
		t.Fatal("Payment ID was not generated")
	}

	if p.Status != PaymentStatusPending {
		t.Fatalf("Expected status %s, got %s", PaymentStatusPending, p.Status)
	}
}

func TestIdempotency(t *testing.T) {
	key := "idempotency_test_5"

	// First payment
	p1 := &Payment{
		OrderID:        "order_100",
		Amount:         25.00,
		Currency:       "USD",
		Method:         "credit_card",
		CustomerID:     "cust_100",
		IdempotencyKey: key,
	}

	err := CreatePayment(p1)
	if err != nil {
		t.Fatalf("Failed to create first payment: %v", err)
	}

	// Second payment with same idempotency key
	p2 := &Payment{
		OrderID:        "order_101",
		Amount:         30.00,
		Currency:       "USD",
		Method:         "credit_card",
		CustomerID:     "cust_101",
		IdempotencyKey: key, // Same key!
	}

	err = CreatePayment(p2)
	if err != ErrDuplicateIdempotencyKey {
		t.Fatalf("Expected ErrDuplicateIdempotencyKey, got: %v", err)
	}

	// p2 should now have the same data as p1
	if p2.ID != p1.ID {
		t.Fatalf("Expected same payment ID, got p1=%s, p2=%s", p1.ID, p2.ID)
	}
}

func TestProcessPayment(t *testing.T) {

	p := &Payment{
		OrderID:        "order_200",
		Amount:         75.00,
		Currency:       "AED",
		Method:         "debit_card",
		CustomerID:     "cust_200",
		IdempotencyKey: "idempotency_test_6",
	}

	err := CreatePayment(p)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	err = ProcessPayment(p.ID)
	if err != nil {
		t.Fatalf("Failed to process payment: %v", err)
	}

	processed, err := GetPaymentByID(p.ID)
	if err != nil {
		t.Fatalf("Failed to get payment: %v", err)
	}

	if processed.Status != PaymentStatusCompleted {
		t.Fatalf("Expected status %s, got %s", PaymentStatusCompleted, processed.Status)
	}

	if processed.TransactionID == "" {
		t.Fatal("Transaction ID was not set")
	}

	if processed.ProcessedAt == nil {
		t.Fatal("ProcessedAt timestamp was not set")
	}
}

func TestProcessPaymentFailure(t *testing.T) {

	p := &Payment{
		OrderID:        "order_300",
		Amount:         666,
		Currency:       "USD",
		Method:         "credit_card",
		CustomerID:     "cust_300",
		IdempotencyKey: "idempotency_test_7",
	}

	err := CreatePayment(p)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	err = ProcessPayment(p.ID)
	if err != nil {
		t.Fatalf("ProcessPayment returned error: %v", err)
	}

	processed, err := GetPaymentByID(p.ID)
	if err != nil {
		t.Fatalf("Failed to get payment: %v", err)
	}

	if processed.Status != PaymentStatusFailed {
		t.Fatalf("Expected status %s, got %s", PaymentStatusFailed, processed.Status)
	}

	if processed.ErrorMessage == "" {
		t.Fatal("Error message was not set for failed payment")
	}
}
