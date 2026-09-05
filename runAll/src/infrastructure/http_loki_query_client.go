package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"runAll/src/domain"
)

// HTTPLokiQueryClient queries a remote Loki HTTP API.
type HTTPLokiQueryClient struct {
	BaseURL string
	Client  *http.Client
}

func NewHTTPLokiQueryClient(baseURL string) *HTTPLokiQueryClient {
	return &HTTPLokiQueryClient{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy: nil, // never inherit shell HTTP(S)_PROXY — Loki is local/infra
			},
			Timeout: 15 * time.Second,
		},
	}
}

func (c *HTTPLokiQueryClient) httpClient() *http.Client {
	if c != nil && c.Client != nil {
		return c.Client
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: nil, // never inherit shell HTTP(S)_PROXY — Loki is local/infra
		},
		Timeout: 15 * time.Second,
	}
}

func (c *HTTPLokiQueryClient) Ready(ctx context.Context) error {
	if c == nil || c.BaseURL == "" {
		return fmt.Errorf("loki url not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/ready", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki ready status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *HTTPLokiQueryClient) CountLogsByJob(ctx context.Context, traceID string, since time.Duration) (map[string]int, error) {
	if c == nil || c.BaseURL == "" {
		return nil, fmt.Errorf("loki url not configured")
	}
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return nil, fmt.Errorf("trace_id is required")
	}
	if since <= 0 {
		since = 10 * time.Minute
	}

	// Instant query: count log lines per job label containing the probe trace id.
	expr := fmt.Sprintf(`sum by (job) (count_over_time({job=~".+"}|~%q[%ds]))`, traceID, int(since.Seconds()))
	end := time.Now()
	q := url.Values{}
	q.Set("query", expr)
	q.Set("time", strconv.FormatInt(end.UnixNano(), 10))

	reqURL := c.BaseURL + "/loki/api/v1/query?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("loki query status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload lokiQueryResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode loki response: %w", err)
	}
	if payload.Status != "success" {
		return nil, fmt.Errorf("loki query failed: %s", payload.Status)
	}

	out := make(map[string]int)
	for _, stream := range payload.Data.Result {
		job := strings.TrimSpace(stream.Metric["job"])
		if job == "" {
			continue
		}
		count := 0
		if len(stream.Value) >= 2 {
			switch v := stream.Value[1].(type) {
			case string:
				if n, err := strconv.Atoi(v); err == nil {
					count = n
				}
			case float64:
				count = int(v)
			}
		}
		if count > 0 {
			out[job] = count
		}
	}
	return out, nil
}

type lokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

var _ domain.LokiQueryClient = (*HTTPLokiQueryClient)(nil)
