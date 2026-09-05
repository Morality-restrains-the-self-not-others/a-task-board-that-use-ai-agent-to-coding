package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"tracelog"
)

const (
	smsKindVerification  = "verification"
	smsKindPasswordReset = "password_reset"
)

func smsEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	// 环境变量未设时，回退本服务 sync 片段 sms.yaml（loadConfig 已载入）
	for _, k := range keys {
		if v := smsConfigValue(k); v != "" {
			return v
		}
	}
	return ""
}

func smsConfigValue(key string) string {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "SMS_PROVIDER", "TASKAUTH_SMS_PROVIDER":
		return strings.TrimSpace(cfg.SMSProvider)
	case "SMS_ALIYUN_ACCESS_KEY_ID":
		return strings.TrimSpace(cfg.SMSAliyunAccessKeyID)
	case "SMS_ALIYUN_ACCESS_KEY_SECRET":
		return strings.TrimSpace(cfg.SMSAliyunAccessKeySecret)
	case "SMS_ALIYUN_REGION_ID":
		return strings.TrimSpace(cfg.SMSAliyunRegionID)
	case "SMS_ALIYUN_SIGN_NAME":
		return strings.TrimSpace(cfg.SMSAliyunSignName)
	case "SMS_ALIYUN_TEMPLATE_CODE_NOTIFICATION":
		return strings.TrimSpace(cfg.SMSAliyunTemplateCodeNotification)
	case "SMS_TEMPLATE_CODE_VERIFICATION":
		return strings.TrimSpace(cfg.SMSTemplateCodeVerification)
	case "SMS_TEMPLATE_CODE_PASSWORD_RESET":
		return strings.TrimSpace(cfg.SMSTemplateCodePasswordReset)
	case "SMS_TEMPLATE_CODE_NOTIFICATION":
		return strings.TrimSpace(cfg.SMSTemplateCodeNotification)
	default:
		return ""
	}
}

func smsTemplateCode(kind string) string {
	switch kind {
	case smsKindPasswordReset:
		if v := smsEnv("SMS_TEMPLATE_CODE_PASSWORD_RESET"); v != "" {
			return v
		}
	default:
		if v := smsEnv("SMS_TEMPLATE_CODE_VERIFICATION"); v != "" {
			return v
		}
	}
	if v := smsEnv("SMS_TEMPLATE_CODE_NOTIFICATION", "SMS_ALIYUN_TEMPLATE_CODE_NOTIFICATION", "SMS_TENCENT_TEMPLATE_ID_NOTIFICATION"); v != "" {
		return v
	}
	return ""
}

func sendCloudSMS(ctx context.Context, provider, e164Phone, code, kind string) smsSendResult {
	templateCode := smsTemplateCode(kind)
	params := map[string]string{"code": code}
	switch provider {
	case "aliyun":
		return sendAliyunSMS(ctx, e164Phone, templateCode, params)
	case "tencent", "tencentcloud":
		return sendTencentSMS(ctx, e164Phone, templateCode, params)
	default:
		return smsSendResult{
			Success:      false,
			Provider:     provider,
			ErrorCode:    "unsupported_provider",
			ErrorMessage: "不支持的短信渠道",
		}
	}
}

func sendAliyunSMS(ctx context.Context, phone, templateCode string, templateParams map[string]string) smsSendResult {
	accessKeyID := smsEnv("SMS_ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := smsEnv("SMS_ALIYUN_ACCESS_KEY_SECRET")
	signName := smsEnv("SMS_ALIYUN_SIGN_NAME")
	region := smsEnv("SMS_ALIYUN_REGION_ID")
	if region == "" {
		region = "cn-hangzhou"
	}
	if accessKeyID == "" || accessKeySecret == "" || signName == "" || templateCode == "" {
		log.Printf("[taskAuth-sms] aliyun config incomplete trace_id=%s", tracelog.TraceIDFromContext(ctx))
		return smsSendResult{
			Success:      false,
			Provider:     "aliyun",
			ErrorCode:    "config_incomplete",
			ErrorMessage: "短信服务配置不完整",
		}
	}
	paramJSON, err := json.Marshal(templateParams)
	if err != nil {
		return smsSendResult{
			Success:      false,
			Provider:     "aliyun",
			ErrorCode:    "marshal_error",
			ErrorMessage: "短信参数编码失败",
		}
	}
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	query := map[string]string{
		"AccessKeyId":      accessKeyID,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"RegionId":         region,
		"SignName":         signName,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"SignatureVersion": "1.0",
		"TemplateCode":     templateCode,
		"TemplateParam":    string(paramJSON),
		"Timestamp":        ts,
		"Version":          "2017-05-25",
	}
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for i, k := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(percentEncode(k))
		canonical.WriteByte('=')
		canonical.WriteString(percentEncode(query[k]))
	}
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canonical.String())
	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	reqURL := "https://dysmsapi.aliyuncs.com/?" + canonical.String() + "&Signature=" + percentEncode(signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return smsSendResult{
			Success:      false,
			Provider:     "aliyun",
			ErrorCode:    "request_build_error",
			ErrorMessage: "短信请求构建失败",
		}
	}
	resp, err := tracelog.DirectClient(30 * time.Second).Do(req)
	if err != nil {
		log.Printf("[taskAuth-sms] aliyun http error: %v trace_id=%s", err, tracelog.TraceIDFromContext(ctx))
		return smsSendResult{
			Success:      false,
			Provider:     "aliyun",
			ErrorCode:    "http_error",
			ErrorMessage: "短信服务暂时不可用",
		}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed map[string]interface{}
	_ = json.Unmarshal(body, &parsed)
	reqID, _ := parsed["RequestId"].(string)
	codeStr, _ := parsed["Code"].(string)
	msgStr := interfaceString(parsed["Message"])
	if strings.EqualFold(codeStr, "OK") {
		log.Printf("[taskAuth-sms] aliyun ok phone=%s request_id=%s trace_id=%s", phone, reqID, tracelog.TraceIDFromContext(ctx))
		return smsSendResult{Success: true, Provider: "aliyun", RequestID: reqID}
	}
	log.Printf("[taskAuth-sms] aliyun failed code=%v msg=%v request_id=%s trace_id=%s",
		codeStr, msgStr, reqID, tracelog.TraceIDFromContext(ctx))
	return smsSendResult{
		Success:      false,
		Provider:     "aliyun",
		RequestID:    reqID,
		ErrorCode:    codeStr,
		ErrorMessage: msgStr,
	}
}

