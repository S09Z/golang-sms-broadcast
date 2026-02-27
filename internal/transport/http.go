package transport

import (
	"log/slog"

	"golang-sms-broadcast/internal/app"
	"golang-sms-broadcast/internal/domain"
	"golang-sms-broadcast/internal/ports"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler holds all HTTP handlers for the SMS broadcast service.
type Handler struct {
	svc *app.BroadcastService
	log *slog.Logger
}

// NewHandler wires up a Handler with its dependencies.
func NewHandler(svc *app.BroadcastService, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register mounts all routes onto the given Fiber app.
func (h *Handler) Register(router fiber.Router) {
	router.Post("/broadcasts", h.CreateBroadcast)
	router.Post("/dlr", h.HandleDLR)
}

// ── Broadcast API ─────────────────────────────────────────────────────────────

type createBroadcastRequest struct {
	Name       string   `json:"name"`
	Body       string   `json:"body"`
	Recipients []string `json:"recipients"`
}

type createBroadcastResponse struct {
	BroadcastID string `json:"broadcast_id"`
	Queued      int    `json:"queued"`
}

// CreateBroadcast accepts a broadcast request and saves it to the outbox.
//
// POST /broadcasts
// Body: { "name": "...", "body": "...", "recipients": ["...", ...] }
func (h *Handler) CreateBroadcast(c *fiber.Ctx) error {
	var req createBroadcastRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" || req.Body == "" || len(req.Recipients) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name, body and recipients are required"})
	}

	broadcast, err := h.svc.CreateBroadcast(c.Context(), app.CreateBroadcastRequest{
		Name:      req.Name,
		Body:      req.Body,
		Recipient: req.Recipients,
	})
	if err != nil {
		h.log.Error("create broadcast", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	return c.Status(fiber.StatusCreated).JSON(createBroadcastResponse{
		BroadcastID: broadcast.ID.String(),
		Queued:      len(req.Recipients),
	})
}

// ── DLR Webhook ───────────────────────────────────────────────────────────────

type dlrRequest struct {
	ProviderID string `json:"provider_id"`
	Status     string `json:"status"`
}

// HandleDLR receives delivery receipts from the SMS provider.
//
// POST /dlr
// Body: { "provider_id": "...", "status": "delivered"|"failed" }
func (h *Handler) HandleDLR(c *fiber.Ctx) error {
	var req dlrRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.ProviderID == "" || req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provider_id and status are required"})
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provider_id must be a valid UUID"})
	}

	dlr := ports.DLRPayload{
		ProviderID: providerID,
		Status:     statusFromString(req.Status),
	}

	if err := h.svc.HandleDLR(c.Context(), dlr); err != nil {
		h.log.Error("handle dlr", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func statusFromString(s string) domain.Status {
	return domain.Status(s)
}
