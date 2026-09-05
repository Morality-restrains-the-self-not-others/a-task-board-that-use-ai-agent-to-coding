package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

type wechatReceiverStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ensureWechatProfitSharingReceiver 默认 no-op，避免单测打到本机 :8004。
// main() 在 loadConfig 之后接到 live 实现。
var ensureWechatProfitSharingReceiver = func(string) wechatReceiverStatus {
	return wechatReceiverStatus{}
}

// deleteWechatProfitSharingReceiver 默认 no-op，避免单测打到本机 :8004。
// 取消分账资格后 best-effort 删除微信分账接收方（OPT-20260822-019）。
var deleteWechatProfitSharingReceiver = func(string) wechatReceiverStatus {
	return wechatReceiverStatus{Status: "deleted"}
}

func ensureWechatProfitSharingReceiverLive(userID string) wechatReceiverStatus {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return wechatReceiverStatus{Status: "failed", Reason: "empty_user_id"}
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BillServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8004" // 同节点调用，拆分时改 ${subdomains.xxx}
	}
	payload, err := json.Marshal(map[string]string{
		"user_id":    userID,
		"legal_name": lookupLegalName(userID),
	})
	if err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "marshal"}
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/taskbill/profit-sharing/receivers/ensure/", bytes.NewReader(payload))
	if err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "request"}
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.BillInternalSecret); sec != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "bill_unreachable"}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 400 {
		return wechatReceiverStatus{Status: "failed", Reason: "bill_status"}
	}
	var out struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "bill_json"}
	}
	return wechatReceiverStatus{Status: strings.TrimSpace(out.Status), Reason: strings.TrimSpace(out.Reason)}
}

func deleteWechatProfitSharingReceiverLive(userID string) wechatReceiverStatus {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return wechatReceiverStatus{Status: "failed", Reason: "empty_user_id"}
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BillServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8004" // 同节点调用，拆分时改 ${subdomains.xxx}
	}
	payload, err := json.Marshal(map[string]string{"user_id": userID})
	if err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "marshal"}
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/taskbill/profit-sharing/receivers/delete/", bytes.NewReader(payload))
	if err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "request"}
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.BillInternalSecret); sec != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "bill_unreachable"}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 400 {
		return wechatReceiverStatus{Status: "failed", Reason: "bill_status"}
	}
	var out struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return wechatReceiverStatus{Status: "failed", Reason: "bill_json"}
	}
	return wechatReceiverStatus{Status: strings.TrimSpace(out.Status), Reason: strings.TrimSpace(out.Reason)}
}
