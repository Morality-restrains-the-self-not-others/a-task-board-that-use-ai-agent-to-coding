package config

import (
	"path/filepath"

	"confload"
)

type domainEventsGlobalYAML struct {
	Transport      string `yaml:"transport"`
	InternalSecret string `yaml:"internalSecret"`
	Kafka          struct {
		BootstrapServers string `yaml:"bootstrapServers"`
	} `yaml:"kafka"`
	Redis struct {
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		DB              int    `yaml:"db"`
		StreamKeyPrefix string `yaml:"streamKeyPrefix"`
	} `yaml:"redis"`
}

type eventConfigYAML struct {
	Intents map[string]consumerBlock `yaml:"intents"`
}

// FindMonorepoRoot locates the deploy/repo root that contains conf/.
// Must honor CONF_ROOT / DEPLOY_ROOT: clone-run cwd is envs/current/<svc>,
// whose conf/base.yaml wins a cwd walk and has no sibling conf-local secrets.
func FindMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}

func loadDomainEventsGlobal(root string) (domainEventsGlobalYAML, error) {
	var out domainEventsGlobalYAML
	if err := confload.UnmarshalYAMLMerged(root, "events/domain-events/config.yaml", &out); err != nil {
		return out, err
	}
	var frag domainEventsGlobalYAML
	if err := confload.UnmarshalYAMLMerged(root, "events/domain-events/docker-infra.yaml", &frag); err != nil {
		return out, nil
	}
	if frag.Redis.Host != "" {
		out.Redis.Host = frag.Redis.Host
	}
	if frag.Redis.Port != 0 {
		out.Redis.Port = frag.Redis.Port
	}
	if frag.Kafka.BootstrapServers != "" {
		out.Kafka.BootstrapServers = frag.Kafka.BootstrapServers
	}
	return out, nil
}

func intentFromEventYAML(root, eventSlug, intentSlug string) (consumerBlock, bool) {
	rel := filepath.ToSlash(filepath.Join("events/domain-events", eventSlug, "config.yaml"))
	var ev eventConfigYAML
	if err := confload.UnmarshalYAMLMerged(root, rel, &ev); err != nil || len(ev.Intents) == 0 {
		return consumerBlock{}, false
	}
	block, ok := ev.Intents[intentSlug]
	return block, ok
}
