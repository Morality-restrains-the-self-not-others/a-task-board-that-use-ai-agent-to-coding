package relaylifecycle

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"taskEvents/config"
	"taskEvents/domain"
)

// LocalHandler returns a handler with credential service URL from env/config.
func LocalHandler() *Handler {
	base := strings.TrimSpace(os.Getenv("TASK_CREDENTIAL_SERVICE_URL"))
	if base == "" {
		base = "http://127.0.0.1:8015"
	}
	return &Handler{
		CredentialBaseURL: base,
		HTTPClient:        &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{Proxy: nil}},
	}
}

// LocalDelivery implements domain.DomainCommandPort for relay_lifecycle intents.
type LocalDelivery struct {
	Handler *Handler
}

func (d *LocalDelivery) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if d.Handler == nil {
		d.Handler = LocalHandler()
	}
	return d.Handler.Dispatch(ctx, cmd)
}

// CredentialURLFromConfig resolves taskCredentialService base URL.
func CredentialURLFromConfig(cfg config.Config) string {
	_ = cfg
	return LocalHandler().CredentialBaseURL
}

var _ domain.DomainCommandPort = (*LocalDelivery)(nil)
