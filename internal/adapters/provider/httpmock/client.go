package httpmock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang-sms-broadcast/internal/domain"
	"golang-sms-broadcast/internal/ports"
)

// Client implements ports.SMSProvider by forwarding requests to a mock HTTP server.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a Client targeting the given base URL.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type sendRequest struct {
	MessageID string `json:"message_id"`
	To        string `json:"to"`
	Body      string `json:"body"`
	DLRHook   string `json:"dlr_webhook_url"`
}

type sendResponse struct {
	ProviderID string `json:"provider_id"`
}

// Send posts the message to the mock provider's /send endpoint.
func (c *Client) Send(ctx context.Context, msg domain.Message) (ports.SendResult, error) {
	payload := sendRequest{
		MessageID: msg.ID.String(),
		To:        msg.To,
		Body:      msg.Body,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ports.SendResult{}, fmt.Errorf("marshal send request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/send", bytes.NewReader(body))
	if err != nil {
		return ports.SendResult{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ports.SendResult{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return ports.SendResult{}, fmt.Errorf("provider returned %d", resp.StatusCode)
	}

	var sr sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return ports.SendResult{}, fmt.Errorf("decode response: %w", err)
	}

	return ports.SendResult{ProviderID: sr.ProviderID}, nil
}
