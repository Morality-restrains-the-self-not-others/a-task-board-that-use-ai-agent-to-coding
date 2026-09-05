package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	domainservices "valueStream/domain/services"
)

type StreamStatus string

const (
	StreamStatusUnknown StreamStatus = "unknown"
	StreamStatusRunning StreamStatus = "running"
	StreamStatusPassed  StreamStatus = "passed"
	StreamStatusFailed  StreamStatus = "failed"
)

type StepStatus string

const (
	StepStatusPlanned    StepStatus = "planned"
	StepStatusDeprecated StepStatus = "deprecated"
	StepStatusPending    StepStatus = "pending"
	StepStatusRunning StepStatus = "running"
	StepStatusPassed  StepStatus = "passed"
	StepStatusFailed  StepStatus = "failed"
)

type TestRun struct {
	StartedAt  string `json:"started_at"`
	DurationMs int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	Passed     bool   `json:"passed"`
	LogTail    string `json:"log_tail"`
}

type FieldView struct {
	Name            string `json:"name"`
	ProviderService string `json:"provider_service"`
	Table           string `json:"table"`
	Column          string `json:"column"`
	Description     string `json:"description,omitempty"`
}

type StepView struct {
	Name         string      `json:"name"`
	Lifecycle    string      `json:"lifecycle"`
	TestFile     string      `json:"test_file"`
	Fields       []FieldView `json:"fields,omitempty"`
	Status       StepStatus  `json:"status"`
	LastRun      *TestRun    `json:"last_run,omitempty"`
	Impacted     bool        `json:"impacted"`
	ImpactReason string      `json:"impact_reason,omitempty"`
}

type StreamView struct {
	Name            string       `json:"name"`
	Domain          string       `json:"domain"`
	Description     string       `json:"description"`
	StreamStatus    StreamStatus `json:"stream_status"`
	FailedSteps     []string     `json:"failed_steps,omitempty"`
	LastStreamRunAt string       `json:"last_stream_run_at,omitempty"`
	Impacted        bool         `json:"impacted"`
	ImpactedSteps   []string     `json:"impacted_steps,omitempty"`
	ImpactStatus    string       `json:"impact_status"`
	Steps           []StepView   `json:"steps"`
}

type StreamsResponse struct {
	Streams []StreamView `json:"streams"`
}

type stepState struct {
	status  StepStatus
	lastRun *TestRun
}

type streamState struct {
	streamStatus    StreamStatus
	failedSteps     []string
	lastStreamRunAt string
	steps           map[string]*stepState
}

type streamImpactEvaluator interface {
	Evaluate(ctx context.Context, streams []domainservices.ImpactStream) ([]domainservices.StreamImpactResult, error)
}

type StatusStore struct {
	mu   sync.RWMutex
	cfg  *Config
	busy bool

	streams         map[string]*streamState
	impactEvaluator streamImpactEvaluator
}

func NewStatusStore(cfg *Config, impactEvaluator ...streamImpactEvaluator) *StatusStore {
	s := &StatusStore{
		cfg:     cfg,
		streams: make(map[string]*streamState),
	}
	if len(impactEvaluator) > 0 {
		s.impactEvaluator = impactEvaluator[0]
	}
	for _, vs := range cfg.ValueStreams {
		steps := make(map[string]*stepState)
		for _, st := range vs.Steps {
			initial := StepStatusPending
			if st.IsDeprecated() {
				initial = StepStatusDeprecated
			} else if st.IsPlanned() {
				initial = StepStatusPlanned
			}
			steps[st.Name] = &stepState{status: initial}
		}
		s.streams[vs.Name] = &streamState{
			streamStatus: StreamStatusUnknown,
			steps:        steps,
		}
	}
	return s
}

func (s *StatusStore) SetImpactEvaluator(evaluator streamImpactEvaluator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.impactEvaluator = evaluator
}

func (s *StatusStore) TryAcquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return false
	}
	s.busy = true
	return true
}

func (s *StatusStore) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
}

func (s *StatusStore) IsBusy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.busy
}

func (s *StatusStore) ReorderDomainsAndPersist(domainOrder []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errBusy
	}
	if s.cfg == nil {
		return fmt.Errorf("config is not initialized")
	}
	if strings.TrimSpace(s.cfg.ConfigPath) == "" {
		return fmt.Errorf("config path is not configured")
	}
	reordered, err := ReorderValueStreamsByDomain(s.cfg.ValueStreams, domainOrder)
	if err != nil {
		return err
	}
	original := s.cfg.ValueStreams
	s.cfg.ValueStreams = reordered
	if err := SaveConfigAtomic(s.cfg.ConfigPath, s.cfg); err != nil {
		s.cfg.ValueStreams = original
		return err
	}
	return nil
}

func (s *StatusStore) streamSteps(stream string) []Step {
	for _, vs := range s.cfg.ValueStreams {
		if vs.Name == stream {
			return vs.Steps
		}
	}
	return nil
}

func (s *StatusStore) nonPassedStepsInConfigOrder(stream string) []string {
	st := s.streams[stream]
	if st == nil {
		return nil
	}
	var names []string
	for _, stepCfg := range s.streamSteps(stream) {
		if !stepCfg.IsActive() {
			continue
		}
		if stepSt, ok := st.steps[stepCfg.Name]; ok && stepSt.status != StepStatusPassed {
			names = append(names, stepCfg.Name)
		}
	}
	return names
}