func interfaceString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func percentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func sendTencentSMS(ctx context.Context, phone, templateID string, templateParams map[string]string) smsSendResult {
	secretID := smsEnv("SMS_TENCENT_SECRET_ID")
	secretKey := smsEnv("SMS_TENCENT_SECRET_KEY")
	smsSdkAppID := smsEnv("SMS_TENCENT_APP_ID")
	signName := smsEnv("SMS_TENCENT_SIGN_NAME")
	region := smsEnv("SMS_TENCENT_REGION")
	if region == "" {
		region = "ap-guangzhou"
	}
	if secretID == "" || secretKey == "" || smsSdkAppID == "" || signName == "" || templateID == "" {
		log.Printf("[taskAuth-sms] tencent config incomplete trace_id=%s", tracelog.TraceIDFromContext(ctx))
		return smsSendResult{
			Success:      false,
			Provider:     "tencent",
			ErrorCode:    "config_incomplete",
			ErrorMessage: "短信服务配置不完整",
		}
	}
	codeParam := templateParams["code"]
	payload := map[string]interface{}{
		"PhoneNumberSet":   []string{phone},
		"SmsSdkAppId":      smsSdkAppID,
		"SignName":         signName,
		"TemplateId":       templateID,
		"TemplateParamSet": []string{codeParam},
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return smsSendResult{
			Success:      false,
			Provider:     "tencent",
			ErrorCode:    "marshal_error",
			ErrorMessage: "短信参数编码失败",
		}
	}
	host := "sms.tencentcloudapi.com"
	service := "sms"
	action := "SendSms"
	version := "2021-01-11"
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	canonicalHeaders := fmt.Sprintf("content-type:application/json; charset=utf-8\nhost:%s\nx-tc-action:%s\n", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	hashedPayload := sha256Hex(bodyBytes)
	canonicalRequest := strings.Join([]string{
		"POST",
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	}, "\n")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		fmt.Sprintf("%d", timestamp),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	secretDate := hmacSHA256([]byte("TC3"+secretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	authorization := fmt.Sprintf(
		"TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		secretID, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+host, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return smsSendResult{
			Success:      false,
			Provider:     "tencent",
			ErrorCode:    "request_build_error",
			ErrorMessage: "短信请求构建失败",
		}
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Region", region)
	req.Header.Set("Authorization", authorization)

	resp, err := tracelog.DirectClient(30 * time.Second).Do(req)
	if err != nil {
		log.Printf("[taskAuth-sms] tencent http error: %v trace_id=%s", err, tracelog.TraceIDFromContext(ctx))
		return smsSendResult{
			Success:      false,
			Provider:     "tencent",
			ErrorCode:    "http_error",
			ErrorMessage: "短信服务暂时不可用",
		}
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed map[string]interface{}
	_ = json.Unmarshal(respBody, &parsed)
	response, _ := parsed["Response"].(map[string]interface{})
	reqID, _ := response["RequestId"].(string)
	if errObj, ok := response["Error"].(map[string]interface{}); ok && errObj != nil {
		errCode := interfaceString(errObj["Code"])
		errMsg := interfaceString(errObj["Message"])
		log.Printf("[taskAuth-sms] tencent failed code=%s msg=%s request_id=%s trace_id=%s",
			errCode, errMsg, reqID, tracelog.TraceIDFromContext(ctx))
		return smsSendResult{
			Success:      false,
			Provider:     "tencent",
			RequestID:    reqID,
			ErrorCode:    errCode,
			ErrorMessage: errMsg,
		}
	}
	if statuses, ok := response["SendStatusSet"].([]interface{}); ok {
		for _, item := range statuses {
			st, _ := item.(map[string]interface{})
			if st == nil {
				continue
			}
			code := interfaceString(st["Code"])
			if code == "" || strings.EqualFold(code, "Ok") {
				continue
			}
			msg := interfaceString(st["Message"])
			sid := interfaceString(st["SerialNo"])
			if sid == "" {
				sid = reqID
			}
			log.Printf("[taskAuth-sms] tencent status failed code=%s msg=%s request_id=%s serial=%s trace_id=%s",
				code, msg, reqID, sid, tracelog.TraceIDFromContext(ctx))
			return smsSendResult{
				Success:      false,
				Provider:     "tencent",
				RequestID:    sid,
				ErrorCode:    code,
				ErrorMessage: msg,
			}
		}
	}
	log.Printf("[taskAuth-sms] tencent ok phone=%s request_id=%s trace_id=%s", phone, reqID, tracelog.TraceIDFromContext(ctx))
	return smsSendResult{Success: true, Provider: "tencent", RequestID: reqID}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}
