package main

import (
	"os"
	"path/filepath"
	"strings"

	"confload"
)

// GitlabRegionDeployInfo is the deploy-recipe paths shown on SystemAdmin region cards.
// SSOT remains conf/infra/git-service[-<slug>]/config.yaml (ADR-0014); these fields are resolved for display.
type GitlabRegionDeployInfo struct {
	ServiceProcess string `json:"service_process"`
	ServiceStart   string `json:"service_start"`
	ConfigFile     string `json:"config_file"`
	DataDir        string `json:"data_dir"`
	ContainerName  string `json:"container_name,omitempty"`
	ConfApp        string `json:"conf_app"`
	ConfigExists   bool   `json:"config_exists"`
}

type gitServiceDeployYAML struct {
	GitlabHome    string `yaml:"gitlabHome"`
	ContainerName string `yaml:"containerName"`
}

// resolveGitlabRegionDeployInfo maps a region slug to runAll/conf/data paths.
// Prefer conf/infra/git-service-<slug>/ when present; primary seed falls back to conf/infra/git-service/.
func resolveGitlabRegionDeployInfo(repoRoot, slug string) GitlabRegionDeployInfo {
	slug = strings.TrimSpace(slug)
	slugRel := filepath.Join("conf", "infra", "git-service-"+slug)
	legacyRel := filepath.Join("conf", "infra", "git-service")

	confRel := slugRel
	serviceName := "git-service-" + slug
	if slug == "" {
		confRel = legacyRel
		serviceName = "git-service"
	} else if !dirHasConfigYAML(repoRoot, slugRel) && slug == defaultGitlabRegion && dirHasConfigYAML(repoRoot, legacyRel) {
		confRel = legacyRel
		serviceName = "git-service"
	} else if !dirHasConfigYAML(repoRoot, slugRel) && slug == defaultGitlabRegion {
		// Expected primary layout even before conf is created.
		confRel = legacyRel
		serviceName = "git-service"
	}

	configRel := filepath.ToSlash(filepath.Join(confRel, "config.yaml"))
	confApp := strings.TrimPrefix(filepath.ToSlash(confRel), "conf/")
	info := GitlabRegionDeployInfo{
		ServiceProcess: serviceName,
		ServiceStart:   gitlabRegionServiceStart(serviceName),
		ConfigFile:     configRel,
		ConfApp:        confApp,
		ConfigExists:   fileExists(filepath.Join(repoRoot, filepath.FromSlash(configRel))),
	}

	var y gitServiceDeployYAML
	if err := confload.UnmarshalYAMLMerged(repoRoot, strings.TrimPrefix(filepath.ToSlash(configRel), "conf/"), &y); err != nil {
		return info
	}
	info.DataDir = strings.TrimSpace(y.GitlabHome)
	info.ContainerName = strings.TrimSpace(y.ContainerName)
	if info.ContainerName == "" && serviceName == "git-service" {
		info.ContainerName = "gitlab"
	}
	return info
}

// gitlabRegionServiceStart returns the copy-paste start command for a resolved
// git-service process. Primary matches conf/runAll.yaml; SH-1 uses the dedicated
// deploy wrapper; other slugs set GITSERVICE_CONF_APP so run.sh does not hit 8012.
func gitlabRegionServiceStart(serviceName string) string {
	name := strings.TrimSpace(serviceName)
	switch name {
	case "", "git-service":
		return "bash gitService/run.sh start"
	case "git-service-tencent-sh-1":
		return "bash gitService/scripts/deploy_tencent_sh_1.sh"
	default:
		return "GITSERVICE_CONF_APP=" + name + " bash gitService/run.sh start"
	}
}

func dirHasConfigYAML(repoRoot, relDir string) bool {
	return fileExists(filepath.Join(repoRoot, relDir, "config.yaml"))
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func attachDeployInfo(regions []GitlabRegion) []map[string]interface{} {
	root, err := findMonorepoRoot()
	if err != nil {
		root = ""
	}
	out := make([]map[string]interface{}, 0, len(regions))
	for _, r := range regions {
		rd := remainingCapacity(r.TotalDiskGB, r.AllocatedDiskGB)
		rt := remainingCapacity(r.TotalTrafficGB, r.AllocatedTrafficGB)
		rb := clampRemainingBandwidth(r.TotalBandwidthMbps, r.RemainingBandwidthMbps)
		deploy := resolveGitlabRegionDeployInfo(root, r.Slug)
		out = append(out, map[string]interface{}{
			"id":                       r.ID,
			"name":                     r.Name,
			"slug":                     r.Slug,
			"description":              r.Description,
			"gitlab_api_base":          r.GitlabAPIBase,
			"gitlab_web_url":           r.GitlabWebURL,
			"admin_private_token":      r.AdminPrivateToken,
			"cloud_provider":           r.CloudProvider,
			"is_active":                r.IsActive,
			"sort_order":               r.SortOrder,
			"total_disk_gb":            r.TotalDiskGB,
			"total_traffic_gb":         r.TotalTrafficGB,
			"allocated_disk_gb":        r.AllocatedDiskGB,
			"allocated_traffic_gb":     r.AllocatedTrafficGB,
			"remaining_disk_gb":        rd,
			"remaining_traffic_gb":     rt,
			"bandwidth_shared":         r.BandwidthShared,
			"total_bandwidth_mbps":     r.TotalBandwidthMbps,
			"remaining_bandwidth_mbps": rb,
			"access_mode":              NormalizeGitlabRegionAccessMode(r.AccessMode),
			"infra_status":             NormalizeGitlabRegionInfraStatus(r.InfraStatus),
			"service_process":          deploy.ServiceProcess,
			"service_start":            deploy.ServiceStart,
			"config_file":              deploy.ConfigFile,
			"data_dir":                 deploy.DataDir,
			"container_name":           deploy.ContainerName,
			"conf_app":                 deploy.ConfApp,
			"config_exists":            deploy.ConfigExists,
		})
	}
	return out
}
