package taskstatuschanged

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	GracefulShutdownAwaitEvent = "TASK_GRACEFUL_SHUTDOWN_AWAIT"
	notifyHTTPTimeout          = 5 * time.Second
)

// ContainerNotifier sends terminal shutdown to a running container.
type ContainerNotifier interface {
	NotifyShutdown(ctx context.Context, serverURL, terminalKind, taskID, reason string) error
}

type httpContainerNotifier struct {
	Client *http.Client
}

func (n httpContainerNotifier) client() *http.Client {
	if n.Client != nil {
		return n.Client
	}
	return &http.Client{Timeout: notifyHTTPTimeout}
}

func (n httpContainerNotifier) NotifyShutdown(ctx context.Context, serverURL, terminalKind, taskID, reason string) error {
	base := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if base == "" {
		return fmt.Errorf("empty server_url")
	}
	u := base + "/api/task-lifecycle/shutdown"
	body, _ := json.Marshal(map[string]string{
		"terminal_kind": terminalKind,
		"task_id":       taskID,
		"reason":        reason,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shutdown notify HTTP %d", resp.StatusCode)
	}
	return nil
}

func defaultNotifier() ContainerNotifier {
	return httpContainerNotifier{}
}
