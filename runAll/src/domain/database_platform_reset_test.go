package domain

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type fakeStopper struct {
	running map[string]bool
	stopped []string
	names   []string
}

func (f *fakeStopper) StopService(_ context.Context, name string) error {
	f.stopped = append(f.stopped, name)
	f.running[name] = false
	return nil
}

func (f *fakeStopper) IsServiceRunning(name string) bool {
	return f.running[name]
}

func (f *fakeStopper) StopAllApplicationsExcept(_ context.Context, exclude []string, onProgress ProgressCallback) ([]string, []string) {
	ex := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		ex[name] = struct{}{}
	}
	order := f.names
	if len(order) == 0 {
		for name := range f.running {
			order = append(order, name)
		}
		sort.Strings(order)
	}
	var stopped []string
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		if _, skip := ex[name]; skip || !f.running[name] {
			continue
		}
		_ = f.StopService(context.Background(), name)
		stopped = append(stopped, name)
		if onProgress != nil {
			onProgress("已停止服务: " + name)
		}
	}
	sort.Strings(stopped)
	var still []string
	for name, on := range f.running {
		if !on {
			continue
		}
		if _, skip := ex[name]; skip {
			continue
		}
		still = append(still, name)
	}
	sort.Strings(still)
	return stopped, still
}

func (f *fakeStopper) RunningApplicationsExcept(exclude []string) []string {
	ex := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		ex[name] = struct{}{}
	}
	var running []string
	for name, on := range f.running {
		if !on {
			continue
		}
		if _, skip := ex[name]; skip {
			continue
		}
		running = append(running, name)
	}
	sort.Strings(running)
	return running
}

type fakeRemover struct {
	removed []string
	err     error
}

func (f *fakeRemover) RemoveAll(_ context.Context, entries []RegisteredDatabase) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, e := range entries {
		f.removed = append(f.removed, e.Key)
	}
	return f.removed, nil
}

type fakeScripts struct {
	calls []string
	fail  string
}

func (f *fakeScripts) RunScript(_ context.Context, scriptPath string) error {
	f.calls = append(f.calls, scriptPath)
	if f.fail != "" && f.fail == scriptPath {
		return errors.New("script failed")
	}
	return nil
}

func TestDatabasePlatformClearService_Clear_stopsAllApplications(t *testing.T) {
	entries := []RegisteredDatabase{{Key: "saas", Order: 10}}
	stopper := &fakeStopper{
		running: map[string]bool{
			"saas-backend":  true,
			"taskFE":  true,
			"ai-monitor":    true,
			"docker-redis":  true,
			"docker-kafka":  true,
		},
		names: []string{"docker-redis", "docker-kafka", "saas-backend", "taskFE", "ai-monitor"},
	}
	scripts := &fakeScripts{}
	svc := NewDatabasePlatformClearService(
		stopper,
		&fakeRemover{},
		scripts,
		entries,
		"redis.sh",
		"kafka.sh",
		"mysql.sh",
		DevDatabaseClearExcludeServices,
		nil,
	)
	result := svc.Clear(context.Background())
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
	if stopper.running["docker-redis"] != true || stopper.running["docker-kafka"] != true {
		t.Fatalf("infra should remain running: %+v", stopper.running)
	}
	for _, name := range []string{"saas-backend", "taskFE", "ai-monitor"} {
		if stopper.running[name] {
			t.Fatalf("%s should be stopped", name)
		}
	}
}

func TestDatabasePlatformClearService_Clear_failOnRedis(t *testing.T) {
	entries := []RegisteredDatabase{{Key: "saas", Order: 10}}
	scripts := &fakeScripts{fail: "redis.sh"}
	svc := NewDatabasePlatformClearService(nil, &fakeRemover{}, scripts, entries, "redis.sh", "kafka.sh", "", nil, nil)
	result := svc.Clear(context.Background())
	if result.Status != "partial" || result.RedisReset != "failed" {
		t.Fatalf("result = %+v", result)
	}
	// Redis failure should NOT block Kafka — Kafka should still be reached.
	if result.KafkaReset != "ok" {
		t.Fatalf("KafkaReset should be 'ok' (Redis failure is non-fatal), got %q; result=%+v", result.KafkaReset, result)
	}
}