func (s *StatusStore) BeginStreamRun(stream string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.streams[stream]
	if !ok {
		return
	}
	st.streamStatus = StreamStatusRunning
	st.failedSteps = nil
	for _, stepCfg := range s.streamSteps(stream) {
		if !stepCfg.IsActive() {
			continue
		}
		if stepSt, ok := st.steps[stepCfg.Name]; ok {
			stepSt.status = StepStatusPending
		}
	}
}

func (s *StatusStore) EndStreamRun(stream string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.streams[stream]
	if !ok {
		return
	}
	st.lastStreamRunAt = time.Now().Format(time.RFC3339)
	nonPassed := s.nonPassedStepsInConfigOrder(stream)
	allPassed := len(nonPassed) == 0
	activeCount := 0
	for _, stepCfg := range s.streamSteps(stream) {
		if !stepCfg.IsActive() {
			continue
		}
		activeCount++
		stepSt := st.steps[stepCfg.Name]
		if stepSt == nil {
			allPassed = false
			continue
		}
		if stepSt.status != StepStatusPassed {
			allPassed = false
		}
	}
	if allPassed && activeCount > 0 {
		st.streamStatus = StreamStatusPassed
		st.failedSteps = nil
	} else {
		st.streamStatus = StreamStatusFailed
		st.failedSteps = nonPassed
	}
}

func (s *StatusStore) SetStepRunning(stream, step string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.streams[stream]; ok {
		if stepSt, ok := st.steps[step]; ok {
			stepSt.status = StepStatusRunning
		}
	}
}

func (s *StatusStore) FinishStep(stream, step string, run TestRun) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.streams[stream]
	if !ok {
		return
	}
	stepSt, ok := st.steps[step]
	if !ok {
		return
	}
	stepSt.lastRun = &run
	if run.Passed {
		stepSt.status = StepStatusPassed
	} else {
		stepSt.status = StepStatusFailed
	}
}

func buildFieldViews(fields []StepField) []FieldView {
	if len(fields) == 0 {
		return nil
	}
	out := make([]FieldView, 0, len(fields))
	for _, f := range fields {
		prov, tbl, col, err := ParseFieldName(f.Name)
		if err != nil {
			continue
		}
		out = append(out, FieldView{
			Name:            f.Name,
			ProviderService: prov,
			Table:           tbl,
			Column:          col,
			Description:     f.Description,
		})
	}
	return out
}

func (s *StatusStore) AllStreams() StreamsResponse {
	s.mu.RLock()
	var streams []StreamView
	impactEvaluator := s.impactEvaluator
	streamInputs := make([]domainservices.ImpactStream, 0, len(s.cfg.ValueStreams))
	for _, vs := range s.cfg.ValueStreams {
		st := s.streams[vs.Name]
		sv := StreamView{
			Name:            vs.Name,
			Domain:          strings.TrimSpace(vs.Domain),
			Description:     vs.Description,
			StreamStatus:    st.streamStatus,
			FailedSteps:     append([]string(nil), st.failedSteps...),
			LastStreamRunAt: st.lastStreamRunAt,
			ImpactStatus:    string(domainservices.ImpactStatusUnknown),
		}
		input := domainservices.ImpactStream{Name: vs.Name}
		for _, stepCfg := range vs.Steps {
			stepSt := st.steps[stepCfg.Name]
			sv.Steps = append(sv.Steps, StepView{
				Name:      stepCfg.Name,
				Lifecycle: stepCfg.Lifecycle,
				TestFile:  stepCfg.TestFile,
				Fields:    buildFieldViews(stepCfg.Fields),
				Status:    stepSt.status,
				LastRun:   stepSt.lastRun,
			})
			input.Steps = append(input.Steps, domainservices.ImpactStep{
				Name:     stepCfg.Name,
				TestFile: stepCfg.TestFile,
			})
		}
		streamInputs = append(streamInputs, input)
		streams = append(streams, sv)
	}
	s.mu.RUnlock()
	s.applyImpacts(streams, s.evaluateImpacts(impactEvaluator, streamInputs))
	return StreamsResponse{Streams: streams}
}

func (s *StatusStore) evaluateImpacts(evaluator streamImpactEvaluator, streams []domainservices.ImpactStream) map[string]domainservices.StreamImpactResult {
	if evaluator == nil {
		return nil
	}
	results, err := evaluator.Evaluate(context.Background(), streams)
	if err != nil {
		return nil
	}
	byName := make(map[string]domainservices.StreamImpactResult, len(results))
	for _, result := range results {
		byName[result.Name] = result
	}
	return byName
}

func (s *StatusStore) applyImpacts(streams []StreamView, impactByName map[string]domainservices.StreamImpactResult) {
	for i := range streams {
		streamView := &streams[i]
		streamImpact, ok := impactByName[streamView.Name]
		if !ok {
			continue
		}
		streamView.Impacted = streamImpact.Impacted
		streamView.ImpactStatus = string(streamImpact.ImpactStatus)
		streamView.ImpactedSteps = append([]string(nil), streamImpact.ImpactedSteps...)

		stepImpactByName := make(map[string]domainservices.StepImpactResult, len(streamImpact.Steps))
		for _, stepImpact := range streamImpact.Steps {
			stepImpactByName[stepImpact.Name] = stepImpact
		}
		for j := range streamView.Steps {
			stepView := &streamView.Steps[j]
			stepImpact, exists := stepImpactByName[stepView.Name]
			if !exists {
				continue
			}
			stepView.Impacted = stepImpact.Impacted
			stepView.ImpactReason = stepImpact.Reason
		}
	}
}

func (s *StatusStore) StreamView(name string) StreamView {
	resp := s.AllStreams()
	for _, sv := range resp.Streams {
		if sv.Name == name {
			return sv
		}
	}
	return StreamView{}
}
