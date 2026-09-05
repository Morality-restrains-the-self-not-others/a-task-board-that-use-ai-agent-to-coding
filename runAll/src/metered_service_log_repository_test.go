package main

import (
	"testing"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func TestMeteredServiceLogRepository_TracksStats(t *testing.T) {
	repo := infrastructure.NewMeteredServiceLogRepository(nil)

	// Append some log entries.
	for i := 0; i < 5; i++ {
		entry, err := domain.NewLogEntry(time.Now(), "test-svc", domain.StreamStdout, "hello world")
		if err != nil {
			t.Fatal(err)
		}
		repo.Append("test-svc", entry)
	}

	// Check stats.
	stats := repo.Stats("test-svc")
	if stats.TotalLines != 5 {
		t.Errorf("expected 5 lines, got %d", stats.TotalLines)
	}
	if stats.TotalBytes == 0 {
		t.Error("expected non-zero bytes")
	}
	if stats.LastLogAt.IsZero() {
		t.Error("expected non-zero last_log_at")
	}

	// Check AllStats.
	all := repo.AllStats()
	if len(all) != 1 {
		t.Errorf("expected 1 service in AllStats, got %d", len(all))
	}

	// Check Clear.
	repo.Clear("test-svc")
	if got := repo.Stats("test-svc").TotalLines; got != 0 {
		t.Errorf("expected 0 lines after clear, got %d", got)
	}

	// Check unknown service.
	unknown := repo.Stats("nonexistent")
	if unknown.TotalLines != 0 {
		t.Error("expected 0 lines for unknown service")
	}
}

func TestMeteredServiceLogRepository_TailPassthrough(t *testing.T) {
	repo := infrastructure.NewMeteredServiceLogRepository(nil)

	entry, _ := domain.NewLogEntry(time.Now(), "svc", domain.StreamStdout, "msg")
	repo.Append("svc", entry)

	tail := repo.Tail("svc", 10)
	if len(tail) != 1 {
		t.Errorf("expected 1 entry, got %d", len(tail))
	}
	if tail[0].Message != "msg" {
		t.Errorf("expected 'msg', got %q", tail[0].Message)
	}
}

func TestMeteredServiceLogRepository_ImplementsInterfaces(t *testing.T) {
	var repo interface{} = infrastructure.NewMeteredServiceLogRepository(nil)
	if _, ok := repo.(domain.ServiceLogRepository); !ok {
		t.Error("MeteredServiceLogRepository does not implement ServiceLogRepository")
	}
	if _, ok := repo.(domain.ServiceLogStatsProvider); !ok {
		t.Error("MeteredServiceLogRepository does not implement ServiceLogStatsProvider")
	}
}
