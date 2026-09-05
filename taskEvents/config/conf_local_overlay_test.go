package config

import (
	"os"
	"path/filepath"
	"testing"
)

// OPT-20260831-015: worker 读 conf/events/domain-events/config.yaml 时须叠加
// conf-local 同相对路径（internalSecret 等机密只放 conf-local）。
func TestLoadDomainEventsGlobalOverlaysConfLocal(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "events", "domain-events")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(confDir, "config.yaml"),
		[]byte("transport: kafka\ninternalSecret: \"\"\nredis:\n  db: 0\n  streamKeyPrefix: 'domain-events:'\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(root, "conf-local", "events", "domain-events")
	if err := os.MkdirAll(locDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(locDir, "config.yaml"),
		[]byte("internalSecret: secret-from-conf-local\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	out, err := loadDomainEventsGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	if out.InternalSecret != "secret-from-conf-local" {
		t.Fatalf("internalSecret=%q, want conf-local overlay", out.InternalSecret)
	}
	if out.Transport != "kafka" {
		t.Fatalf("transport=%q, want base kafka preserved", out.Transport)
	}
}

func TestLoadDomainEventsGlobalOverlaysDockerInfraConfLocal(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "events", "domain-events")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "conf", "base.yaml"), []byte("scheme: https\nbaseDomain: example.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(confDir, "config.yaml"),
		[]byte("transport: kafka\nredis:\n  host: tracked-redis\n  port: 6379\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(confDir, "docker-infra.yaml"),
		[]byte("redis:\n  host: frag-tracked\n  port: 6379\nkafka:\n  bootstrapServers: tracked:9092\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(root, "conf-local", "events", "domain-events")
	if err := os.MkdirAll(locDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(locDir, "docker-infra.yaml"),
		[]byte("redis:\n  host: from-conf-local\nkafka:\n  bootstrapServers: local:9092\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	out, err := loadDomainEventsGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	if out.Redis.Host != "from-conf-local" {
		t.Fatalf("redis.host=%q, want docker-infra conf-local overlay", out.Redis.Host)
	}
	if out.Redis.Port != 6379 {
		t.Fatalf("redis.port=%d, want tracked fragment port", out.Redis.Port)
	}
	if out.Kafka.BootstrapServers != "local:9092" {
		t.Fatalf("kafka=%q", out.Kafka.BootstrapServers)
	}
	if out.Transport != "kafka" {
		t.Fatalf("transport=%q", out.Transport)
	}
}

func TestLoadDomainEventsGlobalWithoutConfLocal(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "events", "domain-events")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(confDir, "config.yaml"),
		[]byte("transport: kafka\ninternalSecret: base-secret\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	out, err := loadDomainEventsGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	if out.InternalSecret != "base-secret" {
		t.Fatalf("internalSecret=%q, want base value", out.InternalSecret)
	}
}
