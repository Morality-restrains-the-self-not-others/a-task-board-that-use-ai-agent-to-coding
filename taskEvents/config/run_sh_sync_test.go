package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestRunShIntentPathsMatchAllIntents validates AllIntents integrity and that
// run.sh INTENT_PATHS stays in sync (build/start/stop resolve targets via that array).
// Ports remain YAML SSOT; INTENT_PATHS is still required for run.sh target routing.
func TestRunShIntentPathsMatchAllIntents(t *testing.T) {
	seen := make(map[string]bool)
	for i, def := range AllIntents {
		if def.EventSlug == "" {
			t.Errorf("AllIntents[%d]: EventSlug must not be empty", i)
		}
		if def.IntentSlug == "" {
			t.Errorf("AllIntents[%d]: IntentSlug must not be empty", i)
		}
		if def.Port <= 0 {
			t.Errorf("AllIntents[%d]: Port must be >0 for %s/%s", i, def.EventSlug, def.IntentSlug)
		}
		key := IntentPathKey(def.EventSlug, def.IntentSlug)
		if seen[key] {
			t.Errorf("AllIntents[%d]: duplicate path key %q", i, key)
		}
		seen[key] = true
	}

	root := findRepoRoot(t)
	script, err := os.ReadFile(filepath.Join(root, "run.sh"))
	if err != nil {
		t.Fatalf("read run.sh: %v", err)
	}
	runShPaths := parseBashStringArray(t, string(script), "INTENT_PATHS")
	runShSet := make(map[string]bool, len(runShPaths))
	for _, p := range runShPaths {
		if runShSet[p] {
			t.Errorf("run.sh INTENT_PATHS duplicate %q", p)
		}
		runShSet[p] = true
	}
	for key := range seen {
		if !runShSet[key] {
			t.Errorf("AllIntents path %q missing from run.sh INTENT_PATHS", key)
		}
	}
	for p := range runShSet {
		if !seen[p] {
			t.Errorf("run.sh INTENT_PATHS %q not present in AllIntents", p)
		}
	}
}

// TestDomainEventYAMLPortsMatchAllIntents ensures each AllIntents entry has
// conf/events/domain-events/<event>/config.yaml intents.<intent>.port matching
// the registry (run.sh port_for_path SSOT).
func TestDomainEventYAMLPortsMatchAllIntents(t *testing.T) {
	root := findRepoRoot(t)
	mono := filepath.Dir(root)
	for _, def := range AllIntents {
		confPath := filepath.Join(mono, "conf", "events", "domain-events", def.EventSlug, "config.yaml")
		raw, err := os.ReadFile(confPath)
		if err != nil {
			t.Errorf("%s/%s: missing config.yaml: %v", def.EventSlug, def.IntentSlug, err)
			continue
		}
		re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(def.IntentSlug) + `:\s*(?:\n(?:\s{2,}.*))*?\n\s+port:\s*(\d+)`)
		m := re.FindSubmatch(raw)
		if m == nil {
			// Fallback: simpler scan under intents block
			portRe := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(def.IntentSlug) + `:\s*\n(?:[^\n]*\n)*?\s+port:\s*(\d+)`)
			m = portRe.FindSubmatch(raw)
		}
		if m == nil {
			t.Errorf("%s/%s: intents.%s.port not found in %s", def.EventSlug, def.IntentSlug, def.IntentSlug, confPath)
			continue
		}
		got, err := strconv.Atoi(string(m[1]))
		if err != nil {
			t.Errorf("%s/%s: invalid port %q", def.EventSlug, def.IntentSlug, m[1])
			continue
		}
		if got != def.Port {
			t.Errorf("%s/%s: yaml port=%d AllIntents port=%d", def.EventSlug, def.IntentSlug, got, def.Port)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "run.sh")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("taskEvents repo root with run.sh not found")
		}
		dir = parent
	}
}

func parseBashStringArray(t *testing.T, text, name string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(name) + `=\((.*?)\)`)
	m := re.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%s array not found in run.sh", name)
	}
	var out []string
	for _, tok := range strings.Fields(m[1]) {
		if strings.HasPrefix(tok, "#") {
			continue
		}
		out = append(out, tok)
	}
	return out
}

func parseBashIntArray(t *testing.T, text, name string) []int {
	t.Helper()
	raw := parseBashStringArray(t, text, name)
	out := make([]int, 0, len(raw))
	for _, tok := range raw {
		n, err := strconv.Atoi(tok)
		if err != nil {
			t.Fatalf("%s: invalid port %q: %v", name, tok, err)
		}
		out = append(out, n)
	}
	return out
}
