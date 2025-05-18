package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/abkawan/banking-ledger/internal/models"
	"github.com/abkawan/banking-ledger/internal/service"
	"github.com/gorilla/mux"
)

// Handler is for handling api requests
type Handler struct {
	accountService     *service.AccountService
	transactionService *service.TransactionService
}

func NewHandler(accountService *service.AccountService, transactionService *service.TransactionService) *Handler {
	return &Handler{
		accountService:     accountService,
		transactionService: transactionService,
	}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// for error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// account creation
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), req.InitialBalance)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := models.AccountResponse{
		ID:        account.ID,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt,
	}

	respondJSON(w, http.StatusCreated, response)
}

// handles account retrieval
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	account, err := h.accountService.GetAccount(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	response := models.AccountResponse{
		ID:        account.ID,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// handles transaction creation
func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validation for account existance.
	_, err := h.accountService.GetAccount(r.Context(), req.AccountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	tx, err := h.transactionService.CreateTransaction(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := models.TransactionResponse{
		ID:        tx.ID,
		AccountID: tx.AccountID,
		Type:      tx.Type,
		Amount:    tx.Amount,
		Status:    tx.Status,
		CreatedAt: tx.CreatedAt,
	}

	respondJSON(w, http.StatusCreated, response)
}

// GetTransaction handles transaction retrieval
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tx, err := h.transactionService.GetTransaction(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	response := models.TransactionResponse{
		ID:            tx.ID,
		AccountID:     tx.AccountID,
		Type:          tx.Type,
		Amount:        tx.Amount,
		Status:        tx.Status,
		BalanceBefore: tx.BalanceBefore,
		BalanceAfter:  tx.BalanceAfter,
		CreatedAt:     tx.CreatedAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// GetTransactions handles transaction list retrieval
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["accountId"]

	// Parsing the query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// default limit is set to 10
	limit := 10
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	//default offset is set to 0
	offset := 0
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	txs, err := h.transactionService.GetTransactionsByAccountID(r.Context(), accountID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to response objects
	response := make([]models.TransactionResponse, 0, len(txs))
	for _, tx := range txs {
		response = append(response, models.TransactionResponse{
			ID:            tx.ID,
			AccountID:     tx.AccountID,
			Type:          tx.Type,
			Amount:        tx.Amount,
			Status:        tx.Status,
			BalanceBefore: tx.BalanceBefore,
			BalanceAfter:  tx.BalanceAfter,
			CreatedAt:     tx.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, response)
}

// handles health check
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// sets up the API routes
func SetupRoutes(r *mux.Router, accountService *service.AccountService, transactionService *service.TransactionService) {
	h := NewHandler(accountService, transactionService)

	// Health check (check if API is working)
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")

	// Account routes
	r.HandleFunc("/accounts", h.CreateAccount).Methods("POST")
	r.HandleFunc("/accounts/{id}", h.GetAccount).Methods("GET")

	// Transaction routes
	r.HandleFunc("/transactions", h.CreateTransaction).Methods("POST")
	r.HandleFunc("/transactions/{id}", h.GetTransaction).Methods("GET")
	r.HandleFunc("/accounts/{accountId}/transactions", h.GetTransactions).Methods("GET")
}
