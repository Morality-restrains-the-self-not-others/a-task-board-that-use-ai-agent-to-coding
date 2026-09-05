package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// RunDeploySync loads releases.yaml and writes $DEPLOY_ROOT/bin without compiling.
func RunDeploySync(root, releasesPath string, fetcher ArtifactFetcher) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("deploy root is empty")
	}
	rel, err := LoadReleases(releasesPath)
	if err != nil {
		return err
	}
	log.Printf("[deploy-sync] start root=%s artifacts=%d", root, len(rel.Artifacts))
	if err := SyncPinnedArtifacts(root, rel, fetcher); err != nil {
		log.Printf("[deploy-sync] failed err=%v", err)
		return err
	}
	log.Printf("[deploy-sync] complete")
	return nil
}

func runDeploySyncMain(configPath string) int {
	root := strings.TrimSpace(os.Getenv("DEPLOY_ROOT"))
	if root == "" {
		abs, err := filepath.Abs(configPath)
		if err != nil {
			log.Printf("[deploy-sync] resolve config path err=%v", err)
			return 1
		}
		root = filepath.Dir(filepath.Dir(abs))
	}
	rel := filepath.Join(root, "releases.yaml")
	if _, err := os.Stat(rel); err != nil {
		rel = filepath.Join(filepath.Dir(configPath), "releases.yaml")
	}
	log.Printf("[deploy-sync] command releases=%s", rel)
	if err := RunDeploySync(root, rel, NewDefaultArtifactFetcher()); err != nil {
		return 1
	}
	return 0
}
