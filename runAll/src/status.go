package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	"runAll/src/domain"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusStarting   Status = "starting"
	StatusRetrying   Status = "retrying"
	StatusHealthy    Status = "healthy"
	StatusFailed     Status = "failed"
	StatusSkipped    Status = "skipped"
	StatusRestarting Status = "restarting"
	StatusBuilding   Status = "building"
	StatusStopped    Status = "stopped"
)

// ReadinessStatus tracks readiness probe state separately from liveness (Status).
type ReadinessStatus string

const (
	ReadinessUnknown  ReadinessStatus = ""
	ReadinessPending  ReadinessStatus = "pending"
	ReadinessReady    ReadinessStatus = "ready"
	ReadinessDegraded ReadinessStatus = "degraded"
)

type ServiceStatus struct {
	Name           string          `json:"name"`
	Group          string          `json:"group"`
	Status         Status          `json:"status"`
	Readiness      ReadinessStatus `json:"readiness,omitempty"`
	ReadinessError string          `json:"readiness_error,omitempty"`
	Phase          string          `json:"phase,omitempty"`
	FailurePhase   string          `json:"failure_phase,omitempty"`
	FailureCode    string          `json:"failure_code,omitempty"`
	DependsOn      []DepStatus     `json:"depends_on"`
	Command        string          `json:"command"`
	HealthPort     string          `json:"health_port"`
	CommandPort    string          `json:"command_port"`
	URL            string          `json:"url"`
	PID            int             `json:"pid"`
	StartedAt      string          `json:"started_at"`
	LastChecked    string          `json:"last_checked"`
	Error          string          `json:"error,omitempty"`
}

type DepStatus struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
}

type StatusStore struct {
	mu       sync.RWMutex
	services map[string]*ServiceStatus
}

func NewStatusStore() *StatusStore {
	return &StatusStore{
		services: make(map[string]*ServiceStatus),
	}
}

func (s *StatusStore) Init(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range names {
		s.services[name] = &ServiceStatus{
			Name:   name,
			Status: StatusPending,
		}
	}
}

// EnsureNames 仅为缺失的服务名补 pending 条目，不覆盖已有运行状态。
// 热加载磁盘 YAML 时禁止调用 Init（会把 healthy 打回 pending）。
func (s *StatusStore) EnsureNames(names []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := s.services[name]; ok {
			continue
		}
		s.services[name] = &ServiceStatus{
			Name:   name,
			Status: StatusPending,
		}
	}
}

func (s *StatusStore) Update(name string, status Status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return
	}
	svc.Status = status
	svc.Phase = phaseForStatus(status)
	// Update represents a generic state transition, so preflight metadata
	// must not leak from earlier failures.
	svc.FailurePhase = ""
	svc.FailureCode = ""
	if status == StatusStarting && svc.StartedAt == "" {
		svc.StartedAt = time.Now().Format(time.RFC3339)
	}
	if errMsg != "" {
		svc.Error = errMsg
	} else if status == StatusHealthy || status == StatusStopped {
		// Clear stale errors on healthy recoveries and on intentional stop /
		// successful build-only restores (Failed→Stopped after compile).
		svc.Error = ""
	}
	s.syncDependencyReadinessLocked(name)
}

func (s *StatusStore) SetReadiness(name string, readiness ReadinessStatus, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return
	}
	svc.Readiness = readiness
	if errMsg != "" {
		svc.ReadinessError = errMsg
	} else if readiness == ReadinessReady {
		svc.ReadinessError = ""
	}
	s.syncDependencyReadinessLocked(name)
}

func (s *StatusStore) syncDependencyReadinessLocked(depName string) {
	dep := s.services[depName]
	if dep == nil {
		return
	}
	depStatus := dependencyDisplayStatus(dep)
	for _, svc := range s.services {
		for i, d := range svc.DependsOn {
			if d.Name != depName {
				continue
			}
			svc.DependsOn[i].Status = depStatus
		}
	}
}

func dependencyDisplayStatus(dep *ServiceStatus) Status {
	if dep == nil {
		return StatusPending
	}
	if dep.Status == StatusFailed || dep.Status == StatusSkipped || dep.Status == StatusStopped {
		return dep.Status
	}
	if dep.Status == StatusHealthy && dep.Readiness != "" && dep.Readiness != ReadinessReady {
		return StatusRetrying
	}
	return dep.Status
}

