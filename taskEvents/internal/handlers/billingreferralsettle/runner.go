package billingreferralsettle

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

const settleDuePath = "/api/internal/taskbill/referral/settle-due/"

// SettleClient posts to taskBill internal referral settle-due API.
type SettleClient interface {
	SettleOnce(ctx context.Context) error
}

// TaskBillSettleClient calls taskBill referral settle-due.
type TaskBillSettleClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

// NewTaskBillSettleClient builds a client from environment.
func NewTaskBillSettleClient() *TaskBillSettleClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_BILL_INTERNAL_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8004"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_BILL_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &TaskBillSettleClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// SettleOnce triggers one due-referral settlement sweep.
func SettleOnce(ctx context.Context, client SettleClient) error {
	if client == nil {
		client = NewTaskBillSettleClient()
	}
	return client.SettleOnce(ctx)
}

func (c *TaskBillSettleClient) SettleOnce(ctx context.Context) error {
	if c == nil {
		c = NewTaskBillSettleClient()
	}
	url := strings.TrimRight(c.BaseURL, "/") + settleDuePath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if c.InternalSecret != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", c.InternalSecret)
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
		return fmt.Errorf("%s HTTP %d: %s", settleDuePath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	// 记录本次扫描结果（best-effort）
	var out map[string]interface{}
	if json.Unmarshal(body, &out) == nil {
		log.Printf("[billing_referral_settle_scan] sweep result: settled=%v status=%v",
			out["settled"], out["status"])
	}
	return nil
}

// SettleTickInterval reads BILLING_REFERRAL_SETTLE_SCAN_TICK_SEC; values <=0 fall back to 1h.
func SettleTickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("BILLING_REFERRAL_SETTLE_SCAN_TICK_SEC"))
	if raw == "" {
		return 1 * time.Hour
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 1 * time.Hour
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
			Worker:    "settle_due",
		})
	}
}

// StartSettleTicker runs the settlement sweep on interval until ctx is cancelled.
func StartSettleTicker(ctx context.Context, client SettleClient, interval time.Duration) {
	if client == nil {
		client = NewTaskBillSettleClient()
	}
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		if err := SettleOnce(ctx, client); err != nil {
			log.Printf("[billing_referral_settle_scan] sweep: %v", err)
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

// Run starts health HTTP and the settlement scan ticker (no Kafka).
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

	client := NewTaskBillSettleClient()
	interval := SettleTickInterval()
	go StartSettleTicker(ctx, client, interval)

	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", timerHealthHandler(serviceName))
	mux.HandleFunc("/api/health/ready", timerHealthHandler(serviceName))

	log.Printf("[%s] health http://%s/api/health/ worker=settle_due tick=%s", serviceName, addr, interval)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}
