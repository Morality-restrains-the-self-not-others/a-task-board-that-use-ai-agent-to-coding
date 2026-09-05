package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// userId cookie 签名（OPT-20260807-004）：裸用户 ID 可被任何能设置同域 cookie 的侧
// （子域 XSS 等）伪造；即便 OPT-20260807-002 已要求活 token 行校验，攻击者仍可伪造
// `userId=<近期登录过的用户ID>` 冒充该用户。activate-session 现写入
// `userId=<id>.<unix_ts>.<hex hmac-sha256(secret, id|ts)>`，解析侧验签 + 时效（7 天），
// 拒绝未签名 / 签名不符 / 过期的旧值。
//
// HMAC 密钥复用 ssoBridgeSecret（cfg.SSOJwtSecret 真源 conf/core/sso/config.yaml，
// 与 SSO bridge 同域同进程，轮换见 OPT-20260806-062；dev 环境回退本地默认值）。
//
// 兼容性：新格式 cookie 上线后，旧的裸 ID cookie 会被拒绝 —— 用户重新登录一次即可
// （activate-session 在每个登录流都会重写签名 cookie）；插件侧只消费 `token` cookie
// 与 profile API（service-worker.js tryAutoDetectTokenFromCookie），不受影响。

const userIDCookieSigTTLSec = 7 * 24 * 3600 // 签名时效：cookie 本身 30 天，签名保证 7 天内有效
const userIDCookieSkewSec = 300             // 容忍 ±5 分钟时钟偏差

// signUserIDCookieValueAt 以给定时间戳签发 userId cookie 值（ts 参数供测试注入）。
func signUserIDCookieValueAt(userID string, ts int64) (string, error) {
	if strings.TrimSpace(userID) == "" || strings.ContainsAny(userID, ".") {
		return "", fmt.Errorf("invalid userID for cookie signing")
	}
	secret := []byte(ssoBridgeSecret())
	mac := hmac.New(sha256.New, secret)
	io.WriteString(mac, userID)
	io.WriteString(mac, "|")
	io.WriteString(mac, strconv.FormatInt(ts, 10))
	return userID + "." + strconv.FormatInt(ts, 10) + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func signUserIDCookieValue(userID string) (string, error) {
	return signUserIDCookieValueAt(userID, time.Now().Unix())
}

// parseUserIDCookieValue 验签并提取用户 ID；任何失败（未签名 / 篡改 / 过期）都返回
// 错误，调用方按「该 cookie 不可信」处理（不降级为裸 ID 解析 —— 那正是本项修复的漏洞）。
func parseUserIDCookieValue(v string) (string, error) {
	v = strings.TrimSpace(v)
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("malformed signed userId cookie")
	}
	id, tsStr, sigHex := parts[0], parts[1], parts[2]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("bad ts in signed userId cookie")
	}
	now := time.Now().Unix()
	if now-ts > userIDCookieSigTTLSec || ts-now > userIDCookieSkewSec {
		return "", fmt.Errorf("signed userId cookie stale")
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", fmt.Errorf("bad signature encoding in userId cookie")
	}
	secret := []byte(ssoBridgeSecret())
	mac := hmac.New(sha256.New, secret)
	io.WriteString(mac, id)
	io.WriteString(mac, "|")
	io.WriteString(mac, tsStr)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return "", fmt.Errorf("userId cookie signature mismatch")
	}
	return id, nil
}
