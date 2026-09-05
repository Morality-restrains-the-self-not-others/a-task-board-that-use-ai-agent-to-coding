package confload

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func mergeOptionalYAML(base map[string]any, path string) map[string]any {
	loc, err := readYAMLFile(path)
	if err != nil {
		return base
	}
	return deepMergeMaps(base, loc)
}

// MergeConfLocal overlays conf-local/<relUnderConf> onto base. Missing file is a no-op.
func MergeConfLocal(root, relUnderConf string, base map[string]any) map[string]any {
	return mergeOptionalYAML(base, filepath.Join(root, "conf-local", filepath.FromSlash(relUnderConf)))
}

// ConfRelFromPath returns (repoRoot, relUnderConf) when path is <root>/conf/<rel>
// and conf/base.yaml exists. Callers that only have a filesystem path use this
// to overlay the matching conf-local file (ADR-0054).
func ConfRelFromPath(path string) (root, rel string, ok bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", false
	}
	dir := abs
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false
		}
		if filepath.Base(parent) == "conf" {
			if _, err := os.Stat(filepath.Join(parent, "base.yaml")); err == nil {
				rel, err := filepath.Rel(parent, abs)
				if err != nil || strings.HasPrefix(rel, "..") {
					return "", "", false
				}
				return filepath.Dir(parent), filepath.ToSlash(rel), true
			}
		}
		dir = parent
	}
}

// ReadYAMLMerged loads conf/<relUnderConf> then overlays conf-local/<relUnderConf>.
// If the tracked file is missing but conf-local exists, the overlay alone is returned.
func ReadYAMLMerged(root, relUnderConf string) (map[string]any, error) {
	path := filepath.Join(root, "conf", filepath.FromSlash(relUnderConf))
	base, err := readYAMLFile(path)
	if err != nil {
		loc := MergeConfLocal(root, relUnderConf, map[string]any{})
		if len(loc) == 0 {
			return nil, err
		}
		return loc, nil
	}
	return MergeConfLocal(root, relUnderConf, base), nil
}

// UnmarshalYAMLMerged unmarshals ReadYAMLMerged into dest and resolves ${subdomains.*}.
func UnmarshalYAMLMerged(root, relUnderConf string, dest any) error {
	base, err := ReadYAMLMerged(root, relUnderConf)
	if err != nil {
		return err
	}
	raw, err := yaml.Marshal(base)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(raw, dest); err != nil {
		return err
	}
	subs := ResolveBaseYaml(root)
	if len(subs) > 0 {
		resolveStructStrings(dest, subs)
	}
	return nil
}

// MergeYAMLAtPath reads a YAML file and, if it lives under conf/, overlays conf-local.
func MergeYAMLAtPath(path string) (map[string]any, error) {
	base, err := readYAMLFile(path)
	if err != nil {
		return nil, err
	}
	root, rel, ok := ConfRelFromPath(path)
	if !ok {
		return base, nil
	}
	return MergeConfLocal(root, rel, base), nil
}

// ReadOverlaidYAML returns the YAML encoding of MergeYAMLAtPath (ADR-0054).
func ReadOverlaidYAML(path string) ([]byte, error) {
	m, err := MergeYAMLAtPath(path)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(m)
}
