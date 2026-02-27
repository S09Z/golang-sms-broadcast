package ports

import (
	"context"

	"golang-sms-broadcast/internal/domain"

	"github.com/google/uuid"
)

// SendResult is the response from the SMS provider after submitting a message.
type SendResult struct {
	ProviderID string // External message ID assigned by the provider
}

// SMSProvider abstracts the external SMS gateway.
type SMSProvider interface {
	// Send submits an SMS to the provider and returns the provider's message ID.
	Send(ctx context.Context, msg domain.Message) (SendResult, error)
}

// DLRPayload is the normalised delivery receipt from the provider's webhook.
type DLRPayload struct {
	ProviderID uuid.UUID
	Status     domain.Status
}
