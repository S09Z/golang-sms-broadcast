package app

import (
	"context"
	"fmt"
	"log/slog"

	"golang-sms-broadcast/internal/domain"
	"golang-sms-broadcast/internal/ports"
)

// BroadcastService is the central application service that orchestrates
// creating broadcasts, dispatching messages, and handling delivery receipts.
type BroadcastService struct {
	repo      ports.MessageRepository
	publisher ports.MessagePublisher
	provider  ports.SMSProvider
	log       *slog.Logger
}

// NewBroadcastService wires the service with its dependencies.
func NewBroadcastService(
	repo ports.MessageRepository,
	publisher ports.MessagePublisher,
	provider ports.SMSProvider,
	log *slog.Logger,
) *BroadcastService {
	return &BroadcastService{
		repo:      repo,
		publisher: publisher,
		provider:  provider,
		log:       log,
	}
}

// CreateBroadcastRequest is the input for creating a new SMS broadcast.
type CreateBroadcastRequest struct {
	Name      string
	Body      string
	Recipient []string
}

// CreateBroadcast persists a Broadcast and its Messages to the outbox.
func (s *BroadcastService) CreateBroadcast(ctx context.Context, req CreateBroadcastRequest) (domain.Broadcast, error) {
	broadcast := domain.NewBroadcast(req.Name)

	if err := s.repo.SaveBroadcast(ctx, broadcast); err != nil {
		return domain.Broadcast{}, fmt.Errorf("save broadcast: %w", err)
	}

	msgs := make([]domain.Message, 0, len(req.Recipient))
	for _, to := range req.Recipient {
		msgs = append(msgs, domain.NewMessage(broadcast.ID, to, req.Body))
	}

	if err := s.repo.SaveMessages(ctx, msgs); err != nil {
		return domain.Broadcast{}, fmt.Errorf("save messages: %w", err)
	}

	s.log.Info("broadcast created", "broadcast_id", broadcast.ID, "recipients", len(msgs))
	return broadcast, nil
}

// PublishPendingMessages reads pending outbox messages and publishes them to the queue.
// This is called by the outbox-publisher binary on a poll interval.
func (s *BroadcastService) PublishPendingMessages(ctx context.Context, batchSize int) (int, error) {
	msgs, err := s.repo.GetPendingMessages(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("get pending messages: %w", err)
	}

	published := 0
	for _, msg := range msgs {
		if err := s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusQueued); err != nil {
			s.log.Error("mark queued failed", "msg_id", msg.ID, "err", err)
			continue
		}

		if err := s.publisher.Publish(ctx, msg); err != nil {
			// Roll back to pending so the next poll retries it.
			_ = s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusPending)
			s.log.Error("publish failed", "msg_id", msg.ID, "err", err)
			continue
		}

		published++
		s.log.Info("message queued", "msg_id", msg.ID, "to", msg.To)
	}

	return published, nil
}

// SendMessage calls the SMS provider for a single queued message.
// This is called by the sender-worker binary for each message it dequeues.
func (s *BroadcastService) SendMessage(ctx context.Context, msg domain.Message) error {
	result, err := s.provider.Send(ctx, msg)
	if err != nil {
		_ = s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusFailed)
		return fmt.Errorf("provider send: %w", err)
	}

	if err := s.repo.SetProviderID(ctx, msg.ID, result.ProviderID); err != nil {
		s.log.Error("set provider id failed", "msg_id", msg.ID, "err", err)
	}

	if err := s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusSent); err != nil {
		return fmt.Errorf("update status sent: %w", err)
	}

	s.log.Info("message sent", "msg_id", msg.ID, "provider_id", result.ProviderID)
	return nil
}

// HandleDLR processes a delivery receipt from the SMS provider webhook.
func (s *BroadcastService) HandleDLR(ctx context.Context, dlr ports.DLRPayload) error {
	if err := s.repo.UpdateMessageStatusByProviderID(ctx, dlr.ProviderID.String(), dlr.Status); err != nil {
		return fmt.Errorf("update dlr status: %w", err)
	}

	s.log.Info("DLR received", "provider_id", dlr.ProviderID, "status", dlr.Status)
	return nil
}
