package main

import (
	"path/filepath"
	"testing"

	"runAll/src/infrastructure"
)

// ---------- 服务名解析 ----------

func TestResolveRegisteredService(t *testing.T) {
	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{
			{Services: []Service{
				{Name: "task-auth", ConfApp: "auth/task-auth", WorkingDir: "taskAuth"},
				{Name: "task-bill", ConfApp: "billing/task-bill", WorkingDir: "taskBill"},
				{Name: "taskFE", ConfApp: "frontend/vue", WorkingDir: "taskFE/app"},
				{Name: "saas-backend", ConfApp: "django", WorkingDir: "task2app"},
			}},
		},
	}
	// 生产路径：resolveWorkingDirs 在配置加载时把相对 working_dir 绝对化为
	// <monorepoRoot>/taskFE/app —— 别名匹配必须对绝对化后的路径仍生效
	// （此前按 "/" 切分首段得空串，导致 taskFE 等别名报 "unknown service"）。
	root, rootErr := infrastructure.ResolveMonorepoRoot("")
	if rootErr == nil && root != "" {
		cfg.Groups[0].Services = append(cfg.Groups[0].Services,
			Service{Name: "abs-frontend", WorkingDir: filepath.Join(root, "absfe", "app")})
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		want string
	}{
		{"task-auth", "task-auth"}, // 精确 name
		{"django", "saas-backend"}, // conf_app
		{"taskBill", "task-bill"},  // 工作目录名
		{"taskFE", "taskFE"},       // 精确 name（迁移后工作目录父目录名与 name 同名）
		{"unknown-svc", ""},        // 无法解析
	}
	if rootErr == nil && root != "" {
		cases = append(cases, struct {
			name string
			want string
		}{"absfe", "abs-frontend"}) // 绝对化工作目录首段别名
	}
	for _, c := range cases {
		svc := runner.resolveRegisteredService(c.name)
		got := ""
		if svc != nil {
			got = svc.Name
		}
		if got != c.want {
			t.Errorf("resolveRegisteredService(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestResolveRegisteredServicesExpandsSharedWorkingDir — taskEvents 等工作目录
// 挂多个消费者时，别名须展开为全部服务，不得报 unknown（此前 len!=1 → nil）。
func TestResolveRegisteredServicesExpandsSharedWorkingDir(t *testing.T) {
	store := NewStatusStore()
	cfg := &Config{
		Groups: []Group{
			{Services: []Service{
				{Name: "task-events-a", WorkingDir: "taskEvents"},
				{Name: "task-events-b", WorkingDir: "taskEvents"},
				{Name: "task-auth", WorkingDir: "taskAuth"},
			}},
		},
	}
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatal(err)
	}

	// 共享 working_dir 别名 → 全部展开
	got := runner.resolveRegisteredServices("taskEvents")
	if len(got) != 2 {
		t.Fatalf("resolveRegisteredServices(taskEvents) len=%d, want 2", len(got))
	}
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if !names["task-events-a"] || !names["task-events-b"] {
		t.Fatalf("expanded names = %v, want task-events-a/b", names)
	}

	// 单服务别名仍只返回 1 个（兼容 resolveRegisteredService）
	one := runner.resolveRegisteredServices("taskAuth")
	if len(one) != 1 || one[0].Name != "task-auth" {
		t.Fatalf("resolveRegisteredServices(taskAuth) = %v, want [task-auth]", one)
	}
	if svc := runner.resolveRegisteredService("taskEvents"); svc != nil {
		t.Fatalf("resolveRegisteredService(taskEvents) 单值 API 对多匹配仍应返回 nil（避免误选一个）, got %q", svc.Name)
	}
	if svc := runner.resolveRegisteredService("taskAuth"); svc == nil || svc.Name != "task-auth" {
		t.Fatalf("resolveRegisteredService(taskAuth) = %v, want task-auth", svc)
	}
}
