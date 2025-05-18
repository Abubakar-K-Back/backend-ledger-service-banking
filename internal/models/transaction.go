package models

import (
	"time"
)

type TransactionType string

const (
	// Deposit represents a deposit transaction
	Deposit TransactionType = "deposit"

	// Withdrawal represents a withdrawal transaction
	Withdrawal TransactionType = "withdrawal"
)

type TransactionStatus string

const (
	// Pending indicates the transaction is in processing state.
	Pending TransactionStatus = "pending"

	// Completed indicates the transaction successfully processed
	Completed TransactionStatus = "completed"

	// Failed indicates the transaction failed to process
	Failed TransactionStatus = "failed"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID            string            `json:"id" bson:"_id"`
	AccountID     string            `json:"account_id" bson:"account_id"`
	Type          TransactionType   `json:"type" bson:"type"`
	Amount        float64           `json:"amount" bson:"amount"`
	Status        TransactionStatus `json:"status" bson:"status"`
	Reference     string            `json:"reference" bson:"reference"`
	BalanceBefore float64           `json:"balance_before,omitempty" bson:"balance_before,omitempty"`
	BalanceAfter  float64           `json:"balance_after,omitempty" bson:"balance_after,omitempty"`
	CreatedAt     time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" bson:"updated_at"`
}

// represents the request to creation of a new transaction
type TransactionRequest struct {
	AccountID string          `json:"account_id" validate:"required"`
	Type      TransactionType `json:"type" validate:"required,oneof=deposit withdrawal"`
	Amount    float64         `json:"amount" validate:"required,gt=0"`
	Reference string          `json:"reference,omitempty"`
}

// represents the API response for transaction data
type TransactionResponse struct {
	ID            string            `json:"id"`
	AccountID     string            `json:"account_id"`
	Type          TransactionType   `json:"type"`
	Amount        float64           `json:"amount"`
	Status        TransactionStatus `json:"status"`
	BalanceBefore float64           `json:"balance_before,omitempty"`
	BalanceAfter  float64           `json:"balance_after,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}
