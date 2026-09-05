// Package cli implements the CLI subcommands for claude-agent.
package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudeAgent/src/agent"
	"claudeAgent/src/config"
	"claudeAgent/src/console"
	"claudeAgent/src/process"
	"claudeAgent/src/sessionhub"
	"claudeAgent/src/trajectory"
)

// Common flags shared across commands.
type commonFlags struct {
	configFile string
	provider   string
	model      string
	apiKey     string
	maxSteps   int
	workingDir string
}

func (f *commonFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&f.configFile, "config-file", defaultConfigFile(), "Path to configuration file")
	fs.StringVar(&f.provider, "provider", "", "LLM provider")
	fs.StringVar(&f.provider, "p", "", "LLM provider (short)")
	fs.StringVar(&f.model, "model", "", "Model name")
	fs.StringVar(&f.model, "m", "", "Model name (short)")
	fs.StringVar(&f.apiKey, "api-key", "", "API key")
	fs.StringVar(&f.apiKey, "k", "", "API key (short)")
	fs.IntVar(&f.maxSteps, "max-steps", 0, "Maximum execution turns")
	fs.StringVar(&f.workingDir, "working-dir", "", "Working directory")
	fs.StringVar(&f.workingDir, "w", "", "Working directory (short)")
}

func defaultConfigFile() string {
	if v := os.Getenv("CLAUDE_CONFIG_FILE"); v != "" {
		return v
	}
	return "claude_config.yaml"
}

// resolveConfigFile finds the config file, trying .yaml/.yml extensions.
func resolveConfigFile(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	for _, ext := range []string{".yaml", ".yml"} {
		alt := path + ext
		if _, err := os.Stat(alt); err == nil {
			return alt, nil
		}
	}
	return "", fmt.Errorf("config file not found: %s", path)
}

