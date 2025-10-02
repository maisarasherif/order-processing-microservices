package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
)

// GetPayment handles GET /payments/{id} - retrieves a specific payment
// This is used by:
// - Order Service to check payment status
// - Admin dashboard to view payment details
// - Customer to check their payment
func (p *Payments) GetPayment(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payment by ID")

	// Set response content type
	w.Header().Set("Content-Type", "application/json")

	// Extract payment ID from URL parameters
	// URL format: /payments/pay_1
	vars := mux.Vars(r)
	paymentID := vars["id"]

	// Retrieve payment from storage
	payment, err := data.GetPaymentByID(paymentID)

	if err == data.ErrPaymentNotFound {
		response := PaymentResponse{
			Success: false,
			Message: "Payment not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		p.l.Println("[ERROR] retrieving payment:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Error retrieving payment",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return the payment
	response := PaymentResponse{
		Success: true,
		Message: "Payment retrieved successfully",
		Payment: payment,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
