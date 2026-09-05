package domain

import "testing"

type inMemoryServiceTopologyRepository struct {
	nodes []ServiceTopologyNode
}

func (r *inMemoryServiceTopologyRepository) ListAll() ([]ServiceTopologyNode, error) {
	return append([]ServiceTopologyNode(nil), r.nodes...), nil
}

func TestServiceCascadeOrchestrationService_PlanStartCascade(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "a", DependsOn: nil},
			{Name: "b", DependsOn: []string{"a"}},
			{Name: "c", DependsOn: []string{"b"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "a", Status: ServiceStatusStopped},
			{Name: "b", Status: ServiceStatusStopped},
			{Name: "c", Status: ServiceStatusStopped},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartCascade("c")
	if err != nil {
		t.Fatalf("PlanStartCascade: %v", err)
	}
	if plan.Operation != LifecycleOperationStart {
		t.Fatalf("operation = %q, want start", plan.Operation)
	}
	want := []string{"a", "b", "c"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestServiceCascadeOrchestrationService_PlanStartCascade_IncludesPendingUpstream(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "git-oauth", DependsOn: nil},
			{Name: "saas-backend", DependsOn: []string{"git-oauth"}},
			{Name: "taskFE", DependsOn: []string{"saas-backend"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "git-oauth", Status: ServiceStatusHealthy},
			{Name: "saas-backend", Status: ServiceStatusPending},
			{Name: "taskFE", Status: ServiceStatusPending},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartCascade("taskFE")
	if err != nil {
		t.Fatalf("PlanStartCascade: %v", err)
	}
	want := []string{"saas-backend", "taskFE"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestServiceCascadeOrchestrationService_PlanStartCascade_SkipsHealthyUpstream(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "a", DependsOn: nil},
			{Name: "b", DependsOn: []string{"a"}},
			{Name: "c", DependsOn: []string{"b"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "a", Status: ServiceStatusHealthy},
			{Name: "b", Status: ServiceStatusStopped},
			{Name: "c", Status: ServiceStatusStopped},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartCascade("c")
	if err != nil {
		t.Fatalf("PlanStartCascade: %v", err)
	}
	want := []string{"b", "c"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
}

func TestServiceCascadeOrchestrationService_PlanStopCascade_IncludesFailedDownstream(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "git-oauth", DependsOn: nil},
			{Name: "saas-backend", DependsOn: []string{"git-oauth"}},
			{Name: "taskFE", DependsOn: []string{"saas-backend"}},
			{Name: "ai-provider", DependsOn: []string{"saas-backend"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "git-oauth", Status: ServiceStatusHealthy},
			{Name: "saas-backend", Status: ServiceStatusFailed},
			{Name: "taskFE", Status: ServiceStatusStopped},
			{Name: "ai-provider", Status: ServiceStatusHealthy},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStopCascade("git-oauth")
	if err != nil {
		t.Fatalf("PlanStopCascade: %v", err)
	}
	want := []string{"ai-provider", "saas-backend", "git-oauth"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestServiceCascadeOrchestrationService_PlanStopCascade(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "a", DependsOn: nil},
			{Name: "b", DependsOn: []string{"a"}},
			{Name: "c", DependsOn: []string{"b"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "a", Status: ServiceStatusHealthy},
			{Name: "b", Status: ServiceStatusHealthy},
			{Name: "c", Status: ServiceStatusHealthy},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStopCascade("a")
	if err != nil {
		t.Fatalf("PlanStopCascade: %v", err)
	}
	want := []string{"c", "b", "a"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestServiceCascadeOrchestrationService_PlanStartGroup(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "a", GroupName: "g1", DependsOn: nil},
			{Name: "b", GroupName: "g1", DependsOn: []string{"a"}},
			{Name: "x", GroupName: "g2", DependsOn: []string{"a"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "a", GroupName: "g1", Status: ServiceStatusStopped},
			{Name: "b", GroupName: "g1", Status: ServiceStatusStopped},
			{Name: "x", GroupName: "g2", Status: ServiceStatusStopped},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartGroup("g1")
	if err != nil {
		t.Fatalf("PlanStartGroup: %v", err)
	}
	want := []string{"a", "b"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
}

func TestServiceCascadeOrchestrationService_PlanStartGroup_IncludesExternalUpstream(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "upstream", GroupName: "infra", DependsOn: nil},
			{Name: "api", GroupName: "platform", DependsOn: []string{"upstream"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "upstream", GroupName: "infra", Status: ServiceStatusStopped},
			{Name: "api", GroupName: "platform", Status: ServiceStatusStopped},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartGroup("platform")
	if err != nil {
		t.Fatalf("PlanStartGroup: %v", err)
	}
	want := []string{"upstream", "api"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
}

func TestServiceCascadeOrchestrationService_PlanStartAll(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "infra", GroupName: "g-infra", DependsOn: nil},
			{Name: "api", GroupName: "g-platform", DependsOn: []string{"infra"}},
			{Name: "web", GroupName: "g-platform", DependsOn: []string{"api"}},
			{Name: "worker", GroupName: "g-stack", DependsOn: nil},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "infra", GroupName: "g-infra", Status: ServiceStatusHealthy},
			{Name: "api", GroupName: "g-platform", Status: ServiceStatusStopped},
			{Name: "web", GroupName: "g-platform", Status: ServiceStatusStopped},
			{Name: "worker", GroupName: "g-stack", Status: ServiceStatusFailed},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll: %v", err)
	}
	want := []string{"api", "web", "worker"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestServiceCascadeOrchestrationService_PlanStartAll_AllHealthyIsEmptyNoOp(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "infra", DependsOn: nil},
			{Name: "api", DependsOn: []string{"infra"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "infra", Status: ServiceStatusHealthy},
			{Name: "api", Status: ServiceStatusHealthy},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll all healthy: %v", err)
	}
	if plan.Operation != LifecycleOperationStart {
		t.Fatalf("operation = %q, want start", plan.Operation)
	}
	if len(plan.OrderedNames) != 0 {
		t.Fatalf("ordered names = %#v, want empty no-op", plan.OrderedNames)
	}
}

func TestServiceCascadeOrchestrationService_PlanStartAll_OmitsSkipStartAll(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "docker-redis", GroupName: "infrastructure", DependsOn: nil},
			{Name: "git-service", GroupName: "gitlab-regions", DependsOn: []string{"docker-redis"}, SkipStartAll: true},
			{Name: "git-service-tencent-sh-1", GroupName: "gitlab-regions", SkipStartAll: true},
			{Name: "task-auth", GroupName: "platform", DependsOn: []string{"docker-redis"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "docker-redis", Status: ServiceStatusStopped},
			{Name: "git-service", Status: ServiceStatusStopped},
			{Name: "git-service-tencent-sh-1", Status: ServiceStatusStopped},
			{Name: "task-auth", Status: ServiceStatusStopped},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll: %v", err)
	}
	want := []string{"docker-redis", "task-auth"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestServiceCascadeOrchestrationService_PlanStartGroup_StillStartsSkipStartAll(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "docker-redis", GroupName: "infrastructure", DependsOn: nil},
			{Name: "git-service", GroupName: "gitlab-regions", DependsOn: []string{"docker-redis"}, SkipStartAll: true},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "docker-redis", Status: ServiceStatusStopped},
			{Name: "git-service", Status: ServiceStatusStopped},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartGroup("gitlab-regions")
	if err != nil {
		t.Fatalf("PlanStartGroup: %v", err)
	}
	want := []string{"docker-redis", "git-service"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v (panel group start must still work)", plan.OrderedNames, want)
	}
}

func TestServiceCascadeOrchestrationService_PlanStartAll_EmptyTopology(t *testing.T) {
	service := NewServiceCascadeOrchestrationService(
		&inMemoryServiceTopologyRepository{},
		&inMemoryServiceRuntimeContextRepository{},
	)
	plan, err := service.PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll empty topology: %v", err)
	}
	if len(plan.OrderedNames) != 0 {
		t.Fatalf("ordered names = %#v, want empty", plan.OrderedNames)
	}
}

func TestServiceCascadeOrchestrationService_PlanStartCascade_AllHealthyIsEmptyNoOp(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "a", DependsOn: nil},
			{Name: "b", DependsOn: []string{"a"}},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "a", Status: ServiceStatusHealthy},
			{Name: "b", Status: ServiceStatusHealthy},
		},
	}
	service := NewServiceCascadeOrchestrationService(topology, runtime)

	plan, err := service.PlanStartCascade("b")
	if err != nil {
		t.Fatalf("PlanStartCascade all healthy: %v", err)
	}
	if len(plan.OrderedNames) != 0 {
		t.Fatalf("ordered names = %#v, want empty no-op", plan.OrderedNames)
	}
}

// OPT-20260822-002 回归：task_status_changed 释放消费者在全部重启时对
// task-cloud-service 连接被拒会进 DLT。声明 depends_on task-cloud-service 后，
// start-all 中消费者必须排在 cloud 之后（等其健康），stop-all 逆序中先于
// cloud 停止 —— 重启空窗消费者不再裸奔。
func TestServiceCascadeOrchestrationService_ReleaseConsumerOrderedAfterCloud(t *testing.T) {
	const consumer = "task-events-task-status-changed-1-release-servers-on-terminal"
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "docker-kafka", DependsOn: nil},
			{Name: "task-cloud-service", DependsOn: []string{"docker-kafka"}},
			{Name: consumer, DependsOn: []string{"docker-kafka", "task-cloud-service"}},
		},
	}

	// start-all: all stopped so every node is startable.
	startRuntime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "docker-kafka", Status: ServiceStatusStopped},
			{Name: "task-cloud-service", Status: ServiceStatusStopped},
			{Name: consumer, Status: ServiceStatusStopped},
		},
	}
	startPlan, err := NewServiceCascadeOrchestrationService(topology, startRuntime).PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll: %v", err)
	}
	cloudIdx, consumerIdx := -1, -1
	for i, name := range startPlan.OrderedNames {
		if name == "task-cloud-service" {
			cloudIdx = i
		}
		if name == consumer {
			consumerIdx = i
		}
	}
	if cloudIdx < 0 || consumerIdx < 0 {
		t.Fatalf("start plan missing cloud(%d)/consumer(%d): %v", cloudIdx, consumerIdx, startPlan.OrderedNames)
	}
	if consumerIdx <= cloudIdx {
		t.Errorf("start-all: consumer should start after task-cloud-service, got cloud@%d consumer@%d: %v", cloudIdx, consumerIdx, startPlan.OrderedNames)
	}

	// stop-all: healthy nodes are all stop candidates.
	stopRuntime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "docker-kafka", Status: ServiceStatusHealthy},
			{Name: "task-cloud-service", Status: ServiceStatusHealthy},
			{Name: consumer, Status: ServiceStatusHealthy},
		},
	}
	stopPlan, err := NewServiceCascadeOrchestrationService(topology, stopRuntime).PlanStopAll()
	if err != nil {
		t.Fatalf("PlanStopAll: %v", err)
	}
	cloudIdx, consumerIdx = -1, -1
	for i, name := range stopPlan.OrderedNames {
		if name == "task-cloud-service" {
			cloudIdx = i
		}
		if name == consumer {
			consumerIdx = i
		}
	}
	if cloudIdx < 0 || consumerIdx < 0 {
		t.Fatalf("stop plan missing cloud(%d)/consumer(%d): %v", cloudIdx, consumerIdx, stopPlan.OrderedNames)
	}
	if consumerIdx >= cloudIdx {
		t.Errorf("stop-all: consumer should stop before task-cloud-service, got consumer@%d cloud@%d: %v", consumerIdx, cloudIdx, stopPlan.OrderedNames)
	}
}

