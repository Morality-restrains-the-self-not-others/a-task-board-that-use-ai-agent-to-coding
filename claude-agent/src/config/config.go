// Package config provides YAML configuration parsing and value resolution
// for claude-agent. It mirrors the Python version's layered config design:
// CLI flag > environment variable > config file > default.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModelProvider holds LLM provider configuration.
type ModelProvider struct {
	APIKey     string `yaml:"api_key"`
	Provider   string `yaml:"provider"`
	BaseURL    string `yaml:"base_url,omitempty"`
	APIVersion string `yaml:"api_version,omitempty"`
}

// ModelConfig holds model-level configuration.
type ModelConfig struct {
	Model         string         `yaml:"model"`
	ModelProvider *ModelProvider `yaml:"-"`
	ProviderName  string         `yaml:"model_provider"`
	Temperature   float64        `yaml:"temperature"`
	TopP          float64        `yaml:"top_p"`
	MaxTokens     int            `yaml:"max_tokens"`
	MaxRetries    int            `yaml:"max_retries"`
}

// MCPServerConfig holds MCP server configuration.
type MCPServerConfig struct {
	Command     string            `yaml:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	URL         string            `yaml:"url,omitempty"`
	Description string            `yaml:"description,omitempty"`
}

// DockerConfig holds Docker execution configuration.
type DockerConfig struct {
	Image              string `yaml:"image"`
	Keep               bool   `yaml:"keep"`
	ContainerWorkspace string `yaml:"container_workspace"`
	ContainerID        string `yaml:"container_id,omitempty"`
}

// AgentConfig holds agent-level configuration.
type AgentConfig struct {
	Model            string `yaml:"model"`
	MaxSteps         int    `yaml:"max_steps"`
	WorkingDir       string `yaml:"working_dir"`
	EnableTrajectory bool   `yaml:"enable_trajectory"`
	EnableDocker     bool   `yaml:"enable_docker"`
	PermissionMode   string `yaml:"permission_mode"` // "skip" → --dangerously-skip-permissions (autonomous pipelines)
}

// Top-level config mirrors the Python Config dataclass.
type TopLevel struct {
	ModelProviders map[string]ModelProvider   `yaml:"model_providers"`
	Models         map[string]ModelConfig     `yaml:"models"`
	Agents         map[string]AgentConfig     `yaml:"agents"`
	Docker         DockerConfig               `yaml:"docker"`
	MCPServers     map[string]MCPServerConfig `yaml:"mcp_servers"`
	AllowMCP       []string                   `yaml:"allow_mcp_servers"`
}

// Load reads and parses a YAML config file, expanding ${ENV_VAR} placeholders.
func Load(configFile string) (*TopLevel, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", configFile, err)
	}

	// Expand env vars before YAML parsing
	expanded := expandEnvVars(string(data))

	cfg := &TopLevel{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set defaults
	if cfg.Agents == nil {
		return nil, fmt.Errorf("no agents defined in config")
	}
	if cfg.ModelProviders == nil {
		return nil, fmt.Errorf("no model_providers defined in config")
	}
	if cfg.Docker.ContainerWorkspace == "" {
		cfg.Docker.ContainerWorkspace = "/workspace"
	}
	cfg.Docker.Keep = true // default keep=true

	// Wire model providers into model configs
	for name, mc := range cfg.Models {
		prov, ok := cfg.ModelProviders[mc.ProviderName]
		if !ok {
			return nil, fmt.Errorf("model provider %q not found for model %q", mc.ProviderName, name)
		}
		mc.ModelProvider = &prov
		// Set defaults
		if mc.Temperature == 0 {
			mc.Temperature = 0.5
		}
		if mc.TopP == 0 {
			mc.TopP = 1.0
		}
		if mc.MaxTokens == 0 {
			mc.MaxTokens = 4096
		}
		if mc.MaxRetries == 0 {
			mc.MaxRetries = 3
		}
		cfg.Models[name] = mc
	}

	return cfg, nil
}

// ResolvedConfig represents a fully resolved configuration with CLI overrides applied.
type ResolvedConfig struct {
	Provider       string
	Model          string
	BaseURL        string
	APIKey         string
	MaxSteps       int
	WorkingDir     string
	ConfigFile     string
	PermissionMode string
}

// Resolve applies CLI overrides on top of config values with priority:
// CLI flag > env var > config file > default.
func (cfg *TopLevel) Resolve(overrides *ResolvedConfig) (*ResolvedConfig, error) {
	agentName := "claude_agent"
	agentCfg, ok := cfg.Agents[agentName]
	if !ok {
		// fallback: try trae_agent
		agentCfg, ok = cfg.Agents["trae_agent"]
		if !ok {
			return nil, fmt.Errorf("no 'claude_agent' configuration found")
		}
	}

	modelCfg, ok := cfg.Models[agentCfg.Model]
	if !ok {
		return nil, fmt.Errorf("model %q not found in models section", agentCfg.Model)
	}

	result := &ResolvedConfig{
		Provider:       modelCfg.ModelProvider.Provider,
		Model:          modelCfg.Model,
		BaseURL:        modelCfg.ModelProvider.BaseURL,
		APIKey:         modelCfg.ModelProvider.APIKey,
		MaxSteps:       agentCfg.MaxSteps,
		WorkingDir:     agentCfg.WorkingDir,
		PermissionMode: agentCfg.PermissionMode,
	}

	// Apply resolution: CLI > ENV > Config
	if overrides != nil {
		if overrides.Provider != "" {
			result.Provider = overrides.Provider
		} else if v := os.Getenv("CLAUDE_MODEL_PROVIDER"); v != "" {
			result.Provider = v
		}

		if overrides.Model != "" {
			result.Model = overrides.Model
		}

		if overrides.BaseURL != "" {
			result.BaseURL = overrides.BaseURL
		} else {
			envKey := strings.ToUpper(result.Provider) + "_BASE_URL"
			if v := os.Getenv(envKey); v != "" {
				result.BaseURL = v
			}
		}

		if overrides.APIKey != "" {
			result.APIKey = overrides.APIKey
		} else {
			envKey := strings.ToUpper(result.Provider) + "_API_KEY"
			if v := os.Getenv(envKey); v != "" {
				result.APIKey = v
			}
		}

		if overrides.MaxSteps > 0 {
			result.MaxSteps = overrides.MaxSteps
		} else if v := os.Getenv("CLAUDE_MAX_STEPS"); v != "" {
			fmt.Sscanf(v, "%d", &result.MaxSteps)
		}

		if overrides.WorkingDir != "" {
			result.WorkingDir = overrides.WorkingDir
		}
	}

	// Apply env var overrides for config-loaded values too
	if result.APIKey == "" {
		envKey := strings.ToUpper(result.Provider) + "_API_KEY"
		result.APIKey = os.Getenv(envKey)
	}
	if result.BaseURL == "" {
		envKey := strings.ToUpper(result.Provider) + "_BASE_URL"
		result.BaseURL = os.Getenv(envKey)
	}

	if result.MaxSteps == 0 {
		result.MaxSteps = 100
	}

	return result, nil
}

var envVarRe = regexp.MustCompile(`\$\{(\w+)(?::([^}]*))?\}`)

// expandEnvVars replaces ${VAR} and ${VAR:default} placeholders in a string.
// Mirrors the Python version: if VAR is set, returns its value; if VAR has a
// default (${VAR:default}), returns the default; otherwise keeps the literal.
func expandEnvVars(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		varName := parts[1]
		defaultVal := parts[2]
		hasDefault := len(parts) >= 3 && parts[2] != ""

		if val, exists := os.LookupEnv(varName); exists {
			return val
		}
		if hasDefault {
			return defaultVal
		}
		// Keep literal ${VAR} when env var is unset and no default
		return match
	})
}
