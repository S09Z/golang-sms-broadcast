# golang-sms-broadcast

🚀 **Scalable SMS broadcast system** using Go microservices, PostgreSQL, RabbitMQ, and the Transactional Outbox pattern.

## Architecture

```
┌──────────────────┐
│  broadcast-api   │  ← REST API (create broadcast)
│   (Port 8080)    │
└────────┬─────────┘
         │ writes
         ▼
┌─────────────────────────┐
│  PostgreSQL Database    │
│  • broadcasts table     │
│  • messages table       │
└──────────┬──────────────┘
           │
           │ polls every 5s
           ▼
┌──────────────────────┐
│  outbox-publisher    │  ← Reads pending, publishes to queue
└─────────┬────────────┘
          │
          ▼
┌─────────────────┐
│    RabbitMQ     │  ← Message queue
└────────┬────────┘
         │ consumes
         ▼
┌──────────────────┐
│  sender-worker   │  ← Sends SMS via provider
└─────────┬────────┘
          │ HTTP POST
          ▼
┌─────────────────────┐
│ mock-sms-provider   │  ← Fake SMS gateway
│   (Port 9090)       │
└──────────┬──────────┘
           │ async webhook
           ▼
┌──────────────────┐
│   dlr-webhook    │  ← Delivery receipt handler
│   (Port 8081)    │
└──────────────────┘
```

## Features

- ✅ **Transactional Outbox** - No message loss, DB + queue consistency
- ✅ **Hexagonal Architecture** - Clean separation: domain → ports → adapters
- ✅ **5 Microservices** - API, outbox publisher, worker, webhook, mock provider
- ✅ **Status Tracking** - pending → queued → sent → delivered/failed
- ✅ **Idempotent** - Safe retries using provider message IDs
- ✅ **Observable** - Structured JSON logging (slog)

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22 |
| HTTP Framework | Fiber v2 |
| Database | PostgreSQL 16 |
| Message Queue | RabbitMQ 3 |
| Containerization | Docker Compose |

## Quick Start

### 1. Start Infrastructure
```bash
docker-compose up -d
```

### 2. Start Services (5 terminals)

**Note:** Database migrations are handled automatically by GORM on first service startup.

```bash
# Terminal 1: Mock SMS Provider
go run cmd/mock-sms-provider/main.go

# Terminal 2: DLR Webhook
go run cmd/dlr-webhook/main.go

# Terminal 3: Broadcast API
go run cmd/broadcast-api/main.go

# Terminal 4: Outbox Publisher
go run cmd/outbox-publisher/main.go

# Terminal 5: Sender Worker
go run cmd/sender-worker/main.go
```

### 3. Test
```bash
# Create broadcast
curl -X POST http://localhost:8080/api/broadcasts \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello from SMS Broadcast!",
    "recipients": ["+66812345678", "+66887654321"]
  }'

# Check status (replace with actual broadcast_id from response)
curl http://localhost:8080/api/broadcasts/{broadcast_id}
```

See [TESTING.md](TESTING.md) for detailed testing guide.

## Project Structure

```
cmd/                            # Entry points (5 microservices)
├── broadcast-api/              # REST API to create broadcasts
├── dlr-webhook/                # Receives delivery receipts from provider
├── mock-sms-provider/          # Fake SMS gateway for testing
├── outbox-publisher/           # Polls DB, publishes to RabbitMQ
└── sender-worker/              # Consumes queue, calls SMS provider

internal/
├── domain/                     # Business entities (Message, Status) with GORM tags
├── ports/                      # Interfaces (Repository, Queue, Provider)
├── app/                        # Use cases (BroadcastService)
├── adapters/
│   ├── db/postgres/            # PostgreSQL implementation using GORM
│   ├── queue/rabbitmq/         # RabbitMQ pub/sub
│   └── provider/httpmock/      # HTTP client to SMS provider
└── transport/                  # HTTP handlers (Fiber routes)

config.go                       # Configuration from environment
docker-compose.yml              # Local infrastructure (Postgres + RabbitMQ)
```

## Database Schema

The database schema is managed by **GORM Auto-Migration**. When any service starts, it will automatically:
- Create tables if they don't exist
- Add missing columns
- Create indexes

### Models:

**broadcasts** table:
- `id` (UUID, primary key)
- `name` (text)
- `created_at` (timestamp)

**messages** table:
- `id` (UUID, primary key)
- `broadcast_id` (UUID, foreign key → broadcasts.id)
- `to_number` (text)
- `body` (text)
- `status` (text: pending/queued/sent/delivered/failed)
- `provider_id` (text, nullable)
- `created_at` (timestamp)
- `updated_at` (timestamp)

**Indexes:**
- `idx_messages_status_created` on (status, created_at)
- `idx_messages_provider_id` on (provider_id) WHERE provider_id IS NOT NULL
- `idx_messages_broadcast` on (broadcast_id)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | Broadcast API listen address |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/sms?sslmode=disable` | PostgreSQL connection |
| `AMQP_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection |
| `PROVIDER_URL` | `http://localhost:9090` | SMS provider endpoint |
| `DLR_WEBHOOK_URL` | `http://localhost:8081/dlr` | Delivery receipt webhook |

## API Endpoints

### POST /api/broadcasts
Create a new SMS broadcast.

**Request:**
```json
{
  "message": "Your message here",
  "recipients": ["+66812345678", "+66887654321"]
}
```

**Response:**
```json
{
  "broadcast_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

### GET /api/broadcasts/:id
Get broadcast status and all messages.

**Response:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "message": "Your message here",
  "total_recipients": 2,
  "messages": [
    {
      "id": "...",
      "phone_number": "+66812345678",
      "status": "delivered",
      "created_at": "2026-02-27T10:00:00Z",
      "sent_at": "2026-02-27T10:00:05Z",
      "delivered_at": "2026-02-27T10:00:08Z"
    }
  ]
}
```

## Message Status Flow

```
pending → queued → sent → delivered
                        ↘ failed
```

- **pending**: Just created, waiting for outbox publisher
- **queued**: Published to RabbitMQ, waiting for worker
- **sent**: Sent to SMS provider, waiting for delivery receipt
- **delivered**: Confirmed delivery from provider
- **failed**: Provider reported failure

## Development

### Run Tests (when implemented)
```bash
go test ./...
```

### Format Code
```bash
go fmt ./...
```

### Build All Services
```bash
go build ./cmd/broadcast-api
go build ./cmd/dlr-webhook
go build ./cmd/mock-sms-provider
go build ./cmd/outbox-publisher
go build ./cmd/sender-worker
```

## Production Considerations

For production deployment:
- [ ] Add proper authentication/authorization
- [ ] Implement rate limiting
- [ ] Add retry logic with exponential backoff
- [ ] Use proper DB migrations tool (goose, migrate)
- [ ] Add metrics (Prometheus)
- [ ] Add distributed tracing (OpenTelemetry)
- [ ] Use proper secrets management
- [ ] Add health check endpoints
- [ ] Implement graceful shutdown
- [ ] Add circuit breakers for external calls
- [ ] Set up monitoring and alerting

## License

MIT
