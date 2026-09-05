package main

import (
	"os"
	"path/filepath"
	"testing"
)

// OPT-20260901-012: 配置加载改走 confload.UnmarshalYAMLMerged 深合并后，
// conf-local/billing/wechatPay/conf.yaml 的 nested 键必须覆盖而不得丢 skeleton 其它键，
// 且 conf/config.local.yaml（已废弃）不得被读取。
func TestLoadWechatPayConfigOverlaysConfLocalDeepMerge(t *testing.T) {
	orig := wechatCfg
	t.Cleanup(func() { wechatCfg = orig })
	wechatCfg = wechatPayConfig{}

	root := t.TempDir()
	app := filepath.Join(root, "conf", "billing", "wechatPay")
	if err := os.MkdirAll(app, 0755); err != nil {
		t.Fatal(err)
	}
	skeleton := "" +
		"mode: mock\n" +
		"merchant_serial_no: \"\"\n" +
		"fapiao:\n" +
		"  enabled: true\n" +
		"  tax_rate: 6\n"
	if err := os.WriteFile(filepath.Join(app, "conf.yaml"), []byte(skeleton), 0644); err != nil {
		t.Fatal(err)
	}
	// 已废弃的 config.local.yaml 不得被读取
	if err := os.WriteFile(filepath.Join(app, "config.local.yaml"), []byte("api_v3_key: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	localDir := filepath.Join(root, "conf-local", "billing", "wechatPay")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatal(err)
	}
	overlay := "" +
		"merchant_serial_no: overlay-serial\n" +
		"fapiao:\n" +
		"  tax_code: \"010204\"\n"
	if err := os.WriteFile(filepath.Join(localDir, "conf.yaml"), []byte(overlay), 0644); err != nil {
		t.Fatal(err)
	}

	loadWechatPayConfig(root)
	if wechatCfg.MerchantSerialNo != "overlay-serial" {
		t.Fatalf("merchant_serial_no=%q, want conf-local overlay (empty tracked must not win)", wechatCfg.MerchantSerialNo)
	}
	if !wechatCfg.Fapiao.Enabled {
		t.Fatalf("fapiao.enabled=false, want tracked nested key preserved by deep merge")
	}
	if wechatCfg.Fapiao.TaxCode != "010204" {
		t.Fatalf("fapiao.tax_code=%q, want conf-local nested overlay", wechatCfg.Fapiao.TaxCode)
	}
}

// 回归：conf-local 缺失时保留 tracked conf.yaml 的密钥/配置。
func TestLoadWechatPayConfigWithoutConfLocalKeepsSkeleton(t *testing.T) {
	orig := wechatCfg
	t.Cleanup(func() { wechatCfg = orig })
	wechatCfg = wechatPayConfig{}

	root := t.TempDir()
	app := filepath.Join(root, "conf", "billing", "wechatPay")
	if err := os.MkdirAll(app, 0755); err != nil {
		t.Fatal(err)
	}
	skeleton := "" +
		"mode: mock\n" +
		"merchant_serial_no: skeleton-serial\n"
	if err := os.WriteFile(filepath.Join(app, "conf.yaml"), []byte(skeleton), 0644); err != nil {
		t.Fatal(err)
	}

	loadWechatPayConfig(root)
	if wechatCfg.MerchantSerialNo != "skeleton-serial" {
		t.Fatalf("merchant_serial_no=%q, want skeleton when conf-local missing", wechatCfg.MerchantSerialNo)
	}
}
