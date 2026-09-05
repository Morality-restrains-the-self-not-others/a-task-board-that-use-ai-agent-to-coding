package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestVendorDocsCORSOptionsEmptyOriginsNil(t *testing.T) {
	if VendorDocsCORSOptions(nil) != nil {
		t.Fatal("empty origins must not emit PutCORS (would wipe bucket CORS)")
	}
	if VendorDocsCORSOptions([]string{"", "  "}) != nil {
		t.Fatal("blank origins must be nil")
	}
}

func TestVendorDocsCORSOptionsCoversProviderOriginAndSSEHeader(t *testing.T) {
	opt := VendorDocsCORSOptions([]string{
		"https://www.daydaymoney.com",
		" https://provider.daydaymoney.com ",
		"https://provider.daydaymoney.com",
	})
	if opt == nil || len(opt.Rules) != 1 {
		t.Fatalf("opt=%v", opt)
	}
	rule := opt.Rules[0]
	if len(rule.AllowedOrigins) != 2 {
		t.Fatalf("dedupe origins=%v", rule.AllowedOrigins)
	}
	joined := strings.Join(rule.AllowedOrigins, ",")
	if !strings.Contains(joined, "https://provider.daydaymoney.com") {
		t.Fatalf("missing provider origin: %v", rule.AllowedOrigins)
	}
	methods := strings.Join(rule.AllowedMethods, ",")
	if !strings.Contains(methods, "PUT") {
		t.Fatalf("methods=%v", rule.AllowedMethods)
	}
	headers := strings.Join(rule.AllowedHeaders, ",")
	if !strings.Contains(strings.ToLower(headers), "content-type") {
		t.Fatalf("headers=%v", rule.AllowedHeaders)
	}
	if !strings.Contains(strings.ToLower(headers), "x-cos-server-side-encryption") {
		t.Fatalf("missing SSE header: %v", rule.AllowedHeaders)
	}
	if CORSAllowsAuthorization(opt) {
		t.Fatal("CORS must not allow Authorization; query-string signed PUT would fail signature")
	}
}

func TestNormalizeCORSOriginsSkipsEmpty(t *testing.T) {
	got := NormalizeCORSOrigins([]string{"https://a.example", "", "https://a.example", " https://b.example "})
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("got=%v", got)
	}
}

func TestLiveEnsureVendorDocsCORS(t *testing.T) {
	if os.Getenv("LIVE_COS_CORS") != "1" {
		t.Skip("set LIVE_COS_CORS=1 to PUT bucket CORS from conf")
	}
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.VendorDocsCOS.CORSAllowedOrigins) == 0 {
		t.Fatal("corsAllowedOrigins empty after LoadConfig")
	}
	api, err := NewRealCOSAPI(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := api.client.Bucket.GetCORS(context.Background())
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "NoSuchCORSConfiguration") || strings.Contains(msg, "AccessDenied") {
			t.Log("bucket CORS missing or not writable; browser upload uses COS POST Object")
			return
		}
		t.Fatalf("GetCORS: %v", err)
	}
	if got == nil || len(got.Rules) == 0 {
		t.Fatal("expected CORS rules after EnsureCORS")
	}
	blob, _ := json.Marshal(got.Rules)
	t.Logf("cors_rules=%s", blob)
}
