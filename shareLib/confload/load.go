// Package confload reads monorepo conf/<app>/config.yaml.
package confload

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// resolveAppDir finds the config directory for an app. It first tries the direct
// path conf/<app>/, then searches one level of subdirectories (conf/*/<app>/).
// Returns the relative path from the conf root (e.g. "core/sms" or "ai/task-ai-endpoint"). OPT-20260806-057: django 退役
func resolveAppDir(root, app string) (string, error) {
	direct := filepath.Join(root, "conf", app)
	if info, err := os.Stat(direct); err == nil && info.IsDir() {
		return app, nil
	}
	// Search one level of subdirectories for <app>.
	confDir := filepath.Join(root, "conf")
	entries, err := os.ReadDir(confDir)
	if err != nil {
		return "", fmt.Errorf("conf/: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(confDir, entry.Name(), app)
		if info, err2 := os.Stat(candidate); err2 == nil && info.IsDir() {
			return filepath.Join(entry.Name(), app), nil
		}
	}
	return "", fmt.Errorf("conf/%s/ not found (also searched conf/*/<app>/)", app)
}

// ReadAppConfig loads conf/<app>/config.yaml, then conf-local/<app>/config.yaml
// (ADR-0054). It does not read config.local.yaml. GENERATED docker-infra.yaml
// in the same conf directory is still merged if present.
// If the direct path is not found, it searches one level of conf subdirectories:
//
//	conf/core/sms/, conf/ai/task-ai-endpoint/, conf/billing/task-bill/, etc.
func ReadAppConfig(root, app string, dest any) error {
	resolvedApp, err := resolveAppDir(root, app)
	if err != nil {
		log.Printf("[confload] WARNING: %v — using defaults", err)
		return err
	}
	if resolvedApp != app {
		log.Printf("[confload] resolved %q → conf/%s/", app, resolvedApp)
	}
	base, err := readYAMLFile(filepath.Join(root, "conf", resolvedApp, "config.yaml"))
	if err != nil {
		log.Printf("[confload] WARNING: conf/%s/config.yaml: %v — using defaults", resolvedApp, err)
		return err
	}
	base = MergeConfLocal(root, filepath.ToSlash(filepath.Join(resolvedApp, "config.yaml")), base)
	// GENERATED docker-infra.yaml 也必须叠 conf-local（ADR-0054）；禁止只读 tracked 片段。
	base = mergeOptionalFragment(root, filepath.ToSlash(filepath.Join(resolvedApp, "docker-infra.yaml")), base)
	raw, err := yaml.Marshal(base)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(raw, dest); err != nil {
		return err
	}
	// Auto-resolve ${subdomains.xxx} template variables from base.yaml.
	// Strings without ${...} placeholders are unchanged (no-op).
	// Missing base.yaml is silently ignored (empty map returned).
	subs := ResolveBaseYaml(root)
	if len(subs) > 0 {
		resolveStructStrings(dest, subs)
	}
	return nil
}

func mergeOptionalFragment(root, relUnderConf string, base map[string]any) map[string]any {
	path := filepath.Join(root, "conf", filepath.FromSlash(relUnderConf))
	frag, err := readYAMLFile(path)
	if err != nil {
		frag = map[string]any{}
	}
	frag = MergeConfLocal(root, relUnderConf, frag)
	if len(frag) == 0 {
		return base
	}
	return deepMergeMaps(base, frag)
}

func readYAMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func deepMergeMaps(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		bv, ok := out[k]
		bm, bok := bv.(map[string]any)
		vm, vok := v.(map[string]any)
		if ok && bok && vok {
			out[k] = deepMergeMaps(bm, vm)
			continue
		}
		if bl, bok := bv.([]any); ok && bok {
			if vl, vok := v.([]any); vok {
				merged := append([]any{}, bl...)
				for i, ov := range vl {
					if i < len(merged) {
						bm, bmap := merged[i].(map[string]any)
						om, omap := ov.(map[string]any)
						if bmap && omap {
							merged[i] = deepMergeMaps(bm, om)
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
		}
		out[k] = v
	}
	return out
}

// ── Template resolution ( ${subdomains.xxx} from conf/base.yaml ) ──────────

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// resolveEnvVars replaces shell-style ${VAR:-default} placeholders in a string.
func resolveEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		varName := parts[1]
		defaultVal := ""
		if len(parts) >= 3 {
			defaultVal = parts[2]
		}
		if v := strings.TrimSpace(os.Getenv(varName)); v != "" {
			return v
		}
		return defaultVal
	})
}

// ResolveBaseYaml reads conf/base.yaml, overlays conf-local/base.yaml (ADR-0054),
// and returns a map from template key (e.g. "subdomains.gateway") to its
// resolved domain value. Callers use these mappings with ResolveTemplate to
// expand ${subdomains.xxx} placeholders. Addressing is domain-only (no local/IP mode).
func ResolveBaseYaml(repoRoot string) map[string]string {
	result := make(map[string]string)

	baseMap, err := readYAMLFile(filepath.Join(repoRoot, "conf", "base.yaml"))
	if err != nil {
		log.Printf("[confload] WARNING: cannot read base.yaml: %v — template resolution disabled", err)
		return result
	}
	baseMap = MergeConfLocal(repoRoot, "base.yaml", baseMap)
	data, err := yaml.Marshal(baseMap)
	if err != nil {
		log.Printf("[confload] WARNING: base.yaml marshal error: %v", err)
		return result
	}

	var base struct {
		Scheme     string            `yaml:"scheme"`
		BaseDomain string            `yaml:"baseDomain"`
		Subdomains map[string]string `yaml:"subdomains"`
	}
	if err := yaml.Unmarshal(data, &base); err != nil {
		log.Printf("[confload] WARNING: base.yaml parse error: %v", err)
		return result
	}

	// Resolve env vars on scheme / baseDomain; subdomain values may contain
	// ${scheme} / ${baseDomain} / ${infraHost} which are template variables (not env vars).
	scheme := strings.ToLower(strings.Trim(strings.TrimSpace(resolveEnvVars(base.Scheme)), ":/"))
	if scheme == "" {
		scheme = "https"
	}
	baseDomain := resolveEnvVars(base.BaseDomain)
	log.Printf("[confload] base.yaml scheme=%s baseDomain=%s", scheme, baseDomain)
	result["scheme"] = scheme
	result["baseDomain"] = baseDomain

	// Resolve infraHost from base.yaml (raw read to capture key not in struct).
	infraHost := ""
	var baseRaw map[string]any
	if err := yaml.Unmarshal(data, &baseRaw); err == nil {
		if v, ok := baseRaw["infraHost"]; ok {
			if s, ok := v.(string); ok {
				infraHost = resolveEnvVars(s)
			}
		}
	}
	if infraHost != "" {
		result["infraHost"] = infraHost
		log.Printf("[confload] base.yaml infraHost=%s", infraHost)
	}

	for k, v := range base.Subdomains {
		val := strings.ReplaceAll(v, "${scheme}", scheme)
		val = strings.ReplaceAll(val, "${baseDomain}", baseDomain)
		if infraHost != "" {
			val = strings.ReplaceAll(val, "${infraHost}", infraHost)
		}
		result["subdomains."+k] = val
	}

	return result
}

// ResolveTemplate replaces ${subdomains.xxx} and ${baseDomain} placeholders
// in s using the mappings returned by ResolveBaseYaml.
func ResolveTemplate(s string, subs map[string]string) string {
	if s == "" || len(subs) == 0 {
		return s
	}
	for k, v := range subs {
		if k == "_base" {
			continue
		}
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

// ReadAppConfigResolved is an alias for ReadAppConfig. Template resolution
// is now built into ReadAppConfig itself, so this function is equivalent.
// Kept for backward compatibility with existing callers.
func ReadAppConfigResolved(root, app string, dest any) error {
	return ReadAppConfig(root, app, dest)
}

// ReadAppFragment loads a sibling YAML under conf/<app>/<filename>
// （如 sync 生成的 sms.yaml / django.yaml）。仅允许本服务配置目录内的 basename，
// 禁止路径穿越或读其它服务 conf。
func ReadAppFragment(root, app, filename string, dest any) error {
	name := strings.TrimSpace(filename)
	if name == "" || name == "." || name == ".." ||
		strings.Contains(name, "/") || strings.Contains(name, `\`) || strings.Contains(name, "..") {
		return fmt.Errorf("confload: invalid fragment filename %q", filename)
	}
	resolvedApp, err := resolveAppDir(root, app)
	if err != nil {
		log.Printf("[confload] WARNING: %v — fragment %s skipped", err, name)
		return err
	}
	path := filepath.Join(root, "conf", resolvedApp, name)
	appDir := filepath.Join(root, "conf", resolvedApp)
	absApp, err := filepath.Abs(appDir)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absApp, absPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("confload: fragment %q escapes conf/%s/", name, resolvedApp)
	}
	base, err := readYAMLFile(path)
	if err != nil {
		log.Printf("[confload] WARNING: conf/%s/%s: %v", resolvedApp, name, err)
		return err
	}
	base = MergeConfLocal(root, filepath.ToSlash(filepath.Join(resolvedApp, name)), base)
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

// resolveStructStrings walks a struct pointer with reflection and applies
// ResolveTemplate to every exported string field (including nested structs,
// slices, and maps — e.g. wechat.apps.<key>.redirectUri).
func resolveStructStrings(v any, subs map[string]string) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}
	resolveStructStringsImpl(val.Elem(), subs)
}

// resolveOne applies ${subdomains.xxx} template substitution followed by
// ${VAR:-default} env-var expansion to a single string.
func resolveOne(s string, subs map[string]string) string {
	newS := ResolveTemplate(s, subs)
	// Also resolve ${VAR:-default} env-var patterns (e.g. ${INFRA_HOST:-127.0.0.1}).
	return resolveEnvVars(newS)
}

// resolveStructStringsImpl walks a struct value and applies template
// resolution to every exported string field, descending into nested structs,
// pointers, slices, maps, and interfaces. Map values are not addressable, so
// they are resolved on a copy and written back via SetMapIndex.
func resolveStructStringsImpl(val reflect.Value, subs map[string]string) {
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		resolveStructStringsImpl(val.Elem(), subs)
		return
	}
	if val.Kind() != reflect.Struct {
		return
	}
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := val.Field(i)
		fieldType := t.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		resolveField(field, subs)
	}
}

// resolveField applies template resolution to a single reflect.Value,
// dispatching on its kind. Maps and interfaces are handled recursively so
// that map-backed config blocks (wechat.apps, headers, etc.) resolve too.
func resolveField(field reflect.Value, subs map[string]string) {
	switch field.Kind() {
	case reflect.String:
		if field.CanSet() {
			old := field.String()
			newS := resolveOne(old, subs)
			if newS != old {
				field.SetString(newS)
			}
		}
	case reflect.Struct:
		resolveStructStringsImpl(field, subs)
	case reflect.Ptr:
		if !field.IsNil() {
			resolveField(field.Elem(), subs)
		}
	case reflect.Slice:
		for j := 0; j < field.Len(); j++ {
			if field.Index(j).CanSet() {
				resolveField(field.Index(j), subs)
			}
		}
	case reflect.Map:
		if field.IsNil() {
			return
		}
		// Map values are not addressable — resolve a copy, then write it back.
		for _, key := range field.MapKeys() {
			elem := field.MapIndex(key)
			copied := reflect.New(elem.Type()).Elem()
			copied.Set(elem)
			resolveField(copied, subs)
			field.SetMapIndex(key, copied)
		}
	case reflect.Interface:
		if field.IsNil() {
			return
		}
		elem := field.Elem()
		switch elem.Kind() {
		case reflect.String:
			newS := resolveOne(elem.String(), subs)
			if newS != elem.String() {
				field.Set(reflect.ValueOf(newS))
			}
		case reflect.Map, reflect.Slice, reflect.Struct:
			copied := reflect.New(elem.Type()).Elem()
			copied.Set(elem)
			resolveField(copied, subs)
			field.Set(copied)
		case reflect.Ptr:
			if !elem.IsNil() {
				resolveField(elem, subs)
			}
		}
	}
}
