package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitlabRegionServiceStart(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"", "bash gitService/run.sh start"},
		{"git-service", "bash gitService/run.sh start"},
		{"git-service-tencent-sh-1", "bash gitService/scripts/deploy_tencent_sh_1.sh"},
		{"git-service-aws-tokyo-1", "GITSERVICE_CONF_APP=git-service-aws-tokyo-1 bash gitService/run.sh start"},
	}
	for _, tc := range cases {
		if got := gitlabRegionServiceStart(tc.name); got != tc.want {
			t.Fatalf("service=%q start=%q want %q", tc.name, got, tc.want)
		}
	}
	if gitlabRegionServiceStart("git-service") == gitlabRegionServiceStart("git-service-tencent-sh-1") {
		t.Fatal("primary and sh-1 start commands must differ")
	}
}

func TestResolveGitlabRegionDeployInfo_PrimaryUsesLegacyConf(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, "conf", "infra", "git-service")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "gitlabHome: /srv/gitlab-primary\ncontainerName: gitlab\n"
	if err := os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	got := resolveGitlabRegionDeployInfo(root, defaultGitlabRegion)
	if got.ConfigFile != "conf/infra/git-service/config.yaml" {
		t.Fatalf("config_file=%q", got.ConfigFile)
	}
	if got.ServiceProcess != "git-service" {
		t.Fatalf("service_process=%q", got.ServiceProcess)
	}
	if got.DataDir != "/srv/gitlab-primary" {
		t.Fatalf("data_dir=%q", got.DataDir)
	}
	if got.ContainerName != "gitlab" {
		t.Fatalf("container_name=%q", got.ContainerName)
	}
	if got.ServiceStart != "bash gitService/run.sh start" {
		t.Fatalf("service_start=%q", got.ServiceStart)
	}
	if !got.ConfigExists {
		t.Fatal("expected config_exists true")
	}
}

func TestResolveGitlabRegionDeployInfo_SlugSpecificConf(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "conf", "infra", "git-service-tencent-sh-1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "gitlabHome: /var/lib/daydaymoney/gitService-tencent-sh-1\ncontainerName: gitlab-tencent-sh-1\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	got := resolveGitlabRegionDeployInfo(root, "tencent-sh-1")
	if got.ConfigFile != "conf/infra/git-service-tencent-sh-1/config.yaml" {
		t.Fatalf("config_file=%q", got.ConfigFile)
	}
	if got.ServiceProcess != "git-service-tencent-sh-1" {
		t.Fatalf("service_process=%q", got.ServiceProcess)
	}
	if got.DataDir != "/var/lib/daydaymoney/gitService-tencent-sh-1" {
		t.Fatalf("data_dir=%q", got.DataDir)
	}
	if got.ContainerName != "gitlab-tencent-sh-1" {
		t.Fatalf("container_name=%q", got.ContainerName)
	}
	if got.ServiceStart != "bash gitService/scripts/deploy_tencent_sh_1.sh" {
		t.Fatalf("service_start=%q", got.ServiceStart)
	}
}

func TestResolveGitlabRegionDeployInfo_MissingConfStillShowsConvention(t *testing.T) {
	root := t.TempDir()
	got := resolveGitlabRegionDeployInfo(root, "aws-tokyo-1")
	if got.ConfigFile != "conf/infra/git-service-aws-tokyo-1/config.yaml" {
		t.Fatalf("config_file=%q", got.ConfigFile)
	}
	if got.ServiceProcess != "git-service-aws-tokyo-1" {
		t.Fatalf("service_process=%q", got.ServiceProcess)
	}
	wantStart := "GITSERVICE_CONF_APP=git-service-aws-tokyo-1 bash gitService/run.sh start"
	if got.ServiceStart != wantStart {
		t.Fatalf("service_start=%q want %q", got.ServiceStart, wantStart)
	}
	if got.DataDir != "" {
		t.Fatalf("expected empty data_dir, got %q", got.DataDir)
	}
	if got.ConfigExists {
		t.Fatal("expected config_exists false")
	}
}

func TestResolveGitlabRegionDeployInfo_LiveMonorepoPaths(t *testing.T) {
	root, err := findMonorepoRoot()
	if err != nil {
		t.Skip(err)
	}
	primary := resolveGitlabRegionDeployInfo(root, defaultGitlabRegion)
	if primary.ConfigFile != "conf/infra/git-service/config.yaml" {
		t.Fatalf("primary config_file=%q", primary.ConfigFile)
	}
	if !primary.ConfigExists {
		t.Fatal("expected primary conf to exist in monorepo")
	}
	if primary.DataDir == "" {
		t.Fatal("expected primary gitlabHome from conf")
	}
	if primary.ServiceProcess != "git-service" {
		t.Fatalf("primary service=%q", primary.ServiceProcess)
	}

	sh1 := resolveGitlabRegionDeployInfo(root, "tencent-sh-1")
	if sh1.ConfigFile != "conf/infra/git-service-tencent-sh-1/config.yaml" {
		t.Fatalf("sh1 config_file=%q", sh1.ConfigFile)
	}
	if !sh1.ConfigExists {
		t.Fatal("expected sh-1 conf to exist")
	}
	if sh1.DataDir != "/var/lib/daydaymoney/gitService-tencent-sh-1" {
		t.Fatalf("sh1 data_dir=%q", sh1.DataDir)
	}
	if sh1.ServiceProcess != "git-service-tencent-sh-1" {
		t.Fatalf("sh1 service=%q", sh1.ServiceProcess)
	}
	if sh1.ContainerName != "gitlab-tencent-sh-1" {
		t.Fatalf("sh1 container=%q", sh1.ContainerName)
	}
	if primary.ServiceStart != "bash gitService/run.sh start" {
		t.Fatalf("primary service_start=%q", primary.ServiceStart)
	}
	if sh1.ServiceStart != "bash gitService/scripts/deploy_tencent_sh_1.sh" {
		t.Fatalf("sh1 service_start=%q", sh1.ServiceStart)
	}
	if primary.ServiceStart == sh1.ServiceStart {
		t.Fatal("primary and sh-1 must not share the same start command")
	}
}

func TestResolveGitlabRegionDeployInfo_PrimaryPrefersSlugDirWhenPresent(t *testing.T) {
	root := t.TempDir()
	slugDir := filepath.Join(root, "conf", "infra", "git-service-"+defaultGitlabRegion)
	if err := os.MkdirAll(slugDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "gitlabHome: /srv/slug-primary\ncontainerName: gitlab-sh5\n"
	if err := os.WriteFile(filepath.Join(slugDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	got := resolveGitlabRegionDeployInfo(root, defaultGitlabRegion)
	wantCfg := "conf/infra/git-service-" + defaultGitlabRegion + "/config.yaml"
	if got.ConfigFile != wantCfg {
		t.Fatalf("config_file=%q want %q", got.ConfigFile, wantCfg)
	}
	if got.ServiceProcess != "git-service-"+defaultGitlabRegion {
		t.Fatalf("service_process=%q", got.ServiceProcess)
	}
	wantStart := "GITSERVICE_CONF_APP=git-service-" + defaultGitlabRegion + " bash gitService/run.sh start"
	if got.ServiceStart != wantStart {
		t.Fatalf("service_start=%q want %q", got.ServiceStart, wantStart)
	}
	if got.DataDir != "/srv/slug-primary" {
		t.Fatalf("data_dir=%q", got.DataDir)
	}
}
