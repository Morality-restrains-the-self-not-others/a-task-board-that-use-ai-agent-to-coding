package emailinviteexpiryscan

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

const expireDuePath = "/api/internal/taskauth/email-invites/expire-due/"

// ExpireClient posts to taskAuth internal email-invite expire-due API.
type ExpireClient interface {
	ExpireOnce(ctx context.Context) error
}

// TaskAuthExpireClient calls taskAuth email-invites expire-due.
type TaskAuthExpireClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskAuthExpireClient builds a client from environment.
func NewTaskAuthExpireClient() *TaskAuthExpireClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL")), "/")
	if base == "" {
		// taskAuth 实际监听 8003（conf/auth/task-auth/config.yaml port: 8003）。
		base = "http://127.0.0.1:8003"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &TaskAuthExpireClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     tracelog.DirectClient(60 * time.Second),
	}
}

// ExpireOnce triggers one pending-invite expiry sweep.
func ExpireOnce(ctx context.Context, client ExpireClient) error {
	if client == nil {
		client = NewTaskAuthExpireClient()
	}
	return client.ExpireOnce(ctx)
}

func (c *TaskAuthExpireClient) ExpireOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskAuthExpireClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + expireDuePath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if c.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", c.InternalSecret)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = tracelog.DirectClient(60 * time.Second)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s HTTP %d: %s", expireDuePath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out map[string]interface{}
	if json.Unmarshal(body, &out) == nil {
		log.Printf("[email_invite_expiry_scan] sweep result: expired=%v", out["expired"])
	}
	return nil
}

// ExpireTickInterval reads EMAIL_INVITE_EXPIRY_SCAN_TICK_SEC; values <=0 fall back to 1h.
func ExpireTickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("EMAIL_INVITE_EXPIRY_SCAN_TICK_SEC"))
	if raw == "" {
		return time.Hour
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return time.Hour
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
			Worker:    "email_invite_expiry_scan",
		})
	}
}

// StartExpireTicker runs the invite expiry sweep on interval until ctx is cancelled.
func StartExpireTicker(ctx context.Context, client ExpireClient, interval time.Duration) {
	if client == nil {
		client = NewTaskAuthExpireClient()
	}
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := ExpireOnce(ctx, client); err != nil {
			log.Printf("[email_invite_expiry_scan] sweep: %v", err)
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

// Run starts health HTTP and the email-invite expiry ticker (no Kafka).
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

	client := NewTaskAuthExpireClient()
	interval := ExpireTickInterval()
	go StartExpireTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=email_invite_expiry_scan tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}
