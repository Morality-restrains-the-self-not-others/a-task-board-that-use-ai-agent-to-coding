package wechatmpegress

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

const MaxBodyBytes = 1 << 20

// AllowedHost is the only upstream host this egress may dial.
const AllowedHost = "api.weixin.qq.com"

// ForwardRequest is the internal JSON body from taskAuth.
type ForwardRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   string `json:"body,omitempty"`
}

// Server forwards allowlisted WeChat MP API calls using the host egress IP.
type Server struct {
	Secret string
	Client *http.Client
	Now    func() time.Time
}

func (s *Server) upstream() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return tracelog.DirectClient(15 * time.Second)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /internal/wechat-mp/forward", s.handleForward)
	return mux
}

func (s *Server) handleForward(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(s.Secret) == "" || r.Header.Get("X-Internal-Secret") != s.Secret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req ForwardRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusBadRequest)
		return
	}
	target, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || target.Scheme != "https" || !strings.EqualFold(target.Hostname(), AllowedHost) {
		http.Error(w, "url host not allowed", http.StatusBadRequest)
		return
	}
	var bodyReader io.Reader
	if method == http.MethodPost {
		bodyReader = bytes.NewReader([]byte(req.Body))
	}
	outReq, err := http.NewRequestWithContext(r.Context(), method, target.String(), bodyReader)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if method == http.MethodPost {
		outReq.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.upstream().Do(outReq)
	if err != nil {
		http.Error(w, "upstream dial failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes))
	if err != nil {
		http.Error(w, "upstream read failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Upstream-Status", http.StatusText(resp.StatusCode))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(respBody),
	})
}
