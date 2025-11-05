# Microservices with SQS

This repository contains two microservices that communicate via AWS SQS:
- **product-service**: REST API for managing products
- **notification-service**: Listens to product events and logs notifications

## Architecture

- **Product Service**: Gin-based REST API with PostgreSQL backend and Prometheus metrics
- **Notification Service**: SQS consumer that processes product notifications
- **Message Broker**: AWS SQS (via LocalStack for local development)
- **Database**: PostgreSQL
- **Metrics**: Prometheus

### Outbox Pattern for Reliable Event Publishing

This application implements the **Outbox Pattern** to ensure atomic operations between database transactions and event publishing. This prevents data inconsistency that could occur if a product is created/deleted in the database but the corresponding event fails to publish to SQS.

**How it works:**

1. **Atomic Storage**: When a product is created or deleted, both the product change and the event are stored in the database within a single transaction. The event is stored in an `events` table with a `pending` status.

2. **Background Worker**: An outbox worker runs every 2 seconds, polling the `events` table for pending events. It processes these events by publishing them to SQS and then marks them as `processed` or `failed`.

3. **Reliability**: If the application crashes after committing the database transaction but before publishing to SQS, the event remains in the `pending` state and will be picked up by the worker on the next poll cycle. This guarantees at-least-once delivery of events.

**Benefits:**
- **Atomicity**: Database changes and event creation happen in the same transaction
- **Reliability**: Events are never lost even if SQS is temporarily unavailable
- **Consistency**: The system state remains consistent across database and message broker

## Prerequisites

- Go 1.25.1+
- Docker and Docker Compose

## Getting Started

1. **Copy environment file:**
   ```bash
   cp example.env .env
   ```

2. **Start infrastructure (PostgreSQL, LocalStack, Prometheus, Grafana):**
   ```bash
   docker compose up -d
   ```

3. **Run product-service:**
   ```bash
   go run cmd/product-service/main.go
   ```

4. **Run notification-service (in another terminal):**
   ```bash
   go run cmd/notification-service/main.go
   ```

## API Endpoints

### Product Service (http://localhost:8080)

#### Create Product
```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 1299.99
  }'
```

#### List Products (with pagination)
```bash
# First page
curl http://localhost:8080/products?limit=10

# Next page (use next_page_token from previous response)
curl http://localhost:8080/products?limit=10&token=<next_page_token>
```

#### Delete Product
```bash
curl -X DELETE http://localhost:8080/products/<product-id>
```

## Metrics

Prometheus metrics are available at:
```
http://localhost:8082/metrics
```

Available metrics:
- `products_created_total`: Counter for created products
- `products_deleted_total`: Counter for deleted products

## Testing

Run all tests:
```bash
go test ./internal/...
```

Run tests with coverage:
```bash
go test -coverprofile=cover.out ./internal/...
go tool cover -html=cover.out
```

## Development

### Building Services

```bash
# Build product-service
go build -o bin/product-service cmd/product-service/main.go

# Build notification-service
go build -o bin/notification-service cmd/notification-service/main.go
```

### Linting

```bash
make lint
```

## Docker Services

- PostgreSQL: `localhost:5432`
- LocalStack (SQS): `localhost:4566`
- Prometheus: `localhost:9090`
- Grafana: `localhost:3004`

## SQS Queue

The product-notifications queue is automatically created by LocalStack on startup.

Queue URL: `http://localhost:4566/000000000000/product-notifications`
