package dbload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDatabaseEntries_ordersByOrderField(t *testing.T) {
	root, err := FindMonorepoRoot("")
	if err != nil {
		t.Skip(err)
	}
	entries, err := LoadDatabaseEntries(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected multiple entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Order < entries[i-1].Order {
			t.Fatalf("not sorted: %+v", entries)
		}
	}
}

func TestRemoveSQLiteFiles_removesMainAndWal(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "db", "registry.yaml")
	if err := os.MkdirAll(filepath.Dir(registry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registry, []byte(`version: "1"
databases:
  testdb:
    path: db/test/test.sqlite3
    owner: test
    order: 1
    migrate_script: db/test/migrate.sh
    init_script: db/test/init.sh
`), 0o644); err != nil {
		t.Fatal(err)
	}
	dbDir := filepath.Join(root, "db", "test")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dbDir, "test.sqlite3")
	for _, p := range []string{main, main + "-wal", main + "-shm"} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries := []DatabaseEntry{{Key: "testdb", Path: "db/test/test.sqlite3", Order: 1}}
	removed, err := RemoveSQLiteFiles(root, entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "testdb" {
		t.Fatalf("removed = %v", removed)
	}
	if _, err := os.Stat(main); !os.IsNotExist(err) {
		t.Fatalf("main still exists")
	}
}
