package confload

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func envPath(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolveExplicitRoot accepts either the conf directory (contains base.yaml)
// or the deploy/repo root (contains conf/base.yaml).
func resolveExplicitRoot(p string) (string, bool) {
	p = filepath.Clean(p)
	if fileExists(filepath.Join(p, "base.yaml")) {
		return filepath.Dir(p), true
	}
	if fileExists(filepath.Join(p, "conf", "base.yaml")) {
		return p, true
	}
	return "", false
}

func findConfigRootByWalk() (string, error) {
	markers := []string{
		filepath.Join("conf", "base.yaml"),
		filepath.Join("dataMigrate"),
		filepath.Join(".gitmodules"),
	}

	findFrom := func(start string) (string, error) {
		dir := filepath.Clean(start)
		for i := 0; i < 16; i++ {
			for _, m := range markers {
				if fileExists(filepath.Join(dir, m)) {
					return dir, nil
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		return "", os.ErrNotExist
	}

	if cwd, err := os.Getwd(); err == nil {
		if root, err := findFrom(cwd); err == nil {
			return root, nil
		}
	}
	if execPath, err := os.Executable(); err == nil {
		if root, err := findFrom(filepath.Dir(execPath)); err == nil {
			return root, nil
		}
	}
	return "", os.ErrNotExist
}

// FindConfigRoot locates the directory that contains conf/ (deploy or repo root).
// Priority: CONF_ROOT, DEPLOY_ROOT, then cwd/executable walk (legacy markers).
func FindConfigRoot() (string, error) {
	if v := envPath("CONF_ROOT"); v != "" {
		if root, ok := resolveExplicitRoot(v); ok {
			log.Printf("[confload] config root from CONF_ROOT")
			return root, nil
		}
		log.Printf("[confload] CONF_ROOT set but no base.yaml; falling through")
	}
	if v := envPath("DEPLOY_ROOT"); v != "" {
		if root, ok := resolveExplicitRoot(v); ok {
			log.Printf("[confload] config root from DEPLOY_ROOT")
			return root, nil
		}
		log.Printf("[confload] DEPLOY_ROOT set but no conf/base.yaml; falling through")
	}
	return findConfigRootByWalk()
}

// FindMonorepoRoot is the historical name; it now delegates to FindConfigRoot
// so binaries on a source-less deploy host honor CONF_ROOT / DEPLOY_ROOT.
func FindMonorepoRoot() (string, error) {
	return FindConfigRoot()
}
