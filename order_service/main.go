package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/order_service/config"
	"github.com/maisarasherif/order-processing-microservices/order_service/handlers"
	"github.com/maisarasherif/order-processing-microservices/order_service/notifications"
	"github.com/maisarasherif/order-processing-microservices/order_service/payment"
	"github.com/maisarasherif/order-processing-microservices/order_service/repository"
)

func main() {

	l := log.New(os.Stdout, "order-service: ", log.LstdFlags)

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

	l.Println("Initializing payment service client...")
	paymentClient := payment.NewClient(cfg.Payment.URL)
	l.Printf("✓ Payment service client configured: %s\n", cfg.Payment.URL)

	notificationClient := notifications.NewClient(cfg.Notification.URL)

	oh := handlers.NewOrdersHandler(l, repo, paymentClient, notificationClient)

	sm := mux.NewRouter()

	sm.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"order-service","database":"connected"}`))
	}).Methods(http.MethodGet)

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

	//sm.HandleFunc("/products", oh.GetProducts).Methods(http.MethodGet)
	//sm.HandleFunc("/products/{id}", oh.GetProduct).Methods(http.MethodGet)

	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/orders", oh.GetOrders)
	getRouter.HandleFunc("/orders/{id}", oh.GetOrder)
	getRouter.HandleFunc("/orders/customer/{id}", oh.GetCustomerOrders)
	getRouter.HandleFunc("/products", oh.GetProducts)     // ADD
	getRouter.HandleFunc("/products/{id}", oh.GetProduct) // ADD

	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/orders", oh.CreateOrder)

	postRouter.Use(oh.MiddlewareOrderValidation)

	s := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      sm,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		l.Println("===========================================")
		l.Println("🛒 Order Service Starting...")
		l.Println("===========================================")
		l.Printf("Port: %s\n", cfg.Server.Port)
		l.Printf("Database: %s@%s:%d/%s\n",
			cfg.Database.User,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName)
		l.Printf("Payment Service: %s\n", cfg.Payment.URL)
		l.Println("-------------------------------------------")
		l.Printf("Health Check:    http://localhost:%s/health\n", cfg.Server.Port)
		l.Printf("Create Order:    POST http://localhost:%s/orders\n", cfg.Server.Port)
		l.Printf("List Orders:     GET http://localhost:%s/orders\n", cfg.Server.Port)
		l.Printf("Get Order:       GET http://localhost:%s/orders/{id}\n", cfg.Server.Port)
		l.Printf("Customer Orders: GET http://localhost:%s/orders/customer/{id}\n", cfg.Server.Port)
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

	if err := repo.Close(); err != nil {
		l.Printf("Error closing database: %v\n", err)
	} else {
		l.Println("✓ Database connection closed")
	}

	if err := s.Shutdown(tc); err != nil {
		l.Printf("Error during shutdown: %v\n", err)
	} else {
		l.Println("✓ Order Service shutdown complete")
	}
}
