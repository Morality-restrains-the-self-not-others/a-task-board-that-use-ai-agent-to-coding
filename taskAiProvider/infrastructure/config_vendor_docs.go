package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"confload"
	"taskAiProvider/domain"

	"gopkg.in/yaml.v3"
)

const (
	vendorDocsConfApp      = "ai/ai-provider"
	vendorDocsPathFragment = "vendor-docs-path.yaml"
	defaultVendorDocsRule  = "{keyPrefix}/{userId}/{kind}_{id}{ext}"
	defaultVendorDocsPref  = "vendor-docs"
)

type VendorDocsCOSConfig struct {
	Bucket             string
	Region             string
	KeyPrefix          string
	PathRule           string
	SSE                string
	PresignTTLSeconds  int
	SecretID           string
	SecretKey          string
	CORSAllowedOrigins []string
}

type VendorDocsPathState struct {
	mu        sync.RWMutex
	KeyPrefix string
	PathRule  string
}

func NewVendorDocsPathState(cfg *Config) *VendorDocsPathState {
	prefix := defaultVendorDocsPref
	rule := defaultVendorDocsRule
	if cfg != nil {
		if v := strings.TrimSpace(cfg.VendorDocsCOS.KeyPrefix); v != "" {
			prefix = v
		}
		if v := strings.TrimSpace(cfg.VendorDocsCOS.PathRule); v != "" {
			rule = v
		}
	}
	return &VendorDocsPathState{KeyPrefix: prefix, PathRule: rule}
}

func (s *VendorDocsPathState) Get() (keyPrefix, pathRule string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.KeyPrefix, s.PathRule
}

func (s *VendorDocsPathState) Set(keyPrefix, pathRule string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.KeyPrefix = keyPrefix
	s.PathRule = pathRule
}

func applyVendorDocsConfig(cfg *Config, root string) {
	if cfg.VendorDocsBackend == "" {
		cfg.VendorDocsBackend = domain.VendorDocBackendLocal
	}
	if cfg.VendorDocsCOS.Bucket == "" {
		cfg.VendorDocsCOS.Bucket = "ai-provider-1259712831"
	}
	if cfg.VendorDocsCOS.Region == "" {
		cfg.VendorDocsCOS.Region = "ap-shanghai"
	}
	if cfg.VendorDocsCOS.KeyPrefix == "" {
		cfg.VendorDocsCOS.KeyPrefix = defaultVendorDocsPref
	}
	if cfg.VendorDocsCOS.PathRule == "" {
		cfg.VendorDocsCOS.PathRule = defaultVendorDocsRule
	}
	if cfg.VendorDocsCOS.SSE == "" {
		cfg.VendorDocsCOS.SSE = "AES256"
	}
	if cfg.VendorDocsCOS.PresignTTLSeconds <= 0 {
		cfg.VendorDocsCOS.PresignTTLSeconds = 300
	}

	var file struct {
		VendorDocsDir string `yaml:"vendorDocsDir"`
		VendorDocs    struct {
			Backend  string `yaml:"backend"`
			LocalDir string `yaml:"localDir"`
			COS      struct {
				Bucket             string   `yaml:"bucket"`
				Region             string   `yaml:"region"`
				KeyPrefix          string   `yaml:"keyPrefix"`
				PathRule           string   `yaml:"pathRule"`
				SSE                string   `yaml:"sse"`
				PresignTTLSeconds  int      `yaml:"presignTTLSeconds"`
				SecretID           string   `yaml:"secretId"`
				SecretKey          string   `yaml:"secretKey"`
				CORSAllowedOrigins []string `yaml:"corsAllowedOrigins"`
			} `yaml:"cos"`
		} `yaml:"vendorDocs"`
	}
	if err := confload.ReadAppConfig(root, vendorDocsConfApp, &file); err == nil {
		if v := strings.TrimSpace(file.VendorDocs.Backend); v != "" {
			cfg.VendorDocsBackend = strings.ToLower(v)
		}
		if v := strings.TrimSpace(file.VendorDocs.LocalDir); v != "" {
			cfg.VendorDocsDir = resolvePlaceholders(root, v)
		} else if v := strings.TrimSpace(file.VendorDocsDir); v != "" {
			cfg.VendorDocsDir = resolvePlaceholders(root, v)
		}
		c := file.VendorDocs.COS
		if v := strings.TrimSpace(c.Bucket); v != "" {
			cfg.VendorDocsCOS.Bucket = v
		}
		if v := strings.TrimSpace(c.Region); v != "" {
			cfg.VendorDocsCOS.Region = v
		}
		if v := strings.TrimSpace(c.KeyPrefix); v != "" {
			cfg.VendorDocsCOS.KeyPrefix = v
		}
		if v := strings.TrimSpace(c.PathRule); v != "" {
			cfg.VendorDocsCOS.PathRule = v
		}
		if v := strings.TrimSpace(c.SSE); v != "" {
			cfg.VendorDocsCOS.SSE = v
		}
		if c.PresignTTLSeconds > 0 {
			cfg.VendorDocsCOS.PresignTTLSeconds = c.PresignTTLSeconds
		}
		if v := strings.TrimSpace(c.SecretID); v != "" {
			cfg.VendorDocsCOS.SecretID = v
		}
		if v := strings.TrimSpace(c.SecretKey); v != "" {
			cfg.VendorDocsCOS.SecretKey = v
		}
		if len(c.CORSAllowedOrigins) > 0 {
			cfg.VendorDocsCOS.CORSAllowedOrigins = c.CORSAllowedOrigins
		}
	}
	var frag struct {
		KeyPrefix string `yaml:"keyPrefix"`
		PathRule  string `yaml:"pathRule"`
	}
	if err := confload.ReadAppFragment(root, vendorDocsConfApp, vendorDocsPathFragment, &frag); err == nil {
		if v := strings.TrimSpace(frag.KeyPrefix); v != "" {
			cfg.VendorDocsCOS.KeyPrefix = v
		}
		if v := strings.TrimSpace(frag.PathRule); v != "" {
			cfg.VendorDocsCOS.PathRule = v
		}
	}
	if env := strings.TrimSpace(os.Getenv("AI_PROVIDER_VENDOR_DOCS_DIR")); env != "" {
		cfg.VendorDocsDir = env
	}
}

func WriteVendorDocsPathFragment(root, keyPrefix, pathRule string) error {
	if err := domain.ValidatePathRule(pathRule); err != nil {
		return err
	}
	if strings.TrimSpace(keyPrefix) == "" || strings.Contains(keyPrefix, "..") || strings.Contains(keyPrefix, "/") {
		return fmt.Errorf("invalid key prefix")
	}
	dir := filepath.Join(root, "conf", "ai", "ai-provider")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	body, err := yaml.Marshal(map[string]string{
		"keyPrefix": strings.TrimSpace(keyPrefix),
		"pathRule":  strings.TrimSpace(pathRule),
	})
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, vendorDocsPathFragment+".tmp")
	if err := os.WriteFile(tmp, body, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, vendorDocsPathFragment))
}
