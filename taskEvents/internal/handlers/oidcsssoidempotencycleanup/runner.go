package oidcsssoidempotencycleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"taskEvents/server"
	"tracelog"
)

const cleanupPath = "/api/internal/taskauth/oidc-sso-idempotency/cleanup/"

// CleanupClient posts to taskAuth internal oidc-sso-idempotency cleanup API.
type CleanupClient interface {
	CleanupOnce(ctx context.Context) error
}

// TaskAuthCleanupClient calls taskAuth oidc-sso-idempotency cleanup.
type TaskAuthCleanupClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskAuthCleanupClient builds a client from environment.
func NewTaskAuthCleanupClient() *TaskAuthCleanupClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL")), "/")
	if base == "" {
		// taskAuth 实际监听 8003（conf/auth/task-auth/config.yaml port: 8003）。
		base = "http://127.0.0.1:8003"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &TaskAuthCleanupClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// CleanupOnce triggers one auth_oidc_sso_idempotency sweep.
func CleanupOnce(ctx context.Context, client CleanupClient) error {
	if client == nil {
		client = NewTaskAuthCleanupClient()
	}
	return client.CleanupOnce(ctx)
}

func (c *TaskAuthCleanupClient) CleanupOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskAuthCleanupClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + cleanupPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if c.InternalSecret != "" {
		// taskAuth requireInternalSecret 校验 X-TaskAuth-Internal-Secret。
		req.Header.Set("X-TaskAuth-Internal-Secret", c.InternalSecret)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s HTTP %d: %s", cleanupPath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	// 记录本次扫描结果（best-effort）
	var out map[string]interface{}
	if json.Unmarshal(body, &out) == nil {
		log.Printf("[oidc_sso_idempotency_cleanup] sweep result: deleted=%v", out["deleted"])
	}
	return nil
}

// CleanupTickInterval reads OIDC_SSO_IDEMPOTENCY_CLEANUP_TICK_SEC; values <=0 fall back to 24h.
func CleanupTickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("OIDC_SSO_IDEMPOTENCY_CLEANUP_TICK_SEC"))
	if raw == "" {
		return 24 * time.Hour
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(sec) * time.Second
}

type timerHealthState struct {
	Status    string `json:"status"`
	Domain    string `json:"domain"`
	Transport string `json:"transport"`
	Worker    string `json:"worker"`
}

func timerHealthHandler(domainName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(timerHealthState{
			Status:    "ok",
			Domain:    domainName,
			Transport: "timer",
			Worker:    "oidc_sso_idempotency_cleanup",
		})
	}
}

// StartCleanupTicker runs the oidc-sso idempotency cleanup sweep on interval until ctx is cancelled.
func StartCleanupTicker(ctx context.Context, client CleanupClient, interval time.Duration) {
	if client == nil {
		client = NewTaskAuthCleanupClient()
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := CleanupOnce(ctx, client); err != nil {
			log.Printf("[oidc_sso_idempotency_cleanup] sweep: %v", err)
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// Run starts health HTTP and the oidc-sso idempotency cleanup ticker (no Kafka).
func Run(serviceName, host string, port int) {
	tracelog.InitConsumer(serviceName)
	shutdownOtel, err := tracelog.InitOtel(context.Background(), serviceName)
	if err != nil {
		log.Printf("[%s] otel disabled: %v", serviceName, err)
	} else {
		defer func() { _ = shutdownOtel(context.Background()) }()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := NewTaskAuthCleanupClient()
	interval := CleanupTickInterval()
	go StartCleanupTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=oidc_sso_idempotency_cleanup tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}
