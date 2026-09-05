package taskstatuschanged

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// stopRelaySidecar POSTs /v1/stop to go_relayToTrae (same path as taskContainerGateway forwardToRelay).
func stopRelaySidecar(ctx context.Context, taskID string) error {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_RELAY_TO_TRAE_URL")), "/")
	}
	if base == "" {
		base = "http://127.0.0.1:8797"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/stop", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if secret := strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_SECRET")); secret != "" {
		req.Header.Set("X-Relay-To-Trae-Secret", secret)
	}
	if strings.TrimSpace(taskID) != "" {
		req.Header.Set("X-Task-Id", taskID)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("relay /v1/stop status %d", resp.StatusCode)
	}
	log.Printf("[task_status_changed] relay sidecar /v1/stop ok task_id=%s", taskID)
	return nil
}

func stopMockRunSidecar(ctx context.Context, taskID string) error {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("MOCK_RUN_CONTAINER_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8796"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/stop", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if secret := strings.TrimSpace(os.Getenv("MOCK_RUN_CONTAINER_SECRET")); secret != "" {
		req.Header.Set("X-Mock-Run-Secret", secret)
	}
	if strings.TrimSpace(taskID) != "" {
		req.Header.Set("X-Task-Id", taskID)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("mock-run /v1/stop status %d", resp.StatusCode)
	}
	log.Printf("[task_status_changed] mock-run /v1/stop ok task_id=%s", taskID)
	return nil
}
