package dbload

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DatabaseEntry is a fully parsed registry row for dev reset orchestration.
type DatabaseEntry struct {
	Key           string
	Driver        string
	Database      string // MySQL database name
	Path          string // legacy SQLite file path
	Owner         string
	Order         int
	MigrateScript string
	InitScript    string
	Description   string
}

type registryEntryYAML struct {
	Driver        string `yaml:"driver"`
	Database      string `yaml:"database"`
	Path          string `yaml:"path"`
	Owner         string `yaml:"owner"`
	Order         int    `yaml:"order"`
	MigrateScript string `yaml:"migrate_script"`
	InitScript    string `yaml:"init_script"`
	Description   string `yaml:"description"`
}

type registryFileFull struct {
	Version   string                       `yaml:"version"`
	MySQL     *MySQLConfig                 `yaml:"mysql"`
	Databases map[string]registryEntryYAML `yaml:"databases"`
}

func LoadDatabaseEntries(monorepoRoot string) ([]DatabaseEntry, error) {
	if monorepoRoot == "" {
		var err error
		monorepoRoot, err = FindMonorepoRoot("")
		if err != nil {
			return nil, err
		}
	}
	path := filepath.Join(monorepoRoot, "db", "registry.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc registryFileFull
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if err := applyMySQLPasswordOverlay(monorepoRoot, &doc.MySQL); err != nil {
		return nil, err
	}
	var out []DatabaseEntry
	for key, block := range doc.Databases {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out = append(out, DatabaseEntry{
			Key:           key,
			Driver:        strings.TrimSpace(block.Driver),
			Database:      strings.TrimSpace(block.Database),
			Path:          strings.TrimSpace(block.Path),
			Owner:         strings.TrimSpace(block.Owner),
			Order:         block.Order,
			MigrateScript: strings.TrimSpace(block.MigrateScript),
			InitScript:    strings.TrimSpace(block.InitScript),
			Description:   strings.TrimSpace(block.Description),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].Key < out[j].Key
		}
		return out[i].Order < out[j].Order
	})
	return out, nil
}

func RemoveSQLiteFiles(monorepoRoot string, entries []DatabaseEntry) ([]string, error) {
	var removed []string
	for _, entry := range entries {
		abs, err := ResolveDatabasePath(entry.Key, monorepoRoot, true)
		if err != nil {
			// MySQL-backed databases may not have legacy SQLite paths; skip gracefully.
			if entry.Driver == "mysql" {
				continue
			}
			return removed, fmt.Errorf("%s: %w", entry.Key, err)
		}
		// Only count as "removed" if at least one file actually existed.
		// This prevents false positives when all databases have been migrated to MySQL
		// but legacy path entries remain in registry.yaml.
		anyRemoved := false
		for _, p := range []string{abs, abs + "-wal", abs + "-shm"} {
			if err := os.Remove(p); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return removed, fmt.Errorf("remove %s: %w", p, err)
			}
			anyRemoved = true
		}
		if anyRemoved {
			removed = append(removed, entry.Key)
		}
	}
	return removed, nil
}
