package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func overlayConfLocalRel(root, relUnderConf string, data []byte) ([]byte, error) {
	loc, err := os.ReadFile(filepath.Join(root, "conf-local", filepath.FromSlash(relUnderConf)))
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, fmt.Errorf("read conf-local %s: %w", relUnderConf, err)
	}
	var base map[string]any
	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	var over map[string]any
	if err := yaml.Unmarshal(loc, &over); err != nil {
		return nil, fmt.Errorf("parse conf-local: %w", err)
	}
	if base == nil {
		base = map[string]any{}
	}
	out, err := yaml.Marshal(deepMergeRunAll(base, over))
	if err != nil {
		return nil, fmt.Errorf("marshal merged yaml: %w", err)
	}
	return out, nil
}

// checkConfLocalDoesNotExtendBase rejects a conf-local overlay that grows the
// top-level `groups` or a group's `services` list beyond the tracked base config.
//
// conf-local is a secrets-only mirror (ADR-0054 / constraint 63/64): it may patch
// existing entries (env, health fields), but must never add a group/service. A
// positional list merge cannot tell a shifted sparse overlay from an intentional
// addition, so appending overlay items here would inject nameless (or even fully
// named) services — e.g. after a group is inserted in the middle of conf/runAll.yaml,
// the stale conf-local overlay silently lands on the wrong group (nightly 2026-09-05).
// Fail loudly with a drift hint instead of letting validate() emit a bare
// "service name is required" or, worse, starting an overlay-injected service.
func checkConfLocalDoesNotExtendBase(baseData []byte, merged *Config, configPath string) error {
	var base Config
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return nil // malformed base is reported by the later parse; not an overlay concern
	}
	if len(merged.Groups) > len(base.Groups) {
		return fmt.Errorf("conf-local overlay adds group(s) beyond base config (base=%d merged=%d); conf-local and %s are out of sync",
			len(base.Groups), len(merged.Groups), configPath)
	}
	for i := range base.Groups {
		if i >= len(merged.Groups) {
			break
		}
		baseN := len(base.Groups[i].Services)
		mergedN := len(merged.Groups[i].Services)
		if mergedN > baseN {
			return fmt.Errorf("conf-local overlay adds %d service(s) to group %q beyond base config (base=%d merged=%d); conf-local and %s are out of sync",
				mergedN-baseN, merged.Groups[i].Name, baseN, mergedN, configPath)
		}
	}
	return nil
}

func overlayConfLocal(configPath string, data []byte) ([]byte, error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}
	confDir := filepath.Dir(abs)
	root := filepath.Dir(confDir)
	rel := filepath.Base(abs)
	return overlayConfLocalRel(root, rel, data)
}

func deepMergeRunAll(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		bv, ok := out[k]
		bm, bok := bv.(map[string]any)
		vm, vok := v.(map[string]any)
		if ok && bok && vok {
			out[k] = deepMergeRunAll(bm, vm)
			continue
		}
		bl, isBaseList := bv.([]any)
		vl, isOverList := v.([]any)
		if ok && isBaseList && isOverList {
			merged := append([]any{}, bl...)
			for i, ov := range vl {
				if i < len(merged) {
					left, leftMap := merged[i].(map[string]any)
					right, rightMap := ov.(map[string]any)
					if leftMap && rightMap {
						merged[i] = deepMergeRunAll(left, right)
						continue
					}
					merged[i] = ov
					continue
				}
				merged = append(merged, ov)
			}
			out[k] = merged
			continue
		}
		out[k] = v
	}
	return out
}
