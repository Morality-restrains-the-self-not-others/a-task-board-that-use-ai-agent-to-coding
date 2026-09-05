package queued_auto_run_scan

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

// dispatchOncePath POST 到 taskTaskService：一次触发执行「排队调度分发」+「窗口自动关闭」。
const dispatchOncePath = "/api/internal/tasks/queued-schedule/dispatch-once/"

// DispatchClient posts to taskTaskService internal dispatch-once API.
type DispatchClient interface {
	DispatchOnce(ctx context.Context) error
}

// TaskTaskServiceDispatchClient calls taskTaskService queued-schedule dispatch-once.
type TaskTaskServiceDispatchClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskTaskServiceDispatchClient builds a client from environment.
func NewTaskTaskServiceDispatchClient() *TaskTaskServiceDispatchClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_BASE_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8017"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &TaskTaskServiceDispatchClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// DispatchOnce triggers one queued-schedule scan.
func DispatchOnce(ctx context.Context, client DispatchClient) error {
	if client == nil {
		client = NewTaskTaskServiceDispatchClient()
	}
	return client.DispatchOnce(ctx)
}

func (c *TaskTaskServiceDispatchClient) DispatchOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskTaskServiceDispatchClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + dispatchOncePath
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
		return fmt.Errorf("%s HTTP %d: %s", dispatchOncePath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// DispatchInterval reads QUEUED_AUTO_RUN_SCAN_TICK_SEC; values <=0 fall back to 30s.
func DispatchInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("QUEUED_AUTO_RUN_SCAN_TICK_SEC"))
	if raw == "" {
		return 30 * time.Second
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 30 * time.Second
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
			Worker:    "dispatch_once",
		})
	}
}

// StartDispatchTicker runs dispatch-once on interval until ctx is cancelled.
func StartDispatchTicker(ctx context.Context, client DispatchClient, interval time.Duration) {
	if client == nil {
		client = NewTaskTaskServiceDispatchClient()
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := DispatchOnce(ctx, client); err != nil {
			log.Printf("[queued_auto_run_scan] dispatch-once: %v", err)
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

// Run starts health HTTP and the dispatch ticker (no Kafka).
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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	client := NewTaskTaskServiceDispatchClient()
	interval := DispatchInterval()
	go StartDispatchTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=dispatch_once tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	<-ctx.Done()
	server.WaitHealthShutdown(httpDone)
	log.Printf("[%s] shutdown", serviceName)
}
