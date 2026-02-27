.PHONY: help build run-all run-broadcast run-dlr run-mock run-outbox run-worker clean test test-coverage test-domain test-app test-adapters docker-up docker-down docker-logs db-check rabbitmq-check deps tidy logs logs-follow stop-background restart kill-ports ps migrate load-test

# Default target
help:
	@echo "📦 SMS Broadcast System - Makefile Commands"
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  make docker-up          Start PostgreSQL & RabbitMQ"
	@echo "  make deps               Install Go dependencies"
	@echo "  make run-all            Run all 5 services (tmux required)"
	@echo ""
	@echo "🔧 Individual Services:"
	@echo "  make run-broadcast      Run Broadcast API (port 8080)"
	@echo "  make run-dlr            Run DLR Webhook (port 8081)"
	@echo "  make run-mock           Run Mock SMS Provider (port 9090)"
	@echo "  make run-outbox         Run Outbox Publisher"
	@echo "  make run-worker         Run Sender Worker"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-up          Start infrastructure"
	@echo "  make docker-down        Stop infrastructure"
	@echo "  make docker-logs        View logs"
	@echo "  make docker-clean       Remove volumes"
	@echo ""
	@echo "🔍 Utilities:"
	@echo "  make logs               View all service logs (tmux)"
	@echo "  make logs-follow        Tail all service logs to files"
	@echo "  make ps                 Show running Go processes"
	@echo "  make kill-ports         Kill processes on ports 8080,8081,9090"
	@echo "  make restart            Restart all services"
	@echo "  make migrate            Force database migration"
	@echo "  make db-check           Verify database connection"
	@echo "  make rabbitmq-check     Check RabbitMQ status"
	@echo "  make test               Run tests"
	@echo "  make test-coverage      Run tests with coverage report"
	@echo "  make test-domain        Test domain layer"
	@echo "  make test-app           Test application layer"
	@echo "  make load-test          Run load test (100 & 1000 requests)"
	@echo "  make clean              Clean build artifacts"
	@echo "  make build              Build all services"
	@echo ""

# Install dependencies
deps:
	@echo "📥 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

tidy:
	go mod tidy

# Start infrastructure
docker-up:
	@echo "🐳 Starting PostgreSQL and RabbitMQ..."
	docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 3
	@echo "✅ Infrastructure ready"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 RabbitMQ: localhost:5672"
	@echo "🌐 RabbitMQ UI: http://localhost:15672 (admin/admin123)"

# Stop infrastructure
docker-down:
	@echo "🛑 Stopping infrastructure..."
	docker-compose down
	@echo "✅ Infrastructure stopped"

# Stop and remove volumes
docker-clean:
	@echo "🧹 Cleaning infrastructure..."
	docker-compose down -v
	@echo "✅ Infrastructure cleaned"

# View docker logs
docker-logs:
	docker-compose logs -f

# Check database connection
db-check:
	@echo "🔍 Checking database connection..."
	@docker exec -it $$(docker ps -qf "name=postgres") psql -U postgres -d sms -c "SELECT 'Connected to SMS database' as status;" || echo "❌ Database not accessible"
	@docker exec -it $$(docker ps -qf "name=postgres") psql -U postgres -d sms -c "\dt" || echo "❌ No tables found (run 'make migrate' to create tables)"

# Force database migration
migrate:
	@echo "🔄 Running database migration..."
	go run cmd/migrate/main.go
	@echo ""
	@echo "✅ Migration complete - refresh DBeaver to see tables"

# Check RabbitMQ
rabbitmq-check:
	@echo "🔍 Checking RabbitMQ..."
	@docker exec -it $$(docker ps -qf "name=rabbitmq") rabbitmqctl status || echo "❌ RabbitMQ not accessible"

# View logs (attach to tmux session)
logs:
	@echo "📋 Attaching to tmux session..."
	@echo "💡 Navigate between panes: Ctrl+b then arrow keys"
	@echo "💡 Scroll in pane: Ctrl+b then [ (q to exit scroll mode)"
	@echo "💡 Zoom pane: Ctrl+b then z"
	@echo "💡 Detach: Ctrl+b then d"
	@echo ""
	@tmux attach -t sms-broadcast || echo "❌ No tmux session found. Run 'make run-all' first"

# Follow logs to files (alternative approach)
logs-follow:
	@echo "📋 Starting all services with file logging..."
	@mkdir -p logs
	@echo "🚀 Broadcasting API logs → logs/broadcast-api.log"
	@nohup go run cmd/broadcast-api/main.go > logs/broadcast-api.log 2>&1 &
	@echo "🚀 DLR Webhook logs → logs/dlr-webhook.log"
	@nohup go run cmd/dlr-webhook/main.go > logs/dlr-webhook.log 2>&1 &
	@echo "🚀 Mock Provider logs → logs/mock-sms-provider.log"
	@nohup go run cmd/mock-sms-provider/main.go > logs/mock-sms-provider.log 2>&1 &
	@echo "🚀 Outbox Publisher logs → logs/outbox-publisher.log"
	@nohup go run cmd/outbox-publisher/main.go > logs/outbox-publisher.log 2>&1 &
	@echo "🚀 Sender Worker logs → logs/sender-worker.log"
	@nohup go run cmd/sender-worker/main.go > logs/sender-worker.log 2>&1 &
	@sleep 2
	@echo ""
	@echo "✅ All services running in background"
	@echo "📋 View logs with: tail -f logs/*.log"
	@echo "📋 View one service: tail -f logs/broadcast-api.log"
	@echo "🛑 Stop all: make stop-background"

