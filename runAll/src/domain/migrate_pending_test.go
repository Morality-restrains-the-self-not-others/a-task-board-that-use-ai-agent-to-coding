package domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDiffStepKeys_missingCountsAsPending(t *testing.T) {
	missing, stale := DiffStepKeys(
		[]string{"001_schema.sql", "002_foo.sql"},
		[]string{"001_schema.sql"},
	)
	if len(missing) != 1 || missing[0] != "002_foo.sql" {
		t.Fatalf("missing=%v", missing)
	}
	if len(stale) != 0 {
		t.Fatalf("stale=%v", stale)
	}
}

func TestDiffStepKeys_staleDoesNotAffectMissing(t *testing.T) {
	missing, stale := DiffStepKeys(
		[]string{"001_schema.sql"},
		[]string{"001_schema.sql", "010_oidc_bootstrap_clients"},
	)
	if len(missing) != 0 {
		t.Fatalf("missing=%v want empty", missing)
	}
	if len(stale) != 1 || stale[0] != "010_oidc_bootstrap_clients" {
		t.Fatalf("stale=%v", stale)
	}
}

func TestDiffStepKeys_emptyAppliedTreatsAllLocalAsMissing(t *testing.T) {
	missing, stale := DiffStepKeys([]string{"001.sql", "002.sql"}, nil)
	if len(missing) != 2 {
		t.Fatalf("missing=%v", missing)
	}
	if len(stale) != 0 {
		t.Fatalf("stale=%v", stale)
	}
}

type mapDirResolver map[string]string

func (m mapDirResolver) Resolve(_ string, migrateScript string) (string, bool) {
	dir, ok := m[migrateScript]
	return dir, ok
}

type mapSQLLister map[string][]string

func (m mapSQLLister) ListSQL(_ context.Context, absDir string) ([]string, error) {
	return append([]string(nil), m[absDir]...), nil
}

type mapAppliedReader struct {
	rows map[string][]string
	errs map[string]error
}

func (m mapAppliedReader) ListApplied(_ context.Context, databaseName string) ([]string, error) {
	if err := m.errs[databaseName]; err != nil {
		return nil, err
	}
	return append([]string(nil), m.rows[databaseName]...), nil
}

func TestBuildMigratePendingReport_missingIncrementsPending(t *testing.T) {
	entries := []RegisteredDatabase{{
		Key: "task-bill", Driver: "mysql", Database: "task_bill",
		MigrateScript: "db/task-bill/migrate.sh",
	}}
	rep := BuildMigratePendingReport(context.Background(), entries, "/repo",
		mapDirResolver{"db/task-bill/migrate.sh": "/repo/dataMigrate/taskBill"},
		mapSQLLister{"/repo/dataMigrate/taskBill": {"001.sql", "002.sql"}},
		mapAppliedReader{rows: map[string][]string{"task_bill": {"001.sql"}}},
	)
	if rep.PendingCount != 1 {
		t.Fatalf("pending=%d", rep.PendingCount)
	}
	if rep.UnreachableCount != 0 {
		t.Fatalf("unreachable=%d", rep.UnreachableCount)
	}
	if len(rep.Databases) != 1 || rep.Databases[0].Status != "pending" {
		t.Fatalf("db=%+v", rep.Databases)
	}
	if len(rep.Databases[0].Missing) != 1 || rep.Databases[0].Missing[0] != "002.sql" {
		t.Fatalf("missing=%v", rep.Databases[0].Missing)
	}
}

func TestBuildMigratePendingReport_staleIsOk(t *testing.T) {
	entries := []RegisteredDatabase{{
		Key: "task-auth", Driver: "mysql", Database: "task_auth",
		MigrateScript: "db/task-auth/migrate.sh",
	}}
	rep := BuildMigratePendingReport(context.Background(), entries, "/repo",
		mapDirResolver{"db/task-auth/migrate.sh": "/repo/dataMigrate/taskAuth"},
		mapSQLLister{"/repo/dataMigrate/taskAuth": {"001.sql"}},
		mapAppliedReader{rows: map[string][]string{"task_auth": {"001.sql", "go-step"}}},
	)
	if rep.PendingCount != 0 {
		t.Fatalf("pending=%d want 0 (stale must not count)", rep.PendingCount)
	}
	if rep.Databases[0].Status != "ok" {
		t.Fatalf("status=%s", rep.Databases[0].Status)
	}
	if len(rep.Databases[0].Stale) != 1 {
		t.Fatalf("stale=%v", rep.Databases[0].Stale)
	}
}

