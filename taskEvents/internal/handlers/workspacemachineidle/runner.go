package workspacemachineidle

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

const recyclePath = "/api/internal/cloud/compute/recycle-idle-machines/"

// RecycleClient posts to taskCloudService internal recycle API.
type RecycleClient interface {
	RecycleOnce(ctx context.Context) error
}

// TaskCloudRecycleClient calls taskCloudService recycle-idle-machines.
type TaskCloudRecycleClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskCloudRecycleClient builds a client from environment.
func NewTaskCloudRecycleClient() *TaskCloudRecycleClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8018"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET"))
	return &TaskCloudRecycleClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// RecycleOnce triggers one idle-machine recycle sweep.
func RecycleOnce(ctx context.Context, client RecycleClient) error {
	if client == nil {
		client = NewTaskCloudRecycleClient()
	}
	return client.RecycleOnce(ctx)
}

func (c *TaskCloudRecycleClient) RecycleOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskCloudRecycleClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + recyclePath
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
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s HTTP %d: %s", recyclePath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// RecycleTickInterval reads IDLE_RECYCLE_TICK_SEC; values <=0 fall back to 60s.
func RecycleTickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("IDLE_RECYCLE_TICK_SEC"))
	if raw == "" {
		return 60 * time.Second
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 60 * time.Second
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
			Worker:    "recycle_idle_nodes",
		})
	}
}

// StartRecycleTicker runs recycle on interval until ctx is cancelled.
func StartRecycleTicker(ctx context.Context, client RecycleClient, interval time.Duration) {
	if client == nil {
		client = NewTaskCloudRecycleClient()
	}
	if interval <= 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := RecycleOnce(ctx, client); err != nil {
			log.Printf("[workspace_machine_idle] recycle: %v", err)
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

// Run starts health HTTP and the recycle ticker (no Kafka).
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

	client := NewTaskCloudRecycleClient()
	interval := RecycleTickInterval()
	go StartRecycleTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=recycle_idle_nodes tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}
