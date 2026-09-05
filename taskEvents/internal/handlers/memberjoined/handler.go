package memberjoined

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

// GitIdentityClient creates/ensures default git identity for a company member.
type GitIdentityClient interface {
	EnsureDefault(ctx context.Context, userID, companyID, memberID, memberName string) error
}

type httpGitIdentityClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

func NewDefaultGitIdentityClient() GitIdentityClient {
	base := strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_BASE_URL"))
	if base == "" {
		base = "http://127.0.0.1:8017"
	}
	secret := strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("TASK_TASK_INTERNAL_SECRET"))
	}
	return &httpGitIdentityClient{
		BaseURL:        strings.TrimRight(base, "/"),
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *httpGitIdentityClient) EnsureDefault(ctx context.Context, userID, companyID, memberID, memberName string) error {
	body, _ := json.Marshal(map[string]string{
		"user_id":     userID,
		"company_id":  companyID,
		"member_id":   memberID,
		"member_name": memberName,
	})
	url := c.BaseURL + "/api/internal/git-identities/ensure-default/"
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
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
		return fmt.Errorf("permanent ensure-default status %d: %s", resp.StatusCode, string(raw))
	}
	return fmt.Errorf("retryable ensure-default status %d: %s", resp.StatusCode, string(raw))
}

// Handler consumes MEMBER_JOINED → ensure default git identity.
type Handler struct {
	Client GitIdentityClient
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "MEMBER_JOINED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	memberID, err := payload.RequiredStrField(data, "member_id")
	if err != nil {
		log.Printf("[member_joined] missing member_id: %v", err)
		return domain.DispatchPermanent, err
	}
	userID, err := payload.RequiredStrField(data, "user_id")
	if err != nil {
		log.Printf("[member_joined] missing user_id: %v", err)
		return domain.DispatchPermanent, err
	}
	companyID, err := payload.RequiredStrField(data, "company_id")
	if err != nil {
		log.Printf("[member_joined] missing company_id: %v", err)
		return domain.DispatchPermanent, err
	}
	memberName := payload.StrField(data, "member_name")
	if memberName == "" {
		memberName = payload.StrField(data, "company_member_name")
	}
	log.Printf("[member_joined] ensure git identity member_id=%s user_id=%s company_id=%s", memberID, userID, companyID)

	if err := h.Client.EnsureDefault(ctx, userID, companyID, memberID, memberName); err != nil {
		log.Printf("[member_joined] ensure failed: %v", err)
		if strings.Contains(err.Error(), "permanent") {
			return domain.DispatchPermanent, err
		}
		return domain.DispatchRetryable, err
	}
	log.Printf("[member_joined] ensure ok member_id=%s", memberID)
	return domain.DispatchSuccess, nil
}
