package service

import (
	"context"
	"fmt"
	"log"

	"github.com/abkawan/banking-ledger/internal/db"
	"github.com/abkawan/banking-ledger/internal/models"
	"github.com/abkawan/banking-ledger/internal/queue"
	"github.com/google/uuid"
)

// handles transaction operations
type TransactionService struct {
	postgres *db.Postgres
	mongodb  *db.MongoDB
	rabbitmq *queue.RabbitMQ
}

// creates a new TransactionService
func NewTransactionService(postgres *db.Postgres, mongodb *db.MongoDB, rabbitmq *queue.RabbitMQ) *TransactionService {
	return &TransactionService{
		postgres: postgres,
		mongodb:  mongodb,
		rabbitmq: rabbitmq,
	}
}

// creates a new transaction
func (s *TransactionService) CreateTransaction(ctx context.Context, req *models.TransactionRequest) (*models.Transaction, error) {
	// Use provided reference or generate a new one
	reference := req.Reference
	if reference == "" {
		reference = uuid.New().String()
	}

	// Check for existing transaction with same reference (idempotency)
	existingTx, err := s.mongodb.GetTransactionByReference(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("Failed to check for existing transaction: %w", err)
	}

	// If transaction already exists, return it
	if existingTx != nil {
		return existingTx, nil
	}

	// Create new transaction
	tx := &models.Transaction{
		AccountID: req.AccountID,
		Type:      req.Type,
		Amount:    req.Amount,
		Status:    models.Pending,
		Reference: reference,
	}

	// saving transaction to MongoDB
	if err := s.mongodb.CreateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("Failed to create transaction: %w", err)
	}

	// sending transaction to RabbitMQ
	if err := s.rabbitmq.PublishTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to queue transaction: %w", err)
	}

	return tx, nil
}

// GetTransaction retrieves a transaction by ID
func (s *TransactionService) GetTransaction(ctx context.Context, id string) (*models.Transaction, error) {
	tx, err := s.mongodb.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, nil
}

// retrieves transactions for an account
func (s *TransactionService) GetTransactionsByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*models.Transaction, error) {
	txs, err := s.mongodb.GetTransactionsByAccountID(ctx, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return txs, nil
}

// processes a transaction
func (s *TransactionService) ProcessTransaction(ctx context.Context, tx *models.Transaction) error {
	// Validate account exists
	_, err := s.postgres.GetAccount(ctx, tx.AccountID)
	if err != nil {
		return s.markTransactionFailed(ctx, tx.ID, fmt.Errorf("account not found: %w", err))
	}

	// check amount (positive for deposit, negative for withdrawal)
	amount := tx.Amount
	if tx.Type == models.Withdrawal {
		amount = -amount
	}

	balanceBefore, balanceAfter, err := s.postgres.UpdateAccountBalance(ctx, tx.AccountID, amount)
	if err != nil {
		return s.markTransactionFailed(ctx, tx.ID, fmt.Errorf("failed to update balance: %w", err))
	}

	if err := s.mongodb.UpdateTransactionStatus(ctx, tx.ID, models.Completed, balanceBefore, balanceAfter); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

func (s *TransactionService) markTransactionFailed(ctx context.Context, id string, err error) error {
	if updateErr := s.mongodb.UpdateTransactionStatus(ctx, id, models.Failed, 0, 0); updateErr != nil {
		log.Printf("Failed to mark transaction %s as failed: %v", id, updateErr)
	}
	return err
}

// starts a transaction processor
func (s *TransactionService) StartProcessor(ctx context.Context) error {
	txChan, err := s.rabbitmq.ConsumeTransactions(ctx)
	if err != nil {
		return fmt.Errorf("failed to consume transactions: %w", err)
	}

	// proccessing transactions in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case tx, ok := <-txChan:
				if !ok {
					return
				}

				// Process the transaction
				if err := s.ProcessTransaction(ctx, &tx); err != nil {
					log.Printf("Failed to process transaction %s: %v", tx.ID, err)
				} else {
					log.Printf("Successfully processed transaction %s", tx.ID)
				}
			}
		}
	}()

	return nil
}