// CmdRun implements the `run` subcommand.
func CmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var cf commonFlags
	cf.register(fs)

	taskFile := fs.String("file", "", "Read task from file")
	taskFileShort := fs.String("f", "", "Read task from file (short)")
	modelBaseURL := fs.String("model-base-url", "", "Custom API base URL")
	trajectoryFile := fs.String("trajectory-file", "", "Trajectory output path")
	trajectoryFileShort := fs.String("t", "", "Trajectory output path (short)")
	dockerImage := fs.String("docker-image", "", "Docker image for isolation")
	dockerContainerID := fs.String("docker-container-id", "", "Attach to existing container")
	_ = fs.Bool("docker-keep", true, "Keep container after task") // parsed for CLI compat
	sessionRepo := fs.String("session-repo", "", "会话互知: 目标仓库名（自动注册会话+加锁，多仓逗号分隔；冲突则等待后跳过）")
	sessionKind := fs.String("session-kind", "headless", "会话类型（与 --session-repo 配合）")

	fs.Parse(args)

	// Resolve task
	task := ""
	if fs.NArg() > 0 {
		task = fs.Arg(0)
	}
	tf := first(*taskFileShort, *taskFile)
	if tf != "" {
		if task != "" {
			console.PrintError("Cannot use both a task string and --file.")
			os.Exit(1)
		}
		data, err := os.ReadFile(tf)
		if err != nil {
			console.PrintError(fmt.Sprintf("File not found: %s", tf))
			os.Exit(1)
		}
		task = string(data)
	}
	if task == "" {
		console.PrintError("Must provide either a task or --file.")
		os.Exit(1)
	}

	// Resolve config
	cfgFile, err := resolveConfigFile(first(cf.configFile, defaultConfigFile()))
	if err != nil {
		console.PrintError(err.Error())
		os.Exit(1)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	resolved, err := cfg.Resolve(&config.ResolvedConfig{
		Provider: first(cf.provider, ""),
		Model:    first(cf.model, ""),
		BaseURL:  first(*modelBaseURL, ""),
		APIKey:   first(cf.apiKey, ""),
		MaxSteps: cf.maxSteps,
	})
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to resolve config: %v", err))
		os.Exit(1)
	}
	resolved.ConfigFile = cfgFile

	// Set working dir
	wd := first(cf.workingDir, resolved.WorkingDir)
	if wd == "" {
		wd, _ = os.Getwd()
	}
	absWd, _ := filepath.Abs(wd)
	os.MkdirAll(absWd, 0755)
	resolved.WorkingDir = absWd
	console.PrintInfo(fmt.Sprintf("Working directory: %s", absWd))

	// Handle Docker mode
	if *dockerImage != "" && *dockerContainerID != "" {
		console.PrintError("--docker-image and --docker-container-id are mutually exclusive.")
		os.Exit(1)
	}
	if *dockerImage != "" || *dockerContainerID != "" {
		console.PrintInfo(fmt.Sprintf("Docker mode enabled. Image: %s%s", *dockerImage, *dockerContainerID))
		// Docker isolation is handled at a higher level; for now, note it's not implemented in Go yet
		console.PrintWarning("Docker isolation not yet implemented in Go version. Running locally.")
	}

	// Set up trajectory
	trajPath := first(*trajectoryFileShort, *trajectoryFile)
	rec := trajectory.New(trajPath)

	// 会话互知（headless）：注册 + 目标仓加锁（字典序）；冲突等待后跳过
	var sessionSID string
	if *sessionRepo != "" {
		h, herr := sessionhub.NewFromEnv(absWd)
		if herr != nil {
			console.PrintError(fmt.Sprintf("session-hub: %v", herr))
			os.Exit(1)
		}
		sid, regErr := h.Register(sessionhub.Session{
			Kind:     *sessionKind,
			PID:      os.Getpid(),
			StartCwd: absWd,
			Note:     "claude-agent run (headless)",
		})
		if regErr != nil {
			console.PrintError(fmt.Sprintf("session register: %v", regErr))
			os.Exit(1)
		}
		sessionSID = sid
		repos := strings.Split(*sessionRepo, ",")
		items, acqErr := h.AcquireMany(repos, sid, 3*time.Minute)
		if acqErr != nil {
			console.PrintError(fmt.Sprintf("session acquire: %v", acqErr))
			_ = h.Unregister(sid)
			os.Exit(1)
		}
		for _, it := range items {
			if it.Result == sessionhub.HeldBy || it.Result == sessionhub.DeadlockDetected {
				console.PrintWarning(fmt.Sprintf("仓库 %s 被其他会话持有，跳过本会话执行", it.Repo))
			} else {
				console.PrintInfo(fmt.Sprintf("已获取仓库锁 %s (%s)", it.Repo, it.Result))
			}
		}
	}
	defer func() {
		if sessionSID != "" {
			if h, err := sessionhub.NewFromEnv(absWd); err == nil {
				_ = h.Unregister(sessionSID)
				console.PrintInfo("会话已注销（锁已释放）")
			}
		}
	}()

	// Create agent and execute
	a, err := agent.NewClaudeAgent(resolved, rec)
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to create agent: %v", err))
		os.Exit(1)
	}

	ctx := context.Background()
	execution, err := a.Run(ctx, task)
	if err != nil {
		console.PrintError(fmt.Sprintf("Unexpected error: %v", err))
		os.Exit(1)
	}

	if execution.Success {
		console.PrintSuccess("Task completed successfully")
		if execution.FinalResult != "" {
			result := execution.FinalResult
			if len(result) > 2000 {
				result = result[:2000]
			}
			fmt.Printf("\n%s\n", result)
		}
	} else {
		console.PrintError("Task failed")
		if execution.ErrorMessage != "" {
			msg := execution.ErrorMessage
			if len(msg) > 1000 {
				msg = msg[:1000]
			}
			fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		}
	}

	console.PrintInfo(fmt.Sprintf("Trajectory saved to: %s", rec.Path()))
}

