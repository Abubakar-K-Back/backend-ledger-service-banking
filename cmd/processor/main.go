package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/abkawan/banking-ledger/internal/db"
	"github.com/abkawan/banking-ledger/internal/queue"
	"github.com/abkawan/banking-ledger/internal/service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	postgresURI := getEnv("POSTGRES_URI", "postgres://postgres:postgres@postgres:5432/ledger?sslmode=disable")
	mongoURI := getEnv("MONGO_URI", "mongodb://mongo:27017")
	mongoDBName := getEnv("MONGO_DB_NAME", "ledger")
	rabbitmqURI := getEnv("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	//connecting to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	postgres, err := db.NewPostgres(postgresURI)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer postgres.Close()

	// Connect to MongoDB
	log.Println("connecting to MongoDB...")
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

	// Create transaction service
	transactionService := service.NewTransactionService(postgres, mongodb, rabbitmq)

	// Start transaction processor
	log.Println("Starting transaction processor...")
	if err := transactionService.StartProcessor(ctx); err != nil {
		log.Fatalf("Failed to start transaction processor: %v", err)
	}

	log.Println("Transaction processor started")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down processor...")
	cancel() // Cancel context to stop processor
	log.Println("Processor shut down successfully")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
