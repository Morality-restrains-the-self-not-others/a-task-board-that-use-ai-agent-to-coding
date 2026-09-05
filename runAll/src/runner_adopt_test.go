package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func TestAdoptListeningServices_MarksHealthyWhenPortAndProbeOK(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	portStr := strconv.Itoa(port)

	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:    "svc-adopt",
				Command: "true",
				HealthCheck: HealthCheck{
					TCP: fmt.Sprintf("127.0.0.1:%d", port),
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(p string) ([]int, error) {
		if p == portStr || p == "127.0.0.1:"+portStr {
			return []int{424242}, nil
		}
		return nil, nil
	}

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 1 {
		t.Fatalf("adopted=%d status=%v", n, store.Get("svc-adopt"))
	}
	st := store.Get("svc-adopt")
	if st == nil || st.Status != StatusHealthy {
		t.Fatalf("status=%v", st)
	}
	if st.PID != 424242 {
		t.Fatalf("pid=%d", st.PID)
	}
}

func TestAdoptListeningServices_SkipsWhenAlreadyHealthy(t *testing.T) {
	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:        "svc-ok",
				Command:     "true",
				HealthCheck: HealthCheck{TCP: "127.0.0.1:1"},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	store.Update("svc-ok", StatusHealthy, "")
	if n := runner.AdoptRunningManagedServices(context.Background()); n != 0 {
		t.Fatalf("adopted=%d want 0", n)
	}
}

func TestAdoptRunningManagedServices_UsesPersistedOwnershipPID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "owned-svc",
					Command: "echo hi",
					HealthCheck: HealthCheck{
						URL:           srv.URL,
						Timeout:       5,
						Retries:       2,
						CheckInterval: 1,
					},
				},
			}},
		},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	selfPID := os.Getpid()
	ownership, err := domain.NewServiceOwnership(
		"owned-svc",
		"prev-session",
		selfPID,
		defaultOwnershipConfigRef,
		srv.URL,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := runner.ownershipRepo.Save(ownership); err != nil {
		t.Fatalf("Save ownership: %v", err)
	}
	runner.listenerPIDsFn = func(port string) ([]int, error) {
		return nil, nil
	}

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 1 {
		t.Fatalf("adopted = %d, want 1", n)
	}
	st := store.Get("owned-svc")
	if st.Status != StatusHealthy {
		t.Fatalf("status = %q, want healthy", st.Status)
	}
	if st.PID != selfPID {
		t.Fatalf("pid = %d, want %d", st.PID, selfPID)
	}
}

