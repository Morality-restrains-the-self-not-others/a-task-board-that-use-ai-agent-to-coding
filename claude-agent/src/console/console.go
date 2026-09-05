// Package console provides terminal output formatting for claude-agent.
package console

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintHeader prints a styled header line.
func PrintHeader(title string) {
	fmt.Printf("\n%s\n", title)
	fmt.Println(strings.Repeat("─", len([]rune(title))))
}

// PrintTable prints aligned key-value pairs.
func PrintTable(title string, rows [][2]string) {
	if title != "" {
		fmt.Printf("\n%s\n", title)
	}
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintf(w, "  %s\t%s\n", row[0], row[1])
	}
	w.Flush()
}

// PrintSuccess prints a green success message.
func PrintSuccess(msg string) {
	fmt.Printf("\n✓ %s\n", msg)
}

// PrintError prints a red error message.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "\n✗ %s\n", msg)
}

// PrintWarning prints a yellow warning message.
func PrintWarning(msg string) {
	fmt.Printf("\n⚠ %s\n", msg)
}

// PrintInfo prints a blue info message.
func PrintInfo(msg string) {
	fmt.Printf("→ %s\n", msg)
}

// PrintBanner prints a panel-like banner.
func PrintBanner(title string, details map[string]string) {
	titleLen := len(title)
	fmt.Printf("\n┌─%s─┐\n", strings.Repeat("─", titleLen))
	fmt.Printf("│ %s │\n", title)
	fmt.Printf("├─%s─┤\n", strings.Repeat("─", titleLen))
	for k, v := range details {
		fmt.Printf("│ %s: %s\n", k, v)
	}
	fmt.Printf("└─%s─┘\n", strings.Repeat("─", titleLen))
}

// ListTools is the available Claude Code built-in tools.
var ListTools = []struct {
	Name, Desc string
}{
	{"Read", "Read file contents with line numbers"},
	{"Write", "Create or overwrite a file"},
	{"Edit", "Perform exact string replacements in files"},
	{"Bash", "Execute shell commands"},
	{"Glob", "Find files matching patterns"},
	{"Grep", "Search file contents with regex"},
	{"WebSearch", "Search the web for information"},
	{"WebFetch", "Fetch and parse web page content"},
	{"Task", "Manage background tasks"},
	{"NotebookEdit", "Edit Jupyter notebook cells"},
}
