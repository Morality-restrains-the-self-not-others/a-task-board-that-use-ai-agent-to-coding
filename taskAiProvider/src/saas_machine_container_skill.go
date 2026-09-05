package main

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
	"tracelog"
)

const saasMachineContainerSkillRel = "docs/skills/saas-container/saas-machine-container.md"

func (a *App) skillCatalog() ([]domain.SkillVersionEntry, string, error) {
	root := ""
	if a != nil && a.Cfg != nil {
		root = a.Cfg.RepoRoot
	}
	return infrastructure.LoadSaasInboundSkillCatalog(root)
}

func (a *App) writableSkillVersion(raw string) (string, error) {
	entries, _, err := a.skillCatalog()
	if err != nil {
		return "", err
	}
	if err := domain.ValidateSaasInboundSkillVersion(raw, entries); err != nil {
		return "", err
	}
	return domain.NormalizeSkillVersion(raw), nil
}

func (a *App) handleSaasInboundSkillVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	entries, current, err := a.skillCatalog()
	if err != nil {
		tracelog.EmitWithTrace(
			tracelog.TraceIDFromContext(r.Context()),
			"warn",
			"SaasInboundSkillCatalogLoadFailed",
			"ai-provider",
			map[string]string{"err": err.Error()},
		)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "无法加载容器→SaaS 接口版本目录"})
		return
	}
	vers := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		n := domain.NormalizeSkillVersion(e.Version)
		vers = append(vers, map[string]any{
			"version":  n,
			"status":   e.Status,
			"released": e.Released,
			"summary":  e.Summary,
			"doc_href": "/saas-machine-container.md?version=" + n,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": current, "versions": vers})
}

func (a *App) handleSaasMachineContainerSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	root := ""
	if a != nil && a.Cfg != nil {
		root = a.Cfg.RepoRoot
	}
	docName := "saas-machine-container.md"
	qVer := strings.TrimSpace(r.URL.Query().Get("version"))
	if qVer != "" {
		entries, _, catErr := a.skillCatalog()
		if catErr != nil {
			tracelog.EmitWithTrace(
				tracelog.TraceIDFromContext(r.Context()),
				"warn",
				"SaasInboundSkillCatalogLoadFailed",
				"ai-provider",
				map[string]string{"err": catErr.Error()},
			)
			http.NotFound(w, r)
			return
		}
		doc, ok := domain.SkillVersionDocFile(entries, qVer)
		if !ok {
			http.NotFound(w, r)
			return
		}
		docName = doc
	}
	raw, err := infrastructure.ReadSaasInboundSkillDoc(root, docName)
	if err != nil {
		tracelog.EmitWithTrace(
			tracelog.TraceIDFromContext(r.Context()),
			"warn",
			"SaasMachineContainerSkillMissing",
			"ai-provider",
			map[string]string{"path": saasMachineContainerSkillRel, "err": err.Error()},
		)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, docName, time.Time{}, bytes.NewReader(raw))
}