// OPT-20260810-041：热替换后健康探针瞬时失败（如 EOF/竞态）时，收养应短退避重试
// 而非一次失败即放弃；前若干次探测失败、随后成功时仍应收养为 healthy。
func TestAdoptListeningServices_RetriesTransientHealthFailure(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	// 前 2 次探测返回 503（模拟瞬时失败），之后成功。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		fail := calls <= 2
		mu.Unlock()
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:    "svc-adopt-retry",
				Command: "true",
				HealthCheck: HealthCheck{
					URL:     srv.URL,
					Timeout: 5,
					Retries: 1,
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(port string) ([]int, error) {
		return []int{424244}, nil
	}
	// 缩短重试间隔避免拖慢测试。
	oldInterval := adoptProbeRetryInterval
	adoptProbeRetryInterval = 20 * time.Millisecond
	t.Cleanup(func() { adoptProbeRetryInterval = oldInterval })

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 1 {
		t.Fatalf("adopted=%d status=%v", n, store.Get("svc-adopt-retry"))
	}
	st := store.Get("svc-adopt-retry")
	if st == nil || st.Status != StatusHealthy {
		t.Fatalf("status=%v, want healthy after transient probe failures", st)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls < 3 {
		t.Fatalf("probe calls=%d, want >=3 (retried after transient failures)", calls)
	}
}

// 连续健康失败（重试后仍失败）应放弃收养，不置为 healthy。
func TestAdoptListeningServices_StillUnhealthyAfterRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:    "svc-adopt-bad",
				Command: "true",
				HealthCheck: HealthCheck{
					URL:     srv.URL,
					Timeout: 5,
					Retries: 1,
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(port string) ([]int, error) {
		return []int{424245}, nil
	}
	oldInterval := adoptProbeRetryInterval
	adoptProbeRetryInterval = 10 * time.Millisecond
	t.Cleanup(func() { adoptProbeRetryInterval = oldInterval })

	if n := runner.AdoptRunningManagedServices(context.Background()); n != 0 {
		t.Fatalf("adopted=%d, want 0 for persistent unhealthy", n)
	}
	if st := store.Get("svc-adopt-bad"); st != nil && st.Status == StatusHealthy {
		t.Fatalf("status=%v, want not healthy", st)
	}
}

// OPT-20260811-012 回归：shutdown-self 热替换后 adoptListeningServices 返回 0。
// 根因：docker/netns 发布的端口对 lsof 不可见、exec 型健康检查无端口可解析，而
// ownership PID 又因历史重启而全部失效，导致「无端口监听 + 无存活 ownership」的服务
// 在探针运行前就被证据门禁 return false，/api/status 全空。
// 回归断言：无端口/无 ownership 证据但健康探针通过的服务，仍应收养为 healthy。
func TestAdoptListeningServices_ProbeAsEvidenceWhenNoPortOrOwnership(t *testing.T) {
	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:    "svc-probe-evidence",
				Command: "true",
				HealthCheck: HealthCheck{
					// exec 型健康检查：无 URL/TCP，resolveServicePorts 解析不出端口。
					Exec: "exit 0",
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	// 无任何端口监听证据。
	runner.listenerPIDsFn = func(port string) ([]int, error) { return nil, nil }

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 1 {
		t.Fatalf("adopted=%d, want 1 (probe passing with no port/ownership evidence must still adopt)", n)
	}
	st := store.Get("svc-probe-evidence")
	if st == nil || st.Status != StatusHealthy {
		t.Fatalf("status=%v, want healthy", st)
	}
}

// 探针作为证据必须有下限：探针失败（服务确实不在运行）时不得被收养为 healthy，
// 否则会把宕机服务误报为健康导致编排不再拉起它。
func TestAdoptListeningServices_ProbeFailsThenNotAdopted(t *testing.T) {
	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:    "svc-probe-fail",
				Command: "true",
				HealthCheck: HealthCheck{
					Exec: "exit 3", // 探针确定性失败
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(port string) ([]int, error) { return nil, nil }

	if n := runner.AdoptRunningManagedServices(context.Background()); n != 0 {
		t.Fatalf("adopted=%d, want 0 for failing probe", n)
	}
	if st := store.Get("svc-probe-fail"); st != nil && st.Status == StatusHealthy {
		t.Fatalf("status=%v, must not be healthy when probe fails", st.Status)
	}
}

func TestAdoptRunningManagedServices_FileOwnershipSurvivesHotReplace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	repo := infrastructure.NewFileServiceOwnershipRepository(filepath.Join(dir, "ownership.json"))
	selfPID := os.Getpid()
	ownership, err := domain.NewServiceOwnership(
		"file-owned",
		"hot-replace-session",
		selfPID,
		defaultOwnershipConfigRef,
		srv.URL,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := repo.Save(ownership); err != nil {
		t.Fatalf("Save: %v", err)
	}

	store := NewStatusStore()
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:    "file-owned",
					Command: "echo hi",
					HealthCheck: HealthCheck{
						URL:           srv.URL,
						Timeout:       5,
						Retries:       2,
						CheckInterval: 1,
					},
				},
			}},
		},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.ownershipRepo = repo
	runner.ownershipGuard = domain.NewServiceOwnershipGuardService(repo)
	runner.listenerPIDsFn = func(port string) ([]int, error) { return nil, nil }

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 1 {
		t.Fatalf("adopted = %d, want 1", n)
	}
	if got := store.Get("file-owned").Status; got != StatusHealthy {
		t.Fatalf("status = %q, want healthy", got)
	}
}

func TestAdoptListeningServices_SkipsSourceTreeELFInDeployMode(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("DEPLOY_ROOT", "/home/ljy/bin/daydaymoney-deploy")
	oldRead := readProcessExePath
	t.Cleanup(func() { readProcessExePath = oldRead })
	readProcessExePath = func(pid int) (string, error) {
		return "/tmp/ram-work/taskAuth/bin/taskAuth", nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	portStr := strconv.Itoa(port)

	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:         "svc-leftover",
				StartCommand: "./bin/taskAuth",
				HealthCheck: HealthCheck{
					TCP: fmt.Sprintf("127.0.0.1:%d", port),
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(p string) ([]int, error) {
		if p == portStr || p == "127.0.0.1:"+portStr {
			return []int{424242}, nil
		}
		return nil, nil
	}

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 0 {
		t.Fatalf("adopted leftover source-tree ELF: n=%d status=%v", n, store.Get("svc-leftover"))
	}
}

func TestAdoptListeningServices_AdoptsDeployRootELFInDeployMode(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("DEPLOY_ROOT", "/home/ljy/bin/daydaymoney-deploy")
	oldRead := readProcessExePath
	t.Cleanup(func() { readProcessExePath = oldRead })
	readProcessExePath = func(pid int) (string, error) {
		return "/home/ljy/bin/daydaymoney-deploy/bin/taskAuth", nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	portStr := strconv.Itoa(port)

	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{{
			Name: "g",
			Services: []Service{{
				Name:         "svc-deploy",
				StartCommand: "./bin/taskAuth",
				HealthCheck: HealthCheck{
					TCP: fmt.Sprintf("127.0.0.1:%d", port),
				},
			}},
		}},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	runner.listenerPIDsFn = func(p string) ([]int, error) {
		if p == portStr || p == "127.0.0.1:"+portStr {
			return []int{424242}, nil
		}
		return nil, nil
	}

	n := runner.AdoptRunningManagedServices(context.Background())
	if n != 1 {
		t.Fatalf("adopted=%d status=%v", n, store.Get("svc-deploy"))
	}
}
