# AP2 Assignment 2: gRPC Migration & Contract-First Development

This repository contains the migration of the Assignment 1 microservices from REST-based internal communication to gRPC.

## Repositories

- Proto repository: `https://github.com/capydoshik/ap2-proto-repository`
- Generated code repository: `https://github.com/capydoshik/ap2-generated-contracts`

The local folders that mirror this flow are:

- `proto-repository` for `.proto` contracts
- `generated-contracts` for generated `.pb.go` files and the GitHub Actions workflow template

## What Changed

- `order-service` still exposes REST endpoints for external clients via Gin.
- `order-service` now calls `payment-service` through gRPC instead of HTTP.
- `payment-service` now exposes a gRPC server with a logging interceptor.
- `order-service` now exposes a gRPC streaming endpoint:
  `SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate)`.
- Order updates are tied to real database state by polling the `orders` table and pushing changes when `status` or `updated_at` changes.
- Bonus requirement is covered by the payment interceptor that logs method name, duration, and resulting gRPC code.

## Environment Variables

Copy `.env.example` into `.env` inside each service directory and adjust values if needed.

### `order-service/.env`

```env
ORDER_DB_URL=postgres://postgres:1234@localhost:5432/order_db?sslmode=disable
ORDER_HTTP_ADDR=:8080
ORDER_GRPC_ADDR=:9090
PAYMENT_GRPC_ADDR=localhost:9091
```

### `payment-service/.env`

```env
PAYMENT_DB_URL=postgres://postgres:1234@localhost:5432/payment_db?sslmode=disable
PAYMENT_GRPC_ADDR=:9091
```

## Database Migrations

Apply the SQL migrations in order for each service:

- `order-service/migrations/001_create_orders.sql`
- `order-service/migrations/002_add_order_updated_at.sql`
- `payment-service/migrations/001_create_payments.sql`
- `payment-service/migrations/002_add_payment_processed_at.sql`

## Run

1. Start PostgreSQL and create `order_db` and `payment_db`.
2. Apply migrations.
3. Start `payment-service`:

```powershell
cd payment-service
go run ./cmd/payment-service
```

4. Start `order-service`:

```powershell
cd order-service
go run ./cmd/order-service
```

## REST and gRPC Usage

Create an order through REST:

```powershell
curl -X POST http://localhost:8080/orders `
  -H "Content-Type: application/json" `
  -d "{\"customer_id\":\"c1\",\"item_name\":\"Keyboard\",\"amount\":5000}"
```

Subscribe to order updates through gRPC:

```powershell
cd order-service
go run ./cmd/order-stream-client localhost:9090 <order-id>
```

Update the order status and observe the stream:

```powershell
curl -X PATCH http://localhost:8080/orders/<order-id>/status `
  -H "Content-Type: application/json" `
  -d "{\"status\":\"Shipped\"}"
```

## Contract-First Workflow

1. Keep only `.proto` files in the proto repository.
2. Keep generated `.pb.go` files in the generated repository.
3. Update the placeholders in `generated-contracts/.github/workflows/generate.yml`.
4. Push both repositories to GitHub.
5. Run the workflow in the generated repository.
6. Create a tag such as `v1.0.0` in the generated repository.
7. Replace the local `replace` directives with the real GitHub module import when publishing.

## Architecture and Evidence

- Architecture diagram: [docs/architecture.md](./docs/architecture.md)
- Evidence checklist: [docs/evidences.md](./docs/evidences.md)
- Assignment checklist: [docs/assignment-checklist.md](./docs/assignment-checklist.md)
