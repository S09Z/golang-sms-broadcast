#!/bin/bash

# Start all services in tmux session with split panes

SESSION="sms-broadcast"

# Check if tmux session exists
tmux has-session -t $SESSION 2>/dev/null

if [ $? != 0 ]; then
    # Create new session with first service
    tmux new-session -d -s $SESSION -n services
    
    # Split into 5 panes
    tmux split-window -h -t $SESSION:services
    tmux split-window -v -t $SESSION:services.0
    tmux split-window -v -t $SESSION:services.2
    tmux split-window -v -t $SESSION:services.0
    
    # Run services in each pane
    tmux send-keys -t $SESSION:services.0 'go run cmd/broadcast-api/main.go' C-m
    tmux send-keys -t $SESSION:services.1 'go run cmd/dlr-webhook/main.go' C-m
    tmux send-keys -t $SESSION:services.2 'go run cmd/mock-sms-provider/main.go' C-m
    tmux send-keys -t $SESSION:services.3 'go run cmd/outbox-publisher/main.go' C-m
    tmux send-keys -t $SESSION:services.4 'go run cmd/sender-worker/main.go' C-m
    
    # Adjust layout
    tmux select-layout -t $SESSION:services tiled
    
    echo "✅ Tmux session '$SESSION' created with all services"
    echo "📌 Attach with: tmux attach -t $SESSION"
else
    echo "⚠️  Tmux session '$SESSION' already exists"
    echo "📌 Attach with: tmux attach -t $SESSION"
fi
