package service

import (
	"context"
	"fmt"

	"github.com/abkawan/banking-ledger/internal/db"
	"github.com/abkawan/banking-ledger/internal/models"
)

// handles account operations
type AccountService struct {
	postgres *db.Postgres
}

// creates a new Account Service
func NewAccountService(postgres *db.Postgres) *AccountService {
	return &AccountService{
		postgres: postgres,
	}
}

// creates a new account
func (s *AccountService) CreateAccount(ctx context.Context, initialBalance float64) (*models.Account, error) {
	// Validate initial balance
	if initialBalance < 0 {
		return nil, fmt.Errorf("initial balance cannot be negative")
	}

	// Create account
	account, err := s.postgres.CreateAccount(ctx, initialBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// retrieves an account by ID
func (s *AccountService) GetAccount(ctx context.Context, id string) (*models.Account, error) {
	account, err := s.postgres.GetAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}
