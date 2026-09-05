package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ArtifactPin is one binary (or dist) pin in releases.yaml.
type ArtifactPin struct {
	SHA     string `yaml:"sha"`
	Package string `yaml:"package"`
	Dest    string `yaml:"dest,omitempty"`
	Unpack  string `yaml:"unpack,omitempty"`
}

// ReleasesFile is the on-disk pin list. Production secrets never appear here.
type ReleasesFile struct {
	Artifacts map[string]ArtifactPin `yaml:"artifacts"`
}

// ArtifactFetcher downloads a pin to a temporary file path.
// Implementations download a prebuilt artifact; the deploy host has no source tree.
type ArtifactFetcher interface {
	Fetch(pin ArtifactPin) (tmpPath string, err error)
}

// ArtifactFetcherFunc adapts a function to ArtifactFetcher.
type ArtifactFetcherFunc func(pin ArtifactPin) (string, error)

func (f ArtifactFetcherFunc) Fetch(pin ArtifactPin) (string, error) {
	return f(pin)
}

// LoadReleases parses releases.yaml. Empty artifacts or a missing sha is an error.
func LoadReleases(path string) (*ReleasesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[deploy-sync] load releases failed path=%s err=%v", path, err)
		return nil, err
	}
	var rel ReleasesFile
	if err := yaml.Unmarshal(data, &rel); err != nil {
		log.Printf("[deploy-sync] parse releases failed path=%s err=%v", path, err)
		return nil, fmt.Errorf("releases.yaml: %w", err)
	}
	if len(rel.Artifacts) == 0 {
		err := fmt.Errorf("releases.yaml: artifacts empty")
		log.Printf("[deploy-sync] %v", err)
		return nil, err
	}
	for name, pin := range rel.Artifacts {
		if strings.TrimSpace(pin.SHA) == "" {
			err := fmt.Errorf("releases.yaml: artifact %s missing sha", name)
			log.Printf("[deploy-sync] %v", err)
			return nil, err
		}
		unpack := strings.TrimSpace(pin.Unpack)
		switch unpack {
		case "", "tar.gz", "tgz":
		default:
			err := fmt.Errorf("releases.yaml: artifact %s unsupported unpack %q", name, unpack)
			log.Printf("[deploy-sync] %v", err)
			return nil, err
		}
		if unpack != "" && strings.TrimSpace(pin.Dest) == "" {
			err := fmt.Errorf("releases.yaml: artifact %s unpack requires dest", name)
			log.Printf("[deploy-sync] %v", err)
			return nil, err
		}
	}
	log.Printf("[deploy-sync] loaded releases artifacts=%d", len(rel.Artifacts))
	return &rel, nil
}

func deployBinPath(root, name string) string {
	return filepath.Join(root, "bin", name)
}

func shaSidecarPath(binPath string) string {
	return binPath + ".sha"
}

func currentSidecarSHA(sidecarPath string) string {
	b, err := os.ReadFile(sidecarPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func installAtomic(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dest + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// SyncPinnedArtifacts writes ELF pins to $DEPLOY_ROOT/bin/<name> and
// archive pins to pin.Dest. Fetch/install failure leaves last-good untouched.
// Matching sha sidecar skips download (idempotent).
func SyncPinnedArtifacts(root string, rel *ReleasesFile, fetcher ArtifactFetcher) error {
	if rel == nil {
		return fmt.Errorf("releases file is nil")
	}
	if fetcher == nil {
		return fmt.Errorf("artifact fetcher is nil")
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "artifacts"), 0o755); err != nil {
		return err
	}
	var firstErr error
	for name, pin := range rel.Artifacts {
		dest, err := pinDestPath(root, name, pin)
		if err != nil {
			log.Printf("[deploy-sync] dest invalid name=%s err=%v", name, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("dest %s: %w", name, err)
			}
			continue
		}
		sidecar := pinSidecarPath(root, name, dest, pin)
		if currentSidecarSHA(sidecar) == pin.SHA {
			if _, err := os.Stat(dest); err == nil {
				log.Printf("[deploy-sync] skip %s sha=%s already present", name, pin.SHA)
				continue
			}
		}
		log.Printf("[deploy-sync] fetch %s sha=%s dest=%s unpack=%s", name, pin.SHA, dest, pin.Unpack)
		tmp, err := fetcher.Fetch(pin)
		if err != nil {
			log.Printf("[deploy-sync] fetch failed name=%s sha=%s err=%v (last-good kept)", name, pin.SHA, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("fetch %s: %w", name, err)
			}
			continue
		}
		if err := installFetchedPin(root, name, pin, tmp); err != nil {
			log.Printf("[deploy-sync] install failed name=%s err=%v (last-good kept)", name, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("install %s: %w", name, err)
			}
			continue
		}
		if err := os.WriteFile(sidecar, []byte(pin.SHA+"\n"), 0o644); err != nil {
			log.Printf("[deploy-sync] sha sidecar write failed name=%s err=%v", name, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("sha sidecar %s: %w", name, err)
			}
			continue
		}
		log.Printf("[deploy-sync] installed %s sha=%s dest=%s", name, pin.SHA, dest)
	}
	return firstErr
}
