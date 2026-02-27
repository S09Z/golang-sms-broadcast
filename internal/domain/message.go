package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Status represents the lifecycle state of an SMS message.
type Status string

const (
	StatusPending   Status = "pending"   // Saved to outbox, not yet queued
	StatusQueued    Status = "queued"    // Published to message queue
	StatusSent      Status = "sent"      // Accepted by SMS provider
	StatusDelivered Status = "delivered" // Confirmed delivered to recipient (DLR)
	StatusFailed    Status = "failed"    // Permanently failed
)

// Message is the core domain entity representing a single SMS.
type Message struct {
	ID          uuid.UUID
	BroadcastID uuid.UUID
	To          string
	Body        string
	Status      Status
	ProviderID  string // External ID returned by the SMS provider
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Broadcast groups a collection of messages sent together.
type Broadcast struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

// NewBroadcast creates a new Broadcast with a generated ID.
func NewBroadcast(name string) Broadcast {
	return Broadcast{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
}

// NewMessage creates a new pending Message for a given broadcast.
func NewMessage(broadcastID uuid.UUID, to, body string) Message {
	now := time.Now().UTC()
	return Message{
		ID:          uuid.New(),
		BroadcastID: broadcastID,
		To:          to,
		Body:        body,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Domain errors
var (
	ErrMessageNotFound   = errors.New("message not found")
	ErrBroadcastNotFound = errors.New("broadcast not found")
	ErrInvalidStatus     = errors.New("invalid status transition")
)
