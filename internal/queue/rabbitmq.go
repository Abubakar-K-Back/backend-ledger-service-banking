package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abkawan/banking-ledger/internal/models"
	"github.com/streadway/amqp"
)

const (
	// queue for transactions
	TransactionQueue = "transactions"
)

// handles RabbitMQ operations
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}
	q, err := ch.QueueDeclare(
		TransactionQueue, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return &RabbitMQ{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (r *RabbitMQ) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}

// publishes a payment/transaction to the queue
func (r *RabbitMQ) PublishTransaction(ctx context.Context, tx *models.Transaction) error {
	body, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Publish a message
	err = r.channel.Publish(
		"",               // exchange
		TransactionQueue, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // make message persistent
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	return nil
}

// consumes transactions from the queue
func (r *RabbitMQ) ConsumeTransactions(ctx context.Context) (<-chan models.Transaction, error) {
	msgs, err := r.channel.Consume(
		TransactionQueue, // queue
		"",               // consumer
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	// Create a channel for transactions
	txChan := make(chan models.Transaction)

	// Process messages in a goroutine
	go func() {
		defer close(txChan)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				var tx models.Transaction
				if err := json.Unmarshal(msg.Body, &tx); err != nil {
					// Log error and continue
					fmt.Printf("failed to unmarshal transaction: %v\n", err)
					msg.Reject(false) // Don't requeue
					continue
				}

				// Send to transaction channel
				txChan <- tx

				// Acknowledge message
				msg.Ack(false)
			}
		}
	}()

	return txChan, nil
}
