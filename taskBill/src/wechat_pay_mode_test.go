package main

import "testing"

func TestWechatIsMockOnlyWhenModeIsMock(t *testing.T) {
	origMode, origOK := wechatCfg.Mode, wechatLiveOK
	t.Cleanup(func() {
		wechatCfg.Mode = origMode
		wechatLiveOK = origOK
	})

	wechatCfg.Mode, wechatLiveOK = "live", false
	if wechatIsMock() {
		t.Fatal("mode=live 且 live 客户端未就绪时不得视为 mock（禁止 silent fallback）")
	}
	if wechatConfigured() {
		t.Fatal("mode=live 且 live 客户端未就绪时不得视为已配置")
	}

	wechatCfg.Mode, wechatLiveOK = "live", true
	if wechatIsMock() {
		t.Fatal("mode=live 且 live 就绪时不得视为 mock")
	}
	if !wechatConfigured() {
		t.Fatal("mode=live 且 live 就绪时应视为已配置")
	}

	wechatCfg.Mode, wechatLiveOK = "mock", false
	if !wechatIsMock() {
		t.Fatal("mode=mock 应为 mock")
	}
	if !wechatConfigured() {
		t.Fatal("mode=mock 仍可供单测预下单")
	}
}
