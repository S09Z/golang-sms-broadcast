#!/bin/bash

# Start all microservices in separate terminal tabs/windows

echo "Starting all services..."

# For macOS (Terminal.app)
if [[ "$OSTYPE" == "darwin"* ]]; then
    osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"' && go run cmd/broadcast-api/main.go"'
    sleep 1
    osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"' && go run cmd/dlr-webhook/main.go"'
    sleep 1
    osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"' && go run cmd/mock-sms-provider/main.go"'
    sleep 1
    osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"' && go run cmd/outbox-publisher/main.go"'
    sleep 1
    osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"' && go run cmd/sender-worker/main.go"'
    
    echo "✅ All services started in separate terminals"
else
    # For Linux/other (using gnome-terminal, xterm, or tmux)
    echo "For Linux, use tmux option below or manually open terminals"
fi
