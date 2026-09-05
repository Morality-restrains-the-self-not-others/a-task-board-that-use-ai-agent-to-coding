package useraccountdeletionexecute

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

const executeDuePath = "/api/internal/taskauth/account-deletion/execute-due/"

type ExecuteClient interface {
	ExecuteOnce(ctx context.Context) error
}

type TaskAuthExecuteClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

func NewTaskAuthExecuteClient() *TaskAuthExecuteClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL")), "/")
	if base == "" {
		// taskAuth 实际监听 8003（conf/auth/task-auth/config.yaml port: 8003）；旧 8001 无监听会导致 execute-due 扫描 connection refused（OPT-20260824-017）。
		base = "http://127.0.0.1:8003"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &TaskAuthExecuteClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

func ExecuteOnce(ctx context.Context, client ExecuteClient) error {
	if client == nil {
		client = NewTaskAuthExecuteClient()
	}
	return client.ExecuteOnce(ctx)
}

func (c *TaskAuthExecuteClient) ExecuteOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskAuthExecuteClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + executeDuePath
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
		return fmt.Errorf("%s HTTP %d: %s", executeDuePath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out map[string]interface{}
	if json.Unmarshal(body, &out) == nil {
		log.Printf("[user_account_deletion_execute_scan] processed=%v", out["processed"])
	}
	return nil
}

func ExecuteTickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("USER_ACCOUNT_DELETION_EXECUTE_SCAN_TICK_SEC"))
	if raw == "" {
		return 15 * time.Minute
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 15 * time.Minute
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
			Worker:    "execute_due",
		})
	}
}

func StartExecuteTicker(ctx context.Context, client ExecuteClient, interval time.Duration) {
	if client == nil {
		client = NewTaskAuthExecuteClient()
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	run := func() {
		if err := ExecuteOnce(ctx, client); err != nil {
			log.Printf("[user_account_deletion_execute_scan] sweep: %v", err)
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

	client := NewTaskAuthExecuteClient()
	interval := ExecuteTickInterval()
	go StartExecuteTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=execute_due tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}
