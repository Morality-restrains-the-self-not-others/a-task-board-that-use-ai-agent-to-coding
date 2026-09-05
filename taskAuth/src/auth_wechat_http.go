package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

// wechatMPEgress holds optional Host-sh forwarder for cgi-bin calls (ADR-0059).
var wechatMPEgressBaseURL string
var wechatMPEgressSecret string

// wechatHTTPGet 可被测试替换以 mock 微信 API（默认直连或经 mpEgress，不走环境 Proxy）。
var wechatHTTPGet = httpGet

func httpGet(urlStr string) ([]byte, error) {
	if useWechatMPEgress(urlStr) {
		return wechatMPEgressDo(http.MethodGet, urlStr, nil)
	}
	resp, err := tracelog.DirectClient(10 * time.Second).Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func wechatHTTPPostJSONLive(urlStr string, payload []byte) ([]byte, error) {
	if useWechatMPEgress(urlStr) {
		return wechatMPEgressDo(http.MethodPost, urlStr, payload)
	}
	req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := tracelog.DirectClient(10 * time.Second).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func useWechatMPEgress(urlStr string) bool {
	if strings.TrimSpace(wechatMPEgressBaseURL) == "" || strings.TrimSpace(wechatMPEgressSecret) == "" {
		return false
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Hostname(), "api.weixin.qq.com")
}

func wechatMPEgressDo(method, targetURL string, payload []byte) ([]byte, error) {
	bodyObj := map[string]string{
		"method": method,
		"url":    targetURL,
	}
	if len(payload) > 0 {
		bodyObj["body"] = string(payload)
	}
	raw, err := json.Marshal(bodyObj)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(wechatMPEgressBaseURL, "/") + "/internal/wechat-mp/forward"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", wechatMPEgressSecret)
	resp, err := tracelog.DirectClient(20 * time.Second).Do(req)
	if err != nil {
		return nil, fmt.Errorf("wechat mp egress: %w", err)
	}
	defer resp.Body.Close()
	respRaw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wechat mp egress status %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	var parsed struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(respRaw, &parsed); err != nil {
		return nil, fmt.Errorf("wechat mp egress decode: %w", err)
	}
	return []byte(parsed.Body), nil
}
