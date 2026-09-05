package cloud_csc_reconcile

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

// orphanPath 先打标（terminal_released=1），leakedPath 再清资源 —— 同一条 sweep 流水线。
const (
	orphanPath        = "/api/internal/cloud/compute/reconcile-orphan-csc/"
	leakedPath        = "/api/internal/cloud/compute/reconcile-leaked-servers/"
	runtimePath       = "/api/internal/cloud/compute/reconcile-workspace-machine-runtimes/"
	staleStartingPath = "/api/internal/cloud/compute/reconcile-stale-starting-bindings/"
	sweepWorker       = "sweep"
)

// SweepClient posts to taskCloudService internal reconcile APIs.
type SweepClient interface {
	SweepOnce(ctx context.Context) error
}

// TaskCloudSweepClient calls taskCloudService orphan + leaked reconcile APIs.
type TaskCloudSweepClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskCloudSweepClient builds a client from environment.
func NewTaskCloudSweepClient() *TaskCloudSweepClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8018"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET"))
	return &TaskCloudSweepClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// SweepOnce triggers one sweep: orphan → leaked → runtime → stale starting.
func SweepOnce(ctx context.Context, client SweepClient) error {
	if client == nil {
		client = NewTaskCloudSweepClient()
	}
	return client.SweepOnce(ctx)
}

func (c *TaskCloudSweepClient) post(ctx context.Context, path string) error {
	url := strings.TrimRight(c.BaseURL, "/") + path
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
		return fmt.Errorf("%s HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *TaskCloudSweepClient) SweepOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskCloudSweepClient()
	}
	// 先标 orphan，再清 leaked：二者共享 terminal_released 流水线，顺序不能反。
	// 看板 GET 不再 Describe；last_runtime_status 由本 tick 第三步刷新。
	// 第四步收口「VM Running 但容器未 register-reachability」过久的 starting binding。
	if err := c.post(ctx, orphanPath); err != nil {
		return fmt.Errorf("orphan sweep: %w", err)
	}
	if err := c.post(ctx, leakedPath); err != nil {
		return fmt.Errorf("leaked sweep: %w", err)
	}
	if err := c.post(ctx, runtimePath); err != nil {
		return fmt.Errorf("runtime sweep: %w", err)
	}
	if err := c.post(ctx, staleStartingPath); err != nil {
		return fmt.Errorf("stale starting sweep: %w", err)
	}
	return nil
}

// SweepInterval reads CLOUD_CSC_RECONCILE_TICK_SEC; values <=0 fall back to 300s.
func SweepInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CLOUD_CSC_RECONCILE_TICK_SEC"))
	if raw == "" {
		return 300 * time.Second
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 300 * time.Second
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
			Worker:    sweepWorker,
		})
	}
}

// StartSweepTicker runs the orphan+leaked sweep on interval until ctx is cancelled.
func StartSweepTicker(ctx context.Context, client SweepClient, interval time.Duration) {
	if client == nil {
		client = NewTaskCloudSweepClient()
	}
	if interval <= 0 {
		interval = 300 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := SweepOnce(ctx, client); err != nil {
			log.Printf("[cloud_csc_reconcile] sweep: %v", err)
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

// Run starts health HTTP and the sweep ticker (no Kafka).
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

	client := NewTaskCloudSweepClient()
	interval := SweepInterval()
	go StartSweepTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=%s tick=%s", serviceName, addr, sweepWorker, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	<-ctx.Done()
	server.WaitHealthShutdown(httpDone)
	log.Printf("[%s] shutdown", serviceName)
}
