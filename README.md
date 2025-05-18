# Banking Ledger Service

A reliable banking ledger service built with Go, designed to handle account operations even under high load.

## Features

- Create accounts with initial balances
- Process deposits and withdrawals of money
- Maintain a detailed transaction history of accounts
- Ensure transaction consistency and durability
- Scale horizontally for high volume

## Architecture

This service uses a multi-tier architecture:

- **API Server**: Exposes RESTful endpoints for account and transaction operations
- **Transaction Processor**: Processes transactions asynchronously
- **PostgreSQL**: Stores account balances with ACID transactions
- **MongoDB**: Stores transaction logs for efficient querying
- **RabbitMQ**: Provides reliable message delivery for transaction processing

## Getting Started

### Prerequisites

- Docker and Docker Compose

### Running the Service

1. Clone the repository
2. Run with Docker Compose:
   ```
   docker-compose up -d
   ```
3. all passwords are right now default so no need to set for testing (please check docker-compose.yml file).
## API Endpoints

### Accounts

- **Create Account**:
  ```
  POST /accounts
  { "initial_balance": 1000.00 }
  ```

- **Get Account by ID**:
  ```
  GET /accounts/{id}
  ```

### Transactions

- **Creating Transaction**:
  ```
  POST /transactions
  {
    "account_id": "account-id",
    "type": "deposit", // or "withdrawal"
    "amount": 100.00,
    "reference": "optional-reference-id"
  }
  ```

- **Get Transaction**:
  ```
  GET /transactions/{id}
  ```

- **List Account Transactions**:
  ```
  GET /accounts/{accountId}/transactions?limit=10&offset=0
  ```

## Test Requirements and fulfillments:

̈1. Support the creation of accounts with specified initial balances.
2. Facilitate deposits and withdrawals of funds 
3. Maintain a detailed transaction log (ledger) for each account¥
4. Ensure ACID-like consistency for core operations to prevent double spending or inconsistent balances¥
5. Scale horizontally to handle high spikes in transaction volume¥
6. Integrate an asynchronous queue or broker to manage transaction requests efficiently¥
7. Include a comprehensive testing strategy, covering feature tests and mocking for robust validation.


- **Transaction Processing**: Asynchronous via RabbitMQ to handle high load
- **Data Storage**: 
  - PostgreSQL for account balances (ACID compliant)
  - MongoDB for transaction history (efficient querying)
- **Consistency**: Optimistic locking for account balance updates
- **Idempotency**: Transaction reference IDs to prevent double processing
- **Error Handling**: Comprehensive error handling with appropriate HTTP status codes

## Implementation Details

### ACID-like Consistency (as required in doc)

While PostgreSQL provides ACID guarantees for individual operations, we extend this to our distributed system:

1. **Atomicity**: We use a two-phase approach:
   - First record the transaction intent
   - Then process it asynchronously

2. **Consistency**: We validate operations before processing:
   - Check account existence
   - Prevent negative balances
   - Maintain consistent transaction logs

3. **Isolation**: We use row-level locking in PostgreSQL to prevent race conditions

4. **Durability**: Both PostgreSQL and MongoDB provide durability guarantees

### Horizontal Scaling

The service is designed to scale horizontally:

- Stateless API servers can be scaled out
- Multiple transaction processors can consume from the same queue
- Database replication/sharding is supported

## Project Structure

```
banking-ledger/
├── cmd/
│   ├── api/            # API server entry point
│   └── processor/      # Transaction processor entry point and initializer
├── internal/
│   ├── api/            # API handlers
│   ├── db/             # Database operations
│   ├── models/         # Data models
│   ├── queue/          # Rabbit Message queue operations
│   └── service/        # Business logic
├── docker/             # Dockerfiles
├── docker-compose.yml  # Service configuration
└── README.md           # This file Readme
```

## Testing
I have created a single file where we are testing the functions and load on system.

The service includes comprehensive testing at multiple levels:

- Unit tests for core business logic
- Integration tests for service interactions
- End-to-end tests for complete flows

Run load test with:
```
go run ./tests/integration/load.go
```

## Future Improvements

- Add authentication and authorization
- Implement transaction retries with exponential backoff
- Add detailed monitoring and metrics for whole system
- Implement database sharding for very high scale
- Add comprehensive API documentation (Swagger)

