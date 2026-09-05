package main

import (
	"os"
	"testing"
)

func TestLoadSMSConfigFromSyncedFragmentAndProvider(t *testing.T) {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	prevCfg := cfg
	prevEnv := os.Getenv("SMS_PROVIDER")
	_ = os.Unsetenv("SMS_PROVIDER")
	_ = os.Unsetenv("TASKAUTH_SMS_PROVIDER")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("SMS_PROVIDER", prevEnv)
	})

	cfg = Config{}
	loadSMSConfigFromSyncedFragment(repoRoot)
	if cfg.SMSProvider != "aliyun" {
		t.Fatalf("expected provider aliyun from auth/task-auth/sms.yaml, got %q", cfg.SMSProvider)
	}
	if cfg.SMSAliyunSignName == "" {
		t.Fatal("expected sign_name from synced sms.yaml")
	}
	if cfg.SMSTemplateCodeVerification == "" {
		t.Fatal("expected verification template fallback from notification template")
	}
	if got := smsProvider(); got != "aliyun" {
		t.Fatalf("smsProvider()=%q want aliyun (synced yaml fallback)", got)
	}
	if got := smsEnv("SMS_ALIYUN_SIGN_NAME"); got != cfg.SMSAliyunSignName {
		t.Fatalf("smsEnv sign=%q want %q", got, cfg.SMSAliyunSignName)
	}
}

func TestSMSProviderEnvOverridesSyncedYAML(t *testing.T) {
	prevCfg := cfg
	prevEnv := os.Getenv("SMS_PROVIDER")
	cfg.SMSProvider = "aliyun"
	_ = os.Setenv("SMS_PROVIDER", "mock")
	t.Cleanup(func() {
		cfg = prevCfg
		_ = os.Setenv("SMS_PROVIDER", prevEnv)
	})
	if got := smsProvider(); got != "mock" {
		t.Fatalf("env should win: got %q", got)
	}
}
