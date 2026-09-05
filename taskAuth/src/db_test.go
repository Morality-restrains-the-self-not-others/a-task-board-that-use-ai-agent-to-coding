package main

import (
	"sync"
	"testing"
	"time"

	dbload "dbload"
)

// repoRoot resolves the monorepo root, mirroring taskCloudService's single
// shared resolution helper (OPT-20260731-019). It returns "." on failure so
// callers can rely on a non-empty value.
func repoRoot() string {
	root, err := findMonorepoRoot()
	if err != nil || root == "" {
		return "."
	}
	return root
}

// testDBPool holds pre-cloned MySQL test databases. taskAuth has 300+ tests
// that each need a fresh schema, and a sequential dbload clone (~460ms) was the
// dominant wall-clock cost of the suite (nightly sweep `signal: killed` on a
// ~480s budget). Clones are instead produced by a background refiller ahead of
// demand, so a test's setup takes its ready DB off the pool instead of waiting
// for a clone. Tests stay strictly sequential (no t.Parallel), so at most one
// DB is in use at a time; pool size is a burst buffer, not a concurrency level.
const testDBPoolSize = 4

type authTestDB struct {
	dsn     string
	cleanup func()
}

var (
	testDBOnce   sync.Once
	testDBPool   chan *authTestDB
	testDBRefill chan struct{}

	testDBRefillWg sync.WaitGroup

	testDBErrMu sync.Mutex
	testDBErr   error
)

func setTestDBErr(err error) {
	testDBErrMu.Lock()
	if testDBErr == nil {
		testDBErr = err
	}
	testDBErrMu.Unlock()
}

func getTestDBErr() error {
	testDBErrMu.Lock()
	defer testDBErrMu.Unlock()
	return testDBErr
}

func createAuthTestDB() (*authTestDB, error) {
	dsn, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"task-auth", repoRoot(), "dataMigrate/taskAuth",
		func(dsn string) error {
			return runDataMigrateFromDir(dsn, repoRoot())
		},
	)
	if err != nil {
		return nil, err
	}
	return &authTestDB{dsn: dsn, cleanup: cleanup}, nil
}

func initAuthTestDBPool() error {
	testDBOnce.Do(func() {
		testDBPool = make(chan *authTestDB, testDBPoolSize)
		testDBRefill = make(chan struct{}, testDBPoolSize)
		// 初始池并行预克隆；dbload 仅对模板 ensure 加 GET_LOCK，克隆本身并发安全。
		var wg sync.WaitGroup
		for i := 0; i < testDBPoolSize; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tdb, err := createAuthTestDB()
				if err != nil {
					setTestDBErr(err)
					return
				}
				testDBPool <- tdb
			}()
		}
		wg.Wait()
		// 后台补充：每个测试归还空位后克隆一个新库（克隆 ~460ms，远快于
		// 平均单测体 ~1.1s，因此池在稳定态恒有备库）。push 用非阻塞+满则丢弃，
		// 使 TestMain 结束时 close(testDBRefill) 后 Wait() 不会被满池阻塞。
		testDBRefillWg.Add(1)
		go func() {
			defer testDBRefillWg.Done()
			for range testDBRefill {
				tdb, err := createAuthTestDB()
				if err != nil {
					setTestDBErr(err)
					continue
				}
				select {
				case testDBPool <- tdb:
				default:
					tdb.cleanup()
				}
			}
		}()
	})
	return getTestDBErr()
}

var testDBDrainOnce sync.Once

// drainAuthTestDBPool stops the refiller and drops every DB still held by the
// pool. Called from TestMain so a test process never leaves the pre-cloned
// databases behind (constraint 44 — self-cleanup instead of relying on the
// nightly sweep's drop_mysql_test_databases.py).
func drainAuthTestDBPool() {
	if testDBPool == nil {
		return
	}
	testDBDrainOnce.Do(func() {
		close(testDBRefill)
		testDBRefillWg.Wait()
		for {
			select {
			case tdb := <-testDBPool:
				if tdb != nil && tdb.cleanup != nil {
					tdb.cleanup()
				}
			default:
				return
			}
		}
	})
}

// acquireAuthTestDB takes a ready cloned test DB, or skips the test when the
// pool cannot be sustained (MySQL unavailable / pool drained).
func acquireAuthTestDB(t *testing.T) *authTestDB {
	t.Helper()
	if err := initAuthTestDBPool(); err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	select {
	case tdb := <-testDBPool:
		return tdb
	case <-time.After(30 * time.Second):
		if err := getTestDBErr(); err != nil {
			t.Skipf("MySQL test DB pool unavailable: %v", err)
		}
		t.Skipf("MySQL test DB pool drained")
		return nil
	}
}

// setupAuthTestDB opens a fresh per-test MySQL database (pre-cloned by the
// test DB pool, dropped on cleanup), runs the dataMigrate SQL files, and opens
// the global db handle. Tests that need the DB should call this first.
func setupAuthTestDB(t *testing.T) {
	t.Helper()
	tdb := acquireAuthTestDB(t)
	// Drop the DB (and ask the refiller for a replacement) even if openDB
	// fails, so the pool never leaks orphan databases.
	t.Cleanup(func() {
		tdb.cleanup()
		select {
		case testDBRefill <- struct{}{}:
		default:
		}
	})
	if err := openDB(tdb.dsn); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := loadUserContentTypeID(); err != nil {
		t.Fatalf("loadUserContentTypeID: %v", err)
	}
}

func archiveUser(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE auth_user SET is_archived = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("archive user %s: %v", userID, err)
	}
}