func (s *StatusStore) RecordPreflightFailure(name, failureCode, errMsg string) {
	s.RecordFailure(name, "preflight", failureCode, errMsg)
}

func (s *StatusStore) RecordFailure(name, failurePhase, failureCode, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return
	}
	svc.Status = StatusFailed
	svc.Phase = failurePhase
	svc.FailurePhase = failurePhase
	svc.FailureCode = failureCode
	svc.Error = errMsg
	s.syncDependencyReadinessLocked(name)
}

// RecordFailureIfStatus records a failure only when the current status matches
// from. Used so an in-flight health wait cannot clobber a concurrent restart.
func (s *StatusStore) RecordFailureIfStatus(name string, from Status, failurePhase, failureCode, errMsg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok || svc.Status != from {
		return false
	}
	svc.Status = StatusFailed
	svc.Phase = failurePhase
	svc.FailurePhase = failurePhase
	svc.FailureCode = failureCode
	svc.Error = errMsg
	s.syncDependencyReadinessLocked(name)
	return true
}

// CompareAndSwapUpdate is CAS that also writes errMsg (empty clears the error).
func (s *StatusStore) CompareAndSwapUpdate(name string, old, new Status, errMsg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok || svc.Status != old {
		return false
	}
	svc.Status = new
	svc.Phase = phaseForStatus(new)
	svc.Error = errMsg
	if new != StatusFailed {
		svc.FailurePhase = ""
		svc.FailureCode = ""
	}
	s.syncDependencyReadinessLocked(name)
	return true
}

func phaseForStatus(status Status) string {
	switch status {
	case StatusStarting:
		return domain.ServiceLifecyclePhaseLaunch
	case StatusRetrying:
		return domain.ServiceLifecyclePhaseReadiness
	case StatusHealthy:
		return domain.ServiceLifecyclePhaseCompleted
	default:
		return ""
	}
}

func (s *StatusStore) SetPID(name string, pid int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.PID = pid
	}
}

// SetDependsOn replaces the dependency list for a service. Status values from
// the caller are ignored when the named dependency is already in the store:
// config reload / precise-restart re-apply depends_on as pending even though
// those services are already healthy, which would otherwise leave UI dots gray.
func (s *StatusStore) SetDependsOn(name string, deps []DepStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return
	}
	resolved := make([]DepStatus, len(deps))
	for i, d := range deps {
		resolved[i] = DepStatus{Name: d.Name, Status: d.Status}
		if dep := s.services[d.Name]; dep != nil {
			resolved[i].Status = dependencyDisplayStatus(dep)
		}
	}
	svc.DependsOn = resolved
}

func (s *StatusStore) SetCommand(name, command string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.Command = command
	}
}

func (s *StatusStore) SetGroup(name, group string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.Group = group
	}
}

func (s *StatusStore) SetHealthPort(name, port string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.HealthPort = port
	}
}

func (s *StatusStore) SetCommandPort(name, port string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.CommandPort = port
	}
}

func (s *StatusStore) SetURL(name, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.URL = url
	}
}

func (s *StatusStore) SetLastChecked(name string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.LastChecked = t.Format(time.RFC3339)
	}
}

func (s *StatusStore) CompareAndSwapStatus(name string, old, new Status) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return false
	}
	if svc.Status != old {
		return false
	}
	svc.Status = new
	svc.Phase = phaseForStatus(new)
	if new != StatusFailed {
		svc.FailurePhase = ""
		svc.FailureCode = ""
	}
	s.syncDependencyReadinessLocked(name)
	return true
}

func (s *StatusStore) UpdateDependencyStatus(name string, _ Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncDependencyReadinessLocked(name)
}

func (s *StatusStore) Get(name string) *ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svc, ok := s.services[name]
	if !ok {
		return nil
	}
	cp := *svc
	return &cp
}

func (s *StatusStore) All() []*ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ServiceStatus, 0, len(s.services))
	for _, svc := range s.services {
		result = append(result, svc)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
