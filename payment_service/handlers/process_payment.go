package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
)

type PaymentResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Payment *data.Payment `json:"payment,omitempty"`
}

func (p *Payments) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle POST Payment - Process Payment")

	w.Header().Set("Content-Type", "application/json")

	payment := r.Context().Value(KeyPayment{}).(data.Payment)

	err := data.CreatePayment(&payment)

	if err == data.ErrDuplicateIdempotencyKey {
		p.l.Printf("Duplicate idempotency key detected: %s, returning existing payment\n",
			payment.IdempotencyKey)

		response := PaymentResponse{
			Success: true,
			Message: "Payment already processed (idempotent request)",
			Payment: &payment,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		p.l.Println("[ERROR] creating payment:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Failed to create payment",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	err = data.ProcessPayment(payment.ID)
	if err != nil {
		p.l.Println("[ERROR] processing payment:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Failed to process payment",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	processedPayment, err := data.GetPaymentByID(payment.ID)
	if err != nil {
		p.l.Println("[ERROR] retrieving processed payment:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Payment processed but failed to retrieve details",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if processedPayment.Status == data.PaymentStatusCompleted {
		response := PaymentResponse{
			Success: true,
			Message: "Payment processed successfully",
			Payment: processedPayment,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		response := PaymentResponse{
			Success: false,
			Message: fmt.Sprintf("Payment failed: %s", processedPayment.ErrorMessage),
			Payment: processedPayment,
		}
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(response)
	}

	p.l.Printf("Payment %s processed with status: %s\n",
		processedPayment.ID, processedPayment.Status)
}
