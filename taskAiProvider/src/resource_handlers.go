package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"

	"tracelog"
)

func (a *App) handleVendorImageGroups(w http.ResponseWriter, r *http.Request) {
	v, ok := a.requireVendor(w, r)
	if !ok {
		return
	}
	switch imageGroupIconSubpath(r.URL.Path) {
	case "upload-url":
		a.handleImageGroupIconUploadURL(w, r, v)
		return
	case "upload-complete":
		a.handleImageGroupIconUploadComplete(w, r, v)
		return
	case "local-put":
		a.handleImageGroupIconLocalPut(w, r, v)
		return
	}
	if id, ok := parseImageGroupPathID(r.URL.Path); ok {
		g, err := a.DB.GetImageGroup(id)
		if err != nil {
			writeJSON(w, 404, map[string]any{"detail": "not found"})
			return
		}
		if g["vendor"] != infrastructure.IDStr(v.ID) {
			writeJSON(w, 403, map[string]any{"detail": "forbidden"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, g)
		case http.MethodPut, http.MethodPatch:
			var body map[string]any
			_ = readJSON(r, &body)
			name, _ := body["name"].(string)
			desc, _ := body["description"].(string)
			iconKey, _ := body["icon_file_key"].(string)
			if name == "" {
				name, _ = g["name"].(string)
			}
			if desc == "" {
				desc, _ = g["description"].(string)
			}
			if strings.TrimSpace(iconKey) == "" {
				iconKey, _ = g["icon_file_key"].(string)
			}
			if err := domain.RequireImageGroupFields(name, desc, iconKey); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			iconKey = strings.TrimSpace(iconKey)
			if !domain.OwnsFileKey(v.ID, iconKey) {
				writeJSON(w, 400, map[string]any{"detail": "图标文件无效"})
				return
			}
			if err := a.DB.UpdateImageGroup(id, strings.TrimSpace(name), strings.TrimSpace(desc), iconKey); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			g, _ = a.DB.GetImageGroup(id)
			_ = a.eventBus().Publish(r.Context(), domain.EventImageGroupUpdated, map[string]any{
				"group_id": id, "vendor_id": v.ID,
			})
			logInfo("event=ImageGroupUpdated id=%d vendor_id=%d", id, v.ID)
			writeJSON(w, 200, g)
		case http.MethodDelete:
			if err := a.DB.DeleteImageGroup(id); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			w.WriteHeader(204)
		default:
			writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.DB.ListImageGroups(v.ID)
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
		name, _ := body["name"].(string)
		desc, _ := body["description"].(string)
		iconKey, _ := body["icon_file_key"].(string)
		if err := domain.RequireImageGroupFields(name, desc, iconKey); err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		iconKey = strings.TrimSpace(iconKey)
		if !domain.OwnsFileKey(v.ID, iconKey) {
			writeJSON(w, 400, map[string]any{"detail": "图标文件无效"})
			return
		}
		id, err := a.DB.CreateImageGroup(v.ID, strings.TrimSpace(name), strings.TrimSpace(desc), iconKey)
		if err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		g, _ := a.DB.GetImageGroup(id)
		_ = a.eventBus().Publish(r.Context(), domain.EventImageGroupCreated, map[string]any{
			"group_id": id, "vendor_id": v.ID,
		})
		logInfo("event=ImageGroupCreated id=%d vendor_id=%d", id, v.ID)
		writeJSON(w, 201, g)
	default:
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
	}
}

func (a *App) handleVendorCloudServerImages(w http.ResponseWriter, r *http.Request) {
	v, ok := a.requireVendor(w, r)
	if !ok {
		return
	}
	if id, ok := parsePathID(r.URL.Path, "/api/vendor/cloud-server-images/"); ok {
		vid, err := a.DB.GetCloudServerImageVendor(id)
		if err != nil || vid != v.ID {
			writeJSON(w, 404, map[string]any{"detail": "not found"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			items, _ := a.DB.ListCloudServerImages(v.ID)
			for _, it := range items {
				if it["id"] == infrastructure.IDStr(id) {
					writeJSON(w, 200, it)
					return
				}
			}
			writeJSON(w, 404, map[string]any{"detail": "not found"})
		case http.MethodDelete:
			_ = a.DB.DeleteCloudServerImage(id)
			w.WriteHeader(204)
		case http.MethodPut, http.MethodPatch:
			var body map[string]any
			_ = readJSON(r, &body)
			if err := a.DB.UpdateCloudServerImage(id, body); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			items, _ := a.DB.ListCloudServerImages(v.ID)
			want := infrastructure.IDStr(id)
			for _, it := range items {
				if it["id"] == want {
					writeJSON(w, 200, it)
					return
				}
			}
			writeJSON(w, 200, map[string]any{"id": want})
		default:
			writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.DB.ListCloudServerImages(v.ID)
		if err != nil {
			writeJSON(w, 500, map[string]any{"detail": err.Error()})
			return
		}
		writeJSON(w, 200, items)
	case http.MethodPost:
		var body map[string]any
		_ = readJSON(r, &body)
		id, err := a.DB.CreateCloudServerImage(v.ID, body)
		if err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		items, _ := a.DB.ListCloudServerImages(v.ID)
		want := infrastructure.IDStr(id)
		for _, it := range items {
			if it["id"] == want {
				writeJSON(w, 201, it)
				return
			}
		}
		writeJSON(w, 201, map[string]any{"id": want})
	default:
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
	}
}

func (a *App) handleVendorUserDataTemplates(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireVendor(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		return
	}
	items, err := a.DB.ListUserDataTemplates(true)
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handleAdminContainerImages(w http.ResponseWriter, r *http.Request) {
	staff, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	if id, action, ok := pathAction(r.URL.Path, "/api/admin/container-images/"); ok {
		if r.Method == http.MethodGet {
			item, err := a.DB.GetContainerImageAdmin(id)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeJSON(w, 404, map[string]any{"detail": "not found"})
					return
				}
				logWarn(r.Context(), "event=AdminContainerImageGetFailed id=%d err=%v", id, err)
				writeJSON(w, 500, map[string]any{"detail": err.Error()})
				return
			}
			writeJSON(w, 200, item)
			return
		}
		img, err := a.DB.GetContainerImage(id)
		if err != nil {
			writeJSON(w, 404, map[string]any{"detail": "not found"})
			return
		}
		if action != "" && r.Method == http.MethodPost {
			var body map[string]any
			_ = readJSON(r, &body)
			note, _ := body["note"].(string)
			if note == "" {
				note, _ = body["review_note"].(string)
			}
			now := time.Now().UTC()
			switch strings.Trim(action, "/") {
			case "approve":
				// OPT-20260820-027：契约 sunset 后禁止批准仍声明旧版本的镜像上架
				// （write 路径虽已拒新写，审批动作仍须兜底，防存量镜像被运营误批）。
				if _, err := a.writableSkillVersion(img.SaasInboundSkillVersion); err != nil {
					writeJSON(w, 400, map[string]any{"detail": err.Error()})
					return
				}
				if err := img.Approve(staff.ID, note, now); err != nil {
					writeJSON(w, 400, map[string]any{"detail": err.Error()})
					return
				}
				rid := staff.ID
				_ = a.DB.UpdateContainerImageStatus(img, &rid)
				// OPT-20260824：同一镜像组允许多个 approved，但仅一个「激活」版本。
				// 组内尚无激活版本时，首个审批通过的版本自动成为激活版本；后续
				// 上架版本保持非激活，由厂商在版本列表显式切换激活。
				_ = a.DB.ActivateIfNoActiveVersion(img.ID, img.ImageGroupID)
				_ = a.DB.InsertReviewHistory(img.ID, "approve", note, staff.ID)
				logInfo("event=ContainerImageApproved id=%d", img.ID)
			case "reject":
				if note == "" {
					note = "驳回"
				}
				if err := img.Reject(staff.ID, note, now); err != nil {
					writeJSON(w, 400, map[string]any{"detail": err.Error()})
					return
				}
				rid := staff.ID
				_ = a.DB.UpdateContainerImageStatus(img, &rid)
				_ = a.DB.InsertReviewHistory(img.ID, "reject", note, staff.ID)
				logInfo("event=ContainerImageRejected id=%d", img.ID)
			case "unpublish":
				if err := img.Unpublish(staff.ID, note, now); err != nil {
					writeJSON(w, 400, map[string]any{"detail": err.Error()})
					return
				}
				rid := staff.ID
				_ = a.DB.UpdateContainerImageStatus(img, &rid)
			default:
				writeJSON(w, 404, map[string]any{"detail": "unknown action"})
				return
			}
			writeJSON(w, 200, map[string]any{"id": infrastructure.IDStr(img.ID), "status": img.Status})
			return
		}
	}
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		return
	}
	statusFilter := r.URL.Query().Get("status")
	items, err := a.DB.ListContainerImagesAdmin(statusFilter)
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handleAdminCloudServerImages(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		// userdata CRUD simplified
		if strings.Contains(r.URL.Path, "/userdata") {
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		return
	}
	items, err := a.DB.ListCloudServerImagesAdmin()
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handleAdminUserDataTemplates(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}
	if id, ok := parsePathID(r.URL.Path, "/api/admin/userdata-templates/"); ok {
		switch r.Method {
		case http.MethodGet:
			it, err := a.DB.GetUserDataTemplate(id)
			if err != nil {
				writeJSON(w, 404, map[string]any{"detail": "not found"})
				return
			}
			writeJSON(w, 200, it)
		case http.MethodPut, http.MethodPatch:
			var body map[string]any
			_ = readJSON(r, &body)
			if err := a.DB.UpdateUserDataTemplate(id, body); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			it, err := a.DB.GetUserDataTemplate(id)
			if err != nil {
				writeJSON(w, 200, map[string]any{"id": infrastructure.IDStr(id)})
				return
			}
			writeJSON(w, 200, it)
		case http.MethodDelete:
			if err := a.DB.DeleteUserDataTemplate(id); err != nil {
				msg := err.Error()
				if strings.Contains(msg, "not found") {
					writeJSON(w, 404, map[string]any{"detail": msg})
					return
				}
				writeJSON(w, 400, map[string]any{"detail": msg})
				return
			}
			w.WriteHeader(204)
		default:
			writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.DB.ListUserDataTemplates(false)
		if err != nil {
			writeJSON(w, 500, map[string]any{"detail": err.Error()})
			return
		}
		writeJSON(w, 200, items)
	case http.MethodPost:
		var body map[string]any
		_ = readJSON(r, &body)
		id, err := a.DB.CreateUserDataTemplate(body)
		if err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		it, err := a.DB.GetUserDataTemplate(id)
		if err != nil {
			writeJSON(w, 201, map[string]any{"id": infrastructure.IDStr(id)})
			return
		}
		writeJSON(w, 201, it)
	default:
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
	}
}

func (a *App) handleCredentialProxy(w http.ResponseWriter, r *http.Request) {
	a.proxyToCloud(w, r, strings.TrimPrefix(r.URL.Path, "/api"))
}

func (a *App) handleCloudQueryProxy(w http.ResponseWriter, r *http.Request) {
	a.proxyToCloud(w, r, strings.TrimPrefix(r.URL.Path, "/api"))
}

func (a *App) proxyToCloud(w http.ResponseWriter, r *http.Request, apiPath string) {
	base := strings.TrimRight(a.Cfg.CloudServiceBaseURL, "/")
	target := base + "/api" + apiPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	var body io.Reader = r.Body
	req, err := http.NewRequest(r.Method, target, body)
	if err != nil {
		writeJSON(w, 502, map[string]any{"detail": "proxy build failed"})
		return
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if a.Cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", a.Cfg.InternalSecret)
	}
	client := tracelog.DirectClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		logWarn(r.Context(), "cloud proxy error path=%s err=%v", apiPath, err)
		writeJSON(w, 502, map[string]any{"detail": "upstream unavailable"})
		return
	}
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func parseIDFlexible(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	case json.Number:
		return t.Int64()
	case int64:
		return t, nil
	default:
		return 0, strconv.ErrSyntax
	}
}
