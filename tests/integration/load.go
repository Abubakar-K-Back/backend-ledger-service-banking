package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL         = "http://localhost:8080"
	numAccounts     = 100        // Number of accounts to create
	numTransactions = 10000      // Total number of transactions
	maxConcurrency  = 200        // Maximum number of concurrent requests
	initialBalance  = 10000.0    // Initial balance for each account
	maxAmount       = 1000.0     // Maximum transaction amount
	successColor    = "\033[32m" // Green
	errorColor      = "\033[31m" // Red
	infoColor       = "\033[34m" // Blue
	resetColor      = "\033[0m"  // Reset color
)

type Account struct {
	ID        string    `json:"id"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Type      string    `json:"type"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Printf("%sstarting a heavy load test with %d accounts and %d transactions%s\n",
		infoColor, numAccounts, numTransactions, resetColor)

	// Create accounts
	accounts := createAccounts(numAccounts)
	fmt.Printf("%sCreated %d accounts%s\n", successColor, len(accounts), resetColor)

	// Create semaphore for limiting concurrency
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	// Track performance
	startTime := time.Now()
	successCount := 0
	errorCount := 0
	var successMutex sync.Mutex

	// Launch transactions
	fmt.Printf("% launching %d transactions with max concurrency of %d%s\n",
		infoColor, numTransactions, maxConcurrency, resetColor)

	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(txNum int) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Randomly select an account
			account := accounts[rand.Intn(len(accounts))]

			// Randomly decide between deposit and withdrawal
			txType := "deposit"
			if rand.Intn(2) == 1 {
				txType = "withdrawal"
			}

			// Random amount between 1 and maxAmount
			amount := 1.0 + rand.Float64()*(maxAmount-1.0)
			amount = float64(int(amount*100)) / 100 // Round to 2 decimal places

			// Create transaction
			txID, err := createTransaction(account.ID, txType, amount)

			successMutex.Lock()
			if err != nil {
				errorCount++
				if txNum%100 == 0 { // Only log some failures to avoid overwhelming output
					fmt.Printf("%sTransaction failed: %v%s\n", errorColor, err, resetColor)
				}
			} else {
				successCount++
				if txNum%500 == 0 { // Log every 500th successful transaction
					fmt.Printf("%sTransaction %d: Created %s of %.2f on account %s (txID: %s)%s\n",
						successColor, txNum, txType, amount, account.ID, txID, resetColor)
				}
			}
			successMutex.Unlock()
		}(i)
	}

	// Wait for all transactions to complete
	wg.Wait()
	duration := time.Since(startTime)

	fmt.Printf("\n%s=== heavy load Test Results ===%s\n", infoColor, resetColor)
	fmt.Printf("Total number of transactions: %d\n", numTransactions)
	fmt.Printf("Successful: %s%d (%.1f%%)%s\n",
		successColor, successCount, float64(successCount)/float64(numTransactions)*100, resetColor)
	fmt.Printf("Failed: %s%d (%.1f%%)%s\n",
		errorColor, errorCount, float64(errorCount)/float64(numTransactions)*100, resetColor)
	fmt.Printf("Duration: %.2f seconds\n", duration.Seconds())
	fmt.Printf("Throughput: %.2f transactions/second\n", float64(numTransactions)/duration.Seconds())

	// Check final balances
	fmt.Printf("\n%sChecking final account balances...%s\n", infoColor, resetColor)
	checkAccountsAndTransactions(accounts)
}

// createAccounts creates the specified number of accounts
func createAccounts(count int) []Account {
	accounts := make([]Account, 0, count)

	for i := 0; i < count; i++ {
		// Create account request
		reqBody := map[string]float64{"initial_balance": initialBalance}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("%sFailed to marshal JSON: %v%s\n", errorColor, err, resetColor)
			continue
		}

		// Send request
		resp, err := http.Post(baseURL+"/accounts", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("%sFailed to create account: %v%s\n", errorColor, err, resetColor)
			continue
		}

		// Parse response
		if resp.StatusCode != http.StatusCreated {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("%sFailed to create account, status: %d, body: %s%s\n",
				errorColor, resp.StatusCode, string(body), resetColor)
			resp.Body.Close()
			continue
		}

		var account Account
		if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
			fmt.Printf("%sFailed to decode response: %v%s\n", errorColor, err, resetColor)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		accounts = append(accounts, account)
		if i%10 == 0 || i == count-1 {
			fmt.Printf("%screated account %d/%d: %s with balance %.2f%s\n",
				successColor, i+1, count, account.ID, account.Balance, resetColor)
		}
	}

	return accounts
}

// createTransaction creates a transaction for the specified account
func createTransaction(accountID, txType string, amount float64) (string, error) {
	// Create transaction request
	reqBody := map[string]interface{}{
		"account_id": accountID,
		"type":       txType,
		"amount":     amount,
		"reference":  fmt.Sprintf("tx-%s-%d", txType, rand.Int()), // Use unique reference IDs
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Send request
	resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("Failed to create transaction: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create transaction, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var transaction Transaction
	if err := json.NewDecoder(resp.Body).Decode(&transaction); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	return transaction.ID, nil
}

// getAccount retrieves account information
func getAccount(accountID string) (*Account, error) {
	// Send request
	resp, err := http.Get(fmt.Sprintf("%s/accounts/%s", baseURL, accountID))
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get account, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var account Account
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &account, nil
}

// getTransactions retrieves transaction history for an account
func getTransactions(accountID string) ([]Transaction, error) {
	// Send request
	resp, err := http.Get(fmt.Sprintf("%s/accounts/%s/transactions", baseURL, accountID))
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get transactions, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var transactions []Transaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return transactions, nil
}

// checkAccountsAndTransactions checks the final state of accounts and their transactions
func checkAccountsAndTransactions(accounts []Account) {
	sampleSize := min(10, len(accounts)) // Check up to 10 accounts
	sampledAccounts := make([]Account, sampleSize)

	// Sample accounts randomly
	for i := 0; i < sampleSize; i++ {
		sampledAccounts[i] = accounts[rand.Intn(len(accounts))]
	}

	for i, originalAccount := range sampledAccounts {
		// Get current account state
		account, err := getAccount(originalAccount.ID)
		if err != nil {
			fmt.Printf("%sError retrieving account %s: %v%s\n",
				errorColor, originalAccount.ID, err, resetColor)
			continue
		}

		// Get transactions
		transactions, err := getTransactions(account.ID)
		if err != nil {
			fmt.Printf("%sError retrieving transactions for account %s: %v%s\n",
				errorColor, account.ID, err, resetColor)
			continue
		}

		// Count deposits and withdrawals
		depositCount := 0
		withdrawalCount := 0
		pendingCount := 0
		completedCount := 0
		failedCount := 0

		for _, tx := range transactions {
			if tx.Type == "deposit" {
				depositCount++
			} else if tx.Type == "withdrawal" {
				withdrawalCount++
			}

			if tx.Status == "pending" {
				pendingCount++
			} else if tx.Status == "completed" {
				completedCount++
			} else if tx.Status == "failed" {
				failedCount++
			}
		}

		// Print account summary
		fmt.Printf("%sAccount %d: %s%s\n", infoColor, i+1, account.ID, resetColor)
		fmt.Printf("  Original balance: %.2f, Current balance: %.2f\n",
			originalAccount.Balance, account.Balance)
		fmt.Printf("  Transactions: %d total (%d deposits, %d withdrawals)\n",
			len(transactions), depositCount, withdrawalCount)
		fmt.Printf("  Status: %d completed, %d pending, %d failed\n",
			completedCount, pendingCount, failedCount)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
