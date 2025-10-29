package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/payment_service/config"
	"github.com/maisarasherif/order-processing-microservices/payment_service/handlers"
	"github.com/maisarasherif/order-processing-microservices/payment_service/logger"
	"github.com/maisarasherif/order-processing-microservices/payment_service/repository"
)

func main() {

	l := logger.NewLogger("payment-service")

	l.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		l.Fatal("Failed to load configuration:", err)
	}

	l.Println("Connecting to PostgreSQL...")
	repo, err := repository.NewPostgresRepository(cfg.GetDatabaseConnectionString())
	if err != nil {
		l.Fatal("Failed to connect to database:", err)
	}
	defer repo.Close()
	l.Println("✓ Database connection established")

	ph := handlers.NewPaymentsWithRepository(l, repo)

	sm := mux.NewRouter()

	sm.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"payment-service","database":"connected"}`))
	}).Methods(http.MethodGet)

	sm.Handle("/metrics", handlers.MetricsHandler()).Methods(http.MethodGet)

	sm.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	sm.Use(handlers.MetricsMiddleware)

	sm.HandleFunc("/payments/stats", ph.GetStatistics).Methods(http.MethodGet)

	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/payments", ph.GetPayments)
	getRouter.HandleFunc("/payments/{id}", ph.GetPayment)

	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/payments", ph.ProcessPayment)
	postRouter.Use(ph.MiddlewarePaymentValidation)

	s := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      sm,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		l.Println("===========================================")
		l.Println("💳 Payment Service Starting...")
		l.Println("===========================================")
		l.Printf("Port: %s\n", cfg.Server.Port)
		l.Printf("Database: %s@%s:%d/%s\n",
			cfg.Database.User,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName)
		l.Println("-------------------------------------------")
		l.Printf("Health Check:    http://localhost:%s/health\n", cfg.Server.Port)
		l.Printf("Process Payment: POST http://localhost:%s/payments\n", cfg.Server.Port)
		l.Printf("List Payments:   GET http://localhost:%s/payments\n", cfg.Server.Port)
		l.Printf("Get Payment:     GET http://localhost:%s/payments/{id}\n", cfg.Server.Port)
		l.Printf("Statistics:      GET http://localhost:%s/payments/stats\n", cfg.Server.Port)
		l.Println("===========================================")

		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			l.Fatal(err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	l.Println("Received terminate signal:", sig)
	l.Println("Initiating graceful shutdown...")

	// Shutdown with timeout
	tc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := repo.Close(); err != nil {
		l.Printf("Error closing database: %v\n", err)
	} else {
		l.Println("✓ Database connection closed")
	}

	if err := s.Shutdown(tc); err != nil {
		l.Printf("Error during shutdown: %v\n", err)
	} else {
		l.Println("✓ Payment Service shutdown complete")
	}
}
