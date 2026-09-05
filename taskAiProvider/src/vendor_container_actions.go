package main

import (
	"net/http"
	"strings"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func (a *App) handleVendorContainerAction(w http.ResponseWriter, r *http.Request, v *domain.Vendor, img *domain.ContainerImage, action string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		return
	}
	switch strings.Trim(action, "/") {
	case "submit":
		assocs, _ := a.DB.ListAssociations(img.ID)
		if len(assocs) == 0 {
			writeJSON(w, 400, map[string]any{"detail": "请先设置运行环境后再提交审核"})
			return
		}
		if err := img.Submit(); err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		_ = a.DB.UpdateContainerImageStatus(img, nil)
		logInfo("event=ContainerImageSubmitted id=%d", img.ID)
		writeJSON(w, 200, map[string]any{"id": infrastructure.IDStr(img.ID), "status": img.Status})
	case "withdraw":
		prev := img.Status
		now := time.Now().UTC()
		if err := img.VendorWithdraw(v.ID, "厂商自行下架", now); err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		rid := v.ID
		if err := a.DB.UpdateContainerImageStatus(img, &rid); err != nil {
			writeJSON(w, 500, map[string]any{"detail": err.Error()})
			return
		}
		switch prev {
		case domain.StatusPendingReview:
			_ = a.eventBus().Publish(r.Context(), domain.EventContainerImageReviewWithdrawn, map[string]any{
				"image_id": infrastructure.IDStr(img.ID), "vendor_id": infrastructure.IDStr(v.ID),
			})
			logInfo("event=ContainerImageReviewWithdrawn id=%d", img.ID)
		case domain.StatusApproved:
			_ = a.eventBus().Publish(r.Context(), domain.EventContainerImageUnpublished, map[string]any{
				"image_id": infrastructure.IDStr(img.ID), "vendor_id": infrastructure.IDStr(v.ID),
			})
			logInfo("event=ContainerImageUnpublished id=%d", img.ID)
		}
		writeJSON(w, 200, map[string]any{"id": infrastructure.IDStr(img.ID), "status": img.Status})
	case "activate":
		// OPT-20260824：同一镜像组允许多个审批通过（approved）的版本，但仅一个
		// 「激活」版本（公开目录生效）。厂商在多个已上架版本间显式切换激活。
		if err := a.DB.ActivateContainerImage(img.ID); err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		logInfo("event=ContainerImageActivated id=%d group=%d", img.ID, img.ImageGroupID)
		items, _ := a.DB.ListContainerImages(v.ID)
		for _, it := range items {
			if it["id"] == infrastructure.IDStr(img.ID) {
				writeJSON(w, 200, it)
				return
			}
		}
		writeJSON(w, 200, map[string]any{"id": infrastructure.IDStr(img.ID), "status": img.Status})
	case "cloud-server-image-associations", "cloud-server-image-association":
		if r.Method == http.MethodGet {
			items, _ := a.DB.ListAssociations(img.ID)
			writeJSON(w, 200, items)
			return
		}
		var body map[string]any
		_ = readJSON(r, &body)
		// Bulk replace when client sends items[]; otherwise upsert one region
		// so saving cn-hongkong does not wipe cn-qingdao (SetAssociations is DELETE-all).
		if raw, ok := body["items"].([]any); ok {
			items := make([]map[string]any, 0, len(raw))
			for _, x := range raw {
				if m, ok := x.(map[string]any); ok {
					items = append(items, m)
				}
			}
			if err := a.DB.SetAssociations(img.ID, items); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			for _, it := range items {
				csiID := infrastructure.CloudServerImageIDFromAssocItem(it)
				if csiID == 0 {
					continue
				}
				if tplID, ok := infrastructure.OptionalUserdataTemplateID(it); ok {
					if err := a.DB.SetCloudServerImageUserdataTemplate(csiID, tplID); err != nil {
						writeJSON(w, 400, map[string]any{"detail": err.Error()})
						return
					}
				}
			}
		} else {
			csiID := infrastructure.CloudServerImageIDFromAssocItem(body)
			platform, region := infrastructure.AssocPlatformRegion(body)
			if err := a.DB.UpsertAssociation(img.ID, platform, region, csiID); err != nil {
				writeJSON(w, 400, map[string]any{"detail": err.Error()})
				return
			}
			// Region-env modal also picks UserData template; persist on the CSI row.
			if csiID > 0 {
				if tplID, ok := infrastructure.OptionalUserdataTemplateID(body); ok {
					if err := a.DB.SetCloudServerImageUserdataTemplate(csiID, tplID); err != nil {
						writeJSON(w, 400, map[string]any{"detail": err.Error()})
						return
					}
				}
			}
		}
		out, _ := a.DB.ListAssociations(img.ID)
		writeJSON(w, 200, out)
	case "set-cloud-server-images":
		var body struct {
			Items []map[string]any `json:"items"`
		}
		_ = readJSON(r, &body)
		if err := a.DB.SetAssociations(img.ID, body.Items); err != nil {
			writeJSON(w, 400, map[string]any{"detail": err.Error()})
			return
		}
		items, _ := a.DB.ListAssociations(img.ID)
		writeJSON(w, 200, items)
	default:
		writeJSON(w, 404, map[string]any{"detail": "unknown action"})
	}
}
