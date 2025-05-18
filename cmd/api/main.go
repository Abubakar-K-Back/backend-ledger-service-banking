package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abkawan/banking-ledger/internal/api"
	"github.com/abkawan/banking-ledger/internal/db"
	"github.com/abkawan/banking-ledger/internal/queue"
	"github.com/abkawan/banking-ledger/internal/service"
	"github.com/gorilla/mux"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get environment variables
	postgresURI := getEnv("POSTGRES_URI", "postgres://postgres:postgres@postgres:5432/ledger?sslmode=disable")
	mongoURI := getEnv("MONGO_URI", "mongodb://mongo:27017")
	mongoDBName := getEnv("MONGO_DB_NAME", "ledger")
	rabbitmqURI := getEnv("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	port := getEnv("PORT", "8080")

	// Connecting to Postgres
	log.Println("Connecting to PostgreSQL...")
	postgres, err := db.NewPostgres(postgresURI)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer postgres.Close()

	// Create schema
	log.Println("Creating the schema...")
	if err := postgres.InitSchema(ctx); err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}

	// Connect to MongoDB
	log.Println("Connecting to MongoDB...")
	mongodb, err := db.NewMongoDB(mongoURI, mongoDBName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(ctx)

	// Connect to RabbitMQ
	log.Println("Connecting to RabbitMQ...")
	rabbitmq, err := queue.NewRabbitMQ(rabbitmqURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	// Create services
	accountService := service.NewAccountService(postgres)
	transactionService := service.NewTransactionService(postgres, mongodb, rabbitmq)

	// Start transaction processor
	log.Println("Starting transaction processor...")
	if err := transactionService.StartProcessor(ctx); err != nil {
		log.Fatalf("Failed to start transaction processor: %v", err)
	}

	// Create router and set up routes
	router := mux.NewRouter()
	api.SetupRoutes(router, accountService, transactionService)

	// Create server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server shut down successfully")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
