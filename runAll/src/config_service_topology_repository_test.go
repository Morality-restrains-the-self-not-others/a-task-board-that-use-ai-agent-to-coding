package main

import (
	"testing"

	"runAll/src/domain"
)

func TestConfigServiceTopologyRepository_ListAll(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{
				Name: "g1",
				Services: []Service{
					{Name: "a", Command: "true", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
					{Name: "b", Command: "true", DependsOn: []string{"a"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}
	repo := newConfigServiceTopologyRepository(cfg)
	nodes, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("len(nodes) = %d, want 2", len(nodes))
	}
	if nodes[0].Name != "a" || nodes[0].GroupName != "g1" {
		t.Fatalf("node[0] = %#v", nodes[0])
	}
	if len(nodes[1].DependsOn) != 1 || nodes[1].DependsOn[0] != "a" {
		t.Fatalf("node[1] = %#v", nodes[1])
	}
}

func TestConfigServiceTopologyRepository_SkipStartAllPropagatesFromGroup(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Groups: []Group{
			{
				Name:         "gitlab-regions",
				SkipStartAll: true,
				Services: []Service{
					{Name: "git-service", Command: "true", DependsOn: []string{"docker-redis"}, HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
			{
				Name: "infrastructure",
				Services: []Service{
					{Name: "docker-redis", Command: "true", HealthCheck: HealthCheck{URL: "http://127.0.0.1:1"}},
				},
			},
		},
	}
	nodes, err := newConfigServiceTopologyRepository(cfg).ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	byName := map[string]domain.ServiceTopologyNode{}
	for _, n := range nodes {
		byName[n.Name] = n
	}
	if !byName["git-service"].SkipStartAll {
		t.Fatal("git-service SkipStartAll = false, want true from group skip_start_all")
	}
	if byName["docker-redis"].SkipStartAll {
		t.Fatal("docker-redis SkipStartAll = true, want false")
	}
}
