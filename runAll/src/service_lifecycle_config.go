package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func lifecycleStrictEnabled() bool {
	v := strings.TrimSpace(os.Getenv("RUNALL_LIFECYCLE_STRICT"))
	return v != "0"
}

func (s *Service) EffectiveStartCommand() string {
	if cmd := strings.TrimSpace(s.StartCommand); cmd != "" {
		return cmd
	}
	return strings.TrimSpace(s.Command)
}

func (s *Service) IsDetachLaunch() bool {
	switch strings.TrimSpace(s.LaunchMode) {
	case "detach":
		return true
	case "attach":
		return false
	default:
		return isDetachLaunchCommand(s.EffectiveStartCommand())
	}
}

func (c *Config) normalizeServiceLifecycleCommands() error {
	strict := lifecycleStrictEnabled()
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			if strings.TrimSpace(svc.StartCommand) == "" && strings.TrimSpace(svc.Command) != "" {
				svc.StartCommand = svc.Command
			}
			if strings.TrimSpace(svc.RestartCommand) != "" {
				log.Printf("service %q: restart_command is deprecated and ignored; restart uses stop_command then start_command", svc.Name)
			}
			if !strict {
				continue
			}
			if strings.TrimSpace(svc.EffectiveStartCommand()) == "" {
				return fmt.Errorf("service %q: start_command is required (lifecycle strict mode)", svc.Name)
			}
			if strings.TrimSpace(svc.StopCommand) == "" {
				return fmt.Errorf("service %q: stop_command is required (lifecycle strict mode)", svc.Name)
			}
		}
	}
	return nil
}
