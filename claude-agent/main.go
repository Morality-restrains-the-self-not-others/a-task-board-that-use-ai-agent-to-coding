// Claude Agent — Go CLI wrapper for Claude Code.
//
// Usage:
//
//	claude-agent [command]
//
// Commands:
//
//	run            Run a task (wraps `claude -p`)
//	interactive    Start interactive session (foreground passthrough)
//	show-config    Display current configuration
//	tools          List available Claude Code tools
package main

import (
	"fmt"
	"os"

	"claudeAgent/src/cli"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Handle --version / -v
	if os.Args[1] == "--version" || os.Args[1] == "-v" {
		fmt.Printf("claude-agent %s\n", version)
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "run":
		cli.CmdRun(args)
	case "interactive":
		cli.CmdInteractive(args)
	case "show-config":
		cli.CmdShowConfig(args)
	case "tools":
		cli.CmdTools()
	case "session":
		cli.CmdSession(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Claude Agent — CLI wrapper for Claude Code

Usage:
  claude-agent [command]

Commands:
  run            Run a task (wraps ` + "`claude -p`" + `)
  interactive    Start interactive session (foreground passthrough)
  show-config    Display current configuration
  tools          List available Claude Code built-in tools

Run 'claude-agent <command> --help' for more information on a command.`)
}
