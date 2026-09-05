package commandhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"taskEvents/domain"
	"tracelog"
)

// Client implements domain.DomainCommandPort via Django internal HTTP.
type Client struct {
	BaseURL string
	Secret  string
	HTTP    *http.Client
}

// NewClient builds a command HTTP client.
func NewClient(baseURL, secret string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Secret:  secret,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Dispatch POSTs the domain event payload to Django.
func (c *Client) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	url := c.BaseURL + cmd.Path
	body := map[string]interface{}{
		"event_type": cmd.EventType,
		"data":       json.RawMessage(cmd.Envelope.Data),
		"key":        cmd.Envelope.Key,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return domain.DispatchPermanent, err
	}
	req.Header.Set("Content-Type", "application/json")
	// 出站统一走 ApplyOutboundHeaders：带 X-Trace-Id + X-Parent-Span-Id + traceparent。
	// 仅发 X-Trace-Id 会被下游 shareLib RejectTraceIdOnlyHTTP 以 400 拒绝（评论 CSC bootstrap
	// 曾卡 Starting，见 OPT-20260817-021），这里保证 ctx 带 span 关联后再应用。
	if corr := tracelog.CorrelationFromContext(ctx); corr.SpanID == "" {
		corr.SpanID = tracelog.NewSpanID()
		if corr.TraceID == "" {
			corr.TraceID = tracelog.NewTraceID()
		}
		ctx = tracelog.ContextWithCorrelation(ctx, corr)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if c.Secret != "" {
		req.Header.Set("X-TaskEvents-Internal-Secret", c.Secret)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return domain.DispatchRetryable, err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return domain.DispatchSuccess, nil
	case resp.StatusCode >= 500:
		return domain.DispatchRetryable, fmt.Errorf("status %d", resp.StatusCode)
	default:
		return domain.DispatchPermanent, fmt.Errorf("status %d", resp.StatusCode)
	}
}
