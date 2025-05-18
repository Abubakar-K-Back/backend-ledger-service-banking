package models

import (
	"time"
)

type Account struct {
	ID        string    `json:"id" db:"id"`
	Balance   float64   `json:"balance" db:"balance"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateAccountRequest struct {
	InitialBalance float64 `json:"initial_balance" validate:"min=0"`
}

type AccountResponse struct {
	ID        string    `json:"id"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}
