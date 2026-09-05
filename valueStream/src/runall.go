package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type runAllConfig struct {
	Groups []struct {
		Services []struct {
			Name string `yaml:"name"`
		} `yaml:"services"`
	} `yaml:"groups"`
}

func LoadRunAllServiceNames(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read runall config: %w", err)
	}
	var cfg runAllConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse runall config: %w", err)
	}
	names := make(map[string]bool)
	for _, g := range cfg.Groups {
		for _, svc := range g.Services {
			if svc.Name == "" {
				continue
			}
			names[svc.Name] = true
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("runall config %q: no services found", path)
	}
	// Orchestrator meta-provider: value-stream documents runAll UI/API with runall.* fields.
	names["runall"] = true
	return names, nil
}