// CmdInteractive implements the `interactive` subcommand.
func CmdInteractive(args []string) {
	fs := flag.NewFlagSet("interactive", flag.ExitOnError)
	var cf commonFlags
	cf.register(fs)

	trajectoryFile := fs.String("trajectory-file", "", "Trajectory output path")
	trajectoryFileShort := fs.String("t", "", "Trajectory output path (short)")

	fs.Parse(args)

	cfgFile, err := resolveConfigFile(first(cf.configFile, defaultConfigFile()))
	if err != nil {
		console.PrintError(err.Error())
		os.Exit(1)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	resolved, err := cfg.Resolve(&config.ResolvedConfig{
		Provider: cf.provider,
		Model:    cf.model,
		APIKey:   cf.apiKey,
		MaxSteps: cf.maxSteps,
	})
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to resolve config: %v", err))
		os.Exit(1)
	}

	wd := first(cf.workingDir, resolved.WorkingDir)
	if wd != "" {
		os.Chdir(wd)
		console.PrintInfo(fmt.Sprintf("Working directory: %s", wd))
		resolved.WorkingDir = wd
	}

	_ = first(*trajectoryFileShort, *trajectoryFile)
	rec := trajectory.New(first(*trajectoryFileShort, *trajectoryFile))

	a, err := agent.NewClaudeAgent(resolved, rec)
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to create agent: %v", err))
		os.Exit(1)
	}

	console.PrintBanner("Claude Agent — Interactive Mode", map[string]string{
		"Provider": resolved.Provider,
		"Model":    resolved.Model,
		"Config":   cfgFile,
	})
	console.PrintWarning("Spawning Claude Code... Type /exit or Ctrl+C to quit.")

	exitCode, err := a.RunInteractive()
	if err != nil {
		console.PrintError(fmt.Sprintf("Session error: %v", err))
		os.Exit(1)
	}
	if exitCode != 0 {
		console.PrintWarning(fmt.Sprintf("Claude Code exited with code %d", exitCode))
	} else {
		console.PrintSuccess("Session ended.")
	}
}

// CmdShowConfig implements the `show-config` subcommand.
func CmdShowConfig(args []string) {
	fs := flag.NewFlagSet("show-config", flag.ExitOnError)
	var cf commonFlags
	cf.register(fs)

	modelBaseURL := fs.String("model-base-url", "", "Custom API base URL")

	fs.Parse(args)

	cfgFile, err := resolveConfigFile(first(cf.configFile, defaultConfigFile()))
	if err != nil {
		console.PrintBanner("Configuration Status", map[string]string{
			"Status": fmt.Sprintf("No config file found at: %s", first(cf.configFile, defaultConfigFile())),
			"Note":   "Using default settings and environment variables.",
		})
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		console.PrintWarning(fmt.Sprintf("Cannot load config: %v — showing defaults", err))
		cfg = &config.TopLevel{}
	}

	resolved, err := cfg.Resolve(&config.ResolvedConfig{
		Provider: cf.provider,
		Model:    cf.model,
		BaseURL:  *modelBaseURL,
		APIKey:   cf.apiKey,
		MaxSteps: cf.maxSteps,
	})
	if err != nil {
		console.PrintError(fmt.Sprintf("Failed to resolve config: %v", err))
		os.Exit(1)
	}

	wd := resolved.WorkingDir
	if wd == "" {
		wd, _ = os.Getwd()
	}

	// General settings table
	console.PrintTable("General Settings", [][2]string{
		{"Setting", "Value"},
		{"Provider", resolved.Provider},
		{"Model", resolved.Model},
		{"Max Steps", fmt.Sprintf("%d", resolved.MaxSteps)},
		{"Working Dir", wd},
	})

	// API key display
	apiKeyDisplay := "Not set"
	if resolved.APIKey != "" && len(resolved.APIKey) > 12 {
		apiKeyDisplay = fmt.Sprintf("Set (%s...%s)", resolved.APIKey[:8], resolved.APIKey[len(resolved.APIKey)-4:])
	} else if resolved.APIKey != "" {
		apiKeyDisplay = "Set"
	}

	console.PrintTable("Model Configuration", [][2]string{
		{"Setting", "Value"},
		{"API Key", apiKeyDisplay},
		{"Base URL", resolved.BaseURL},
	})

	// Claude CLI check
	if process.IsAvailable() {
		console.PrintSuccess("Claude Code CLI detected")
	} else {
		console.PrintWarning("Claude Code CLI not detected in PATH")
	}
}

// CmdTools implements the `tools` subcommand.
func CmdTools() {
	console.PrintTable("Claude Code Built-in Tools", func() [][2]string {
		rows := [][2]string{{"Tool", "Description"}}
		for _, t := range console.ListTools {
			rows = append(rows, [2]string{t.Name, t.Desc})
		}
		return rows
	}())

	fmt.Println("\nThese are Claude Code's built-in tools. Claude Agent wraps the Claude Code CLI")
	fmt.Println("without modifying its tool system. (Claude Code is used under the Apache 2.0 license.)")
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
