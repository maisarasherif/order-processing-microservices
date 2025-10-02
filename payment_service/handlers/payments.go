package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
)

type Payments struct {
	l *log.Logger
}

func NewPayments(l *log.Logger) *Payments {
	return &Payments{l}
}

func (p *Payments) GetPayments(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payments")

	payments := data.GetPayments()

	err := payments.ToJSON(w)
	if err != nil {
		p.l.Println("[ERROR] serializing payments", err)
		http.Error(w, "Unable to marshal json", http.StatusInternalServerError)
		return
	}
}

type KeyPayment struct{}

func (p *Payments) MiddlewarePaymentValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payment := data.Payment{}

		err := payment.FromJSON(r.Body)
		if err != nil {
			p.l.Println("[ERROR] deserializing payment", err)
			http.Error(w, "Error reading payment data", http.StatusBadRequest)
			return
		}

		err = payment.Validate()
		if err != nil {
			p.l.Println("[ERROR] validating payment", err)
			http.Error(
				w,
				fmt.Sprintf("Error validating payment: %s", err),
				http.StatusBadRequest,
			)
			return
		}

		ctx := context.WithValue(r.Context(), KeyPayment{}, payment)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
