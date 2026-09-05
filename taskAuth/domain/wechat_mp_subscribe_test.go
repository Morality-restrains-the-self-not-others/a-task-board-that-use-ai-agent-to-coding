package domain

import (
	"strings"
	"testing"
)

func TestWechatMPCheckSignature(t *testing.T) {
	token := "testtoken"
	timestamp := "1234567890"
	nonce := "nonce1"
	if WechatMPCheckSignature(token, timestamp, nonce, "deadbeef") {
		t.Fatal("wrong signature accepted")
	}
	if WechatMPCheckSignature("", timestamp, nonce, "x") {
		t.Fatal("empty token accepted")
	}
	want := WechatMPSignature(token, timestamp, nonce)
	if !WechatMPCheckSignature(token, timestamp, nonce, want) {
		t.Fatal("valid signature rejected")
	}
	if !WechatMPCheckSignature(token, timestamp, nonce, strings.ToUpper(want)) {
		t.Fatal("case-insensitive signature rejected")
	}
}

func TestWechatMPCheckMsgSignature(t *testing.T) {
	if WechatMPCheckMsgSignature("t", "1", "n", "enc", "nope") {
		t.Fatal("bad msg signature accepted")
	}
	want := WechatMPSignature("t", "1", "n", "enc")
	if !WechatMPCheckMsgSignature("t", "1", "n", "enc", want) {
		t.Fatal("valid msg signature rejected")
	}
}

func TestWechatMPSubscribeEvent(t *testing.T) {
	if !WechatMPIsSubscribeEvent("event", "subscribe") {
		t.Fatal("subscribe")
	}
	if WechatMPIsSubscribeEvent("event", "unsubscribe") {
		t.Fatal("unsubscribe is not subscribe")
	}
	if !WechatMPIsUnsubscribeEvent("event", "unsubscribe") {
		t.Fatal("unsubscribe")
	}
	if WechatMPSubscribeCreatesUser() {
		t.Fatal("follow must not create users")
	}
}

func TestWechatMPIsFollowScanEvent(t *testing.T) {
	if !WechatMPIsFollowScanEvent("event", "SCAN") {
		t.Fatal("SCAN")
	}
	if !WechatMPIsFollowScanEvent("Event", "subscribe") {
		t.Fatal("subscribe")
	}
	if WechatMPIsFollowScanEvent("event", "unsubscribe") {
		t.Fatal("unsubscribe is not follow-scan")
	}
	if WechatMPIsScanEvent("event", "subscribe") {
		t.Fatal("subscribe is not SCAN")
	}
}

func TestWechatMPSceneTempID(t *testing.T) {
	if got := WechatMPSceneTempID("qrscene_880312345"); got != "880312345" {
		t.Fatalf("qrscene prefix: %q", got)
	}
	if got := WechatMPSceneTempID("QRSCENE_8803"); got != "8803" {
		t.Fatalf("case-insensitive prefix: %q", got)
	}
	if got := WechatMPSceneTempID("880312345"); got != "880312345" {
		t.Fatalf("SCAN raw scene: %q", got)
	}
	if got := WechatMPSceneTempID("  "); got != "" {
		t.Fatalf("empty: %q", got)
	}
}
