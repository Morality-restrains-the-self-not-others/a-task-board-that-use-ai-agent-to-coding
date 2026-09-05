package domain

import (
	"context"
	"sort"
	"strings"
	"sync"
)

const (
	MigrateStatusOK          = "ok"
	MigrateStatusPending     = "pending"
	MigrateStatusUnreachable = "unreachable"
	MigrateStatusSkipped     = "skipped"

	// migrateInspectWorkers bounds concurrent ListApplied per registry DB.
	// Kept small: each worker opens its own MySQL connection, and the 9999
	// migrate-status badge must not become a connection storm.
	migrateInspectWorkers = 4
)

// MigrateDatabaseStatus is the pending-migrate inspection of one registry DB.
type MigrateDatabaseStatus struct {
	Key          string   `json:"key"`
	Database     string   `json:"database"`
	Status       string   `json:"status"`
	Missing      []string `json:"missing,omitempty"`
	Stale        []string `json:"stale,omitempty"`
	LocalCount   int      `json:"local_count"`
	AppliedCount int      `json:"applied_count"`
	Error        string   `json:"error,omitempty"`
}

// MigratePendingReport aggregates unapplied dataMigrate SQL vs data_migrate_log.
type MigratePendingReport struct {
	PendingCount     int                     `json:"pending_count"`
	UnreachableCount int                     `json:"unreachable_count"`
	Databases        []MigrateDatabaseStatus `json:"databases"`
}

// AppliedStepReader lists data_migrate_log.step_key values.
// Missing table must return empty slice, not error.
type AppliedStepReader interface {
	ListApplied(ctx context.Context, databaseName string) ([]string, error)
}

// LocalSQLLister lists *.sql basenames in a dataMigrate directory.
type LocalSQLLister interface {
	ListSQL(ctx context.Context, absDir string) ([]string, error)
}

// DataMigrateDirResolver maps migrate.sh path to an absolute dataMigrate dir.
type DataMigrateDirResolver interface {
	Resolve(monorepoRoot, migrateScript string) (absDir string, ok bool)
}

// DiffStepKeys returns local-only (missing) and applied-only (stale) step keys.
func DiffStepKeys(local, applied []string) (missing, stale []string) {
	localSet := make(map[string]struct{}, len(local))
	for _, name := range local {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		localSet[name] = struct{}{}
	}
	appliedSet := make(map[string]struct{}, len(applied))
	for _, name := range applied {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		appliedSet[name] = struct{}{}
	}
	for name := range localSet {
		if _, ok := appliedSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	for name := range appliedSet {
		if _, ok := localSet[name]; !ok {
			stale = append(stale, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	return missing, stale
}

// BuildMigratePendingReport compares registry MySQL databases to local SQL files.
// Inspection runs on a fixed worker pool (migrateInspectWorkers) so the 9999
// badge stays fast with many registry databases; result order follows entries.
func BuildMigratePendingReport(
	ctx context.Context,
	entries []RegisteredDatabase,
	monorepoRoot string,
	resolve DataMigrateDirResolver,
	sqlLister LocalSQLLister,
	appliedReader AppliedStepReader,
) MigratePendingReport {
	out := MigratePendingReport{Databases: make([]MigrateDatabaseStatus, len(entries))}
	if len(entries) == 0 {
		return out
	}
	sem := make(chan struct{}, migrateInspectWorkers)
	var wg sync.WaitGroup
	for i := range entries {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				out.Databases[idx] = MigrateDatabaseStatus{
					Key:      entries[idx].Key,
					Database: entries[idx].Database,
					Status:   MigrateStatusUnreachable,
					Error:    "context canceled before inspect",
				}
				return
			}
			defer func() { <-sem }()
			out.Databases[idx] = inspectOneDatabase(ctx, entries[idx], monorepoRoot, resolve, sqlLister, appliedReader)
		}(i)
	}
	wg.Wait()
	for _, st := range out.Databases {
		switch st.Status {
		case MigrateStatusPending:
			out.PendingCount += len(st.Missing)
		case MigrateStatusUnreachable:
			out.UnreachableCount++
		}
	}
	return out
}

func inspectOneDatabase(
	ctx context.Context,
	e RegisteredDatabase,
	monorepoRoot string,
	resolve DataMigrateDirResolver,
	sqlLister LocalSQLLister,
	appliedReader AppliedStepReader,
) MigrateDatabaseStatus {
	st := MigrateDatabaseStatus{Key: e.Key, Database: e.Database}
	if e.Driver != "mysql" {
		st.Status = MigrateStatusSkipped
		return st
	}
	if strings.TrimSpace(e.Database) == "" {
		st.Status = MigrateStatusSkipped
		st.Error = "mysql database name missing in registry"
		return st
	}
	if resolve == nil || sqlLister == nil || appliedReader == nil {
		st.Status = MigrateStatusUnreachable
		st.Error = "migrate pending ports not configured"
		return st
	}
	dir, ok := resolve.Resolve(monorepoRoot, e.MigrateScript)
	if !ok || dir == "" {
		st.Status = MigrateStatusSkipped
		return st
	}
	local, err := sqlLister.ListSQL(ctx, dir)
	if err != nil {
		st.Status = MigrateStatusSkipped
		st.Error = err.Error()
		return st
	}
	applied, err := appliedReader.ListApplied(ctx, e.Database)
	if err != nil {
		st.Status = MigrateStatusUnreachable
		st.Error = err.Error()
		st.LocalCount = len(local)
		return st
	}
	missing, stale := DiffStepKeys(local, applied)
	st.Missing = missing
	st.Stale = stale
	st.LocalCount = len(local)
	st.AppliedCount = len(applied)
	if len(missing) > 0 {
		st.Status = MigrateStatusPending
		return st
	}
	st.Status = MigrateStatusOK
	return st
}