func TestPlanCanaryRestartAll_IncludesHealthyAndStoppedInDepOrder(t *testing.T) {
	topology := &inMemoryServiceTopologyRepository{
		nodes: []ServiceTopologyNode{
			{Name: "a", DependsOn: nil},
			{Name: "b", DependsOn: []string{"a"}},
			{Name: "skip-me", DependsOn: nil, SkipStartAll: true},
		},
	}
	runtime := &inMemoryServiceRuntimeContextRepository{
		services: []ManagedService{
			{Name: "a", Status: ServiceStatusHealthy},
			{Name: "b", Status: ServiceStatusStopped},
			{Name: "skip-me", Status: ServiceStatusHealthy},
		},
	}
	plan, err := NewServiceCascadeOrchestrationService(topology, runtime).PlanCanaryRestartAll()
	if err != nil {
		t.Fatalf("PlanCanaryRestartAll: %v", err)
	}
	if plan.Operation != LifecycleOperationCanaryRestart {
		t.Fatalf("operation = %q, want canary-restart", plan.Operation)
	}
	want := []string{"a", "b"}
	if len(plan.OrderedNames) != len(want) {
		t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
	}
	for i, name := range want {
		if plan.OrderedNames[i] != name {
			t.Fatalf("ordered names = %#v, want %#v", plan.OrderedNames, want)
		}
	}
}

func TestIsCanaryRestartableStatus(t *testing.T) {
	if !IsCanaryRestartableStatus(ServiceStatusHealthy) {
		t.Fatal("healthy must be canary-restartable")
	}
	if !IsCanaryRestartableStatus(ServiceStatusStopped) {
		t.Fatal("stopped must be canary-restartable")
	}
	if !IsCanaryRestartableStatus(ServiceStatusRetrying) {
		t.Fatal("retrying must be canary-restartable (mid-health-check still needs operator restart)")
	}
	if !IsCanaryRestartableStatus(ServiceStatusSkipped) {
		t.Fatal("skipped must be canary-restartable (idle enough to launch)")
	}
	if IsCanaryRestartableStatus(ServiceStatusBuilding) {
		t.Fatal("building must not be canary-restartable")
	}
}
