package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	BroadcastID uuid.UUID `gorm:"type:uuid;not null;index:idx_messages_broadcast"`
	To          string    `gorm:"column:to_number;type:text;not null"`
	Body        string    `gorm:"type:text;not null"`
	Status      Status    `gorm:"type:text;not null;default:'pending';index:idx_messages_status_created"`
	ProviderID  string    `gorm:"type:text;index:idx_messages_provider_id,where:provider_id IS NOT NULL"`
	CreatedAt   time.Time `gorm:"not null;index:idx_messages_status_created"`
	UpdatedAt   time.Time `gorm:"not null"`
}

// TableName specifies the table name for GORM
func (Message) TableName() string {
	return "messages"
}

// BeforeCreate hook ensures UUID is set before creating
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = time.Now().UTC()
	}
	return nil
}

// BeforeUpdate hook updates the UpdatedAt timestamp
func (m *Message) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now().UTC()
	return nil
}

// Broadcast groups a collection of messages sent together.
type Broadcast struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null"`
	Messages  []Message `gorm:"foreignKey:BroadcastID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (Broadcast) TableName() string {
	return "broadcasts"
}

// BeforeCreate hook ensures UUID is set before creating
func (b *Broadcast) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	return nil
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