# Stop background services
stop-background:
	@echo "🛑 Stopping background services..."
	@pkill -f "go run cmd/" || echo "No services running"
	@echo "✅ Services stopped"

# Build all services
build:
	@echo "🔨 Building all services..."
	@mkdir -p bin
	go build -o bin/broadcast-api cmd/broadcast-api/main.go
	go build -o bin/dlr-webhook cmd/dlr-webhook/main.go
	go build -o bin/mock-sms-provider cmd/mock-sms-provider/main.go
	go build -o bin/outbox-publisher cmd/outbox-publisher/main.go
	go build -o bin/sender-worker cmd/sender-worker/main.go
	@echo "✅ All services built in ./bin/"

# Run individual services
run-broadcast:
	@echo "🚀 Starting Broadcast API on :8080..."
	go run cmd/broadcast-api/main.go

run-dlr:
	@echo "🚀 Starting DLR Webhook on :8081..."
	go run cmd/dlr-webhook/main.go

run-mock:
	@echo "🚀 Starting Mock SMS Provider on :9090..."
	go run cmd/mock-sms-provider/main.go

run-outbox:
	@echo "🚀 Starting Outbox Publisher..."
	go run cmd/outbox-publisher/main.go

run-worker:
	@echo "🚀 Starting Sender Worker..."
	go run cmd/sender-worker/main.go

# Run all services in tmux
run-all:
	@echo "🚀 Starting all services in tmux..."
	@if ! command -v tmux &> /dev/null; then \
		echo "❌ tmux not installed. Install with: brew install tmux"; \
		exit 1; \
	fi
	@chmod +x start-tmux.sh
	@./start-tmux.sh
	@echo "✅ All services running in tmux session 'sms-broadcast'"
	@echo "📌 Attach with: tmux attach -t sms-broadcast"
	@echo "📌 Detach with: Ctrl+b then d"
	@echo "📌 Kill session: make stop-all"

# Stop all services in tmux
stop-all:
	@echo "🛑 Stopping all services..."
	@tmux kill-session -t sms-broadcast 2>/dev/null || echo "No tmux session found"
	@pkill -f "go run cmd/" 2>/dev/null || echo "No background Go processes found"
	@echo "✅ All services stopped"

# Show running Go processes
ps:
	@echo "🔍 Running Go processes:"
	@ps aux | grep "go run cmd/" | grep -v grep || echo "No Go services running"
	@echo ""
	@echo "🔍 Ports in use:"
	@lsof -i :8080 -i :8081 -i :9090 2>/dev/null || echo "No services on ports 8080, 8081, 9090"

# Kill processes on specific ports
kill-ports:
	@echo "🔪 Killing processes on ports 8080, 8081, 9090..."
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || echo "Port 8080 clear"
	@lsof -ti :8081 | xargs kill -9 2>/dev/null || echo "Port 8081 clear"
	@lsof -ti :9090 | xargs kill -9 2>/dev/null || echo "Port 9090 clear"
	@echo "✅ Ports cleared"

# Restart all services
restart: stop-all kill-ports
	@echo "🔄 Restarting all services..."
	@sleep 2
	@$(MAKE) run-all
	@echo "✅ Services restarted"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Run tests for specific package
test-domain:
	@echo "🧪 Testing domain layer..."
	go test -v -race -cover ./internal/domain/...

test-app:
	@echo "🧪 Testing application layer..."
	go test -v -race -cover ./internal/app/...

test-adapters:
	@echo "🧪 Testing adapters..."
	go test -v -race -cover ./internal/adapters/...

# Load testing
load-test:
	@echo "🔥 Running load tests..."
	@echo "⚠️  Make sure broadcast-api is running (make run-broadcast)"
	@echo ""
	go run cmd/load-test/main.go

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	go clean
	@echo "✅ Clean complete"

# Development workflow
dev: docker-up deps
	@echo "🎉 Development environment ready!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make run-all' to start all services"
	@echo "  2. Test with: curl -X POST http://localhost:8080/api/broadcasts \\"
	@echo "     -H 'Content-Type: application/json' \\"
	@echo "     -d '{\"message\":\"Test\",\"recipients\":[\"+66812345678\"]}'"
	@echo ""

# Full setup (first time)
setup: docker-up deps build
	@echo ""
	@echo "✅ Setup complete!"
	@echo "Run 'make run-all' to start all services"

# Quick start everything
start: docker-up run-all

# Stop everything
stop: stop-all docker-down
