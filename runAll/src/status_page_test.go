package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssembledStatusPage_ContainsKeyUISnippets(t *testing.T) {
	data, err := statusHTML.ReadFile("status.html")
	if err != nil {
		t.Fatalf("assembled status page: %v", err)
	}
	html := string(data)
	for _, snippet := range []string{
		"runAll UI Color System",
		"function restartAllServices()",
		"function copyLogsToClipboard()",
		"function resolveDepDotClass(",
		`id="restart-all-btn"`,
		// OPT-20260812-004：约束 46 对称 UI（推送 trae-agent 镜像按钮 + JS）
		`id="trae-agent-push-btn"`,
		`id="trae-agent-push-label"`,
		`function refreshTraeAgentPushStatus()`,
		`/api/trae-agent-push`,
		`id="dev-init-db-pending-label"`,
		`function refreshMigratePendingStatus(`,
		`/api/dev/migrate-status`,
		"/*EMBED_CSS*/",
		"/*EMBED_JS*/",
	} {
		if snippet == "/*EMBED_CSS*/" || snippet == "/*EMBED_JS*/" {
			if strings.Contains(html, snippet) {
				t.Fatalf("assembled page still contains marker %q", snippet)
			}
			continue
		}
		if !strings.Contains(html, snippet) {
			t.Fatalf("assembled page missing snippet %q", snippet)
		}
	}
}

// TestAssembledStatusPage_SharedStateBeforeBootstrapCalls guards the TDZ chain that
// blanked #observability-bar with:
// "加载失败: Cannot access 'observabilityState' before initialization".
//
// Root cause: 06.js top-level restartStatusRefreshTimer() sync-reads devLogState via
// currentStatusRefreshMs() before const devLogState ran → script abort → const
// observabilityState never initialized → loadObservabilityBar() async continuation TDZ.
//
// Since OPT-20260811-011 the 6 top-level bootstrap calls are pinned to the END of the
// last fragment (13.js), so they always run after every const *State declaration
// (logsState/observabilityState/devLogState/layoutState/groupUIState, all in 01.js).
func TestAssembledStatusPage_SharedStateBeforeBootstrapCalls(t *testing.T) {
	data, err := statusHTML.ReadFile("status.html")
	if err != nil {
		t.Fatalf("assembled status page: %v", err)
	}
	html := string(data)

	// The 6 bootstrap calls must appear as a contiguous block at the end of 13.js.
	// Match on the block starting at column 0 (the same calls exist inside the
	// wrapped refresh() in 06.js, indented, so this anchor is unambiguous).
	bootstrapBlock := strings.Join([]string{
		"loadObservabilityBar();",
		"refreshCollectionStatus();",
		"refreshTraceShippingStatus();",
		"refreshInternalAPIsSmokeStatus();",
		"refresh();",
		"restartStatusRefreshTimer();",
	}, "\n")
	bi := strings.Index(html, bootstrapBlock)
	if bi < 0 {
		t.Fatalf("assembled page missing end-of-page bootstrap block:\n%s", bootstrapBlock)
	}

	stateDecls := []string{
		`const logsState = {`,
		`const observabilityState = {`,
		`const devLogState = {`,
		`const layoutState = {`,
		`const groupUIState = {};`,
	}
	for _, decl := range stateDecls {
		si := strings.Index(html, decl)
		if si < 0 {
			t.Fatalf("assembled page missing shared state declaration %q", decl)
		}
		if si > bi {
			t.Fatalf("TDZ hazard: %q must appear before bootstrap block (state@%d bootstrap@%d)", decl, si, bi)
		}
	}
}

func TestStatusUIFragments_EachFileWithinLineLimit(t *testing.T) {
	const maxLines = 1000
	err := fs.WalkDir(statusUI, "status_ui", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".html", ".css", ".js":
		default:
			return nil
		}
		raw, readErr := statusUI.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		n := len(strings.Split(string(raw), "\n"))
		if strings.HasSuffix(string(raw), "\n") {
			n--
		}
		// wc -l counts newlines; empty last line nuance — use line count of content.
		lines := strings.Count(string(raw), "\n")
		if !strings.HasSuffix(string(raw), "\n") && len(raw) > 0 {
			lines++
		}
		if lines > maxLines {
			t.Errorf("%s has %d lines (limit %d)", path, lines, maxLines)
		}
		_ = n
		return nil
	})
	if err != nil {
		t.Fatalf("walk status_ui: %v", err)
	}
}

func TestAssembledStatusPage_ProgressPanelManualClear(t *testing.T) {
	data, err := statusHTML.ReadFile("status.html")
	if err != nil {
		t.Fatalf("assembled status page: %v", err)
	}
	html := string(data)
	for _, snippet := range []string{
		`id="prog-clear-logs-btn"`,
		`function dismissProgressPanel(`,
		`function shouldAutoHideProgressPanel(`,
		`onclick="dismissProgressPanel()"`,
	} {
		if !strings.Contains(html, snippet) {
			t.Fatalf("assembled page missing progress-panel manual-clear snippet %q", snippet)
		}
	}
	if strings.Contains(html, "? 10000 : 5000") {
		t.Fatal("assembled page still auto-hides progress panel after 5s/10s")
	}
	if strings.Contains(html, "Auto-hide faster for single service") {
		t.Fatal("assembled page still auto-hides single-service progress")
	}
}

func TestAssembledStatusPage_RefreshDoesNotPollMigrateStatus(t *testing.T) {
	data, err := statusHTML.ReadFile("status.html")
	if err != nil {
		t.Fatalf("assembled status page: %v", err)
	}
	html := string(data)
	start := strings.Index(html, "async function refresh() {")
	if start < 0 {
		t.Fatal("refresh() missing")
	}
	next := strings.Index(html[start+1:], "\nasync function ")
	if next < 0 {
		t.Fatal("could not bound refresh() body")
	}
	body := html[start : start+1+next]
	if strings.Contains(body, "refreshMigratePendingStatus") {
		t.Fatal("refresh() must not call refreshMigratePendingStatus (avoids 2s MySQL polling)")
	}
	if !strings.Contains(html, "function refreshMigratePendingStatus(") {
		t.Fatal("refreshMigratePendingStatus missing from assembled page")
	}
	if !strings.Contains(html, "库内多余") {
		t.Fatal("stale step_keys must appear in tooltip copy")
	}
}
