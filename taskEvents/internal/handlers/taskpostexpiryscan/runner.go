package taskpostexpiryscan

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

const expirePostsPath = "/api/internal/tasks/expire-posts/"

// ExpireClient posts to taskTaskService internal expire-posts API.
type ExpireClient interface {
	ExpireOnce(ctx context.Context) error
}

// TaskTaskServiceExpireClient calls taskTaskService expire-posts.
type TaskTaskServiceExpireClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskTaskServiceExpireClient builds a client from environment.
func NewTaskTaskServiceExpireClient() *TaskTaskServiceExpireClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_BASE_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8017"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &TaskTaskServiceExpireClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// ExpireOnce triggers one expired-posts sweep.
func ExpireOnce(ctx context.Context, client ExpireClient) error {
	if client == nil {
		client = NewTaskTaskServiceExpireClient()
	}
	return client.ExpireOnce(ctx)
}

func (c *TaskTaskServiceExpireClient) ExpireOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskTaskServiceExpireClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + expirePostsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if c.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.InternalSecret)
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
		return fmt.Errorf("%s HTTP %d: %s", expirePostsPath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	// 记录本次扫描结果（best-effort）
	var out map[string]interface{}
	if json.Unmarshal(body, &out) == nil {
		log.Printf("[task_post_expiry_scan] sweep result: expired=%v published=%v",
			out["expired_count"], out["published"])
	}
	return nil
}

// ExpireTickInterval reads TASK_POST_EXPIRY_SCAN_TICK_SEC; values <=0 fall back to 24h.
func ExpireTickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TASK_POST_EXPIRY_SCAN_TICK_SEC"))
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
			Worker:    "expire_posts",
		})
	}
}

// StartExpireTicker runs the sweep on interval until ctx is cancelled.
func StartExpireTicker(ctx context.Context, client ExpireClient, interval time.Duration) {
	if client == nil {
		client = NewTaskTaskServiceExpireClient()
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := ExpireOnce(ctx, client); err != nil {
			log.Printf("[task_post_expiry_scan] sweep: %v", err)
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

// Run starts health HTTP and the expiry scan ticker (no Kafka).
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

	client := NewTaskTaskServiceExpireClient()
	interval := ExpireTickInterval()
	go StartExpireTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=expire_posts tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}
