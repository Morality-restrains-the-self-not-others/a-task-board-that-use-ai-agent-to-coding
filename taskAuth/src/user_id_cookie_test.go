package main

import (
	"strings"
	"testing"
	"time"
)

// userId cookie 签名/验签测试（OPT-20260807-004）：裸 ID 伪造防护 + 时效 + 篡改拒绝。

func TestSignParseUserIDCookieRoundtrip(t *testing.T) {
	v, err := signUserIDCookieValueAt("850256677331562496", time.Now().Unix())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if strings.Count(v, ".") != 2 {
		t.Fatalf("expected <id>.<ts>.<sig>, got %q", v)
	}
	id, err := parseUserIDCookieValue(v)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if id != "850256677331562496" {
		t.Fatalf("id=%q", id)
	}
}

func TestParseUserIDCookieRejectsBareID(t *testing.T) {
	// 旧形态裸 ID 必须被拒绝 —— 这是本项修复的核心（子域 XSS 伪造向量）。
	if _, err := parseUserIDCookieValue("850256677331562496"); err == nil {
		t.Fatal("expected bare userID cookie to be rejected")
	}
	if _, err := parseUserIDCookieValue("bootstrap-admin"); err == nil {
		t.Fatal("expected bare userID cookie to be rejected")
	}
	if _, err := parseUserIDCookieValue(""); err == nil {
		t.Fatal("expected empty cookie to be rejected")
	}
}

func TestParseUserIDCookieRejectsTamperedSignature(t *testing.T) {
	v, err := signUserIDCookieValueAt("850256677331562496", time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	// 篡改用户 ID（保留原签名）→ 拒绝
	tampered := "999999999999999999" + v[strings.Index(v, "."):]
	if _, err := parseUserIDCookieValue(tampered); err == nil {
		t.Fatal("expected tampered userID to be rejected")
	}
	// 篡改签名尾部一个字符 → 拒绝
	bad := v[:len(v)-1] + string(flipHex(v[len(v)-1]))
	if _, err := parseUserIDCookieValue(bad); err == nil {
		t.Fatal("expected tampered signature to be rejected")
	}
}

func TestParseUserIDCookieRejectsStale(t *testing.T) {
	// 超过 7 天时效 → 拒绝（cookie 本身 30 天，签名保证 7 天内有效）
	v, err := signUserIDCookieValueAt("42", time.Now().Unix()-8*24*3600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseUserIDCookieValue(v); err == nil {
		t.Fatal("expected stale signed cookie to be rejected")
	}
	// 未来时间（时钟偏差 > 5 分钟）→ 拒绝
	v2, err := signUserIDCookieValueAt("42", time.Now().Unix()+3600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseUserIDCookieValue(v2); err == nil {
		t.Fatal("expected future-timestamp cookie to be rejected")
	}
}

func TestSignUserIDCookieRejectsDottedID(t *testing.T) {
	if _, err := signUserIDCookieValueAt("a.b", time.Now().Unix()); err == nil {
		t.Fatal("expected dotted userID to be rejected at sign time")
	}
}

// flipHex 反转一个十六进制字符，保证篡改后签名不同。
func flipHex(c byte) byte {
	if c >= '0' && c < 'f' {
		if c == 'f' {
			return '0'
		}
		return c + 1
	}
	return '0'
}
