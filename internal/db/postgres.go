package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/abkawan/banking-ledger/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Postgres.go handles PostgreSQL database operations
type Postgres struct {
	db *sql.DB
}

// creates a new Postgres instance
func NewPostgres(connStr string) (*Postgres, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}
	return &Postgres{db: db}, nil
}

// closes the database connection
func (p *Postgres) Close() error {
	return p.db.Close()
}

// initialize the database schema
func (p *Postgres) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS accounts (
		id VARCHAR(36) PRIMARY KEY,
		balance DECIMAL(20, 2) NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`

	_, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create accounts table: %w", err)
	}
	return nil
}

// creates a new account
func (p *Postgres) CreateAccount(ctx context.Context, initialBalance float64) (*models.Account, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
	INSERT INTO accounts (id, balance, created_at, updated_at)
	VALUES ($1, $2, $3, $4)
	RETURNING id, balance, created_at, updated_at`

	var account models.Account
	err := p.db.QueryRowContext(
		ctx, query, id, initialBalance, now, now,
	).Scan(&account.ID, &account.Balance, &account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &account, nil
}

// retrieves an account by ID
func (p *Postgres) GetAccount(ctx context.Context, id string) (*models.Account, error) {
	query := `
	SELECT id, balance, created_at, updated_at
	FROM accounts
	WHERE id = $1`

	var account models.Account
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.Balance, &account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// updates the account balance
func (p *Postgres) UpdateAccountBalance(ctx context.Context, id string, amount float64) (balanceBefore, balanceAfter float64, err error) {
	// Start a transaction
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Get current balance with row lock
	var currentBalance float64
	err = tx.QueryRowContext(
		ctx,
		"SELECT balance FROM accounts WHERE id = $1 FOR UPDATE",
		id,
	).Scan(&currentBalance)

	if err != nil {
		return 0, 0, fmt.Errorf("Failed to get current balance: %w", err)
	}

	// Calculate new balance
	newBalance := currentBalance + amount

	// Check for negative balance
	if newBalance < 0 {
		return 0, 0, fmt.Errorf("insufficient funds")
	}

	// Update balance
	_, err = tx.ExecContext(
		ctx,
		"UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3",
		newBalance, time.Now(), id,
	)

	if err != nil {
		return 0, 0, fmt.Errorf("failed to update balance: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return currentBalance, newBalance, nil
}
