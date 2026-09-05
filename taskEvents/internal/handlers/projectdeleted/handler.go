package projectdeleted

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
)

type DetachClient interface {
	DetachByProject(ctx context.Context, projectID string) error
}

type httpDetachClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

func NewDefaultDetachClient() DetachClient {
	base := strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_BASE_URL"))
	if base == "" {
		base = "http://127.0.0.1:8017"
	}
	secret := strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("TASK_TASK_INTERNAL_SECRET"))
	}
	return &httpDetachClient{
		BaseURL:        strings.TrimRight(base, "/"),
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *httpDetachClient) DetachByProject(ctx context.Context, projectID string) error {
	body, _ := json.Marshal(map[string]string{"project_id": projectID})
	url := c.BaseURL + "/api/internal/task-projects/detach-by-project/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Id", "internal")
	if c.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.InternalSecret)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
		return fmt.Errorf("permanent detach status %d: %s", resp.StatusCode, string(raw))
	}
	return fmt.Errorf("retryable detach status %d: %s", resp.StatusCode, string(raw))
}

type Handler struct {
	Client DetachClient
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "PROJECT_DELETED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	projectID, err := payload.RequiredStrField(data, "project_id")
	if err != nil {
		log.Printf("[project_deleted] missing project_id: %v", err)
		return domain.DispatchPermanent, err
	}
	log.Printf("[project_deleted] detach task_projects project_id=%s", projectID)
	if h.Client == nil {
		return domain.DispatchPermanent, fmt.Errorf("detach client is required")
	}
	if err := h.Client.DetachByProject(ctx, projectID); err != nil {
		log.Printf("[project_deleted] detach failed: %v", err)
		if strings.Contains(err.Error(), "permanent") {
			return domain.DispatchPermanent, err
		}
		return domain.DispatchRetryable, err
	}
	return domain.DispatchSuccess, nil
}

var _ domain.DomainCommandPort = (*Handler)(nil)
