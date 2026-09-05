package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

type imageSkillsExtractor interface {
	Extract(imageURL string) infrastructure.ImageSkillsExtractResult
}

type defaultImageSkillsExtractor struct{}

func (defaultImageSkillsExtractor) Extract(imageURL string) infrastructure.ImageSkillsExtractResult {
	return infrastructure.ExtractImageSkillsFromImage(imageURL)
}

type noopImageSkillsExtractor struct{}

func (noopImageSkillsExtractor) Extract(string) infrastructure.ImageSkillsExtractResult {
	return infrastructure.ImageSkillsExtractResult{Status: "not_found"}
}

var skillsExtractInflight sync.Map

func (a *App) imageSkillsExtractor() imageSkillsExtractor {
	if a != nil && a.SkillsExtractor != nil {
		return a.SkillsExtractor
	}
	return defaultImageSkillsExtractor{}
}

func (a *App) scheduleImageSkillsExtract(ctx context.Context, id int64, imageURL, version string) {
	if id == 0 || strings.TrimSpace(imageURL) == "" || a == nil || a.DB == nil {
		return
	}
	if _, loaded := skillsExtractInflight.LoadOrStore(id, true); loaded {
		return
	}
	run := func() {
		defer skillsExtractInflight.Delete(id)
		a.runImageSkillsExtract(ctx, id, imageURL, version)
	}
	if a.ExtractSync {
		run()
		return
	}
	go run()
}

func (a *App) runImageSkillsExtract(ctx context.Context, id int64, imageURL, version string) {
	merged, err := infrastructure.MergeImageURLWithVersion(imageURL, version)
	if err != nil {
		logWarn(ctx, "event=ContainerImageSkillsExtractFailed id=%d err=%s", id, err.Error())
		if dbErr := a.DB.UpdateContainerImageFields(id, map[string]any{
			"image_skills_json":           "",
			"image_skills_extract_status": "failed",
			"image_skills_digest":         "",
		}); dbErr != nil {
			logError(ctx, "event=ContainerImageSkillsExtractPersistFailed id=%d err=%s", id, dbErr.Error())
		}
		return
	}
	start := time.Now()
	if err := a.DB.UpdateContainerImageFields(id, map[string]any{
		"image_skills_extract_status": "pending",
	}); err != nil {
		logError(ctx, "event=ContainerImageSkillsExtractPendingFailed id=%d err=%s", id, err.Error())
		return
	}
	logInfo("event=ContainerImageSkillsExtractStarted id=%d", id)
	result := a.imageSkillsExtractor().Extract(merged)
	a.persistImageSkills(ctx, id, result, start)
}

// persistImageSkills writes one image_skills_* extraction outcome and emits the
// corresponding event. Shared by the single-file and combined runners.
func (a *App) persistImageSkills(ctx context.Context, id int64, result infrastructure.ImageSkillsExtractResult, start time.Time) {
	jsonBody := result.JSON
	if result.Status == "not_found" {
		jsonBody = infrastructure.EmptyImageSkillsJSON()
	}
	if err := a.DB.UpdateContainerImageFields(id, map[string]any{
		"image_skills_json":           jsonBody,
		"image_skills_extract_status": result.Status,
		"image_skills_digest":         result.Digest,
	}); err != nil {
		logError(ctx, "event=ContainerImageSkillsExtractPersistFailed id=%d err=%s", id, err.Error())
		return
	}
	durationMS := time.Since(start).Milliseconds()
	if result.Status == "ok" {
		logInfo("event=ContainerImageSkillsExtracted id=%d status=%s skill_count=%d duration_ms=%d", id, result.Status, len(result.List.Skills), durationMS)
	} else {
		logWarn(ctx, "event=ContainerImageSkillsExtracted id=%d status=%s detail=%s duration_ms=%d", id, result.Status, result.Detail, durationMS)
	}
	_ = a.eventBus().Publish(context.Background(), domain.EventContainerImageSkillsExtracted, map[string]any{
		"image_id":    infrastructure.IDStr(id),
		"status":      result.Status,
		"digest":      result.Digest,
		"skill_count": len(result.List.Skills),
		"duration_ms": durationMS,
	})
}
