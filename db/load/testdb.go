package dbload

import (
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var randomRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randomSuffix(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = randomRunes[rand.Intn(len(randomRunes))]
	}
	return string(b)
}

// dsnUserRe matches user:password@ in a DSN.
var dsnUserRe = regexp.MustCompile(`^([^:]+):([^@]+)@`)

// dsnTcpRe matches tcp(host:port) in a DSN.
var dsnTcpRe = regexp.MustCompile(`tcp\(([^)]+)\)`)

// dsnDBRe matches /dbname?params suffix.
var dsnDBRe = regexp.MustCompile(`/([^/?]*)(\?.*)?$`)

// extractAdminDSN returns a DSN without a database name, for admin operations
// like CREATE DATABASE / DROP DATABASE.
func extractAdminDSN(dsn string) string {
	// Replace /dbname?... with /?...  (regex $2 includes the leading ?)
	dsn = dsnDBRe.ReplaceAllString(dsn, "/$2")
	// Ensure there's a ? for params
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	}
	if !strings.Contains(dsn, "parseTime=true") {
		dsn += "&parseTime=true"
	}
	if !strings.Contains(dsn, "multiStatements=true") {
		dsn += "&multiStatements=true"
	}
	// Fix any double ?? created when $2 already starts with ?
	dsn = strings.ReplaceAll(dsn, "??", "?")
	return dsn
}

// replaceDBName replaces the database name in a MySQL DSN.
// $2 from dsnDBRe already includes the leading ? if present.
func replaceDBName(dsn, newDB string) string {
	return dsnDBRe.ReplaceAllString(dsn, "/"+newDB+"$2")
}

// runMySQLExecFn is overridable in tests (OPT-20260818-003) to assert the cleanup
// failure path is logged instead of silently swallowed.
var runMySQLExecFn = runMySQLExec

// runMySQLExec runs a SQL statement against the MySQL server identified by dsn.
// Tries the mysql CLI first, then falls back to docker exec on the docker-mysql container.
func runMySQLExec(dsn, sql string) error {
	adminDSN := extractAdminDSN(dsn)
	args := append(mysqlCLIArgs(adminDSN), "-e", sql)

	// Try local mysql CLI first
	if _, err := exec.LookPath("mysql"); err == nil {
		cmd := exec.Command("mysql", args...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		return fmt.Errorf("mysql CLI: %v (output: %s)", err, string(out))
	}

	// Fall back to docker exec
	dockerArgs := []string{"exec", "-i", "docker-mysql-mysql-1", "mysql"}
	dockerArgs = append(dockerArgs, args...)
	cmd := exec.Command("docker", dockerArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker mysql: %v (output: %s)", err, string(out))
	}
	return nil
}

// mysqlCLIArgs extracts the mysql CLI arguments from a DSN.
func mysqlCLIArgs(dsn string) []string {
	var args []string
	// Extract user:password
	if m := dsnUserRe.FindStringSubmatch(dsn); len(m) == 3 {
		args = append(args, "-u"+m[1], "-p"+m[2])
	}
	// Extract host:port
	if m := dsnTcpRe.FindStringSubmatch(dsn); len(m) == 2 {
		hostPort := m[1]
		if strings.Contains(hostPort, ":") {
			parts := strings.SplitN(hostPort, ":", 2)
			args = append(args, "-h"+parts[0], "-P"+parts[1])
		} else {
			args = append(args, "-h"+hostPort)
		}
	}
	return args
}

// OpenTestMySQL creates a unique test database on the configured MySQL server
// and returns a DSN for connecting to it. It uses the mysql CLI to create the
// database, so no additional Go dependencies are required.
//
// The returned cleanup function drops the test database. Callers should defer it.
//
// If MySQL is not reachable or the service key cannot be resolved, an error is
// returned. Callers should typically skip the test in that case.
//
// Usage:
//
//	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot)
//	if err != nil {
//	    t.Skipf("MySQL not available: %v", err)
//	}
//	defer cleanup()
//	db, err := sql.Open("mysql", testDSN)
func OpenTestMySQL(serviceKey, monorepoRoot string) (dsn string, cleanup func(), err error) {
	// 1. Resolve the production DSN to get connection parameters
	base, err := ResolveMySQLDSN(serviceKey, monorepoRoot)
	if err != nil {
		return "", nil, fmt.Errorf("OpenTestMySQL: resolve DSN: %w", err)
	}

	// 2. Generate unique test database name
	cleanServiceKey := strings.ReplaceAll(serviceKey, "-", "_")
	testDBName := cleanServiceKey + "_test_" + randomSuffix(8)

	// 3. Use mysql CLI / docker exec to create the test database
	if err := runMySQLExecFn(base, "CREATE DATABASE IF NOT EXISTS `"+testDBName+
		"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return "", nil, fmt.Errorf("OpenTestMySQL: create database %s: %w", testDBName, err)
	}

	// 4. Build DSN pointing to the new test database
	testDSN := replaceDBName(base, testDBName)

	cleanup = makeTestDBCleanup(base, testDBName)

	return testDSN, cleanup, nil
}

// makeTestDBCleanup returns the cleanup function that drops the test database.
// DROP failures panic so leftover `*_test_*` databases fail the test instead of
// silently filling docker-mysql datadir / tmpfs (was log-only, OPT-20260818-003).
func makeTestDBCleanup(baseDSN, testDBName string) func() {
	return func() {
		if err := runMySQLExecFn(baseDSN, "DROP DATABASE IF EXISTS `"+testDBName+"`"); err != nil {
			// Panic (not log-only): a failed DROP leaves *_test_* schemas on the
			// shared docker-mysql datadir and will fill a tmpfs workspace.
			panic(fmt.Sprintf("[dbload] DROP DATABASE %s failed: %v", testDBName, err))
		}
	}
}

// MySQLAvailable checks whether MySQL is reachable via CLI or Docker.
func MySQLAvailable(dsn string) bool {
	return runMySQLExecFn(dsn, "SELECT 1") == nil
}

// ResolveTestMySQLDSN is a convenience wrapper that resolves a DSN for testing.
// If MySQL cannot be resolved, it returns an error suitable for t.Skip.
func ResolveTestMySQLDSN(serviceKey, monorepoRoot string) (string, error) {
	base, err := ResolveMySQLDSN(serviceKey, monorepoRoot)
	if err != nil {
		return "", fmt.Errorf("MySQL not available for %s (set %s env or ensure db/registry.yaml is configured): %w",
			serviceKey, mysqlDSNEnvByKey[serviceKey], err)
	}
	if !MySQLAvailable(base) {
		return "", errors.New("MySQL server not reachable")
	}
	return base, nil
}
