package ports

import (
	"context"

	"golang-sms-broadcast/internal/domain"

	"github.com/google/uuid"
)

// MessageRepository defines persistence operations for messages and broadcasts.
type MessageRepository interface {
	// SaveBroadcast persists a new Broadcast.
	SaveBroadcast(ctx context.Context, b domain.Broadcast) error

	// GetBroadcast retrieves a broadcast by ID with all its messages.
	GetBroadcast(ctx context.Context, id uuid.UUID) (*domain.Broadcast, error)

	// SaveMessages persists a batch of Messages in a single transaction.
	SaveMessages(ctx context.Context, msgs []domain.Message) error

	// GetPendingMessages returns up to limit messages with StatusPending.
	GetPendingMessages(ctx context.Context, limit int) ([]domain.Message, error)

	// UpdateMessageStatus transitions a message to the given status.
	UpdateMessageStatus(ctx context.Context, id uuid.UUID, status domain.Status) error

	// UpdateMessageStatusByProviderID transitions a message by the provider's external ID.
	UpdateMessageStatusByProviderID(ctx context.Context, providerID string, status domain.Status) error

	// SetProviderID stores the external SMS provider ID on a message after submission.
	SetProviderID(ctx context.Context, id uuid.UUID, providerID string) error
}
