package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/payment_service/handlers"
)

func main() {

	l := log.New(os.Stdout, "payment-service: ", log.LstdFlags)

	ph := handlers.NewPayments(l)

	sm := mux.NewRouter()

	sm.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"payment-service"}`))
	}).Methods(http.MethodGet)

	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/payments", ph.GetPayments)
	getRouter.HandleFunc("/payments/{id}", ph.GetPayment)

	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/payments", ph.ProcessPayment)
	postRouter.Use(ph.MiddlewarePaymentValidation)

	s := &http.Server{
		Addr:         ":8082",
		Handler:      sm,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		l.Println("===========================================")
		l.Println("Payment Service Starting...")
		l.Println("===========================================")
		l.Println("Port: 8082")
		l.Println("Health Check: http://localhost:8082/health")
		l.Println("Process Payment: POST http://localhost:8082/payments")
		l.Println("List Payments: GET http://localhost:8082/payments")
		l.Println("Get Payment: GET http://localhost:8082/payments/{id}")
		l.Println("===========================================")

		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			l.Fatal(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	l.Println("Received terminate signal:", sig)
	l.Println("Initiating graceful shutdown...")

	tc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(tc); err != nil {
		l.Printf("Error during shutdown: %v\n", err)
	} else {
		l.Println("Payment Service shutdown complete")
	}
}