func TestDatabasePlatformClearService_Clear_failOnKafka(t *testing.T) {
	entries := []RegisteredDatabase{{Key: "saas", Order: 10}}
	scripts := &fakeScripts{fail: "kafka.sh"}
	svc := NewDatabasePlatformClearService(nil, &fakeRemover{}, scripts, entries, "redis.sh", "kafka.sh", "", nil, nil)
	result := svc.Clear(context.Background())
	if result.Status != "partial" || result.KafkaReset != "failed" {
		t.Fatalf("result = %+v", result)
	}
	// Kafka failure should not affect Redis result.
	if result.RedisReset != "ok" {
		t.Fatalf("RedisReset should be 'ok' (Kafka failure is non-fatal), got %q; result=%+v", result.RedisReset, result)
	}
}

func TestDatabasePlatformInitService_Init_ok(t *testing.T) {
	entries := []RegisteredDatabase{
		{Key: "saas", Order: 10, MigrateScript: "m-saas.sh", InitScript: "i-saas.sh"},
	}
	scripts := &fakeScripts{}
	svc := NewDatabasePlatformInitService(
		&fakeStopper{running: map[string]bool{}},
		scripts,
		entries,
		DevDatabaseClearExcludeServices,
		nil,
	)
	result := svc.Init(context.Background())
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestDatabasePlatformInitService_Init_warnButProceedWhenRunning(t *testing.T) {
	entries := []RegisteredDatabase{
		{Key: "saas", Order: 10, MigrateScript: "m.sh"},
	}
	svc := NewDatabasePlatformInitService(
		&fakeStopper{running: map[string]bool{"taskFE": true}},
		&fakeScripts{},
		entries,
		DevDatabaseClearExcludeServices,
		nil,
	)
	result := svc.Init(context.Background())
	// Init should NOT block when services are running — init scripts
	// (e.g. saas/init.sh) may need Go services for API-driven seed ops.
	if result.Status == "blocked" {
		t.Fatalf("should not block when services are running, got: %+v", result)
	}
}

func TestDatabasePlatformInitService_Init_failOnMigrate(t *testing.T) {
	entries := []RegisteredDatabase{
		{Key: "saas", Order: 10, MigrateScript: "bad.sh", InitScript: "i.sh"},
	}
	scripts := &fakeScripts{fail: "bad.sh"}
	svc := NewDatabasePlatformInitService(nil, scripts, entries, nil, nil)
	result := svc.Init(context.Background())
	if result.Status != "partial" || len(result.Inits) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

// OPT-20260818-005 回归：清库停止阶段必须逐服务上报进度，否则 9999 在
// 「正在停止所有应用服务」后约 3 分钟无新行，运维会误判假死。
func TestClearDatabases_StopPhaseEmitsPerServiceProgress(t *testing.T) {
	stopper := &fakeStopper{
		running: map[string]bool{"svc-a": true, "svc-b": true, "svc-c": false},
	}
	var messages []string
	svc := NewDatabasePlatformClearService(stopper, nil, nil, nil, "", "", "", nil, func(msg string) {
		messages = append(messages, msg)
	})
	res := svc.Clear(context.Background())
	if res.Status != "ok" {
		t.Fatalf("status=%s messages=%v", res.Status, messages)
	}
	var progress []string
	for _, m := range messages {
		if strings.HasPrefix(m, "已停止服务: ") {
			progress = append(progress, m)
		}
	}
	sort.Strings(progress)
	want := []string{"已停止服务: svc-a", "已停止服务: svc-b"}
	if !reflect.DeepEqual(progress, want) {
		t.Fatalf("per-service progress=%v want %v (full messages=%v)", progress, want, messages)
	}
}
