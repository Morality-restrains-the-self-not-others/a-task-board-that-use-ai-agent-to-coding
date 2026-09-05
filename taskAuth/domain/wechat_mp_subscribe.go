package domain

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
)

const WechatMPAppKey = "mp"

func WechatMPSignature(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		cleaned = append(cleaned, strings.TrimSpace(p))
	}
	sort.Strings(cleaned)
	sum := sha1.Sum([]byte(strings.Join(cleaned, "")))
	return hex.EncodeToString(sum[:])
}

// WechatMPCheckSignature reports whether timestamp+nonce+token match signature (WeChat server URL verify).
func WechatMPCheckSignature(token, timestamp, nonce, signature string) bool {
	token = strings.TrimSpace(token)
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	signature = strings.TrimSpace(signature)
	if token == "" || timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	return strings.EqualFold(WechatMPSignature(token, timestamp, nonce), signature)
}

// WechatMPCheckMsgSignature verifies encrypted POST msg_signature (token+timestamp+nonce+Encrypt).
func WechatMPCheckMsgSignature(token, timestamp, nonce, encrypt, signature string) bool {
	token = strings.TrimSpace(token)
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	encrypt = strings.TrimSpace(encrypt)
	signature = strings.TrimSpace(signature)
	if token == "" || timestamp == "" || nonce == "" || encrypt == "" || signature == "" {
		return false
	}
	return strings.EqualFold(WechatMPSignature(token, timestamp, nonce, encrypt), signature)
}

// WechatMPIsSubscribeEvent is true for first follow (including scan-subscribe).
func WechatMPIsSubscribeEvent(msgType, event string) bool {
	return strings.EqualFold(strings.TrimSpace(msgType), "event") &&
		strings.EqualFold(strings.TrimSpace(event), "subscribe")
}

// WechatMPIsScanEvent is true when an already-followed user scans a parametric QR.
func WechatMPIsScanEvent(msgType, event string) bool {
	return strings.EqualFold(strings.TrimSpace(msgType), "event") &&
		strings.EqualFold(strings.TrimSpace(event), "scan")
}

// WechatMPIsFollowScanEvent is subscribe (first follow) or SCAN (already followed).
func WechatMPIsFollowScanEvent(msgType, event string) bool {
	return WechatMPIsSubscribeEvent(msgType, event) || WechatMPIsScanEvent(msgType, event)
}

const wechatMPQRScenePrefix = "qrscene_"

// WechatMPSceneTempID strips the qrscene_ prefix from EventKey. SCAN events use the
// raw scene_str; subscribe-from-QR uses qrscene_<scene_str>. Empty means no ticket path.
func WechatMPSceneTempID(eventKey string) string {
	k := strings.TrimSpace(eventKey)
	if k == "" {
		return ""
	}
	if len(k) >= len(wechatMPQRScenePrefix) && strings.EqualFold(k[:len(wechatMPQRScenePrefix)], wechatMPQRScenePrefix) {
		return strings.TrimSpace(k[len(wechatMPQRScenePrefix):])
	}
	return k
}

// WechatMPIsUnsubscribeEvent is true when the user unfollows the official account.
func WechatMPIsUnsubscribeEvent(msgType, event string) bool {
	return strings.EqualFold(strings.TrimSpace(msgType), "event") &&
		strings.EqualFold(strings.TrimSpace(event), "unsubscribe")
}

// WechatMPSubscribeCreatesUser is always false: follow must lock an existing identity via unionId.
func WechatMPSubscribeCreatesUser() bool {
	return false
}
