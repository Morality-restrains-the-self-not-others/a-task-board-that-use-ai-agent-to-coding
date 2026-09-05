package cfg

import (
	"os"
	"path/filepath"
	"testing"
)

// 回归：email-sent worker 曾 os.ReadFile(conf/.../email.yaml) 而不叠 conf-local，
// host_password 空串导致 QQ SMTP 535，UI 却因 Kafka 入队成功显示「验证码已发送」。
func TestLoadSettingsOverlaysConfLocalHostPassword(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "events", "domain-events")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skeleton := "" +
		"email:\n" +
		"  host: smtp.qq.com\n" +
		"  port: 465\n" +
		"  use_ssl: true\n" +
		"  timeout: 10\n" +
		"  host_user: smtp-user@example.com\n" +
		"  host_password: \"\"\n" +
		"  default_from: smtp-user@example.com\n"
	if err := os.WriteFile(filepath.Join(confDir, "email.yaml"), []byte(skeleton), 0o644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(root, "conf-local", "events", "domain-events")
	if err := os.MkdirAll(locDir, 0o755); err != nil {
		t.Fatal(err)
	}
	overlay := "" +
		"email:\n" +
		"  host_password: overlay-smtp-password\n"
	if err := os.WriteFile(filepath.Join(locDir, "email.yaml"), []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email.Password != "overlay-smtp-password" {
		t.Fatalf("password=%q, want conf-local overlay (empty tracked skeleton must not win)", got.Email.Password)
	}
	if got.Email.Host != "smtp.qq.com" {
		t.Fatalf("host=%q, want base host preserved", got.Email.Host)
	}
	if got.Email.User != "smtp-user@example.com" {
		t.Fatalf("user=%q, want base user preserved", got.Email.User)
	}
}

func TestLoadSettingsWithoutConfLocalKeepsSkeletonPassword(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "events", "domain-events")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skeleton := "" +
		"email:\n" +
		"  host: smtp.example.com\n" +
		"  port: 465\n" +
		"  host_user: a@example.com\n" +
		"  host_password: skeleton-pass\n" +
		"  default_from: a@example.com\n"
	if err := os.WriteFile(filepath.Join(confDir, "email.yaml"), []byte(skeleton), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email.Password != "skeleton-pass" {
		t.Fatalf("password=%q, want skeleton when conf-local missing", got.Email.Password)
	}
}
