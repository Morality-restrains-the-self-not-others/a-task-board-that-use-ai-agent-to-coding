package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var designDocFieldNameLine = regexp.MustCompile(`^\s*-\s+name:\s+(\S+)\s*$`)

func TestDesignDocFieldNames(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	specsDir := filepath.Join(root, "docs", "superpowers", "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		t.Skipf("specs dir not found: %v", err)
	}

	var violations []string
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), "-design.md") {
			continue
		}
		path := filepath.Join(specsDir, ent.Name())
		violations = append(violations, scanDesignDocFieldNames(path)...)
	}
	if len(violations) > 0 {
		t.Fatalf("invalid value-stream field names in design docs:\n%s", strings.Join(violations, "\n"))
	}
}

func scanDesignDocFieldNames(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return []string{path + ": open: " + err.Error()}
	}
	defer f.Close()

	var out []string
	inYAML := false
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```yaml") || strings.HasPrefix(trimmed, "```yml") {
			inYAML = true
			continue
		}
		if inYAML && strings.HasPrefix(trimmed, "```") {
			inYAML = false
			continue
		}
		if !inYAML {
			continue
		}
		m := designDocFieldNameLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if !strings.Contains(name, ".") {
			continue
		}
		if _, _, _, err := ParseFieldName(name); err != nil {
			out = append(out, filepath.Base(path)+":"+strconv.Itoa(lineNo)+": "+err.Error())
		}
	}
	if err := sc.Err(); err != nil {
		out = append(out, path+": scan: "+err.Error())
	}
	return out
}