func TestBuildMigratePendingReport_unreachableNotPending(t *testing.T) {
	entries := []RegisteredDatabase{{
		Key: "task-auth", Driver: "mysql", Database: "task_auth",
		MigrateScript: "db/task-auth/migrate.sh",
	}}
	rep := BuildMigratePendingReport(context.Background(), entries, "/repo",
		mapDirResolver{"db/task-auth/migrate.sh": "/repo/dataMigrate/taskAuth"},
		mapSQLLister{"/repo/dataMigrate/taskAuth": {"001.sql"}},
		mapAppliedReader{errs: map[string]error{"task_auth": errors.New("can't connect")}},
	)
	if rep.PendingCount != 0 {
		t.Fatalf("pending=%d", rep.PendingCount)
	}
	if rep.UnreachableCount != 1 {
		t.Fatalf("unreachable=%d", rep.UnreachableCount)
	}
	if rep.Databases[0].Status != "unreachable" {
		t.Fatalf("status=%s", rep.Databases[0].Status)
	}
}

func TestBuildMigratePendingReport_sqliteSkipped(t *testing.T) {
	entries := []RegisteredDatabase{{
		Key: "legacy", Driver: "sqlite", Path: "db/x.sqlite3",
	}}
	rep := BuildMigratePendingReport(context.Background(), entries, "/repo",
		mapDirResolver{}, mapSQLLister{}, mapAppliedReader{})
	if rep.PendingCount != 0 || len(rep.Databases) != 1 || rep.Databases[0].Status != "skipped" {
		t.Fatalf("%+v", rep)
	}
}

// trackingAppliedReader records the max number of concurrent ListApplied calls
// and sleeps per call so overlap can be asserted without wall-clock flakiness.
type trackingAppliedReader struct {
	rows      map[string][]string
	delay     time.Duration
	mu        sync.Mutex
	active    int
	maxActive int
}

func (m *trackingAppliedReader) ListApplied(_ context.Context, databaseName string) ([]string, error) {
	m.mu.Lock()
	m.active++
	if m.active > m.maxActive {
		m.maxActive = m.active
	}
	m.mu.Unlock()
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.active--
	m.mu.Unlock()
	return append([]string(nil), m.rows[databaseName]...), nil
}

func mustMySQLEntries(n int) ([]RegisteredDatabase, map[string][]string) {
	entries := make([]RegisteredDatabase, 0, n)
	rows := make(map[string][]string, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("svc-%02d", i)
		db := fmt.Sprintf("db_%02d", i)
		entries = append(entries, RegisteredDatabase{
			Key: key, Driver: "mysql", Database: db, MigrateScript: "db/x/migrate.sh",
		})
		rows[db] = nil
	}
	return entries, rows
}

// The worker pool must actually overlap ListApplied calls, otherwise the badge
// stays serial and the concurrency is fake.
func TestBuildMigratePendingReport_inspectsDatabasesConcurrently(t *testing.T) {
	entries, rows := mustMySQLEntries(12)
	reader := &trackingAppliedReader{rows: rows, delay: 30 * time.Millisecond}
	rep := BuildMigratePendingReport(context.Background(), entries, "/repo",
		mapDirResolver{"db/x/migrate.sh": "/repo/dataMigrate/x"},
		mapSQLLister{"/repo/dataMigrate/x": {"001.sql"}},
		reader,
	)
	if rep.PendingCount != 12 || rep.UnreachableCount != 0 {
		t.Fatalf("rep=%+v", rep)
	}
	if reader.maxActive < 2 {
		t.Fatalf("maxActive=%d want >=2 (ListApplied must overlap)", reader.maxActive)
	}
}

// With a slow fake reader the pooled report must complete well under the
// serial estimate (n * per-db delay), proving the 4-worker pool is effective.
func TestBuildMigratePendingReport_concurrentFasterThanSerial(t *testing.T) {
	const perDB = 50 * time.Millisecond
	entries, rows := mustMySQLEntries(12)
	reader := &trackingAppliedReader{rows: rows, delay: perDB}
	start := time.Now()
	rep := BuildMigratePendingReport(context.Background(), entries, "/repo",
		mapDirResolver{"db/x/migrate.sh": "/repo/dataMigrate/x"},
		mapSQLLister{"/repo/dataMigrate/x": {"001.sql"}},
		reader,
	)
	elapsed := time.Since(start)
	if rep.PendingCount != 12 || rep.UnreachableCount != 0 {
		t.Fatalf("rep=%+v", rep)
	}
	serialEstimate := perDB * time.Duration(len(entries))
	if elapsed >= serialEstimate {
		t.Fatalf("concurrent elapsed=%v not faster than serial estimate=%v (12 DBs x %v)", elapsed, serialEstimate, perDB)
	}
}
