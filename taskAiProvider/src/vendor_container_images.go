package main

import (
	"encoding/json"
	"net/http"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func (a *App) handleVendorContainerImages(w http.ResponseWriter, r *http.Request) {
	v, ok := a.requireVendor(w, r)
	if !ok {
		return
	}
	if isResolveTargetArchitecturesPath(r.URL.Path) {
		a.handleResolveTargetArchitectures(w, r)
		return
	}
	if id, action, ok := pathAction(r.URL.Path, "/api/vendor/container-images/"); ok {
		img, err := a.DB.GetContainerImage(id)
		if err != nil || img.VendorID != v.ID {
			writeJSON(w, 404, map[string]any{"detail": "not found"})
			return
		}
		if action != "" {
			a.handleVendorContainerAction(w, r, v, img, action)
			return
		}
		switch r.Method {
		case http.MethodGet:
			items, _ := a.DB.ListContainerImages(v.ID)
			for _, it := range items {
				if it["id"] == infrastructure.IDStr(id) {
					writeJSON(w, 200, it)
					return
				}
			}
			writeJSON(w, 404, map[string]any{"detail": "not found"})
		case http.MethodPut, http.MethodPatch:
			var body map[string]any
			_ = readJSON(r, &body)
			fields := map[string]any{}
			if s, ok := body["image_url"].(string); ok {
				fields["image_url"] = s
			}
			if s, ok := body["version"].(string); ok {
				fields["version"] = s
			}
			if s, ok := body["saas_inbound_skill_version"].(string); ok {
				sv, err := a.writableSkillVersion(s)
				if err != nil {
					writeJSON(w, 400, map[string]any{"detail": err.Error()})
					return
				}
				fields["saas_inbound_skill_version"] = sv
			}
			if arch, ok := body["target_architectures"]; ok {
				b, _ := json.Marshal(arch)
				fields["target_architectures"] = string(b)
			}
			needExtract := false
			extractURL, extractVer := img.ImageURL, img.Version
			if s, ok := fields["image_url"].(string); ok {
				extractURL = s
				needExtract = true
			}
			if s, ok := fields["version"].(string); ok {
				extractVer = s
				needExtract = true
			}
			if len(fields) > 0 {
				_ = a.DB.UpdateContainerImageFields(id, fields)
			}
			if needExtract {
				a.scheduleAutoRunAndSkillsExtract(r.Context(), id, extractURL, extractVer)
			}
			if sv, ok := fields["saas_inbound_skill_version"].(string); ok {
				_ = a.eventBus().Publish(r.Context(), domain.EventContainerImageSaasInboundSkillVersionAssigned, map[string]any{
					"image_id":                   infrastructure.IDStr(id),
					"vendor_id":                  infrastructure.IDStr(v.ID),
					"saas_inbound_skill_version": sv,
				})
				logInfo("event=ContainerImageSaasInboundSkillVersionAssigned id=%d skill_version=%s", id, sv)
			}
			items, _ := a.DB.ListContainerImages(v.ID)
			for _, it := range items {
				if it["id"] == infrastructure.IDStr(id) {
					writeJSON(w, 200, it)
					return
				}
			}
			writeJSON(w, 200, map[string]any{"id": infrastructure.IDStr(id)})
		case http.MethodDelete:
			if err := img.DeleteGuard(); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			if err := a.DB.DeleteContainerImage(id); err != nil {
				writeJSON(w, 500, map[string]any{"detail": err.Error()})
				return
			}
			_ = a.eventBus().Publish(r.Context(), domain.EventContainerImageDeleted, map[string]any{
				"image_id":       infrastructure.IDStr(id),
				"vendor_id":      infrastructure.IDStr(v.ID),
				"image_group_id": infrastructure.IDStr(img.ImageGroupID),
			})
			logInfo("event=ContainerImageDeleted id=%d", id)
			w.WriteHeader(204)
		default:
			writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.DB.ListContainerImages(v.ID)
		if err != nil {
			writeJSON(w, 500, map[string]any{"detail": err.Error()})
			return
		}
		writeJSON(w, 200, items)
	case http.MethodPost:
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]any{"detail": "无效 JSON"})
			return
		}
		gid, _ := parseIDFlexible(body["image_group"])
		if gid == 0 {
			gid, _ = parseIDFlexible(body["image_group_id"])
		}
		version, _ := body["version"].(string)
		imageURL, _ := body["image_url"].(string)
		skillRaw, _ := body["saas_inbound_skill_version"].(string)
		skillVer, err := a.writableSkillVersion(skillRaw)
		if err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		var arch []any
		if a, ok := body["target_architectures"].([]any); ok {
			arch = a
		}
		id, err := a.DB.CreateContainerImage(v.ID, gid, version, imageURL, arch, skillVer)
		if err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		a.scheduleAutoRunAndSkillsExtract(r.Context(), id, imageURL, version)
		_ = a.eventBus().Publish(r.Context(), domain.EventContainerImageSaasInboundSkillVersionAssigned, map[string]any{
			"image_id":                   infrastructure.IDStr(id),
			"vendor_id":                  infrastructure.IDStr(v.ID),
			"saas_inbound_skill_version": skillVer,
		})
		logInfo("event=ContainerImageSaasInboundSkillVersionAssigned id=%d skill_version=%s", id, skillVer)
		writeJSON(w, 201, map[string]any{
			"id": infrastructure.IDStr(id), "status": "draft", "version": version,
			"image_url": imageURL, "saas_inbound_skill_version": skillVer,
		})
	default:
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
	}
}
