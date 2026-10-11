package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"runAll/src/infrastructure"

	"gopkg.in/yaml.v3"
)

const bootstrapAdminConfRel = "auth/task-auth/config.yaml"

func validateBootstrapAdminEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("bootstrapAdmin.email is empty or invalid")
	}
	if strings.ContainsAny(email, "'\\\"\n\r\x00;") {
		return "", fmt.Errorf("bootstrapAdmin.email contains unsafe characters")
	}
	return email, nil
}

func resolveRepoRootForBootstrapAdmin(runner *Runner) (string, error) {
	cfgPath := ""
	if runner != nil {
		cfgPath = strings.TrimSpace(runner.cfgPath)
	}
	return infrastructure.ResolveMonorepoRoot(cfgPath)
}

func readBootstrapAdminEmail(repoRoot string) (string, error) {
	trackedPath := filepath.Join(repoRoot, "conf", filepath.FromSlash(bootstrapAdminConfRel))
	data, err := os.ReadFile(trackedPath)
	if err != nil {
		return "", fmt.Errorf("read tracked conf: %w", err)
	}
	merged, err := overlayConfLocalRel(repoRoot, bootstrapAdminConfRel, data)
	if err != nil {
		return "", err
	}
	var block struct {
		BootstrapAdmin *struct {
			Email string `yaml:"email"`
		} `yaml:"bootstrapAdmin"`
	}
	if err := yaml.Unmarshal(merged, &block); err != nil {
		return "", fmt.Errorf("parse bootstrapAdmin: %w", err)
	}
	if block.BootstrapAdmin == nil {
		return "", nil
	}
	return strings.TrimSpace(block.BootstrapAdmin.Email), nil
}

func writeBootstrapAdminEmailConfLocal(repoRoot, email string) error {
	email, err := validateBootstrapAdminEmail(email)
	if err != nil {
		return err
	}
	localPath := filepath.Join(repoRoot, "conf-local", filepath.FromSlash(bootstrapAdminConfRel))
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	existing := map[string]any{}
	if raw, err := os.ReadFile(localPath); err == nil {
		if err := yaml.Unmarshal(raw, &existing); err != nil {
			return fmt.Errorf("parse existing conf-local: %w", err)
		}
		if existing == nil {
			existing = map[string]any{}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	ba, _ := existing["bootstrapAdmin"].(map[string]any)
	if ba == nil {
		ba = map[string]any{}
	}
	ba["email"] = email
	existing["bootstrapAdmin"] = ba
	out, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath, out, 0o600)
}

func handleBootstrapAdminEmail(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if !allowDevDatabaseReset(r) {
		writeJSONErrorWithStatus(w, http.StatusForbidden, "dev bootstrap-admin email not allowed on this host")
		return
	}
	root, err := resolveRepoRootForBootstrapAdmin(runner)
	if err != nil {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		email, err := readBootstrapAdminEmail(root)
		if err != nil {
			writeJSONErrorWithStatus(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]any{
			"email":   email,
			"set":     email != "" && strings.Contains(email, "@"),
			"source":  "conf+conf-local",
			"message": "初始化全部数据库前须设置管理员邮箱；密码由 bootstrap-admin 随机生成",
		})
	case http.MethodPost:
		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, "invalid json")
			return
		}
		if err := writeBootstrapAdminEmailConfLocal(root, body.Email); err != nil {
			writeJSONErrorWithStatus(w, http.StatusBadRequest, err.Error())
			return
		}
		got, err := readBootstrapAdminEmail(root)
		if err != nil {
			writeJSONErrorWithStatus(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]any{
			"status": "ok",
			"email":  got,
		})
	default:
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
