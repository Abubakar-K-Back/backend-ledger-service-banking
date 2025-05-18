package db

import (
	"context"
	"fmt"
	"time"

	"github.com/abkawan/banking-ledger/internal/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// for handling MongoDB operations
type MongoDB struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// creates a new MongoDB instance
func NewMongoDB(uri, dbName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Mongodb: %w", err)
	}

	// pinging the database
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping Mongodb: %w", err)
	}

	collection := client.Database(dbName).Collection("transactions")

	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "account_id", Value: 1}},
			Options: options.Index().SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "reference", Value: 1}},
			Options: options.Index().SetUnique(true).SetBackground(true),
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoDB{
		client:     client,
		collection: collection,
	}, nil
}

// closes the mongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// creates a new transaction
func (m *MongoDB) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	if tx.ID == "" {
		tx.ID = uuid.New().String()
	}

	now := time.Now()
	tx.CreatedAt = now
	tx.UpdatedAt = now

	_, err := m.collection.InsertOne(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	return nil
}

// GetTransactionByID: retrieves the transaction by ID
func (m *MongoDB) GetTransactionByID(ctx context.Context, id string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := m.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// retrieves a transaction by reference
func (m *MongoDB) GetTransactionByReference(ctx context.Context, reference string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := m.collection.FindOne(ctx, bson.M{"reference": reference}).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found, but not an error
		}
		return nil, fmt.Errorf("failed to get transaction by reference: %w", err)
	}

	return &transaction, nil
}

// updates a transaction's status
func (m *MongoDB) UpdateTransactionStatus(ctx context.Context, id string, status models.TransactionStatus, balanceBefore, balanceAfter float64) error {
	update := bson.M{
		"$set": bson.M{
			"status":         status,
			"balance_before": balanceBefore,
			"balance_after":  balanceAfter,
			"updated_at":     time.Now(),
		},
	}

	_, err := m.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

// retrieves transactions for an account
func (m *MongoDB) GetTransactionsByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*models.Transaction, error) {
	options := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := m.collection.Find(ctx, bson.M{"account_id": accountID}, options)
	if err != nil {
		return nil, fmt.Errorf("failed to find transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	return transactions, nil
}
