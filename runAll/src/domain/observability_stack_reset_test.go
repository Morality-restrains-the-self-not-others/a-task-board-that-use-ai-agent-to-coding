package domain

import (
	"context"
	"errors"
	"testing"
)

type stubLogRepository struct {
	cleared []string
}

func (s *stubLogRepository) Append(string, LogEntry) {}
func (s *stubLogRepository) Tail(string, int) []LogEntry {
	return nil
}
func (s *stubLogRepository) Clear(service string) {
	s.cleared = append(s.cleared, service)
}

type stubFileTruncator struct {
	truncated int
	err       error
}

func (s *stubFileTruncator) AppendLine(string, string, string) error { return nil }
func (s *stubFileTruncator) Close() error                            { return nil }
func (s *stubFileTruncator) TruncateService(string) error            { return nil }
func (s *stubFileTruncator) TruncateAll() (int, error) {
	return s.truncated, s.err
}

type stubStorageResetter struct {
	outcome StorageResetOutcome
	err     error
}

func (s *stubStorageResetter) Reset(context.Context) (StorageResetOutcome, error) {
	return s.outcome, s.err
}

func TestObservabilityStackResetService_ClearAll(t *testing.T) {
	repo := &stubLogRepository{}
	file := &stubFileTruncator{truncated: 3}
	storage := &stubStorageResetter{outcome: StorageResetOutcome{
		LokiReset:       "ok",
		PromtailReset:   "ok",
		TempoReset:      "ok",
		PrometheusReset: "ok",
	}}
	svc := NewObservabilityStackResetService(repo, file, storage)

	result := svc.ClearAll(context.Background(), []string{"saas-backend", "task-auth"})
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
	if result.MemoryServicesCleared != 2 {
		t.Fatalf("memory cleared = %d, want 2", result.MemoryServicesCleared)
	}
	if result.FilesTruncated != 3 {
		t.Fatalf("files truncated = %d, want 3", result.FilesTruncated)
	}
	if len(repo.cleared) != 2 {
		t.Fatalf("cleared services = %#v", repo.cleared)
	}
}

func TestObservabilityStackResetService_ClearAll_NilStorageSkipsRemoteReset(t *testing.T) {
	repo := &stubLogRepository{}
	file := &stubFileTruncator{truncated: 2}
	svc := NewObservabilityStackResetService(repo, file, nil)

	result := svc.ClearAll(context.Background(), []string{"saas-backend", "task-auth"})
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
	if result.MemoryServicesCleared != 2 || result.FilesTruncated != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.LokiReset != "skipped" || result.PromtailReset != "skipped" {
		t.Fatalf("remote reset should be skipped, got loki=%q promtail=%q", result.LokiReset, result.PromtailReset)
	}
}

func TestObservabilityStackResetService_PartialOnStorageError(t *testing.T) {
	repo := &stubLogRepository{}
	file := &stubFileTruncator{truncated: 1}
	storage := &stubStorageResetter{
		outcome: StorageResetOutcome{LokiReset: "error: docker missing"},
		err:     errors.New("docker missing"),
	}
	svc := NewObservabilityStackResetService(repo, file, storage)

	result := svc.ClearAll(context.Background(), []string{"saas-backend"})
	if result.Status != "partial" {
		t.Fatalf("status = %q, want partial", result.Status)
	}
	if result.MemoryServicesCleared != 1 {
		t.Fatalf("memory cleared = %d, want 1", result.MemoryServicesCleared)
	}
}

func TestStorageResetOutcome_AllOK(t *testing.T) {
	if !(StorageResetOutcome{
		LokiReset: "ok", PromtailReset: "ok", TempoReset: "ok", PrometheusReset: "skipped",
	}).AllOK() {
		t.Fatal("expected AllOK true")
	}
	if (StorageResetOutcome{LokiReset: "error: x"}).AllOK() {
		t.Fatal("expected AllOK false")
	}
}
