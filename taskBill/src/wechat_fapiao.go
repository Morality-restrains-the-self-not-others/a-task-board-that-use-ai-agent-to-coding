package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
)

// WechatPay-SDK-OK: no services/fapiao for /v3/new-tax-control-fapiao/fapiao-applications

type wechatFapiaoIssueRequest struct {
	OrderID        int64
	FapiaoApplyID  string
	FapiaoID       string
	TotalAmountFen int64
	Buyer          invoiceBuyerInput
	Purpose        string
}

type wechatFapiaoIssueResult struct {
	ApplyID  string
	FapiaoID string
}

type wechatFapiaoReverseRequest struct {
	FapiaoApplyID string
	FapiaoID      string
	FapiaoNumber  string
	ReverseReason string
}

var issueWechatFapiaoFn = issueWechatFapiaoImpl
var reverseWechatFapiaoFn = reverseWechatFapiaoImpl

func fapiaoConfOrDefault() wechatFapiaoConfig {
	cfg := wechatCfg.Fapiao
	if strings.TrimSpace(cfg.TaxCode) == "" {
		cfg.TaxCode = "3040201990000000000"
	}
	if cfg.TaxRate <= 0 {
		cfg.TaxRate = 600
	}
	if strings.TrimSpace(cfg.GoodsName) == "" {
		cfg.GoodsName = "平台资源服务"
	}
	if strings.TrimSpace(cfg.GoodsCategory) == "" {
		cfg.GoodsCategory = "现代服务"
	}
	return cfg
}

func buildWechatFapiaoPayload(req wechatFapiaoIssueRequest) map[string]interface{} {
	fc := fapiaoConfOrDefault()
	buyer := map[string]interface{}{
		"type": req.Buyer.Type,
		"name": req.Buyer.Name,
	}
	if req.Buyer.TaxpayerID != "" {
		buyer["taxpayer_id"] = req.Buyer.TaxpayerID
	}
	if req.Buyer.Address != "" {
		buyer["address"] = req.Buyer.Address
	}
	if req.Buyer.Telephone != "" {
		buyer["telephone"] = req.Buyer.Telephone
	}
	if req.Buyer.BankName != "" {
		buyer["bank_name"] = req.Buyer.BankName
	}
	if req.Buyer.BankAccount != "" {
		buyer["bank_account"] = req.Buyer.BankAccount
	}
	item := map[string]interface{}{
		"tax_code":        fc.TaxCode,
		"goods_category":  fc.GoodsCategory,
		"goods_name":      fc.GoodsName,
		"quantity":        100000000,
		"total_amount":    req.TotalAmountFen,
		"tax_rate":        fc.TaxRate,
		"tax_prefer_mark": "NO_FAVORABLE",
		"discount":        false,
	}
	return map[string]interface{}{
		"scene":             "WITH_WECHATPAY",
		"fapiao_apply_id":   req.FapiaoApplyID,
		"buyer_information": buyer,
		"fapiao_information": []map[string]interface{}{
			{
				"fapiao_id":    req.FapiaoID,
				"total_amount": req.TotalAmountFen,
				"need_list":    false,
				"items":        []map[string]interface{}{item},
			},
		},
	}
}

func issueWechatFapiaoImpl(ctx context.Context, req wechatFapiaoIssueRequest) (wechatFapiaoIssueResult, error) {
	if !wechatCfg.Fapiao.Enabled {
		return wechatFapiaoIssueResult{}, fmt.Errorf("电子发票未开通")
	}
	if strings.TrimSpace(req.FapiaoApplyID) == "" || req.TotalAmountFen <= 0 {
		return wechatFapiaoIssueResult{}, fmt.Errorf("开票参数不完整")
	}
	payload := buildWechatFapiaoPayload(req)
	slog.InfoContext(ctx, "wechat_fapiao_issue_request",
		"level", "info",
		"order_id", formatID(req.OrderID),
		"fapiao_id", req.FapiaoID,
		"purpose", req.Purpose,
		"amount_fen", req.TotalAmountFen,
	)
	status, body, err := wechatV3PostWithResp(ctx, "/v3/new-tax-control-fapiao/fapiao-applications", payload)
	if err != nil {
		return wechatFapiaoIssueResult{}, err
	}
	if status == 202 || (status >= 200 && status < 300) {
		return wechatFapiaoIssueResult{ApplyID: req.FapiaoApplyID, FapiaoID: req.FapiaoID}, nil
	}
	if status == 400 && strings.Contains(string(body), "RESOURCE_ALREADY_EXISTS") {
		slog.InfoContext(ctx, "wechat_fapiao_already_exists",
			"level", "info",
			"fapiao_apply_id_len", len(req.FapiaoApplyID),
		)
		return wechatFapiaoIssueResult{ApplyID: req.FapiaoApplyID, FapiaoID: req.FapiaoID}, nil
	}
	return wechatFapiaoIssueResult{}, fmt.Errorf("wechat fapiao issue HTTP %d: %s", status, truncateBytes(body, 300))
}

func reverseWechatFapiaoImpl(ctx context.Context, req wechatFapiaoReverseRequest) error {
	if !wechatCfg.Fapiao.Enabled {
		return fmt.Errorf("电子发票未开通")
	}
	path := fmt.Sprintf("/v3/new-tax-control-fapiao/fapiao-applications/%s/reverse", url.PathEscape(req.FapiaoApplyID))
	info := map[string]interface{}{"fapiao_id": req.FapiaoID}
	if strings.TrimSpace(req.FapiaoNumber) != "" {
		info["fapiao_number"] = req.FapiaoNumber
	}
	payload := map[string]interface{}{
		"reverse_reason":     firstNonEmpty(req.ReverseReason, "订单退款全额红冲"),
		"fapiao_information": []map[string]interface{}{info},
	}
	slog.InfoContext(ctx, "wechat_fapiao_reverse_request",
		"level", "info",
		"fapiao_id", req.FapiaoID,
	)
	status, body, err := wechatV3PostWithResp(ctx, path, payload)
	if err != nil {
		return err
	}
	if status == 202 || (status >= 200 && status < 300) {
		return nil
	}
	return fmt.Errorf("wechat fapiao reverse HTTP %d: %s", status, truncateBytes(body, 300))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func parseFapiaoNotifyResource(raw []byte) (eventType, applyID, fapiaoID, fapiaoNumber, status string) {
	var envelope struct {
		EventType string          `json:"event_type"`
		Resource  json.RawMessage `json:"resource"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.EventType != "" {
		eventType = envelope.EventType
		raw = envelope.Resource
		if len(raw) == 0 {
			raw = []byte("{}")
		}
	}
	var res struct {
		FapiaoApplyID     string `json:"fapiao_apply_id"`
		FapiaoInformation []struct {
			FapiaoID     string `json:"fapiao_id"`
			FapiaoNumber string `json:"fapiao_number"`
			Status       string `json:"status"`
		} `json:"fapiao_information"`
	}
	_ = json.Unmarshal(raw, &res)
	applyID = strings.TrimSpace(res.FapiaoApplyID)
	if len(res.FapiaoInformation) > 0 {
		fapiaoID = strings.TrimSpace(res.FapiaoInformation[0].FapiaoID)
		fapiaoNumber = strings.TrimSpace(res.FapiaoInformation[0].FapiaoNumber)
		status = strings.TrimSpace(res.FapiaoInformation[0].Status)
	}
	return eventType, applyID, fapiaoID, fapiaoNumber, status
}
