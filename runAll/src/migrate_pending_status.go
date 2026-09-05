package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

const migratePendingCacheTTL = 15 * time.Second

type migratePendingCache struct {
	mu     sync.Mutex
	at     time.Time
	report domain.MigratePendingReport
}

func (r *Runner) invalidateMigratePendingCache() {
	if r == nil {
		return
	}
	r.migratePendingCache.mu.Lock()
	r.migratePendingCache.at = time.Time{}
	r.migratePendingCache.mu.Unlock()
}

func (r *Runner) MigratePendingStatus(ctx context.Context, bypassCache bool) domain.MigratePendingReport {
	if r == nil {
		return domain.MigratePendingReport{}
	}
	if !bypassCache {
		r.migratePendingCache.mu.Lock()
		if !r.migratePendingCache.at.IsZero() && time.Since(r.migratePendingCache.at) < migratePendingCacheTTL {
			rep := r.migratePendingCache.report
			r.migratePendingCache.mu.Unlock()
			return rep
		}
		r.migratePendingCache.mu.Unlock()
	}

	root, entries, _, err := r.loadDevDatabaseContext("")
	if err != nil {
		log.Printf("[runall] migrate-status load registry failed err=%v", err)
		return domain.MigratePendingReport{
			Databases: []domain.MigrateDatabaseStatus{{
				Key:    "registry",
				Status: domain.MigrateStatusUnreachable,
				Error:  err.Error(),
			}},
			UnreachableCount: 1,
		}
	}
	resolve := r.migrateDirResolver
	if resolve == nil {
		resolve = infrastructure.FilesystemDataMigrateDirResolver{}
	}
	sqlLister := r.migrateSQLLister
	if sqlLister == nil {
		sqlLister = infrastructure.FilesystemSQLLister{}
	}
	applied := r.migrateAppliedReader
	if applied == nil {
		applied = infrastructure.NewMySQLAppliedStepReader(root)
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rep := domain.BuildMigratePendingReport(ctx, entries, root, resolve, sqlLister, applied)
	log.Printf("[runall] migrate-status pending=%d unreachable=%d databases=%d",
		rep.PendingCount, rep.UnreachableCount, len(rep.Databases))

	r.migratePendingCache.mu.Lock()
	r.migratePendingCache.at = time.Now()
	r.migratePendingCache.report = rep
	r.migratePendingCache.mu.Unlock()
	return rep
}

// bulkStartMigrationBlock returns a non-empty, actionable message when a bulk
// start/restart must be refused because a registry database has unapplied
// dataMigrate SQL. On a fresh clone-run with an empty MySQL volume the schema
// gap would otherwise surface as LAUNCH_PROCESS_EXITED per service (e.g. the
// task-auth loadUserContentTypeID 1146) instead of one message pointing the
// operator at「初始化全部数据库」. Fail-open: when the migrate status cannot be
// determined (registry missing, MySQL unreachable) it returns "" so an empty
// volume can still be brought up through the UI init flow first.
func (r *Runner) bulkStartMigrationBlock(ctx context.Context) string {
	if r == nil {
		return ""
	}
	// Only enforce on a deploy instance that explicitly targets a monorepo root
	// (cutover.env exports MONOREPO_ROOT). In bare unit tests the registry walk
	// can accidentally resolve to the host monorepo and hit the real MySQL,
	// adding latency that breaks UI-mode adoption races in StartAllWithActor.
	if strings.TrimSpace(os.Getenv("MONOREPO_ROOT")) == "" {
		return ""
	}
	rep := r.MigratePendingStatus(ctx, false)
	if rep.PendingCount == 0 {
		return ""
	}
	var dbs []string
	for _, st := range rep.Databases {
		if st.Status == domain.MigrateStatusPending {
			dbs = append(dbs, st.Database)
		}
	}
	sort.Strings(dbs)
	return fmt.Sprintf("有 %d 个数据库存在未应用的 dataMigrate SQL（%s），请先点「初始化全部数据库」再重试",
		rep.PendingCount, strings.Join(dbs, ", "))
}
